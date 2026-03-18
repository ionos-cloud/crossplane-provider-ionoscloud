//go:build lan_mock

package lan

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
	lanClient "github.com/ionos-cloud/crossplane-provider-ionoscloud/internal/clients/compute/lan"
)

// ---------------------------------------------------------------------------
// Test HTTP server behavior modes
// ---------------------------------------------------------------------------

type requestStatusMode int32

const (
	statusModeDone    requestStatusMode = iota // return DONE
	statusModeRunning                          // return RUNNING
	statusModeFailed                           // return FAILED with message
	statusModeError                            // return HTTP 500
	statusMode404                              // return HTTP 404
)

// ---------------------------------------------------------------------------
// Configurable test HTTP server
// ---------------------------------------------------------------------------

type testHTTPServer struct {
	server *httptest.Server
	mode   atomic.Int32 // stores requestStatusMode
}

func newTestHTTPServer() *testHTTPServer {
	ts := &testHTTPServer{}
	mux := http.NewServeMux()

	// Handle request status endpoint: /cloudapi/v6/requests/{id}/status
	mux.HandleFunc("/cloudapi/v6/requests/", func(w http.ResponseWriter, r *http.Request) {
		mode := requestStatusMode(ts.mode.Load())
		w.Header().Set("Content-Type", "application/json")
		switch mode {
		case statusModeDone:
			status := "DONE"
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"id":   "test-request-id",
				"type": "request-status",
				"metadata": map[string]interface{}{
					"status": status,
				},
			})
		case statusModeRunning:
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"id":   "test-request-id",
				"type": "request-status",
				"metadata": map[string]interface{}{
					"status": "RUNNING",
				},
			})
		case statusModeFailed:
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"id":   "test-request-id",
				"type": "request-status",
				"metadata": map[string]interface{}{
					"status":  "FAILED",
					"message": "simulated failure",
				},
			})
		case statusModeError:
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"error": "internal server error",
			})
		case statusMode404:
			w.WriteHeader(http.StatusNotFound)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"error": "not found",
			})
		}
	})

	ts.server = httptest.NewServer(mux)
	return ts
}

func (ts *testHTTPServer) setMode(mode requestStatusMode) {
	ts.mode.Store(int32(mode))
}

func (ts *testHTTPServer) stop() {
	ts.server.Close()
}

// ---------------------------------------------------------------------------
// Fake LAN service implementing lan.Client
// ---------------------------------------------------------------------------

type storedLan struct {
	lan          sdkgo.Lan
	datacenterID string
}

type fakeLanService struct {
	mu   sync.Mutex
	lans map[string]storedLan // key: "datacenterID/lanID"
	nextID int

	// Configurable per-method errors
	createErr error
	getErr    error
	updateErr error
	deleteErr error

	// Track calls for verification
	createCalls []sdkgo.Lan
	updateCalls []sdkgo.LanProperties
	deleteCalls []string // lanIDs

	apiClient *sdkgo.APIClient
	serverURL string
}

func newFakeLanService(serverURL string) *fakeLanService {
	cfg := sdkgo.NewConfiguration("", "", "test-token", serverURL)
	cfg.PollInterval = 100 * time.Millisecond
	apiClient := sdkgo.NewAPIClient(cfg)

	return &fakeLanService{
		lans:      make(map[string]storedLan),
		nextID:    1,
		apiClient: apiClient,
		serverURL: serverURL,
	}
}

func (f *fakeLanService) key(datacenterID, lanID string) string {
	return datacenterID + "/" + lanID
}

func (f *fakeLanService) CheckDuplicateLan(_ context.Context, datacenterID, lanName string) (*sdkgo.Lan, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
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
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.getErr != nil {
		return sdkgo.Lan{}, &sdkgo.APIResponse{
			Response: &http.Response{StatusCode: http.StatusInternalServerError},
		}, f.getErr
	}
	sl, ok := f.lans[f.key(datacenterID, lanID)]
	if !ok {
		return sdkgo.Lan{}, &sdkgo.APIResponse{
			Response: &http.Response{StatusCode: http.StatusNotFound},
		}, fmt.Errorf("lan %s not found", lanID)
	}
	return sl.lan, &sdkgo.APIResponse{
		Response: &http.Response{StatusCode: http.StatusOK},
	}, nil
}

func (f *fakeLanService) GetLanIPFailovers(_ context.Context, datacenterID, lanID string) ([]sdkgo.IPFailover, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
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
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.createErr != nil {
		return sdkgo.Lan{}, &sdkgo.APIResponse{
			Response: &http.Response{StatusCode: http.StatusInternalServerError},
		}, f.createErr
	}

	f.createCalls = append(f.createCalls, l)

	lanID := strconv.Itoa(f.nextID)
	f.nextID++

	// Build clean properties — avoid passing through SDK explicit-nil sentinels
	props := &sdkgo.LanProperties{}
	if l.Properties != nil {
		if l.Properties.Name != nil {
			props.Name = l.Properties.Name
		}
		if l.Properties.Public != nil {
			props.Public = l.Properties.Public
		}
		// Only set Ipv6CidrBlock if it was explicitly set to a real value (not explicit nil)
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
		Metadata: &sdkgo.DatacenterElementMetadata{
			State: ptr.To("AVAILABLE"),
		},
	}
	f.lans[f.key(datacenterID, lanID)] = storedLan{lan: newLan, datacenterID: datacenterID}

	// Build response with Location header pointing to test server
	header := http.Header{}
	requestURL := f.serverURL + "/cloudapi/v6/requests/create-req-" + lanID + "/status"
	header.Set("Location", requestURL)
	resp := &http.Response{
		StatusCode: http.StatusAccepted,
		Header:     header,
	}

	return newLan, &sdkgo.APIResponse{Response: resp}, nil
}

func (f *fakeLanService) UpdateLan(_ context.Context, datacenterID, lanID string, props sdkgo.LanProperties) (sdkgo.Lan, *sdkgo.APIResponse, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.updateErr != nil {
		return sdkgo.Lan{}, &sdkgo.APIResponse{
			Response: &http.Response{StatusCode: http.StatusInternalServerError},
		}, f.updateErr
	}

	f.updateCalls = append(f.updateCalls, props)

	sl, ok := f.lans[f.key(datacenterID, lanID)]
	if !ok {
		return sdkgo.Lan{}, &sdkgo.APIResponse{
			Response: &http.Response{StatusCode: http.StatusNotFound},
		}, fmt.Errorf("lan %s not found", lanID)
	}

	// Apply update
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

	header := http.Header{}
	requestURL := f.serverURL + "/cloudapi/v6/requests/update-req-" + lanID + "/status"
	header.Set("Location", requestURL)
	resp := &http.Response{
		StatusCode: http.StatusAccepted,
		Header:     header,
	}

	return sl.lan, &sdkgo.APIResponse{Response: resp}, nil
}

func (f *fakeLanService) DeleteLan(_ context.Context, datacenterID, lanID string) (*sdkgo.APIResponse, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.deleteErr != nil {
		resp := &sdkgo.APIResponse{
			Response: &http.Response{StatusCode: http.StatusNotFound},
		}
		return resp, f.deleteErr
	}

	f.deleteCalls = append(f.deleteCalls, lanID)
	delete(f.lans, f.key(datacenterID, lanID))

	header := http.Header{}
	requestURL := f.serverURL + "/cloudapi/v6/requests/delete-req-" + lanID + "/status"
	header.Set("Location", requestURL)
	resp := &http.Response{
		StatusCode: http.StatusAccepted,
		Header:     header,
	}

	return &sdkgo.APIResponse{Response: resp}, nil
}

func (f *fakeLanService) GetAPIClient() *sdkgo.APIClient {
	return f.apiClient
}

// Helper methods for tests

func (f *fakeLanService) setError(method string, err error) {
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

func (f *fakeLanService) clearErrors() {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.createErr = nil
	f.getErr = nil
	f.updateErr = nil
	f.deleteErr = nil
}

func (f *fakeLanService) getLan(datacenterID, lanID string) (sdkgo.Lan, bool) {
	f.mu.Lock()
	defer f.mu.Unlock()
	sl, ok := f.lans[f.key(datacenterID, lanID)]
	return sl.lan, ok
}

func (f *fakeLanService) storeLan(datacenterID, lanID string, l sdkgo.Lan) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.lans[f.key(datacenterID, lanID)] = storedLan{lan: l, datacenterID: datacenterID}
}

func (f *fakeLanService) removeLan(datacenterID, lanID string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	delete(f.lans, f.key(datacenterID, lanID))
}

func (f *fakeLanService) getUpdateCalls() []sdkgo.LanProperties {
	f.mu.Lock()
	defer f.mu.Unlock()
	result := make([]sdkgo.LanProperties, len(f.updateCalls))
	copy(result, f.updateCalls)
	return result
}

func (f *fakeLanService) getDeleteCalls() []string {
	f.mu.Lock()
	defer f.mu.Unlock()
	result := make([]string, len(f.deleteCalls))
	copy(result, f.deleteCalls)
	return result
}

// ---------------------------------------------------------------------------
// Test connector: injects fake service into the reconciler
// ---------------------------------------------------------------------------

type testConnectorLan struct {
	service lanClient.Client
	log     logging.Logger
}

func (c *testConnectorLan) Connect(_ context.Context, _ resource.Managed) (managed.ExternalClient, error) {
	return &externalLan{
		service: c.service,
		log:     c.log,
	}, nil
}

// ---------------------------------------------------------------------------
// Test globals
// ---------------------------------------------------------------------------

var (
	k8sClient  client.Client
	cancel     context.CancelFunc
	testEnv    *envtest.Environment
	fakeSvc    *fakeLanService
	testServer *testHTTPServer
)

var logger = zap.New(zap.UseDevMode(true))

const (
	timeout  = 60 * time.Second
	interval = 500 * time.Millisecond

	testDatacenterID = "a1b2c3d4-e5f6-7890-abcd-ef1234567890"
)

func TestLanController_Mock(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "LAN Controller Mock Suite")
}

var _ = BeforeSuite(func() {
	logf.SetLogger(logger)
	var ctx context.Context
	ctx, cancel = context.WithCancel(context.Background())

	By("starting test HTTP server")
	testServer = newTestHTTPServer()
	testServer.setMode(statusModeDone)

	By("creating fake LAN service")
	fakeSvc = newFakeLanService(testServer.server.URL)

	By("bootstrapping test environment")
	testEnv = &envtest.Environment{
		CRDDirectoryPaths: []string{
			filepath.Join("..", "..", "..", "..", "package", "crds"),
		},
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
	Expect(k8sClient).NotTo(BeNil())

	By("creating controller manager")
	ctrl.SetLogger(logger)
	mgr, err := ctrl.NewManager(cfg, ctrl.Options{
		Scheme: scheme.Scheme,
		Logger: logger,
	})
	Expect(err).NotTo(HaveOccurred())

	By("registering LAN controller with fake connector")
	name := managed.ControllerName(v1alpha1.LanGroupKind)
	metricRecorder := managed.NewMRMetricRecorder()
	stateMetrics := statemetrics.NewMRStateMetrics()

	stateMetricsRecorder := statemetrics.NewMRStateRecorder(
		mgr.GetClient(), logging.NewLogrLogger(logger),
		stateMetrics, &v1alpha1.LanList{}, 5*time.Minute,
	)
	err = mgr.Add(stateMetricsRecorder)
	Expect(err).NotTo(HaveOccurred())

	err = ctrl.NewControllerManagedBy(mgr).
		Named(name).
		WithOptions(controller.Options{
			MaxConcurrentReconciles: 1,
			RateLimiter:             ratelimiter.NewController(),
			RecoverPanic:            ptr.To(true),
		}).
		For(&v1alpha1.Lan{}).
		Complete(managed.NewReconciler(mgr,
			resource.ManagedKind(v1alpha1.LanGroupVersionKind),
			managed.WithExternalConnecter(&testConnectorLan{
				service: fakeSvc,
				log:     logging.NewLogrLogger(logger),
			}),
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

	By("starting controller manager")
	go func() {
		defer GinkgoRecover()
		err = mgr.Start(ctx)
		Expect(err).NotTo(HaveOccurred())
	}()

	Eventually(func() bool {
		return mgr.GetCache().WaitForCacheSync(ctx)
	}, timeout, interval).Should(BeTrue())
})

var _ = AfterSuite(func() {
	cancel()
	By("stopping test HTTP server")
	testServer.stop()
	By("tearing down the test environment")
	_ = testEnv.Stop()
})

// ---------------------------------------------------------------------------
// Helper: create a LAN CR
// ---------------------------------------------------------------------------

func newLanCR(name string, public bool, lanName string) *v1alpha1.Lan {
	return &v1alpha1.Lan{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
		Spec: v1alpha1.LanSpec{
			ResourceSpec: xpv1.ResourceSpec{
				// No ProviderConfigReference needed — test connector ignores it
				DeletionPolicy:          xpv1.DeletionDelete,
				ManagementPolicies:      xpv1.ManagementPolicies{xpv1.ManagementActionAll},
			},
			ForProvider: v1alpha1.LanParameters{
				DatacenterCfg: v1alpha1.DatacenterConfig{
					DatacenterID: testDatacenterID,
				},
				Name:   lanName,
				Public: public,
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

	// Scenario 1: Successful creation lifecycle
	Describe("Scenario 1: Successful creation lifecycle", Ordered, func() {
		var crName string

		BeforeAll(func() {
			testServer.setMode(statusModeDone)
			fakeSvc.clearErrors()
			crName = "test-lan-create"
		})

		AfterAll(func() {
			cr, err := getLanCR(context.Background(), crName)
			if err == nil {
				_ = k8sClient.Delete(context.Background(), cr)
			}
		})

		It("should create a LAN CR and reconcile to AVAILABLE", func() {
			ctx := context.Background()
			cr := newLanCR(crName, true, "test-lan")
			Expect(k8sClient.Create(ctx, cr)).Should(Succeed())

			Eventually(func(g Gomega) {
				fetched, err := getLanCR(ctx, crName)
				g.Expect(err).NotTo(HaveOccurred())
				g.Expect(meta.GetExternalName(fetched)).NotTo(BeEmpty())
				g.Expect(fetched.Status.AtProvider.LanID).NotTo(BeEmpty())
				g.Expect(fetched.Status.AtProvider.State).To(Equal("AVAILABLE"))
				g.Expect(fetched.Status.GetCondition(xpv1.TypeReady).Equal(xpv1.Available())).To(BeTrue())
			}, timeout, interval).Should(Succeed())
		})
	})

	// Scenario 2: Observe stability after create
	Describe("Scenario 2: Observe stability after create", Ordered, func() {
		var crName string

		BeforeAll(func() {
			testServer.setMode(statusModeDone)
			fakeSvc.clearErrors()
			crName = "test-lan-stable"
		})

		AfterAll(func() {
			cr, err := getLanCR(context.Background(), crName)
			if err == nil {
				_ = k8sClient.Delete(context.Background(), cr)
			}
		})

		It("should stay stable after creation", func() {
			ctx := context.Background()
			cr := newLanCR(crName, true, "stable-lan")
			Expect(k8sClient.Create(ctx, cr)).Should(Succeed())

			// Wait for it to become available
			Eventually(func(g Gomega) {
				fetched, err := getLanCR(ctx, crName)
				g.Expect(err).NotTo(HaveOccurred())
				g.Expect(fetched.Status.AtProvider.State).To(Equal("AVAILABLE"))
			}, timeout, interval).Should(Succeed())

			// Record the generation
			fetched, err := getLanCR(ctx, crName)
			Expect(err).NotTo(HaveOccurred())
			gen := fetched.Generation

			// Wait a bit and verify it stays stable
			Consistently(func(g Gomega) {
				fetched, err := getLanCR(ctx, crName)
				g.Expect(err).NotTo(HaveOccurred())
				g.Expect(fetched.Status.AtProvider.State).To(Equal("AVAILABLE"))
				g.Expect(fetched.Generation).To(Equal(gen))
			}, 5*time.Second, 1*time.Second).Should(Succeed())
		})
	})

	// Scenario 3: Update lifecycle
	Describe("Scenario 3: Update lifecycle", Ordered, func() {
		var crName string

		BeforeAll(func() {
			testServer.setMode(statusModeDone)
			fakeSvc.clearErrors()
			crName = "test-lan-update"
		})

		AfterAll(func() {
			cr, err := getLanCR(context.Background(), crName)
			if err == nil {
				_ = k8sClient.Delete(context.Background(), cr)
			}
		})

		It("should update the LAN when spec changes", func() {
			ctx := context.Background()
			cr := newLanCR(crName, true, "update-lan")
			Expect(k8sClient.Create(ctx, cr)).Should(Succeed())

			// Wait for creation to complete
			Eventually(func(g Gomega) {
				fetched, err := getLanCR(ctx, crName)
				g.Expect(err).NotTo(HaveOccurred())
				g.Expect(fetched.Status.AtProvider.State).To(Equal("AVAILABLE"))
			}, timeout, interval).Should(Succeed())

			// Now update the spec
			By("updating LAN name and public flag")
			Eventually(func(g Gomega) {
				fetched, err := getLanCR(ctx, crName)
				g.Expect(err).NotTo(HaveOccurred())
				fetched.Spec.ForProvider.Name = "updated-lan"
				fetched.Spec.ForProvider.Public = false
				g.Expect(k8sClient.Update(ctx, fetched)).To(Succeed())
			}, timeout, interval).Should(Succeed())

			// Wait for update to complete
			Eventually(func(g Gomega) {
				fetched, err := getLanCR(ctx, crName)
				g.Expect(err).NotTo(HaveOccurred())
				g.Expect(fetched.Status.AtProvider.State).To(Equal("AVAILABLE"))
				g.Expect(fetched.Status.AtProvider.Name).To(Equal("updated-lan"))
			}, timeout, interval).Should(Succeed())

			// Verify the fake received UpdateLan with the correct params
			updateCalls := fakeSvc.getUpdateCalls()
			Expect(len(updateCalls)).To(BeNumerically(">", 0))
			lastUpdate := updateCalls[len(updateCalls)-1]
			Expect(*lastUpdate.Name).To(Equal("updated-lan"))
			Expect(*lastUpdate.Public).To(BeFalse())
		})
	})

	// Scenario 4: Delete lifecycle
	Describe("Scenario 4: Delete lifecycle", Ordered, func() {
		var crName string

		BeforeAll(func() {
			testServer.setMode(statusModeDone)
			fakeSvc.clearErrors()
			crName = "test-lan-delete"
		})

		It("should delete the LAN CR and remove it from K8s", func() {
			ctx := context.Background()
			cr := newLanCR(crName, true, "delete-lan")
			Expect(k8sClient.Create(ctx, cr)).Should(Succeed())

			// Wait for creation to complete
			Eventually(func(g Gomega) {
				fetched, err := getLanCR(ctx, crName)
				g.Expect(err).NotTo(HaveOccurred())
				g.Expect(fetched.Status.AtProvider.State).To(Equal("AVAILABLE"))
			}, timeout, interval).Should(Succeed())

			// Delete the CR
			By("deleting the LAN CR")
			fetched, err := getLanCR(ctx, crName)
			Expect(err).NotTo(HaveOccurred())
			Expect(k8sClient.Delete(ctx, fetched)).Should(Succeed())

			// Wait for it to be gone
			Eventually(func() bool {
				_, err := getLanCR(ctx, crName)
				return err != nil
			}, timeout, interval).Should(BeTrue())

			// Verify fake received delete
			deleteCalls := fakeSvc.getDeleteCalls()
			Expect(len(deleteCalls)).To(BeNumerically(">", 0))
		})
	})

	// Scenario 5: Create error — API returns error on CreateLan
	Describe("Scenario 5: Create error — API returns error", Ordered, func() {
		var crName string

		BeforeAll(func() {
			testServer.setMode(statusModeDone)
			crName = "test-lan-create-err"
		})

		AfterAll(func() {
			fakeSvc.clearErrors()
			cr, err := getLanCR(context.Background(), crName)
			if err == nil {
				_ = k8sClient.Delete(context.Background(), cr)
				Eventually(func() bool {
					_, err := getLanCR(context.Background(), crName)
					return err != nil
				}, timeout, interval).Should(BeTrue())
			}
		})

		It("should fail then recover when error is cleared", func() {
			ctx := context.Background()
			fakeSvc.setError("create", fmt.Errorf("simulated create error"))

			cr := newLanCR(crName, true, "error-lan")
			Expect(k8sClient.Create(ctx, cr)).Should(Succeed())

			// CR should NOT reach Available
			Consistently(func(g Gomega) {
				fetched, err := getLanCR(ctx, crName)
				g.Expect(err).NotTo(HaveOccurred())
				g.Expect(fetched.Status.AtProvider.State).NotTo(Equal("AVAILABLE"))
			}, 5*time.Second, 1*time.Second).Should(Succeed())

			// Clear error → reconciler retries → should reach Available
			By("clearing the create error")
			fakeSvc.clearErrors()

			Eventually(func(g Gomega) {
				fetched, err := getLanCR(ctx, crName)
				g.Expect(err).NotTo(HaveOccurred())
				g.Expect(fetched.Status.AtProvider.State).To(Equal("AVAILABLE"))
				g.Expect(fetched.Status.GetCondition(xpv1.TypeReady).Equal(xpv1.Available())).To(BeTrue())
			}, timeout, interval).Should(Succeed())
		})
	})

	// Scenario 6: WaitForRequest timeout/error during Create
	Describe("Scenario 6: WaitForRequest error during Create", Ordered, func() {
		var crName string

		BeforeAll(func() {
			crName = "test-lan-waitreq-err"
			fakeSvc.clearErrors()
		})

		AfterAll(func() {
			testServer.setMode(statusModeDone)
			fakeSvc.clearErrors()
			cr, err := getLanCR(context.Background(), crName)
			if err == nil {
				_ = k8sClient.Delete(context.Background(), cr)
				Eventually(func() bool {
					_, err := getLanCR(context.Background(), crName)
					return err != nil
				}, timeout, interval).Should(BeTrue())
			}
		})

		It("should recover via Observe after WaitForRequest fails", func() {
			ctx := context.Background()

			// Set HTTP server to return RUNNING — WaitForRequest will poll and eventually the
			// reconciler context will timeout, causing an error.
			testServer.setMode(statusModeRunning)

			cr := newLanCR(crName, true, "waitreq-lan")
			Expect(k8sClient.Create(ctx, cr)).Should(Succeed())

			// Wait for the LAN to be stored in the fake (CreateLan succeeded, WaitForRequest will fail)
			Eventually(func(g Gomega) {
				fetched, err := getLanCR(ctx, crName)
				g.Expect(err).NotTo(HaveOccurred())
				g.Expect(meta.GetExternalName(fetched)).NotTo(BeEmpty())
			}, timeout, interval).Should(Succeed())

			// Now switch to DONE — on next reconcile, Observe should find the LAN
			By("switching HTTP server to DONE")
			testServer.setMode(statusModeDone)

			Eventually(func(g Gomega) {
				fetched, err := getLanCR(ctx, crName)
				g.Expect(err).NotTo(HaveOccurred())
				g.Expect(fetched.Status.AtProvider.State).To(Equal("AVAILABLE"))
				g.Expect(fetched.Status.GetCondition(xpv1.TypeReady).Equal(xpv1.Available())).To(BeTrue())
			}, timeout, interval).Should(Succeed())
		})
	})

	// Scenario 7: IsRequestDone — request still running
	// Simulates: a previous Create set external name + annotation, but the cloud
	// resource hasn't materialized yet (Observe returns 404 → ResourceExists: false).
	// Create sees the annotation, calls IsRequestDone, which returns RUNNING.
	// Once the request is DONE and the resource appears, CR reaches AVAILABLE.
	Describe("Scenario 7: IsRequestDone — request still running", Ordered, func() {
		var crName string
		const lanID = "prereq-running-1"

		BeforeAll(func() {
			crName = "test-lan-isreqdone-running"
			fakeSvc.clearErrors()
		})

		AfterAll(func() {
			testServer.setMode(statusModeDone)
			fakeSvc.clearErrors()
			cr, err := getLanCR(context.Background(), crName)
			if err == nil {
				_ = k8sClient.Delete(context.Background(), cr)
				Eventually(func() bool {
					_, err := getLanCR(context.Background(), crName)
					return err != nil
				}, timeout, interval).Should(BeTrue())
			}
		})

		It("should wait for request then reconcile successfully", func() {
			ctx := context.Background()

			// Do NOT store the LAN in the fake — Observe will return 404
			// Set HTTP server to return RUNNING for IsRequestDone
			testServer.setMode(statusModeRunning)

			// Create CR with external name + annotation pre-set
			// (simulates a previous Create that set both, but resource not yet visible)
			cr := newLanCR(crName, true, "isreqdone-lan")
			cr.Annotations = map[string]string{
				compute.POSTRequestIDAnnotationKey: "simulated-post-req-1",
			}
			meta.SetExternalName(cr, lanID)
			Expect(k8sClient.Create(ctx, cr)).Should(Succeed())

			// Observe → 404 → ResourceExists: false → Create → annotation found →
			// IsRequestDone returns false → no-op. CR stays not Available.
			Consistently(func(g Gomega) {
				fetched, err := getLanCR(ctx, crName)
				g.Expect(err).NotTo(HaveOccurred())
				g.Expect(fetched.Status.AtProvider.State).NotTo(Equal("AVAILABLE"))
			}, 5*time.Second, 500*time.Millisecond).Should(Succeed())

			// Switch to DONE and store the LAN (simulating cloud resource is now visible)
			By("switching HTTP server to DONE and materializing the LAN")
			testServer.setMode(statusModeDone)
			fakeSvc.storeLan(testDatacenterID, lanID, sdkgo.Lan{
				Id: ptr.To(lanID),
				Properties: &sdkgo.LanProperties{
					Name:   ptr.To("isreqdone-lan"),
					Public: ptr.To(true),
				},
				Metadata: &sdkgo.DatacenterElementMetadata{
					State: ptr.To("AVAILABLE"),
				},
			})

			// Now: Create → isDone=true → success → Observe finds LAN → AVAILABLE
			Eventually(func(g Gomega) {
				fetched, err := getLanCR(ctx, crName)
				g.Expect(err).NotTo(HaveOccurred())
				g.Expect(fetched.Status.AtProvider.State).To(Equal("AVAILABLE"))
				g.Expect(fetched.Status.GetCondition(xpv1.TypeReady).Equal(xpv1.Available())).To(BeTrue())
			}, timeout, interval).Should(Succeed())
		})
	})

	// Scenario 8: IsRequestDone — request failed
	// Simulates: a previous Create set external name + annotation, resource not yet visible.
	// Observe returns 404 → Create sees annotation → IsRequestDone returns FAILED with error.
	Describe("Scenario 8: IsRequestDone — request failed", Ordered, func() {
		var crName string

		BeforeAll(func() {
			crName = "test-lan-isreqdone-failed"
			fakeSvc.clearErrors()
		})

		AfterAll(func() {
			testServer.setMode(statusModeDone)
			fakeSvc.clearErrors()
			cr, err := getLanCR(context.Background(), crName)
			if err == nil {
				_ = k8sClient.Delete(context.Background(), cr)
				Eventually(func() bool {
					_, err := getLanCR(context.Background(), crName)
					return err != nil
				}, timeout, interval).Should(BeTrue())
			}
		})

		It("should propagate error when request status is FAILED", func() {
			ctx := context.Background()

			// Do NOT store the LAN — Observe returns 404 → ResourceExists: false → Create is called
			// Set HTTP server to return FAILED
			testServer.setMode(statusModeFailed)

			cr := newLanCR(crName, true, "failed-lan")
			cr.Annotations = map[string]string{
				compute.POSTRequestIDAnnotationKey: "simulated-post-req-failed",
			}
			meta.SetExternalName(cr, "prereq-failed-1")
			Expect(k8sClient.Create(ctx, cr)).Should(Succeed())

			// Create → annotation found → IsRequestDone returns error (FAILED)
			// Reconciler propagates error → Synced condition should show False
			Eventually(func(g Gomega) {
				fetched, err := getLanCR(ctx, crName)
				g.Expect(err).NotTo(HaveOccurred())
				syncedCond := fetched.Status.GetCondition(xpv1.TypeSynced)
				// Use corev1.ConditionFalse since crossplane conditions use corev1.ConditionStatus
				g.Expect(syncedCond.Status).To(Equal(corev1.ConditionFalse))
			}, timeout, interval).Should(Succeed())

			Consistently(func(g Gomega) {
				fetched, err := getLanCR(ctx, crName)
				g.Expect(err).NotTo(HaveOccurred())
				readyCond := fetched.Status.GetCondition(xpv1.TypeReady)
				g.Expect(readyCond.Equal(xpv1.Available())).To(BeFalse())
			}, 3*time.Second, 500*time.Millisecond).Should(Succeed())
		})
	})

	// Scenario 9: IsRequestDone — request returns 404 (lost request)
	Describe("Scenario 9: IsRequestDone — 404 lost request", Ordered, func() {
		var crName string

		BeforeAll(func() {
			crName = "test-lan-isreqdone-404"
			fakeSvc.clearErrors()
		})

		AfterAll(func() {
			testServer.setMode(statusModeDone)
			fakeSvc.clearErrors()
			cr, err := getLanCR(context.Background(), crName)
			if err == nil {
				_ = k8sClient.Delete(context.Background(), cr)
				Eventually(func() bool {
					_, err := getLanCR(context.Background(), crName)
					return err != nil
				}, timeout, interval).Should(BeTrue())
			}
		})

		It("should recover after annotation is manually removed", func() {
			ctx := context.Background()

			// Do NOT store the LAN in the fake — it was never actually created
			// Set HTTP server to return 404 for the request status
			testServer.setMode(statusMode404)

			cr := newLanCR(crName, true, "lost-req-lan")
			cr.Annotations = map[string]string{
				compute.POSTRequestIDAnnotationKey: "simulated-post-req-lost",
			}
			Expect(k8sClient.Create(ctx, cr)).Should(Succeed())

			// CR stays stuck — the annotation prevents a fresh CreateLan call
			// IsRequestDone returns (false, nil) on 404
			Consistently(func(g Gomega) {
				fetched, err := getLanCR(ctx, crName)
				g.Expect(err).NotTo(HaveOccurred())
				g.Expect(fetched.Status.AtProvider.State).NotTo(Equal("AVAILABLE"))
			}, 5*time.Second, 1*time.Second).Should(Succeed())

			// Manual intervention: remove the POST request ID annotation
			By("removing the POST request ID annotation")
			testServer.setMode(statusModeDone)
			Eventually(func(g Gomega) {
				fetched, err := getLanCR(ctx, crName)
				g.Expect(err).NotTo(HaveOccurred())
				annotations := fetched.GetAnnotations()
				delete(annotations, compute.POSTRequestIDAnnotationKey)
				fetched.SetAnnotations(annotations)
				g.Expect(k8sClient.Update(ctx, fetched)).To(Succeed())
			}, timeout, interval).Should(Succeed())

			// After annotation removal, a new LAN is created and CR reaches AVAILABLE
			Eventually(func(g Gomega) {
				fetched, err := getLanCR(ctx, crName)
				g.Expect(err).NotTo(HaveOccurred())
				g.Expect(fetched.Status.AtProvider.State).To(Equal("AVAILABLE"))
				g.Expect(fetched.Status.GetCondition(xpv1.TypeReady).Equal(xpv1.Available())).To(BeTrue())
			}, timeout, interval).Should(Succeed())
		})
	})

	// Scenario 10: Delete when API returns 404
	Describe("Scenario 10: Delete when API returns 404", Ordered, func() {
		var crName string

		BeforeAll(func() {
			testServer.setMode(statusModeDone)
			fakeSvc.clearErrors()
			crName = "test-lan-delete-404"
		})

		AfterAll(func() {
			fakeSvc.clearErrors()
		})

		It("should handle 404 gracefully on delete", func() {
			ctx := context.Background()
			cr := newLanCR(crName, true, "delete-404-lan")
			Expect(k8sClient.Create(ctx, cr)).Should(Succeed())

			// Wait for creation to complete
			Eventually(func(g Gomega) {
				fetched, err := getLanCR(ctx, crName)
				g.Expect(err).NotTo(HaveOccurred())
				g.Expect(fetched.Status.AtProvider.State).To(Equal("AVAILABLE"))
			}, timeout, interval).Should(Succeed())

			// Get the external LAN ID and remove it from the fake (simulate cloud-side deletion)
			fetched, err := getLanCR(ctx, crName)
			Expect(err).NotTo(HaveOccurred())
			lanID := meta.GetExternalName(fetched)
			fakeSvc.removeLan(testDatacenterID, lanID)

			// Set deleteErr with a 404-like error
			fakeSvc.setError("delete", fmt.Errorf("lan not found"))

			// Delete the LAN CR
			By("deleting the LAN CR")
			Expect(k8sClient.Delete(ctx, fetched)).Should(Succeed())

			// CR should be cleaned up gracefully (ErrorUnlessNotFound swallows 404)
			Eventually(func() bool {
				_, err := getLanCR(ctx, crName)
				return err != nil
			}, timeout, interval).Should(BeTrue())
		})
	})

	// Scenario 11: IPv6 AUTO late initialization
	Describe("Scenario 11: IPv6 AUTO late initialization", Ordered, func() {
		var crName string

		BeforeAll(func() {
			testServer.setMode(statusModeDone)
			fakeSvc.clearErrors()
			crName = "test-lan-ipv6-auto"
		})

		AfterAll(func() {
			cr, err := getLanCR(context.Background(), crName)
			if err == nil {
				_ = k8sClient.Delete(context.Background(), cr)
				Eventually(func() bool {
					_, err := getLanCR(context.Background(), crName)
					return err != nil
				}, timeout, interval).Should(BeTrue())
			}
		})

		It("should late-initialize ipv6Cidr from AUTO to the server-assigned value", func() {
			ctx := context.Background()
			cr := newLanCR(crName, true, "ipv6-auto-lan")
			cr.Spec.ForProvider.Ipv6Cidr = v1alpha1.LANAuto
			Expect(k8sClient.Create(ctx, cr)).Should(Succeed())

			// Wait for creation to complete and get the LAN ID
			var lanID string
			Eventually(func(g Gomega) {
				fetched, err := getLanCR(ctx, crName)
				g.Expect(err).NotTo(HaveOccurred())
				lanID = meta.GetExternalName(fetched)
				g.Expect(lanID).NotTo(BeEmpty())
				g.Expect(fetched.Status.AtProvider.State).To(Equal("AVAILABLE"))
			}, timeout, interval).Should(Succeed())

			// Now update the fake's stored LAN to simulate server-assigned IPv6
			By("simulating server-assigned IPv6 CIDR")
			l, ok := fakeSvc.getLan(testDatacenterID, lanID)
			Expect(ok).To(BeTrue())
			assignedCidr := "fd00::/64"
			l.Properties.SetIpv6CidrBlock(assignedCidr)
			fakeSvc.storeLan(testDatacenterID, lanID, l)

			// The late initializer should pick up the assigned CIDR
			Eventually(func(g Gomega) {
				fetched, err := getLanCR(ctx, crName)
				g.Expect(err).NotTo(HaveOccurred())
				g.Expect(fetched.Spec.ForProvider.Ipv6Cidr).To(Equal(assignedCidr))
			}, timeout, interval).Should(Succeed())
		})
	})
})
