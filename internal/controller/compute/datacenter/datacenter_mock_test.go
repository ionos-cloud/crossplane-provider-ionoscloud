//go:build datacenter_mock

package datacenter

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
	datacenterClient "github.com/ionos-cloud/crossplane-provider-ionoscloud/internal/clients/compute/datacenter"
	"github.com/ionos-cloud/crossplane-provider-ionoscloud/internal/controller/compute/mocktest"
)

// ---------------------------------------------------------------------------
// Fake Datacenter service
// ---------------------------------------------------------------------------

type storedDatacenter struct {
	datacenter sdkgo.Datacenter
}

type fakeDatacenterService struct {
	mocktest.FakeServiceBase
	datacenters map[string]storedDatacenter
	nextID      int

	createCalls []sdkgo.DatacenterPost
	updateCalls []sdkgo.DatacenterPropertiesPut
}

func newFakeDatacenterService(serverURL string) *fakeDatacenterService {
	return &fakeDatacenterService{
		FakeServiceBase: mocktest.NewFakeServiceBase(serverURL),
		datacenters:     make(map[string]storedDatacenter),
		nextID:          1,
	}
}

func (f *fakeDatacenterService) CheckDuplicateDatacenter(_ context.Context, datacenterName, location string) (*sdkgo.Datacenter, error) {
	f.Mu.Lock()
	defer f.Mu.Unlock()
	for _, sd := range f.datacenters {
		if sd.datacenter.Properties != nil &&
			sd.datacenter.Properties.Name != nil && *sd.datacenter.Properties.Name == datacenterName &&
			sd.datacenter.Properties.Location != nil && *sd.datacenter.Properties.Location == location {
			d := sd.datacenter
			return &d, nil
		}
	}
	return nil, nil
}

func (f *fakeDatacenterService) GetDatacenterID(d *sdkgo.Datacenter) (string, error) {
	if d != nil {
		if id, ok := d.GetIdOk(); ok && id != nil {
			return *id, nil
		}
		return "", fmt.Errorf("error: getting datacenter id")
	}
	return "", nil
}

func (f *fakeDatacenterService) GetDatacenter(_ context.Context, datacenterID string) (sdkgo.Datacenter, *sdkgo.APIResponse, error) {
	f.Mu.Lock()
	defer f.Mu.Unlock()
	if f.GetErr != nil {
		return sdkgo.Datacenter{}, mocktest.ErrorResponse(http.StatusInternalServerError), f.GetErr
	}
	sd, ok := f.datacenters[datacenterID]
	if !ok {
		return sdkgo.Datacenter{}, mocktest.ErrorResponse(http.StatusNotFound), fmt.Errorf("datacenter %s not found", datacenterID)
	}
	return sd.datacenter, mocktest.OKResponse(), nil
}

func (f *fakeDatacenterService) CreateDatacenter(_ context.Context, d sdkgo.DatacenterPost) (sdkgo.Datacenter, *sdkgo.APIResponse, error) {
	f.Mu.Lock()
	defer f.Mu.Unlock()
	if f.CreateErr != nil {
		return sdkgo.Datacenter{}, mocktest.ErrorResponse(http.StatusInternalServerError), f.CreateErr
	}

	f.createCalls = append(f.createCalls, d)
	dcID := strconv.Itoa(f.nextID)
	f.nextID++

	props := &sdkgo.DatacenterProperties{}
	if d.Properties != nil {
		if d.Properties.Name != nil {
			props.Name = d.Properties.Name
		}
		if d.Properties.Location != nil {
			props.Location = d.Properties.Location
		}
		if d.Properties.Description != nil {
			props.Description = d.Properties.Description
		}
	}
	newDC := sdkgo.Datacenter{
		Id:         &dcID,
		Properties: props,
		Metadata:   &sdkgo.DatacenterElementMetadata{State: ptr.To("AVAILABLE")},
	}
	f.datacenters[dcID] = storedDatacenter{datacenter: newDC}

	return newDC, f.AcceptedResponse("create", dcID), nil
}

func (f *fakeDatacenterService) UpdateDatacenter(_ context.Context, datacenterID string, props sdkgo.DatacenterPropertiesPut) (sdkgo.Datacenter, *sdkgo.APIResponse, error) {
	f.Mu.Lock()
	defer f.Mu.Unlock()
	if f.UpdateErr != nil {
		return sdkgo.Datacenter{}, mocktest.ErrorResponse(http.StatusInternalServerError), f.UpdateErr
	}

	f.updateCalls = append(f.updateCalls, props)
	sd, ok := f.datacenters[datacenterID]
	if !ok {
		return sdkgo.Datacenter{}, mocktest.ErrorResponse(http.StatusNotFound), fmt.Errorf("datacenter %s not found", datacenterID)
	}

	if props.Name != nil {
		sd.datacenter.Properties.Name = props.Name
	}
	if props.Description != nil {
		sd.datacenter.Properties.Description = props.Description
	}
	f.datacenters[datacenterID] = sd

	return sd.datacenter, f.AcceptedResponse("update", datacenterID), nil
}

func (f *fakeDatacenterService) DeleteDatacenter(_ context.Context, datacenterID string) (*sdkgo.APIResponse, error) {
	f.Mu.Lock()
	defer f.Mu.Unlock()
	if f.DeleteErr != nil {
		return mocktest.ErrorResponse(http.StatusNotFound), f.DeleteErr
	}

	f.DeleteCalls = append(f.DeleteCalls, datacenterID)
	delete(f.datacenters, datacenterID)

	return f.AcceptedResponse("delete", datacenterID), nil
}

func (f *fakeDatacenterService) GetCPUFamiliesForDatacenter(_ context.Context, _ string) ([]string, error) {
	return []string{"INTEL_SKYLAKE", "AMD_EPYC"}, nil
}

// Test-only helpers

func (f *fakeDatacenterService) storeDatacenter(datacenterID string, d sdkgo.Datacenter) {
	f.Mu.Lock()
	defer f.Mu.Unlock()
	f.datacenters[datacenterID] = storedDatacenter{datacenter: d}
}

func (f *fakeDatacenterService) removeDatacenter(datacenterID string) {
	f.Mu.Lock()
	defer f.Mu.Unlock()
	delete(f.datacenters, datacenterID)
}

// ---------------------------------------------------------------------------
// Test connector
// ---------------------------------------------------------------------------

type testConnectorDatacenter struct {
	service datacenterClient.Client
	log     logging.Logger
}

func (c *testConnectorDatacenter) Connect(_ context.Context, _ resource.Managed) (managed.ExternalClient, error) {
	return &externalDatacenter{service: c.service, log: c.log}, nil
}

// ---------------------------------------------------------------------------
// Test globals
// ---------------------------------------------------------------------------

var (
	k8sClient  client.Client
	fakeSvc    *fakeDatacenterService
	testServer *mocktest.TestHTTPServer
	harness    *mocktest.EnvTestHarness
)

func TestDatacenterController_Mock(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Datacenter Controller Mock Suite")
}

var _ = BeforeSuite(func() {
	testServer = mocktest.NewTestHTTPServer()
	testServer.SetMode(mocktest.StatusModeDone)
	fakeSvc = newFakeDatacenterService(testServer.URL())

	harness = mocktest.SetupEnvTest(mocktest.ControllerSetup{
		GroupKind:        v1alpha1.DatacenterGroupKind,
		GroupVersionKind: v1alpha1.DatacenterGroupVersionKind,
		ManagedResource:  &v1alpha1.Datacenter{},
		ManagedList:      &v1alpha1.DatacenterList{},
		Connector: &testConnectorDatacenter{
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

func newDatacenterCR(name string) *v1alpha1.Datacenter {
	return &v1alpha1.Datacenter{
		ObjectMeta: metav1.ObjectMeta{Name: name},
		Spec: v1alpha1.DatacenterSpec{
			ResourceSpec: xpv1.ResourceSpec{
				DeletionPolicy:     xpv1.DeletionDelete,
				ManagementPolicies: xpv1.ManagementPolicies{xpv1.ManagementActionAll},
			},
			ForProvider: v1alpha1.DatacenterParameters{
				Name:     name,
				Location: "de/fra",
			},
		},
	}
}

func getDatacenterCR(ctx context.Context, name string) (*v1alpha1.Datacenter, error) {
	cr := &v1alpha1.Datacenter{}
	err := k8sClient.Get(ctx, types.NamespacedName{Name: name}, cr)
	return cr, err
}

// ---------------------------------------------------------------------------
// Test scenarios
// ---------------------------------------------------------------------------

var _ = Describe("Datacenter Controller E2E Tests", func() {
	mocktest.RunStandardScenarios(mocktest.ScenarioConfig{
		ResourceName: "Datacenter",
		CRPrefix:     "test-dc",
		K8sClient:    func() client.Client { return k8sClient },
		TestServer:   func() *mocktest.TestHTTPServer { return testServer },
		CreateCR: func(name string) client.Object {
			return newDatacenterCR(name)
		},
		CreateCRWithAnnotation: func(name, externalName, annotationValue string) client.Object {
			cr := newDatacenterCR(name)
			cr.Annotations = map[string]string{
				"ionos.cloud/post-request-id": annotationValue,
			}
			if externalName != "" {
				meta.SetExternalName(cr, externalName)
			}
			return cr
		},
		GetCR: func(ctx context.Context, name string) (client.Object, error) {
			return getDatacenterCR(ctx, name)
		},
		GetState: func(obj client.Object) string {
			return obj.(*v1alpha1.Datacenter).Status.AtProvider.State
		},
		GetResourceID: func(obj client.Object) string {
			return obj.(*v1alpha1.Datacenter).Status.AtProvider.DatacenterID
		},
		MutateForUpdate: func(obj client.Object) {
			cr := obj.(*v1alpha1.Datacenter)
			cr.Spec.ForProvider.Name = "updated-datacenter"
		},
		StoreResource: func(externalID string) {
			fakeSvc.storeDatacenter(externalID, sdkgo.Datacenter{
				Id: ptr.To(externalID),
				Properties: &sdkgo.DatacenterProperties{
					Name:     ptr.To("isreqdone-datacenter"),
					Location: ptr.To("de/fra"),
				},
				Metadata: &sdkgo.DatacenterElementMetadata{State: ptr.To("AVAILABLE")},
			})
		},
		RemoveResource: func(externalID string) {
			fakeSvc.removeDatacenter(externalID)
		},
		SetError:       func(method string, err error) { fakeSvc.SetError(method, err) },
		ClearErrors:    func() { fakeSvc.ClearErrors() },
		GetDeleteCalls: func() []string { return fakeSvc.GetDeleteCalls() },
	})
})
