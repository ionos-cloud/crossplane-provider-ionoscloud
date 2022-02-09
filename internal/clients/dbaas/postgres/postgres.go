package postgres

import (
	"context"
	"strings"
	"time"

	ionoscloud "github.com/ionos-cloud/sdk-go-dbaas-postgres"

	"github.com/ionos-cloud/crossplane-provider-ionoscloud/apis/dbaas/postgres/v1alpha1"
	"github.com/ionos-cloud/crossplane-provider-ionoscloud/internal/clients"
)

// ClusterAPIClient is a wrapper around IONOS Service
type ClusterAPIClient struct {
	*clients.IonosServices
}

// ClusterClient is a wrapper around IONOS Service methods
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
func GenerateCreateClusterInput(cr *v1alpha1.Cluster) (*ionoscloud.CreateClusterRequest, error) {
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
func GenerateUpdateClusterInput(cr *v1alpha1.Cluster) (*ionoscloud.PatchClusterRequest, error) {
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

// IsClusterUpToDate returns true if the cluster is up-to-date or false if it does not
func IsClusterUpToDate(cr *v1alpha1.Cluster, clusterResponse ionoscloud.ClusterResponse) bool {
	switch {
	case cr == nil && clusterResponse.Properties == nil:
		return true
	case cr == nil && clusterResponse.Properties != nil:
		return false
	case cr != nil && clusterResponse.Properties == nil:
		return false
	}
	if *clusterResponse.Metadata.State == ionoscloud.BUSY {
		return true
	}
	if strings.Compare(cr.Spec.ForProvider.DisplayName, *clusterResponse.Properties.DisplayName) != 0 {
		return false
	}
	return true
}

func clusterConnections(connections []v1alpha1.Connection) *[]ionoscloud.Connection {
	connects := make([]ionoscloud.Connection, 0)
	for _, connection := range connections {
		datacenterID := connection.DatacenterID
		lanID := connection.LanID
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
