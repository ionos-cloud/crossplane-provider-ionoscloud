//go:build volume_mock

package volume

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"testing"

	xpv1 "github.com/crossplane/crossplane-runtime/apis/common/v1"
	"github.com/crossplane/crossplane-runtime/pkg/logging"
	"github.com/crossplane/crossplane-runtime/pkg/meta"
	"github.com/crossplane/crossplane-runtime/pkg/reconciler/managed"
	"github.com/crossplane/crossplane-runtime/pkg/resource"
	sdkgo "github.com/ionos-cloud/sdk-go/v6"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/ionos-cloud/crossplane-provider-ionoscloud/apis/compute/v1alpha1"
	volumeClient "github.com/ionos-cloud/crossplane-provider-ionoscloud/internal/clients/compute/volume"
	"github.com/ionos-cloud/crossplane-provider-ionoscloud/internal/controller/compute/mocktest"
)

// ---------------------------------------------------------------------------
// Fake Volume service
// ---------------------------------------------------------------------------

type storedVolume struct {
	volume       sdkgo.Volume
	datacenterID string
}

type fakeVolumeService struct {
	mocktest.FakeServiceBase
	volumes map[string]storedVolume
	nextID  int

	createCalls []sdkgo.Volume
	updateCalls []sdkgo.VolumeProperties
}

func newFakeVolumeService(serverURL string) *fakeVolumeService {
	return &fakeVolumeService{
		FakeServiceBase: mocktest.NewFakeServiceBase(serverURL),
		volumes:         make(map[string]storedVolume),
		nextID:          1,
	}
}

func (f *fakeVolumeService) key(dcID, volID string) string { return dcID + "/" + volID }

func (f *fakeVolumeService) CheckDuplicateVolume(_ context.Context, datacenterID, volumeName, _, _, _, _ string) (*sdkgo.Volume, error) {
	f.Mu.Lock()
	defer f.Mu.Unlock()
	for _, sv := range f.volumes {
		if sv.datacenterID == datacenterID && sv.volume.Properties != nil &&
			sv.volume.Properties.Name != nil && *sv.volume.Properties.Name == volumeName {
			v := sv.volume
			return &v, nil
		}
	}
	return nil, nil
}

func (f *fakeVolumeService) GetVolumeID(v *sdkgo.Volume) (string, error) {
	if v != nil {
		if id, ok := v.GetIdOk(); ok && id != nil {
			return *id, nil
		}
		return "", fmt.Errorf("error: getting volume id")
	}
	return "", nil
}

func (f *fakeVolumeService) GetVolume(_ context.Context, datacenterID, volumeID string) (sdkgo.Volume, *sdkgo.APIResponse, error) {
	f.Mu.Lock()
	defer f.Mu.Unlock()
	if f.GetErr != nil {
		return sdkgo.Volume{}, mocktest.ErrorResponse(http.StatusInternalServerError), f.GetErr
	}
	sv, ok := f.volumes[f.key(datacenterID, volumeID)]
	if !ok {
		return sdkgo.Volume{}, mocktest.ErrorResponse(http.StatusNotFound), fmt.Errorf("volume %s not found", volumeID)
	}
	return sv.volume, mocktest.OKResponse(), nil
}

func (f *fakeVolumeService) CreateVolume(_ context.Context, datacenterID string, v sdkgo.Volume) (sdkgo.Volume, *sdkgo.APIResponse, error) {
	f.Mu.Lock()
	defer f.Mu.Unlock()
	if f.CreateErr != nil {
		return sdkgo.Volume{}, mocktest.ErrorResponse(http.StatusInternalServerError), f.CreateErr
	}
	f.createCalls = append(f.createCalls, v)
	volID := strconv.Itoa(f.nextID)
	f.nextID++

	props := &sdkgo.VolumeProperties{}
	if v.Properties != nil {
		if v.Properties.Name != nil {
			props.Name = v.Properties.Name
		}
		if v.Properties.Type != nil {
			props.Type = v.Properties.Type
		}
		if v.Properties.Size != nil {
			props.Size = v.Properties.Size
		}
		if v.Properties.LicenceType != nil {
			props.LicenceType = v.Properties.LicenceType
		}
	}
	newVol := sdkgo.Volume{
		Id:         &volID,
		Properties: props,
		Metadata:   &sdkgo.DatacenterElementMetadata{State: ptr.To("AVAILABLE")},
	}
	f.volumes[f.key(datacenterID, volID)] = storedVolume{volume: newVol, datacenterID: datacenterID}

	return newVol, f.AcceptedResponse("create", volID), nil
}

func (f *fakeVolumeService) UpdateVolume(_ context.Context, datacenterID, volumeID string, props sdkgo.VolumeProperties) (sdkgo.Volume, *sdkgo.APIResponse, error) {
	f.Mu.Lock()
	defer f.Mu.Unlock()
	if f.UpdateErr != nil {
		return sdkgo.Volume{}, mocktest.ErrorResponse(http.StatusInternalServerError), f.UpdateErr
	}
	f.updateCalls = append(f.updateCalls, props)
	sv, ok := f.volumes[f.key(datacenterID, volumeID)]
	if !ok {
		return sdkgo.Volume{}, mocktest.ErrorResponse(http.StatusNotFound), fmt.Errorf("volume %s not found", volumeID)
	}
	if props.Name != nil {
		sv.volume.Properties.Name = props.Name
	}
	if props.Size != nil {
		sv.volume.Properties.Size = props.Size
	}
	f.volumes[f.key(datacenterID, volumeID)] = sv

	return sv.volume, f.AcceptedResponse("update", volumeID), nil
}

func (f *fakeVolumeService) DeleteVolume(_ context.Context, datacenterID, volumeID string) (*sdkgo.APIResponse, error) {
	f.Mu.Lock()
	defer f.Mu.Unlock()
	if f.DeleteErr != nil {
		return mocktest.ErrorResponse(http.StatusNotFound), f.DeleteErr
	}
	f.DeleteCalls = append(f.DeleteCalls, volumeID)
	delete(f.volumes, f.key(datacenterID, volumeID))

	return f.AcceptedResponse("delete", volumeID), nil
}

func (f *fakeVolumeService) GetServerNameByID(_ context.Context, _, _ string) (string, error) {
	return "", nil
}

// Test-only helpers

func (f *fakeVolumeService) storeVolume(dcID, volID string, v sdkgo.Volume) {
	f.Mu.Lock()
	defer f.Mu.Unlock()
	f.volumes[f.key(dcID, volID)] = storedVolume{volume: v, datacenterID: dcID}
}

func (f *fakeVolumeService) removeVolume(dcID, volID string) {
	f.Mu.Lock()
	defer f.Mu.Unlock()
	delete(f.volumes, f.key(dcID, volID))
}

// ---------------------------------------------------------------------------
// Test connector
// ---------------------------------------------------------------------------

type testConnectorVolume struct {
	service volumeClient.Client
	log     logging.Logger
}

func (c *testConnectorVolume) Connect(_ context.Context, _ resource.Managed) (managed.ExternalClient, error) {
	return &externalVolume{service: c.service, log: c.log}, nil
}

// ---------------------------------------------------------------------------
// Test globals
// ---------------------------------------------------------------------------

var (
	k8sClient  client.Client
	fakeSvc    *fakeVolumeService
	testServer *mocktest.TestHTTPServer
	harness    *mocktest.EnvTestHarness
)

func TestVolumeController_Mock(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Volume Controller Mock Suite")
}

var _ = BeforeSuite(func() {
	testServer = mocktest.NewTestHTTPServer()
	testServer.SetMode(mocktest.StatusModeDone)
	fakeSvc = newFakeVolumeService(testServer.URL())

	harness = mocktest.SetupEnvTest(mocktest.ControllerSetup{
		GroupKind:        v1alpha1.VolumeGroupKind,
		GroupVersionKind: v1alpha1.VolumeGroupVersionKind,
		ManagedResource:  &v1alpha1.Volume{},
		ManagedList:      &v1alpha1.VolumeList{},
		Connector: &testConnectorVolume{
			service: fakeSvc,
			log:     logging.NewLogrLogger(mocktest.Logger),
		},
	})
	k8sClient = harness.K8sClient
})

var _ = AfterSuite(func() {
	mocktest.TeardownEnvTest(harness, testServer)
})

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

func newVolumeCR(name string) *v1alpha1.Volume {
	return &v1alpha1.Volume{
		ObjectMeta: metav1.ObjectMeta{Name: name},
		Spec: v1alpha1.VolumeSpec{
			ResourceSpec: xpv1.ResourceSpec{
				DeletionPolicy:     xpv1.DeletionDelete,
				ManagementPolicies: xpv1.ManagementPolicies{xpv1.ManagementActionAll},
			},
			ForProvider: v1alpha1.VolumeParameters{
				DatacenterCfg: v1alpha1.DatacenterConfig{DatacenterID: mocktest.TestDatacenterID},
				Name:          name,
				Type:          "HDD",
				Size:          10,
				LicenceType:   "LINUX",
			},
		},
	}
}

func getVolumeCR(ctx context.Context, name string) (*v1alpha1.Volume, error) {
	cr := &v1alpha1.Volume{}
	return cr, k8sClient.Get(ctx, types.NamespacedName{Name: name}, cr)
}

// ---------------------------------------------------------------------------
// Test scenarios
// ---------------------------------------------------------------------------

var _ = Describe("Volume Controller E2E Tests", func() {
	mocktest.RunStandardScenarios(mocktest.ScenarioConfig{
		ResourceName: "Volume",
		CRPrefix:     "test-vol",
		K8sClient:    func() client.Client { return k8sClient },
		TestServer:   func() *mocktest.TestHTTPServer { return testServer },
		CreateCR: func(name string) client.Object {
			return newVolumeCR(name)
		},
		CreateCRWithAnnotation: func(name, externalName, annotationValue string) client.Object {
			cr := newVolumeCR(name)
			cr.Annotations = map[string]string{
				"ionos.cloud/post-request-id": annotationValue,
			}
			if externalName != "" {
				meta.SetExternalName(cr, externalName)
			}
			return cr
		},
		GetCR: func(ctx context.Context, name string) (client.Object, error) {
			return getVolumeCR(ctx, name)
		},
		GetState: func(obj client.Object) string {
			return obj.(*v1alpha1.Volume).Status.AtProvider.State
		},
		GetResourceID: func(obj client.Object) string {
			return obj.(*v1alpha1.Volume).Status.AtProvider.VolumeID
		},
		MutateForUpdate: func(obj client.Object) {
			cr := obj.(*v1alpha1.Volume)
			cr.Spec.ForProvider.Name = "updated-volume"
			cr.Spec.ForProvider.Size = 20
		},
		StoreResource: func(externalID string) {
			fakeSvc.storeVolume(mocktest.TestDatacenterID, externalID, sdkgo.Volume{
				Id:         ptr.To(externalID),
				Properties: &sdkgo.VolumeProperties{Name: ptr.To("isreqdone-volume"), Type: ptr.To("HDD"), Size: ptr.To(float32(10))},
				Metadata:   &sdkgo.DatacenterElementMetadata{State: ptr.To("AVAILABLE")},
			})
		},
		RemoveResource: func(externalID string) {
			fakeSvc.removeVolume(mocktest.TestDatacenterID, externalID)
		},
		SetError:       func(method string, err error) { fakeSvc.SetError(method, err) },
		ClearErrors:    func() { fakeSvc.ClearErrors() },
		GetDeleteCalls: func() []string { return fakeSvc.GetDeleteCalls() },
	})
})
