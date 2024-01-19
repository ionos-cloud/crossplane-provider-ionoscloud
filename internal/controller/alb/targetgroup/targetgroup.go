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

	"github.com/ionos-cloud/crossplane-provider-ionoscloud/apis/alb/v1alpha1"
	apisv1alpha1 "github.com/ionos-cloud/crossplane-provider-ionoscloud/apis/v1alpha1"
	"github.com/ionos-cloud/crossplane-provider-ionoscloud/internal/clients"
	"github.com/ionos-cloud/crossplane-provider-ionoscloud/internal/clients/alb/targetgroup"
	"github.com/ionos-cloud/crossplane-provider-ionoscloud/internal/clients/compute"
	"github.com/ionos-cloud/crossplane-provider-ionoscloud/internal/utils"
)

const errNotTargetGroup = "managed resource is not a TargetGroup custom resource"

// Setup adds a controller that reconciles TargetGroup managed resources.
func Setup(mgr ctrl.Manager, l logging.Logger, rl workqueue.RateLimiter, opts *utils.ConfigurationOptions) error {
	name := managed.ControllerName(v1alpha1.TargetGroupGroupKind)

	return ctrl.NewControllerManagedBy(mgr).
		Named(name).
		WithOptions(controller.Options{
			RateLimiter: ratelimiter.NewController(),
		}).
		For(&v1alpha1.TargetGroup{}).
		Complete(managed.NewReconciler(mgr,
			resource.ManagedKind(v1alpha1.TargetGroupGroupVersionKind),
			managed.WithExternalConnecter(&connectorTargetGroup{
				kube:                 mgr.GetClient(),
				usage:                resource.NewProviderConfigUsageTracker(mgr.GetClient(), &apisv1alpha1.ProviderConfigUsage{}),
				log:                  l,
				isUniqueNamesEnabled: opts.GetIsUniqueNamesEnabled()}),
			managed.WithReferenceResolver(managed.NewAPISimpleReferenceResolver(mgr.GetClient())),
			managed.WithInitializers(),
			managed.WithCreationGracePeriod(opts.GetCreationGracePeriod()),
			managed.WithPollInterval(opts.GetPollInterval()),
			managed.WithTimeout(opts.GetTimeout()),
			managed.WithLogger(l.WithValues("controller", name)),
			managed.WithRecorder(event.NewAPIRecorder(mgr.GetEventRecorderFor(name)))))
}

// A connectorTargetGroup is expected to produce an ExternalClient when its Connect method
// is called.
type connectorTargetGroup struct {
	kube                 client.Client
	usage                resource.Tracker
	log                  logging.Logger
	isUniqueNamesEnabled bool
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
	svc, err := clients.ConnectForCRD(ctx, mg, c.kube, c.usage)
	return &externalTargetGroup{
		service:              &targetgroup.APIClient{IonosServices: svc},
		log:                  c.log,
		isUniqueNamesEnabled: c.isUniqueNamesEnabled}, err
}

// An ExternalClient observes, then either creates, updates, or deletes an
// externalTargetGroup resource to ensure it reflects the managed resource's desired state.
type externalTargetGroup struct {
	// A 'client' used to connect to the externalTargetGroup resource API. In practice this
	// would be something like an IONOS Cloud SDK client.
	service              targetgroup.Client
	log                  logging.Logger
	isUniqueNamesEnabled bool
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
		retErr := fmt.Errorf("failed to get target group by id. error: %w", err)
		return managed.ExternalObservation{}, compute.ErrorUnlessNotFound(resp, retErr)
	}
	current := cr.Spec.ForProvider.DeepCopy()
	targetgroup.LateInitializer(&cr.Spec.ForProvider, &observed)
	cr.Status.AtProvider.TargetGroupID = meta.GetExternalName(cr)
	cr.Status.AtProvider.State = clients.GetCoreResourceState(&observed)
	c.log.Debug(fmt.Sprintf("Observing state: %v", cr.Status.AtProvider.State))
	clients.UpdateCondition(cr, cr.Status.AtProvider.State)
	return managed.ExternalObservation{
		ResourceExists:          true,
		ResourceUpToDate:        targetgroup.IsTargetGroupUpToDate(cr, observed),
		ConnectionDetails:       managed.ConnectionDetails{},
		ResourceLateInitialized: !cmp.Equal(current, &cr.Spec.ForProvider),
	}, nil
}

func (c *externalTargetGroup) Create(ctx context.Context, mg resource.Managed) (managed.ExternalCreation, error) {
	cr, ok := mg.(*v1alpha1.TargetGroup)
	if !ok {
		return managed.ExternalCreation{}, errors.New(errNotTargetGroup)
	}
	cr.SetConditions(xpv1.Creating())
	if cr.Status.AtProvider.State == compute.BUSY {
		return managed.ExternalCreation{}, nil
	}

	if c.isUniqueNamesEnabled {
		// TargetGroup should have unique names per account.
		// Check if there are any existing target groups with the same name.
		// If there are multiple, an error will be returned.
		instance, err := c.service.CheckDuplicateTargetGroup(ctx, cr.Spec.ForProvider.Name)
		if err != nil {
			return managed.ExternalCreation{}, err
		}
		targetGroupID, err := c.service.GetTargetGroupID(instance)
		if err != nil {
			return managed.ExternalCreation{}, err
		}
		if targetGroupID != "" {
			// "Import" existing target group.
			cr.Status.AtProvider.TargetGroupID = targetGroupID
			meta.SetExternalName(cr, targetGroupID)
			return managed.ExternalCreation{}, nil
		}
	}

	instanceInput, err := targetgroup.GenerateCreateTargetGroupInput(cr)
	if err != nil {
		return managed.ExternalCreation{}, err
	}
	newInstance, apiResponse, err := c.service.CreateTargetGroup(ctx, *instanceInput)
	creation := managed.ExternalCreation{ConnectionDetails: managed.ConnectionDetails{}}
	if err != nil {
		retErr := fmt.Errorf("failed to create target group. error: %w", err)
		return creation, compute.AddAPIResponseInfo(apiResponse, retErr)
	}
	if err = compute.WaitForRequest(ctx, c.service.GetAPIClient(), apiResponse); err != nil {
		return managed.ExternalCreation{}, err
	}
	// Set External Name
	cr.Status.AtProvider.TargetGroupID = *newInstance.Id
	meta.SetExternalName(cr, *newInstance.Id)
	return creation, nil
}

func (c *externalTargetGroup) Update(ctx context.Context, mg resource.Managed) (managed.ExternalUpdate, error) {
	cr, ok := mg.(*v1alpha1.TargetGroup)
	if !ok {
		return managed.ExternalUpdate{}, errors.New(errNotTargetGroup)
	}
	if cr.Status.AtProvider.State == compute.BUSY || cr.Status.AtProvider.State == compute.UPDATING {
		return managed.ExternalUpdate{}, nil
	}

	instanceInput, err := targetgroup.GenerateUpdateTargetGroupInput(cr)
	if err != nil {
		return managed.ExternalUpdate{}, err
	}
	_, apiResponse, err := c.service.UpdateTargetGroup(ctx, cr.Status.AtProvider.TargetGroupID, *instanceInput)
	if err != nil {
		retErr := fmt.Errorf("failed to update target group. error: %w", err)
		return managed.ExternalUpdate{}, compute.AddAPIResponseInfo(apiResponse, retErr)
	}
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
	if cr.Status.AtProvider.State == compute.DESTROYING {
		return nil
	}
	apiResponse, err := c.service.DeleteTargetGroup(ctx, cr.Status.AtProvider.TargetGroupID)
	if err != nil {
		retErr := fmt.Errorf("failed to delete target group. error: %w", err)
		return compute.ErrorUnlessNotFound(apiResponse, retErr)
	}
	if err = compute.WaitForRequest(ctx, c.service.GetAPIClient(), apiResponse); err != nil {
		return err
	}
	return nil
}
