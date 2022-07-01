package backupunit

import (
	"context"
	"fmt"

	sdkgo "github.com/ionos-cloud/sdk-go/v6"

	"github.com/ionos-cloud/crossplane-provider-ionoscloud/apis/backup/v1alpha1"
	"github.com/ionos-cloud/crossplane-provider-ionoscloud/internal/clients"
	"github.com/ionos-cloud/crossplane-provider-ionoscloud/internal/utils"
)

// APIClient is a wrapper around IONOS Service
type APIClient struct {
	*clients.IonosServices
}

// Client is a wrapper around IONOS Service BackupUnit methods
type Client interface {
	GetBackupUnit(ctx context.Context, backupUnitID string) (sdkgo.BackupUnit, *sdkgo.APIResponse, error)
	GetBackupUnits(ctx context.Context) (sdkgo.BackupUnits, *sdkgo.APIResponse, error)
	GetBackupUnitIDByName(ctx context.Context, backupUnitName string) (string, error)
	CreateBackupUnit(ctx context.Context, backupUnit sdkgo.BackupUnit) (sdkgo.BackupUnit, *sdkgo.APIResponse, error)
	UpdateBackupUnit(ctx context.Context, backupUnitID string, backupUnit sdkgo.BackupUnitProperties) (sdkgo.BackupUnit, *sdkgo.APIResponse, error)
	DeleteBackupUnit(ctx context.Context, backupUnitID string) (*sdkgo.APIResponse, error)
	GetAPIClient() *sdkgo.APIClient
}

// GetBackupUnit based on backupUnitID
func (cp *APIClient) GetBackupUnit(ctx context.Context, backupUnitID string) (sdkgo.BackupUnit, *sdkgo.APIResponse, error) {
	return cp.ComputeClient.BackupUnitsApi.BackupunitsFindById(ctx, backupUnitID).Depth(utils.DepthQueryParam).Execute()
}

// GetBackupUnits returns all existing BackupUnits
func (cp *APIClient) GetBackupUnits(ctx context.Context) (sdkgo.BackupUnits, *sdkgo.APIResponse, error) {
	return cp.ComputeClient.BackupUnitsApi.BackupunitsGet(ctx).Depth(utils.DepthQueryParam).Execute()
}

// GetBackupUnitIDByName returns BackupUnit with the name specified
func (cp *APIClient) GetBackupUnitIDByName(ctx context.Context, backupUnitName string) (string, error) {
	backupUnits, _, err := cp.ComputeClient.BackupUnitsApi.BackupunitsGet(ctx).Depth(utils.DepthQueryParam).Filter("name", backupUnitName).Execute()
	if err != nil {
		return "", err
	}
	if items, ok := backupUnits.GetItemsOk(); ok && items != nil {
		if len(*items) == 0 {
			return "", fmt.Errorf("error getting ID of the BackupUnit named: %s - no BackupUnits found", backupUnitName)
		}
		if len(*items) > 1 {
			return "", fmt.Errorf("error getting ID of the BackupUnit named: %s - multiple BackupUnits with the same name found", backupUnitName)
		}
		if len(*items) == 1 {
			units := *items
			if idOk, ok := units[0].GetIdOk(); ok && idOk != nil {
				return *idOk, nil
			}
		}
	}
	return "", fmt.Errorf("error getting ID of the BackupUnit named: %s", backupUnitName)
}

// CreateBackupUnit based on BackupUnit properties
func (cp *APIClient) CreateBackupUnit(ctx context.Context, backupUnit sdkgo.BackupUnit) (sdkgo.BackupUnit, *sdkgo.APIResponse, error) {
	return cp.ComputeClient.BackupUnitsApi.BackupunitsPost(ctx).BackupUnit(backupUnit).Execute()
}

// UpdateBackupUnit based on backupUnitID and BackupUnit properties
func (cp *APIClient) UpdateBackupUnit(ctx context.Context, backupUnitID string, backupUnit sdkgo.BackupUnitProperties) (sdkgo.BackupUnit, *sdkgo.APIResponse, error) {
	return cp.ComputeClient.BackupUnitsApi.BackupunitsPatch(ctx, backupUnitID).BackupUnit(backupUnit).Execute()
}

// DeleteBackupUnit based on backupUnitID
func (cp *APIClient) DeleteBackupUnit(ctx context.Context, backupUnitID string) (*sdkgo.APIResponse, error) {
	return cp.ComputeClient.BackupUnitsApi.BackupunitsDelete(ctx, backupUnitID).Execute()
}

// GetAPIClient gets the APIClient
func (cp *APIClient) GetAPIClient() *sdkgo.APIClient {
	return cp.ComputeClient
}

// GenerateCreateBackupUnitInput returns BackupUnit based on the CR spec
func GenerateCreateBackupUnitInput(cr *v1alpha1.BackupUnit) (*sdkgo.BackupUnit, error) {
	instanceCreateInput := sdkgo.BackupUnit{
		Properties: &sdkgo.BackupUnitProperties{
			Name:     &cr.Spec.ForProvider.Name,
			Password: &cr.Spec.ForProvider.Password,
			Email:    &cr.Spec.ForProvider.Email,
		},
	}
	return &instanceCreateInput, nil
}

// GenerateUpdateBackupUnitInput returns BackupUnitProperties based on the CR spec modifications
func GenerateUpdateBackupUnitInput(cr *v1alpha1.BackupUnit) (*sdkgo.BackupUnitProperties, error) {
	instanceUpdateInput := sdkgo.BackupUnitProperties{
		Password: &cr.Spec.ForProvider.Password,
		Email:    &cr.Spec.ForProvider.Email,
	}
	return &instanceUpdateInput, nil
}

// IsBackupUnitUpToDate returns true if the BackupUnit is up-to-date or false if it does not
func IsBackupUnitUpToDate(cr *v1alpha1.BackupUnit, backupUnit sdkgo.BackupUnit) bool { // nolint:gocyclo
	switch {
	case cr == nil && backupUnit.Properties == nil:
		return true
	case cr == nil && backupUnit.Properties != nil:
		return false
	case cr != nil && backupUnit.Properties == nil:
		return false
	case backupUnit.Metadata != nil && backupUnit.Metadata.State != nil && *backupUnit.Metadata.State == "BUSY":
		return true
	case backupUnit.Properties.Name != nil && *backupUnit.Properties.Name != cr.Spec.ForProvider.Name:
		return false
	case backupUnit.Properties.Email != nil && *backupUnit.Properties.Email != cr.Spec.ForProvider.Email:
		return false
	default:
		return true
	}
}
