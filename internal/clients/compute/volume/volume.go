package volume

import (
	"context"
	"fmt"
	"reflect"

	sdkgo "github.com/ionos-cloud/sdk-go/v6"

	"github.com/ionos-cloud/crossplane-provider-ionoscloud/apis/compute/v1alpha1"
	"github.com/ionos-cloud/crossplane-provider-ionoscloud/internal/clients"
	"github.com/ionos-cloud/crossplane-provider-ionoscloud/internal/utils"
)

// APIClient is a wrapper around IONOS Service
type APIClient struct {
	*clients.IonosServices
}

// Client is a wrapper around IONOS Service Volume methods
type Client interface {
	CheckDuplicateVolume(ctx context.Context, datacenterID, volumeName, storageType, availabilityZone, licenceType, image string) (*sdkgo.Volume, error)
	GetVolumeID(datacenter *sdkgo.Volume) (string, error)
	GetVolume(ctx context.Context, datacenterID, volumeID string) (sdkgo.Volume, *sdkgo.APIResponse, error)
	CreateVolume(ctx context.Context, datacenterID string, volume sdkgo.Volume) (sdkgo.Volume, *sdkgo.APIResponse, error)
	UpdateVolume(ctx context.Context, datacenterID, volumeID string, volume sdkgo.VolumeProperties) (sdkgo.Volume, *sdkgo.APIResponse, error)
	DeleteVolume(ctx context.Context, datacenterID, volumeID string) (*sdkgo.APIResponse, error)
	GetAPIClient() *sdkgo.APIClient
	GetServerNameByID(ctx context.Context, datacenterID, serverID string) (string, error)
}

// CheckDuplicateVolume based on datacenterID, volumeName
func (cp *APIClient) CheckDuplicateVolume(ctx context.Context, datacenterID, volumeName, storageType, availabilityZone, licenceType, image string) (*sdkgo.Volume, error) { // nolint: gocyclo
	volumes, _, err := cp.ComputeClient.VolumesApi.DatacentersVolumesGet(ctx, datacenterID).Depth(utils.DepthQueryParam).Execute()
	if err != nil {
		return nil, err
	}
	matchedItems := make([]sdkgo.Volume, 0)
	if itemsOk, ok := volumes.GetItemsOk(); ok && itemsOk != nil {
		for _, item := range *itemsOk {
			if propertiesOk, ok := item.GetPropertiesOk(); ok && propertiesOk != nil {
				if nameOk, ok := propertiesOk.GetNameOk(); ok && nameOk != nil {
					if *nameOk == volumeName {
						// After checking the name, check the immutable properties
						if typeOk, ok := propertiesOk.GetTypeOk(); ok && typeOk != nil {
							if *typeOk != storageType {
								return nil, fmt.Errorf("error: found volume with the name %v, but immutable property type different. expected: %v actual: %v", volumeName, storageType, *typeOk)
							}
						}
						if availabilityZoneOk, ok := propertiesOk.GetAvailabilityZoneOk(); ok && availabilityZoneOk != nil {
							if *availabilityZoneOk != availabilityZone {
								return nil, fmt.Errorf("error: found volume with the name %v, but immutable property availabilityZone different. expected: %v actual: %v", volumeName, availabilityZone, *availabilityZoneOk)
							}
						}
						if licenceTypeOk, ok := propertiesOk.GetLicenceTypeOk(); ok && licenceTypeOk != nil && licenceType != "" {
							if *licenceTypeOk != licenceType {
								return nil, fmt.Errorf("error: found volume with the name %v, but immutable property licenceType different. expected: %v actual: %v", volumeName, licenceType, *licenceTypeOk)
							}
						}
						if imageOk, ok := propertiesOk.GetImageOk(); ok && imageOk != nil && image != "" {
							if *imageOk != image {
								return nil, fmt.Errorf("error: found volume with the name %v, but immutable property image different. expected: %v actual: %v", volumeName, image, *imageOk)
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
		return nil, fmt.Errorf("error: found multiple volumes with the name %v", volumeName)
	}
	return &matchedItems[0], nil
}

// GetVolumeID based on volume
func (cp *APIClient) GetVolumeID(volume *sdkgo.Volume) (string, error) {
	if volume != nil {
		if idOk, ok := volume.GetIdOk(); ok && idOk != nil {
			return *idOk, nil
		}
		return "", fmt.Errorf("error: getting volume id")
	}
	return "", nil
}

// GetServerNameByID based on boot server ID
func (cp *APIClient) GetServerNameByID(ctx context.Context, datacenterID, serverID string) (string, error) {
	if serverID != "" && datacenterID != "" {
		server, apiResponse, err := cp.ComputeClient.ServersApi.DatacentersServersFindById(ctx, datacenterID, serverID).Execute()
		if apiResponse.HttpNotFound() {
			return "", nil
		}
		if err != nil {
			return "", err
		}
		if server.Properties == nil || server.Properties.Name == nil {
			return "", fmt.Errorf("error: getting server properties")
		}
		return *server.Properties.Name, nil
	}
	return "", nil
}

// GetVolume based on datacenterID and volumeID
func (cp *APIClient) GetVolume(ctx context.Context, datacenterID, volumeID string) (sdkgo.Volume, *sdkgo.APIResponse, error) {
	return cp.ComputeClient.VolumesApi.DatacentersVolumesFindById(ctx, datacenterID, volumeID).Depth(utils.DepthQueryParam).Execute()
}

// CreateVolume based on Volume properties
func (cp *APIClient) CreateVolume(ctx context.Context, datacenterID string, volume sdkgo.Volume) (sdkgo.Volume, *sdkgo.APIResponse, error) {
	return cp.ComputeClient.VolumesApi.DatacentersVolumesPost(ctx, datacenterID).Volume(volume).Execute()
}

// UpdateVolume based on datacenterID, volumeID and Volume properties
func (cp *APIClient) UpdateVolume(ctx context.Context, datacenterID, volumeID string, volume sdkgo.VolumeProperties) (sdkgo.Volume, *sdkgo.APIResponse, error) {
	return cp.ComputeClient.VolumesApi.DatacentersVolumesPatch(ctx, datacenterID, volumeID).Volume(volume).Execute()
}

// DeleteVolume based on datacenterID, volumeID
func (cp *APIClient) DeleteVolume(ctx context.Context, datacenterID, volumeID string) (*sdkgo.APIResponse, error) {
	return cp.ComputeClient.VolumesApi.DatacentersVolumesDelete(ctx, datacenterID, volumeID).Execute()
}

// GetAPIClient gets the APIClient
func (cp *APIClient) GetAPIClient() *sdkgo.APIClient {
	return cp.ComputeClient
}

// GenerateCreateVolumeInput returns sdkgo.Volume based on the CR spec
func GenerateCreateVolumeInput(cr *v1alpha1.Volume) (*sdkgo.Volume, error) { // nolint:gocyclo
	instanceCreateInput := sdkgo.Volume{
		Properties: &sdkgo.VolumeProperties{
			Name:                &cr.Spec.ForProvider.Name,
			Type:                &cr.Spec.ForProvider.Type,
			Size:                &cr.Spec.ForProvider.Size,
			CpuHotPlug:          &cr.Spec.ForProvider.CPUHotPlug,
			RamHotPlug:          &cr.Spec.ForProvider.RAMHotPlug,
			NicHotPlug:          &cr.Spec.ForProvider.NicHotPlug,
			NicHotUnplug:        &cr.Spec.ForProvider.NicHotUnplug,
			DiscVirtioHotPlug:   &cr.Spec.ForProvider.DiscVirtioHotPlug,
			DiscVirtioHotUnplug: &cr.Spec.ForProvider.DiscVirtioHotUnplug,
		},
	}
	if !utils.IsEmptyValue(reflect.ValueOf(cr.Spec.ForProvider.AvailabilityZone)) {
		instanceCreateInput.Properties.SetAvailabilityZone(cr.Spec.ForProvider.AvailabilityZone)
	}
	if !utils.IsEmptyValue(reflect.ValueOf(cr.Spec.ForProvider.Bus)) {
		instanceCreateInput.Properties.SetBus(cr.Spec.ForProvider.Bus)
	}
	if !utils.IsEmptyValue(reflect.ValueOf(cr.Spec.ForProvider.Image)) {
		instanceCreateInput.Properties.SetImage(cr.Spec.ForProvider.Image)
	}
	if !utils.IsEmptyValue(reflect.ValueOf(cr.Spec.ForProvider.ImageAlias)) {
		instanceCreateInput.Properties.SetImageAlias(cr.Spec.ForProvider.ImageAlias)
	}
	if !utils.IsEmptyValue(reflect.ValueOf(cr.Spec.ForProvider.ImagePassword)) {
		instanceCreateInput.Properties.SetImagePassword(cr.Spec.ForProvider.ImagePassword)
	}
	if !utils.IsEmptyValue(reflect.ValueOf(cr.Spec.ForProvider.SSHKeys)) {
		instanceCreateInput.Properties.SetSshKeys(cr.Spec.ForProvider.SSHKeys)
	}
	if !utils.IsEmptyValue(reflect.ValueOf(cr.Spec.ForProvider.LicenceType)) {
		instanceCreateInput.Properties.SetLicenceType(cr.Spec.ForProvider.LicenceType)
	}
	if !utils.IsEmptyValue(reflect.ValueOf(cr.Spec.ForProvider.BackupUnitCfg.BackupUnitID)) {
		instanceCreateInput.Properties.SetBackupunitId(cr.Spec.ForProvider.BackupUnitCfg.BackupUnitID)
	}
	if !utils.IsEmptyValue(reflect.ValueOf(cr.Spec.ForProvider.UserData)) {
		instanceCreateInput.Properties.SetUserData(cr.Spec.ForProvider.UserData)
	}
	return &instanceCreateInput, nil
}

// GenerateUpdateVolumeInput returns sdkgo.VolumeProperties based on the CR spec modifications
func GenerateUpdateVolumeInput(cr *v1alpha1.Volume, properties *sdkgo.VolumeProperties) (*sdkgo.VolumeProperties, error) {
	instanceUpdateInput := sdkgo.VolumeProperties{
		Name:                &cr.Spec.ForProvider.Name,
		CpuHotPlug:          &cr.Spec.ForProvider.CPUHotPlug,
		RamHotPlug:          &cr.Spec.ForProvider.RAMHotPlug,
		NicHotPlug:          &cr.Spec.ForProvider.NicHotPlug,
		NicHotUnplug:        &cr.Spec.ForProvider.NicHotUnplug,
		DiscVirtioHotPlug:   &cr.Spec.ForProvider.DiscVirtioHotPlug,
		DiscVirtioHotUnplug: &cr.Spec.ForProvider.DiscVirtioHotUnplug,
	}
	if !utils.IsEmptyValue(reflect.ValueOf(cr.Spec.ForProvider.Bus)) {
		instanceUpdateInput.SetBus(cr.Spec.ForProvider.Bus)
	}
	if properties != nil {
		if properties.HasSize() {
			if *properties.Size != cr.Spec.ForProvider.Size {
				instanceUpdateInput.SetSize(cr.Spec.ForProvider.Size)
			}
		}
	}
	return &instanceUpdateInput, nil
}

// IsVolumeUpToDate returns true if the Volume is up-to-date or false if it does not
func IsVolumeUpToDate(cr *v1alpha1.Volume, volume *sdkgo.Volume) bool { // nolint:gocyclo
	switch {
	case cr == nil && volume.Properties == nil:
		return true
	case cr == nil && volume.Properties != nil:
		return false
	case cr != nil && volume.Properties == nil:
		return false
	case volume.Metadata.State != nil && *volume.Metadata.State == "BUSY":
		return true
	case volume.Properties.Name != nil && *volume.Properties.Name != cr.Spec.ForProvider.Name:
		return false
	case volume.Properties.Name == nil && cr.Spec.ForProvider.Name != "":
		return false
	case volume.Properties.Size != nil && *volume.Properties.Size != cr.Spec.ForProvider.Size:
		return false
	case volume.Properties.CpuHotPlug != nil && *volume.Properties.CpuHotPlug != cr.Spec.ForProvider.CPUHotPlug:
		return false
	case volume.Properties.RamHotPlug != nil && *volume.Properties.RamHotPlug != cr.Spec.ForProvider.RAMHotPlug:
		return false
	case volume.Properties.NicHotPlug != nil && *volume.Properties.NicHotPlug != cr.Spec.ForProvider.NicHotPlug:
		return false
	case volume.Properties.NicHotUnplug != nil && *volume.Properties.NicHotUnplug != cr.Spec.ForProvider.NicHotUnplug:
		return false
	case volume.Properties.DiscVirtioHotPlug != nil && *volume.Properties.DiscVirtioHotPlug != cr.Spec.ForProvider.DiscVirtioHotPlug:
		return false
	case volume.Properties.DiscVirtioHotUnplug != nil && *volume.Properties.DiscVirtioHotUnplug != cr.Spec.ForProvider.DiscVirtioHotUnplug:
		return false
	case volume.Properties.Bus != nil && *volume.Properties.Bus != cr.Spec.ForProvider.Bus:
		return false
	default:
		return true
	}
}
