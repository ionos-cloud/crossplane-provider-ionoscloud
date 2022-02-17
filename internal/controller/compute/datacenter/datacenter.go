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

package datacenter

import (
	"context"
	"fmt"
	"net/http"

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

	"github.com/ionos-cloud/crossplane-provider-ionoscloud/apis/compute/v1alpha1"
	apisv1alpha1 "github.com/ionos-cloud/crossplane-provider-ionoscloud/apis/v1alpha1"
	"github.com/ionos-cloud/crossplane-provider-ionoscloud/internal/clients"
	"github.com/ionos-cloud/crossplane-provider-ionoscloud/internal/clients/compute/datacenter"
)

const (
	errNotDatacenter = "managed resource is not a Datacenter custom resource"
	errTrackPCUsage  = "cannot track ProviderConfig usage"
	errGetPC         = "cannot get ProviderConfig"
	errGetCreds      = "cannot get credentials"

	errNewClient = "cannot create new Service"
)

// Setup adds a controller that reconciles Datacenter managed resources.
func Setup(mgr ctrl.Manager, l logging.Logger, rl workqueue.RateLimiter) error {
	name := managed.ControllerName(v1alpha1.DatacenterGroupKind)

	return ctrl.NewControllerManagedBy(mgr).
		Named(name).
		WithOptions(controller.Options{
			RateLimiter: ratelimiter.NewDefaultManagedRateLimiter(rl),
		}).
		For(&v1alpha1.Datacenter{}).
		Complete(managed.NewReconciler(mgr,
			resource.ManagedKind(v1alpha1.DatacenterGroupVersionKind),
			managed.WithExternalConnecter(&connectorDatacenter{
				kube:  mgr.GetClient(),
				usage: resource.NewProviderConfigUsageTracker(mgr.GetClient(), &apisv1alpha1.ProviderConfigUsage{}),
				log:   l}),
			managed.WithLogger(l.WithValues("controller", name)),
			managed.WithRecorder(event.NewAPIRecorder(mgr.GetEventRecorderFor(name)))))
}

// A connectorDatacenter is expected to produce an ExternalClient when its Connect method
// is called.
type connectorDatacenter struct {
	kube  client.Client
	usage resource.Tracker
	log   logging.Logger
}

// Connect typically produces an ExternalClient by:
// 1. Tracking that the managed resource is using a ProviderConfig.
// 2. Getting the managed resource's ProviderConfig.
// 3. Getting the credentials specified by the ProviderConfig.
// 4. Using the credentials to form a client.
func (c *connectorDatacenter) Connect(ctx context.Context, mg resource.Managed) (managed.ExternalClient, error) {
	_, ok := mg.(*v1alpha1.Datacenter)
	if !ok {
		return nil, errors.New(errNotDatacenter)
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
	return &externalDatacenter{service: &datacenter.APIClient{IonosServices: svc}, log: c.log}, nil
}

// An ExternalClient observes, then either creates, updates, or deletes an
// externalDatacenter resource to ensure it reflects the managed resource's desired state.
type externalDatacenter struct {
	// A 'client' used to connect to the externalDatacenter resource API. In practice this
	// would be something like an AWS SDK client.
	service datacenter.Client
	log     logging.Logger
}

func (c *externalDatacenter) Observe(ctx context.Context, mg resource.Managed) (managed.ExternalObservation, error) {
	cr, ok := mg.(*v1alpha1.Datacenter)
	if !ok {
		return managed.ExternalObservation{}, errors.New(errNotDatacenter)
	}

	// External Name of the CR is the DBaaS Postgres Datacenter ID
	id := meta.GetExternalName(cr)
	if id == "" {
		return managed.ExternalObservation{
			ResourceExists:    false,
			ResourceUpToDate:  false,
			ConnectionDetails: managed.ConnectionDetails{},
		}, nil
	}
	cr.Status.AtProvider.DatacenterID = id
	instance, resp, err := c.service.GetDatacenter(ctx, id)
	if err != nil {
		retErr := fmt.Errorf("failed to get datacenter by id. Request: %v: %w", resp.RequestURL, err)
		if resp.StatusCode == http.StatusNotFound {
			retErr = nil
		}
		return managed.ExternalObservation{
			ResourceExists:    false,
			ResourceUpToDate:  false,
			ConnectionDetails: managed.ConnectionDetails{},
		}, retErr
	}

	cr.Status.AtProvider.State = *instance.Metadata.State
	c.log.Debug(fmt.Sprintf("Observing state %v...", cr.Status.AtProvider.State))
	// Set Ready condition based on State
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
		ResourceUpToDate:  datacenter.IsDatacenterUpToDate(cr, instance),
		ConnectionDetails: managed.ConnectionDetails{},
	}, nil
}

func (c *externalDatacenter) Create(ctx context.Context, mg resource.Managed) (managed.ExternalCreation, error) {
	cr, ok := mg.(*v1alpha1.Datacenter)
	if !ok {
		return managed.ExternalCreation{}, errors.New(errNotDatacenter)
	}

	cr.SetConditions(xpv1.Creating())
	if cr.Status.AtProvider.State == string(ionoscloud.BUSY) {
		return managed.ExternalCreation{}, nil
	}
	instanceInput, err := datacenter.GenerateCreateDatacenterInput(cr)
	if err != nil {
		return managed.ExternalCreation{}, err
	}

	instance, apiResponse, err := c.service.CreateDatacenter(ctx, *instanceInput)
	creation := managed.ExternalCreation{
		ConnectionDetails: managed.ConnectionDetails{},
	}
	if err != nil {
		return creation, fmt.Errorf("failed to create Datacenter: %w, apiResponse: %v", err, apiResponse.Status)
	}

	// Set External Name
	cr.Status.AtProvider.DatacenterID = *instance.Id
	meta.SetExternalName(cr, *instance.Id)
	creation.ExternalNameAssigned = true
	return creation, nil
}

func (c *externalDatacenter) Update(ctx context.Context, mg resource.Managed) (managed.ExternalUpdate, error) {
	cr, ok := mg.(*v1alpha1.Datacenter)
	if !ok {
		return managed.ExternalUpdate{}, errors.New(errNotDatacenter)
	}
	if cr.Status.AtProvider.State == string(ionoscloud.BUSY) {
		return managed.ExternalUpdate{}, nil
	}

	datacenterID := cr.Status.AtProvider.DatacenterID
	instanceInput, err := datacenter.GenerateUpdateDatacenterInput(cr)
	if err != nil {
		return managed.ExternalUpdate{}, nil
	}

	_, apiResponse, err := c.service.UpdateDatacenter(ctx, datacenterID, *instanceInput)
	update := managed.ExternalUpdate{
		ConnectionDetails: managed.ConnectionDetails{},
	}
	if err != nil {
		return update, fmt.Errorf("failed to update Datacenter: %w, apiResponse: %v", err, apiResponse.Status)
	}
	return update, nil
}

func (c *externalDatacenter) Delete(ctx context.Context, mg resource.Managed) error {
	cr, ok := mg.(*v1alpha1.Datacenter)
	if !ok {
		return errors.New(errNotDatacenter)
	}

	cr.SetConditions(xpv1.Deleting())
	if cr.Status.AtProvider.State == string(ionoscloud.DESTROYING) {
		return nil
	}

	err := c.service.DeleteDatacenter(ctx, cr.Status.AtProvider.DatacenterID)
	return errors.Wrap(err, "failed to delete datacenter")
}
