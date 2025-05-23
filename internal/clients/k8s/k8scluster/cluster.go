package k8scluster

import (
	"context"
	"fmt"
	"reflect"

	sdkgo "github.com/ionos-cloud/sdk-go/v6"

	"github.com/ionos-cloud/crossplane-provider-ionoscloud/apis/k8s/v1alpha1"
	"github.com/ionos-cloud/crossplane-provider-ionoscloud/internal/clients"
	"github.com/ionos-cloud/crossplane-provider-ionoscloud/internal/clients/k8s"
	"github.com/ionos-cloud/crossplane-provider-ionoscloud/internal/compare"
	"github.com/ionos-cloud/crossplane-provider-ionoscloud/internal/utils"
)

// APIClient is a wrapper around IONOS Service
type APIClient struct {
	*clients.IonosServices
}

// Client is a wrapper around IONOS Service K8s Cluster methods
type Client interface {
	CheckDuplicateK8sCluster(ctx context.Context, clusterName string) (*sdkgo.KubernetesCluster, error)
	GetK8sClusterID(cluster *sdkgo.KubernetesCluster) (string, error)
	GetK8sCluster(ctx context.Context, clusterID string) (sdkgo.KubernetesCluster, *sdkgo.APIResponse, error)
	GetKubeConfig(ctx context.Context, clusterID string) (string, *sdkgo.APIResponse, error)
	CreateK8sCluster(ctx context.Context, cluster sdkgo.KubernetesClusterForPost) (sdkgo.KubernetesCluster, *sdkgo.APIResponse, error)
	UpdateK8sCluster(ctx context.Context, clusterID string, cluster sdkgo.KubernetesClusterForPut) (sdkgo.KubernetesCluster, *sdkgo.APIResponse, error)
	DeleteK8sCluster(ctx context.Context, clusterID string) (*sdkgo.APIResponse, error)
	HasActiveK8sNodePools(ctx context.Context, clusterID string) (bool, error)
	GetAPIClient() *sdkgo.APIClient
}

// CheckDuplicateK8sCluster based on clusterName
func (cp *APIClient) CheckDuplicateK8sCluster(ctx context.Context, clusterName string) (*sdkgo.KubernetesCluster, error) { // nolint: gocyclo
	kubernetesClusters, _, err := cp.IonosServices.ComputeClient.KubernetesApi.K8sGet(ctx).Depth(utils.DepthQueryParam).Execute()
	if err != nil {
		return nil, err
	}
	matchedItems := make([]sdkgo.KubernetesCluster, 0)
	if itemsOk, ok := kubernetesClusters.GetItemsOk(); ok && itemsOk != nil {
		for _, item := range *itemsOk {
			if propertiesOk, ok := item.GetPropertiesOk(); ok && propertiesOk != nil {
				if nameOk, ok := propertiesOk.GetNameOk(); ok && nameOk != nil {
					if *nameOk == clusterName {
						matchedItems = append(matchedItems, item)
					}
				}
			}
		}
	}
	if len(matchedItems) == 0 {
		return nil, nil
	}
	if len(matchedItems) > 1 {
		return nil, fmt.Errorf("error: found multiple clusters with the name %v", clusterName)
	}
	return &matchedItems[0], nil
}

// GetK8sClusterID based on cluster
func (cp *APIClient) GetK8sClusterID(cluster *sdkgo.KubernetesCluster) (string, error) {
	if cluster != nil {
		if idOk, ok := cluster.GetIdOk(); ok && idOk != nil {
			return *idOk, nil
		}
		return "", fmt.Errorf("error: getting cluster id")
	}
	return "", nil
}

// GetK8sCluster based on clusterID
func (cp *APIClient) GetK8sCluster(ctx context.Context, clusterID string) (sdkgo.KubernetesCluster, *sdkgo.APIResponse, error) {
	return cp.IonosServices.ComputeClient.KubernetesApi.K8sFindByClusterId(ctx, clusterID).Depth(utils.DepthQueryParam).Execute()
}

// GetKubeConfig based on clusterID
func (cp *APIClient) GetKubeConfig(ctx context.Context, clusterID string) (string, *sdkgo.APIResponse, error) {
	return cp.IonosServices.ComputeClient.KubernetesApi.K8sKubeconfigGet(ctx, clusterID).Depth(utils.DepthQueryParam).Execute()
}

// CreateK8sCluster based on KubernetesClusterForPost
func (cp *APIClient) CreateK8sCluster(ctx context.Context, cluster sdkgo.KubernetesClusterForPost) (sdkgo.KubernetesCluster, *sdkgo.APIResponse, error) {
	return cp.IonosServices.ComputeClient.KubernetesApi.K8sPost(ctx).KubernetesCluster(cluster).Execute()
}

// UpdateK8sCluster based on clusterID and KubernetesClusterForPut
func (cp *APIClient) UpdateK8sCluster(ctx context.Context, clusterID string, cluster sdkgo.KubernetesClusterForPut) (sdkgo.KubernetesCluster, *sdkgo.APIResponse, error) {
	return cp.IonosServices.ComputeClient.KubernetesApi.K8sPut(ctx, clusterID).KubernetesCluster(cluster).Execute()
}

// DeleteK8sCluster based on clusterID
func (cp *APIClient) DeleteK8sCluster(ctx context.Context, clusterID string) (*sdkgo.APIResponse, error) {
	resp, err := cp.IonosServices.ComputeClient.KubernetesApi.K8sDelete(ctx, clusterID).Execute()
	return resp, err
}

// HasActiveK8sNodePools based on clusterID
func (cp *APIClient) HasActiveK8sNodePools(ctx context.Context, clusterID string) (bool, error) {
	cluster, _, err := cp.IonosServices.ComputeClient.KubernetesApi.K8sFindByClusterId(ctx, clusterID).Depth(utils.DepthQueryParam).Execute()
	if err != nil {
		return false, err
	}
	if cluster.HasEntities() {
		if cluster.Entities.HasNodepools() {
			if cluster.Entities.Nodepools.HasItems() {
				if len(*cluster.Entities.Nodepools.Items) > 0 {
					return true, nil
				}
			}
		}
	}
	return false, nil
}

// GetAPIClient gets the APIClient
func (cp *APIClient) GetAPIClient() *sdkgo.APIClient {
	return cp.IonosServices.ComputeClient
}

// GenerateCreateK8sClusterInput returns sdkgo.KubernetesClusterForPost based on the CR spec
func GenerateCreateK8sClusterInput(cr *v1alpha1.Cluster, natGatewayIP string) *sdkgo.KubernetesClusterForPost {
	instanceCreateInput := sdkgo.KubernetesClusterForPost{
		Properties: &sdkgo.KubernetesClusterPropertiesForPost{
			Name:   &cr.Spec.ForProvider.Name,
			Public: &cr.Spec.ForProvider.Public,
		},
	}
	if !utils.IsEmptyValue(reflect.ValueOf(cr.Spec.ForProvider.K8sVersion)) {
		instanceCreateInput.Properties.SetK8sVersion(cr.Spec.ForProvider.K8sVersion)
	}
	if !utils.IsEmptyValue(reflect.ValueOf(cr.Spec.ForProvider.APISubnetAllowList)) {
		instanceCreateInput.Properties.SetApiSubnetAllowList(cr.Spec.ForProvider.APISubnetAllowList)
	}
	if !utils.IsEmptyValue(reflect.ValueOf(cr.Spec.ForProvider.S3Buckets)) {
		instanceCreateInput.Properties.SetS3Buckets(s3Buckets(cr.Spec.ForProvider.S3Buckets))
	}
	if window := clusterMaintenanceWindow(cr.Spec.ForProvider.MaintenanceWindow); window != nil {
		instanceCreateInput.Properties.SetMaintenanceWindow(*window)
	}
	if cr.Spec.ForProvider.Location != "" {
		instanceCreateInput.Properties.SetLocation(cr.Spec.ForProvider.Location)
	}
	if natGatewayIP != "" {
		instanceCreateInput.Properties.SetNatGatewayIp(natGatewayIP)
	}
	if cr.Spec.ForProvider.NodeSubnet != "" {
		instanceCreateInput.Properties.SetNodeSubnet(cr.Spec.ForProvider.NodeSubnet)
	}

	return &instanceCreateInput
}

// GenerateUpdateK8sClusterInput returns sdkgo.KubernetesClusterForPut based on the CR spec modifications
func GenerateUpdateK8sClusterInput(cr *v1alpha1.Cluster) *sdkgo.KubernetesClusterForPut {
	instanceUpdateInput := sdkgo.KubernetesClusterForPut{
		Properties: &sdkgo.KubernetesClusterPropertiesForPut{
			Name: &cr.Spec.ForProvider.Name,
		},
	}
	if !utils.IsEmptyValue(reflect.ValueOf(cr.Spec.ForProvider.APISubnetAllowList)) {
		instanceUpdateInput.Properties.ApiSubnetAllowList = apiSubnetAllowList(cr.Spec.ForProvider.APISubnetAllowList)
	}
	if !utils.IsEmptyValue(reflect.ValueOf(cr.Spec.ForProvider.K8sVersion)) {
		instanceUpdateInput.Properties.SetK8sVersion(cr.Spec.ForProvider.K8sVersion)
	}
	if !utils.IsEmptyValue(reflect.ValueOf(cr.Spec.ForProvider.S3Buckets)) {
		instanceUpdateInput.Properties.SetS3Buckets(s3Buckets(cr.Spec.ForProvider.S3Buckets))
	}
	if window := clusterMaintenanceWindow(cr.Spec.ForProvider.MaintenanceWindow); window != nil {
		instanceUpdateInput.Properties.SetMaintenanceWindow(*window)
	}

	return &instanceUpdateInput
}

// LateInitializer fills the empty fields in *v1alpha1.ClusterParameters with
// the values seen in sdkgo.KubernetesCluster.
func LateInitializer(in *v1alpha1.ClusterParameters, sg *sdkgo.KubernetesCluster) { // nolint:gocyclo
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
		if apiSubnetAllowListOk, ok := propertiesOk.GetApiSubnetAllowListOk(); ok && apiSubnetAllowListOk != nil {
			if !utils.IsEmptyValue(reflect.ValueOf(in.APISubnetAllowList)) && utils.ContainsStringSlices(in.APISubnetAllowList, *apiSubnetAllowListOk) {
				in.APISubnetAllowList = *apiSubnetAllowListOk
			}
		}
	}
}

// LateStatusInitializer fills the empty fields in *v1alpha1.ClusterObservation with
// the values seen in sdkgo.KubernetesCluster.
func LateStatusInitializer(in *v1alpha1.ClusterObservation, sg *sdkgo.KubernetesCluster) {
	if sg == nil {
		return
	}
	// Add Properties to the Status, if they were set by the API
	if propertiesOk, ok := sg.GetPropertiesOk(); ok && propertiesOk != nil {
		if availableUpgradeVersionsOk, ok := propertiesOk.GetAvailableUpgradeVersionsOk(); ok && availableUpgradeVersionsOk != nil {
			in.AvailableUpgradeVersions = *availableUpgradeVersionsOk
		}
		if viableNodePoolVersionsOk, ok := propertiesOk.GetViableNodePoolVersionsOk(); ok && viableNodePoolVersionsOk != nil {
			in.ViableNodePoolVersions = *viableNodePoolVersionsOk
		}
		if versionOk, ok := propertiesOk.GetK8sVersionOk(); ok && versionOk != nil {
			in.K8sVersion = *versionOk
		}
	}
}

// IsK8sClusterUpToDate returns true if the K8sCluster is up-to-date or false if it does not
func IsK8sClusterUpToDate(cr *v1alpha1.Cluster, cluster sdkgo.KubernetesCluster) bool { // nolint:gocyclo
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
	case cluster.Properties.K8sVersion != nil && cr.Spec.ForProvider.K8sVersion != "" && *cluster.Properties.K8sVersion != cr.Spec.ForProvider.K8sVersion:
		return false
	case cluster.Properties.S3Buckets != nil && !isEqS3Buckets(cr.Spec.ForProvider.S3Buckets, *cluster.Properties.S3Buckets):
		return false
	case cluster.Properties.ApiSubnetAllowList != nil && !utils.ContainsStringSlices(*cluster.Properties.ApiSubnetAllowList, cr.Spec.ForProvider.APISubnetAllowList):
		return false
	case !compare.EqualKubernetesMaintenanceWindow(cr.Spec.ForProvider.MaintenanceWindow, cluster.Properties.MaintenanceWindow):
		return false
	case cluster.Properties.Public != nil && *cluster.Properties.Public != cr.Spec.ForProvider.Public:
		return false
	default:
		return true
	}
}

func clusterMaintenanceWindow(window v1alpha1.MaintenanceWindow) *sdkgo.KubernetesMaintenanceWindow {
	if window.Time != "" && window.DayOfTheWeek != "" {
		return &sdkgo.KubernetesMaintenanceWindow{
			Time:         &window.Time,
			DayOfTheWeek: &window.DayOfTheWeek,
		}
	}
	return nil
}

func s3Buckets(s3BucketSpecs []v1alpha1.S3Bucket) []sdkgo.S3Bucket {
	buckets := make([]sdkgo.S3Bucket, 0)
	for _, s3BucketSpec := range s3BucketSpecs {
		s3BucketName := s3BucketSpec.Name
		if s3BucketName != "" {
			buckets = append(buckets, sdkgo.S3Bucket{
				Name: &s3BucketName,
			})
		}
	}
	return buckets
}

func apiSubnetAllowList(setAPISubnetAllowList []string) *[]string {
	apiSubnets := make([]string, 0)
	for _, apiSubnet := range setAPISubnetAllowList {
		if apiSubnet != "" {
			apiSubnets = append(apiSubnets, apiSubnet)
		}
	}
	return &apiSubnets
}

func isEqS3Buckets(crBuckets []v1alpha1.S3Bucket, buckets []sdkgo.S3Bucket) bool {
	if len(crBuckets) != len(buckets) {
		return false
	}
	for i, crBucket := range crBuckets {
		lan := buckets[i]
		if lan.Name != nil && crBucket.Name != *lan.Name {
			return false
		}
	}
	return true
}
