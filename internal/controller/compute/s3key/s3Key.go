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

package s3key

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	xpv1 "github.com/crossplane/crossplane-runtime/apis/common/v1"
	"github.com/crossplane/crossplane-runtime/pkg/connection"
	"github.com/crossplane/crossplane-runtime/pkg/event"
	"github.com/crossplane/crossplane-runtime/pkg/logging"
	"github.com/crossplane/crossplane-runtime/pkg/meta"
	"github.com/crossplane/crossplane-runtime/pkg/ratelimiter"
	"github.com/crossplane/crossplane-runtime/pkg/reconciler/managed"
	"github.com/crossplane/crossplane-runtime/pkg/resource"
	"github.com/crossplane/crossplane-runtime/pkg/statemetrics"
	"github.com/google/go-cmp/cmp"
	ionosdk "github.com/ionos-cloud/sdk-go/v6"
	"github.com/pkg/errors"
	"k8s.io/utils/ptr"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"

	"github.com/ionos-cloud/crossplane-provider-ionoscloud/apis/compute/v1alpha1"
	apisv1alpha1 "github.com/ionos-cloud/crossplane-provider-ionoscloud/apis/v1alpha1"
	"github.com/ionos-cloud/crossplane-provider-ionoscloud/internal/clients"
	"github.com/ionos-cloud/crossplane-provider-ionoscloud/internal/clients/compute"
	"github.com/ionos-cloud/crossplane-provider-ionoscloud/internal/clients/compute/s3key"
	"github.com/ionos-cloud/crossplane-provider-ionoscloud/internal/features"
	"github.com/ionos-cloud/crossplane-provider-ionoscloud/internal/utils"
)

const errNotS3Key = "managed resource is not a S3Key custom resource"

// Setup adds a controller that reconciles S3Key managed resources.
func Setup(mgr ctrl.Manager, opts *utils.ConfigurationOptions) error {
	name := managed.ControllerName(v1alpha1.S3KeyGroupKind)
	logger := opts.CtrlOpts.Logger
	if opts.CtrlOpts.MetricOptions != nil && opts.CtrlOpts.MetricOptions.MRStateMetrics != nil {
		stateMetricsRecorder := statemetrics.NewMRStateRecorder(
			mgr.GetClient(), opts.CtrlOpts.Logger, opts.CtrlOpts.MetricOptions.MRStateMetrics, &v1alpha1.S3KeyList{}, opts.CtrlOpts.MetricOptions.PollStateMetricInterval,
		)
		if err := mgr.Add(stateMetricsRecorder); err != nil {
			return errors.Wrap(err, "cannot register MR state metrics recorder for kind"+name)
		}
	}

	cps := []managed.ConnectionPublisher{managed.NewAPISecretPublisher(mgr.GetClient(), mgr.GetScheme())}
	if opts.CtrlOpts.Features.Enabled(features.EnableAlphaExternalSecretStores) {
		cps = append(cps, connection.NewDetailsManager(mgr.GetClient(), apisv1alpha1.StoreConfigGroupVersionKind))
	}

	return ctrl.NewControllerManagedBy(mgr).
		Named(name).
		WithOptions(controller.Options{
			MaxConcurrentReconciles: opts.GetMaxConcurrentReconcileRate(v1alpha1.S3KeyKind),
			RateLimiter:             ratelimiter.NewController(),
			RecoverPanic:            ptr.To(true),
		}).
		WithEventFilter(resource.DesiredStateChanged()).
		For(&v1alpha1.S3Key{}).
		Complete(managed.NewReconciler(mgr,
			resource.ManagedKind(v1alpha1.S3KeyGroupVersionKind),
			managed.WithExternalConnecter(&connectorS3Key{
				kube:                 mgr.GetClient(),
				usage:                resource.NewProviderConfigUsageTracker(mgr.GetClient(), &apisv1alpha1.ProviderConfigUsage{}),
				log:                  logger,
				isUniqueNamesEnabled: opts.GetIsUniqueNamesEnabled()}),
			managed.WithReferenceResolver(managed.NewAPISimpleReferenceResolver(mgr.GetClient())),
			managed.WithInitializers(),
			managed.WithPollInterval(opts.GetPollInterval()),
			managed.WithTimeout(opts.GetTimeout()),
			managed.WithCreationGracePeriod(opts.GetCreationGracePeriod()),
			managed.WithLogger(logger.WithValues("controller", name)),
			managed.WithMetricRecorder(opts.CtrlOpts.MetricOptions.MRMetrics),
			managed.WithRecorder(event.NewAPIRecorder(mgr.GetEventRecorderFor(name))),
			managed.WithConnectionPublishers(cps...)),
		)
}

// A connectorS3Key is expected to produce an ExternalClient when its Connect method
// is called.
type connectorS3Key struct {
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
func (c *connectorS3Key) Connect(ctx context.Context, mg resource.Managed) (managed.ExternalClient, error) {
	_, ok := mg.(*v1alpha1.S3Key)
	if !ok {
		return nil, errors.New(errNotS3Key)
	}
	svc, err := clients.ConnectForCRD(ctx, mg, c.kube, c.usage)
	return &externalS3Key{
		service: s3key.APIClient{IonosServices: svc},
		log:     c.log,
	}, err
}

// An ExternalClient observes, then either creates, updates, or deletes an
// externalS3Key resource to ensure it reflects the managed resource's desired state.
type externalS3Key struct {
	// A 'client' used to connect to the externalS3Key resource API. In practice this
	// would be something like an IONOS Cloud SDK client.
	service s3key.APIClient
	log     logging.Logger
}

func (c *externalS3Key) Observe(ctx context.Context, mg resource.Managed) (managed.ExternalObservation, error) {
	cr, ok := mg.(*v1alpha1.S3Key)
	if !ok {
		return managed.ExternalObservation{}, errors.New(errNotS3Key)
	}

	// External Name of the CR is the S3Key ID
	if meta.GetExternalName(cr) == "" {
		return managed.ExternalObservation{}, nil
	}
	observed, apiResponse, err := c.service.GetS3Key(ctx, cr.Spec.ForProvider.UserID, meta.GetExternalName(cr))
	if err != nil {
		retErr := fmt.Errorf("failed to get S3Key by id. error: %w", err)
		// we get a 422 error response on key not found instead of 404
		if apiResponse != nil && apiResponse.Response != nil && apiResponse.StatusCode == http.StatusUnprocessableEntity && strings.Contains(err.Error(), "The access key cannot be found") {
			return managed.ExternalObservation{}, nil
		}
		return managed.ExternalObservation{}, compute.ErrorUnlessNotFound(apiResponse, retErr)
	}
	current := cr.Spec.ForProvider.DeepCopy()
	cr.Status.AtProvider.S3KeyID = meta.GetExternalName(cr)
	cr.SetConditions(xpv1.Available())

	return managed.ExternalObservation{
		ResourceExists:          true,
		ResourceUpToDate:        s3key.IsS3KeyUpToDate(cr, observed),
		ConnectionDetails:       s3ConnectionDetailsTo(observed),
		ResourceLateInitialized: !cmp.Equal(current, &cr.Spec.ForProvider),
	}, nil
}

func (c *externalS3Key) Create(ctx context.Context, mg resource.Managed) (managed.ExternalCreation, error) {
	cr, ok := mg.(*v1alpha1.S3Key)
	if !ok {
		return managed.ExternalCreation{}, errors.New(errNotS3Key)
	}
	cr.SetConditions(xpv1.Creating())

	newInstance, apiResponse, err := c.service.CreateS3Key(ctx, cr.Spec.ForProvider.UserID)
	if err != nil {
		retErr := fmt.Errorf("failed to create S3Key. error: %w", err)
		return managed.ExternalCreation{}, compute.AddAPIResponseInfo(apiResponse, retErr)
	}

	meta.SetExternalName(cr, *newInstance.Id)
	return managed.ExternalCreation{ConnectionDetails: s3ConnectionDetailsTo(newInstance)}, nil
}

func (c *externalS3Key) Update(ctx context.Context, mg resource.Managed) (managed.ExternalUpdate, error) {
	cr, ok := mg.(*v1alpha1.S3Key)
	if !ok {
		return managed.ExternalUpdate{}, errors.New("could not update, " + errNotS3Key)
	}

	S3KeyID := cr.Status.AtProvider.S3KeyID
	instanceInput, err := s3key.GenerateUpdateSeKeyInput(cr)
	if err != nil {
		return managed.ExternalUpdate{}, err
	}

	observed, apiResponse, err := c.service.UpdateS3Key(ctx, cr.Spec.ForProvider.UserID, S3KeyID, *instanceInput)
	if err != nil {
		retErr := fmt.Errorf("failed to update S3Key. error: %w", err)
		return managed.ExternalUpdate{}, compute.AddAPIResponseInfo(apiResponse, retErr)
	}
	if err = compute.WaitForRequest(ctx, c.service.GetAPIClient(), apiResponse); err != nil {
		return managed.ExternalUpdate{}, fmt.Errorf("while waiting for request. %w", err)
	}
	return managed.ExternalUpdate{
		ConnectionDetails: s3ConnectionDetailsTo(observed),
	}, nil
}

func (c *externalS3Key) Delete(ctx context.Context, mg resource.Managed) (managed.ExternalDelete, error) {
	cr, ok := mg.(*v1alpha1.S3Key)
	if !ok {
		return managed.ExternalDelete{}, errors.New(errNotS3Key)
	}

	cr.SetConditions(xpv1.Deleting())

	apiResponse, err := c.service.DeleteS3Key(ctx, cr.Spec.ForProvider.UserID, cr.Status.AtProvider.S3KeyID)
	if err != nil {
		retErr := fmt.Errorf("failed to delete S3Key. error: %w", err)
		return managed.ExternalDelete{}, compute.ErrorUnlessNotFound(apiResponse, retErr)
	}

	return managed.ExternalDelete{}, compute.WaitForRequest(ctx, c.service.GetAPIClient(), apiResponse)
}

func s3ConnectionDetailsTo(observed ionosdk.S3Key) managed.ConnectionDetails {
	var details = make(managed.ConnectionDetails)
	if observed.Id == nil || observed.Properties == nil {
		return details
	}
	props := observed.GetProperties()
	details["s3KeyID"] = []byte(utils.DereferenceOrZero(observed.Id))
	details["s3SecretKey"] = []byte(*props.SecretKey)

	return details
}

func (c *externalS3Key) Disconnect(_ context.Context) error {
	return nil
}
