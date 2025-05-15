package server

import (
	"context"
	"fmt"
	"reflect"

	sdkgo "github.com/ionos-cloud/sdk-go/v6"

	"github.com/ionos-cloud/crossplane-provider-ionoscloud/apis/compute/v1alpha1"
	"github.com/ionos-cloud/crossplane-provider-ionoscloud/internal/clients"
	"github.com/ionos-cloud/crossplane-provider-ionoscloud/internal/utils"
)

var serverCubeType = "CUBE"

// APIClient is a wrapper around IONOS Service
type APIClient struct {
	*clients.IonosServices
}

// Client is a wrapper around IONOS Service Server methods
type Client interface {
	CheckDuplicateServer(ctx context.Context, datacenterID, serverName, cpuFamily string) (*sdkgo.Server, error)
	CheckDuplicateCubeServer(ctx context.Context, datacenterID, serverName, templateID string) (string, error)
	GetServer(ctx context.Context, datacenterID, serverID string) (sdkgo.Server, *sdkgo.APIResponse, error)
	CreateServer(ctx context.Context, datacenterID string, server sdkgo.Server) (sdkgo.Server, *sdkgo.APIResponse, error)
	UpdateServer(ctx context.Context, datacenterID, serverID string, server sdkgo.ServerProperties) (sdkgo.Server, *sdkgo.APIResponse, error)
	DeleteServer(ctx context.Context, datacenterID, serverID string) (*sdkgo.APIResponse, error)
	IsVolumeAttached(ctx context.Context, datacenterID, serverID, volumeID string) (bool, error)
	AttachVolume(ctx context.Context, datacenterID, serverID string, volume sdkgo.Volume) (sdkgo.Volume, *sdkgo.APIResponse, error)
	DetachVolume(ctx context.Context, datacenterID, serverID, volumeID string) (*sdkgo.APIResponse, error)
	AttachCdrom(ctx context.Context, datacenterID, serverID string, cdrom sdkgo.Image) (sdkgo.Image, *sdkgo.APIResponse, error)
	DetachCdrom(ctx context.Context, datacenterID, serverID, imageID string) (*sdkgo.APIResponse, error)
	GetAPIClient() *sdkgo.APIClient
	GetServerID(server *sdkgo.Server) (string, error)
}

// CheckDuplicateServer based on serverName, and the immutable property location
func (cp *APIClient) CheckDuplicateServer(ctx context.Context, datacenterID, serverName, cpuFamily string) (*sdkgo.Server, error) { // nolint: gocyclo
	servers, _, err := cp.IonosServices.ComputeClient.ServersApi.DatacentersServersGet(ctx, datacenterID).Depth(utils.DepthQueryParam).Execute()
	if err != nil {
		return nil, err
	}
	matchedItems := make([]sdkgo.Server, 0)
	if itemsOk, ok := servers.GetItemsOk(); ok && itemsOk != nil {
		for _, item := range *itemsOk {
			if propertiesOk, ok := item.GetPropertiesOk(); ok && propertiesOk != nil {
				if nameOk, ok := propertiesOk.GetNameOk(); ok && nameOk != nil {
					if *nameOk == serverName {
						// After checking the name, check the immutable properties
						if cpuFamilyOk, ok := propertiesOk.GetCpuFamilyOk(); ok && cpuFamilyOk != nil {
							if cpuFamily == "" || cpuFamily == *cpuFamilyOk {
								matchedItems = append(matchedItems, item)
							} else {
								return nil, fmt.Errorf("error: found server with the name %v, but immutable property cpuFamily different. expected: %v actual: %v", serverName, cpuFamily, *cpuFamilyOk)
							}
						}
					}
				}
			}
		}
	}
	if len(matchedItems) == 0 {
		return nil, nil
	}
	if len(matchedItems) > 1 {
		return nil, fmt.Errorf("error: found multiple servers with the name %v", serverName)
	}
	return &matchedItems[0], nil
}

// CheckDuplicateCubeServer based on serverName, and on the immutable properties
func (cp *APIClient) CheckDuplicateCubeServer(ctx context.Context, datacenterID, serverName, templateID string) (string, error) { // nolint: gocyclo
	servers, _, err := cp.IonosServices.ComputeClient.ServersApi.DatacentersServersGet(ctx, datacenterID).Depth(utils.DepthQueryParam).Execute()
	if err != nil {
		return "", err
	}
	matchedItems := make([]sdkgo.Server, 0)
	if servers.Items != nil {
		for _, item := range *servers.Items {
			if item.Properties != nil && item.Properties.Name != nil && *item.Properties.Name == serverName {
				if item.Properties.TemplateUuid != nil && *item.Properties.TemplateUuid != templateID {
					return "", fmt.Errorf("error: found cube server with the name %v, but immutable property templateId different. expected: %v actual: %v",
						serverName, templateID, *item.Properties.TemplateUuid)
				}
				matchedItems = append(matchedItems, item)
			}
		}
	}
	if len(matchedItems) == 0 {
		return "", nil
	}
	if len(matchedItems) > 1 {
		return "", fmt.Errorf("error: found multiple cube servers with the name %v", serverName)
	}
	if matchedItems[0].Id == nil {
		return "", fmt.Errorf("error getting ID for Cube Server named: %v", serverName)
	}
	return *matchedItems[0].Id, nil
}

// GetServerID based on datacenter
func (cp *APIClient) GetServerID(server *sdkgo.Server) (string, error) {
	if server != nil {
		if idOk, ok := server.GetIdOk(); ok && idOk != nil {
			return *idOk, nil
		}
		return "", fmt.Errorf("error: getting server id")
	}
	return "", nil
}

// GetServer based on datacenterID and serverID
func (cp *APIClient) GetServer(ctx context.Context, datacenterID, serverID string) (sdkgo.Server, *sdkgo.APIResponse, error) {
	// Use depth=2 to make sure the Managed Resource Server has access to server's entities
	return cp.IonosServices.ComputeClient.ServersApi.DatacentersServersFindById(ctx, datacenterID, serverID).Depth(int32(2)).Execute()
}

// CreateServer based on Server properties
func (cp *APIClient) CreateServer(ctx context.Context, datacenterID string, server sdkgo.Server) (sdkgo.Server, *sdkgo.APIResponse, error) {
	return cp.IonosServices.ComputeClient.ServersApi.DatacentersServersPost(ctx, datacenterID).Server(server).Execute()
}

// UpdateServer based on datacenterID, serverID and Server properties
func (cp *APIClient) UpdateServer(ctx context.Context, datacenterID, serverID string, server sdkgo.ServerProperties) (sdkgo.Server, *sdkgo.APIResponse, error) {
	return cp.IonosServices.ComputeClient.ServersApi.DatacentersServersPatch(ctx, datacenterID, serverID).Server(server).Execute()
}

// DeleteServer based on datacenterID, serverID
func (cp *APIClient) DeleteServer(ctx context.Context, datacenterID, serverID string) (*sdkgo.APIResponse, error) {
	return cp.IonosServices.ComputeClient.ServersApi.DatacentersServersDelete(ctx, datacenterID, serverID).Execute()
}

// AttachVolume based on datacenterID, serverID, and volume
func (cp *APIClient) AttachVolume(ctx context.Context, datacenterID, serverID string, volume sdkgo.Volume) (sdkgo.Volume, *sdkgo.APIResponse, error) {
	return cp.IonosServices.ComputeClient.ServersApi.DatacentersServersVolumesPost(ctx, datacenterID, serverID).Volume(volume).Execute()
}

// IsVolumeAttached based on datacenterID, serverID, and volume
func (cp *APIClient) IsVolumeAttached(ctx context.Context, datacenterID, serverID string, volumeID string) (bool, error) {
	_, apiResponse, err := cp.IonosServices.ComputeClient.ServersApi.DatacentersServersVolumesFindById(ctx, datacenterID, serverID, volumeID).Execute()
	if apiResponse.HttpNotFound() {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, nil
}

// DetachVolume based on datacenterID, serverID, and volume
func (cp *APIClient) DetachVolume(ctx context.Context, datacenterID, serverID, volumeID string) (*sdkgo.APIResponse, error) {
	return cp.IonosServices.ComputeClient.ServersApi.DatacentersServersVolumesDelete(ctx, datacenterID, serverID, volumeID).Execute()
}

// AttachCdrom based on datacenterID, serverID, and image
func (cp *APIClient) AttachCdrom(ctx context.Context, datacenterID, serverID string, cdrom sdkgo.Image) (sdkgo.Image, *sdkgo.APIResponse, error) {
	return cp.IonosServices.ComputeClient.ServersApi.DatacentersServersCdromsPost(ctx, datacenterID, serverID).Cdrom(cdrom).Execute()
}

// DetachCdrom based on datacenterID, serverID, and imageId
func (cp *APIClient) DetachCdrom(ctx context.Context, datacenterID, serverID, imageID string) (*sdkgo.APIResponse, error) {
	return cp.IonosServices.ComputeClient.ServersApi.DatacentersServersCdromsDelete(ctx, datacenterID, serverID, imageID).Execute()
}

// GetAPIClient gets the APIClient
func (cp *APIClient) GetAPIClient() *sdkgo.APIClient {
	return cp.IonosServices.ComputeClient
}

// GenerateCreateServerInput returns sdkgo.Server based on the CR spec
func GenerateCreateServerInput(cr *v1alpha1.Server) *sdkgo.Server {
	properties := GenerateUpdateServerInput(cr)

	return &sdkgo.Server{
		Properties: properties,
	}
}

// GenerateUpdateServerInput returns sdkgo.ServerProperties based on the CR spec modifications
func GenerateUpdateServerInput(cr *v1alpha1.Server) *sdkgo.ServerProperties {
	instanceUpdateInput := sdkgo.ServerProperties{
		Name:  &cr.Spec.ForProvider.Name,
		Cores: &cr.Spec.ForProvider.Cores,
		Ram:   &cr.Spec.ForProvider.RAM,
	}
	// Set CPUFamily only if it is specified into the Spec.
	// If not, the CPUFamily will be set corresponding with the datacenter's location
	if cr.Spec.ForProvider.CPUFamily != "" {
		instanceUpdateInput.SetCpuFamily(cr.Spec.ForProvider.CPUFamily)
	}
	// Set AvailabilityZone only if it is specified into the Spec.
	// If not, the AvailabilityZone will be set to the default value by the API
	if cr.Spec.ForProvider.AvailabilityZone != "" {
		instanceUpdateInput.SetAvailabilityZone(cr.Spec.ForProvider.AvailabilityZone)
	}
	if cr.Spec.ForProvider.BootCdromID != "" {
		instanceUpdateInput.SetBootCdrom(sdkgo.ResourceReference{Id: &cr.Spec.ForProvider.BootCdromID})
	}
	if cr.Spec.ForProvider.VolumeCfg.VolumeID != "" {
		instanceUpdateInput.SetBootVolume(sdkgo.ResourceReference{Id: &cr.Spec.ForProvider.VolumeCfg.VolumeID})
	}
	if cr.Spec.ForProvider.PlacementGroupID != "" {
		instanceUpdateInput.SetPlacementGroupId(cr.Spec.ForProvider.PlacementGroupID)
	}
	return &instanceUpdateInput
}

// LateStatusInitializer fills the empty fields in *v1alpha1.ServerParameters with
// the values seen in sdkgo.Server.
func LateStatusInitializer(in *v1alpha1.ServerObservation, sg *sdkgo.Server) {
	if sg == nil {
		return
	}
	// Add options to the Spec, if they were updated by the API
	if propertiesOk, ok := sg.GetPropertiesOk(); ok && propertiesOk != nil {
		if cpuFamilyOk, ok := propertiesOk.GetCpuFamilyOk(); ok && cpuFamilyOk != nil {
			if utils.IsEmptyValue(reflect.ValueOf(in.CPUFamily)) {
				in.CPUFamily = *cpuFamilyOk
			}
		}
	}
}

// LateInitializer fills the empty fields in *v1alpha1.ServerParameters with
// the values seen in sdkgo.Server.
func LateInitializer(in *v1alpha1.ServerParameters, sg *sdkgo.Server) {
	if sg == nil {
		return
	}
	// Add options to the Spec, if they were updated by the API
	if propertiesOk, ok := sg.GetPropertiesOk(); ok && propertiesOk != nil {
		if bootCdromOk, ok := propertiesOk.GetBootCdromOk(); ok && bootCdromOk != nil {
			if utils.IsEmptyValue(reflect.ValueOf(in.BootCdromID)) {
				in.BootCdromID = *bootCdromOk.Id
			}
		}
	}
}

// IsServerUpToDate returns true if the Server is up-to-date or false if it does not
func IsServerUpToDate(cr *v1alpha1.Server, server sdkgo.Server) bool { // nolint:gocyclo
	switch {
	case cr == nil && server.Properties == nil:
		return true
	case cr == nil && server.Properties != nil:
		return false
	case cr != nil && server.Properties == nil:
		return false
	case server.Properties.Name != nil && cr.Spec.ForProvider.Name != *server.Properties.Name:
		return false
	case server.Properties.Name == nil && cr.Spec.ForProvider.Name != "":
		return false
	case server.Properties.Cores != nil && cr.Spec.ForProvider.Cores != *server.Properties.Cores:
		return false
	case server.Properties.Ram != nil && cr.Spec.ForProvider.RAM != *server.Properties.Ram:
		return false
	case server.Properties.CpuFamily != nil && cr.Spec.ForProvider.CPUFamily != "" && cr.Spec.ForProvider.CPUFamily != *server.Properties.CpuFamily:
		return false
	case server.Properties.AvailabilityZone != nil && cr.Spec.ForProvider.AvailabilityZone != *server.Properties.AvailabilityZone:
		return false
	case server.Metadata != nil && server.Metadata.State != nil && *server.Metadata.State == sdkgo.Busy:
		return true
	case cr.Spec.ForProvider.VolumeCfg.VolumeID != cr.Status.AtProvider.VolumeID:
		return false
	case server.Properties.PlacementGroupId != nil && cr.Spec.ForProvider.PlacementGroupID != *server.Properties.PlacementGroupId:
		return false
	default:
		return true
	}
}

// GenerateCreateCubeServerInput returns CreateServerRequest based on the CR spec
func GenerateCreateCubeServerInput(cr *v1alpha1.CubeServer, templateID string) (*sdkgo.Server, error) { // nolint:gocyclo
	// Cube Server has a DAS Volume attached to it
	var volumeType = "DAS"

	// Get DAS Volume Input
	dasVolumeInput := sdkgo.Volume{
		Properties: &sdkgo.VolumeProperties{
			Name:                &cr.Spec.ForProvider.DasVolumeProperties.Name,
			Type:                &volumeType,
			Bus:                 &cr.Spec.ForProvider.DasVolumeProperties.Bus,
			CpuHotPlug:          &cr.Spec.ForProvider.DasVolumeProperties.CPUHotPlug,
			RamHotPlug:          &cr.Spec.ForProvider.DasVolumeProperties.RAMHotPlug,
			NicHotPlug:          &cr.Spec.ForProvider.DasVolumeProperties.NicHotPlug,
			NicHotUnplug:        &cr.Spec.ForProvider.DasVolumeProperties.NicHotUnplug,
			DiscVirtioHotPlug:   &cr.Spec.ForProvider.DasVolumeProperties.DiscVirtioHotPlug,
			DiscVirtioHotUnplug: &cr.Spec.ForProvider.DasVolumeProperties.DiscVirtioHotUnplug,
		},
	}
	if !utils.IsEmptyValue(reflect.ValueOf(cr.Spec.ForProvider.DasVolumeProperties.Image)) {
		dasVolumeInput.Properties.SetImage(cr.Spec.ForProvider.DasVolumeProperties.Image)
	}
	if !utils.IsEmptyValue(reflect.ValueOf(cr.Spec.ForProvider.DasVolumeProperties.ImageAlias)) {
		dasVolumeInput.Properties.SetImageAlias(cr.Spec.ForProvider.DasVolumeProperties.ImageAlias)
	}
	if !utils.IsEmptyValue(reflect.ValueOf(cr.Spec.ForProvider.DasVolumeProperties.ImagePassword)) {
		dasVolumeInput.Properties.SetImagePassword(cr.Spec.ForProvider.DasVolumeProperties.ImagePassword)
	}
	if !utils.IsEmptyValue(reflect.ValueOf(cr.Spec.ForProvider.DasVolumeProperties.SSHKeys)) {
		dasVolumeInput.Properties.SetSshKeys(cr.Spec.ForProvider.DasVolumeProperties.SSHKeys)
	}
	if !utils.IsEmptyValue(reflect.ValueOf(cr.Spec.ForProvider.DasVolumeProperties.LicenceType)) {
		dasVolumeInput.Properties.SetLicenceType(cr.Spec.ForProvider.DasVolumeProperties.LicenceType)
	}
	if !utils.IsEmptyValue(reflect.ValueOf(cr.Spec.ForProvider.DasVolumeProperties.BackupUnitCfg.BackupUnitID)) {
		dasVolumeInput.Properties.SetBackupunitId(cr.Spec.ForProvider.DasVolumeProperties.BackupUnitCfg.BackupUnitID)
	}
	if !utils.IsEmptyValue(reflect.ValueOf(cr.Spec.ForProvider.DasVolumeProperties.UserData)) {
		dasVolumeInput.Properties.SetUserData(cr.Spec.ForProvider.DasVolumeProperties.UserData)
	}

	// Create Server Input
	instanceCreateInput := sdkgo.Server{
		Properties: &sdkgo.ServerProperties{
			Name:             &cr.Spec.ForProvider.Name,
			TemplateUuid:     &templateID,
			AvailabilityZone: &cr.Spec.ForProvider.AvailabilityZone,
			Type:             &serverCubeType,
		},
		Entities: &sdkgo.ServerEntities{Volumes: &sdkgo.AttachedVolumes{Items: &[]sdkgo.Volume{dasVolumeInput}}},
	}
	return &instanceCreateInput, nil
}

// GenerateUpdateCubeServerInput returns PatchServerRequest based on the CR spec modifications
func GenerateUpdateCubeServerInput(cr *v1alpha1.CubeServer) (*sdkgo.ServerProperties, error) {
	instanceUpdateInput := sdkgo.ServerProperties{
		Name: &cr.Spec.ForProvider.Name,
	}
	if !utils.IsEmptyValue(reflect.ValueOf(cr.Status.AtProvider.VolumeID)) {
		instanceUpdateInput.SetBootVolume(sdkgo.ResourceReference{Id: &cr.Status.AtProvider.VolumeID})
	}
	return &instanceUpdateInput, nil
}

// GenerateUpdateVolumeInput returns VolumeProperties based on the CR spec modifications
func GenerateUpdateVolumeInput(cr *v1alpha1.CubeServer) (*sdkgo.VolumeProperties, error) {
	instanceUpdateInput := sdkgo.VolumeProperties{
		Name:                &cr.Spec.ForProvider.DasVolumeProperties.Name,
		Bus:                 &cr.Spec.ForProvider.DasVolumeProperties.Bus,
		CpuHotPlug:          &cr.Spec.ForProvider.DasVolumeProperties.CPUHotPlug,
		RamHotPlug:          &cr.Spec.ForProvider.DasVolumeProperties.RAMHotPlug,
		NicHotPlug:          &cr.Spec.ForProvider.DasVolumeProperties.NicHotPlug,
		NicHotUnplug:        &cr.Spec.ForProvider.DasVolumeProperties.NicHotUnplug,
		DiscVirtioHotPlug:   &cr.Spec.ForProvider.DasVolumeProperties.DiscVirtioHotPlug,
		DiscVirtioHotUnplug: &cr.Spec.ForProvider.DasVolumeProperties.DiscVirtioHotUnplug,
	}
	return &instanceUpdateInput, nil
}

// LateInitializerCube fills the empty fields in *v1alpha1.CubeServerProperties with
// the values seen in sdkgo.Server.
func LateInitializerCube(in *v1alpha1.CubeServerProperties, sg *sdkgo.Server) {
	if sg == nil {
		return
	}
	// Add options to the Spec, if they were updated by the API
	if propertiesOk, ok := sg.GetPropertiesOk(); ok && propertiesOk != nil {
		if templateUUIDOk, ok := propertiesOk.GetTemplateUuidOk(); ok && templateUUIDOk != nil {
			if utils.IsEmptyValue(reflect.ValueOf(in.Template.TemplateID)) {
				in.Template.TemplateID = *templateUUIDOk
			}
		}
	}
}

// IsCubeServerUpToDate returns true if the Server is up-to-date or false if it does not
func IsCubeServerUpToDate(cr *v1alpha1.CubeServer, server sdkgo.Server) bool { // nolint:gocyclo
	switch {
	case cr == nil && server.Properties == nil:
		return true
	case cr == nil && server.Properties != nil:
		return false
	case cr != nil && server.Properties == nil:
		return false
	case server.Metadata.State != nil && *server.Metadata.State == sdkgo.Busy:
		return true
	case server.Properties.Name != nil && *server.Properties.Name != cr.Spec.ForProvider.Name:
		return false
	case server.Properties.Name == nil && cr.Spec.ForProvider.Name != "":
		return false
	case server.Properties.BootVolume != nil && *server.Properties.BootVolume.Id != cr.Status.AtProvider.VolumeID:
		return false
	case cr.Status.AtProvider.VolumeID != "" && !server.Properties.HasBootVolume():
		return false
	}
	if server.HasEntities() && server.Entities.HasVolumes() && server.Entities.Volumes.HasItems() {
		items := *server.Entities.Volumes.Items
		if len(items) > 0 {
			if propertiesOk, ok := items[0].GetPropertiesOk(); ok && propertiesOk != nil {
				if nameOk, ok := propertiesOk.GetNameOk(); ok && *nameOk != cr.Spec.ForProvider.DasVolumeProperties.Name {
					return false
				}
				if busOk, ok := propertiesOk.GetBusOk(); ok && *busOk != cr.Spec.ForProvider.DasVolumeProperties.Bus {
					return false
				}
				if cpuHotPlugOk, ok := propertiesOk.GetCpuHotPlugOk(); ok && *cpuHotPlugOk != cr.Spec.ForProvider.DasVolumeProperties.CPUHotPlug {
					return false
				}
				if ramHotPlugOk, ok := propertiesOk.GetRamHotPlugOk(); ok && *ramHotPlugOk != cr.Spec.ForProvider.DasVolumeProperties.RAMHotPlug {
					return false
				}
				if nicHotPlugOk, ok := propertiesOk.GetNicHotPlugOk(); ok && *nicHotPlugOk != cr.Spec.ForProvider.DasVolumeProperties.NicHotPlug {
					return false
				}
				if nicHotUnplugOk, ok := propertiesOk.GetNicHotUnplugOk(); ok && *nicHotUnplugOk != cr.Spec.ForProvider.DasVolumeProperties.NicHotUnplug {
					return false
				}
				if discVirtioHotPlugOk, ok := propertiesOk.GetDiscVirtioHotPlugOk(); ok && *discVirtioHotPlugOk != cr.Spec.ForProvider.DasVolumeProperties.DiscVirtioHotPlug {
					return false
				}
				if discVirtioHotUnplugOk, ok := propertiesOk.GetDiscVirtioHotUnplugOk(); ok && *discVirtioHotUnplugOk != cr.Spec.ForProvider.DasVolumeProperties.DiscVirtioHotUnplug {
					return false
				}
			}
		}
	}
	return true
}
