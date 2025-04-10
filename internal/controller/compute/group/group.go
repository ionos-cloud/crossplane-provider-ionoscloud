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

package group

import (
	"context"
	"fmt"

	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	xpv1 "github.com/crossplane/crossplane-runtime/apis/common/v1"
	"github.com/crossplane/crossplane-runtime/pkg/event"
	"github.com/crossplane/crossplane-runtime/pkg/logging"
	"github.com/crossplane/crossplane-runtime/pkg/meta"
	"github.com/crossplane/crossplane-runtime/pkg/reconciler/managed"
	"github.com/crossplane/crossplane-runtime/pkg/resource"

	"github.com/ionos-cloud/crossplane-provider-ionoscloud/apis/compute/v1alpha1"
	apisv1alpha1 "github.com/ionos-cloud/crossplane-provider-ionoscloud/apis/v1alpha1"
	"github.com/ionos-cloud/crossplane-provider-ionoscloud/internal/clients"
	"github.com/ionos-cloud/crossplane-provider-ionoscloud/internal/clients/compute"
	"github.com/ionos-cloud/crossplane-provider-ionoscloud/internal/clients/compute/group"
	"github.com/ionos-cloud/crossplane-provider-ionoscloud/internal/utils"
)

const (
	errNotGroup                 = "managed resource is not a Group custom resource"
	eventCannotResolveReference = "CannotResolveReference"
)

// Setup adds a controller that reconciles Group managed resources.
func Setup(mgr ctrl.Manager, opts *utils.ConfigurationOptions) error {
	name := managed.ControllerName(v1alpha1.GroupGroupKind)
	r := event.NewAPIRecorder(mgr.GetEventRecorderFor(name))
	logger := opts.CtrlOpts.Logger

	return ctrl.NewControllerManagedBy(mgr).
		Named(name).
		WithOptions(opts.CtrlOpts.ForControllerRuntime()).
		WithEventFilter(resource.DesiredStateChanged()).
		For(&v1alpha1.Group{}).
		Complete(managed.NewReconciler(mgr,
			resource.ManagedKind(v1alpha1.GroupGroupVersionKind),
			managed.WithExternalConnecter(&connectorGroup{
				kube:                 mgr.GetClient(),
				usage:                resource.NewProviderConfigUsageTracker(mgr.GetClient(), &apisv1alpha1.ProviderConfigUsage{}),
				log:                  logger,
				isUniqueNamesEnabled: opts.GetIsUniqueNamesEnabled()}),
			managed.WithReferenceResolver(managed.NewAPISimpleReferenceResolver(mgr.GetClient())),
			managed.WithInitializers(resourceShareInitializer{kube: mgr.GetClient(), log: logger, eventRecorder: r}),
			managed.WithPollInterval(opts.GetPollInterval()),
			managed.WithTimeout(opts.GetTimeout()),
			managed.WithCreationGracePeriod(opts.GetCreationGracePeriod()),
			managed.WithLogger(logger.WithValues("controller", name)),
			managed.WithMetricRecorder(opts.CtrlOpts.MetricOptions.MRMetrics),
			managed.WithRecorder(r)))

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
	_, ok := mg.(*v1alpha1.Group)
	if !ok {
		return nil, errors.New(errNotGroup)
	}
	svc, err := clients.ConnectForCRD(ctx, mg, c.kube, c.usage)
	return &externalGroup{
		kube:                 c.kube,
		service:              &group.APIClient{IonosServices: svc},
		log:                  c.log,
		isUniqueNamesEnabled: c.isUniqueNamesEnabled}, err
}

// externalGroup observes, then either creates, updates or deletes a group in ionoscloud
// to ensure it reflects the desired state of the managed resource
type externalGroup struct {
	// service is the ionoscloud API client
	kube                 client.Client
	service              group.Client
	log                  logging.Logger
	isUniqueNamesEnabled bool
}

func (eg *externalGroup) Observe(ctx context.Context, mg resource.Managed) (managed.ExternalObservation, error) {
	cr, ok := mg.(*v1alpha1.Group)
	if !ok {
		return managed.ExternalObservation{}, errors.New(errNotGroup)
	}
	groupID := meta.GetExternalName(cr)
	if groupID == "" {
		return managed.ExternalObservation{}, nil
	}
	observed, apiResponse, err := eg.service.GetGroup(ctx, groupID)
	if err != nil {
		err = fmt.Errorf("failed to get group by ID: %w", err)
		return managed.ExternalObservation{}, compute.ErrorUnlessNotFound(apiResponse, err)
	}

	members, apiResponse, err := eg.service.GetGroupMembers(ctx, groupID)
	if err != nil {
		err = fmt.Errorf("failed to get group members: %w", err)
		return managed.ExternalObservation{}, compute.ErrorUnlessNotFound(apiResponse, err)
	}

	shares, apiResponse, err := eg.service.GetGroupResourceShares(ctx, groupID)
	if err != nil {
		err = fmt.Errorf("failed to get group shares: %w", err)
		return managed.ExternalObservation{}, compute.ErrorUnlessNotFound(apiResponse, err)
	}

	cr.Status.AtProvider.GroupID = groupID
	cr.Status.AtProvider.UserIDs = members
	cr.Status.AtProvider.ResourceShares = shares
	cr.SetConditions(xpv1.Available())
	return managed.ExternalObservation{
		ResourceExists:    true,
		ResourceUpToDate:  group.IsGroupUpToDate(cr, observed),
		ConnectionDetails: managed.ConnectionDetails{},
	}, nil
}

func (eg *externalGroup) Create(ctx context.Context, mg resource.Managed) (managed.ExternalCreation, error) {
	cr, ok := mg.(*v1alpha1.Group)
	if !ok {
		return managed.ExternalCreation{}, errors.New(errNotGroup)
	}
	cr.SetConditions(xpv1.Creating())
	// Group names should be unique per account
	// Multiple groups with the same name will trigger an error
	// If only one group exists with the same name, it will be "imported"
	if eg.isUniqueNamesEnabled {
		duplicateGroupID, err := eg.service.CheckDuplicateGroup(ctx, cr.Spec.ForProvider.Name)
		if err != nil {
			return managed.ExternalCreation{}, err
		}
		if duplicateGroupID != "" {
			cr.Status.AtProvider.GroupID = duplicateGroupID
			meta.SetExternalName(cr, duplicateGroupID)
			return managed.ExternalCreation{}, nil
		}
	}

	groupInput, memberIDs, resourceShares := group.GenerateCreateGroupInput(cr)
	newGroup, apiResponse, err := eg.service.CreateGroup(ctx, *groupInput)
	if err != nil {
		err = fmt.Errorf("failed to create new group. error: %w", err)
		return managed.ExternalCreation{}, compute.AddAPIResponseInfo(apiResponse, err)
	}
	if err = compute.WaitForRequest(ctx, eg.service.GetAPIClient(), apiResponse); err != nil {
		return managed.ExternalCreation{}, err
	}
	cr.Status.AtProvider.GroupID = *newGroup.Id
	meta.SetExternalName(cr, *newGroup.Id)

	if err = eg.service.UpdateGroupMembers(ctx, *newGroup.Id, group.MembersUpdateOp{Add: memberIDs}); err != nil {
		err = fmt.Errorf("error occurred while adding members at group creation: %w", err)
		return managed.ExternalCreation{}, err
	}

	if err = eg.service.UpdateGroupResourceShares(ctx, *newGroup.Id, group.SharesUpdateOp{Add: resourceShares}); err != nil {
		err = fmt.Errorf("error occurred while adding resource shares at group creation: %w", err)
		return managed.ExternalCreation{}, err
	}

	return managed.ExternalCreation{}, nil
}

func (eg *externalGroup) Update(ctx context.Context, mg resource.Managed) (managed.ExternalUpdate, error) {
	cr, ok := mg.(*v1alpha1.Group)
	if !ok {
		return managed.ExternalUpdate{}, errors.New(errNotGroup)
	}

	groupID := cr.Status.AtProvider.GroupID
	observedMemberIDs := cr.Status.AtProvider.UserIDs
	observedShares := cr.Status.AtProvider.ResourceShares
	groupInput, membersInput, sharesInput := group.GenerateUpdateGroupInput(cr, observedMemberIDs, observedShares)

	_, apiResponse, err := eg.service.UpdateGroup(ctx, groupID, *groupInput)
	if err != nil {
		err = fmt.Errorf("failed to update group. error: %w", err)
		return managed.ExternalUpdate{}, compute.AddAPIResponseInfo(apiResponse, err)
	}
	if err = compute.WaitForRequest(ctx, eg.service.GetAPIClient(), apiResponse); err != nil {
		return managed.ExternalUpdate{}, err
	}
	if err = eg.service.UpdateGroupMembers(ctx, groupID, membersInput); err != nil {
		err = fmt.Errorf("error occurred while updating group members: %w", err)
		return managed.ExternalUpdate{}, err
	}

	if err = eg.service.UpdateGroupResourceShares(ctx, groupID, sharesInput); err != nil {
		err = fmt.Errorf("error occurred while updating group resource shares: %w", err)
		return managed.ExternalUpdate{}, err
	}

	return managed.ExternalUpdate{}, nil
}

func (eg *externalGroup) Delete(ctx context.Context, mg resource.Managed) (managed.ExternalDelete, error) {
	cr, ok := mg.(*v1alpha1.Group)
	if !ok {
		return managed.ExternalDelete{}, errors.New(errNotGroup)
	}
	cr.SetConditions(xpv1.Deleting())

	apiResponse, err := eg.service.DeleteGroup(ctx, cr.Status.AtProvider.GroupID)
	if err != nil {
		err = fmt.Errorf("failed to delete group. error: %w", err)
		return managed.ExternalDelete{}, compute.ErrorUnlessNotFound(apiResponse, err)
	}
	if err = compute.WaitForRequest(ctx, eg.service.GetAPIClient(), apiResponse); err != nil {
		return managed.ExternalDelete{}, err
	}
	return managed.ExternalDelete{}, nil
}

// Initializers are called to initialize a Managed Resource
// before any External Client methods are called during a reconciliation loop cycle
//
// resourceShareInitializer initializes the Group MR by resolving resource share references
type resourceShareInitializer struct {
	kube          client.Client
	log           logging.Logger
	eventRecorder event.Recorder
}

// Initialize resolves and sets a ResourceID for resource share references which do not have one set directly
func (in resourceShareInitializer) Initialize(ctx context.Context, mg resource.Managed) error {

	cr, ok := mg.(*v1alpha1.Group)
	if !ok {
		return errors.New(errNotGroup)
	}

	for i, ref := range cr.Spec.ForProvider.ResourceShareCfg {
		if ref.ResourceID != "" {
			continue
		}
		u := unstructured.Unstructured{}
		u.SetGroupVersionKind(schema.GroupVersionKind{
			Version: ref.Version,
			Kind:    ref.Kind,
		})
		// We only log and emit a warning of the error instead of also returning it
		// to avoid blocking the Group reconciliation loop if any of the shared resources cannot be resolved
		if err := in.kube.Get(ctx, types.NamespacedName{Name: ref.Name}, &u); err != nil {
			msg := fmt.Errorf("unable to resolve shared resource reference: %w", err)
			in.log.Info(msg.Error())
			in.eventRecorder.Event(mg, event.Warning(eventCannotResolveReference, msg))
		} else {
			cr.Spec.ForProvider.ResourceShareCfg[i].ResourceID = meta.GetExternalName(&u)
		}
	}

	return nil
}

// Disconnect does nothing because there are no resources to release. Needs to be implemented starting from crossplane-runtime v0.17
func (eg *externalGroup) Disconnect(_ context.Context) error {
	return nil
}
