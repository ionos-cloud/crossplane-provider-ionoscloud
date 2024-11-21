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

package user

import (
	"context"

	"github.com/crossplane/crossplane-runtime/pkg/connection"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	ionosdk "github.com/ionos-cloud/sdk-go/v6"

	"github.com/pkg/errors"

	userapi "github.com/ionos-cloud/crossplane-provider-ionoscloud/internal/clients/compute/user"

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
	"github.com/ionos-cloud/crossplane-provider-ionoscloud/internal/features"
	"github.com/ionos-cloud/crossplane-provider-ionoscloud/internal/utils"
)

const (
	errUserObserve   = "failed to get user by id"
	errUserDelete    = "failed to delete user"
	errUserUpdate    = "failed to update user"
	errUserCreate    = "failed to create user"
	errGetUserGroups = "failed to fetch user groups"
	errNotUser       = "managed resource is not of a User type"
)

// Setup adds a controller that reconciles User managed resources.
func Setup(mgr ctrl.Manager, opts *utils.ConfigurationOptions) error {
	name := managed.ControllerName(v1alpha1.UserGroupKind)
	logger := opts.CtrlOpts.Logger

	cps := []managed.ConnectionPublisher{managed.NewAPISecretPublisher(mgr.GetClient(), mgr.GetScheme())}
	if opts.CtrlOpts.Features.Enabled(features.EnableAlphaExternalSecretStores) {
		cps = append(cps, connection.NewDetailsManager(mgr.GetClient(), apisv1alpha1.StoreConfigGroupVersionKind))
	}

	return ctrl.NewControllerManagedBy(mgr).
		Named(name).
		WithOptions(opts.CtrlOpts.ForControllerRuntime()).
		For(&v1alpha1.User{}).
		Complete(managed.NewReconciler(mgr,
			resource.ManagedKind(v1alpha1.UserGroupVersionKind),
			managed.WithExternalConnecter(&connectorUser{
				kube:  mgr.GetClient(),
				usage: resource.NewProviderConfigUsageTracker(mgr.GetClient(), &apisv1alpha1.ProviderConfigUsage{}),
				log:   logger,
			}),
			managed.WithReferenceResolver(managed.NewAPISimpleReferenceResolver(mgr.GetClient())),
			managed.WithInitializers(),
			managed.WithPollInterval(opts.GetPollInterval()),
			managed.WithTimeout(opts.GetTimeout()),
			managed.WithCreationGracePeriod(opts.GetCreationGracePeriod()),
			managed.WithLogger(logger.WithValues("controller", name)),
			managed.WithRecorder(event.NewAPIRecorder(mgr.GetEventRecorderFor(name))),
			managed.WithConnectionPublishers(cps...),
		))
}

// A connectorUser is expected to produce an ExternalClient when its Connect method
// is called.
type connectorUser struct {
	kube  client.Client
	usage resource.Tracker
	log   logging.Logger
}

// Connect typically produces an ExternalClient by:
// 1. Tracking that the managed resource is using a ProviderConfig.
// 2. Getting the managed resource's ProviderConfig.
// 3. Getting the credentials specified by the ProviderConfig.
// 4. Using the credentials to form a client.
func (c *connectorUser) Connect(ctx context.Context, mg resource.Managed) (managed.ExternalClient, error) {
	_, ok := mg.(*v1alpha1.User)
	if !ok {
		return nil, errors.New(errNotUser)
	}
	svc, err := clients.ConnectForCRD(ctx, mg, c.kube, c.usage)
	if err != nil {
		return nil, err
	}
	return &externalUser{
		service: userapi.NewAPIClient(svc, compute.WaitForRequest),
		log:     c.log,
	}, err
}

// externalUser observes, then either creates, updates, or deletes a
// user in the cloud api. it ensures it reflects a desired state.
type externalUser struct {
	// service is the ionos cloud api client to manage users.
	// see https://api.ionos.com/docs/cloud/v6/#tag/User-management
	service userapi.Client
	log     logging.Logger
}

func connectionDetails(cr *v1alpha1.User, observed ionosdk.User) managed.ConnectionDetails {
	var details = make(managed.ConnectionDetails)
	props := observed.GetProperties()
	details["email"] = []byte(utils.DereferenceOrZero(props.GetEmail()))
	if cr.Spec.ForProvider.Password != "" {
		pw := cr.Spec.ForProvider.Password
		// passwords are sensitive thus should not be part of the cr
		// they are stored as a secret in the connection details.
		cr.Spec.ForProvider.Password = ""
		details[xpv1.ResourceCredentialsSecretPasswordKey] = []byte(pw)
	}

	return details
}

func setStatus(cr *v1alpha1.User, observed ionosdk.User, groupIDs []string) {
	props := observed.GetProperties()
	if !observed.HasProperties() {
		return
	}

	cr.Status.AtProvider.UserID = utils.DereferenceOrZero(observed.GetId())
	cr.Status.AtProvider.Active = utils.DereferenceOrZero(props.GetActive())
	cr.Status.AtProvider.S3CanonicalUserID = utils.DereferenceOrZero(props.GetS3CanonicalUserId())
	cr.Status.AtProvider.SecAuthActive = utils.DereferenceOrZero(props.GetSecAuthActive())
	cr.Status.AtProvider.GroupIDs = groupIDs
}

func (eu *externalUser) Observe(ctx context.Context, mg resource.Managed) (managed.ExternalObservation, error) {
	cr, ok := mg.(*v1alpha1.User)
	if !ok {
		return managed.ExternalObservation{}, errors.New(errNotUser)
	}

	userID := meta.GetExternalName(cr)
	if userID == "" {
		return managed.ExternalObservation{}, nil
	}

	observed, resp, err := eu.service.GetUser(ctx, userID)
	if resp.HttpNotFound() {
		return managed.ExternalObservation{}, nil
	}
	if err != nil {
		return managed.ExternalObservation{}, errors.Wrap(err, errUserObserve)
	}

	groupIDs, err := eu.service.GetUserGroups(ctx, userID)
	if err != nil {
		return managed.ExternalObservation{ResourceExists: true}, errors.Wrap(err, errGetUserGroups)
	}

	setStatus(cr, observed, groupIDs)
	cr.SetConditions(xpv1.Available())

	linit := cr.Spec.ForProvider.Password != ""
	conn := connectionDetails(cr, observed)

	return managed.ExternalObservation{
		ResourceExists:          true,
		ResourceUpToDate:        userapi.IsUserUpToDate(cr.Spec.ForProvider, observed, groupIDs),
		ConnectionDetails:       conn,
		ResourceLateInitialized: linit,
	}, nil
}

func (eu *externalUser) Create(ctx context.Context, mg resource.Managed) (managed.ExternalCreation, error) {
	cr, ok := mg.(*v1alpha1.User)
	if !ok {
		return managed.ExternalCreation{}, errors.Wrap(errors.New(errNotUser), "create error")
	}
	cr.SetConditions(xpv1.Creating())

	observed, resp, err := eu.service.CreateUser(ctx, cr.Spec.ForProvider)
	if err != nil {
		werr := errors.Wrap(err, errUserCreate)
		return managed.ExternalCreation{}, compute.AddAPIResponseInfo(resp, werr)
	}

	meta.SetExternalName(cr, *observed.GetId())
	conn := connectionDetails(cr, observed)

	err = eu.service.UpdateUserGroups(ctx, *observed.GetId(), nil, cr.Spec.ForProvider.GroupIDs)
	if err != nil {
		return managed.ExternalCreation{ConnectionDetails: conn}, err
	}

	setStatus(cr, observed, cr.Spec.ForProvider.GroupIDs)

	return managed.ExternalCreation{ConnectionDetails: conn}, nil
}

func (eu *externalUser) Update(ctx context.Context, mg resource.Managed) (managed.ExternalUpdate, error) {
	cr, ok := mg.(*v1alpha1.User)
	if !ok {
		return managed.ExternalUpdate{}, errors.Wrap(errors.New(errNotUser), "update error")
	}

	userID := cr.Status.AtProvider.UserID

	observed, resp, err := eu.service.UpdateUser(ctx, userID, cr.Spec.ForProvider)
	if err != nil {
		werr := errors.Wrap(err, errUserUpdate)
		return managed.ExternalUpdate{}, compute.AddAPIResponseInfo(resp, werr)
	}
	conn := connectionDetails(cr, observed)

	err = eu.service.UpdateUserGroups(ctx, userID, cr.Status.AtProvider.GroupIDs, cr.Spec.ForProvider.GroupIDs)
	if err != nil {
		return managed.ExternalUpdate{ConnectionDetails: conn}, err
	}

	return managed.ExternalUpdate{ConnectionDetails: conn}, nil
}

func (eu *externalUser) Delete(ctx context.Context, mg resource.Managed) (managed.ExternalDelete, error) {
	user, ok := mg.(*v1alpha1.User)
	if !ok {
		return managed.ExternalDelete{}, errors.Wrap(errors.New(errNotUser), "delete error")
	}
	user.SetConditions(xpv1.Deleting())

	userID := user.Status.AtProvider.UserID
	resp, err := eu.service.DeleteUser(ctx, userID)
	return managed.ExternalDelete{}, compute.ErrorUnlessNotFound(resp, errors.Wrap(err, errUserDelete))
}

// Disconnect does nothing because there are no resources to release. Needs to be implemented starting from crossplane-runtime v0.17
func (eu *externalUser) Disconnect(_ context.Context) error {
	return nil
}
