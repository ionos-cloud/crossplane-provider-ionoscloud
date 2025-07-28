package volume_test

import (
	"context"
	"testing"

	"github.com/crossplane/crossplane-runtime/pkg/logging"
	"github.com/crossplane/crossplane-runtime/pkg/meta"
	sdkgo "github.com/ionos-cloud/sdk-go/v6"
	"github.com/stretchr/testify/assert"

	"github.com/ionos-cloud/crossplane-provider-ionoscloud/apis/compute/v1alpha1"
	"github.com/ionos-cloud/crossplane-provider-ionoscloud/internal/controller/compute/volume"
)

type mockVolumeClient struct {
	callCount int
}

func (m *mockVolumeClient) CheckDuplicateVolume(ctx context.Context, datacenterID, volumeName, storageType, availabilityZone, licenceType, image string) (*sdkgo.Volume, error) {
	panic("implement me")
}

func (m *mockVolumeClient) GetVolumeID(datacenter *sdkgo.Volume) (string, error) {
	panic("implement me")
}

func (m *mockVolumeClient) CreateVolume(ctx context.Context, datacenterID string, volume sdkgo.Volume) (sdkgo.Volume, *sdkgo.APIResponse, error) {
	panic("implement me")
}

func (m *mockVolumeClient) UpdateVolume(ctx context.Context, datacenterID, volumeID string, volume sdkgo.VolumeProperties) (sdkgo.Volume, *sdkgo.APIResponse, error) {
	panic("implement me")
}

func (m *mockVolumeClient) DeleteVolume(ctx context.Context, datacenterID, volumeID string) (*sdkgo.APIResponse, error) {
	panic("implement me")
}

func (m *mockVolumeClient) GetAPIClient() *sdkgo.APIClient {
	panic("implement me")
}

var (
	firstPciSLot  int32 = 5
	secondPciSlot int32 = 7
	name                = "volName"
	size                = float32(10.0)
	serverName          = "serverName"
	volID               = "vol-id"
)

func (m *mockVolumeClient) GetVolume(ctx context.Context, dcID, volID string) (sdkgo.Volume, *sdkgo.APIResponse, error) {
	m.callCount++
	pciSlot := firstPciSLot
	if m.callCount == 2 {
		pciSlot = secondPciSlot
	}

	return sdkgo.Volume{
		Properties: &sdkgo.VolumeProperties{
			Name:       &name,
			Size:       &size,
			PciSlot:    &pciSlot,
			BootServer: &serverName,
		},
	}, &sdkgo.APIResponse{}, nil
}

// Implement other methods as no-ops for the interface
func (m *mockVolumeClient) GetServerNameByID(ctx context.Context, dcID, serverID string) (string, error) {
	return serverName, nil
}

func TestObserve_PciSlotUpdate(t *testing.T) {
	cr := &v1alpha1.Volume{}
	cr.Spec.ForProvider.DatacenterCfg.DatacenterID = "dc"
	meta.SetExternalName(cr, volID)

	ext := &volume.ExternalVolume{
		Service: &mockVolumeClient{},
		Log:     logging.NewNopLogger(),
	}

	// First Observe: PciSlot should be "5" - firstPciSLot
	_, err := ext.Observe(context.Background(), cr)
	assert.NoError(t, err)
	assert.Equal(t, firstPciSLot, cr.Status.AtProvider.PCISlot)
	assert.Equal(t, name, cr.Status.AtProvider.Name)
	assert.Equal(t, size, cr.Status.AtProvider.Size)
	assert.Equal(t, serverName, cr.Status.AtProvider.ServerName)
	assert.Equal(t, volID, cr.Status.AtProvider.VolumeID)

	// Second Observe: PciSlot should be "7" - secondPciSlot
	_, err = ext.Observe(context.Background(), cr)
	assert.NoError(t, err)
	assert.Equal(t, secondPciSlot, cr.Status.AtProvider.PCISlot)
	assert.Equal(t, name, cr.Status.AtProvider.Name)
	assert.Equal(t, size, cr.Status.AtProvider.Size)
	assert.Equal(t, serverName, cr.Status.AtProvider.ServerName)
	assert.Equal(t, volID, cr.Status.AtProvider.VolumeID)
}
