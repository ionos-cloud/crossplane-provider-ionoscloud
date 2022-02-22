package server

import (
	"context"
	"strings"

	sdkgo "github.com/ionos-cloud/sdk-go/v6"

	"github.com/ionos-cloud/crossplane-provider-ionoscloud/apis/compute/v1alpha1"
	"github.com/ionos-cloud/crossplane-provider-ionoscloud/internal/clients"
)

var (
	serverEnterpriseType = "ENTERPRISE"
	serverCubeType       = "CUBE"
)

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

// GetAPIClient gets the APIClient
func (cp *APIClient) GetAPIClient() *sdkgo.APIClient {
	return cp.ComputeClient
}

// GenerateCreateServerInput returns CreateServerRequest based on the CR spec
func GenerateCreateServerInput(cr *v1alpha1.Server) (*sdkgo.Server, error) {
	instanceCreateInput := sdkgo.Server{
		Properties: &sdkgo.ServerProperties{
			Name:             &cr.Spec.ForProvider.Name,
			Cores:            &cr.Spec.ForProvider.Cores,
			Ram:              &cr.Spec.ForProvider.RAM,
			AvailabilityZone: &cr.Spec.ForProvider.AvailabilityZone,
			CpuFamily:        &cr.Spec.ForProvider.CPUFamily,
			Type:             &serverEnterpriseType,
		},
	}
	return &instanceCreateInput, nil
}

// GenerateUpdateServerInput returns PatchServerRequest based on the CR spec modifications
func GenerateUpdateServerInput(cr *v1alpha1.Server) (*sdkgo.ServerProperties, error) {
	instanceUpdateInput := sdkgo.ServerProperties{
		Name:             &cr.Spec.ForProvider.Name,
		Cores:            &cr.Spec.ForProvider.Cores,
		Ram:              &cr.Spec.ForProvider.RAM,
		AvailabilityZone: &cr.Spec.ForProvider.AvailabilityZone,
		CpuFamily:        &cr.Spec.ForProvider.CPUFamily,
	}
	return &instanceUpdateInput, nil
}

// IsServerUpToDate returns true if the Server is up-to-date or false if it does not
func IsServerUpToDate(cr *v1alpha1.Server, server sdkgo.Server) bool {
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

// GenerateCreateCubeServerInput returns CreateServerRequest based on the CR spec
func GenerateCreateCubeServerInput(cr *v1alpha1.CubeServer) (*sdkgo.Server, error) {
	instanceCreateInput := sdkgo.Server{
		Properties: &sdkgo.ServerProperties{
			Name:             &cr.Spec.ForProvider.Name,
			TemplateUuid:     &cr.Spec.ForProvider.TemplateID,
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
