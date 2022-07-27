package postgrescluster

import (
	"context"
	"fmt"
	"reflect"
	"time"

	"github.com/ionos-cloud/crossplane-provider-ionoscloud/internal/compare"
	"github.com/ionos-cloud/crossplane-provider-ionoscloud/internal/utils"

	ionoscloud "github.com/ionos-cloud/sdk-go-dbaas-postgres"

	"github.com/ionos-cloud/crossplane-provider-ionoscloud/apis/dbaas/postgres/v1alpha1"
	"github.com/ionos-cloud/crossplane-provider-ionoscloud/internal/clients"
)

// ClusterAPIClient is a wrapper around IONOS Service DBaaS Postgres Cluster
type ClusterAPIClient struct {
	*clients.IonosServices
}

// ClusterClient is a wrapper around IONOS Service DBaaS Postgres Cluster methods
type ClusterClient interface {
	CheckDuplicateCluster(ctx context.Context, clusterName string, cr *v1alpha1.PostgresCluster) (*ionoscloud.ClusterResponse, error)
	GetClusterID(cluster *ionoscloud.ClusterResponse) (string, error)
	GetCluster(ctx context.Context, clusterID string) (ionoscloud.ClusterResponse, *ionoscloud.APIResponse, error)
	DeleteCluster(ctx context.Context, clusterID string) (*ionoscloud.APIResponse, error)
	CreateCluster(ctx context.Context, cluster ionoscloud.CreateClusterRequest) (ionoscloud.ClusterResponse, *ionoscloud.APIResponse, error)
	UpdateCluster(ctx context.Context, clusterID string, cluster ionoscloud.PatchClusterRequest) (ionoscloud.ClusterResponse, *ionoscloud.APIResponse, error)
}

// CheckDuplicateCluster based on clusterName and on multiple properties from CR spec
func (cp *ClusterAPIClient) CheckDuplicateCluster(ctx context.Context, clusterName string, cr *v1alpha1.PostgresCluster) (*ionoscloud.ClusterResponse, error) { // nolint: gocyclo
	clusterList, _, err := cp.DBaaSPostgresClient.ClustersApi.ClustersGet(ctx).Execute()
	if err != nil {
		return nil, err
	}
	matchedItems := make([]ionoscloud.ClusterResponse, 0)
	if itemsOk, ok := clusterList.GetItemsOk(); ok && itemsOk != nil {
		for _, item := range *itemsOk {
			if propertiesOk, ok := item.GetPropertiesOk(); ok && propertiesOk != nil {
				if nameOk, ok := propertiesOk.GetDisplayNameOk(); ok && nameOk != nil {
					if *nameOk == clusterName {
						// After checking the name, check the immutable properties
						if locationOk, ok := propertiesOk.GetLocationOk(); ok && locationOk != nil {
							if *locationOk != cr.Spec.ForProvider.Location {
								return nil, fmt.Errorf("error: found cluster with the name %v, but immutable property location different. expected: %v actual: %v", clusterName, cr.Spec.ForProvider.Location, *locationOk)
							}
						}
						if storageTypeOk, ok := propertiesOk.GetStorageTypeOk(); ok && storageTypeOk != nil {
							if string(*storageTypeOk) != cr.Spec.ForProvider.StorageType {
								return nil, fmt.Errorf("error: found cluster with the name %v, but immutable property storageType different. expected: %v actual: %v", clusterName, cr.Spec.ForProvider.StorageType, string(*storageTypeOk))
							}
						}
						if backupLocationOk, ok := propertiesOk.GetBackupLocationOk(); ok && backupLocationOk != nil && cr.Spec.ForProvider.BackupLocation != "" {
							if *backupLocationOk != cr.Spec.ForProvider.BackupLocation {
								return nil, fmt.Errorf("error: found cluster with the name %v, but immutable property backupLocation different. expected: %v actual: %v", clusterName, cr.Spec.ForProvider.BackupLocation, *backupLocationOk)
							}
						}
						if synchronizationModeOk, ok := propertiesOk.GetSynchronizationModeOk(); ok && synchronizationModeOk != nil {
							if string(*synchronizationModeOk) != cr.Spec.ForProvider.SynchronizationMode {
								return nil, fmt.Errorf("error: found cluster with the name %v, but immutable property synchronizationMode different. expected: %v actual: %v", clusterName, cr.Spec.ForProvider.SynchronizationMode, *synchronizationModeOk)
							}
						}
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

// GetClusterID based on cluster
func (cp *ClusterAPIClient) GetClusterID(cluster *ionoscloud.ClusterResponse) (string, error) {
	if cluster != nil {
		if idOk, ok := cluster.GetIdOk(); ok && idOk != nil {
			return *idOk, nil
		}
		return "", fmt.Errorf("error: getting cluster id")
	}
	return "", nil
}

// GetCluster based on clusterID
func (cp *ClusterAPIClient) GetCluster(ctx context.Context, clusterID string) (ionoscloud.ClusterResponse, *ionoscloud.APIResponse, error) {
	return cp.DBaaSPostgresClient.ClustersApi.ClustersFindById(ctx, clusterID).Execute()
}

// DeleteCluster based on clusterID
func (cp *ClusterAPIClient) DeleteCluster(ctx context.Context, clusterID string) (*ionoscloud.APIResponse, error) {
	_, apiResponse, err := cp.DBaaSPostgresClient.ClustersApi.ClustersDelete(ctx, clusterID).Execute()
	return apiResponse, err
}

// CreateCluster based on cluster properties
func (cp *ClusterAPIClient) CreateCluster(ctx context.Context, cluster ionoscloud.CreateClusterRequest) (ionoscloud.ClusterResponse, *ionoscloud.APIResponse, error) {
	return cp.DBaaSPostgresClient.ClustersApi.ClustersPost(ctx).CreateClusterRequest(cluster).Execute()
}

// UpdateCluster based on clusterID and cluster properties
func (cp *ClusterAPIClient) UpdateCluster(ctx context.Context, clusterID string, cluster ionoscloud.PatchClusterRequest) (ionoscloud.ClusterResponse, *ionoscloud.APIResponse, error) {
	return cp.DBaaSPostgresClient.ClustersApi.ClustersPatch(ctx, clusterID).PatchClusterRequest(cluster).Execute()
}

// GenerateCreateClusterInput returns CreateClusterRequest based on the CR spec
func GenerateCreateClusterInput(cr *v1alpha1.PostgresCluster) (*ionoscloud.CreateClusterRequest, error) {
	instanceCreateInput := ionoscloud.CreateClusterRequest{
		Properties: &ionoscloud.CreateClusterProperties{
			PostgresVersion:     &cr.Spec.ForProvider.PostgresVersion,
			Instances:           &cr.Spec.ForProvider.Instances,
			Cores:               &cr.Spec.ForProvider.Cores,
			Ram:                 &cr.Spec.ForProvider.RAM,
			StorageSize:         &cr.Spec.ForProvider.StorageSize,
			StorageType:         (*ionoscloud.StorageType)(&cr.Spec.ForProvider.StorageType),
			Connections:         clusterConnections(cr.Spec.ForProvider.Connections),
			Location:            &cr.Spec.ForProvider.Location,
			DisplayName:         &cr.Spec.ForProvider.DisplayName,
			Credentials:         clusterCredentials(cr.Spec.ForProvider.Credentials),
			SynchronizationMode: (*ionoscloud.SynchronizationMode)(&cr.Spec.ForProvider.SynchronizationMode),
		},
	}
	fromBackup, err := clusterFromBackup(cr.Spec.ForProvider.FromBackup)
	if err != nil {
		return nil, err
	}
	if fromBackup != nil {
		instanceCreateInput.Properties.SetFromBackup(*fromBackup)
	}
	if window := clusterMaintenanceWindow(cr.Spec.ForProvider.MaintenanceWindow); window != nil {
		instanceCreateInput.Properties.SetMaintenanceWindow(*window)
	}
	if !utils.IsEmptyValue(reflect.ValueOf(cr.Spec.ForProvider.BackupLocation)) {
		instanceCreateInput.Properties.SetBackupLocation(cr.Spec.ForProvider.BackupLocation)
	}
	return &instanceCreateInput, err
}

// GenerateUpdateClusterInput returns PatchClusterRequest based on the CR spec modifications
func GenerateUpdateClusterInput(cr *v1alpha1.PostgresCluster) (*ionoscloud.PatchClusterRequest, error) {
	instanceUpdateInput := ionoscloud.PatchClusterRequest{
		Properties: &ionoscloud.PatchClusterProperties{
			PostgresVersion: &cr.Spec.ForProvider.PostgresVersion,
			Instances:       &cr.Spec.ForProvider.Instances,
			Cores:           &cr.Spec.ForProvider.Cores,
			Ram:             &cr.Spec.ForProvider.RAM,
			StorageSize:     &cr.Spec.ForProvider.StorageSize,
			Connections:     clusterConnections(cr.Spec.ForProvider.Connections),
			DisplayName:     &cr.Spec.ForProvider.DisplayName,
		},
	}
	if window := clusterMaintenanceWindow(cr.Spec.ForProvider.MaintenanceWindow); window != nil {
		instanceUpdateInput.Properties.SetMaintenanceWindow(*window)
	}
	return &instanceUpdateInput, nil
}

// LateInitializer fills the empty fields in *v1alpha1.ClusterParameters with
// the values seen in ionoscloud.ClusterResponse.
func LateInitializer(in *v1alpha1.ClusterParameters, sg *ionoscloud.ClusterResponse) { // nolint:gocyclo
	if sg == nil {
		return
	}
	// Add Maintenance Window to the Spec, if it was set by the API
	if propertiesOk, ok := sg.GetPropertiesOk(); ok && propertiesOk != nil {
		if maintenanceWindowOk, ok := propertiesOk.GetMaintenanceWindowOk(); ok && maintenanceWindowOk != nil {
			if timeOk, ok := maintenanceWindowOk.GetTimeOk(); ok && timeOk != nil {
				if utils.IsEmptyValue(reflect.ValueOf(in.MaintenanceWindow.Time)) {
					in.MaintenanceWindow.Time = *timeOk
				}
			}
			if dayOfTheWeekOk, ok := maintenanceWindowOk.GetDayOfTheWeekOk(); ok && dayOfTheWeekOk != nil {
				if utils.IsEmptyValue(reflect.ValueOf(in.MaintenanceWindow.DayOfTheWeek)) {
					in.MaintenanceWindow.DayOfTheWeek = string(*dayOfTheWeekOk)
				}
			}
		}
	}
}

// IsClusterUpToDate returns true if the cluster is up-to-date or false if it does not
func IsClusterUpToDate(cr *v1alpha1.PostgresCluster, clusterResponse ionoscloud.ClusterResponse) bool { // nolint:gocyclo
	switch {
	case cr == nil && clusterResponse.Properties == nil:
		return true
	case cr == nil && clusterResponse.Properties != nil:
		return false
	case cr != nil && clusterResponse.Properties == nil:
		return false
	case clusterResponse.Metadata.State != nil && *clusterResponse.Metadata.State == ionoscloud.BUSY:
		return true
	case clusterResponse.Properties.DisplayName != nil && *clusterResponse.Properties.DisplayName != cr.Spec.ForProvider.DisplayName:
		return false
	case clusterResponse.Properties.DisplayName == nil && cr.Spec.ForProvider.DisplayName != "":
		return false
	case clusterResponse.Properties.PostgresVersion != nil && *clusterResponse.Properties.PostgresVersion != cr.Spec.ForProvider.PostgresVersion:
		return false
	case clusterResponse.Properties.Instances != nil && *clusterResponse.Properties.Instances != cr.Spec.ForProvider.Instances:
		return false
	case clusterResponse.Properties.Cores != nil && *clusterResponse.Properties.Cores != cr.Spec.ForProvider.Cores:
		return false
	case clusterResponse.Properties.Ram != nil && *clusterResponse.Properties.Ram != cr.Spec.ForProvider.RAM:
		return false
	case clusterResponse.Properties.StorageSize != nil && *clusterResponse.Properties.StorageSize != cr.Spec.ForProvider.StorageSize:
		return false
	case clusterResponse.Properties.Connections != nil && !reflect.DeepEqual(*clusterResponse.Properties.Connections, cr.Spec.ForProvider.Connections):
		return false
	case !compare.EqualDatabaseMaintenanceWindow(cr.Spec.ForProvider.MaintenanceWindow, clusterResponse.Properties.MaintenanceWindow):
		return false
	default:
		return true
	}
}

func clusterConnections(connections []v1alpha1.Connection) *[]ionoscloud.Connection {
	connects := make([]ionoscloud.Connection, 0)
	for _, connection := range connections {
		datacenterID := connection.DatacenterCfg.DatacenterID
		lanID := connection.LanCfg.LanID
		cidr := connection.Cidr
		connects = append(connects, ionoscloud.Connection{
			DatacenterId: &datacenterID,
			LanId:        &lanID,
			Cidr:         &cidr,
		})
	}
	return &connects
}

func clusterMaintenanceWindow(window v1alpha1.MaintenanceWindow) *ionoscloud.MaintenanceWindow {
	if window.Time != "" && window.DayOfTheWeek != "" {
		return &ionoscloud.MaintenanceWindow{
			Time:         &window.Time,
			DayOfTheWeek: (*ionoscloud.DayOfTheWeek)(&window.DayOfTheWeek),
		}
	}
	return nil
}

func clusterCredentials(creds v1alpha1.DBUser) *ionoscloud.DBUser {
	return &ionoscloud.DBUser{
		Username: &creds.Username,
		Password: &creds.Password,
	}
}

func clusterFromBackup(req v1alpha1.CreateRestoreRequest) (*ionoscloud.CreateRestoreRequest, error) {
	if req.BackupID != "" && req.RecoveryTargetTime != "" {
		recoveryTime, err := time.Parse(time.RFC3339, req.RecoveryTargetTime)
		if err != nil {
			return nil, err
		}
		return &ionoscloud.CreateRestoreRequest{
			BackupId:           &req.BackupID,
			RecoveryTargetTime: &ionoscloud.IonosTime{Time: recoveryTime},
		}, nil
	}
	return nil, nil
}
