//go:build server_mock

package server

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
	serverClient "github.com/ionos-cloud/crossplane-provider-ionoscloud/internal/clients/compute/server"
	"github.com/ionos-cloud/crossplane-provider-ionoscloud/internal/controller/compute/mocktest"
)

// ---------------------------------------------------------------------------
// Fake Server service
// ---------------------------------------------------------------------------

type storedServer struct {
	server       sdkgo.Server
	datacenterID string
}

type fakeServerService struct {
	mocktest.FakeServiceBase
	servers map[string]storedServer
	nextID  int

	createCalls []sdkgo.Server
	updateCalls []sdkgo.ServerProperties
}

func newFakeServerService(serverURL string) *fakeServerService {
	return &fakeServerService{
		FakeServiceBase: mocktest.NewFakeServiceBase(serverURL),
		servers:         make(map[string]storedServer),
		nextID:          1,
	}
}

func (f *fakeServerService) key(datacenterID, serverID string) string {
	return datacenterID + "/" + serverID
}

func (f *fakeServerService) CheckDuplicateServer(_ context.Context, datacenterID, serverName, cpuFamily string) (*sdkgo.Server, error) {
	f.Mu.Lock()
	defer f.Mu.Unlock()
	for _, ss := range f.servers {
		if ss.datacenterID == datacenterID && ss.server.Properties != nil &&
			ss.server.Properties.Name != nil && *ss.server.Properties.Name == serverName {
			s := ss.server
			return &s, nil
		}
	}
	return nil, nil
}

func (f *fakeServerService) CheckDuplicateCubeServer(_ context.Context, _, _, _ string) (string, error) {
	return "", nil
}

func (f *fakeServerService) GetServerID(s *sdkgo.Server) (string, error) {
	if s != nil {
		if id, ok := s.GetIdOk(); ok && id != nil {
			return *id, nil
		}
		return "", fmt.Errorf("error: getting server id")
	}
	return "", nil
}

func (f *fakeServerService) GetServer(_ context.Context, datacenterID, serverID string) (sdkgo.Server, *sdkgo.APIResponse, error) {
	f.Mu.Lock()
	defer f.Mu.Unlock()
	if f.GetErr != nil {
		return sdkgo.Server{}, mocktest.ErrorResponse(http.StatusInternalServerError), f.GetErr
	}
	ss, ok := f.servers[f.key(datacenterID, serverID)]
	if !ok {
		return sdkgo.Server{}, mocktest.ErrorResponse(http.StatusNotFound), fmt.Errorf("server %s not found", serverID)
	}
	return ss.server, mocktest.OKResponse(), nil
}

func (f *fakeServerService) CreateServer(_ context.Context, datacenterID string, s sdkgo.Server) (sdkgo.Server, *sdkgo.APIResponse, error) {
	f.Mu.Lock()
	defer f.Mu.Unlock()
	if f.CreateErr != nil {
		return sdkgo.Server{}, mocktest.ErrorResponse(http.StatusInternalServerError), f.CreateErr
	}

	f.createCalls = append(f.createCalls, s)
	srvID := strconv.Itoa(f.nextID)
	f.nextID++

	props := &sdkgo.ServerProperties{}
	if s.Properties != nil {
		if s.Properties.Name != nil {
			props.Name = s.Properties.Name
		}
		if s.Properties.Cores != nil {
			props.Cores = s.Properties.Cores
		}
		if s.Properties.Ram != nil {
			props.Ram = s.Properties.Ram
		}
		if s.Properties.CpuFamily != nil {
			props.CpuFamily = s.Properties.CpuFamily
		}
	}
	newSrv := sdkgo.Server{
		Id:         &srvID,
		Properties: props,
		Metadata:   &sdkgo.DatacenterElementMetadata{State: ptr.To("AVAILABLE")},
	}
	f.servers[f.key(datacenterID, srvID)] = storedServer{server: newSrv, datacenterID: datacenterID}

	return newSrv, f.AcceptedResponse("create", srvID), nil
}

func (f *fakeServerService) UpdateServer(_ context.Context, datacenterID, serverID string, props sdkgo.ServerProperties) (sdkgo.Server, *sdkgo.APIResponse, error) {
	f.Mu.Lock()
	defer f.Mu.Unlock()
	if f.UpdateErr != nil {
		return sdkgo.Server{}, mocktest.ErrorResponse(http.StatusInternalServerError), f.UpdateErr
	}

	f.updateCalls = append(f.updateCalls, props)
	ss, ok := f.servers[f.key(datacenterID, serverID)]
	if !ok {
		return sdkgo.Server{}, mocktest.ErrorResponse(http.StatusNotFound), fmt.Errorf("server %s not found", serverID)
	}

	if props.Name != nil {
		ss.server.Properties.Name = props.Name
	}
	if props.Cores != nil {
		ss.server.Properties.Cores = props.Cores
	}
	if props.Ram != nil {
		ss.server.Properties.Ram = props.Ram
	}
	f.servers[f.key(datacenterID, serverID)] = ss

	return ss.server, f.AcceptedResponse("update", serverID), nil
}

func (f *fakeServerService) DeleteServer(_ context.Context, datacenterID, serverID string) (*sdkgo.APIResponse, error) {
	f.Mu.Lock()
	defer f.Mu.Unlock()
	if f.DeleteErr != nil {
		return mocktest.ErrorResponse(http.StatusNotFound), f.DeleteErr
	}

	f.DeleteCalls = append(f.DeleteCalls, serverID)
	delete(f.servers, f.key(datacenterID, serverID))

	return f.AcceptedResponse("delete", serverID), nil
}

func (f *fakeServerService) IsVolumeAttached(_ context.Context, _, _, _ string) (bool, error) {
	return false, nil
}

func (f *fakeServerService) AttachVolume(_ context.Context, _, _ string, vol sdkgo.Volume) (sdkgo.Volume, *sdkgo.APIResponse, error) {
	return vol, f.AcceptedResponse("attach-vol", "req"), nil
}

func (f *fakeServerService) DetachVolume(_ context.Context, _, _, _ string) (*sdkgo.APIResponse, error) {
	return f.AcceptedResponse("detach-vol", "req"), nil
}

func (f *fakeServerService) AttachCdrom(_ context.Context, _, _ string, img sdkgo.Image) (sdkgo.Image, *sdkgo.APIResponse, error) {
	return img, f.AcceptedResponse("attach-cdrom", "req"), nil
}

func (f *fakeServerService) DetachCdrom(_ context.Context, _, _, _ string) (*sdkgo.APIResponse, error) {
	return f.AcceptedResponse("detach-cdrom", "req"), nil
}

func (f *fakeServerService) StartServer(_ context.Context, _, _ string) (*sdkgo.APIResponse, error) {
	return f.AcceptedResponse("start-srv", "req"), nil
}

func (f *fakeServerService) StopServer(_ context.Context, _, _ string) (*sdkgo.APIResponse, error) {
	return f.AcceptedResponse("stop-srv", "req"), nil
}

func (f *fakeServerService) SuspendServer(_ context.Context, _, _ string) (*sdkgo.APIResponse, error) {
	return f.AcceptedResponse("suspend-srv", "req"), nil
}

func (f *fakeServerService) ResumeServer(_ context.Context, _, _ string) (*sdkgo.APIResponse, error) {
	return f.AcceptedResponse("resume-srv", "req"), nil
}

// Test-only helpers

func (f *fakeServerService) storeServer(datacenterID, serverID string, s sdkgo.Server) {
	f.Mu.Lock()
	defer f.Mu.Unlock()
	f.servers[f.key(datacenterID, serverID)] = storedServer{server: s, datacenterID: datacenterID}
}

func (f *fakeServerService) removeServer(datacenterID, serverID string) {
	f.Mu.Lock()
	defer f.Mu.Unlock()
	delete(f.servers, f.key(datacenterID, serverID))
}

// ---------------------------------------------------------------------------
// Test connector
// ---------------------------------------------------------------------------

type testConnectorServer struct {
	service serverClient.Client
	log     logging.Logger
}

func (c *testConnectorServer) Connect(_ context.Context, _ resource.Managed) (managed.ExternalClient, error) {
	return &externalServer{service: c.service, log: c.log}, nil
}

// ---------------------------------------------------------------------------
// Test globals
// ---------------------------------------------------------------------------

var (
	k8sClient  client.Client
	fakeSvc    *fakeServerService
	testServer *mocktest.TestHTTPServer
	harness    *mocktest.EnvTestHarness
)

func TestServerController_Mock(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Server Controller Mock Suite")
}

var _ = BeforeSuite(func() {
	testServer = mocktest.NewTestHTTPServer()
	testServer.SetMode(mocktest.StatusModeDone)
	fakeSvc = newFakeServerService(testServer.URL())

	harness = mocktest.SetupEnvTest(mocktest.ControllerSetup{
		GroupKind:        v1alpha1.ServerGroupKind,
		GroupVersionKind: v1alpha1.ServerGroupVersionKind,
		ManagedResource:  &v1alpha1.Server{},
		ManagedList:      &v1alpha1.ServerList{},
		Connector: &testConnectorServer{
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

func newServerCR(name string) *v1alpha1.Server {
	return &v1alpha1.Server{
		ObjectMeta: metav1.ObjectMeta{Name: name},
		Spec: v1alpha1.ServerSpec{
			ResourceSpec: xpv1.ResourceSpec{
				DeletionPolicy:     xpv1.DeletionDelete,
				ManagementPolicies: xpv1.ManagementPolicies{xpv1.ManagementActionAll},
			},
			ForProvider: v1alpha1.ServerParameters{
				DatacenterCfg: v1alpha1.DatacenterConfig{DatacenterID: mocktest.TestDatacenterID},
				Name:          name,
				Cores:         2,
				RAM:           2048,
			},
		},
	}
}

func getServerCR(ctx context.Context, name string) (*v1alpha1.Server, error) {
	cr := &v1alpha1.Server{}
	err := k8sClient.Get(ctx, types.NamespacedName{Name: name}, cr)
	return cr, err
}

// ---------------------------------------------------------------------------
// Test scenarios
// ---------------------------------------------------------------------------

var _ = Describe("Server Controller E2E Tests", func() {
	mocktest.RunStandardScenarios(mocktest.ScenarioConfig{
		ResourceName: "Server",
		CRPrefix:     "test-srv",
		K8sClient:    func() client.Client { return k8sClient },
		TestServer:   func() *mocktest.TestHTTPServer { return testServer },
		CreateCR: func(name string) client.Object {
			return newServerCR(name)
		},
		CreateCRWithAnnotation: func(name, externalName, annotationValue string) client.Object {
			cr := newServerCR(name)
			cr.Annotations = map[string]string{
				"ionos.cloud/post-request-id": annotationValue,
			}
			if externalName != "" {
				meta.SetExternalName(cr, externalName)
			}
			return cr
		},
		GetCR: func(ctx context.Context, name string) (client.Object, error) {
			return getServerCR(ctx, name)
		},
		GetState: func(obj client.Object) string {
			return obj.(*v1alpha1.Server).Status.AtProvider.State
		},
		GetResourceID: func(obj client.Object) string {
			return obj.(*v1alpha1.Server).Status.AtProvider.ServerID
		},
		MutateForUpdate: func(obj client.Object) {
			cr := obj.(*v1alpha1.Server)
			cr.Spec.ForProvider.Name = "updated-server"
			cr.Spec.ForProvider.Cores = 4
			cr.Spec.ForProvider.RAM = 4096
		},
		StoreResource: func(externalID string) {
			fakeSvc.storeServer(mocktest.TestDatacenterID, externalID, sdkgo.Server{
				Id: ptr.To(externalID),
				Properties: &sdkgo.ServerProperties{
					Name:  ptr.To("isreqdone-server"),
					Cores: ptr.To(int32(2)),
					Ram:   ptr.To(int32(2048)),
				},
				Metadata: &sdkgo.DatacenterElementMetadata{State: ptr.To("AVAILABLE")},
			})
		},
		RemoveResource: func(externalID string) {
			fakeSvc.removeServer(mocktest.TestDatacenterID, externalID)
		},
		SetError:       func(method string, err error) { fakeSvc.SetError(method, err) },
		ClearErrors:    func() { fakeSvc.ClearErrors() },
		GetDeleteCalls: func() []string { return fakeSvc.GetDeleteCalls() },
	})
})
