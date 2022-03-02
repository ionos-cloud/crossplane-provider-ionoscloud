package server

import (
	"context"
	"fmt"
	"reflect"
	"strings"

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
	return cp.ComputeClient.ServersApi.DatacentersServersFindById(ctx, datacenterID, serverID).Execute()
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

// IsServerUpToDate returns true if the Server is up-to-date or false if it does not
func IsServerUpToDate(cr *v1alpha1.Server, server sdkgo.Server) bool { // nolint:gocyclo
	switch {
	case cr == nil && server.Properties == nil:
		return true
	case cr == nil && server.Properties != nil:
		return false
	case cr != nil && server.Properties == nil:
		return false
	case cr.Spec.ForProvider.Name != *server.Properties.Name:
		return false
	case *server.Metadata.State == "BUSY":
		return true
	case cr.Spec.ForProvider.VolumeCfg.VolumeID != cr.Status.AtProvider.VolumeID:
		return false
	default:
		return true
	}
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
			in.BootCdromID = *bootCdromOk.Id
		}
	}
}

// GenerateCreateCubeServerInput returns CreateServerRequest based on the CR spec
func GenerateCreateCubeServerInput(cr *v1alpha1.CubeServer, client *sdkgo.APIClient) (*sdkgo.Server, error) {
	// TODO: to be updated with DAS Volume Properties
	var templateID string
	if cr.Spec.ForProvider.Template.ID == "" {
		if client != nil {
			templates, _, err := client.TemplatesApi.TemplatesGet(context.TODO()).
				Filter("name", cr.Spec.ForProvider.Template.Name).Depth(1).Execute()
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
		templateID = cr.Spec.ForProvider.Template.ID
	}
	instanceCreateInput := sdkgo.Server{
		Properties: &sdkgo.ServerProperties{
			Name:             &cr.Spec.ForProvider.Name,
			TemplateUuid:     &templateID,
			AvailabilityZone: &cr.Spec.ForProvider.AvailabilityZone,
			CpuFamily:        &cr.Spec.ForProvider.CPUFamily,
			Type:             &serverCubeType,
		},
	}
	return &instanceCreateInput, nil
}

// GenerateUpdateCubeServerInput returns PatchServerRequest based on the CR spec modifications
func GenerateUpdateCubeServerInput(cr *v1alpha1.CubeServer) (*sdkgo.ServerProperties, error) {
	instanceUpdateInput := sdkgo.ServerProperties{
		Name:             &cr.Spec.ForProvider.Name,
		AvailabilityZone: &cr.Spec.ForProvider.AvailabilityZone,
		CpuFamily:        &cr.Spec.ForProvider.CPUFamily,
	}
	return &instanceUpdateInput, nil
}

// IsCubeServerUpToDate returns true if the Server is up-to-date or false if it does not
func IsCubeServerUpToDate(cr *v1alpha1.CubeServer, server sdkgo.Server) bool {
	switch {
	case cr == nil && server.Properties == nil:
		return true
	case cr == nil && server.Properties != nil:
		return false
	case cr != nil && server.Properties == nil:
		return false
	}
	if *server.Metadata.State == "BUSY" {
		return true
	}
	if strings.Compare(cr.Spec.ForProvider.Name, *server.Properties.Name) != 0 {
		return false
	}
	return true
}
