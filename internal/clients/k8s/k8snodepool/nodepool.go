package k8snodepool

import (
	"context"
	"reflect"

	sdkgo "github.com/ionos-cloud/sdk-go/v6"

	"github.com/ionos-cloud/crossplane-provider-ionoscloud/apis/k8s/v1alpha1"
	"github.com/ionos-cloud/crossplane-provider-ionoscloud/internal/clients"
	"github.com/ionos-cloud/crossplane-provider-ionoscloud/internal/utils"
)

// APIClient is a wrapper around IONOS Service
type APIClient struct {
	*clients.IonosServices
}

// Client is a wrapper around IONOS Service K8s Cluster methods
type Client interface {
	GetK8sNodePool(ctx context.Context, clusterID, nodepoolID string) (sdkgo.KubernetesNodePool, *sdkgo.APIResponse, error)
	CreateK8sNodePool(ctx context.Context, clusterID string, nodepool sdkgo.KubernetesNodePoolForPost) (sdkgo.KubernetesNodePool, *sdkgo.APIResponse, error)
	UpdateK8sNodePool(ctx context.Context, clusterID, nodepoolID string, nodepool sdkgo.KubernetesNodePoolForPut) (sdkgo.KubernetesNodePool, *sdkgo.APIResponse, error)
	DeleteK8sNodePool(ctx context.Context, clusterID, nodepoolID string) (*sdkgo.APIResponse, error)
	GetAPIClient() *sdkgo.APIClient
}

// GetK8sNodePool based on clusterID, nodepoolID
func (cp *APIClient) GetK8sNodePool(ctx context.Context, clusterID, nodepoolID string) (sdkgo.KubernetesNodePool, *sdkgo.APIResponse, error) {
	return cp.ComputeClient.KubernetesApi.K8sNodepoolsFindById(ctx, clusterID, nodepoolID).Depth(utils.DepthQueryParam).Execute()
}

// CreateK8sNodePool based on clusterID, KubernetesNodePoolForPost
func (cp *APIClient) CreateK8sNodePool(ctx context.Context, clusterID string, nodepool sdkgo.KubernetesNodePoolForPost) (sdkgo.KubernetesNodePool, *sdkgo.APIResponse, error) {
	return cp.ComputeClient.KubernetesApi.K8sNodepoolsPost(ctx, clusterID).KubernetesNodePool(nodepool).Execute()
}

// UpdateK8sNodePool based on clusterID, nodepoolID and KubernetesNodePoolForPut
func (cp *APIClient) UpdateK8sNodePool(ctx context.Context, clusterID, nodepoolID string, nodepool sdkgo.KubernetesNodePoolForPut) (sdkgo.KubernetesNodePool, *sdkgo.APIResponse, error) {
	return cp.ComputeClient.KubernetesApi.K8sNodepoolsPut(ctx, clusterID, nodepoolID).KubernetesNodePool(nodepool).Execute()
}

// DeleteK8sNodePool based on clusterID, nodepoolID
func (cp *APIClient) DeleteK8sNodePool(ctx context.Context, clusterID, nodepoolID string) (*sdkgo.APIResponse, error) {
	resp, err := cp.ComputeClient.KubernetesApi.K8sNodepoolsDelete(ctx, clusterID, nodepoolID).Execute()
	return resp, err
}

// GetAPIClient gets the APIClient
func (cp *APIClient) GetAPIClient() *sdkgo.APIClient {
	return cp.ComputeClient
}

// GenerateCreateK8sNodePoolInput returns sdkgo.KubernetesNodePoolForPost based on the CR spec
func GenerateCreateK8sNodePoolInput(cr *v1alpha1.NodePool) (*sdkgo.KubernetesNodePoolForPost, error) {
	instanceCreateInput := sdkgo.KubernetesNodePoolForPost{
		Properties: &sdkgo.KubernetesNodePoolPropertiesForPost{
			Name:             &cr.Spec.ForProvider.Name,
			DatacenterId:     &cr.Spec.ForProvider.DatacenterCfg.DatacenterID,
			NodeCount:        &cr.Spec.ForProvider.NodeCount,
			CpuFamily:        &cr.Spec.ForProvider.CPUFamily,
			CoresCount:       &cr.Spec.ForProvider.CoresCount,
			RamSize:          &cr.Spec.ForProvider.RAMSize,
			AvailabilityZone: &cr.Spec.ForProvider.AvailabilityZone,
			StorageType:      &cr.Spec.ForProvider.StorageType,
			StorageSize:      &cr.Spec.ForProvider.StorageSize,
		},
	}
	// TODO: ADD AUTOSCALING + LANS
	if !utils.IsEmptyValue(reflect.ValueOf(cr.Spec.ForProvider.K8sVersion)) {
		instanceCreateInput.Properties.SetK8sVersion(cr.Spec.ForProvider.K8sVersion)
	}
	if window := nodepoolMaintenanceWindow(cr.Spec.ForProvider.MaintenanceWindow); window != nil {
		instanceCreateInput.Properties.SetMaintenanceWindow(*window)
	}
	if !utils.IsEmptyValue(reflect.ValueOf(cr.Spec.ForProvider.Labels)) {
		instanceCreateInput.Properties.SetLabels(cr.Spec.ForProvider.Labels)
	}
	if !utils.IsEmptyValue(reflect.ValueOf(cr.Spec.ForProvider.Annotations)) {
		instanceCreateInput.Properties.SetAnnotations(cr.Spec.ForProvider.Annotations)
	}
	if !utils.IsEmptyValue(reflect.ValueOf(cr.Spec.ForProvider.PublicIPs)) {
		instanceCreateInput.Properties.SetPublicIps(cr.Spec.ForProvider.PublicIPs)
	}
	if !utils.IsEmptyValue(reflect.ValueOf(cr.Spec.ForProvider.GatewayIP)) {
		instanceCreateInput.Properties.SetGatewayIp(cr.Spec.ForProvider.GatewayIP)
	}
	return &instanceCreateInput, nil
}

// GenerateUpdateK8sNodePoolInput returns sdkgo.KubernetesNodePoolForPut based on the CR spec modifications
func GenerateUpdateK8sNodePoolInput(cr *v1alpha1.NodePool) (*sdkgo.KubernetesNodePoolForPut, error) {
	instanceUpdateInput := sdkgo.KubernetesNodePoolForPut{
		Properties: &sdkgo.KubernetesNodePoolPropertiesForPut{
			Name:      &cr.Spec.ForProvider.Name,
			NodeCount: &cr.Spec.ForProvider.NodeCount,
		},
	}
	// TODO: ADD OTHER OPTIONS FOR UPDATE
	if !utils.IsEmptyValue(reflect.ValueOf(cr.Spec.ForProvider.K8sVersion)) {
		instanceUpdateInput.Properties.SetK8sVersion(cr.Spec.ForProvider.K8sVersion)
	}
	if window := nodepoolMaintenanceWindow(cr.Spec.ForProvider.MaintenanceWindow); window != nil {
		instanceUpdateInput.Properties.SetMaintenanceWindow(*window)
	}
	return &instanceUpdateInput, nil
}

// LateInitializer fills the empty fields in *v1alpha1.NodePoolParameters with
// the values seen in sdkgo.KubernetesNodePool.
func LateInitializer(in *v1alpha1.NodePoolParameters, sg *sdkgo.KubernetesNodePool) { // nolint:gocyclo
	if sg == nil {
		return
	}
	// Add Properties to the Spec, if they were set by the API
	if propertiesOk, ok := sg.GetPropertiesOk(); ok && propertiesOk != nil {
		if maintenanceWindowOk, ok := propertiesOk.GetMaintenanceWindowOk(); ok && maintenanceWindowOk != nil {
			if timeOk, ok := maintenanceWindowOk.GetTimeOk(); ok && timeOk != nil {
				if utils.IsEmptyValue(reflect.ValueOf(in.MaintenanceWindow.Time)) {
					in.MaintenanceWindow.Time = *timeOk
				}
			}
			if dayOfTheWeekOk, ok := maintenanceWindowOk.GetDayOfTheWeekOk(); ok && dayOfTheWeekOk != nil {
				if utils.IsEmptyValue(reflect.ValueOf(in.MaintenanceWindow.DayOfTheWeek)) {
					in.MaintenanceWindow.DayOfTheWeek = *dayOfTheWeekOk
				}
			}
		}
		if versionOk, ok := propertiesOk.GetK8sVersionOk(); ok && versionOk != nil {
			if utils.IsEmptyValue(reflect.ValueOf(in.K8sVersion)) {
				in.K8sVersion = *versionOk
			}
		}
	}
}

// LateStatusInitializer fills the empty fields in *v1alpha1.ClusterObservation with
// the values seen in sdkgo.KubernetesCluster.
func LateStatusInitializer(in *v1alpha1.NodePoolObservation, sg *sdkgo.KubernetesNodePool) {
	if sg == nil {
		return
	}
	// Add Properties to the Spec, if they were set by the API
	if propertiesOk, ok := sg.GetPropertiesOk(); ok && propertiesOk != nil {
		if availableUpgradeVersionsOk, ok := propertiesOk.GetAvailableUpgradeVersionsOk(); ok && availableUpgradeVersionsOk != nil {
			in.AvailableUpgradeVersions = *availableUpgradeVersionsOk
		}
	}
}

// IsK8sNodePoolUpToDate returns true if the NodePool is up-to-date or false if it does not
func IsK8sNodePoolUpToDate(cr *v1alpha1.NodePool, nodepool sdkgo.KubernetesNodePool) bool { // nolint:gocyclo
	// TODO: add more updatable options to check
	switch {
	case cr == nil && nodepool.Properties == nil:
		return true
	case cr == nil && nodepool.Properties != nil:
		return false
	case cr != nil && nodepool.Properties == nil:
		return false
	case nodepool.Metadata.State != nil && *nodepool.Metadata.State == "BUSY" || *nodepool.Metadata.State == "DEPLOYING":
		return true
	case nodepool.Properties.Name != nil && *nodepool.Properties.Name != cr.Spec.ForProvider.Name:
		return false
	case nodepool.Properties.K8sVersion != nil && *nodepool.Properties.K8sVersion != cr.Spec.ForProvider.K8sVersion:
		return false
	case nodepool.Properties.MaintenanceWindow != nil && nodepool.Properties.MaintenanceWindow.Time != nil && *nodepool.Properties.MaintenanceWindow.Time != cr.Spec.ForProvider.MaintenanceWindow.Time:
		return false
	case nodepool.Properties.MaintenanceWindow != nil && nodepool.Properties.MaintenanceWindow.DayOfTheWeek != nil && *nodepool.Properties.MaintenanceWindow.DayOfTheWeek != cr.Spec.ForProvider.MaintenanceWindow.DayOfTheWeek:
		return false
	default:
		return true
	}
}

func nodepoolMaintenanceWindow(window v1alpha1.MaintenanceWindow) *sdkgo.KubernetesMaintenanceWindow {
	if window.Time != "" && window.DayOfTheWeek != "" {
		return &sdkgo.KubernetesMaintenanceWindow{
			Time:         &window.Time,
			DayOfTheWeek: &window.DayOfTheWeek,
		}
	}
	return nil
}
