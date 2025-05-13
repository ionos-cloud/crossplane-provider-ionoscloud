package dataplatformcluster

import (
	"context"
	"fmt"
	"time"

	xpv1 "github.com/crossplane/crossplane-runtime/apis/common/v1"
	"github.com/crossplane/crossplane-runtime/pkg/errors"
	"github.com/crossplane/crossplane-runtime/pkg/event"
	"github.com/crossplane/crossplane-runtime/pkg/logging"
	"github.com/crossplane/crossplane-runtime/pkg/meta"
	"github.com/crossplane/crossplane-runtime/pkg/reconciler/managed"
	"github.com/crossplane/crossplane-runtime/pkg/resource"
	"github.com/crossplane/crossplane-runtime/pkg/statemetrics"
	"github.com/google/go-cmp/cmp"

	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/ionos-cloud/crossplane-provider-ionoscloud/apis/dataplatform/v1alpha1"
	apisv1alpha1 "github.com/ionos-cloud/crossplane-provider-ionoscloud/apis/v1alpha1"
	"github.com/ionos-cloud/crossplane-provider-ionoscloud/internal/clients"
	"github.com/ionos-cloud/crossplane-provider-ionoscloud/internal/clients/compute"
	"github.com/ionos-cloud/crossplane-provider-ionoscloud/internal/clients/dataplatform/dataplatformcluster"
	"github.com/ionos-cloud/crossplane-provider-ionoscloud/internal/utils"
)

const errNotDataplatformCluster = "managed resource is not a Dataplatform custom resource"

// Setup adds a controller that reconciles Dataplatform managed resources.
func Setup(mgr ctrl.Manager, opts *utils.ConfigurationOptions) error {
	name := managed.ControllerName(v1alpha1.DataplatformClusterGroupKind)
	logger := opts.CtrlOpts.Logger
	if opts.CtrlOpts.MetricOptions != nil && opts.CtrlOpts.MetricOptions.MRStateMetrics != nil {
		stateMetricsRecorder := statemetrics.NewMRStateRecorder(
			mgr.GetClient(), opts.CtrlOpts.Logger, opts.CtrlOpts.MetricOptions.MRStateMetrics, &v1alpha1.DataplatformClusterList{}, opts.CtrlOpts.MetricOptions.PollStateMetricInterval,
		)
		if err := mgr.Add(stateMetricsRecorder); err != nil {
			return errors.Wrap(err, "cannot register MR state metrics recorder for kind"+name)
		}
	}

	return ctrl.NewControllerManagedBy(mgr).
		Named(name).
		WithOptions(opts.CtrlOpts.ForControllerRuntime()).
		WithEventFilter(resource.DesiredStateChanged()).
		For(&v1alpha1.DataplatformCluster{}).
		Complete(managed.NewReconciler(mgr,
			resource.ManagedKind(v1alpha1.DataplatformClusterGroupVersionKind),
			managed.WithExternalConnecter(&connectorDataplatform{
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
			managed.WithRecorder(event.NewAPIRecorder(mgr.GetEventRecorderFor(name)))))
}

// A connectorDataplatform is expected to produce an ExternalClient when its Connect method
// is called.
type connectorDataplatform struct {
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
func (c *connectorDataplatform) Connect(ctx context.Context, mg resource.Managed) (managed.ExternalClient, error) {
	_, ok := mg.(*v1alpha1.DataplatformCluster)
	if !ok {
		return nil, errors.New(errNotDataplatformCluster)
	}
	svc, err := clients.ConnectForCRD(ctx, mg, c.kube, c.usage)
	return &externalDataplatform{
		service:              &dataplatformcluster.APIClient{IonosServices: svc},
		log:                  c.log,
		isUniqueNamesEnabled: c.isUniqueNamesEnabled}, err
}

// An ExternalClient observes, then either creates, updates, or deletes an
// externalDataplatform resource to ensure it reflects the managed resource's desired state.
type externalDataplatform struct {
	// A 'client' used to connect to the externalDataplatform resource API. In practice this
	// would be something like an IONOS Cloud SDK client.
	service              dataplatformcluster.Client
	log                  logging.Logger
	isUniqueNamesEnabled bool
}

func (c *externalDataplatform) Observe(ctx context.Context, mg resource.Managed) (managed.ExternalObservation, error) {
	cr, ok := mg.(*v1alpha1.DataplatformCluster)
	if !ok {
		return managed.ExternalObservation{}, errors.New(errNotDataplatformCluster)
	}

	// External Name of the CR is the Dataplatform ID
	if meta.GetExternalName(cr) == "" {
		return managed.ExternalObservation{}, nil
	}
	instance, apiResponse, err := c.service.GetDataplatformClusterByID(ctx, meta.GetExternalName(cr))
	if err != nil {
		if apiResponse.HttpNotFound() {
			return managed.ExternalObservation{}, nil
		}
		err = fmt.Errorf("failed to get dataplatform cluster by id. error: %w", err)
		return managed.ExternalObservation{}, err
	}
	current := cr.Spec.ForProvider.DeepCopy()
	dataplatformcluster.LateInitializer(&cr.Spec.ForProvider, &instance)
	dataplatformcluster.LateStatusInitializer(&cr.Status.AtProvider, &instance)
	cr.Status.AtProvider.DataplatformID = meta.GetExternalName(cr)
	cr.Status.AtProvider.State = *instance.Metadata.State
	c.log.Debug(fmt.Sprintf("Observing state: %v", cr.Status.AtProvider.State))
	clients.UpdateCondition(cr, cr.Status.AtProvider.State)

	return managed.ExternalObservation{
		ResourceExists:          true,
		ResourceUpToDate:        dataplatformcluster.IsUpToDate(cr, instance),
		ConnectionDetails:       managed.ConnectionDetails{},
		ResourceLateInitialized: !cmp.Equal(current, &cr.Spec.ForProvider),
	}, nil
}

func (c *externalDataplatform) Create(ctx context.Context, mg resource.Managed) (managed.ExternalCreation, error) {
	cr, ok := mg.(*v1alpha1.DataplatformCluster)
	if !ok {
		return managed.ExternalCreation{}, errors.New(errNotDataplatformCluster)
	}
	cr.SetConditions(xpv1.Creating())
	if cr.Status.AtProvider.State == compute.BUSY {
		return managed.ExternalCreation{}, nil
	}

	// Create new Dataplatform instance accordingly
	// with the properties set.
	instanceInput := dataplatformcluster.GenerateCreateInput(cr)

	newInstance, _, err := c.service.CreateDataplatformCluster(ctx, *instanceInput)
	if err != nil {
		retErr := fmt.Errorf("failed to create Dataplatform. error: %w", err)
		return managed.ExternalCreation{}, retErr
	}

	cr.Status.AtProvider.DataplatformID = *newInstance.Id
	meta.SetExternalName(cr, *newInstance.Id)
	return managed.ExternalCreation{}, nil
}

func (c *externalDataplatform) Update(ctx context.Context, mg resource.Managed) (managed.ExternalUpdate, error) {
	cr, ok := mg.(*v1alpha1.DataplatformCluster)
	if !ok {
		return managed.ExternalUpdate{}, errors.New(errNotDataplatformCluster)
	}
	if cr.Status.AtProvider.State == compute.BUSY || cr.Status.AtProvider.State == compute.UPDATING {
		return managed.ExternalUpdate{}, nil
	}

	DataplatformID := cr.Status.AtProvider.DataplatformID

	instanceInput := dataplatformcluster.GenerateUpdateInput(cr)
	_, _, err := c.service.PatchDataPlatformCluster(ctx, DataplatformID, *instanceInput)
	if err != nil {
		retErr := fmt.Errorf("failed to update dataplatform cluster. error: %w", err)
		return managed.ExternalUpdate{}, retErr
	}

	return managed.ExternalUpdate{}, nil
}

func (c *externalDataplatform) Delete(ctx context.Context, mg resource.Managed) (managed.ExternalDelete, error) {
	cr, ok := mg.(*v1alpha1.DataplatformCluster)
	if !ok {
		return managed.ExternalDelete{}, errors.New(errNotDataplatformCluster)
	}

	cr.SetConditions(xpv1.Deleting())
	if cr.Status.AtProvider.State == compute.DESTROYING {
		return managed.ExternalDelete{}, nil
	}

	apiResponse, err := c.service.DeleteDataPlatformCluster(ctx, cr.Status.AtProvider.DataplatformID)
	if err != nil {
		if apiResponse.HttpNotFound() {
			return managed.ExternalDelete{}, nil
		}
		retErr := fmt.Errorf("failed to delete dataplatform cluster. error: %w", err)
		return managed.ExternalDelete{}, retErr
	}
	err = utils.WaitForResourceToBeDeleted(ctx, 30*time.Minute, c.service.IsDataplatformDeleted, cr.Status.AtProvider.DataplatformID)
	if err != nil {
		return managed.ExternalDelete{}, fmt.Errorf("an error occurred while deleting %w", err)
	}

	return managed.ExternalDelete{}, nil
}

// Disconnect does nothing because there are no resources to release. Needs to be implemented starting from crossplane-runtime v0.17
func (c *externalDataplatform) Disconnect(_ context.Context) error {
	return nil
}
