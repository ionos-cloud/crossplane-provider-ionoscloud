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

	"github.com/crossplane/crossplane-runtime/pkg/statemetrics"
	"github.com/ionos-cloud/sdk-go-bundle/shared"
	"github.com/pkg/errors"
	"golang.org/x/crypto/bcrypt"

	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	xpv1 "github.com/crossplane/crossplane-runtime/apis/common/v1"
	"github.com/crossplane/crossplane-runtime/pkg/event"
	"github.com/crossplane/crossplane-runtime/pkg/logging"
	"github.com/crossplane/crossplane-runtime/pkg/meta"
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
func Setup(mgr ctrl.Manager, opts *utils.ConfigurationOptions) error {
	name := managed.ControllerName(v1alpha1.PostgresUserGroupKind)
	logger := opts.CtrlOpts.Logger
	if opts.CtrlOpts.MetricOptions != nil && opts.CtrlOpts.MetricOptions.MRStateMetrics != nil {
		stateMetricsRecorder := statemetrics.NewMRStateRecorder(
			mgr.GetClient(), opts.CtrlOpts.Logger, opts.CtrlOpts.MetricOptions.MRStateMetrics, &v1alpha1.PostgresUserList{}, opts.CtrlOpts.MetricOptions.PollStateMetricInterval,
		)
		if err := mgr.Add(stateMetricsRecorder); err != nil {
			return errors.Wrap(err, "cannot register MR state metrics recorder for kind"+name)
		}
	}

	return ctrl.NewControllerManagedBy(mgr).
		Named(name).
		WithOptions(opts.CtrlOpts.ForControllerRuntime()).
		WithEventFilter(resource.DesiredStateChanged()).
		For(&v1alpha1.PostgresUser{}).
		Complete(managed.NewReconciler(mgr,
			resource.ManagedKind(v1alpha1.PostgresUserGroupVersionKind),
			managed.WithExternalConnecter(&connectorUser{
				kube:  mgr.GetClient(),
				usage: resource.NewProviderConfigUsageTracker(mgr.GetClient(), &apisv1alpha1.ProviderConfigUsage{}),
				log:   logger}),
			managed.WithReferenceResolver(managed.NewAPISimpleReferenceResolver(mgr.GetClient())),
			managed.WithInitializers(),
			managed.WithPollInterval(opts.GetPollInterval()),
			managed.WithTimeout(opts.GetTimeout()),
			managed.WithCreationGracePeriod(opts.GetCreationGracePeriod()),
			managed.WithLogger(logger.WithValues("controller", name)),
			managed.WithMetricRecorder(opts.CtrlOpts.MetricOptions.MRMetrics),
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
	lateInitialized := u.lateInitialize(ctx, cr)
	cr.Status.AtProvider.UserID = meta.GetExternalName(cr)
	cr.SetConditions(xpv1.Available())
	return managed.ExternalObservation{
		ResourceExists:          true,
		ResourceUpToDate:        postgrescluster.IsUserUpToDate(cr, observed) && !lateInitialized,
		ConnectionDetails:       managed.ConnectionDetails{},
		ResourceLateInitialized: lateInitialized,
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
	// Get credentials from spec initially
	instanceInput := postgrescluster.GenerateCreateUserInput(cr)
	// Overwrite with values retrieved from Secret
	if cr.Spec.ForProvider.Credentials.Source != "" && cr.Spec.ForProvider.Credentials.Source != xpv1.CredentialsSourceNone {
		creds, err := u.readCredentials(ctx, cr)
		if err != nil {
			return managed.ExternalCreation{}, err
		}
		instanceInput.Properties.Username = creds.Username
		*instanceInput.Properties.Password = creds.Password
	}
	if (instanceInput.Properties.Username == "") ||
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

	meta.SetExternalName(cr, newInstance.Properties.Username)
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
	instanceInput := postgrescluster.GenerateUpdateUserInput(cr)
	// Password from credential source overrides value from the Spec
	// This is because the API does not return a password value, so we cannot trigger password changes by comparing observed password to spec password.
	if cr.Spec.ForProvider.Credentials.Source != "" && cr.Spec.ForProvider.Credentials.Source != xpv1.CredentialsSourceNone {
		creds, err := u.readCredentials(ctx, cr)
		if err != nil {
			return managed.ExternalUpdate{}, err
		}
		instanceInput.Properties.Password = shared.ToPtr(creds.Password)
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
func (u *externalUser) Delete(ctx context.Context, mg resource.Managed) (managed.ExternalDelete, error) {
	cr, ok := mg.(*v1alpha1.PostgresUser)
	if !ok {
		return managed.ExternalDelete{}, errors.New(errNotUser)
	}

	cr.SetConditions(xpv1.Deleting())
	apiResponse, err := u.service.DeleteUser(ctx, cr.Spec.ForProvider.ClusterCfg.ClusterID, meta.GetExternalName(cr))
	if err != nil {
		if apiResponse.HttpNotFound() {
			return managed.ExternalDelete{}, nil
		}
		return managed.ExternalDelete{}, fmt.Errorf("failed to delete postgres user. error: %w", err)
	}
	return managed.ExternalDelete{}, nil
}

func (u *externalUser) readCredentials(ctx context.Context, cr *v1alpha1.PostgresUser) (v1alpha1.DBUser, error) {
	creds := v1alpha1.DBUser{}

	data, err := resource.CommonCredentialExtractor(ctx, cr.Spec.ForProvider.Credentials.Source, u.client, cr.Spec.ForProvider.Credentials.CommonCredentialSelectors)
	if err != nil {
		return v1alpha1.DBUser{}, errors.Wrap(err, "cannot get psql credentials")
	}
	if err = json.Unmarshal(data, &creds); err != nil {
		return v1alpha1.DBUser{}, fmt.Errorf("failed to decode psql credentials: %w", err)
	}
	return creds, nil
}

// If credentials are supplied through credentials Source, set the hashed password to the Spec
func (u *externalUser) lateInitialize(ctx context.Context, cr *v1alpha1.PostgresUser) bool {
	if cr.Spec.ForProvider.Credentials.Source == "" || cr.Spec.ForProvider.Credentials.Source == xpv1.CredentialsSourceNone {
		return false
	}
	creds, err := u.readCredentials(ctx, cr)
	if err != nil {
		return false
	}
	var hash []byte
	if hash, err = bcrypt.GenerateFromPassword([]byte(creds.Password), bcrypt.MinCost); err != nil {
		return false
	}
	if err = bcrypt.CompareHashAndPassword([]byte(cr.Spec.ForProvider.Credentials.Password), []byte(creds.Password)); err == nil {
		return false
	}
	hashStr := string(hash)
	cr.Spec.ForProvider.Credentials.Password = hashStr
	return true
}

// Disconnect does nothing because there are no resources to release. Needs to be implemented starting from crossplane-runtime v0.17
func (c *externalUser) Disconnect(_ context.Context) error {
	return nil
}
