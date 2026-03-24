//go:build lan_mock

package lan

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
	lanClient "github.com/ionos-cloud/crossplane-provider-ionoscloud/internal/clients/compute/lan"
	"github.com/ionos-cloud/crossplane-provider-ionoscloud/internal/controller/compute/mocktest"
)

// ---------------------------------------------------------------------------
// Fake LAN service implementing lan.Client
// ---------------------------------------------------------------------------

type storedLan struct {
	lan          sdkgo.Lan
	datacenterID string
}

type fakeLanService struct {
	mocktest.FakeServiceBase
	lans   map[string]storedLan
	nextID int

	createCalls []sdkgo.Lan
	updateCalls []sdkgo.LanProperties
}

func newFakeLanService(serverURL string) *fakeLanService {
	return &fakeLanService{
		FakeServiceBase: mocktest.NewFakeServiceBase(serverURL),
		lans:            make(map[string]storedLan),
		nextID:          1,
	}
}

func (f *fakeLanService) key(datacenterID, lanID string) string {
	return datacenterID + "/" + lanID
}

func (f *fakeLanService) CheckDuplicateLan(_ context.Context, datacenterID, lanName string) (*sdkgo.Lan, error) {
	f.Mu.Lock()
	defer f.Mu.Unlock()
	for _, sl := range f.lans {
		if sl.datacenterID == datacenterID && sl.lan.Properties != nil &&
			sl.lan.Properties.Name != nil && *sl.lan.Properties.Name == lanName {
			l := sl.lan
			return &l, nil
		}
	}
	return nil, nil
}

func (f *fakeLanService) GetLanID(l *sdkgo.Lan) (string, error) {
	if l != nil {
		if id, ok := l.GetIdOk(); ok && id != nil {
			return *id, nil
		}
		return "", fmt.Errorf("error: getting lan id")
	}
	return "", nil
}

func (f *fakeLanService) GetLan(_ context.Context, datacenterID, lanID string) (sdkgo.Lan, *sdkgo.APIResponse, error) {
	f.Mu.Lock()
	defer f.Mu.Unlock()
	if f.GetErr != nil {
		return sdkgo.Lan{}, mocktest.ErrorResponse(http.StatusInternalServerError), f.GetErr
	}
	sl, ok := f.lans[f.key(datacenterID, lanID)]
	if !ok {
		return sdkgo.Lan{}, mocktest.ErrorResponse(http.StatusNotFound), fmt.Errorf("lan %s not found", lanID)
	}
	return sl.lan, mocktest.OKResponse(), nil
}

func (f *fakeLanService) GetLanIPFailovers(_ context.Context, datacenterID, lanID string) ([]sdkgo.IPFailover, error) {
	f.Mu.Lock()
	defer f.Mu.Unlock()
	sl, ok := f.lans[f.key(datacenterID, lanID)]
	if !ok {
		return nil, fmt.Errorf("lan %s not found", lanID)
	}
	if sl.lan.Properties != nil {
		if ips, ok := sl.lan.Properties.GetIpFailoverOk(); ok && ips != nil {
			return *ips, nil
		}
	}
	return nil, fmt.Errorf("no ip failovers")
}

func (f *fakeLanService) CreateLan(_ context.Context, datacenterID string, l sdkgo.Lan) (sdkgo.Lan, *sdkgo.APIResponse, error) {
	f.Mu.Lock()
	defer f.Mu.Unlock()
	if f.CreateErr != nil {
		return sdkgo.Lan{}, mocktest.ErrorResponse(http.StatusInternalServerError), f.CreateErr
	}

	f.createCalls = append(f.createCalls, l)

	lanID := strconv.Itoa(f.nextID)
	f.nextID++

	props := &sdkgo.LanProperties{}
	if l.Properties != nil {
		if l.Properties.Name != nil {
			props.Name = l.Properties.Name
		}
		if l.Properties.Public != nil {
			props.Public = l.Properties.Public
		}
		if v, ok := l.Properties.GetIpv6CidrBlockOk(); ok && v != nil && *v != "" && *v != sdkgo.Nilstring {
			props.SetIpv6CidrBlock(*v)
		}
		if l.Properties.Pcc != nil {
			props.Pcc = l.Properties.Pcc
		}
	}
	newLan := sdkgo.Lan{
		Id:         &lanID,
		Properties: props,
		Metadata:   &sdkgo.DatacenterElementMetadata{State: ptr.To("AVAILABLE")},
	}
	f.lans[f.key(datacenterID, lanID)] = storedLan{lan: newLan, datacenterID: datacenterID}

	return newLan, f.AcceptedResponse("create", lanID), nil
}

func (f *fakeLanService) UpdateLan(_ context.Context, datacenterID, lanID string, props sdkgo.LanProperties) (sdkgo.Lan, *sdkgo.APIResponse, error) {
	f.Mu.Lock()
	defer f.Mu.Unlock()
	if f.UpdateErr != nil {
		return sdkgo.Lan{}, mocktest.ErrorResponse(http.StatusInternalServerError), f.UpdateErr
	}

	f.updateCalls = append(f.updateCalls, props)

	sl, ok := f.lans[f.key(datacenterID, lanID)]
	if !ok {
		return sdkgo.Lan{}, mocktest.ErrorResponse(http.StatusNotFound), fmt.Errorf("lan %s not found", lanID)
	}

	if props.Name != nil {
		sl.lan.Properties.Name = props.Name
	}
	if props.Public != nil {
		sl.lan.Properties.Public = props.Public
	}
	if v, ok := props.GetIpv6CidrBlockOk(); ok && v != nil && *v != "" && *v != sdkgo.Nilstring {
		sl.lan.Properties.SetIpv6CidrBlock(*v)
	}
	f.lans[f.key(datacenterID, lanID)] = sl

	return sl.lan, f.AcceptedResponse("update", lanID), nil
}

func (f *fakeLanService) DeleteLan(_ context.Context, datacenterID, lanID string) (*sdkgo.APIResponse, error) {
	f.Mu.Lock()
	defer f.Mu.Unlock()
	if f.DeleteErr != nil {
		return mocktest.ErrorResponse(http.StatusNotFound), f.DeleteErr
	}

	f.DeleteCalls = append(f.DeleteCalls, lanID)
	delete(f.lans, f.key(datacenterID, lanID))

	return f.AcceptedResponse("delete", lanID), nil
}

// Test-only helpers

func (f *fakeLanService) getLan(datacenterID, lanID string) (sdkgo.Lan, bool) {
	f.Mu.Lock()
	defer f.Mu.Unlock()
	sl, ok := f.lans[f.key(datacenterID, lanID)]
	return sl.lan, ok
}

func (f *fakeLanService) storeLan(datacenterID, lanID string, l sdkgo.Lan) {
	f.Mu.Lock()
	defer f.Mu.Unlock()
	f.lans[f.key(datacenterID, lanID)] = storedLan{lan: l, datacenterID: datacenterID}
}

func (f *fakeLanService) removeLan(datacenterID, lanID string) {
	f.Mu.Lock()
	defer f.Mu.Unlock()
	delete(f.lans, f.key(datacenterID, lanID))
}

func (f *fakeLanService) getUpdateCalls() []sdkgo.LanProperties {
	f.Mu.Lock()
	defer f.Mu.Unlock()
	result := make([]sdkgo.LanProperties, len(f.updateCalls))
	copy(result, f.updateCalls)
	return result
}

// ---------------------------------------------------------------------------
// Test connector
// ---------------------------------------------------------------------------

type testConnectorLan struct {
	service lanClient.Client
	log     logging.Logger
}

func (c *testConnectorLan) Connect(_ context.Context, _ resource.Managed) (managed.ExternalClient, error) {
	return &externalLan{service: c.service, log: c.log}, nil
}

// ---------------------------------------------------------------------------
// Test globals
// ---------------------------------------------------------------------------

var (
	k8sClient  client.Client
	fakeSvc    *fakeLanService
	testServer *mocktest.TestHTTPServer
	harness    *mocktest.EnvTestHarness
)

func TestLanController_Mock(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "LAN Controller Mock Suite")
}

var _ = BeforeSuite(func() {
	testServer = mocktest.NewTestHTTPServer()
	testServer.SetMode(mocktest.StatusModeDone)
	fakeSvc = newFakeLanService(testServer.URL())

	harness = mocktest.SetupEnvTest(mocktest.ControllerSetup{
		GroupKind:        v1alpha1.LanGroupKind,
		GroupVersionKind: v1alpha1.LanGroupVersionKind,
		ManagedResource:  &v1alpha1.Lan{},
		ManagedList:      &v1alpha1.LanList{},
		Connector: &testConnectorLan{
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

func newLanCR(name string) *v1alpha1.Lan {
	return &v1alpha1.Lan{
		ObjectMeta: metav1.ObjectMeta{Name: name},
		Spec: v1alpha1.LanSpec{
			ResourceSpec: xpv1.ResourceSpec{
				DeletionPolicy:     xpv1.DeletionDelete,
				ManagementPolicies: xpv1.ManagementPolicies{xpv1.ManagementActionAll},
			},
			ForProvider: v1alpha1.LanParameters{
				DatacenterCfg: v1alpha1.DatacenterConfig{DatacenterID: mocktest.TestDatacenterID},
				Name:          name,
				Public:        true,
			},
		},
	}
}

func getLanCR(ctx context.Context, name string) (*v1alpha1.Lan, error) {
	cr := &v1alpha1.Lan{}
	err := k8sClient.Get(ctx, types.NamespacedName{Name: name}, cr)
	return cr, err
}

// ---------------------------------------------------------------------------
// Test scenarios
// ---------------------------------------------------------------------------

var _ = Describe("LAN Controller E2E Tests", func() {
	mocktest.RunStandardScenarios(mocktest.ScenarioConfig{
		ResourceName: "LAN",
		CRPrefix:     "test-lan",
		K8sClient:    func() client.Client { return k8sClient },
		TestServer:   func() *mocktest.TestHTTPServer { return testServer },
		CreateCR: func(name string) client.Object {
			return newLanCR(name)
		},
		CreateCRWithAnnotation: func(name, externalName, annotationValue string) client.Object {
			cr := newLanCR(name)
			cr.Annotations = map[string]string{
				compute.POSTRequestIDAnnotationKey: annotationValue,
			}
			if externalName != "" {
				meta.SetExternalName(cr, externalName)
			}
			return cr
		},
		GetCR: func(ctx context.Context, name string) (client.Object, error) {
			return getLanCR(ctx, name)
		},
		GetState: func(obj client.Object) string {
			return obj.(*v1alpha1.Lan).Status.AtProvider.State
		},
		GetResourceID: func(obj client.Object) string {
			return obj.(*v1alpha1.Lan).Status.AtProvider.LanID
		},
		MutateForUpdate: func(obj client.Object) {
			cr := obj.(*v1alpha1.Lan)
			cr.Spec.ForProvider.Name = "updated-lan"
			cr.Spec.ForProvider.Public = false
		},
		StoreResource: func(externalID string) {
			fakeSvc.storeLan(mocktest.TestDatacenterID, externalID, sdkgo.Lan{
				Id: ptr.To(externalID),
				Properties: &sdkgo.LanProperties{
					Name:   ptr.To("isreqdone-lan"),
					Public: ptr.To(true),
				},
				Metadata: &sdkgo.DatacenterElementMetadata{State: ptr.To("AVAILABLE")},
			})
		},
		RemoveResource: func(externalID string) {
			fakeSvc.removeLan(mocktest.TestDatacenterID, externalID)
		},
		SetError:       func(method string, err error) { fakeSvc.SetError(method, err) },
		ClearErrors:    func() { fakeSvc.ClearErrors() },
		GetDeleteCalls: func() []string { return fakeSvc.GetDeleteCalls() },
	})

	// Scenario 11: IPv6 AUTO late initialization (LAN-specific)
	Describe("Scenario 11: IPv6 AUTO late initialization", Ordered, func() {
		var crName string

		BeforeAll(func() {
			testServer.SetMode(mocktest.StatusModeDone)
			fakeSvc.ClearErrors()
			crName = "test-lan-ipv6-auto"
		})

		AfterAll(func() {
			cr, err := getLanCR(context.Background(), crName)
			if err == nil {
				_ = k8sClient.Delete(context.Background(), cr)
				Eventually(func() bool {
					_, err := getLanCR(context.Background(), crName)
					return err != nil
				}, mocktest.Timeout, mocktest.Interval).Should(BeTrue())
			}
		})

		It("should late-initialize ipv6Cidr from AUTO to the server-assigned value", func() {
			ctx := context.Background()
			cr := newLanCR(crName)
			cr.Spec.ForProvider.Ipv6Cidr = v1alpha1.LANAuto
			Expect(k8sClient.Create(ctx, cr)).Should(Succeed())

			var lanID string
			Eventually(func(g Gomega) {
				fetched, err := getLanCR(ctx, crName)
				g.Expect(err).NotTo(HaveOccurred())
				lanID = meta.GetExternalName(fetched)
				g.Expect(lanID).NotTo(BeEmpty())
				g.Expect(fetched.Status.AtProvider.State).To(Equal("AVAILABLE"))
			}, mocktest.Timeout, mocktest.Interval).Should(Succeed())

			By("simulating server-assigned IPv6 CIDR")
			l, ok := fakeSvc.getLan(mocktest.TestDatacenterID, lanID)
			Expect(ok).To(BeTrue())
			assignedCidr := "fd00::/64"
			l.Properties.SetIpv6CidrBlock(assignedCidr)
			fakeSvc.storeLan(mocktest.TestDatacenterID, lanID, l)

			Eventually(func(g Gomega) {
				fetched, err := getLanCR(ctx, crName)
				g.Expect(err).NotTo(HaveOccurred())
				g.Expect(fetched.Spec.ForProvider.Ipv6Cidr).To(Equal(assignedCidr))
			}, mocktest.Timeout, mocktest.Interval).Should(Succeed())
		})
	})
})
