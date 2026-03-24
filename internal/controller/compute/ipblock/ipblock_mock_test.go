//go:build ipblock_mock

package ipblock

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
	"github.com/ionos-cloud/crossplane-provider-ionoscloud/internal/clients/compute"
	ipblockClient "github.com/ionos-cloud/crossplane-provider-ionoscloud/internal/clients/compute/ipblock"
	"github.com/ionos-cloud/crossplane-provider-ionoscloud/internal/controller/compute/mocktest"
)

// ---------------------------------------------------------------------------
// Fake IPBlock service
// ---------------------------------------------------------------------------

type storedIPBlock struct {
	ipBlock sdkgo.IpBlock
}

type fakeIPBlockService struct {
	mocktest.FakeServiceBase
	ipBlocks map[string]storedIPBlock
	nextID   int
}

func newFakeIPBlockService(serverURL string) *fakeIPBlockService {
	return &fakeIPBlockService{
		FakeServiceBase: mocktest.NewFakeServiceBase(serverURL),
		ipBlocks:        make(map[string]storedIPBlock),
		nextID:          1,
	}
}

func generateIPs(size int32) *[]string {
	ips := make([]string, size)
	for i := int32(0); i < size; i++ {
		ips[i] = fmt.Sprintf("10.0.0.%d", i+1)
	}
	return &ips
}

func (f *fakeIPBlockService) CheckDuplicateIPBlock(_ context.Context, ipBlockName, location string) (*sdkgo.IpBlock, error) {
	f.Mu.Lock()
	defer f.Mu.Unlock()
	for _, sb := range f.ipBlocks {
		if sb.ipBlock.Properties != nil &&
			sb.ipBlock.Properties.Name != nil && *sb.ipBlock.Properties.Name == ipBlockName &&
			sb.ipBlock.Properties.Location != nil && *sb.ipBlock.Properties.Location == location {
			b := sb.ipBlock
			return &b, nil
		}
	}
	return nil, nil
}

func (f *fakeIPBlockService) GetIPBlockID(b *sdkgo.IpBlock) (string, error) {
	if b != nil {
		if id, ok := b.GetIdOk(); ok && id != nil {
			return *id, nil
		}
		return "", fmt.Errorf("error: getting ipblock id")
	}
	return "", nil
}

func (f *fakeIPBlockService) GetIPBlock(_ context.Context, ipBlockID string) (sdkgo.IpBlock, *sdkgo.APIResponse, error) {
	f.Mu.Lock()
	defer f.Mu.Unlock()
	if f.GetErr != nil {
		return sdkgo.IpBlock{}, mocktest.ErrorResponse(http.StatusInternalServerError), f.GetErr
	}
	sb, ok := f.ipBlocks[ipBlockID]
	if !ok {
		return sdkgo.IpBlock{}, mocktest.ErrorResponse(http.StatusNotFound), fmt.Errorf("ipblock %s not found", ipBlockID)
	}
	return sb.ipBlock, mocktest.OKResponse(), nil
}

func (f *fakeIPBlockService) CreateIPBlock(_ context.Context, b sdkgo.IpBlock) (sdkgo.IpBlock, *sdkgo.APIResponse, error) {
	f.Mu.Lock()
	defer f.Mu.Unlock()
	if f.CreateErr != nil {
		return sdkgo.IpBlock{}, mocktest.ErrorResponse(http.StatusInternalServerError), f.CreateErr
	}
	blockID := strconv.Itoa(f.nextID)
	f.nextID++

	var size int32 = 1
	props := &sdkgo.IpBlockProperties{}
	if b.Properties != nil {
		if b.Properties.Name != nil {
			props.Name = b.Properties.Name
		}
		if b.Properties.Location != nil {
			props.Location = b.Properties.Location
		}
		if b.Properties.Size != nil {
			size = *b.Properties.Size
			props.Size = b.Properties.Size
		}
	}
	props.Ips = generateIPs(size)

	newBlock := sdkgo.IpBlock{
		Id:         &blockID,
		Properties: props,
		Metadata:   &sdkgo.DatacenterElementMetadata{State: ptr.To("AVAILABLE")},
	}
	f.ipBlocks[blockID] = storedIPBlock{ipBlock: newBlock}

	return newBlock, f.AcceptedResponse("create", blockID), nil
}

func (f *fakeIPBlockService) UpdateIPBlock(_ context.Context, ipBlockID string, props sdkgo.IpBlockProperties) (sdkgo.IpBlock, *sdkgo.APIResponse, error) {
	f.Mu.Lock()
	defer f.Mu.Unlock()
	if f.UpdateErr != nil {
		return sdkgo.IpBlock{}, mocktest.ErrorResponse(http.StatusInternalServerError), f.UpdateErr
	}
	sb, ok := f.ipBlocks[ipBlockID]
	if !ok {
		return sdkgo.IpBlock{}, mocktest.ErrorResponse(http.StatusNotFound), fmt.Errorf("ipblock %s not found", ipBlockID)
	}
	if props.Name != nil {
		sb.ipBlock.Properties.Name = props.Name
	}
	f.ipBlocks[ipBlockID] = sb

	return sb.ipBlock, f.AcceptedResponse("update", ipBlockID), nil
}

func (f *fakeIPBlockService) DeleteIPBlock(_ context.Context, ipBlockID string) (*sdkgo.APIResponse, error) {
	f.Mu.Lock()
	defer f.Mu.Unlock()
	if f.DeleteErr != nil {
		return mocktest.ErrorResponse(http.StatusNotFound), f.DeleteErr
	}
	f.DeleteCalls = append(f.DeleteCalls, ipBlockID)
	delete(f.ipBlocks, ipBlockID)

	return f.AcceptedResponse("delete", ipBlockID), nil
}

func (f *fakeIPBlockService) GetIPs(_ context.Context, ipBlockID string, indexes ...int) ([]string, error) {
	f.Mu.Lock()
	defer f.Mu.Unlock()
	sb, ok := f.ipBlocks[ipBlockID]
	if !ok {
		return nil, fmt.Errorf("ipblock %s not found", ipBlockID)
	}
	if sb.ipBlock.Properties == nil || sb.ipBlock.Properties.Ips == nil {
		return nil, nil
	}
	allIPs := *sb.ipBlock.Properties.Ips
	if len(indexes) == 0 {
		return allIPs, nil
	}
	var result []string
	for _, idx := range indexes {
		if idx >= 0 && idx < len(allIPs) {
			result = append(result, allIPs[idx])
		}
	}
	return result, nil
}

// Test-only helpers

func (f *fakeIPBlockService) storeIPBlock(id string, b sdkgo.IpBlock) {
	f.Mu.Lock()
	defer f.Mu.Unlock()
	f.ipBlocks[id] = storedIPBlock{ipBlock: b}
}

func (f *fakeIPBlockService) removeIPBlock(id string) {
	f.Mu.Lock()
	defer f.Mu.Unlock()
	delete(f.ipBlocks, id)
}

// ---------------------------------------------------------------------------
// Test connector
// ---------------------------------------------------------------------------

type testConnectorIPBlock struct {
	service ipblockClient.Client
	log     logging.Logger
}

func (c *testConnectorIPBlock) Connect(_ context.Context, _ resource.Managed) (managed.ExternalClient, error) {
	return &externalIPBlock{service: c.service, log: c.log}, nil
}

// ---------------------------------------------------------------------------
// Test globals
// ---------------------------------------------------------------------------

var (
	k8sClient  client.Client
	fakeSvc    *fakeIPBlockService
	testServer *mocktest.TestHTTPServer
	harness    *mocktest.EnvTestHarness
)

func TestIPBlockController_Mock(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "IPBlock Controller Mock Suite")
}

var _ = BeforeSuite(func() {
	testServer = mocktest.NewTestHTTPServer()
	testServer.SetMode(mocktest.StatusModeDone)
	fakeSvc = newFakeIPBlockService(testServer.URL())

	harness = mocktest.SetupEnvTest(mocktest.ControllerSetup{
		GroupKind:        v1alpha1.IPBlockGroupKind,
		GroupVersionKind: v1alpha1.IPBlockGroupVersionKind,
		ManagedResource:  &v1alpha1.IPBlock{},
		ManagedList:      &v1alpha1.IPBlockList{},
		Connector: &testConnectorIPBlock{
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

func newIPBlockCR(name string) *v1alpha1.IPBlock {
	return &v1alpha1.IPBlock{
		ObjectMeta: metav1.ObjectMeta{Name: name},
		Spec: v1alpha1.IPBlockSpec{
			ResourceSpec: xpv1.ResourceSpec{
				DeletionPolicy:     xpv1.DeletionDelete,
				ManagementPolicies: xpv1.ManagementPolicies{xpv1.ManagementActionAll},
			},
			ForProvider: v1alpha1.IPBlockParameters{
				Name:     name,
				Location: "de/fra",
				Size:     2,
			},
		},
	}
}

func getIPBlockCR(ctx context.Context, name string) (*v1alpha1.IPBlock, error) {
	cr := &v1alpha1.IPBlock{}
	return cr, k8sClient.Get(ctx, types.NamespacedName{Name: name}, cr)
}

// ---------------------------------------------------------------------------
// Test scenarios
// ---------------------------------------------------------------------------

var _ = Describe("IPBlock Controller E2E Tests", func() {
	mocktest.RunStandardScenarios(mocktest.ScenarioConfig{
		ResourceName: "IPBlock",
		CRPrefix:     "test-ipb",
		K8sClient:    func() client.Client { return k8sClient },
		TestServer:   func() *mocktest.TestHTTPServer { return testServer },
		CreateCR: func(name string) client.Object {
			return newIPBlockCR(name)
		},
		CreateCRWithAnnotation: func(name, externalName, annotationValue string) client.Object {
			cr := newIPBlockCR(name)
			cr.Annotations = map[string]string{
				compute.POSTRequestIDAnnotationKey: annotationValue,
			}
			if externalName != "" {
				meta.SetExternalName(cr, externalName)
			}
			return cr
		},
		GetCR: func(ctx context.Context, name string) (client.Object, error) {
			return getIPBlockCR(ctx, name)
		},
		GetState: func(obj client.Object) string {
			return obj.(*v1alpha1.IPBlock).Status.AtProvider.State
		},
		GetResourceID: func(obj client.Object) string {
			return obj.(*v1alpha1.IPBlock).Status.AtProvider.IPBlockID
		},
		MutateForUpdate: func(obj client.Object) {
			cr := obj.(*v1alpha1.IPBlock)
			cr.Spec.ForProvider.Name = "updated-ipblock"
		},
		StoreResource: func(externalID string) {
			ips := generateIPs(2)
			fakeSvc.storeIPBlock(externalID, sdkgo.IpBlock{
				Id:         ptr.To(externalID),
				Properties: &sdkgo.IpBlockProperties{Name: ptr.To("isreqdone-ipblock"), Location: ptr.To("de/fra"), Size: ptr.To(int32(2)), Ips: ips},
				Metadata:   &sdkgo.DatacenterElementMetadata{State: ptr.To("AVAILABLE")},
			})
		},
		RemoveResource: func(externalID string) {
			fakeSvc.removeIPBlock(externalID)
		},
		SetError:       func(method string, err error) { fakeSvc.SetError(method, err) },
		ClearErrors:    func() { fakeSvc.ClearErrors() },
		GetDeleteCalls: func() []string { return fakeSvc.GetDeleteCalls() },
	})
})
