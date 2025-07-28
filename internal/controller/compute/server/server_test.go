package server_test

import (
	"context"
	"net/http"
	"testing"

	"github.com/crossplane/crossplane-runtime/pkg/logging"
	"github.com/crossplane/crossplane-runtime/pkg/meta"
	sdkgo "github.com/ionos-cloud/sdk-go/v6"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"k8s.io/utils/ptr"

	"github.com/ionos-cloud/crossplane-provider-ionoscloud/apis/compute/v1alpha1"
	"github.com/ionos-cloud/crossplane-provider-ionoscloud/internal/controller/compute/server"
)

type mockServerClient struct {
	createdServerID  string
	attachedVolumeID string
	observedVolumeID string
}

func (m *mockServerClient) GetAPIClient() *sdkgo.APIClient {
	return nil
}

func (m *mockServerClient) WaitForRequest(ctx context.Context, response *sdkgo.APIResponse) error {
	return nil
}

func (m *mockServerClient) CreateServer(ctx context.Context, dcID string, srv sdkgo.Server) (sdkgo.Server, *sdkgo.APIResponse, error) {
	m.createdServerID = "srv-1"
	return sdkgo.Server{Id: &m.createdServerID}, &sdkgo.APIResponse{
		Response: &http.Response{
			StatusCode: http.StatusOK,
			Status:     "200 OK",
		},
	}, nil
}

func (m *mockServerClient) GetServer(ctx context.Context, dcID, serverID string) (sdkgo.Server, *sdkgo.APIResponse, error) {
	return sdkgo.Server{
			Id: &m.createdServerID,
			Properties: &sdkgo.ServerProperties{
				Name:       ptr.To("test-server"),
				BootVolume: &sdkgo.ResourceReference{Id: &m.observedVolumeID},
			},
		}, &sdkgo.APIResponse{
			Response: &http.Response{
				StatusCode: http.StatusOK,
				Status:     "200 OK",
			},
		}, nil
}
func (m *mockServerClient) AttachVolume(ctx context.Context, dcID, serverID string, vol sdkgo.Volume) (sdkgo.Volume, *sdkgo.APIResponse, error) {
	m.attachedVolumeID = *vol.Id
	m.observedVolumeID = m.attachedVolumeID
	return sdkgo.Volume{Id: &m.attachedVolumeID}, &sdkgo.APIResponse{
		Response: &http.Response{
			StatusCode: http.StatusOK,
			Status:     "200 OK",
		},
	}, nil
}
func (m *mockServerClient) UpdateServer(ctx context.Context, dcID, serverID string, props sdkgo.ServerProperties) (sdkgo.Server, *sdkgo.APIResponse, error) {
	return sdkgo.Server{}, &sdkgo.APIResponse{}, nil
}
func (m *mockServerClient) DeleteServer(ctx context.Context, dcID, serverID string) (*sdkgo.APIResponse, error) {
	return &sdkgo.APIResponse{}, nil
}
func (m *mockServerClient) IsVolumeAttached(ctx context.Context, dcID, serverID, volumeID string) (bool, error) {
	return true, nil
}
func (m *mockServerClient) DetachVolume(ctx context.Context, dcID, serverID, volumeID string) (*sdkgo.APIResponse, error) {
	return &sdkgo.APIResponse{}, nil
}
func (m *mockServerClient) AttachCdrom(ctx context.Context, dcID, serverID string, cdrom sdkgo.Image) (sdkgo.Image, *sdkgo.APIResponse, error) {
	return sdkgo.Image{}, &sdkgo.APIResponse{}, nil
}
func (m *mockServerClient) DetachCdrom(ctx context.Context, dcID, serverID, imageID string) (*sdkgo.APIResponse, error) {
	return &sdkgo.APIResponse{}, nil
}

func (m *mockServerClient) CheckDuplicateServer(ctx context.Context, dcID, name, cpuFamily string) (*sdkgo.Server, error) {
	return nil, nil
}
func (m *mockServerClient) CheckDuplicateCubeServer(ctx context.Context, dcID, name, templateID string) (string, error) {
	return "", nil
}
func (m *mockServerClient) GetServerID(srv *sdkgo.Server) (string, error) {
	return m.createdServerID, nil
}

func TestServer_CreateObserveUpdateVolumeID(t *testing.T) {
	client := &mockServerClient{}
	ext := &server.ExternalServer{
		Service: client,
		Log:     logging.NewNopLogger(),
	}

	cr := &v1alpha1.Server{}
	cr.Spec.ForProvider.Name = "test-server"
	cr.Spec.ForProvider.DatacenterCfg.DatacenterID = "dc-1"
	cr.Spec.ForProvider.VolumeCfg.VolumeID = "vol-1"
	meta.SetExternalName(cr, "")

	// Create
	_, err := ext.Create(context.Background(), cr)
	assert.NoError(t, err)
	assert.Equal(t, "srv-1", cr.Status.AtProvider.ServerID)

	meta.SetExternalName(cr, "srv-1")
	client.observedVolumeID = "vol-1"

	// First Observe
	_, err = ext.Observe(context.Background(), cr)
	require.NoError(t, err)
	assert.Equal(t, "vol-1", cr.Status.AtProvider.VolumeID)

	// Update VolumeID
	cr.Spec.ForProvider.VolumeCfg.VolumeID = "vol-2"
	client.observedVolumeID = "vol-2"

	_, err = ext.Update(context.Background(), cr)
	require.NoError(t, err)

	// Second Observe
	_, err = ext.Observe(context.Background(), cr)
	require.NoError(t, err)
	assert.Equal(t, "vol-2", cr.Status.AtProvider.VolumeID)
}
