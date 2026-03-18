//go:build nic_mock

package nic

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	xpv1 "github.com/crossplane/crossplane-runtime/apis/common/v1"
	"github.com/crossplane/crossplane-runtime/pkg/event"
	"github.com/crossplane/crossplane-runtime/pkg/logging"
	"github.com/crossplane/crossplane-runtime/pkg/meta"
	"github.com/crossplane/crossplane-runtime/pkg/ratelimiter"
	"github.com/crossplane/crossplane-runtime/pkg/reconciler/managed"
	"github.com/crossplane/crossplane-runtime/pkg/resource"
	"github.com/crossplane/crossplane-runtime/pkg/statemetrics"
	sdkgo "github.com/ionos-cloud/sdk-go/v6"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/utils/ptr"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"

	"github.com/ionos-cloud/crossplane-provider-ionoscloud/apis"
	"github.com/ionos-cloud/crossplane-provider-ionoscloud/apis/compute/v1alpha1"
	"github.com/ionos-cloud/crossplane-provider-ionoscloud/internal/clients/compute"
	ipblockClient "github.com/ionos-cloud/crossplane-provider-ionoscloud/internal/clients/compute/ipblock"
	nicClient "github.com/ionos-cloud/crossplane-provider-ionoscloud/internal/clients/compute/nic"
)

type requestStatusMode int32

const (
	statusModeDone    requestStatusMode = iota
	statusModeRunning
	statusModeFailed
	statusModeError
	statusMode404
)

type testHTTPServer struct {
	server *httptest.Server
	mode   atomic.Int32
}

func newTestHTTPServer() *testHTTPServer {
	ts := &testHTTPServer{}
	mux := http.NewServeMux()
	mux.HandleFunc("/cloudapi/v6/requests/", func(w http.ResponseWriter, r *http.Request) {
		mode := requestStatusMode(ts.mode.Load())
		w.Header().Set("Content-Type", "application/json")
		switch mode {
		case statusModeDone:
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(map[string]interface{}{"id": "test-request-id", "type": "request-status", "metadata": map[string]interface{}{"status": "DONE"}})
		case statusModeRunning:
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(map[string]interface{}{"id": "test-request-id", "type": "request-status", "metadata": map[string]interface{}{"status": "RUNNING"}})
		case statusModeFailed:
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(map[string]interface{}{"id": "test-request-id", "type": "request-status", "metadata": map[string]interface{}{"status": "FAILED", "message": "simulated failure"}})
		case statusModeError:
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(map[string]interface{}{"error": "internal server error"})
		case statusMode404:
			w.WriteHeader(http.StatusNotFound)
			json.NewEncoder(w).Encode(map[string]interface{}{"error": "not found"})
		}
	})
	ts.server = httptest.NewServer(mux)
	return ts
}

func (ts *testHTTPServer) setMode(mode requestStatusMode) { ts.mode.Store(int32(mode)) }
func (ts *testHTTPServer) stop()                           { ts.server.Close() }

// ---------------------------------------------------------------------------
// Fake NIC service
// ---------------------------------------------------------------------------

type storedNic struct {
	nic          sdkgo.Nic
	datacenterID string
	serverID     string
}

type fakeNicService struct {
	mu   sync.Mutex
	nics map[string]storedNic
	nextID int

	createErr, getErr, updateErr, deleteErr error
	deleteCalls                             []string

	apiClient *sdkgo.APIClient
	serverURL string
}

func newFakeNicService(serverURL string) *fakeNicService {
	cfg := sdkgo.NewConfiguration("", "", "test-token", serverURL)
	cfg.PollInterval = 100 * time.Millisecond
	return &fakeNicService{
		nics:      make(map[string]storedNic),
		nextID:    1,
		apiClient: sdkgo.NewAPIClient(cfg),
		serverURL: serverURL,
	}
}

func (f *fakeNicService) key(dcID, srvID, nicID string) string {
	return dcID + "/" + srvID + "/" + nicID
}

func (f *fakeNicService) CheckDuplicateNic(_ context.Context, datacenterID, serverID, nicName string) (*sdkgo.Nic, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
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
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.getErr != nil {
		return sdkgo.Nic{}, &sdkgo.APIResponse{Response: &http.Response{StatusCode: http.StatusInternalServerError}}, f.getErr
	}
	sn, ok := f.nics[f.key(datacenterID, serverID, nicID)]
	if !ok {
		return sdkgo.Nic{}, &sdkgo.APIResponse{Response: &http.Response{StatusCode: http.StatusNotFound}}, fmt.Errorf("nic %s not found", nicID)
	}
	return sn.nic, &sdkgo.APIResponse{Response: &http.Response{StatusCode: http.StatusOK}}, nil
}

func (f *fakeNicService) CreateNic(_ context.Context, datacenterID, serverID string, n sdkgo.Nic) (sdkgo.Nic, *sdkgo.APIResponse, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.createErr != nil {
		return sdkgo.Nic{}, &sdkgo.APIResponse{Response: &http.Response{StatusCode: http.StatusInternalServerError}}, f.createErr
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

	header := http.Header{}
	header.Set("Location", f.serverURL+"/cloudapi/v6/requests/create-req-"+nicID+"/status")
	return newNic, &sdkgo.APIResponse{Response: &http.Response{StatusCode: http.StatusAccepted, Header: header}}, nil
}

func (f *fakeNicService) UpdateNic(_ context.Context, datacenterID, serverID, nicID string, props sdkgo.NicProperties) (sdkgo.Nic, *sdkgo.APIResponse, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.updateErr != nil {
		return sdkgo.Nic{}, &sdkgo.APIResponse{Response: &http.Response{StatusCode: http.StatusInternalServerError}}, f.updateErr
	}
	sn, ok := f.nics[f.key(datacenterID, serverID, nicID)]
	if !ok {
		return sdkgo.Nic{}, &sdkgo.APIResponse{Response: &http.Response{StatusCode: http.StatusNotFound}}, fmt.Errorf("nic %s not found", nicID)
	}
	if props.Name != nil {
		sn.nic.Properties.Name = props.Name
	}
	if props.Dhcp != nil {
		sn.nic.Properties.Dhcp = props.Dhcp
	}
	f.nics[f.key(datacenterID, serverID, nicID)] = sn

	header := http.Header{}
	header.Set("Location", f.serverURL+"/cloudapi/v6/requests/update-req-"+nicID+"/status")
	return sn.nic, &sdkgo.APIResponse{Response: &http.Response{StatusCode: http.StatusAccepted, Header: header}}, nil
}

func (f *fakeNicService) DeleteNic(_ context.Context, datacenterID, serverID, nicID string) (*sdkgo.APIResponse, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.deleteErr != nil {
		return &sdkgo.APIResponse{Response: &http.Response{StatusCode: http.StatusNotFound}}, f.deleteErr
	}
	f.deleteCalls = append(f.deleteCalls, nicID)
	delete(f.nics, f.key(datacenterID, serverID, nicID))

	header := http.Header{}
	header.Set("Location", f.serverURL+"/cloudapi/v6/requests/delete-req-"+nicID+"/status")
	return &sdkgo.APIResponse{Response: &http.Response{StatusCode: http.StatusAccepted, Header: header}}, nil
}

func (f *fakeNicService) GetAPIClient() *sdkgo.APIClient { return f.apiClient }

func (f *fakeNicService) setError(method string, err error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	switch method {
	case "create":
		f.createErr = err
	case "get":
		f.getErr = err
	case "update":
		f.updateErr = err
	case "delete":
		f.deleteErr = err
	}
}

func (f *fakeNicService) clearErrors() {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.createErr = nil
	f.getErr = nil
	f.updateErr = nil
	f.deleteErr = nil
}

func (f *fakeNicService) storeNic(dcID, srvID, nicID string, n sdkgo.Nic) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.nics[f.key(dcID, srvID, nicID)] = storedNic{nic: n, datacenterID: dcID, serverID: srvID}
}

func (f *fakeNicService) removeNic(dcID, srvID, nicID string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	delete(f.nics, f.key(dcID, srvID, nicID))
}

func (f *fakeNicService) getDeleteCalls() []string {
	f.mu.Lock()
	defer f.mu.Unlock()
	result := make([]string, len(f.deleteCalls))
	copy(result, f.deleteCalls)
	return result
}

// ---------------------------------------------------------------------------
// Fake IPBlock service (minimal, for NIC's ipBlockService dependency)
// ---------------------------------------------------------------------------

type fakeIPBlockService struct {
	apiClient *sdkgo.APIClient
}

func newFakeIPBlockService(serverURL string) *fakeIPBlockService {
	cfg := sdkgo.NewConfiguration("", "", "test-token", serverURL)
	return &fakeIPBlockService{apiClient: sdkgo.NewAPIClient(cfg)}
}

func (f *fakeIPBlockService) CheckDuplicateIPBlock(_ context.Context, _, _ string) (*sdkgo.IpBlock, error) {
	return nil, nil
}
func (f *fakeIPBlockService) GetIPBlockID(_ *sdkgo.IpBlock) (string, error) { return "", nil }
func (f *fakeIPBlockService) GetIPBlock(_ context.Context, _ string) (sdkgo.IpBlock, *sdkgo.APIResponse, error) {
	return sdkgo.IpBlock{}, nil, nil
}
func (f *fakeIPBlockService) CreateIPBlock(_ context.Context, _ sdkgo.IpBlock) (sdkgo.IpBlock, *sdkgo.APIResponse, error) {
	return sdkgo.IpBlock{}, nil, nil
}
func (f *fakeIPBlockService) UpdateIPBlock(_ context.Context, _ string, _ sdkgo.IpBlockProperties) (sdkgo.IpBlock, *sdkgo.APIResponse, error) {
	return sdkgo.IpBlock{}, nil, nil
}
func (f *fakeIPBlockService) DeleteIPBlock(_ context.Context, _ string) (*sdkgo.APIResponse, error) {
	return nil, nil
}
func (f *fakeIPBlockService) GetIPs(_ context.Context, _ string, _ ...int) ([]string, error) {
	return nil, nil
}
func (f *fakeIPBlockService) GetAPIClient() *sdkgo.APIClient { return f.apiClient }

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
	cancel     context.CancelFunc
	testEnv    *envtest.Environment
	fakeSvc    *fakeNicService
	testServer *testHTTPServer
)

var logger = zap.New(zap.UseDevMode(true))

const (
	timeout          = 60 * time.Second
	interval         = 500 * time.Millisecond
	testDatacenterID = "a1b2c3d4-e5f6-7890-abcd-ef1234567890"
	testServerID     = "b2c3d4e5-f6a7-8901-bcde-f12345678901"
	testLanID        = "1"
)

func TestNicController_Mock(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "NIC Controller Mock Suite")
}

var _ = BeforeSuite(func() {
	logf.SetLogger(logger)
	var ctx context.Context
	ctx, cancel = context.WithCancel(context.Background())

	testServer = newTestHTTPServer()
	testServer.setMode(statusModeDone)
	fakeSvc = newFakeNicService(testServer.server.URL)
	fakeIPSvc := newFakeIPBlockService(testServer.server.URL)

	testEnv = &envtest.Environment{
		CRDDirectoryPaths:     []string{filepath.Join("..", "..", "..", "..", "package", "crds")},
		ErrorIfCRDPathMissing: true,
		DownloadBinaryAssets:  true,
		BinaryAssetsDirectory: filepath.Join(os.TempDir(), "envtest-binaries"),
	}
	cfg, err := testEnv.Start()
	Expect(err).NotTo(HaveOccurred())
	Expect(cfg).NotTo(BeNil())

	err = apis.AddToScheme(scheme.Scheme)
	Expect(err).NotTo(HaveOccurred())
	k8sClient, err = client.New(cfg, client.Options{Scheme: scheme.Scheme})
	Expect(err).NotTo(HaveOccurred())

	ctrl.SetLogger(logger)
	mgr, err := ctrl.NewManager(cfg, ctrl.Options{Scheme: scheme.Scheme, Logger: logger})
	Expect(err).NotTo(HaveOccurred())

	name := managed.ControllerName(v1alpha1.NicGroupKind)
	metricRecorder := managed.NewMRMetricRecorder()
	stateMetrics := statemetrics.NewMRStateMetrics()
	stateMetricsRecorder := statemetrics.NewMRStateRecorder(mgr.GetClient(), logging.NewLogrLogger(logger), stateMetrics, &v1alpha1.NicList{}, 5*time.Minute)
	err = mgr.Add(stateMetricsRecorder)
	Expect(err).NotTo(HaveOccurred())

	err = ctrl.NewControllerManagedBy(mgr).
		Named(name).
		WithOptions(controller.Options{MaxConcurrentReconciles: 1, RateLimiter: ratelimiter.NewController(), RecoverPanic: ptr.To(true)}).
		For(&v1alpha1.Nic{}).
		Complete(managed.NewReconciler(mgr,
			resource.ManagedKind(v1alpha1.NicGroupVersionKind),
			managed.WithExternalConnecter(&testConnectorNic{service: fakeSvc, ipBlockService: fakeIPSvc, log: logging.NewLogrLogger(logger)}),
			managed.WithReferenceResolver(managed.NewAPISimpleReferenceResolver(mgr.GetClient())),
			managed.WithInitializers(),
			managed.WithPollInterval(1*time.Second),
			managed.WithTimeout(30*time.Second),
			managed.WithCreationGracePeriod(5*time.Second),
			managed.WithLogger(logging.NewLogrLogger(logger).WithValues("controller", name)),
			managed.WithMetricRecorder(metricRecorder),
			managed.WithRecorder(event.NewAPIRecorder(mgr.GetEventRecorderFor(name))),
		))
	Expect(err).NotTo(HaveOccurred())

	go func() {
		defer GinkgoRecover()
		err = mgr.Start(ctx)
		Expect(err).NotTo(HaveOccurred())
	}()
	Eventually(func() bool { return mgr.GetCache().WaitForCacheSync(ctx) }, timeout, interval).Should(BeTrue())
})

var _ = AfterSuite(func() {
	cancel()
	testServer.stop()
	_ = testEnv.Stop()
})

func newNicCR(name string, nicName string, dhcp bool) *v1alpha1.Nic {
	lanID, _ := strconv.Atoi(testLanID)
	return &v1alpha1.Nic{
		ObjectMeta: metav1.ObjectMeta{Name: name},
		Spec: v1alpha1.NicSpec{
			ResourceSpec: xpv1.ResourceSpec{DeletionPolicy: xpv1.DeletionDelete, ManagementPolicies: xpv1.ManagementPolicies{xpv1.ManagementActionAll}},
			ForProvider: v1alpha1.NicParameters{
				DatacenterCfg: v1alpha1.DatacenterConfig{DatacenterID: testDatacenterID},
				ServerCfg:     v1alpha1.ServerConfig{ServerID: testServerID},
				LanCfg:        v1alpha1.LanConfig{LanID: strconv.Itoa(lanID)},
				Dhcp:          dhcp,
				Name:          nicName,
			},
		},
	}
}

func getNicCR(ctx context.Context, name string) (*v1alpha1.Nic, error) {
	cr := &v1alpha1.Nic{}
	return cr, k8sClient.Get(ctx, types.NamespacedName{Name: name}, cr)
}

var _ = Describe("NIC Controller E2E Tests", func() {
	Describe("Scenario 1: Successful creation lifecycle", Ordered, func() {
		var crName string
		BeforeAll(func() { testServer.setMode(statusModeDone); fakeSvc.clearErrors(); crName = "test-nic-create" })
		AfterAll(func() {
			if cr, err := getNicCR(context.Background(), crName); err == nil {
				_ = k8sClient.Delete(context.Background(), cr)
			}
		})
		It("should create a NIC CR and reconcile to AVAILABLE", func() {
			ctx := context.Background()
			Expect(k8sClient.Create(ctx, newNicCR(crName, "test-nic", true))).Should(Succeed())
			Eventually(func(g Gomega) {
				fetched, err := getNicCR(ctx, crName)
				g.Expect(err).NotTo(HaveOccurred())
				g.Expect(meta.GetExternalName(fetched)).NotTo(BeEmpty())
				g.Expect(fetched.Status.AtProvider.NicID).NotTo(BeEmpty())
				g.Expect(fetched.Status.AtProvider.State).To(Equal("AVAILABLE"))
				g.Expect(fetched.Status.GetCondition(xpv1.TypeReady).Equal(xpv1.Available())).To(BeTrue())
			}, timeout, interval).Should(Succeed())
		})
	})

	Describe("Scenario 2: Observe stability after create", Ordered, func() {
		var crName string
		BeforeAll(func() { testServer.setMode(statusModeDone); fakeSvc.clearErrors(); crName = "test-nic-stable" })
		AfterAll(func() {
			if cr, err := getNicCR(context.Background(), crName); err == nil {
				_ = k8sClient.Delete(context.Background(), cr)
			}
		})
		It("should stay stable", func() {
			ctx := context.Background()
			Expect(k8sClient.Create(ctx, newNicCR(crName, "stable-nic", true))).Should(Succeed())
			Eventually(func(g Gomega) {
				fetched, err := getNicCR(ctx, crName)
				g.Expect(err).NotTo(HaveOccurred())
				g.Expect(fetched.Status.AtProvider.State).To(Equal("AVAILABLE"))
			}, timeout, interval).Should(Succeed())
			fetched, _ := getNicCR(ctx, crName)
			gen := fetched.Generation
			Consistently(func(g Gomega) {
				fetched, err := getNicCR(ctx, crName)
				g.Expect(err).NotTo(HaveOccurred())
				g.Expect(fetched.Status.AtProvider.State).To(Equal("AVAILABLE"))
				g.Expect(fetched.Generation).To(Equal(gen))
			}, 5*time.Second, 1*time.Second).Should(Succeed())
		})
	})

	Describe("Scenario 3: Update lifecycle", Ordered, func() {
		var crName string
		BeforeAll(func() { testServer.setMode(statusModeDone); fakeSvc.clearErrors(); crName = "test-nic-update" })
		AfterAll(func() {
			if cr, err := getNicCR(context.Background(), crName); err == nil {
				_ = k8sClient.Delete(context.Background(), cr)
			}
		})
		It("should update the NIC", func() {
			ctx := context.Background()
			Expect(k8sClient.Create(ctx, newNicCR(crName, "update-nic", true))).Should(Succeed())
			Eventually(func(g Gomega) {
				fetched, err := getNicCR(ctx, crName)
				g.Expect(err).NotTo(HaveOccurred())
				g.Expect(fetched.Status.AtProvider.State).To(Equal("AVAILABLE"))
			}, timeout, interval).Should(Succeed())
			Eventually(func(g Gomega) {
				fetched, err := getNicCR(ctx, crName)
				g.Expect(err).NotTo(HaveOccurred())
				fetched.Spec.ForProvider.Name = "updated-nic"
				fetched.Spec.ForProvider.Dhcp = false
				g.Expect(k8sClient.Update(ctx, fetched)).To(Succeed())
			}, timeout, interval).Should(Succeed())
			Eventually(func(g Gomega) {
				fetched, err := getNicCR(ctx, crName)
				g.Expect(err).NotTo(HaveOccurred())
				g.Expect(fetched.Status.AtProvider.State).To(Equal("AVAILABLE"))
			}, timeout, interval).Should(Succeed())
		})
	})

	Describe("Scenario 4: Delete lifecycle", Ordered, func() {
		var crName string
		BeforeAll(func() { testServer.setMode(statusModeDone); fakeSvc.clearErrors(); crName = "test-nic-delete" })
		It("should delete the NIC CR", func() {
			ctx := context.Background()
			Expect(k8sClient.Create(ctx, newNicCR(crName, "delete-nic", true))).Should(Succeed())
			Eventually(func(g Gomega) {
				fetched, err := getNicCR(ctx, crName)
				g.Expect(err).NotTo(HaveOccurred())
				g.Expect(fetched.Status.AtProvider.State).To(Equal("AVAILABLE"))
			}, timeout, interval).Should(Succeed())
			fetched, _ := getNicCR(ctx, crName)
			Expect(k8sClient.Delete(ctx, fetched)).Should(Succeed())
			Eventually(func() bool { _, err := getNicCR(ctx, crName); return err != nil }, timeout, interval).Should(BeTrue())
			Expect(len(fakeSvc.getDeleteCalls())).To(BeNumerically(">", 0))
		})
	})

	Describe("Scenario 5: Create error — API returns error", Ordered, func() {
		var crName string
		BeforeAll(func() { testServer.setMode(statusModeDone); crName = "test-nic-create-err" })
		AfterAll(func() {
			fakeSvc.clearErrors()
			if cr, err := getNicCR(context.Background(), crName); err == nil {
				_ = k8sClient.Delete(context.Background(), cr)
				Eventually(func() bool { _, err := getNicCR(context.Background(), crName); return err != nil }, timeout, interval).Should(BeTrue())
			}
		})
		It("should fail then recover", func() {
			ctx := context.Background()
			fakeSvc.setError("create", fmt.Errorf("simulated create error"))
			Expect(k8sClient.Create(ctx, newNicCR(crName, "error-nic", true))).Should(Succeed())
			Consistently(func(g Gomega) {
				fetched, err := getNicCR(ctx, crName)
				g.Expect(err).NotTo(HaveOccurred())
				g.Expect(fetched.Status.AtProvider.State).NotTo(Equal("AVAILABLE"))
			}, 5*time.Second, 1*time.Second).Should(Succeed())
			fakeSvc.clearErrors()
			Eventually(func(g Gomega) {
				fetched, err := getNicCR(ctx, crName)
				g.Expect(err).NotTo(HaveOccurred())
				g.Expect(fetched.Status.AtProvider.State).To(Equal("AVAILABLE"))
			}, timeout, interval).Should(Succeed())
		})
	})

	Describe("Scenario 6: WaitForRequest error during Create", Ordered, func() {
		var crName string
		BeforeAll(func() { crName = "test-nic-waitreq-err"; fakeSvc.clearErrors() })
		AfterAll(func() {
			testServer.setMode(statusModeDone); fakeSvc.clearErrors()
			if cr, err := getNicCR(context.Background(), crName); err == nil {
				_ = k8sClient.Delete(context.Background(), cr)
				Eventually(func() bool { _, err := getNicCR(context.Background(), crName); return err != nil }, timeout, interval).Should(BeTrue())
			}
		})
		It("should recover via Observe", func() {
			ctx := context.Background()
			testServer.setMode(statusModeRunning)
			Expect(k8sClient.Create(ctx, newNicCR(crName, "waitreq-nic", true))).Should(Succeed())
			Eventually(func(g Gomega) {
				fetched, err := getNicCR(ctx, crName)
				g.Expect(err).NotTo(HaveOccurred())
				g.Expect(meta.GetExternalName(fetched)).NotTo(BeEmpty())
			}, timeout, interval).Should(Succeed())
			testServer.setMode(statusModeDone)
			Eventually(func(g Gomega) {
				fetched, err := getNicCR(ctx, crName)
				g.Expect(err).NotTo(HaveOccurred())
				g.Expect(fetched.Status.AtProvider.State).To(Equal("AVAILABLE"))
			}, timeout, interval).Should(Succeed())
		})
	})

	Describe("Scenario 7: IsRequestDoneNEW — request still running", Ordered, func() {
		var crName string
		const nicID = "prereq-running-1"
		BeforeAll(func() { crName = "test-nic-isreqdone-running"; fakeSvc.clearErrors() })
		AfterAll(func() {
			testServer.setMode(statusModeDone); fakeSvc.clearErrors()
			if cr, err := getNicCR(context.Background(), crName); err == nil {
				_ = k8sClient.Delete(context.Background(), cr)
				Eventually(func() bool { _, err := getNicCR(context.Background(), crName); return err != nil }, timeout, interval).Should(BeTrue())
			}
		})
		It("should wait for request then reconcile", func() {
			ctx := context.Background()
			testServer.setMode(statusModeRunning)
			cr := newNicCR(crName, "isreqdone-nic", true)
			cr.Annotations = map[string]string{compute.POSTRequestIDAnnotationKey: "simulated-post-req-1"}
			meta.SetExternalName(cr, nicID)
			Expect(k8sClient.Create(ctx, cr)).Should(Succeed())
			Consistently(func(g Gomega) {
				fetched, err := getNicCR(ctx, crName)
				g.Expect(err).NotTo(HaveOccurred())
				g.Expect(fetched.Status.AtProvider.State).NotTo(Equal("AVAILABLE"))
			}, 5*time.Second, 500*time.Millisecond).Should(Succeed())
			testServer.setMode(statusModeDone)
			lanInt, _ := strconv.Atoi(testLanID)
			fakeSvc.storeNic(testDatacenterID, testServerID, nicID, sdkgo.Nic{
				Id:         ptr.To(nicID),
				Properties: &sdkgo.NicProperties{Name: ptr.To("isreqdone-nic"), Dhcp: ptr.To(true), Lan: ptr.To(int32(lanInt))},
				Metadata:   &sdkgo.DatacenterElementMetadata{State: ptr.To("AVAILABLE")},
			})
			Eventually(func(g Gomega) {
				fetched, err := getNicCR(ctx, crName)
				g.Expect(err).NotTo(HaveOccurred())
				g.Expect(fetched.Status.AtProvider.State).To(Equal("AVAILABLE"))
			}, timeout, interval).Should(Succeed())
		})
	})

	Describe("Scenario 8: IsRequestDoneNEW — request failed", Ordered, func() {
		var crName string
		BeforeAll(func() { crName = "test-nic-isreqdone-failed"; fakeSvc.clearErrors() })
		AfterAll(func() {
			testServer.setMode(statusModeDone); fakeSvc.clearErrors()
			if cr, err := getNicCR(context.Background(), crName); err == nil {
				_ = k8sClient.Delete(context.Background(), cr)
				Eventually(func() bool { _, err := getNicCR(context.Background(), crName); return err != nil }, timeout, interval).Should(BeTrue())
			}
		})
		It("should propagate error when FAILED", func() {
			ctx := context.Background()
			testServer.setMode(statusModeFailed)
			cr := newNicCR(crName, "failed-nic", true)
			cr.Annotations = map[string]string{compute.POSTRequestIDAnnotationKey: "simulated-post-req-failed"}
			meta.SetExternalName(cr, "prereq-failed-1")
			Expect(k8sClient.Create(ctx, cr)).Should(Succeed())
			Eventually(func(g Gomega) {
				fetched, err := getNicCR(ctx, crName)
				g.Expect(err).NotTo(HaveOccurred())
				g.Expect(fetched.Status.GetCondition(xpv1.TypeSynced).Status).To(Equal(corev1.ConditionFalse))
			}, timeout, interval).Should(Succeed())
			Consistently(func(g Gomega) {
				fetched, err := getNicCR(ctx, crName)
				g.Expect(err).NotTo(HaveOccurred())
				g.Expect(fetched.Status.GetCondition(xpv1.TypeReady).Equal(xpv1.Available())).To(BeFalse())
			}, 3*time.Second, 500*time.Millisecond).Should(Succeed())
		})
	})

	Describe("Scenario 9: IsRequestDoneNEW — 404 lost request", Ordered, func() {
		var crName string
		BeforeAll(func() { crName = "test-nic-isreqdone-404"; fakeSvc.clearErrors() })
		AfterAll(func() {
			testServer.setMode(statusModeDone); fakeSvc.clearErrors()
			if cr, err := getNicCR(context.Background(), crName); err == nil {
				_ = k8sClient.Delete(context.Background(), cr)
				Eventually(func() bool { _, err := getNicCR(context.Background(), crName); return err != nil }, timeout, interval).Should(BeTrue())
			}
		})
		It("should recover after annotation removal", func() {
			ctx := context.Background()
			testServer.setMode(statusMode404)
			cr := newNicCR(crName, "lost-req-nic", true)
			cr.Annotations = map[string]string{compute.POSTRequestIDAnnotationKey: "simulated-post-req-lost"}
			Expect(k8sClient.Create(ctx, cr)).Should(Succeed())
			Consistently(func(g Gomega) {
				fetched, err := getNicCR(ctx, crName)
				g.Expect(err).NotTo(HaveOccurred())
				g.Expect(fetched.Status.AtProvider.State).NotTo(Equal("AVAILABLE"))
			}, 5*time.Second, 1*time.Second).Should(Succeed())
			testServer.setMode(statusModeDone)
			Eventually(func(g Gomega) {
				fetched, err := getNicCR(ctx, crName)
				g.Expect(err).NotTo(HaveOccurred())
				annotations := fetched.GetAnnotations()
				delete(annotations, compute.POSTRequestIDAnnotationKey)
				fetched.SetAnnotations(annotations)
				g.Expect(k8sClient.Update(ctx, fetched)).To(Succeed())
			}, timeout, interval).Should(Succeed())
			Eventually(func(g Gomega) {
				fetched, err := getNicCR(ctx, crName)
				g.Expect(err).NotTo(HaveOccurred())
				g.Expect(fetched.Status.AtProvider.State).To(Equal("AVAILABLE"))
			}, timeout, interval).Should(Succeed())
		})
	})

	Describe("Scenario 10: Delete when API returns 404", Ordered, func() {
		var crName string
		BeforeAll(func() { testServer.setMode(statusModeDone); fakeSvc.clearErrors(); crName = "test-nic-delete-404" })
		AfterAll(func() { fakeSvc.clearErrors() })
		It("should handle 404 gracefully on delete", func() {
			ctx := context.Background()
			Expect(k8sClient.Create(ctx, newNicCR(crName, "delete-404-nic", true))).Should(Succeed())
			Eventually(func(g Gomega) {
				fetched, err := getNicCR(ctx, crName)
				g.Expect(err).NotTo(HaveOccurred())
				g.Expect(fetched.Status.AtProvider.State).To(Equal("AVAILABLE"))
			}, timeout, interval).Should(Succeed())
			fetched, _ := getNicCR(ctx, crName)
			nicID := meta.GetExternalName(fetched)
			fakeSvc.removeNic(testDatacenterID, testServerID, nicID)
			fakeSvc.setError("delete", fmt.Errorf("nic not found"))
			Expect(k8sClient.Delete(ctx, fetched)).Should(Succeed())
			Eventually(func() bool { _, err := getNicCR(ctx, crName); return err != nil }, timeout, interval).Should(BeTrue())
		})
	})
})
