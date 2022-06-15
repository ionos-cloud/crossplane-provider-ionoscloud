/*
Copyright 2020 The Crossplane Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package targetgroup

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/util/workqueue"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"

	xpv1 "github.com/crossplane/crossplane-runtime/apis/common/v1"
	"github.com/crossplane/crossplane-runtime/pkg/event"
	"github.com/crossplane/crossplane-runtime/pkg/logging"
	"github.com/crossplane/crossplane-runtime/pkg/meta"
	"github.com/crossplane/crossplane-runtime/pkg/ratelimiter"
	"github.com/crossplane/crossplane-runtime/pkg/reconciler/managed"
	"github.com/crossplane/crossplane-runtime/pkg/resource"

	ionoscloud "github.com/ionos-cloud/sdk-go-dbaas-postgres"

	"github.com/ionos-cloud/crossplane-provider-ionoscloud/apis/alb/v1alpha1"
	apisv1alpha1 "github.com/ionos-cloud/crossplane-provider-ionoscloud/apis/v1alpha1"
	"github.com/ionos-cloud/crossplane-provider-ionoscloud/internal/clients"
	"github.com/ionos-cloud/crossplane-provider-ionoscloud/internal/clients/alb/targetgroup"
	"github.com/ionos-cloud/crossplane-provider-ionoscloud/internal/clients/compute"
	"github.com/ionos-cloud/crossplane-provider-ionoscloud/internal/clients/compute/ipblock"
)

const (
	errNotTargetGroup = "managed resource is not a TargetGroup custom resource"
	errTrackPCUsage   = "cannot track ProviderConfig usage"
	errGetPC          = "cannot get ProviderConfig"
	errGetCreds       = "cannot get credentials"

	errNewClient = "cannot create new Service"
)

// Setup adds a controller that reconciles TargetGroup managed resources.
func Setup(mgr ctrl.Manager, l logging.Logger, rl workqueue.RateLimiter, poll, createGracePeriod time.Duration) error {
	name := managed.ControllerName(v1alpha1.TargetGroupGroupKind)

	return ctrl.NewControllerManagedBy(mgr).
		Named(name).
		WithOptions(controller.Options{
			RateLimiter: ratelimiter.NewDefaultManagedRateLimiter(rl),
		}).
		For(&v1alpha1.TargetGroup{}).
		Complete(managed.NewReconciler(mgr,
			resource.ManagedKind(v1alpha1.TargetGroupGroupVersionKind),
			managed.WithExternalConnecter(&connectorTargetGroup{
				kube:  mgr.GetClient(),
				usage: resource.NewProviderConfigUsageTracker(mgr.GetClient(), &apisv1alpha1.ProviderConfigUsage{}),
				log:   l}),
			managed.WithReferenceResolver(managed.NewAPISimpleReferenceResolver(mgr.GetClient())),
			managed.WithInitializers(),
			managed.WithCreationGracePeriod(createGracePeriod),
			managed.WithPollInterval(poll),
			managed.WithLogger(l.WithValues("controller", name)),
			managed.WithRecorder(event.NewAPIRecorder(mgr.GetEventRecorderFor(name)))))
}

// A connectorTargetGroup is expected to produce an ExternalClient when its Connect method
// is called.
type connectorTargetGroup struct {
	kube  client.Client
	usage resource.Tracker
	log   logging.Logger
}

// Connect typically produces an ExternalClient by:
// 1. Tracking that the managed resource is using a ProviderConfig.
// 2. Getting the managed resource's ProviderConfig.
// 3. Getting the credentials specified by the ProviderConfig.
// 4. Using the credentials to form a client.
func (c *connectorTargetGroup) Connect(ctx context.Context, mg resource.Managed) (managed.ExternalClient, error) {
	_, ok := mg.(*v1alpha1.TargetGroup)
	if !ok {
		return nil, errors.New(errNotTargetGroup)
	}

	if err := c.usage.Track(ctx, mg); err != nil {
		return nil, errors.Wrap(err, errTrackPCUsage)
	}

	pc := &apisv1alpha1.ProviderConfig{}
	if err := c.kube.Get(ctx, types.NamespacedName{Name: mg.GetProviderConfigReference().Name}, pc); err != nil {
		return nil, errors.Wrap(err, errGetPC)
	}

	cd := pc.Spec.Credentials
	data, err := resource.CommonCredentialExtractor(ctx, cd.Source, c.kube, cd.CommonCredentialSelectors)
	if err != nil {
		return nil, errors.Wrap(err, errGetCreds)
	}

	svc, err := clients.NewIonosClients(data)
	if err != nil {
		return nil, errors.Wrap(err, errNewClient)
	}
	return &externalTargetGroup{service: &targetgroup.APIClient{IonosServices: svc}, ipblockService: &ipblock.APIClient{IonosServices: svc}, log: c.log}, nil
}

// An ExternalClient observes, then either creates, updates, or deletes an
// externalTargetGroup resource to ensure it reflects the managed resource's desired state.
type externalTargetGroup struct {
	// A 'client' used to connect to the externalTargetGroup resource API. In practice this
	// would be something like an IONOS Cloud SDK client.
	service        targetgroup.Client
	ipblockService ipblock.Client
	log            logging.Logger
}

func (c *externalTargetGroup) Observe(ctx context.Context, mg resource.Managed) (managed.ExternalObservation, error) { // nolint:gocyclo
	cr, ok := mg.(*v1alpha1.TargetGroup)
	if !ok {
		return managed.ExternalObservation{}, errors.New(errNotTargetGroup)
	}

	if meta.GetExternalName(cr) == "" {
		return managed.ExternalObservation{}, nil
	}
	observed, resp, err := c.service.GetTargetGroup(ctx, meta.GetExternalName(cr))
	if err != nil {
		if resp != nil && resp.Response != nil && resp.StatusCode == http.StatusNotFound {
			return managed.ExternalObservation{}, nil
		}
		return managed.ExternalObservation{}, fmt.Errorf("failed to get target group by id. err: %w", err)
	}
	cr.Status.AtProvider.TargetGroupID = meta.GetExternalName(cr)
	if observed.HasMetadata() {
		if observed.Metadata.HasState() {
			cr.Status.AtProvider.State = *observed.Metadata.State
		}
	}
	c.log.Debug(fmt.Sprintf("Observing state: %v", cr.Status.AtProvider.State))
	switch cr.Status.AtProvider.State {
	case string(ionoscloud.AVAILABLE):
		cr.SetConditions(xpv1.Available())
	case string(ionoscloud.DESTROYING):
		cr.SetConditions(xpv1.Deleting())
	case string(ionoscloud.BUSY):
		cr.SetConditions(xpv1.Creating())
	default:
		cr.SetConditions(xpv1.Unavailable())
	}
	return managed.ExternalObservation{
		ResourceExists:    true,
		ResourceUpToDate:  targetgroup.IsTargetGroupUpToDate(cr, observed),
		ConnectionDetails: managed.ConnectionDetails{},
	}, nil
}

func (c *externalTargetGroup) Create(ctx context.Context, mg resource.Managed) (managed.ExternalCreation, error) {
	cr, ok := mg.(*v1alpha1.TargetGroup)
	if !ok {
		return managed.ExternalCreation{}, errors.New(errNotTargetGroup)
	}

	cr.SetConditions(xpv1.Creating())
	instanceInput, err := targetgroup.GenerateCreateTargetGroupInput(cr)
	if err != nil {
		return managed.ExternalCreation{}, err
	}
	instance, apiResponse, err := c.service.CreateTargetGroup(ctx, *instanceInput)
	creation := managed.ExternalCreation{ConnectionDetails: managed.ConnectionDetails{}}
	if err != nil {
		retErr := fmt.Errorf("failed to create target group: %w", err)
		if apiResponse != nil && apiResponse.Response != nil {
			retErr = fmt.Errorf("%w API Response Status: %v", retErr, apiResponse.Status)
		}
		return creation, retErr
	}

	// Set External Name
	cr.Status.AtProvider.TargetGroupID = *instance.Id
	meta.SetExternalName(cr, *instance.Id)
	return creation, nil
}

func (c *externalTargetGroup) Update(ctx context.Context, mg resource.Managed) (managed.ExternalUpdate, error) {
	cr, ok := mg.(*v1alpha1.TargetGroup)
	if !ok {
		return managed.ExternalUpdate{}, errors.New(errNotTargetGroup)
	}
	if cr.Status.AtProvider.State == string(ionoscloud.BUSY) {
		return managed.ExternalUpdate{}, nil
	}

	instanceInput, err := targetgroup.GenerateUpdateTargetGroupInput(cr)
	if err != nil {
		return managed.ExternalUpdate{}, nil
	}
	_, apiResponse, err := c.service.UpdateTargetGroup(ctx, cr.Status.AtProvider.TargetGroupID, *instanceInput)
	if err != nil {
		retErr := fmt.Errorf("failed to update target group: %w", err)
		if apiResponse != nil && apiResponse.Response != nil {
			retErr = fmt.Errorf("%w API Response Status: %v", retErr, apiResponse.Status)
		}
		return managed.ExternalUpdate{}, retErr
	}
	// This is a temporary solution until API requests for ALB are processed faster.
	c.log.Debug("Waiting for request...")
	ctx, cancel := context.WithTimeout(ctx, 30*time.Minute)
	defer cancel()
	if err = compute.WaitForRequest(ctx, c.service.GetAPIClient(), apiResponse); err != nil {
		return managed.ExternalUpdate{}, err
	}
	return managed.ExternalUpdate{}, nil
}

func (c *externalTargetGroup) Delete(ctx context.Context, mg resource.Managed) error {
	cr, ok := mg.(*v1alpha1.TargetGroup)
	if !ok {
		return errors.New(errNotTargetGroup)
	}

	cr.SetConditions(xpv1.Deleting())
	if cr.Status.AtProvider.State == string(ionoscloud.DESTROYING) || cr.Status.AtProvider.State == string(ionoscloud.BUSY) {
		return nil
	}

	apiResponse, err := c.service.DeleteTargetGroup(ctx, cr.Status.AtProvider.TargetGroupID)
	if err != nil {
		retErr := fmt.Errorf("failed to delete target group. error: %w", err)
		return compute.AddAPIResponseInfo(apiResponse, retErr)
	}
	// This is a temporary solution until API requests for ALB are processed faster.
	c.log.Debug("Waiting for request...")
	ctx, cancel := context.WithTimeout(ctx, 30*time.Minute)
	defer cancel()
	if err = compute.WaitForRequest(ctx, c.service.GetAPIClient(), apiResponse); err != nil {
		return err
	}
	return nil
}