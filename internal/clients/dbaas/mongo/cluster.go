package mongo

import (
	"context"
	"fmt"
	"reflect"
	"time"

	"github.com/ionos-cloud/crossplane-provider-ionoscloud/internal/compare"
	"github.com/ionos-cloud/crossplane-provider-ionoscloud/internal/utils"

	ionoscloud "github.com/ionos-cloud/sdk-go-dbaas-mongo"

	"github.com/ionos-cloud/crossplane-provider-ionoscloud/apis/dbaas/mongo/v1alpha1"
	"github.com/ionos-cloud/crossplane-provider-ionoscloud/internal/clients"
)

// ClusterAPIClient is a wrapper around IONOS Service DBaaS Mongo Cluster
type ClusterAPIClient struct {
	*clients.IonosServices
}

// ClusterClient is a wrapper around IONOS Service DBaaS Mongo Cluster methods
type ClusterClient interface {
	CheckDuplicateCluster(ctx context.Context, clusterName string, cr *v1alpha1.MongoCluster) (*ionoscloud.ClusterResponse, error)
	GetClusterID(cluster *ionoscloud.ClusterResponse) (string, error)
	GetCluster(ctx context.Context, clusterID string) (ionoscloud.ClusterResponse, *ionoscloud.APIResponse, error)
	DeleteCluster(ctx context.Context, clusterID string) (*ionoscloud.APIResponse, error)
	DeleteUser(ctx context.Context, clusterID, userName string) (*ionoscloud.APIResponse, error)
	CreateCluster(ctx context.Context, cluster ionoscloud.CreateClusterRequest) (ionoscloud.ClusterResponse, *ionoscloud.APIResponse, error)
	UpdateCluster(ctx context.Context, clusterID string, cluster ionoscloud.PatchClusterRequest) (ionoscloud.ClusterResponse, *ionoscloud.APIResponse, error)
}

// CheckDuplicateCluster based on clusterName and on multiple properties from CR spec
func (cp *ClusterAPIClient) CheckDuplicateCluster(ctx context.Context, clusterName string, cr *v1alpha1.MongoCluster) (*ionoscloud.ClusterResponse, error) { // nolint: gocyclo
	clusterList, _, err := cp.DBaaSMongoClient.ClustersApi.ClustersGet(ctx).Execute()
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
func (cp *ClusterAPIClient) CheckDuplicateUser(ctx context.Context, clusterID, userName string) (*ionoscloud.User, error) { // nolint: gocyclo
	_, resp, err := cp.DBaaSMongoClient.UsersApi.ClustersUsersFindById(ctx, clusterID, userName).Execute()
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

// GetCluster based on clusterID
func (cp *ClusterAPIClient) GetCluster(ctx context.Context, clusterID string) (ionoscloud.ClusterResponse, *ionoscloud.APIResponse, error) {
	return cp.DBaaSMongoClient.ClustersApi.ClustersFindById(ctx, clusterID).Execute()
}

// DeleteCluster based on clusterID
func (cp *ClusterAPIClient) DeleteCluster(ctx context.Context, clusterID string) (*ionoscloud.APIResponse, error) {
	_, apiResponse, err := cp.DBaaSMongoClient.ClustersApi.ClustersDelete(ctx, clusterID).Execute()
	return apiResponse, err
}

// DeleteUser based on clusterID
func (cp *ClusterAPIClient) DeleteUser(ctx context.Context, clusterID, userName string) (*ionoscloud.APIResponse, error) {
	_, response, err := cp.DBaaSMongoClient.UsersApi.ClustersUsersDelete(ctx, clusterID, userName).Execute()
	return response, err
}

// CreateCluster based on cluster properties
func (cp *ClusterAPIClient) CreateCluster(ctx context.Context, cluster ionoscloud.CreateClusterRequest) (ionoscloud.ClusterResponse, *ionoscloud.APIResponse, error) {
	return cp.DBaaSMongoClient.ClustersApi.ClustersPost(ctx).CreateClusterRequest(cluster).Execute()
}

// CreateUser based on clusterID and user properties
func (cp *ClusterAPIClient) CreateUser(ctx context.Context, clusterID string, user ionoscloud.User) (ionoscloud.User, *ionoscloud.APIResponse, error) {
	return cp.DBaaSMongoClient.UsersApi.ClustersUsersPost(ctx, clusterID).User(user).Execute()
}

// PatchUser based on clusterID, username and user properties
func (cp *ClusterAPIClient) PatchUser(ctx context.Context, clusterID, username string, patchReq ionoscloud.PatchUserRequest) (ionoscloud.User, *ionoscloud.APIResponse, error) {
	return cp.DBaaSMongoClient.UsersApi.ClustersUsersPatch(ctx, clusterID, username).PatchUserRequest(patchReq).Execute()
}

// UpdateCluster based on clusterID and cluster properties
func (cp *ClusterAPIClient) UpdateCluster(ctx context.Context, clusterID string, cluster ionoscloud.PatchClusterRequest) (ionoscloud.ClusterResponse, *ionoscloud.APIResponse, error) {
	return cp.DBaaSMongoClient.ClustersApi.ClustersPatch(ctx, clusterID).PatchClusterRequest(cluster).Execute()
}

// UpdateUser based on clusterID and cluster properties
func (cp *ClusterAPIClient) UpdateUser(ctx context.Context, clusterID, userName string, patchReq ionoscloud.PatchUserRequest) (ionoscloud.User, *ionoscloud.APIResponse, error) {
	return cp.DBaaSMongoClient.UsersApi.ClustersUsersPatch(ctx, clusterID, userName).PatchUserRequest(patchReq).Execute()
}

// GenerateCreateClusterInput returns CreateClusterRequest based on the CR spec
func GenerateCreateClusterInput(cr *v1alpha1.MongoCluster) (*ionoscloud.CreateClusterRequest, error) { // nolint: gocyclo
	instanceCreateInput := ionoscloud.CreateClusterRequest{
		Properties: &ionoscloud.CreateClusterProperties{
			MongoDBVersion:    &cr.Spec.ForProvider.MongoDBVersion,
			Instances:         &cr.Spec.ForProvider.Instances,
			Connections:       clusterConnections(cr.Spec.ForProvider.Connections),
			Location:          &cr.Spec.ForProvider.Location,
			Backup:            clusterBackup(cr.Spec.ForProvider.Backup),
			DisplayName:       &cr.Spec.ForProvider.DisplayName,
			MaintenanceWindow: clusterMaintenanceWindow(cr.Spec.ForProvider.MaintenanceWindow),
			BiConnector:       clusterBiConnector(cr.Spec.ForProvider.BiConnector),
		},
	}
	if cr.Spec.ForProvider.Type != "" {
		instanceCreateInput.Properties.Type = &cr.Spec.ForProvider.Type
	}
	if cr.Spec.ForProvider.TemplateID != "" {
		instanceCreateInput.Properties.TemplateID = &cr.Spec.ForProvider.TemplateID
	}
	if cr.Spec.ForProvider.StorageType != "" {
		instanceCreateInput.Properties.StorageType = (*ionoscloud.StorageType)(&cr.Spec.ForProvider.StorageType)
	}
	if cr.Spec.ForProvider.Edition != "" {
		instanceCreateInput.Properties.Edition = &cr.Spec.ForProvider.Edition
	}
	if cr.Spec.ForProvider.RAM != 0 {
		instanceCreateInput.Properties.Ram = &cr.Spec.ForProvider.RAM
	}
	if cr.Spec.ForProvider.StorageSize != 0 {
		instanceCreateInput.Properties.StorageSize = &cr.Spec.ForProvider.StorageSize
	}
	if cr.Spec.ForProvider.Cores != 0 {
		instanceCreateInput.Properties.Cores = &cr.Spec.ForProvider.Cores
	}
	if cr.Spec.ForProvider.Shards != 0 {
		instanceCreateInput.Properties.Shards = &cr.Spec.ForProvider.Shards
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

// GenerateCreateUserInput returns mongo User based on the CR spec
// func GenerateCreateUserInput(cr *v1alpha1.MongoUser) *ionoscloud.User {
// 	instanceCreateInput := ionoscloud.User{
// 		Properties: &ionoscloud.UserProperties{
// 			Username: &cr.Spec.ForProvider.Credentials.Username,
// 			Password: &cr.Spec.ForProvider.Credentials.Password,
// 		},
// 	}
// 	return &instanceCreateInput
// }

// GenerateUpdateClusterInput returns PatchClusterRequest based on the CR spec modifications
func GenerateUpdateClusterInput(cr *v1alpha1.MongoCluster) (*ionoscloud.PatchClusterRequest, error) { // nolint: gocyclo
	instanceUpdateInput := ionoscloud.PatchClusterRequest{
		Properties: &ionoscloud.PatchClusterProperties{
			Instances:   &cr.Spec.ForProvider.Instances,
			Connections: clusterConnections(cr.Spec.ForProvider.Connections),
			DisplayName: &cr.Spec.ForProvider.DisplayName,
		},
	}
	if cr.Spec.ForProvider.Type != "" {
		instanceUpdateInput.Properties.Type = &cr.Spec.ForProvider.Type
	}
	if cr.Spec.ForProvider.TemplateID != "" {
		instanceUpdateInput.Properties.TemplateID = &cr.Spec.ForProvider.TemplateID
	}
	if cr.Spec.ForProvider.StorageType != "" {
		instanceUpdateInput.Properties.StorageType = (*ionoscloud.StorageType)(&cr.Spec.ForProvider.StorageType)
	}
	if cr.Spec.ForProvider.Edition != "" {
		instanceUpdateInput.Properties.Edition = &cr.Spec.ForProvider.Edition
	}
	if cr.Spec.ForProvider.RAM != 0 {
		instanceUpdateInput.Properties.Ram = &cr.Spec.ForProvider.RAM
	}
	if cr.Spec.ForProvider.StorageSize != 0 {
		instanceUpdateInput.Properties.StorageSize = &cr.Spec.ForProvider.StorageSize
	}
	if cr.Spec.ForProvider.Cores != 0 {
		instanceUpdateInput.Properties.Cores = &cr.Spec.ForProvider.Cores
	}
	if cr.Spec.ForProvider.Shards != 0 {
		instanceUpdateInput.Properties.Shards = &cr.Spec.ForProvider.Shards
	}
	if window := clusterMaintenanceWindow(cr.Spec.ForProvider.MaintenanceWindow); window != nil {
		instanceUpdateInput.Properties.SetMaintenanceWindow(*window)
	}
	if biConnector := clusterBiConnector(cr.Spec.ForProvider.BiConnector); biConnector != nil {
		instanceUpdateInput.Properties.SetBiConnector(*biConnector)
	}
	return &instanceUpdateInput, nil
}

//// GenerateUpdateUserInput returns PatchClusterRequest based on the CR spec modifications
//func GenerateUpdateUserInput(cr *v1alpha1.MongoUser) (*ionoscloud.UsersPatchRequest, error) {
//	instanceUpdateInput := ionoscloud.UsersPatchRequest{
//		Properties: &ionoscloud.PatchUserProperties{
//			Password: &cr.Spec.ForProvider.Credentials.Password,
//		},
//	}
//
//	return &instanceUpdateInput, nil
//}

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
func IsClusterUpToDate(cr *v1alpha1.MongoCluster, clusterResponse ionoscloud.ClusterResponse) bool { // nolint:gocyclo
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
	case clusterResponse.Properties.MongoDBVersion != nil && *clusterResponse.Properties.MongoDBVersion != cr.Spec.ForProvider.MongoDBVersion:
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
	case clusterResponse.Properties.BiConnector != nil && !reflect.DeepEqual(*clusterResponse.Properties.BiConnector, cr.Spec.ForProvider.BiConnector):
		return false
	case clusterResponse.Properties.Edition != nil && *clusterResponse.Properties.Edition != cr.Spec.ForProvider.Edition:
		return false
	case !compare.EqualMongoDatabaseMaintenanceWindow(cr.Spec.ForProvider.MaintenanceWindow, clusterResponse.Properties.MaintenanceWindow):
		return false
	default:
		return true
	}
}

// IsUserUpToDate returns true if the user is up-to-date or false if it does not
//func IsUserUpToDate(cr *v1alpha1.MongoUser, user ionoscloud.UserResource) bool { // nolint:gocyclo
//	switch {
//	case cr == nil && user.Properties == nil:
//		return true
//	case cr == nil && user.Properties != nil:
//		return false
//	case cr != nil && user.Properties == nil:
//		return false
//	case user.Properties.Username != nil && *user.Properties.Username != cr.Spec.ForProvider.Credentials.Username:
//		return false
//	case user.Properties.Username != nil && *user.Properties.Password != cr.Spec.ForProvider.Credentials.Password:
//		return false
//	default:
//		return true
//	}
//}

func clusterConnections(connections []v1alpha1.Connection) *[]ionoscloud.Connection {
	connects := make([]ionoscloud.Connection, 0)
	for _, connection := range connections {
		datacenterID := connection.DatacenterCfg.DatacenterID
		lanID := connection.LanCfg.LanID
		cidr := connection.CidrList
		connects = append(connects, ionoscloud.Connection{
			DatacenterId: &datacenterID,
			LanId:        &lanID,
			CidrList:     &cidr,
		})
	}
	return &connects
}

func clusterBiConnector(biConnector v1alpha1.BiConnectorProperties) *ionoscloud.BiConnectorProperties {
	if biConnector.Port != "" && biConnector.Host != "" {
		return &ionoscloud.BiConnectorProperties{
			Port:    &biConnector.Port,
			Host:    &biConnector.Host,
			Enabled: &biConnector.Enabled,
		}
	}
	return nil
}

func clusterBackup(backup v1alpha1.BackupProperties) *ionoscloud.BackupProperties {
	if backup.Location != "" {
		return &ionoscloud.BackupProperties{
			Location: &backup.Location,
		}
	}
	return nil
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

func clusterCredentials(creds v1alpha1.DBUser) *ionoscloud.User {
	return &ionoscloud.User{
		Properties: &ionoscloud.UserProperties{
			Username: &creds.Username,
			Password: &creds.Password,
			// TODO add roles
			Roles: nil,
		},
	}
}

func clusterFromBackup(req v1alpha1.CreateRestoreRequest) (*ionoscloud.CreateRestoreRequest, error) {
	if req.SnapshotID != "" && req.RecoveryTargetTime != "" {
		recoveryTime, err := time.Parse(time.RFC3339, req.RecoveryTargetTime)
		if err != nil {
			return nil, err
		}
		return &ionoscloud.CreateRestoreRequest{
			SnapshotId:         &req.SnapshotID,
			RecoveryTargetTime: &ionoscloud.IonosTime{Time: recoveryTime},
		}, nil
	}
	return nil, nil
}
