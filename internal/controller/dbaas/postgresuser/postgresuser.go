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

package postgresuser

import (
	"context"
	"encoding/json"
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

	"github.com/ionos-cloud/crossplane-provider-ionoscloud/apis/dbaas/postgres/v1alpha1"
	apisv1alpha1 "github.com/ionos-cloud/crossplane-provider-ionoscloud/apis/v1alpha1"
	"github.com/ionos-cloud/crossplane-provider-ionoscloud/internal/clients"
	"github.com/ionos-cloud/crossplane-provider-ionoscloud/internal/clients/dbaas/postgrescluster"
	"github.com/ionos-cloud/crossplane-provider-ionoscloud/internal/utils"
)

const errNotUser = "managed resource is not a Postgres user custom resource"

// Setup adds a controller that reconciles User managed resources.
func Setup(mgr ctrl.Manager, l logging.Logger, rl workqueue.RateLimiter, opts *utils.ConfigurationOptions) error {
	name := managed.ControllerName(v1alpha1.PostgresUserGroupKind)

	return ctrl.NewControllerManagedBy(mgr).
		Named(name).
		WithOptions(controller.Options{
			RateLimiter: ratelimiter.NewController(),
		}).
		For(&v1alpha1.PostgresUser{}).
		Complete(managed.NewReconciler(mgr,
			resource.ManagedKind(v1alpha1.PostgresUserGroupVersionKind),
			managed.WithExternalConnecter(&connectorUser{
				kube:  mgr.GetClient(),
				usage: resource.NewProviderConfigUsageTracker(mgr.GetClient(), &apisv1alpha1.ProviderConfigUsage{}),
				log:   l}),
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
	_, ok := mg.(*v1alpha1.PostgresUser)
	if !ok {
		return nil, errors.New(errNotUser)
	}
	svc, err := clients.ConnectForCRD(ctx, mg, c.kube, c.usage)
	return &externalUser{
		service: &postgrescluster.ClusterAPIClient{IonosServices: svc},
		log:     c.log,
		client:  c.kube}, err

}

// An ExternalClient observes, then either creates, updates, or deletes an
// externalUser resource to ensure it reflects the managed resource's desired state.
type externalUser struct {
	// A 'client' used to connect to the externalUser resource API. In practice this
	// would be something like an IONOS Cloud SDK client.
	service postgrescluster.ClusterClient
	client  client.Client
	log     logging.Logger
}

// Observe checks whether the specified externalUser resource exists
func (u *externalUser) Observe(ctx context.Context, mg resource.Managed) (managed.ExternalObservation, error) { // nolint:gocyclo
	cr, ok := mg.(*v1alpha1.PostgresUser)
	if !ok {
		return managed.ExternalObservation{}, errors.New(errNotUser)
	}

	// External Name of the CR is the DBaaS Postgres User ID
	if meta.GetExternalName(cr) == "" {
		return managed.ExternalObservation{}, nil
	}
	observed, resp, err := u.service.GetUser(ctx, cr.Spec.ForProvider.ClusterCfg.ClusterID, meta.GetExternalName(cr))
	if err != nil {
		if resp.HttpNotFound() {
			return managed.ExternalObservation{}, nil
		}
		return managed.ExternalObservation{}, fmt.Errorf("failed to get postgres user by id. err: %w", err)
	}

	current := cr.Spec.ForProvider.DeepCopy()

	cr.Status.AtProvider.UserID = meta.GetExternalName(cr)

	return managed.ExternalObservation{
		ResourceExists:          true,
		ResourceUpToDate:        postgrescluster.IsUserUpToDate(cr, observed),
		ConnectionDetails:       managed.ConnectionDetails{},
		ResourceLateInitialized: !cmp.Equal(current, &cr.Spec.ForProvider),
	}, nil
}

// Create creates the user
func (u *externalUser) Create(ctx context.Context, mg resource.Managed) (managed.ExternalCreation, error) { // nolint: gocyclo
	cr, ok := mg.(*v1alpha1.PostgresUser)
	if !ok {
		return managed.ExternalCreation{}, errors.New(errNotUser)
	}
	cr.SetConditions(xpv1.Creating())

	clusterID := cr.Spec.ForProvider.ClusterCfg.ClusterID

	instanceInput := postgrescluster.GenerateCreateUserInput(cr)
	// time to get the credentials from the secret
	if instanceInput.Properties.Password != nil || *instanceInput.Properties.Password == "" {
		data, err := resource.CommonCredentialExtractor(ctx, cr.Spec.ForProvider.Credentials.Source, u.client, cr.Spec.ForProvider.Credentials.CommonCredentialSelectors)
		if err != nil {
			return managed.ExternalCreation{}, errors.Wrap(err, "cannot get psql credentials")
		}
		creds := v1alpha1.DBUser{}
		if err := json.Unmarshal(data, &creds); err != nil {
			return managed.ExternalCreation{}, fmt.Errorf("failed to decode psql credentials: %w", err)
		}
		*instanceInput.Properties.Username = creds.Username
		*instanceInput.Properties.Password = creds.Password
	}
	if (instanceInput.Properties.Username == nil || *instanceInput.Properties.Username == "") ||
		(instanceInput.Properties.Password == nil || *instanceInput.Properties.Password == "") {
		return managed.ExternalCreation{}, fmt.Errorf("need to provide credentials, either directly or from a secret")
	}
	newInstance, apiResponse, err := u.service.CreateUser(ctx, clusterID, *instanceInput)
	creation := managed.ExternalCreation{ConnectionDetails: managed.ConnectionDetails{}}
	if err != nil {
		retErr := fmt.Errorf("failed to create postgres user: %w", err)
		if apiResponse != nil && apiResponse.Response != nil {
			retErr = fmt.Errorf("%w API Response Status: %v", retErr, apiResponse.Status)
		}
		return creation, retErr
	}

	// Set External Name
	if instanceInput.Properties.Username != nil {
		cr.Status.AtProvider.UserID = *instanceInput.Properties.Username
	}
	meta.SetExternalName(cr, *newInstance.Properties.Username)
	return creation, nil
}

// Update updates the user
func (u *externalUser) Update(ctx context.Context, mg resource.Managed) (managed.ExternalUpdate, error) {
	cr, ok := mg.(*v1alpha1.PostgresUser)
	if !ok {
		return managed.ExternalUpdate{}, errors.New(errNotUser)
	}

	clusterID := cr.Spec.ForProvider.ClusterCfg.ClusterID
	userName := meta.GetExternalName(cr)
	instanceInput, err := postgrescluster.GenerateUpdateUserInput(cr)
	if err != nil {
		return managed.ExternalUpdate{}, err
	}
	if instanceInput.Properties.Password != nil && *instanceInput.Properties.Password == "" {
		data, err := resource.CommonCredentialExtractor(ctx, cr.Spec.ForProvider.Credentials.Source, u.client, cr.Spec.ForProvider.Credentials.CommonCredentialSelectors)
		if err != nil {
			return managed.ExternalUpdate{}, errors.Wrap(err, "cannot get psql credentials")
		}
		creds := v1alpha1.DBUser{}
		if err := json.Unmarshal(data, &creds); err != nil {
			return managed.ExternalUpdate{}, fmt.Errorf("failed to decode psql credentials: %w", err)
		}
		*instanceInput.Properties.Password = creds.Password
	}

	_, apiResponse, err := u.service.UpdateUser(ctx, clusterID, userName, *instanceInput)
	if err != nil {
		retErr := fmt.Errorf("failed to update postgres user: %w", err)
		if apiResponse != nil && apiResponse.Response != nil {
			retErr = fmt.Errorf("%w API Response Status: %v", retErr, apiResponse.Status)
		}
		return managed.ExternalUpdate{}, retErr
	}
	return managed.ExternalUpdate{}, nil
}

// Delete deletes the user
func (u *externalUser) Delete(ctx context.Context, mg resource.Managed) error {
	cr, ok := mg.(*v1alpha1.PostgresUser)
	if !ok {
		return errors.New(errNotUser)
	}

	cr.SetConditions(xpv1.Deleting())
	apiResponse, err := u.service.DeleteUser(ctx, cr.Spec.ForProvider.ClusterCfg.ClusterID, meta.GetExternalName(cr))
	if err != nil {
		if apiResponse.HttpNotFound() {
			return nil
		}
		return fmt.Errorf("failed to delete postgres user. error: %w", err)
	}
	return nil
}
