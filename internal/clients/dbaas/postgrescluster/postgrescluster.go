package postgrescluster

import (
	"context"
	"fmt"
	"reflect"
	"time"

	"github.com/ionos-cloud/sdk-go-bundle/shared"

	"github.com/ionos-cloud/crossplane-provider-ionoscloud/internal/compare"
	"github.com/ionos-cloud/crossplane-provider-ionoscloud/internal/utils"

	ionoscloud "github.com/ionos-cloud/sdk-go-bundle/products/dbaas/psql/v2"

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
	CheckDuplicateUser(ctx context.Context, clusterID, userName string) (*ionoscloud.UserResource, error)
	GetClusterID(cluster *ionoscloud.ClusterResponse) (string, error)
	GetUserID(user *ionoscloud.UserResource) (string, error)
	GetCluster(ctx context.Context, clusterID string) (ionoscloud.ClusterResponse, *ionoscloud.APIResponse, error)
	GetUser(ctx context.Context, clusterID, userName string) (ionoscloud.UserResource, *ionoscloud.APIResponse, error)
	DeleteCluster(ctx context.Context, clusterID string) (*shared.APIResponse, error)
	DeleteUser(ctx context.Context, clusterID, userName string) (*shared.APIResponse, error)
	CreateCluster(ctx context.Context, cluster ionoscloud.CreateClusterRequest) (ionoscloud.ClusterResponse, *ionoscloud.APIResponse, error)
	CreateUser(ctx context.Context, clusterID string, user ionoscloud.User) (ionoscloud.UserResource, *ionoscloud.APIResponse, error)
	UpdateCluster(ctx context.Context, clusterID string, cluster ionoscloud.PatchClusterRequest) (ionoscloud.ClusterResponse, *ionoscloud.APIResponse, error)
	UpdateUser(ctx context.Context, clusterID, userName string, cluster ionoscloud.UsersPatchRequest) (ionoscloud.UserResource, *ionoscloud.APIResponse, error)
	// Database stuff
	CreateDatabase(ctx context.Context, clusterID string, database v1alpha1.PostgresDatabase) (ionoscloud.DatabaseResource, *ionoscloud.APIResponse, error)
	GetDatabase(ctx context.Context, clusterID, databaseName string) (ionoscloud.DatabaseResource, *ionoscloud.APIResponse, error)
	DeleteDatabase(ctx context.Context, clusterID, databaseID string) (*ionoscloud.APIResponse, error)
}

// CheckDuplicateCluster based on clusterName and on multiple properties from CR spec
func (cp *ClusterAPIClient) CheckDuplicateCluster(ctx context.Context, clusterName string, cr *v1alpha1.PostgresCluster) (*ionoscloud.ClusterResponse, error) { // nolint: gocyclo
	clusterList, _, err := cp.DBaaSPostgresClient.ClustersApi.ClustersGet(ctx).Execute()
	if err != nil {
		return nil, err
	}
	matchedItems := make([]ionoscloud.ClusterResponse, 0)
	if itemsOk, ok := clusterList.GetItemsOk(); ok && itemsOk != nil {
		for _, item := range itemsOk {
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

// CheckDuplicateUser based on clusterName and on multiple properties from CR spec
func (cp *ClusterAPIClient) CheckDuplicateUser(ctx context.Context, clusterID, userName string) (*ionoscloud.UserResource, error) { // nolint: gocyclo
	_, resp, err := cp.DBaaSPostgresClient.UsersApi.UsersGet(ctx, clusterID, userName).Execute()
	if err != nil && !resp.HttpNotFound() {
		return nil, err
	}
	return nil, nil
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

// GetUserID based on cluster
func (cp *ClusterAPIClient) GetUserID(user *ionoscloud.UserResource) (string, error) {
	if user == nil {
		return "", fmt.Errorf("error: getting Username")
	}
	return user.Properties.Username, nil
}

// GetCluster based on clusterID
func (cp *ClusterAPIClient) GetCluster(ctx context.Context, clusterID string) (ionoscloud.ClusterResponse, *shared.APIResponse, error) {
	return cp.DBaaSPostgresClient.ClustersApi.ClustersFindById(ctx, clusterID).Execute()
}

// GetUser based on clusterID and username
func (cp *ClusterAPIClient) GetUser(ctx context.Context, clusterID, userName string) (ionoscloud.UserResource, *shared.APIResponse, error) {
	return cp.DBaaSPostgresClient.UsersApi.UsersGet(ctx, clusterID, userName).Execute()
}

// DeleteCluster based on clusterID
func (cp *ClusterAPIClient) DeleteCluster(ctx context.Context, clusterID string) (*shared.APIResponse, error) {
	_, apiResponse, err := cp.DBaaSPostgresClient.ClustersApi.ClustersDelete(ctx, clusterID).Execute()
	return apiResponse, err
}

// DeleteUser based on clusterID
func (cp *ClusterAPIClient) DeleteUser(ctx context.Context, clusterID, userName string) (*shared.APIResponse, error) {
	return cp.DBaaSPostgresClient.UsersApi.UsersDelete(ctx, clusterID, userName).Execute()
}

// CreateCluster based on cluster properties
func (cp *ClusterAPIClient) CreateCluster(ctx context.Context, cluster ionoscloud.CreateClusterRequest) (ionoscloud.ClusterResponse, *shared.APIResponse, error) {
	return cp.DBaaSPostgresClient.ClustersApi.ClustersPost(ctx).CreateClusterRequest(cluster).Execute()
}

// CreateUser based on clusterID and user properties
func (cp *ClusterAPIClient) CreateUser(ctx context.Context, clusterID string, user ionoscloud.User) (ionoscloud.UserResource, *shared.APIResponse, error) {
	return cp.DBaaSPostgresClient.UsersApi.UsersPost(ctx, clusterID).User(user).Execute()
}

// PatchUser based on clusterID, username and user properties
func (cp *ClusterAPIClient) PatchUser(ctx context.Context, clusterID, username string, patchReq ionoscloud.UsersPatchRequest) (ionoscloud.UserResource, *shared.APIResponse, error) {
	return cp.DBaaSPostgresClient.UsersApi.UsersPatch(ctx, clusterID, username).UsersPatchRequest(patchReq).Execute()
}

// UpdateCluster based on clusterID and cluster properties
func (cp *ClusterAPIClient) UpdateCluster(ctx context.Context, clusterID string, cluster ionoscloud.PatchClusterRequest) (ionoscloud.ClusterResponse, *shared.APIResponse, error) {
	return cp.DBaaSPostgresClient.ClustersApi.ClustersPatch(ctx, clusterID).PatchClusterRequest(cluster).Execute()
}

// UpdateUser based on clusterID and cluster properties
func (cp *ClusterAPIClient) UpdateUser(ctx context.Context, clusterID, userName string, patchReq ionoscloud.UsersPatchRequest) (ionoscloud.UserResource, *shared.APIResponse, error) {
	return cp.DBaaSPostgresClient.UsersApi.UsersPatch(ctx, clusterID, userName).UsersPatchRequest(patchReq).Execute()
}

// GenerateCreateClusterInput returns CreateClusterRequest based on the CR spec
func GenerateCreateClusterInput(cr *v1alpha1.PostgresCluster) (*ionoscloud.CreateClusterRequest, error) {
	instanceCreateInput := ionoscloud.CreateClusterRequest{
		Properties: &ionoscloud.CreateClusterProperties{
			PostgresVersion:     cr.Spec.ForProvider.PostgresVersion,
			Instances:           cr.Spec.ForProvider.Instances,
			Cores:               cr.Spec.ForProvider.Cores,
			Ram:                 cr.Spec.ForProvider.RAM,
			StorageSize:         cr.Spec.ForProvider.StorageSize,
			StorageType:         (ionoscloud.StorageType)(cr.Spec.ForProvider.StorageType),
			Connections:         clusterConnections(cr.Spec.ForProvider.Connections),
			Location:            cr.Spec.ForProvider.Location,
			DisplayName:         cr.Spec.ForProvider.DisplayName,
			Credentials:         clusterCredentials(cr.Spec.ForProvider.Credentials),
			SynchronizationMode: (ionoscloud.SynchronizationMode)(cr.Spec.ForProvider.SynchronizationMode),
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
	if pooler := clusterConnectionPooler(cr.Spec.ForProvider.ConnectionPooler); pooler != nil {
		instanceCreateInput.Properties.SetConnectionPooler(*pooler)
	}
	return &instanceCreateInput, err
}

// GenerateCreateUserInput returns psql User based on the CR spec
func GenerateCreateUserInput(cr *v1alpha1.PostgresUser) *ionoscloud.User {
	instanceCreateInput := ionoscloud.User{
		Properties: ionoscloud.UserProperties{
			Username: cr.Spec.ForProvider.Credentials.Username,
			// We don't want to store the secret provided password in the spec
			Password: shared.ToPtr(cr.Spec.ForProvider.Credentials.Password),
		},
	}
	return &instanceCreateInput
}

func generateDatabaseInput(cr v1alpha1.PostgresDatabase) ionoscloud.Database {
	instanceCreateInput := ionoscloud.Database{
		Properties: &ionoscloud.DatabaseProperties{
			Name:  &cr.Spec.ForProvider.Name,
			Owner: &cr.Spec.ForProvider.Owner.UserName,
		},
	}
	return instanceCreateInput
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
	if pooler := clusterConnectionPooler(cr.Spec.ForProvider.ConnectionPooler); pooler != nil {
		instanceUpdateInput.Properties.SetConnectionPooler(*pooler)
	}
	return &instanceUpdateInput, nil
}

// GenerateUpdateUserInput returns PatchClusterRequest based on the CR spec modifications
func GenerateUpdateUserInput(cr *v1alpha1.PostgresUser) *ionoscloud.UsersPatchRequest {
	instanceUpdateInput := ionoscloud.UsersPatchRequest{
		Properties: ionoscloud.PatchUserProperties{
			// We don't want to store the secret provided password in the spec
			Password: shared.ToPtr(cr.Spec.ForProvider.Credentials.Password),
		},
	}

	return &instanceUpdateInput
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
	case clusterResponse.Metadata.State != nil && *clusterResponse.Metadata.State == ionoscloud.STATE_BUSY:
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
	case clusterResponse.Properties.Connections != nil && !compare.EqualConnections(cr.Spec.ForProvider.Connections, clusterResponse.Properties.Connections):
		return false
	case !compare.EqualDatabaseMaintenanceWindow(cr.Spec.ForProvider.MaintenanceWindow, clusterResponse.Properties.MaintenanceWindow):
		return false
	case !compare.EqualConnectionPooler(cr.Spec.ForProvider.ConnectionPooler, clusterResponse.Properties.ConnectionPooler):
		return false
	default:
		return true
	}
}

// IsUserUpToDate returns true if the user is up-to-date or false if it does not
func IsUserUpToDate(cr *v1alpha1.PostgresUser, user ionoscloud.UserResource) bool { // nolint:gocyclo
	switch {
	case user.Properties.Username != cr.Spec.ForProvider.Credentials.Username:
		return false
	default:
		return true
	}
}

func clusterConnections(connections []v1alpha1.Connection) []ionoscloud.Connection {
	connects := make([]ionoscloud.Connection, 0)
	for _, connection := range connections {
		datacenterID := connection.DatacenterCfg.DatacenterID
		lanID := connection.LanCfg.LanID
		cidr := connection.Cidr
		connects = append(connects, ionoscloud.Connection{
			DatacenterId: datacenterID,
			LanId:        lanID,
			Cidr:         cidr,
		})
	}
	return connects
}

func clusterMaintenanceWindow(window v1alpha1.MaintenanceWindow) *ionoscloud.MaintenanceWindow {
	if window.Time != "" && window.DayOfTheWeek != "" {
		return &ionoscloud.MaintenanceWindow{
			Time:         window.Time,
			DayOfTheWeek: (ionoscloud.DayOfTheWeek)(window.DayOfTheWeek),
		}
	}
	return nil
}

func clusterConnectionPooler(pooler v1alpha1.ConnectionPooler) *ionoscloud.ConnectionPooler {
	if pooler.PoolMode != "" {
		return &ionoscloud.ConnectionPooler{
			Enabled:  &pooler.Enabled,
			PoolMode: &pooler.PoolMode,
		}
	}
	return nil
}

func clusterCredentials(creds v1alpha1.DBUser) ionoscloud.DBUser {
	return ionoscloud.DBUser{
		Username: creds.Username,
		Password: creds.Password,
	}
}

func clusterFromBackup(req v1alpha1.CreateRestoreRequest) (*ionoscloud.CreateRestoreRequest, error) {
	if req.BackupID != "" && req.RecoveryTargetTime != "" {
		recoveryTime, err := time.Parse(time.RFC3339, req.RecoveryTargetTime)
		if err != nil {
			return nil, err
		}
		return &ionoscloud.CreateRestoreRequest{
			BackupId:           req.BackupID,
			RecoveryTargetTime: &ionoscloud.IonosTime{Time: recoveryTime},
		}, nil
	}
	return nil, nil
}
