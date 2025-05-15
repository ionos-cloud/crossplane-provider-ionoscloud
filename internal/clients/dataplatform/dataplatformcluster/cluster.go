package dataplatformcluster

import (
	"context"
	"fmt"

	sdkgo "github.com/ionos-cloud/sdk-go-dataplatform"

	"github.com/ionos-cloud/crossplane-provider-ionoscloud/apis/dataplatform/v1alpha1"
	"github.com/ionos-cloud/crossplane-provider-ionoscloud/internal/clients"
	"github.com/ionos-cloud/crossplane-provider-ionoscloud/internal/clients/k8s"
	"github.com/ionos-cloud/crossplane-provider-ionoscloud/internal/compare"
)

// APIClient is a wrapper around IONOS Service
type APIClient struct {
	*clients.IonosServices
}

// Client is a wrapper around IONOS Service K8s Cluster methods
type Client interface {
	GetDataplatformClusterByID(ctx context.Context, clusterID string) (sdkgo.ClusterResponseData, *sdkgo.APIResponse, error)
	GetKubeConfig(ctx context.Context, clusterID string) (map[string]any, *sdkgo.APIResponse, error)
	CreateDataplatformCluster(ctx context.Context, cluster sdkgo.CreateClusterRequest) (sdkgo.ClusterResponseData, *sdkgo.APIResponse, error)
	PatchDataPlatformCluster(ctx context.Context, clusterID string, cluster sdkgo.PatchClusterRequest) (sdkgo.ClusterResponseData, *sdkgo.APIResponse, error)
	DeleteDataPlatformCluster(ctx context.Context, clusterID string) (*sdkgo.APIResponse, error)
	IsDataplatformDeleted(ctx context.Context, id ...string) (bool, error)
	GetAPIClient() *sdkgo.APIClient
}

// GetDataplatformClusterByID based on clusterID
func (dp *APIClient) GetDataplatformClusterByID(ctx context.Context, clusterID string) (sdkgo.ClusterResponseData, *sdkgo.APIResponse, error) {
	return dp.DataplatformClient.DataPlatformClusterApi.ClustersFindById(ctx, clusterID).Execute()
}

// GetKubeConfig based on clusterID
func (dp *APIClient) GetKubeConfig(ctx context.Context, clusterID string) (map[string]any, *sdkgo.APIResponse, error) {
	return dp.DataplatformClient.DataPlatformClusterApi.ClustersKubeconfigFindByClusterId(ctx, clusterID).Execute()
}

// CreateDataplatformCluster based on ClustersPost
func (dp *APIClient) CreateDataplatformCluster(ctx context.Context, cluster sdkgo.CreateClusterRequest) (sdkgo.ClusterResponseData, *sdkgo.APIResponse, error) {
	return dp.DataplatformClient.DataPlatformClusterApi.ClustersPost(ctx).CreateClusterRequest(cluster).Execute()
}

// PatchDataPlatformCluster based on clusterID and ClustersPatch
func (dp *APIClient) PatchDataPlatformCluster(ctx context.Context, clusterID string, cluster sdkgo.PatchClusterRequest) (sdkgo.ClusterResponseData, *sdkgo.APIResponse, error) {
	return dp.DataplatformClient.DataPlatformClusterApi.ClustersPatch(ctx, clusterID).PatchClusterRequest(cluster).Execute()
}

// DeleteDataPlatformCluster based on clusterID
func (dp *APIClient) DeleteDataPlatformCluster(ctx context.Context, clusterID string) (*sdkgo.APIResponse, error) {
	_, resp, err := dp.DataplatformClient.DataPlatformClusterApi.ClustersDelete(ctx, clusterID).Execute()
	return resp, err
}

// IsDataplatformDeleted returns true if the dataplatform cluster is deleted
func (dp *APIClient) IsDataplatformDeleted(ctx context.Context, ids ...string) (bool, error) {
	if len(ids) != 1 {
		return false, fmt.Errorf("error checking dataplatform nodepool deletion status: %w", fmt.Errorf("expected 2 ids, got %d", len(ids)))
	}
	id := ids[0]
	_, apiResponse, err := dp.GetDataplatformClusterByID(ctx, id)
	if err != nil {
		if apiResponse.HttpNotFound() {
			return true, nil
		}
		return false, fmt.Errorf("error checking dataplatform cluster deletion status: %w", err)
	}
	return false, nil
}

// GetAPIClient gets the APIClient
func (dp *APIClient) GetAPIClient() *sdkgo.APIClient {
	return dp.DataplatformClient
}

// GenerateCreateInput returns sdkgo.KubernetesClusterForPost based on the CR spec
func GenerateCreateInput(cr *v1alpha1.DataplatformCluster) *sdkgo.CreateClusterRequest {
	instanceCreateInput := sdkgo.CreateClusterRequest{
		Properties: &sdkgo.CreateClusterProperties{
			Name:         &cr.Spec.ForProvider.Name,
			DatacenterId: &cr.Spec.ForProvider.DatacenterCfg.DatacenterID,
		},
	}
	if cr.Spec.ForProvider.Version != "" {
		instanceCreateInput.Properties.DataPlatformVersion = &cr.Spec.ForProvider.Version
	}
	if window := clusterMaintenanceWindow(cr.Spec.ForProvider.MaintenanceWindow); window != nil {
		instanceCreateInput.Properties.SetMaintenanceWindow(*window)
	}
	return &instanceCreateInput
}

// GenerateUpdateInput returns sdkgo.KubernetesClusterForPut based on the CR spec modifications
func GenerateUpdateInput(cr *v1alpha1.DataplatformCluster) *sdkgo.PatchClusterRequest {
	instanceUpdateInput := sdkgo.PatchClusterRequest{
		Properties: &sdkgo.PatchClusterProperties{
			Name: &cr.Spec.ForProvider.Name,
		},
	}
	if cr.Spec.ForProvider.Version != "" {
		instanceUpdateInput.Properties.DataPlatformVersion = &cr.Spec.ForProvider.Version
	}
	if window := clusterMaintenanceWindow(cr.Spec.ForProvider.MaintenanceWindow); window != nil {
		instanceUpdateInput.Properties.SetMaintenanceWindow(*window)
	}
	return &instanceUpdateInput
}

// LateInitializer fills the empty fields in *v1alpha1.ClusterParameters with
// the values seen in sdkgo.KubernetesCluster.
func LateInitializer(in *v1alpha1.DataplatformClusterParameters, sg *sdkgo.ClusterResponseData) { // nolint:gocyclo
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

// LateStatusInitializer fills the empty fields in *v1alpha1.ClusterObservation with
// the values seen in sdkgo.KubernetesCluster.
func LateStatusInitializer(in *v1alpha1.DataplatformClusterObservation, sg *sdkgo.ClusterResponseData) {
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
func IsUpToDate(cr *v1alpha1.DataplatformCluster, cluster sdkgo.ClusterResponseData) bool { // nolint:gocyclo
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
