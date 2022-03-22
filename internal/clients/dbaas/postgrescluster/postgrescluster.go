package postgrescluster

import (
	"context"
	"reflect"
	"time"

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
	GetCluster(ctx context.Context, clusterID string) (ionoscloud.ClusterResponse, *ionoscloud.APIResponse, error)
	DeleteCluster(ctx context.Context, clusterID string) error
	CreateCluster(ctx context.Context, cluster ionoscloud.CreateClusterRequest) (ionoscloud.ClusterResponse, *ionoscloud.APIResponse, error)
	UpdateCluster(ctx context.Context, clusterID string, cluster ionoscloud.PatchClusterRequest) (ionoscloud.ClusterResponse, *ionoscloud.APIResponse, error)
}

// GetCluster based on clusterID
func (cp *ClusterAPIClient) GetCluster(ctx context.Context, clusterID string) (ionoscloud.ClusterResponse, *ionoscloud.APIResponse, error) {
	return cp.DBaaSPostgresClient.ClustersApi.ClustersFindById(ctx, clusterID).Execute()
}

// DeleteCluster based on clusterID
func (cp *ClusterAPIClient) DeleteCluster(ctx context.Context, clusterID string) error {
	_, _, err := cp.DBaaSPostgresClient.ClustersApi.ClustersDelete(ctx, clusterID).Execute()
	return err
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
			Location:            (*ionoscloud.Location)(&cr.Spec.ForProvider.Location),
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
