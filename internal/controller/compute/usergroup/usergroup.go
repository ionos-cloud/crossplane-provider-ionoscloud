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

package usergroup

import (
	"cmp"
	"context"
	ionosdk "github.com/ionos-cloud/sdk-go/v6"
	"k8s.io/client-go/util/workqueue"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"slices"

	usergroupapi "github.com/ionos-cloud/crossplane-provider-ionoscloud/internal/clients/compute/usergroup"

	xpv1 "github.com/crossplane/crossplane-runtime/apis/common/v1"
	"github.com/crossplane/crossplane-runtime/pkg/event"
	"github.com/crossplane/crossplane-runtime/pkg/logging"
	"github.com/crossplane/crossplane-runtime/pkg/meta"
	"github.com/crossplane/crossplane-runtime/pkg/ratelimiter"
	"github.com/crossplane/crossplane-runtime/pkg/reconciler/managed"
	"github.com/crossplane/crossplane-runtime/pkg/resource"
	"github.com/pkg/errors"

	"github.com/ionos-cloud/crossplane-provider-ionoscloud/apis/compute/v1alpha1"
	apisv1alpha1 "github.com/ionos-cloud/crossplane-provider-ionoscloud/apis/v1alpha1"
	"github.com/ionos-cloud/crossplane-provider-ionoscloud/internal/clients"
	"github.com/ionos-cloud/crossplane-provider-ionoscloud/internal/clients/compute"
	"github.com/ionos-cloud/crossplane-provider-ionoscloud/internal/utils"
)

const (
	errUserGroupObserve = "failed to get user group by id"
	errUserGroupDelete  = "failed to delete user group"
	errUserGroupUpdate  = "failed to update user group"
	errUserGroupCreate  = "failed to create user group"
	errAddUserToGroup   = "failed to add user to the group id"
	errGetUserGroups    = "failed to fetch user groups"
	errRequestWait      = "error waiting for request"
	errNotUserGroup     = "managed resource is not of a UserGroup type"
)

// A connectorUserGroup is expected to produce an ExternalClient when its Connect method
// is called.
type connectorUserGroup struct {
	kube  client.Client
	usage resource.Tracker
	log   logging.Logger
}

// Connect typically produces an ExternalClient by:
// 1. Tracking that the managed resource is using a ProviderConfig.
// 2. Getting the managed resource's ProviderConfig.
// 3. Getting the credentials specified by the ProviderConfig.
// 4. Using the credentials to form a client.
func (c *connectorUserGroup) Connect(ctx context.Context, mg resource.Managed) (managed.ExternalClient, error) {
	svc, err := clients.ConnectForCRD(ctx, mg, c.kube, c.usage)
	return &externalUserGroup{
		service: usergroupapi.NewAPIClient(svc, compute.WaitForRequest),
		log:     c.log,
	}, err
}

// Setup adds a controller that reconciles UserGroup managed resources.
func Setup(mgr ctrl.Manager, l logging.Logger, _ workqueue.RateLimiter, opts *utils.ConfigurationOptions) error {
	name := managed.ControllerName(v1alpha1.UserGroupGroupKind)

	return ctrl.NewControllerManagedBy(mgr).
		Named(name).
		WithOptions(controller.Options{
			RateLimiter: ratelimiter.NewController(),
		}).
		For(&v1alpha1.UserGroup{}).
		Complete(managed.NewReconciler(mgr,
			resource.ManagedKind(v1alpha1.UserGroupGroupVersionKind),
			managed.WithExternalConnecter(&connectorUserGroup{
				kube:  mgr.GetClient(),
				usage: resource.NewProviderConfigUsageTracker(mgr.GetClient(), &apisv1alpha1.ProviderConfigUsage{}),
				log:   l,
			}),
			managed.WithReferenceResolver(managed.NewAPISimpleReferenceResolver(mgr.GetClient())),
			managed.WithInitializers(),
			managed.WithPollInterval(opts.GetPollInterval()),
			managed.WithTimeout(opts.GetTimeout()),
			managed.WithCreationGracePeriod(opts.GetCreationGracePeriod()),
			managed.WithLogger(l.WithValues("controller", name)),
			managed.WithRecorder(event.NewAPIRecorder(mgr.GetEventRecorderFor(name)))))
}

// externalUser observes, then either creates, updates, or deletes a
// user in the cloud api. it ensures it reflects a desired state.
type externalUserGroup struct {
	// service is the ionos cloud api client to manage users.
	// see https://api.ionos.com/docs/cloud/v6/#tag/User-management
	service usergroupapi.Client
	log     logging.Logger
}

func (eu *externalUserGroup) connectionDetails(cr *v1alpha1.UserGroup, observed ionosdk.Group) managed.ConnectionDetails {
	var details = make(managed.ConnectionDetails)
	props := observed.GetProperties()
	if props == nil {
		eu.log.Info("IONOS group properties is nil")
		return details
	}
	details["name"] = []byte(utils.DereferenceOrZero(props.GetName()))

	return details
}

func setStatus(cr *v1alpha1.UserGroup, observed ionosdk.Group) {
	cr.Status.AtProvider.UserGroupID = utils.DereferenceOrZero(observed.GetId())
}

func (eu *externalUserGroup) Observe(ctx context.Context, mg resource.Managed) (managed.ExternalObservation, error) {
	cr, ok := mg.(*v1alpha1.UserGroup)
	if !ok {
		return managed.ExternalObservation{}, errors.New(errNotUserGroup)
	}

	userGroupID := meta.GetExternalName(cr)
	if userGroupID == "" {
		return managed.ExternalObservation{}, nil
	}

	observed, resp, err := eu.service.GetGroup(ctx, userGroupID)
	if resp.HttpNotFound() {
		return managed.ExternalObservation{}, nil
	}
	if err != nil {
		return managed.ExternalObservation{}, errors.Wrap(err, errUserGroupObserve)
	}

	setStatus(cr, observed)
	cr.SetConditions(xpv1.Available())

	conn := eu.connectionDetails(cr, observed)

	return managed.ExternalObservation{
		ResourceExists:          true,
		ResourceUpToDate:        isUserUpToDate(cr.Spec.ForProvider, observed),
		ConnectionDetails:       conn,
		ResourceLateInitialized: false,
	}, nil
}

func (eu *externalUserGroup) Create(ctx context.Context, mg resource.Managed) (managed.ExternalCreation, error) {
	cr, ok := mg.(*v1alpha1.UserGroup)
	if !ok {
		return managed.ExternalCreation{}, errors.Wrap(errors.New(errNotUserGroup), "create error")
	}
	cr.SetConditions(xpv1.Creating())

	observed, resp, err := eu.service.CreateGroup(ctx, cr.Spec.ForProvider)
	if err != nil {
		werr := errors.Wrap(err, errUserGroupCreate)
		return managed.ExternalCreation{}, compute.AddAPIResponseInfo(resp, werr)
	}

	meta.SetExternalName(cr, *observed.GetId())
	conn := eu.connectionDetails(cr, observed)

	setStatus(cr, observed)

	return managed.ExternalCreation{ConnectionDetails: conn}, nil
}

func (eu *externalUserGroup) Update(ctx context.Context, mg resource.Managed) (managed.ExternalUpdate, error) {
	cr, ok := mg.(*v1alpha1.UserGroup)
	if !ok {
		return managed.ExternalUpdate{}, errors.Wrap(errors.New(errNotUserGroup), "update error")
	}

	groupID := cr.Status.AtProvider.UserGroupID

	observed, resp, err := eu.service.UpdateGroup(ctx, groupID, cr.Spec.ForProvider)
	if err != nil {
		werr := errors.Wrap(err, errUserGroupUpdate)
		return managed.ExternalUpdate{}, compute.AddAPIResponseInfo(resp, werr)
	}
	conn := eu.connectionDetails(cr, observed)

	return managed.ExternalUpdate{ConnectionDetails: conn}, nil
}

func (eu *externalUserGroup) Delete(ctx context.Context, mg resource.Managed) error {
	user, ok := mg.(*v1alpha1.UserGroup)
	if !ok {
		return errors.Wrap(errors.New(errNotUserGroup), "delete error")
	}
	user.SetConditions(xpv1.Deleting())

	groupID := user.Status.AtProvider.UserGroupID

	resp, err := eu.service.DeleteGroup(ctx, groupID)
	return compute.ErrorUnlessNotFound(resp, errors.Wrap(err, errUserGroupDelete))
}

// isUserUpToDate returns true if the User is up-to-date or false otherwise.
func isUserUpToDate(params v1alpha1.GroupParameters, observed ionosdk.Group) bool { //nolint:gocyclo
	if !observed.HasProperties() {
		return false
	}

	props := observed.GetProperties()
	name := props.GetName()

	switch {
	case name != nil && params.Name != *name:
		return false
	}

	return true
}

func isSetEqual[T cmp.Ordered](sl0, sl1 []T) bool {
	if len(sl0) != len(sl1) {
		return false
	}

	s0, s1 := slices.Clone(sl0), slices.Clone(sl1)
	slices.Sort(s0)
	slices.Sort(s1)

	return slices.Equal(s0, s1)
}
