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

const depthQueryParam = int32(5)

var serverCubeType = "CUBE"

// APIClient is a wrapper around IONOS Service
type APIClient struct {
	*clients.IonosServices
}

// Client is a wrapper around IONOS Service Server methods
type Client interface {
	GetServer(ctx context.Context, datacenterID, serverID string) (sdkgo.Server, *sdkgo.APIResponse, error)
	CreateServer(ctx context.Context, datacenterID string, server sdkgo.Server) (sdkgo.Server, *sdkgo.APIResponse, error)
	UpdateServer(ctx context.Context, datacenterID, serverID string, server sdkgo.ServerProperties) (sdkgo.Server, *sdkgo.APIResponse, error)
	DeleteServer(ctx context.Context, datacenterID, serverID string) (*sdkgo.APIResponse, error)
	AttachVolume(ctx context.Context, datacenterID, serverID string, volume sdkgo.Volume) (sdkgo.Volume, *sdkgo.APIResponse, error)
	DetachVolume(ctx context.Context, datacenterID, serverID, volumeID string) (*sdkgo.APIResponse, error)
	AttachCdrom(ctx context.Context, datacenterID, serverID string, cdrom sdkgo.Image) (sdkgo.Image, *sdkgo.APIResponse, error)
	DetachCdrom(ctx context.Context, datacenterID, serverID, imageID string) (*sdkgo.APIResponse, error)
	GetAPIClient() *sdkgo.APIClient
}

// GetServer based on datacenterID and serverID
func (cp *APIClient) GetServer(ctx context.Context, datacenterID, serverID string) (sdkgo.Server, *sdkgo.APIResponse, error) {
	return cp.ComputeClient.ServersApi.DatacentersServersFindById(ctx, datacenterID, serverID).Depth(depthQueryParam).Execute()
}

// CreateServer based on Server properties
func (cp *APIClient) CreateServer(ctx context.Context, datacenterID string, server sdkgo.Server) (sdkgo.Server, *sdkgo.APIResponse, error) {
	return cp.ComputeClient.ServersApi.DatacentersServersPost(ctx, datacenterID).Server(server).Execute()
}

// UpdateServer based on datacenterID, serverID and Server properties
func (cp *APIClient) UpdateServer(ctx context.Context, datacenterID, serverID string, server sdkgo.ServerProperties) (sdkgo.Server, *sdkgo.APIResponse, error) {
	return cp.ComputeClient.ServersApi.DatacentersServersPatch(ctx, datacenterID, serverID).Server(server).Execute()
}

// DeleteServer based on datacenterID, serverID
func (cp *APIClient) DeleteServer(ctx context.Context, datacenterID, serverID string) (*sdkgo.APIResponse, error) {
	return cp.ComputeClient.ServersApi.DatacentersServersDelete(ctx, datacenterID, serverID).Execute()
}

// AttachVolume based on datacenterID, serverID, and volume
func (cp *APIClient) AttachVolume(ctx context.Context, datacenterID, serverID string, volume sdkgo.Volume) (sdkgo.Volume, *sdkgo.APIResponse, error) {
	return cp.ComputeClient.ServersApi.DatacentersServersVolumesPost(ctx, datacenterID, serverID).Volume(volume).Execute()
}

// DetachVolume based on datacenterID, serverID, and volume
func (cp *APIClient) DetachVolume(ctx context.Context, datacenterID, serverID, volumeID string) (*sdkgo.APIResponse, error) {
	return cp.ComputeClient.ServersApi.DatacentersServersVolumesDelete(ctx, datacenterID, serverID, volumeID).Execute()
}

// AttachCdrom based on datacenterID, serverID, and image
func (cp *APIClient) AttachCdrom(ctx context.Context, datacenterID, serverID string, cdrom sdkgo.Image) (sdkgo.Image, *sdkgo.APIResponse, error) {
	return cp.ComputeClient.ServersApi.DatacentersServersCdromsPost(ctx, datacenterID, serverID).Cdrom(cdrom).Execute()
}

// DetachCdrom based on datacenterID, serverID, and imageId
func (cp *APIClient) DetachCdrom(ctx context.Context, datacenterID, serverID, imageID string) (*sdkgo.APIResponse, error) {
	return cp.ComputeClient.ServersApi.DatacentersServersCdromsDelete(ctx, datacenterID, serverID, imageID).Execute()
}

// GetAPIClient gets the APIClient
func (cp *APIClient) GetAPIClient() *sdkgo.APIClient {
	return cp.ComputeClient
}

// GenerateCreateServerInput returns CreateServerRequest based on the CR spec
func GenerateCreateServerInput(cr *v1alpha1.Server) (*sdkgo.Server, error) {
	instanceCreateInput := sdkgo.Server{
		Properties: &sdkgo.ServerProperties{
			Name:  &cr.Spec.ForProvider.Name,
			Cores: &cr.Spec.ForProvider.Cores,
			Ram:   &cr.Spec.ForProvider.RAM,
		},
	}
	// Set CPUFamily only if it is specified into the Spec.
	// If not, the CPUFamily will be set corresponding with the datacenter's location
	if !utils.IsEmptyValue(reflect.ValueOf(cr.Spec.ForProvider.CPUFamily)) {
		instanceCreateInput.Properties.SetCpuFamily(cr.Spec.ForProvider.CPUFamily)
	}
	// Set AvailabilityZone only if it is specified into the Spec.
	// If not, the AvailabilityZone will be set to the default value by the API
	if !utils.IsEmptyValue(reflect.ValueOf(cr.Spec.ForProvider.AvailabilityZone)) {
		instanceCreateInput.Properties.SetAvailabilityZone(cr.Spec.ForProvider.AvailabilityZone)
	}
	if !utils.IsEmptyValue(reflect.ValueOf(cr.Spec.ForProvider.BootCdromID)) {
		instanceCreateInput.Properties.SetBootCdrom(sdkgo.ResourceReference{Id: &cr.Spec.ForProvider.BootCdromID})
	}
	if !utils.IsEmptyValue(reflect.ValueOf(cr.Spec.ForProvider.VolumeCfg.VolumeID)) {
		instanceCreateInput.Properties.SetBootVolume(sdkgo.ResourceReference{Id: &cr.Spec.ForProvider.VolumeCfg.VolumeID})
	}
	return &instanceCreateInput, nil
}

// GenerateUpdateServerInput returns PatchServerRequest based on the CR spec modifications
func GenerateUpdateServerInput(cr *v1alpha1.Server) (*sdkgo.ServerProperties, error) {
	instanceUpdateInput := sdkgo.ServerProperties{
		Name:  &cr.Spec.ForProvider.Name,
		Cores: &cr.Spec.ForProvider.Cores,
		Ram:   &cr.Spec.ForProvider.RAM,
	}
	if !utils.IsEmptyValue(reflect.ValueOf(cr.Spec.ForProvider.CPUFamily)) {
		instanceUpdateInput.SetCpuFamily(cr.Spec.ForProvider.CPUFamily)
	}
	if !utils.IsEmptyValue(reflect.ValueOf(cr.Spec.ForProvider.AvailabilityZone)) {
		instanceUpdateInput.SetAvailabilityZone(cr.Spec.ForProvider.AvailabilityZone)
	}
	if !utils.IsEmptyValue(reflect.ValueOf(cr.Spec.ForProvider.BootCdromID)) {
		instanceUpdateInput.SetBootCdrom(sdkgo.ResourceReference{Id: &cr.Spec.ForProvider.BootCdromID})
	}
	if !utils.IsEmptyValue(reflect.ValueOf(cr.Spec.ForProvider.VolumeCfg.VolumeID)) {
		instanceUpdateInput.SetBootVolume(sdkgo.ResourceReference{Id: &cr.Spec.ForProvider.VolumeCfg.VolumeID})
	}
	return &instanceUpdateInput, nil
}

// LateInitializer fills the empty fields in *v1alpha1.ServerParameters with
// the values seen in sdkgo.Server.
func LateInitializer(in *v1alpha1.ServerParameters, sg *sdkgo.Server) {
	if sg == nil {
		return
	}
	// Add Boot CD-ROM ID to the Spec, if it was updated via other tool (e.g. DCD)
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
	case server.Properties.Cores != nil && cr.Spec.ForProvider.Cores != *server.Properties.Cores:
		return false
	case server.Properties.Ram != nil && cr.Spec.ForProvider.RAM != *server.Properties.Ram:
		return false
	case server.Properties.CpuFamily != nil && cr.Spec.ForProvider.CPUFamily != *server.Properties.CpuFamily:
		return false
	case server.Properties.AvailabilityZone != nil && cr.Spec.ForProvider.AvailabilityZone != *server.Properties.AvailabilityZone:
		return false
	case server.Metadata.State != nil && *server.Metadata.State == "BUSY":
		return true
	case cr.Spec.ForProvider.VolumeCfg.VolumeID != cr.Status.AtProvider.VolumeID:
		return false
	default:
		return true
	}
}

// GenerateCreateCubeServerInput returns CreateServerRequest based on the CR spec
func GenerateCreateCubeServerInput(cr *v1alpha1.CubeServer, client *sdkgo.APIClient) (*sdkgo.Server, error) { // nolint:gocyclo
	// Cube Server needs a template ID in order to create a Volume
	var templateID string
	// Cube Server has a DAS Volume attached to it
	var volumeType = "DAS"

	// Find the corresponding template ID
	if utils.IsEmptyValue(reflect.ValueOf(cr.Spec.ForProvider.Template.TemplateID)) {
		if client != nil {
			templates, _, err := client.TemplatesApi.TemplatesGet(context.TODO()).Filter("name", cr.Spec.ForProvider.Template.Name).Depth(1).Execute()
			if err != nil {
				return nil, err
			}
			if items, ok := templates.GetItemsOk(); ok && items != nil {
				templatesItems := *items
				if len(templatesItems) > 0 {
					templateID = *templatesItems[0].Id
				} else {
					return nil, fmt.Errorf("error: no templates with the %v name found", cr.Spec.ForProvider.Template.Name)
				}
			}
		} else {
			return nil, fmt.Errorf("error: APIClient must not be nil")
		}
	} else {
		templateID = cr.Spec.ForProvider.Template.TemplateID
	}

	// Get DAS Volume Input
	dasVolumeInput := sdkgo.Volume{
		Properties: &sdkgo.VolumeProperties{
			Name: &cr.Spec.ForProvider.DasVolumeProperties.Name,
			Type: &volumeType,
			Bus:  &cr.Spec.ForProvider.DasVolumeProperties.Bus,
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

	// Create Server Input
	instanceCreateInput := sdkgo.Server{
		Properties: &sdkgo.ServerProperties{
			Name:             &cr.Spec.ForProvider.Name,
			TemplateUuid:     &templateID,
			AvailabilityZone: &cr.Spec.ForProvider.AvailabilityZone,
			CpuFamily:        &cr.Spec.ForProvider.CPUFamily,
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

// GenerateUpdateVolumeInput returns Volume based on the CR spec modifications
func GenerateUpdateVolumeInput(cr *v1alpha1.CubeServer) (*sdkgo.Volume, error) {
	instanceUpdateInput := sdkgo.VolumeProperties{
		Name: &cr.Spec.ForProvider.DasVolumeProperties.Name,
	}
	if !utils.IsEmptyValue(reflect.ValueOf(cr.Spec.ForProvider.DasVolumeProperties.Bus)) {
		instanceUpdateInput.SetBus(cr.Spec.ForProvider.DasVolumeProperties.Bus)
	}
	return &sdkgo.Volume{Properties: &instanceUpdateInput}, nil
}

// LateInitializerCube fills the empty fields in *v1alpha1.CubeServerProperties with
// the values seen in sdkgo.Server.
func LateInitializerCube(in *v1alpha1.CubeServerProperties, sg *sdkgo.Server) {
	if sg == nil {
		return
	}
	// Add Template ID to the Spec, if it was updated via other tool (e.g. DCD)
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
	case server.Metadata.State != nil && *server.Metadata.State == "BUSY":
		return true
	case server.Properties.Name != nil && *server.Properties.Name != cr.Spec.ForProvider.Name:
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
				if propertiesOk.Name != nil && *propertiesOk.Name != cr.Spec.ForProvider.DasVolumeProperties.Name {
					return false
				}
				if propertiesOk.Bus != nil && *propertiesOk.Bus != cr.Spec.ForProvider.DasVolumeProperties.Bus {
					return false
				}
			}
		}
	}
	return true
}
