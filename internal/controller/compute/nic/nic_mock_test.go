//go:build nic_mock

package nic

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
	ipblockClient "github.com/ionos-cloud/crossplane-provider-ionoscloud/internal/clients/compute/ipblock"
	nicClient "github.com/ionos-cloud/crossplane-provider-ionoscloud/internal/clients/compute/nic"
	"github.com/ionos-cloud/crossplane-provider-ionoscloud/internal/controller/compute/mocktest"
)

// ---------------------------------------------------------------------------
// Fake NIC service
// ---------------------------------------------------------------------------

type storedNic struct {
	nic          sdkgo.Nic
	datacenterID string
	serverID     string
}

type fakeNicService struct {
	mocktest.FakeServiceBase
	nics   map[string]storedNic
	nextID int
}

func newFakeNicService(serverURL string) *fakeNicService {
	return &fakeNicService{
		FakeServiceBase: mocktest.NewFakeServiceBase(serverURL),
		nics:            make(map[string]storedNic),
		nextID:          1,
	}
}

func (f *fakeNicService) key(dcID, srvID, nicID string) string {
	return dcID + "/" + srvID + "/" + nicID
}

func (f *fakeNicService) CheckDuplicateNic(_ context.Context, datacenterID, serverID, nicName string) (*sdkgo.Nic, error) {
	f.Mu.Lock()
	defer f.Mu.Unlock()
	for _, sn := range f.nics {
		if sn.datacenterID == datacenterID && sn.serverID == serverID && sn.nic.Properties != nil &&
			sn.nic.Properties.Name != nil && *sn.nic.Properties.Name == nicName {
			n := sn.nic
			return &n, nil
		}
	}
	return nil, nil
}

func (f *fakeNicService) GetNicID(n *sdkgo.Nic) (string, error) {
	if n != nil {
		if id, ok := n.GetIdOk(); ok && id != nil {
			return *id, nil
		}
		return "", fmt.Errorf("error: getting nic id")
	}
	return "", nil
}

func (f *fakeNicService) GetNic(_ context.Context, datacenterID, serverID, nicID string) (sdkgo.Nic, *sdkgo.APIResponse, error) {
	f.Mu.Lock()
	defer f.Mu.Unlock()
	if f.GetErr != nil {
		return sdkgo.Nic{}, mocktest.ErrorResponse(http.StatusInternalServerError), f.GetErr
	}
	sn, ok := f.nics[f.key(datacenterID, serverID, nicID)]
	if !ok {
		return sdkgo.Nic{}, mocktest.ErrorResponse(http.StatusNotFound), fmt.Errorf("nic %s not found", nicID)
	}
	return sn.nic, mocktest.OKResponse(), nil
}

func (f *fakeNicService) CreateNic(_ context.Context, datacenterID, serverID string, n sdkgo.Nic) (sdkgo.Nic, *sdkgo.APIResponse, error) {
	f.Mu.Lock()
	defer f.Mu.Unlock()
	if f.CreateErr != nil {
		return sdkgo.Nic{}, mocktest.ErrorResponse(http.StatusInternalServerError), f.CreateErr
	}
	nicID := strconv.Itoa(f.nextID)
	f.nextID++

	props := &sdkgo.NicProperties{}
	if n.Properties != nil {
		if n.Properties.Name != nil {
			props.Name = n.Properties.Name
		}
		if n.Properties.Dhcp != nil {
			props.Dhcp = n.Properties.Dhcp
		}
		if n.Properties.Lan != nil {
			props.Lan = n.Properties.Lan
		}
	}
	newNic := sdkgo.Nic{
		Id:         &nicID,
		Properties: props,
		Metadata:   &sdkgo.DatacenterElementMetadata{State: ptr.To("AVAILABLE")},
	}
	f.nics[f.key(datacenterID, serverID, nicID)] = storedNic{nic: newNic, datacenterID: datacenterID, serverID: serverID}

	return newNic, f.AcceptedResponse("create", nicID), nil
}

func (f *fakeNicService) UpdateNic(_ context.Context, datacenterID, serverID, nicID string, props sdkgo.NicProperties) (sdkgo.Nic, *sdkgo.APIResponse, error) {
	f.Mu.Lock()
	defer f.Mu.Unlock()
	if f.UpdateErr != nil {
		return sdkgo.Nic{}, mocktest.ErrorResponse(http.StatusInternalServerError), f.UpdateErr
	}
	sn, ok := f.nics[f.key(datacenterID, serverID, nicID)]
	if !ok {
		return sdkgo.Nic{}, mocktest.ErrorResponse(http.StatusNotFound), fmt.Errorf("nic %s not found", nicID)
	}
	if props.Name != nil {
		sn.nic.Properties.Name = props.Name
	}
	if props.Dhcp != nil {
		sn.nic.Properties.Dhcp = props.Dhcp
	}
	f.nics[f.key(datacenterID, serverID, nicID)] = sn

	return sn.nic, f.AcceptedResponse("update", nicID), nil
}

func (f *fakeNicService) DeleteNic(_ context.Context, datacenterID, serverID, nicID string) (*sdkgo.APIResponse, error) {
	f.Mu.Lock()
	defer f.Mu.Unlock()
	if f.DeleteErr != nil {
		return mocktest.ErrorResponse(http.StatusNotFound), f.DeleteErr
	}
	f.DeleteCalls = append(f.DeleteCalls, nicID)
	delete(f.nics, f.key(datacenterID, serverID, nicID))

	return f.AcceptedResponse("delete", nicID), nil
}

// Test-only helpers

func (f *fakeNicService) storeNic(dcID, srvID, nicID string, n sdkgo.Nic) {
	f.Mu.Lock()
	defer f.Mu.Unlock()
	f.nics[f.key(dcID, srvID, nicID)] = storedNic{nic: n, datacenterID: dcID, serverID: srvID}
}

func (f *fakeNicService) removeNic(dcID, srvID, nicID string) {
	f.Mu.Lock()
	defer f.Mu.Unlock()
	delete(f.nics, f.key(dcID, srvID, nicID))
}

// ---------------------------------------------------------------------------
// Test connector
// ---------------------------------------------------------------------------

type testConnectorNic struct {
	service        nicClient.Client
	ipBlockService ipblockClient.Client
	log            logging.Logger
}

func (c *testConnectorNic) Connect(_ context.Context, _ resource.Managed) (managed.ExternalClient, error) {
	return &externalNic{service: c.service, ipBlockService: c.ipBlockService, log: c.log}, nil
}

// ---------------------------------------------------------------------------
// Test globals
// ---------------------------------------------------------------------------

var (
	k8sClient  client.Client
	fakeSvc    *fakeNicService
	testServer *mocktest.TestHTTPServer
	harness    *mocktest.EnvTestHarness
)

func TestNicController_Mock(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "NIC Controller Mock Suite")
}

var _ = BeforeSuite(func() {
	testServer = mocktest.NewTestHTTPServer()
	testServer.SetMode(mocktest.StatusModeDone)
	fakeSvc = newFakeNicService(testServer.URL())
	fakeIPSvc := mocktest.NewStubIPBlockService(testServer.URL())

	harness = mocktest.SetupEnvTest(mocktest.ControllerSetup{
		GroupKind:        v1alpha1.NicGroupKind,
		GroupVersionKind: v1alpha1.NicGroupVersionKind,
		ManagedResource:  &v1alpha1.Nic{},
		ManagedList:      &v1alpha1.NicList{},
		Connector: &testConnectorNic{
			service:        fakeSvc,
			ipBlockService: fakeIPSvc,
			log:            logging.NewLogrLogger(mocktest.Logger),
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

func newNicCR(name string) *v1alpha1.Nic {
	lanID, _ := strconv.Atoi(mocktest.TestLanID)
	return &v1alpha1.Nic{
		ObjectMeta: metav1.ObjectMeta{Name: name},
		Spec: v1alpha1.NicSpec{
			ResourceSpec: xpv1.ResourceSpec{
				DeletionPolicy:     xpv1.DeletionDelete,
				ManagementPolicies: xpv1.ManagementPolicies{xpv1.ManagementActionAll},
			},
			ForProvider: v1alpha1.NicParameters{
				DatacenterCfg: v1alpha1.DatacenterConfig{DatacenterID: mocktest.TestDatacenterID},
				ServerCfg:     v1alpha1.ServerConfig{ServerID: mocktest.TestServerID},
				LanCfg:        v1alpha1.LanConfig{LanID: strconv.Itoa(lanID)},
				Dhcp:          true,
				Name:          name,
			},
		},
	}
}

func getNicCR(ctx context.Context, name string) (*v1alpha1.Nic, error) {
	cr := &v1alpha1.Nic{}
	return cr, k8sClient.Get(ctx, types.NamespacedName{Name: name}, cr)
}

// ---------------------------------------------------------------------------
// Test scenarios
// ---------------------------------------------------------------------------

var _ = Describe("NIC Controller E2E Tests", func() {
	mocktest.RunStandardScenarios(mocktest.ScenarioConfig{
		ResourceName: "NIC",
		CRPrefix:     "test-nic",
		K8sClient:    func() client.Client { return k8sClient },
		TestServer:   func() *mocktest.TestHTTPServer { return testServer },
		CreateCR: func(name string) client.Object {
			return newNicCR(name)
		},
		CreateCRWithAnnotation: func(name, externalName, annotationValue string) client.Object {
			cr := newNicCR(name)
			cr.Annotations = map[string]string{
				"ionos.cloud/post-request-id": annotationValue,
			}
			if externalName != "" {
				meta.SetExternalName(cr, externalName)
			}
			return cr
		},
		GetCR: func(ctx context.Context, name string) (client.Object, error) {
			return getNicCR(ctx, name)
		},
		GetState: func(obj client.Object) string {
			return obj.(*v1alpha1.Nic).Status.AtProvider.State
		},
		GetResourceID: func(obj client.Object) string {
			return obj.(*v1alpha1.Nic).Status.AtProvider.NicID
		},
		MutateForUpdate: func(obj client.Object) {
			cr := obj.(*v1alpha1.Nic)
			cr.Spec.ForProvider.Name = "updated-nic"
			cr.Spec.ForProvider.Dhcp = false
		},
		StoreResource: func(externalID string) {
			lanInt, _ := strconv.Atoi(mocktest.TestLanID)
			fakeSvc.storeNic(mocktest.TestDatacenterID, mocktest.TestServerID, externalID, sdkgo.Nic{
				Id:         ptr.To(externalID),
				Properties: &sdkgo.NicProperties{Name: ptr.To("isreqdone-nic"), Dhcp: ptr.To(true), Lan: ptr.To(int32(lanInt))},
				Metadata:   &sdkgo.DatacenterElementMetadata{State: ptr.To("AVAILABLE")},
			})
		},
		RemoveResource: func(externalID string) {
			fakeSvc.removeNic(mocktest.TestDatacenterID, mocktest.TestServerID, externalID)
		},
		SetError:       func(method string, err error) { fakeSvc.SetError(method, err) },
		ClearErrors:    func() { fakeSvc.ClearErrors() },
		GetDeleteCalls: func() []string { return fakeSvc.GetDeleteCalls() },
	})
})
