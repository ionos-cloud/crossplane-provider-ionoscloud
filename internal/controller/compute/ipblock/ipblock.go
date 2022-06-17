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

package ipblock

import (
	"context"
	"fmt"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/pkg/errors"
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

	"github.com/ionos-cloud/crossplane-provider-ionoscloud/apis/compute/v1alpha1"
	apisv1alpha1 "github.com/ionos-cloud/crossplane-provider-ionoscloud/apis/v1alpha1"
	"github.com/ionos-cloud/crossplane-provider-ionoscloud/internal/clients"
	"github.com/ionos-cloud/crossplane-provider-ionoscloud/internal/clients/compute"
	"github.com/ionos-cloud/crossplane-provider-ionoscloud/internal/clients/compute/ipblock"
)

const errNotIPBlock = "managed resource is not a IPBlock custom resource"

// Setup adds a controller that reconciles IPBlock managed resources.
func Setup(mgr ctrl.Manager, l logging.Logger, rl workqueue.RateLimiter, poll time.Duration, creationGracePeriod time.Duration) error {
	name := managed.ControllerName(v1alpha1.IPBlockGroupKind)

	return ctrl.NewControllerManagedBy(mgr).
		Named(name).
		WithOptions(controller.Options{
			RateLimiter: ratelimiter.NewDefaultManagedRateLimiter(rl),
		}).
		For(&v1alpha1.IPBlock{}).
		Complete(managed.NewReconciler(mgr,
			resource.ManagedKind(v1alpha1.IPBlockGroupVersionKind),
			managed.WithExternalConnecter(&connectorIPBlock{
				kube:  mgr.GetClient(),
				usage: resource.NewProviderConfigUsageTracker(mgr.GetClient(), &apisv1alpha1.ProviderConfigUsage{}),
				log:   l}),
			managed.WithReferenceResolver(managed.NewAPISimpleReferenceResolver(mgr.GetClient())),
			managed.WithInitializers(),
			managed.WithPollInterval(poll),
			managed.WithCreationGracePeriod(creationGracePeriod),
			managed.WithLogger(l.WithValues("controller", name)),
			managed.WithRecorder(event.NewAPIRecorder(mgr.GetEventRecorderFor(name)))))
}

// A connectorIPBlock is expected to produce an ExternalClient when its Connect method
// is called.
type connectorIPBlock struct {
	kube  client.Client
	usage resource.Tracker
	log   logging.Logger
}

// Connect typically produces an ExternalClient by:
// 1. Tracking that the managed resource is using a ProviderConfig.
// 2. Getting the managed resource's ProviderConfig.
// 3. Getting the credentials specified by the ProviderConfig.
// 4. Using the credentials to form a client.
func (c *connectorIPBlock) Connect(ctx context.Context, mg resource.Managed) (managed.ExternalClient, error) {
	_, ok := mg.(*v1alpha1.IPBlock)
	if !ok {
		return nil, errors.New(errNotIPBlock)
	}
	svc, err := clients.ConnectForCRD(ctx, mg, c.kube, c.usage)
	return &externalIPBlock{
		service: &ipblock.APIClient{IonosServices: svc},
		log:     c.log}, err
}

// An ExternalClient observes, then either creates, updates, or deletes an
// externalIPBlock resource to ensure it reflects the managed resource's desired state.
type externalIPBlock struct {
	// A 'client' used to connect to the externalIPBlock resource API. In practice this
	// would be something like an IONOS Cloud SDK client.
	service ipblock.Client
	log     logging.Logger
}

func (c *externalIPBlock) Observe(ctx context.Context, mg resource.Managed) (managed.ExternalObservation, error) {
	cr, ok := mg.(*v1alpha1.IPBlock)
	if !ok {
		return managed.ExternalObservation{}, errors.New(errNotIPBlock)
	}

	// External Name of the CR is the IPBlock ID
	if meta.GetExternalName(cr) == "" {
		return managed.ExternalObservation{}, nil
	}
	observed, apiResponse, err := c.service.GetIPBlock(ctx, meta.GetExternalName(cr))
	if err != nil {
		retErr := fmt.Errorf("failed to get ipBlock by id. error: %w", err)
		return managed.ExternalObservation{}, compute.CheckAPIResponseInfo(apiResponse, retErr)
	}
	current := cr.Spec.ForProvider.DeepCopy()
	ipblock.LateStatusInitializer(&cr.Status.AtProvider, &observed)

	cr.Status.AtProvider.IPBlockID = meta.GetExternalName(cr)
	cr.Status.AtProvider.State = *observed.Metadata.State
	c.log.Debug(fmt.Sprintf("Observing state: %v", cr.Status.AtProvider.State))
	// Set Ready condition based on State
	switch cr.Status.AtProvider.State {
	case compute.AVAILABLE, compute.ACTIVE:
		cr.SetConditions(xpv1.Available())
	case compute.BUSY, compute.UPDATING:
		cr.SetConditions(xpv1.Creating())
	case compute.DESTROYING:
		cr.SetConditions(xpv1.Deleting())
	default:
		cr.SetConditions(xpv1.Unavailable())
	}

	return managed.ExternalObservation{
		ResourceExists:          true,
		ResourceUpToDate:        ipblock.IsIPBlockUpToDate(cr, observed),
		ConnectionDetails:       managed.ConnectionDetails{},
		ResourceLateInitialized: !cmp.Equal(current, &cr.Spec.ForProvider),
	}, nil
}

func (c *externalIPBlock) Create(ctx context.Context, mg resource.Managed) (managed.ExternalCreation, error) {
	cr, ok := mg.(*v1alpha1.IPBlock)
	if !ok {
		return managed.ExternalCreation{}, errors.New(errNotIPBlock)
	}

	cr.SetConditions(xpv1.Creating())
	if cr.Status.AtProvider.State == compute.BUSY {
		return managed.ExternalCreation{}, nil
	}
	instanceInput, err := ipblock.GenerateCreateIPBlockInput(cr)
	if err != nil {
		return managed.ExternalCreation{}, err
	}

	instance, apiResponse, err := c.service.CreateIPBlock(ctx, *instanceInput)
	creation := managed.ExternalCreation{ConnectionDetails: managed.ConnectionDetails{}}
	if err != nil {
		retErr := fmt.Errorf("failed to create ipBlock. error: %w", err)
		return creation, compute.AddAPIResponseInfo(apiResponse, retErr)
	}
	if err = compute.WaitForRequest(ctx, c.service.GetAPIClient(), apiResponse); err != nil {
		return creation, err
	}

	// Set External Name
	cr.Status.AtProvider.IPBlockID = *instance.Id
	meta.SetExternalName(cr, *instance.Id)
	return creation, nil
}

func (c *externalIPBlock) Update(ctx context.Context, mg resource.Managed) (managed.ExternalUpdate, error) {
	cr, ok := mg.(*v1alpha1.IPBlock)
	if !ok {
		return managed.ExternalUpdate{}, errors.New(errNotIPBlock)
	}
	if cr.Status.AtProvider.State == compute.BUSY || cr.Status.AtProvider.State == compute.UPDATING {
		return managed.ExternalUpdate{}, nil
	}

	ipBlockID := cr.Status.AtProvider.IPBlockID
	instanceInput, err := ipblock.GenerateUpdateIPBlockInput(cr)
	if err != nil {
		return managed.ExternalUpdate{}, err
	}

	_, apiResponse, err := c.service.UpdateIPBlock(ctx, ipBlockID, *instanceInput)
	if err != nil {
		retErr := fmt.Errorf("failed to update ipBlock. error: %w", err)
		return managed.ExternalUpdate{}, compute.AddAPIResponseInfo(apiResponse, retErr)
	}
	if err = compute.WaitForRequest(ctx, c.service.GetAPIClient(), apiResponse); err != nil {
		return managed.ExternalUpdate{}, err
	}
	return managed.ExternalUpdate{}, nil
}

func (c *externalIPBlock) Delete(ctx context.Context, mg resource.Managed) error {
	cr, ok := mg.(*v1alpha1.IPBlock)
	if !ok {
		return errors.New(errNotIPBlock)
	}

	cr.SetConditions(xpv1.Deleting())
	if cr.Status.AtProvider.State == compute.DESTROYING {
		return nil
	}

	apiResponse, err := c.service.DeleteIPBlock(ctx, cr.Status.AtProvider.IPBlockID)
	if err != nil {
		retErr := fmt.Errorf("failed to delete ipBlock. error: %w", err)
		return compute.AddAPIResponseInfo(apiResponse, retErr)
	}
	if err = compute.WaitForRequest(ctx, c.service.GetAPIClient(), apiResponse); err != nil {
		return err
	}
	return nil
}
