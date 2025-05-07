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
	"fmt"

	xpv1 "github.com/crossplane/crossplane-runtime/apis/common/v1"
	"github.com/crossplane/crossplane-runtime/pkg/connection"
	"github.com/crossplane/crossplane-runtime/pkg/event"
	"github.com/crossplane/crossplane-runtime/pkg/logging"
	"github.com/crossplane/crossplane-runtime/pkg/meta"
	"github.com/crossplane/crossplane-runtime/pkg/reconciler/managed"
	"github.com/crossplane/crossplane-runtime/pkg/resource"
	ionosdk "github.com/ionos-cloud/sdk-go/v6"
	"github.com/pkg/errors"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	userapi "github.com/ionos-cloud/crossplane-provider-ionoscloud/internal/clients/compute/user"

	"github.com/ionos-cloud/crossplane-provider-ionoscloud/apis/compute/v1alpha1"
	apisv1alpha1 "github.com/ionos-cloud/crossplane-provider-ionoscloud/apis/v1alpha1"
	"github.com/ionos-cloud/crossplane-provider-ionoscloud/internal/clients"
	"github.com/ionos-cloud/crossplane-provider-ionoscloud/internal/clients/compute"
	"github.com/ionos-cloud/crossplane-provider-ionoscloud/internal/features"
	"github.com/ionos-cloud/crossplane-provider-ionoscloud/internal/utils"
)

const (
	errUserObserve                             = "failed to get user by id"
	errUserDelete                              = "failed to delete user"
	errUserUpdate                              = "failed to update user"
	errUserCreate                              = "failed to create user"
	errGetUserGroups                           = "failed to fetch user groups"
	errGetCredentialsSecret                    = "cannot get credentials secret"
	errNotUser                                 = "managed resource is not of a User type"
	warningCannotResolveReference event.Reason = "CannotResolveReference"
	warningCannotResolveKey       event.Reason = "CannotResolveKey"
	warningDeprecatedField        event.Reason = "DeprecatedField"
)

// Setup adds a controller that reconciles User managed resources.
func Setup(mgr ctrl.Manager, opts *utils.ConfigurationOptions) error {
	name := managed.ControllerName(v1alpha1.UserGroupKind)
	logger := opts.CtrlOpts.Logger

	cps := []managed.ConnectionPublisher{managed.NewAPISecretPublisher(mgr.GetClient(), mgr.GetScheme())}
	if opts.CtrlOpts.Features.Enabled(features.EnableAlphaExternalSecretStores) {
		cps = append(cps, connection.NewDetailsManager(mgr.GetClient(), apisv1alpha1.StoreConfigGroupVersionKind))
	}
	eventRecorder := event.NewAPIRecorder(mgr.GetEventRecorderFor(name))

	return ctrl.NewControllerManagedBy(mgr).
		Named(name).
		WithOptions(opts.CtrlOpts.ForControllerRuntime()).
		WithEventFilter(resource.DesiredStateChanged()).
		For(&v1alpha1.User{}).
		Complete(
			managed.NewReconciler(
				mgr,
				resource.ManagedKind(v1alpha1.UserGroupVersionKind),
				managed.WithExternalConnecter(
					&connectorUser{
						kube: mgr.GetClient(),
						usage: resource.NewProviderConfigUsageTracker(
							mgr.GetClient(), &apisv1alpha1.ProviderConfigUsage{},
						),
						log: logger,
					},
				),
				managed.WithReferenceResolver(managed.NewAPISimpleReferenceResolver(mgr.GetClient())),
				managed.WithInitializers(resourceInitializer{kube: mgr.GetClient(), eventRecorder: eventRecorder}),
				managed.WithPollInterval(opts.GetPollInterval()),
				managed.WithTimeout(opts.GetTimeout()),
				managed.WithCreationGracePeriod(opts.GetCreationGracePeriod()),
				managed.WithLogger(logger.WithValues("controller", name)),
				managed.WithRecorder(eventRecorder),
				managed.WithConnectionPublishers(cps...),
			),
		)
}

// resourceInitializer is intended to pre-check the user managed resource.
type resourceInitializer struct {
	kube          client.Client
	eventRecorder event.Recorder
}

// Initialize will check if the user's credentials secret exists.
func (ri resourceInitializer) Initialize(ctx context.Context, mg resource.Managed) error {
	user := mg.(*v1alpha1.User)
	if user.DeletionTimestamp != nil {
		return nil
	}

	if user.Spec.ForProvider.Password != "" {
		err := errors.New("spec.ForProvider.Password is deprecated, please use spec.ForProvider.PasswordSecretRef.")
		ri.eventRecorder.Event(user, event.Warning(warningDeprecatedField, err))
	}

	if !user.HasCredentialsSecretRef() {
		return nil
	}

	secret, err := getPasswordSecret(ctx, ri.kube, user.Spec.ForProvider.PasswordSecretRef)
	if err != nil {
		ri.eventRecorder.Event(user, event.Warning(warningCannotResolveReference, err))
		return nil
	}
	key := user.Spec.ForProvider.PasswordSecretRef.Key
	if _, ok := secret.Data[key]; !ok {
		err := fmt.Errorf("credentials secret key %q not found in secret", key)
		ri.eventRecorder.Event(user, event.Warning(warningCannotResolveKey, err))
	}
	return nil
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
		client:  c.kube,
	}, err
}

// externalUser observes, then either creates, updates, or deletes a
// user in the cloud api. it ensures it reflects a desired state.
type externalUser struct {
	// service is the ionos cloud api client to manage users.
	// see https://api.ionos.com/docs/cloud/v6/#tag/User-management
	service userapi.Client
	log     logging.Logger
	client  client.Client
}

func getPasswordSecret(ctx context.Context, c client.Client, selector xpv1.SecretKeySelector) (*v1.Secret, error) {
	secret := &v1.Secret{}
	key := types.NamespacedName{
		Namespace: selector.Namespace,
		Name:      selector.Name,
	}
	if err := c.Get(ctx, key, secret); err != nil {
		return nil, errors.Wrap(err, errGetCredentialsSecret)
	}
	return secret, nil
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

	isUserUpToDate := userapi.IsUserUpToDate(cr.Spec.ForProvider, observed, groupIDs)
	if cr.HasCredentialsSecretRef() {
		secret, err := getPasswordSecret(ctx, eu.client, cr.Spec.ForProvider.PasswordSecretRef)
		if err != nil {
			return managed.ExternalObservation{}, errors.Wrap(err, errUserObserve)
		}
		if cr.Status.AtProvider.CredentialsVersion != secret.GetResourceVersion() {
			isUserUpToDate = false
			conn[xpv1.ResourceCredentialsSecretPasswordKey] = secret.Data[cr.Spec.ForProvider.PasswordSecretRef.Key]
			cr.Status.AtProvider.CredentialsVersion = secret.GetResourceVersion()
		}
	}

	return managed.ExternalObservation{
		ResourceExists:          true,
		ResourceUpToDate:        isUserUpToDate,
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

	// Deprecated: this functionality is deprecated as of v1.1.10
	passw := cr.Spec.ForProvider.Password

	if cr.HasCredentialsSecretRef() {
		secret, err := getPasswordSecret(ctx, eu.client, cr.Spec.ForProvider.PasswordSecretRef)
		if err != nil {
			return managed.ExternalCreation{}, errors.Wrap(err, errUserCreate)
		}
		passw = string(secret.Data[cr.Spec.ForProvider.PasswordSecretRef.Key])
		cr.Status.AtProvider.CredentialsVersion = secret.GetResourceVersion()
	}

	observed, resp, err := eu.service.CreateUser(ctx, cr.Spec.ForProvider, passw)
	if err != nil {
		werr := errors.Wrap(err, errUserCreate)
		return managed.ExternalCreation{}, compute.AddAPIResponseInfo(resp, werr)
	}

	meta.SetExternalName(cr, *observed.GetId())
	conn := connectionDetails(cr, observed)
	if cr.HasCredentialsSecretRef() {
		conn[xpv1.ResourceCredentialsSecretPasswordKey] = []byte(passw)
	}

	if cr.Spec.ForProvider.GroupIDs == nil {
		setStatus(cr, observed, []string{})
		return managed.ExternalCreation{ConnectionDetails: conn}, nil
	}

	eu.log.Info("Checking user groups value", "groups", *cr.Spec.ForProvider.GroupIDs)
	err = eu.service.UpdateUserGroups(ctx, *observed.GetId(), nil, cr.Spec.ForProvider.GroupIDs)
	if err != nil {
		return managed.ExternalCreation{ConnectionDetails: conn}, err
	}

	setStatus(cr, observed, *cr.Spec.ForProvider.GroupIDs)

	return managed.ExternalCreation{ConnectionDetails: conn}, nil
}

func (eu *externalUser) Update(ctx context.Context, mg resource.Managed) (managed.ExternalUpdate, error) {
	cr, ok := mg.(*v1alpha1.User)
	if !ok {
		return managed.ExternalUpdate{}, errors.Wrap(errors.New(errNotUser), "update error")
	}

	userID := cr.Status.AtProvider.UserID

	// Deprecated: this functionality is deprecated as of v1.1.10
	passw := cr.Spec.ForProvider.Password

	if cr.HasCredentialsSecretRef() {
		secret, err := getPasswordSecret(ctx, eu.client, cr.Spec.ForProvider.PasswordSecretRef)
		if err != nil {
			return managed.ExternalUpdate{}, errors.Wrap(err, errUserUpdate)
		}
		passw = string(secret.Data[cr.Spec.ForProvider.PasswordSecretRef.Key])
	}

	observed, resp, err := eu.service.UpdateUser(ctx, userID, cr.Spec.ForProvider, passw)
	if err != nil {
		werr := errors.Wrap(err, errUserUpdate)
		return managed.ExternalUpdate{}, compute.AddAPIResponseInfo(resp, werr)
	}
	conn := connectionDetails(cr, observed)
	if cr.HasCredentialsSecretRef() {
		conn[xpv1.ResourceCredentialsSecretPasswordKey] = []byte(passw)
	}

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
