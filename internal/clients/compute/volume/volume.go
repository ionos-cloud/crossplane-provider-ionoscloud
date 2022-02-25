package volume

import (
	"context"
	"reflect"
	"strings"

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
	GetVolume(ctx context.Context, datacenterID, volumeID string) (sdkgo.Volume, *sdkgo.APIResponse, error)
	CreateVolume(ctx context.Context, datacenterID string, volume sdkgo.Volume) (sdkgo.Volume, *sdkgo.APIResponse, error)
	UpdateVolume(ctx context.Context, datacenterID, volumeID string, volume sdkgo.VolumeProperties) (sdkgo.Volume, *sdkgo.APIResponse, error)
	DeleteVolume(ctx context.Context, datacenterID, volumeID string) (*sdkgo.APIResponse, error)
	GetAPIClient() *sdkgo.APIClient
}

// GetVolume based on datacenterID and volumeID
func (cp *APIClient) GetVolume(ctx context.Context, datacenterID, volumeID string) (sdkgo.Volume, *sdkgo.APIResponse, error) {
	return cp.ComputeClient.VolumesApi.DatacentersVolumesFindById(ctx, datacenterID, volumeID).Execute()
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

// GenerateCreateVolumeInput returns CreateVolumeRequest based on the CR spec
//nolint
func GenerateCreateVolumeInput(cr *v1alpha1.Volume) (*sdkgo.Volume, error) {
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
	if !utils.IsEmptyValue(reflect.ValueOf(cr.Spec.ForProvider.BackupunitID)) {
		instanceCreateInput.Properties.SetBackupunitId(cr.Spec.ForProvider.BackupunitID)
	}
	if !utils.IsEmptyValue(reflect.ValueOf(cr.Spec.ForProvider.UserData)) {
		instanceCreateInput.Properties.SetUserData(cr.Spec.ForProvider.UserData)
	}
	return &instanceCreateInput, nil
}

// GenerateUpdateVolumeInput returns PatchVolumeRequest based on the CR spec modifications
//nolint
func GenerateUpdateVolumeInput(cr *v1alpha1.Volume) (*sdkgo.VolumeProperties, error) {
	instanceUpdateInput := sdkgo.VolumeProperties{
		Name:                &cr.Spec.ForProvider.Name,
		Type:                &cr.Spec.ForProvider.Type,
		Size:                &cr.Spec.ForProvider.Size,
		CpuHotPlug:          &cr.Spec.ForProvider.CPUHotPlug,
		RamHotPlug:          &cr.Spec.ForProvider.RAMHotPlug,
		NicHotPlug:          &cr.Spec.ForProvider.NicHotPlug,
		NicHotUnplug:        &cr.Spec.ForProvider.NicHotUnplug,
		DiscVirtioHotPlug:   &cr.Spec.ForProvider.DiscVirtioHotPlug,
		DiscVirtioHotUnplug: &cr.Spec.ForProvider.DiscVirtioHotUnplug,
	}
	if !utils.IsEmptyValue(reflect.ValueOf(cr.Spec.ForProvider.AvailabilityZone)) {
		instanceUpdateInput.SetAvailabilityZone(cr.Spec.ForProvider.AvailabilityZone)
	}
	if !utils.IsEmptyValue(reflect.ValueOf(cr.Spec.ForProvider.Bus)) {
		instanceUpdateInput.SetBus(cr.Spec.ForProvider.Bus)
	}
	if !utils.IsEmptyValue(reflect.ValueOf(cr.Spec.ForProvider.Image)) {
		instanceUpdateInput.SetImage(cr.Spec.ForProvider.Image)
	}
	if !utils.IsEmptyValue(reflect.ValueOf(cr.Spec.ForProvider.ImageAlias)) {
		instanceUpdateInput.SetImageAlias(cr.Spec.ForProvider.ImageAlias)
	}
	if !utils.IsEmptyValue(reflect.ValueOf(cr.Spec.ForProvider.ImagePassword)) {
		instanceUpdateInput.SetImagePassword(cr.Spec.ForProvider.ImagePassword)
	}
	if !utils.IsEmptyValue(reflect.ValueOf(cr.Spec.ForProvider.SSHKeys)) {
		instanceUpdateInput.SetSshKeys(cr.Spec.ForProvider.SSHKeys)
	}
	if !utils.IsEmptyValue(reflect.ValueOf(cr.Spec.ForProvider.LicenceType)) {
		instanceUpdateInput.SetLicenceType(cr.Spec.ForProvider.LicenceType)
	}
	if !utils.IsEmptyValue(reflect.ValueOf(cr.Spec.ForProvider.BackupunitID)) {
		instanceUpdateInput.SetBackupunitId(cr.Spec.ForProvider.BackupunitID)
	}
	if !utils.IsEmptyValue(reflect.ValueOf(cr.Spec.ForProvider.UserData)) {
		instanceUpdateInput.SetUserData(cr.Spec.ForProvider.UserData)
	}
	return &instanceUpdateInput, nil
}

// IsVolumeUpToDate returns true if the Volume is up-to-date or false if it does not
//nolint
func IsVolumeUpToDate(cr *v1alpha1.Volume, volume *sdkgo.Volume) bool {
	switch {
	case cr == nil && volume.Properties == nil:
		return true
	case cr == nil && volume.Properties != nil:
		return false
	case cr != nil && volume.Properties == nil:
		return false
	}
	if volume != nil {
		if metadataOk, ok := volume.GetMetadataOk(); ok && metadataOk != nil {
			if strings.Compare("BUSY", *metadataOk.State) == 0 {
				return false
			}
		}
		if propertiesOk, ok := volume.GetPropertiesOk(); ok && propertiesOk != nil {
			if strings.Compare(cr.Spec.ForProvider.Name, *propertiesOk.Name) != 0 {
				return false
			}
		}
	}
	return true
}
