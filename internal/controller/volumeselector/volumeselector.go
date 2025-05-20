package volumeselector

import (
	"context"
	"fmt"
	"strconv"

	xpv1 "github.com/crossplane/crossplane-runtime/apis/common/v1"
	"github.com/crossplane/crossplane-runtime/pkg/event"
	"github.com/crossplane/crossplane-runtime/pkg/logging"
	"github.com/crossplane/crossplane-runtime/pkg/meta"
	"github.com/crossplane/crossplane-runtime/pkg/ratelimiter"
	"github.com/crossplane/crossplane-runtime/pkg/reconciler/managed"
	"github.com/crossplane/crossplane-runtime/pkg/resource"
	"github.com/crossplane/crossplane-runtime/pkg/statemetrics"
	sdkgo "github.com/ionos-cloud/sdk-go/v6"
	"github.com/pkg/errors"
	"k8s.io/utils/ptr"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"

	"github.com/ionos-cloud/crossplane-provider-ionoscloud/apis/compute/v1alpha1"
	apisv1alpha1 "github.com/ionos-cloud/crossplane-provider-ionoscloud/apis/v1alpha1"
	"github.com/ionos-cloud/crossplane-provider-ionoscloud/internal/clients"
	"github.com/ionos-cloud/crossplane-provider-ionoscloud/internal/clients/compute"
	"github.com/ionos-cloud/crossplane-provider-ionoscloud/internal/clients/compute/server"
	"github.com/ionos-cloud/crossplane-provider-ionoscloud/internal/controller/serverset"
	"github.com/ionos-cloud/crossplane-provider-ionoscloud/internal/utils"
)

const errNotVolumeSelector = "managed resource is not a Volumeselector custom resource"

const (
	// IndexLabel <serverset_serverset_name>-<resource_type>-ri
	IndexLabel = "%s-%s-ri"
	// VolumeIndexLabel <serverset_serverset_name>-<resource_type>-i
	VolumeIndexLabel = "%s-%s-i"
)

// ResourceDataVolume is the res name for the volume
const ResourceDataVolume = "dv"

// Setup adds a controller that reconciles Volumeselector managed resources.
func Setup(mgr ctrl.Manager, opts *utils.ConfigurationOptions) error {
	name := managed.ControllerName(v1alpha1.VolumeselectorGroupKind)
	logger := opts.CtrlOpts.Logger
	if opts.CtrlOpts.MetricOptions != nil && opts.CtrlOpts.MetricOptions.MRStateMetrics != nil {
		stateMetricsRecorder := statemetrics.NewMRStateRecorder(
			mgr.GetClient(), opts.CtrlOpts.Logger, opts.CtrlOpts.MetricOptions.MRStateMetrics, &v1alpha1.VolumeselectorList{}, opts.CtrlOpts.MetricOptions.PollStateMetricInterval,
		)
		if err := mgr.Add(stateMetricsRecorder); err != nil {
			return errors.Wrap(err, "cannot register MR state metrics recorder for kind"+name)
		}
	}
	return ctrl.NewControllerManagedBy(mgr).
		Named(name).
		WithOptions(controller.Options{
			MaxConcurrentReconciles: opts.GetMaxConcurrentReconcileRate(v1alpha1.VolumeselectorKind),
			RateLimiter:             ratelimiter.NewController(),
			RecoverPanic:            ptr.To(true),
		}).
		WithEventFilter(resource.DesiredStateChanged()).
		For(&v1alpha1.Volumeselector{}).
		Complete(managed.NewReconciler(mgr,
			resource.ManagedKind(v1alpha1.VolumeselectorGroupVersionKind),
			managed.WithExternalConnecter(&connectorVolumeselector{
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

// A connectorVolumeselector is expected to produce an ExternalClient when its Connect method
// is called.
type connectorVolumeselector struct {
	kube  client.Client
	usage resource.Tracker
	log   logging.Logger
}

// Connect typically produces an ExternalClient by:
// 1. Tracking that the managed resource is using a ProviderConfig.
// 2. Getting the managed resource's ProviderConfig.
// 3. Getting the credentials specified by the ProviderConfig.
// 4. Using the credentials to form a client.
func (c *connectorVolumeselector) Connect(ctx context.Context, mg resource.Managed) (managed.ExternalClient, error) {
	_, ok := mg.(*v1alpha1.Volumeselector)
	if !ok {
		return nil, errors.New(errNotVolumeSelector)
	}
	svc, err := clients.ConnectForCRD(ctx, mg, c.kube, c.usage)
	return &externalVolumeselector{
		serverClient: &server.APIClient{IonosServices: svc},
		kube:         c.kube,
		log:          c.log}, err
}

// An ExternalClient observes, then either creates, updates, or deletes an
// externalVolumeselector resource to ensure it reflects the managed resource's desired state.
type externalVolumeselector struct {
	kube         client.Client
	serverClient server.Client
	log          logging.Logger
}

func (c *externalVolumeselector) Observe(ctx context.Context, mg resource.Managed) (managed.ExternalObservation, error) {
	cr, ok := mg.(*v1alpha1.Volumeselector)
	if !ok {
		return managed.ExternalObservation{}, errors.New(errNotVolumeSelector)
	}

	// External Name of the CR is the Volumeselector ID
	if meta.GetExternalName(cr) == "" || meta.WasDeleted(cr) {
		return managed.ExternalObservation{}, nil
	}

	for replicaIndex := 0; replicaIndex < cr.Spec.ForProvider.Replicas; replicaIndex++ {
		volumeList, serverList, err := c.getVolumesAndServers(ctx, cr.Spec.ForProvider.ServersetName, replicaIndex)
		if err != nil {
			return managed.ExternalObservation{}, err
		}
		if !c.areVolumesAndServersReady(volumeList, serverList) {
			return managed.ExternalObservation{
				ResourceExists:    true,
				ResourceUpToDate:  false,
				ConnectionDetails: managed.ConnectionDetails{},
			}, nil
		}
		isAttached := false
		for _, volume := range volumeList.Items {
			if isAttached, err = c.serverClient.IsVolumeAttached(ctx, serverList.Items[0].Spec.ForProvider.DatacenterCfg.DatacenterID,
				serverList.Items[0].Status.AtProvider.ServerID, volume.Status.AtProvider.VolumeID); err != nil {
				return managed.ExternalObservation{}, err
			}
			if !isAttached {
				return managed.ExternalObservation{
					ResourceExists:    true,
					ResourceUpToDate:  false,
					ConnectionDetails: managed.ConnectionDetails{},
				}, nil
			}
		}
	}
	cr.Status.AtProvider.State = sdkgo.Available
	cr.Status.SetConditions(xpv1.Available())
	return managed.ExternalObservation{
		ResourceExists:    true,
		ResourceUpToDate:  true,
		ConnectionDetails: managed.ConnectionDetails{},
	}, nil
}

func (c *externalVolumeselector) Create(ctx context.Context, mg resource.Managed) (managed.ExternalCreation, error) {
	cr, ok := mg.(*v1alpha1.Volumeselector)
	c.log.Debug("Create volumeselector", "name", cr.Name, "serverset", cr.Spec.ForProvider.ServersetName)

	if !ok {
		return managed.ExternalCreation{}, errors.New(errNotVolumeSelector)
	}
	cr.SetConditions(xpv1.Creating())

	meta.SetExternalName(cr, cr.Name)
	return managed.ExternalCreation{}, nil
}

func (c *externalVolumeselector) Update(ctx context.Context, mg resource.Managed) (managed.ExternalUpdate, error) {
	cr, ok := mg.(*v1alpha1.Volumeselector)
	c.log.Debug("Update volumeselector	", "name", cr.Name, "serverset", cr.Spec.ForProvider.ServersetName)
	if !ok {
		return managed.ExternalUpdate{}, errors.New(errNotVolumeSelector)
	}
	if meta.GetExternalName(cr) == "" {
		return managed.ExternalUpdate{}, nil
	}
	for replicaIndex := 0; replicaIndex < cr.Spec.ForProvider.Replicas; replicaIndex++ {
		volumeList, serverList, err := c.getVolumesAndServers(ctx, cr.Spec.ForProvider.ServersetName, replicaIndex)
		if err != nil {
			return managed.ExternalUpdate{}, err
		}
		if !c.areVolumesAndServersReady(volumeList, serverList) {
			continue
		}
		for _, volume := range volumeList.Items {
			if err = c.attachVolume(ctx, serverList.Items[0].Spec.ForProvider.DatacenterCfg.DatacenterID,
				serverList.Items[0].Status.AtProvider.ServerID, volume.Status.AtProvider.VolumeID); err != nil {
				return managed.ExternalUpdate{}, err
			}
		}

	}
	return managed.ExternalUpdate{}, nil
}

func (c *externalVolumeselector) Delete(ctx context.Context, mg resource.Managed) (managed.ExternalDelete, error) {
	cr, ok := mg.(*v1alpha1.Volumeselector)
	if !ok {
		return managed.ExternalDelete{}, errors.New(errNotVolumeSelector)
	}
	if meta.GetExternalName(cr) == "" {
		return managed.ExternalDelete{}, nil
	}
	meta.SetExternalName(cr, "")
	cr.SetConditions(xpv1.Deleting())
	return managed.ExternalDelete{}, nil
}

// listResFromSSetWithIndex - lists resources from a stateful server set with a specific index label
func listResFromSSetWithIndex(ctx context.Context, kube client.Client, label string, index int, list client.ObjectList) error {
	return kube.List(ctx, list, client.MatchingLabels{
		label: strconv.Itoa(index),
	})
}

func (c *externalVolumeselector) attachVolume(ctx context.Context, datacenterID, serverID, volumeID string) error {
	if datacenterID == "" || serverID == "" || volumeID == "" {
		return errors.New("datacenterID, serverID and volumeID cannot be empty")
	}
	c.log.Debug("attachVolume, starting to attach Volume", "volumeID", volumeID)
	isAttached := false
	var err error
	if isAttached, err = c.serverClient.IsVolumeAttached(ctx, datacenterID, serverID, volumeID); err != nil {
		return err
	}
	if isAttached {
		return nil
	}

	_, apiResponse, err := c.serverClient.AttachVolume(ctx, datacenterID, serverID, sdkgo.Volume{Id: &volumeID})
	if err != nil {
		return err
	}
	if err = compute.WaitForRequest(ctx, c.serverClient.GetAPIClient(), apiResponse); err != nil {
		return err
	}
	c.log.Debug("attachVolume, finished attaching Volume", "volumeID", volumeID)
	return nil
}

func (c *externalVolumeselector) areVolumesAndServersReady(volumeList v1alpha1.VolumeList, serverList v1alpha1.ServerList) bool {
	if len(volumeList.Items) == 0 {
		c.log.Info("volumeselector no Volumes found")
		return false
	}
	if len(serverList.Items) == 0 {
		c.log.Info("volumeselector no Servers found")
		return false
	}
	for _, volume := range volumeList.Items {
		if volume.Status.AtProvider.VolumeID == "" {
			c.log.Info("volumeselector Volume does not have ID", "name", volume.Name)
			return false
		}
	}
	if serverList.Items[0].Spec.ForProvider.DatacenterCfg.DatacenterID == "" {
		c.log.Info("Server does not have dcID", "name", serverList.Items[0].Name)
		return false
	}
	if serverList.Items[0].Status.AtProvider.ServerID == "" {
		c.log.Info("Server does not have ID", "name", serverList.Items[0].Name)
		return false
	}

	return true
}

func (c *externalVolumeselector) getVolumesAndServers(ctx context.Context, serversetName string, replicaIndex int) (v1alpha1.VolumeList, v1alpha1.ServerList, error) {
	volumeList := v1alpha1.VolumeList{}
	serverList := v1alpha1.ServerList{}
	err := listResFromSSetWithIndex(ctx, c.kube, fmt.Sprintf(IndexLabel, serversetName, ResourceDataVolume), replicaIndex, &volumeList)
	if err != nil {
		return volumeList, serverList, err
	}
	// get server by index

	err = serverset.ListResFromSSetWithIndex(ctx, c.kube, serversetName, serverset.ResourceServer, replicaIndex, &serverList)
	if err != nil {
		return volumeList, serverList, err
	}
	return volumeList, serverList, err
}

// Disconnect does nothing because there are no resources to release. Needs to be implemented starting from crossplane-runtime v0.17
func (c *externalVolumeselector) Disconnect(_ context.Context) error {
	return nil
}
