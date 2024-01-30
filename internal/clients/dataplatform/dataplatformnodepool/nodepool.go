package dataplatformnodepool

import (
	"context"
	"fmt"

	sdkgo "github.com/ionos-cloud/sdk-go-dataplatform"

	"github.com/ionos-cloud/crossplane-provider-ionoscloud/apis/dataplatform/v1alpha1"
	"github.com/ionos-cloud/crossplane-provider-ionoscloud/internal/clients"
	"github.com/ionos-cloud/crossplane-provider-ionoscloud/internal/clients/k8s"
	"github.com/ionos-cloud/crossplane-provider-ionoscloud/internal/compare"
	"github.com/ionos-cloud/crossplane-provider-ionoscloud/internal/utils"
)

// APIClient is a wrapper around IONOS Service
type APIClient struct {
	*clients.IonosServices
}

// Client is a wrapper around IONOS Service K8s Nodepool methods
type Client interface {
	GetDataplatformNodepoolByID(ctx context.Context, clusterID, nodepoolID string) (sdkgo.NodePoolResponseData, *sdkgo.APIResponse, error)
	CreateDataplatformNodepool(ctx context.Context, clusterID string, cluster sdkgo.CreateNodePoolRequest) (sdkgo.NodePoolResponseData, *sdkgo.APIResponse, error)
	PatchDataPlatformNodepool(ctx context.Context, clusterID, nodepoolID string, cluster sdkgo.PatchNodePoolRequest) (sdkgo.NodePoolResponseData, *sdkgo.APIResponse, error)
	DeleteDataPlatformNodepool(ctx context.Context, clusterID, nodepoolID string) (*sdkgo.APIResponse, error)
	IsDataplatformDeleted(ctx context.Context, ids ...string) (bool, error)
	GetAPIClient() *sdkgo.APIClient
}

// GetDataplatformNodepoolByID based on clusterID
func (dp *APIClient) GetDataplatformNodepoolByID(ctx context.Context, clusterID, nodepoolID string) (sdkgo.NodePoolResponseData, *sdkgo.APIResponse, error) {
	return dp.DataplatformClient.DataPlatformNodePoolApi.ClustersNodepoolsFindById(ctx, clusterID, nodepoolID).Execute()
}

// CreateDataplatformNodepool based on NodepoolsPost
func (dp *APIClient) CreateDataplatformNodepool(ctx context.Context, clusterID string, cluster sdkgo.CreateNodePoolRequest) (sdkgo.NodePoolResponseData, *sdkgo.APIResponse, error) {
	return dp.DataplatformClient.DataPlatformNodePoolApi.ClustersNodepoolsPost(ctx, clusterID).CreateNodePoolRequest(cluster).Execute()
}

// PatchDataPlatformNodepool based on clusterID and NodePoolsPatch
func (dp *APIClient) PatchDataPlatformNodepool(ctx context.Context, clusterID, nodepoolID string, cluster sdkgo.PatchNodePoolRequest) (sdkgo.NodePoolResponseData, *sdkgo.APIResponse, error) {
	return dp.DataplatformClient.DataPlatformNodePoolApi.ClustersNodepoolsPatch(ctx, clusterID, nodepoolID).PatchNodePoolRequest(cluster).Execute()
}

// DeleteDataPlatformNodepool based on clusterID and nodepoolID
func (dp *APIClient) DeleteDataPlatformNodepool(ctx context.Context, clusterID, nodepoolID string) (*sdkgo.APIResponse, error) {
	_, resp, err := dp.DataplatformClient.DataPlatformNodePoolApi.ClustersNodepoolsDelete(ctx, clusterID, nodepoolID).Execute()
	return resp, err
}

// IsDataplatformDeleted returns true if the dataplatform cluster is deleted
func (dp *APIClient) IsDataplatformDeleted(ctx context.Context, ids ...string) (bool, error) {
	if len(ids) != 2 {
		return false, fmt.Errorf("error checking dataplatform nodepool deletion status: %w", fmt.Errorf("expected 2 ids, got %d", len(ids)))
	}
	clusterID := ids[0]
	nodepoolID := ids[1]
	_, apiResponse, err := dp.GetDataplatformNodepoolByID(ctx, clusterID, nodepoolID)
	if err != nil {
		if apiResponse.HttpNotFound() {
			return true, nil
		}
		return false, fmt.Errorf("error checking dataplatform nodepool deletion status: %w", err)
	}
	return false, nil
}

// GetAPIClient gets the APIClient
func (dp *APIClient) GetAPIClient() *sdkgo.APIClient {
	return dp.DataplatformClient
}

// GenerateCreateInput returns sdkgo.KubernetesNodepoolForPost based on the CR spec
func GenerateCreateInput(cr *v1alpha1.DataplatformNodepool) *sdkgo.CreateNodePoolRequest {
	instanceCreateInput := sdkgo.CreateNodePoolRequest{
		Properties: &sdkgo.CreateNodePoolProperties{
			Name:      &cr.Spec.ForProvider.Name,
			NodeCount: &cr.Spec.ForProvider.NodeCount,
		},
	}
	if cpuFamily := cr.Spec.ForProvider.CpuFamily; cpuFamily != "" {
		instanceCreateInput.Properties.CpuFamily = &cpuFamily
	}
	if coresCount := cr.Spec.ForProvider.CoresCount; coresCount != 0 {
		instanceCreateInput.Properties.CoresCount = &coresCount
	}
	if ramSize := cr.Spec.ForProvider.RamSize; ramSize != 0 {
		instanceCreateInput.Properties.RamSize = &ramSize
	}
	if availabilityZone := cr.Spec.ForProvider.AvailabilityZone; availabilityZone != "" {
		instanceCreateInput.Properties.AvailabilityZone = (*sdkgo.AvailabilityZone)(&availabilityZone)
	}
	if labels := cr.Spec.ForProvider.Labels; len(labels) != 0 {
		anyLabels := utils.MapStringToAny(labels)
		instanceCreateInput.Properties.Labels = &anyLabels
	}
	if annotations := cr.Spec.ForProvider.Annotations; len(annotations) != 0 {
		anyAnnotations := utils.MapStringToAny(annotations)
		instanceCreateInput.Properties.Annotations = &anyAnnotations
	}
	if window := clusterMaintenanceWindow(cr.Spec.ForProvider.MaintenanceWindow); window != nil {
		instanceCreateInput.Properties.MaintenanceWindow = window
	}
	return &instanceCreateInput
}

// GenerateUpdateInput returns sdkgo.KubernetesNodepoolForPut based on the CR spec modifications
func GenerateUpdateInput(cr *v1alpha1.DataplatformNodepool) *sdkgo.PatchNodePoolRequest {
	instanceUpdateInput := sdkgo.PatchNodePoolRequest{
		Properties: &sdkgo.PatchNodePoolProperties{
			NodeCount: &cr.Spec.ForProvider.NodeCount,
		},
	}
	if labels := cr.Spec.ForProvider.Labels; len(labels) != 0 {
		anyLabels := utils.MapStringToAny(labels)
		instanceUpdateInput.Properties.Labels = &anyLabels
	}
	if annotations := cr.Spec.ForProvider.Annotations; len(annotations) != 0 {
		anyAnnotations := utils.MapStringToAny(annotations)
		instanceUpdateInput.Properties.Annotations = &anyAnnotations
	}
	if window := clusterMaintenanceWindow(cr.Spec.ForProvider.MaintenanceWindow); window != nil {
		instanceUpdateInput.Properties.SetMaintenanceWindow(*window)
	}
	return &instanceUpdateInput
}

// LateInitializer fills the empty fields in *v1alpha1.NodepoolParameters with
// the values seen in sdkgo.KubernetesNodepool.
func LateInitializer(in *v1alpha1.DataplatformNodepoolParameters, sg *sdkgo.NodePoolResponseData) { // nolint:gocyclo
	if sg == nil {
		return
	}
	// Add Properties to the Spec, if they were set by the API
	if propertiesOk, ok := sg.GetPropertiesOk(); ok && propertiesOk != nil {
		if maintenanceWindowOk, ok := propertiesOk.GetMaintenanceWindowOk(); ok && maintenanceWindowOk != nil {
			if timeOk, ok := maintenanceWindowOk.GetTimeOk(); ok && timeOk != nil {
				if in.MaintenanceWindow.Time == "" {
					in.MaintenanceWindow.Time = *timeOk
				}
			}
			if dayOfTheWeekOk, ok := maintenanceWindowOk.GetDayOfTheWeekOk(); ok && dayOfTheWeekOk != nil {
				if in.MaintenanceWindow.DayOfTheWeek == "" {
					in.MaintenanceWindow.DayOfTheWeek = *dayOfTheWeekOk
				}
			}
		}
		if version, ok := propertiesOk.GetDataPlatformVersionOk(); ok && version != nil {
			if in.Version == "" {
				in.Version = *version
			}
		}
	}
}

// LateStatusInitializer fills the empty fields in *v1alpha1.NodepoolObservation with
// the values seen in sdkgo.KubernetesNodepool.
func LateStatusInitializer(in *v1alpha1.DataplatformNodepoolObservation, sg *sdkgo.NodePoolResponseData) {
	if sg == nil {
		return
	}
	// Add Properties to the Spec, if they were set by the API
	if propertiesOk, ok := sg.GetPropertiesOk(); ok && propertiesOk != nil {
		if version, ok := propertiesOk.GetDataPlatformVersionOk(); ok && version != nil {
			in.Version = *version
		}
	}
}

// IsUpToDate returns true if the dataplatform cluster is up-to-date or false if it does not
func IsUpToDate(cr *v1alpha1.DataplatformNodepool, cluster sdkgo.NodePoolResponseData) bool { // nolint:gocyclo
	switch {
	case cr == nil && cluster.Properties == nil:
		return true
	case cr == nil && cluster.Properties != nil:
		return false
	case cr != nil && cluster.Properties == nil:
		return false
	case cluster.Metadata != nil && cluster.Metadata.State != nil && (*cluster.Metadata.State == k8s.BUSY || *cluster.Metadata.State == k8s.DEPLOYING):
		return true
	case cluster.Properties.Name != nil && *cluster.Properties.Name != cr.Spec.ForProvider.Name:
		return false
	case cluster.Properties.Name == nil && cr.Spec.ForProvider.Name != "":
		return false
	case !compare.EqualMaintenanceWindow(&cr.Spec.ForProvider.MaintenanceWindow, cluster.Properties.MaintenanceWindow):
		return false
	default:
		return true
	}
}

func clusterMaintenanceWindow(window v1alpha1.MaintenanceWindow) *sdkgo.MaintenanceWindow {
	if window.Time != "" && window.DayOfTheWeek != "" {
		return &sdkgo.MaintenanceWindow{
			Time:         &window.Time,
			DayOfTheWeek: &window.DayOfTheWeek,
		}
	}
	return nil
}
