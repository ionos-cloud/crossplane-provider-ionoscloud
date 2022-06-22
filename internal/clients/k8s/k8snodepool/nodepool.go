package k8snodepool

import (
	"context"
	"reflect"
	"strconv"

	sdkgo "github.com/ionos-cloud/sdk-go/v6"

	"github.com/ionos-cloud/crossplane-provider-ionoscloud/apis/k8s/v1alpha1"
	"github.com/ionos-cloud/crossplane-provider-ionoscloud/internal/clients"
	"github.com/ionos-cloud/crossplane-provider-ionoscloud/internal/compare"
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
func GenerateCreateK8sNodePoolInput(cr *v1alpha1.NodePool, publicIPs []string) *sdkgo.KubernetesNodePoolForPost {
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
	if !utils.IsEmptyValue(reflect.ValueOf(cr.Spec.ForProvider.K8sVersion)) {
		instanceCreateInput.Properties.SetK8sVersion(cr.Spec.ForProvider.K8sVersion)
	}
	if !utils.IsEmptyValue(reflect.ValueOf(cr.Spec.ForProvider.Labels)) {
		instanceCreateInput.Properties.SetLabels(cr.Spec.ForProvider.Labels)
	}
	if !utils.IsEmptyValue(reflect.ValueOf(cr.Spec.ForProvider.AutoScaling)) {
		instanceCreateInput.Properties.SetAutoScaling(sdkgo.KubernetesAutoScaling{
			MinNodeCount: &cr.Spec.ForProvider.AutoScaling.MinNodeCount,
			MaxNodeCount: &cr.Spec.ForProvider.AutoScaling.MaxNodeCount,
		})
	}
	if !utils.IsEmptyValue(reflect.ValueOf(cr.Spec.ForProvider.Annotations)) {
		instanceCreateInput.Properties.SetAnnotations(cr.Spec.ForProvider.Annotations)
	}
	if !utils.IsEmptyValue(reflect.ValueOf(publicIPs)) {
		instanceCreateInput.Properties.SetPublicIps(publicIPs)
	}
	if window := nodepoolMaintenanceWindow(cr.Spec.ForProvider.MaintenanceWindow); window != nil {
		instanceCreateInput.Properties.SetMaintenanceWindow(*window)
	}
	if poolLans := kubernetesNodePoolLans(cr.Spec.ForProvider.Lans); poolLans != nil {
		instanceCreateInput.Properties.SetLans(*poolLans)
	}
	return &instanceCreateInput
}

// GenerateUpdateK8sNodePoolInput returns sdkgo.KubernetesNodePoolForPut based on the CR spec modifications
func GenerateUpdateK8sNodePoolInput(cr *v1alpha1.NodePool, publicIps []string) *sdkgo.KubernetesNodePoolForPut {
	instanceUpdateInput := sdkgo.KubernetesNodePoolForPut{
		Properties: &sdkgo.KubernetesNodePoolPropertiesForPut{
			NodeCount:  &cr.Spec.ForProvider.NodeCount,
			K8sVersion: &cr.Spec.ForProvider.K8sVersion,
		},
	}
	if !utils.IsEmptyValue(reflect.ValueOf(cr.Spec.ForProvider.AutoScaling.MinNodeCount)) {
		instanceUpdateInput.Properties.SetAutoScaling(sdkgo.KubernetesAutoScaling{
			MinNodeCount: &cr.Spec.ForProvider.AutoScaling.MinNodeCount,
			MaxNodeCount: &cr.Spec.ForProvider.AutoScaling.MaxNodeCount,
		})
	}
	if !utils.IsEmptyValue(reflect.ValueOf(cr.Spec.ForProvider.Annotations)) {
		instanceUpdateInput.Properties.SetAnnotations(cr.Spec.ForProvider.Annotations)
	} else {
		instanceUpdateInput.Properties.SetAnnotations(map[string]string{})
	}
	if !utils.IsEmptyValue(reflect.ValueOf(cr.Spec.ForProvider.Labels)) {
		instanceUpdateInput.Properties.SetLabels(cr.Spec.ForProvider.Labels)
	} else {
		instanceUpdateInput.Properties.SetLabels(map[string]string{})
	}
	if !utils.IsEmptyValue(reflect.ValueOf(publicIps)) {
		instanceUpdateInput.Properties.SetPublicIps(publicIps)
	} else {
		instanceUpdateInput.Properties.SetPublicIps([]string{})
	}
	if window := nodepoolMaintenanceWindow(cr.Spec.ForProvider.MaintenanceWindow); window != nil {
		instanceUpdateInput.Properties.SetMaintenanceWindow(*window)
	}
	if poolLans := kubernetesNodePoolLans(cr.Spec.ForProvider.Lans); poolLans != nil {
		instanceUpdateInput.Properties.SetLans(*poolLans)
	}
	return &instanceUpdateInput
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
		if publicIPsOk, ok := propertiesOk.GetPublicIpsOk(); ok && publicIPsOk != nil {
			in.PublicIPs = *publicIPsOk
		} else {
			in.PublicIPs = []string{}
		}

		if cpuFamilyOk, ok := propertiesOk.GetCpuFamilyOk(); ok && cpuFamilyOk != nil {
			in.CPUFamily = *cpuFamilyOk
		}
	}

}

// IsK8sNodePoolUpToDate returns true if the NodePool is up-to-date or false if it does not
func IsK8sNodePoolUpToDate(cr *v1alpha1.NodePool, nodepool sdkgo.KubernetesNodePool, publicIPs []string) bool { // nolint:gocyclo
	switch {
	case cr == nil && nodepool.Properties == nil:
		return true
	case cr == nil && nodepool.Properties != nil:
		return false
	case cr != nil && nodepool.Properties == nil:
		return false
	case nodepool.Metadata != nil && nodepool.Metadata.State != nil && (*nodepool.Metadata.State == "BUSY" || *nodepool.Metadata.State == "DEPLOYING"):
		return true
	case nodepool.Properties.K8sVersion != nil && *nodepool.Properties.K8sVersion != cr.Spec.ForProvider.K8sVersion:
		return false
	case nodepool.Properties.NodeCount != nil && *nodepool.Properties.NodeCount != cr.Spec.ForProvider.NodeCount:
		return false
	case nodepool.Properties.PublicIps != nil && !utils.ContainsStringSlices(*nodepool.Properties.PublicIps, publicIPs):
		return false
	case nodepool.Properties.Labels != nil && !utils.IsEqStringMaps(*nodepool.Properties.Labels, cr.Spec.ForProvider.Labels):
		return false
	case nodepool.Properties.Annotations != nil && !utils.IsEqStringMaps(*nodepool.Properties.Annotations, cr.Spec.ForProvider.Annotations):
		return false
	case nodepool.Properties.AutoScaling != nil && nodepool.Properties.AutoScaling.MinNodeCount != nil && *nodepool.Properties.AutoScaling.MinNodeCount != cr.Spec.ForProvider.AutoScaling.MinNodeCount:
		return false
	case nodepool.Properties.AutoScaling != nil && nodepool.Properties.AutoScaling.MaxNodeCount != nil && *nodepool.Properties.AutoScaling.MaxNodeCount != cr.Spec.ForProvider.AutoScaling.MaxNodeCount:
		return false
	case !compare.EqualKubernetesMaintenanceWindow(cr.Spec.ForProvider.MaintenanceWindow, nodepool.Properties.MaintenanceWindow):
		return false
	case nodepool.Properties.Lans != nil && !isEqKubernetesNodePoolLans(cr.Spec.ForProvider.Lans, *nodepool.Properties.Lans):
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

func kubernetesNodePoolLans(crLans []v1alpha1.KubernetesNodePoolLan) *[]sdkgo.KubernetesNodePoolLan {
	lans := make([]sdkgo.KubernetesNodePoolLan, 0)
	if len(crLans) > 0 {
		for _, crLan := range crLans {
			lanIDConverted, _ := strconv.ParseInt(crLan.LanCfg.LanID, 10, 64)
			lanID := int32(lanIDConverted)
			newNodePoolLan := sdkgo.KubernetesNodePoolLan{
				Id:   &lanID,
				Dhcp: &crLan.Dhcp,
			}
			if len(crLan.Routes) > 0 {
				routes := make([]sdkgo.KubernetesNodePoolLanRoutes, 0)
				for _, route := range crLan.Routes {
					network := route.Network
					gatewayIP := route.GatewayIP
					routes = append(routes, sdkgo.KubernetesNodePoolLanRoutes{
						Network:   &network,
						GatewayIp: &gatewayIP,
					})
				}
				newNodePoolLan.SetRoutes(routes)
			}
			lans = append(lans, newNodePoolLan)
		}
	}
	return &lans
}

func isEqKubernetesNodePoolLans(crLans []v1alpha1.KubernetesNodePoolLan, lans []sdkgo.KubernetesNodePoolLan) bool { // nolint:gocyclo
	if len(crLans) != len(lans) {
		return false
	}
	for i, crLan := range crLans {
		lan := lans[i]
		if lan.Dhcp != nil && crLan.Dhcp != *lan.Dhcp {
			return false
		}
		if lan.Routes != nil && len(*lan.Routes) != len(crLan.Routes) {
			return false
		}
		if lan.Routes == nil && len(crLan.Routes) > 0 {
			return false
		}
		lanIDConverted, _ := strconv.ParseInt(crLan.LanCfg.LanID, 10, 64)
		if lan.Id != nil && *lan.Id != int32(lanIDConverted) {
			return false
		}
		for j, crRoute := range crLan.Routes {
			if lan.Routes != nil {
				routes := *lan.Routes
				route := routes[j]
				if route.GatewayIp != nil && *route.GatewayIp != crRoute.GatewayIP {
					return false
				}
				if route.Network != nil && *route.Network != crRoute.Network {
					return false
				}
			}
		}
	}
	return true
}
