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

	ionosdk "github.com/ionos-cloud/sdk-go/v6"
	"k8s.io/client-go/util/workqueue"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"

	userapi "github.com/ionos-cloud/crossplane-provider-ionoscloud/internal/clients/compute/user"

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

const errNotUser = "managed resource is not of a User type"

// Setup adds a controller that reconciles User managed resources.
func Setup(mgr ctrl.Manager, l logging.Logger, _ workqueue.RateLimiter, opts *utils.ConfigurationOptions) error {
	name := managed.ControllerName(v1alpha1.UserGroupKind)

	return ctrl.NewControllerManagedBy(mgr).
		Named(name).
		WithOptions(controller.Options{
			RateLimiter: ratelimiter.NewController(),
		}).
		For(&v1alpha1.User{}).
		Complete(managed.NewReconciler(mgr,
			resource.ManagedKind(v1alpha1.UserGroupVersionKind),
			managed.WithExternalConnecter(&connectorUser{
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
	svc, err := clients.ConnectForCRD(ctx, mg, c.kube, c.usage)
	return &externalUser{
		service: &userapi.APIClient{IonosServices: svc},
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
		return managed.ExternalObservation{}, errors.Wrap(err, "failed to get user by id")
	}

	props := observed.GetProperties()
	cr.Status.AtProvider.UserID = *observed.GetId()
	cr.Status.AtProvider.S3CanonicalUserID = *props.GetS3CanonicalUserId()

	cr.SetConditions(xpv1.Available())
	return managed.ExternalObservation{
		ResourceExists:    true,
		ResourceUpToDate:  userapi.IsUserUpToDate(cr, observed),
		ConnectionDetails: connectionDetails(cr, observed),
	}, nil
}

func connectionDetails(cr *v1alpha1.User, observed ionosdk.User) managed.ConnectionDetails {
	props := observed.GetProperties()
	return managed.ConnectionDetails{
		"email": []byte(*props.GetEmail()),
		xpv1.ResourceCredentialsSecretPasswordKey: []byte(cr.Spec.ForProvider.Password),
	}
}

func (eu *externalUser) Create(ctx context.Context, mg resource.Managed) (managed.ExternalCreation, error) {
	cr, ok := mg.(*v1alpha1.User)
	if !ok {
		return managed.ExternalCreation{}, errors.Wrap(errors.New(errNotUser), "create error")
	}
	cr.SetConditions(xpv1.Creating())

	nprops := ionosdk.NewUserPropertiesPost()
	userapi.SetUserProperties(*cr, nprops)
	observed, resp, err := eu.service.CreateUser(ctx, *ionosdk.NewUserPost(*nprops))
	if err != nil {
		werr := errors.Wrap(err, "failed to create user")
		return managed.ExternalCreation{}, compute.AddAPIResponseInfo(resp, werr)
	}

	props := observed.GetProperties()
	cr.Status.AtProvider.UserID = *observed.GetId()
	cr.Status.AtProvider.S3CanonicalUserID = *props.GetS3CanonicalUserId()

	meta.SetExternalName(cr, *observed.GetId())

	conn := connectionDetails(cr, observed)
	cr.Spec.ForProvider.Password = ""

	return managed.ExternalCreation{ConnectionDetails: conn}, nil
}

func (eu *externalUser) Update(ctx context.Context, mg resource.Managed) (managed.ExternalUpdate, error) {
	cr, ok := mg.(*v1alpha1.User)
	if !ok {
		return managed.ExternalUpdate{}, errors.Wrap(errors.New(errNotUser), "update error")
	}

	userID := cr.Status.AtProvider.UserID

	props := ionosdk.NewUserPropertiesPut()
	userapi.SetUserProperties(*cr, props)
	observed, resp, err := eu.service.UpdateUser(ctx, userID, *ionosdk.NewUserPut(*props))
	if err != nil {
		werr := errors.Wrap(err, "failed to update user")
		return managed.ExternalUpdate{}, compute.AddAPIResponseInfo(resp, werr)
	}

	if err = compute.WaitForRequest(ctx, eu.service.GetAPIClient(), resp); err != nil {
		return managed.ExternalUpdate{}, errors.Wrap(err, "error waiting for request")
	}

	conn := connectionDetails(cr, observed)
	cr.Spec.ForProvider.Password = ""

	return managed.ExternalUpdate{ConnectionDetails: conn}, nil
}

func (eu *externalUser) Delete(ctx context.Context, mg resource.Managed) error {
	user, ok := mg.(*v1alpha1.User)
	if !ok {
		return errors.Wrap(errors.New(errNotUser), "delete error")
	}
	user.SetConditions(xpv1.Deleting())

	resp, err := eu.service.DeleteUser(ctx, user.Status.AtProvider.UserID)
	if err != nil {
		return compute.ErrorUnlessNotFound(resp, errors.Wrap(err, "failed to delete user"))
	}

	return compute.WaitForRequest(ctx, eu.service.GetAPIClient(), resp)
}
