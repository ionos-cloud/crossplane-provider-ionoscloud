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

package managementgroup

import (
	"context"
	"fmt"

	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/util/sets"

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
	"github.com/ionos-cloud/crossplane-provider-ionoscloud/internal/clients/compute/managementgroup"
	"github.com/ionos-cloud/crossplane-provider-ionoscloud/internal/utils"
)

const errNotGroup = "managed resource is not a ManagementGroup custom resource"

// Setup adds a controller that reconciles Group managed resources.
func Setup(mgr ctrl.Manager, l logging.Logger, _ workqueue.RateLimiter, opts *utils.ConfigurationOptions) error {
	name := managed.ControllerName(v1alpha1.ManagementGroupKind)

	return ctrl.NewControllerManagedBy(mgr).
		Named(name).
		WithOptions(controller.Options{
			RateLimiter: ratelimiter.NewController(),
		}).
		For(&v1alpha1.ManagementGroup{}).
		Complete(managed.NewReconciler(mgr,
			resource.ManagedKind(v1alpha1.ManagementGroupGroupVersionKind),
			managed.WithExternalConnecter(&connectorGroup{
				kube:                 mgr.GetClient(),
				usage:                resource.NewProviderConfigUsageTracker(mgr.GetClient(), &apisv1alpha1.ProviderConfigUsage{}),
				log:                  l,
				isUniqueNamesEnabled: opts.GetIsUniqueNamesEnabled()}),
			managed.WithReferenceResolver(managed.NewAPISimpleReferenceResolver(mgr.GetClient())),
			managed.WithInitializers(),
			managed.WithPollInterval(opts.GetPollInterval()),
			managed.WithTimeout(opts.GetTimeout()),
			managed.WithCreationGracePeriod(opts.GetCreationGracePeriod()),
			managed.WithLogger(l.WithValues("controller", name)),
			managed.WithRecorder(event.NewAPIRecorder(mgr.GetEventRecorderFor(name)))))

}

// A connectorUser is expected to produce an ExternalClient when its Connect method is called.
type connectorGroup struct {
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
func (c *connectorGroup) Connect(ctx context.Context, mg resource.Managed) (managed.ExternalClient, error) {
	_, ok := mg.(*v1alpha1.ManagementGroup)
	if !ok {
		return nil, errors.New(errNotGroup)
	}
	svc, err := clients.ConnectForCRD(ctx, mg, c.kube, c.usage)
	return &externalGroup{
		service:              &managementgroup.APIClient{IonosServices: svc},
		log:                  c.log,
		isUniqueNamesEnabled: c.isUniqueNamesEnabled}, err
}

// externalGroup observes, then either creates, updates or deletes a group in ionoscloud
// to ensure it reflects the desired state of the managed resource
type externalGroup struct {
	// service is the ionoscloud API client (https://api.ionos.com/docs/cloud/v6/#tag/User-management)
	service              managementgroup.Client
	log                  logging.Logger
	isUniqueNamesEnabled bool
}

func (eg *externalGroup) Observe(ctx context.Context, mg resource.Managed) (managed.ExternalObservation, error) {
	cr, ok := mg.(*v1alpha1.ManagementGroup)
	if !ok {
		return managed.ExternalObservation{}, errors.New(errNotGroup)
	}
	groupID := meta.GetExternalName(cr)
	if groupID == "" {
		return managed.ExternalObservation{}, nil
	}
	observed, apiResponse, err := eg.service.GetGroup(ctx, groupID)
	if err != nil {
		err = fmt.Errorf("failed to get management group by ID: %w", err)
		return managed.ExternalObservation{}, compute.ErrorUnlessNotFound(apiResponse, err)
	}

	members, apiResponse, err := eg.service.GetGroupMembers(ctx, groupID)
	if err != nil {
		err = fmt.Errorf("failed to get management group members: %w", err)
		return managed.ExternalObservation{}, compute.ErrorUnlessNotFound(apiResponse, err)
	}

	cr.Status.AtProvider.ManagementGroupID = groupID
	cr.Status.AtProvider.UserIDs = members
	cr.SetConditions(xpv1.Available())
	return managed.ExternalObservation{
		ResourceExists:    true,
		ResourceUpToDate:  managementgroup.IsManagementGroupUpToDate(cr, observed, sets.New[string](members...)),
		ConnectionDetails: managed.ConnectionDetails{},
	}, nil
}

func (eg *externalGroup) Create(ctx context.Context, mg resource.Managed) (managed.ExternalCreation, error) {
	cr, ok := mg.(*v1alpha1.ManagementGroup)
	if !ok {
		return managed.ExternalCreation{}, errors.New(errNotGroup)
	}
	cr.SetConditions(xpv1.Creating())
	// Group names should be unique per account
	// Multiple groups with the same name will trigger an error
	// If only one group exists with the same name, it will be "imported"
	if eg.isUniqueNamesEnabled {
		group, err := eg.service.CheckDuplicateGroup(ctx, cr.Spec.ForProvider.Name)
		if err != nil {
			return managed.ExternalCreation{}, err
		}
		groupID, err := eg.service.GetGroupID(group)
		if err != nil {
			return managed.ExternalCreation{}, err
		}
		if groupID != "" {
			cr.Status.AtProvider.ManagementGroupID = groupID
			meta.SetExternalName(cr, groupID)
			return managed.ExternalCreation{}, nil
		}
	}

	groupInput, memberIDs := managementgroup.GenerateCreateGroupInput(cr)
	newGroup, apiResponse, err := eg.service.CreateGroup(ctx, *groupInput)
	if err != nil {
		err = fmt.Errorf("failed to create new management group. error: %w", err)
		return managed.ExternalCreation{}, compute.AddAPIResponseInfo(apiResponse, err)
	}
	if err = compute.WaitForRequest(ctx, eg.service.GetAPIClient(), apiResponse); err != nil {
		return managed.ExternalCreation{}, err
	}
	cr.Status.AtProvider.ManagementGroupID = *newGroup.Id
	meta.SetExternalName(cr, *newGroup.Id)

	if err = eg.service.UpdateGroupMembers(ctx, *newGroup.Id, memberIDs, eg.service.AddGroupMember); err != nil {
		err = fmt.Errorf("error occurred while adding members at group creation: %w", err)
		return managed.ExternalCreation{}, err
	}

	return managed.ExternalCreation{}, nil
}

func (eg *externalGroup) Update(ctx context.Context, mg resource.Managed) (managed.ExternalUpdate, error) {
	cr, ok := mg.(*v1alpha1.ManagementGroup)
	if !ok {
		return managed.ExternalUpdate{}, errors.New(errNotGroup)
	}

	groupID := cr.Status.AtProvider.ManagementGroupID
	groupInput, addMemberIDs, delMemberIDs := managementgroup.GenerateUpdateGroupInput(cr, sets.New[string](cr.Status.AtProvider.UserIDs...))

	_, apiResponse, err := eg.service.UpdateGroup(ctx, groupID, *groupInput)
	if err != nil {
		err = fmt.Errorf("failed to update management group. error: %w", err)
		return managed.ExternalUpdate{}, compute.AddAPIResponseInfo(apiResponse, err)
	}
	if err = compute.WaitForRequest(ctx, eg.service.GetAPIClient(), apiResponse); err != nil {
		return managed.ExternalUpdate{}, err
	}
	if err = eg.service.UpdateGroupMembers(ctx, groupID, addMemberIDs, eg.service.AddGroupMember); err != nil {
		err = fmt.Errorf("error occurred while adding members at group update: %w", err)
		return managed.ExternalUpdate{}, err
	}

	if err = eg.service.UpdateGroupMembers(ctx, groupID, delMemberIDs, eg.service.RemoveGroupMember); err != nil {
		err = fmt.Errorf("error occurred while removing members at group update: %w", err)
		return managed.ExternalUpdate{}, err
	}

	return managed.ExternalUpdate{}, nil
}

func (eg *externalGroup) Delete(ctx context.Context, mg resource.Managed) error {
	cr, ok := mg.(*v1alpha1.ManagementGroup)
	if !ok {
		return errors.New(errNotGroup)
	}
	cr.SetConditions(xpv1.Deleting())

	apiResponse, err := eg.service.DeleteGroup(ctx, cr.Status.AtProvider.ManagementGroupID)
	if err != nil {
		err = fmt.Errorf("failed to delete management group. error: %w", err)
		return compute.ErrorUnlessNotFound(apiResponse, err)
	}
	if err = compute.WaitForRequest(ctx, eg.service.GetAPIClient(), apiResponse); err != nil {
		return err
	}
	return nil
}
