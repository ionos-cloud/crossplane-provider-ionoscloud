//go:build datacenter_mock

package datacenter

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
	datacenterClient "github.com/ionos-cloud/crossplane-provider-ionoscloud/internal/clients/compute/datacenter"
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

	mux.HandleFunc("/cloudapi/v6/requests/", func(w http.ResponseWriter, r *http.Request) {
		mode := requestStatusMode(ts.mode.Load())
		w.Header().Set("Content-Type", "application/json")
		switch mode {
		case statusModeDone:
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"id":   "test-request-id",
				"type": "request-status",
				"metadata": map[string]interface{}{
					"status": "DONE",
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
// Fake Datacenter service implementing datacenter.Client
// ---------------------------------------------------------------------------

type storedDatacenter struct {
	datacenter sdkgo.Datacenter
}

type fakeDatacenterService struct {
	mu          sync.Mutex
	datacenters map[string]storedDatacenter // key: datacenterID
	nextID      int

	createErr error
	getErr    error
	updateErr error
	deleteErr error

	createCalls []sdkgo.DatacenterPost
	updateCalls []sdkgo.DatacenterPropertiesPut
	deleteCalls []string

	apiClient *sdkgo.APIClient
	serverURL string
}

func newFakeDatacenterService(serverURL string) *fakeDatacenterService {
	cfg := sdkgo.NewConfiguration("", "", "test-token", serverURL)
	cfg.PollInterval = 100 * time.Millisecond
	apiClient := sdkgo.NewAPIClient(cfg)

	return &fakeDatacenterService{
		datacenters: make(map[string]storedDatacenter),
		nextID:      1,
		apiClient:   apiClient,
		serverURL:   serverURL,
	}
}

func (f *fakeDatacenterService) CheckDuplicateDatacenter(_ context.Context, datacenterName, location string) (*sdkgo.Datacenter, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
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
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.getErr != nil {
		return sdkgo.Datacenter{}, &sdkgo.APIResponse{
			Response: &http.Response{StatusCode: http.StatusInternalServerError},
		}, f.getErr
	}
	sd, ok := f.datacenters[datacenterID]
	if !ok {
		return sdkgo.Datacenter{}, &sdkgo.APIResponse{
			Response: &http.Response{StatusCode: http.StatusNotFound},
		}, fmt.Errorf("datacenter %s not found", datacenterID)
	}
	return sd.datacenter, &sdkgo.APIResponse{
		Response: &http.Response{StatusCode: http.StatusOK},
	}, nil
}

func (f *fakeDatacenterService) CreateDatacenter(_ context.Context, d sdkgo.DatacenterPost) (sdkgo.Datacenter, *sdkgo.APIResponse, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.createErr != nil {
		return sdkgo.Datacenter{}, &sdkgo.APIResponse{
			Response: &http.Response{StatusCode: http.StatusInternalServerError},
		}, f.createErr
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
		Metadata: &sdkgo.DatacenterElementMetadata{
			State: ptr.To("AVAILABLE"),
		},
	}
	f.datacenters[dcID] = storedDatacenter{datacenter: newDC}

	header := http.Header{}
	requestURL := f.serverURL + "/cloudapi/v6/requests/create-req-" + dcID + "/status"
	header.Set("Location", requestURL)
	resp := &http.Response{
		StatusCode: http.StatusAccepted,
		Header:     header,
	}

	return newDC, &sdkgo.APIResponse{Response: resp}, nil
}

func (f *fakeDatacenterService) UpdateDatacenter(_ context.Context, datacenterID string, props sdkgo.DatacenterPropertiesPut) (sdkgo.Datacenter, *sdkgo.APIResponse, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.updateErr != nil {
		return sdkgo.Datacenter{}, &sdkgo.APIResponse{
			Response: &http.Response{StatusCode: http.StatusInternalServerError},
		}, f.updateErr
	}

	f.updateCalls = append(f.updateCalls, props)

	sd, ok := f.datacenters[datacenterID]
	if !ok {
		return sdkgo.Datacenter{}, &sdkgo.APIResponse{
			Response: &http.Response{StatusCode: http.StatusNotFound},
		}, fmt.Errorf("datacenter %s not found", datacenterID)
	}

	if props.Name != nil {
		sd.datacenter.Properties.Name = props.Name
	}
	if props.Description != nil {
		sd.datacenter.Properties.Description = props.Description
	}
	f.datacenters[datacenterID] = sd

	header := http.Header{}
	requestURL := f.serverURL + "/cloudapi/v6/requests/update-req-" + datacenterID + "/status"
	header.Set("Location", requestURL)
	resp := &http.Response{
		StatusCode: http.StatusAccepted,
		Header:     header,
	}

	return sd.datacenter, &sdkgo.APIResponse{Response: resp}, nil
}

func (f *fakeDatacenterService) DeleteDatacenter(_ context.Context, datacenterID string) (*sdkgo.APIResponse, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.deleteErr != nil {
		resp := &sdkgo.APIResponse{
			Response: &http.Response{StatusCode: http.StatusNotFound},
		}
		return resp, f.deleteErr
	}

	f.deleteCalls = append(f.deleteCalls, datacenterID)
	delete(f.datacenters, datacenterID)

	header := http.Header{}
	requestURL := f.serverURL + "/cloudapi/v6/requests/delete-req-" + datacenterID + "/status"
	header.Set("Location", requestURL)
	resp := &http.Response{
		StatusCode: http.StatusAccepted,
		Header:     header,
	}

	return &sdkgo.APIResponse{Response: resp}, nil
}

func (f *fakeDatacenterService) GetCPUFamiliesForDatacenter(_ context.Context, _ string) ([]string, error) {
	return []string{"INTEL_SKYLAKE", "AMD_EPYC"}, nil
}

func (f *fakeDatacenterService) GetAPIClient() *sdkgo.APIClient {
	return f.apiClient
}

// Helper methods for tests

func (f *fakeDatacenterService) setError(method string, err error) {
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

func (f *fakeDatacenterService) clearErrors() {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.createErr = nil
	f.getErr = nil
	f.updateErr = nil
	f.deleteErr = nil
}

func (f *fakeDatacenterService) getDatacenter(datacenterID string) (sdkgo.Datacenter, bool) {
	f.mu.Lock()
	defer f.mu.Unlock()
	sd, ok := f.datacenters[datacenterID]
	return sd.datacenter, ok
}

func (f *fakeDatacenterService) storeDatacenter(datacenterID string, d sdkgo.Datacenter) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.datacenters[datacenterID] = storedDatacenter{datacenter: d}
}

func (f *fakeDatacenterService) removeDatacenter(datacenterID string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	delete(f.datacenters, datacenterID)
}

func (f *fakeDatacenterService) getDeleteCalls() []string {
	f.mu.Lock()
	defer f.mu.Unlock()
	result := make([]string, len(f.deleteCalls))
	copy(result, f.deleteCalls)
	return result
}

// ---------------------------------------------------------------------------
// Test connector
// ---------------------------------------------------------------------------

type testConnectorDatacenter struct {
	service datacenterClient.Client
	log     logging.Logger
}

func (c *testConnectorDatacenter) Connect(_ context.Context, _ resource.Managed) (managed.ExternalClient, error) {
	return &externalDatacenter{
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
	fakeSvc    *fakeDatacenterService
	testServer *testHTTPServer
)

var logger = zap.New(zap.UseDevMode(true))

const (
	timeout  = 60 * time.Second
	interval = 500 * time.Millisecond
)

func TestDatacenterController_Mock(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Datacenter Controller Mock Suite")
}

var _ = BeforeSuite(func() {
	logf.SetLogger(logger)
	var ctx context.Context
	ctx, cancel = context.WithCancel(context.Background())

	By("starting test HTTP server")
	testServer = newTestHTTPServer()
	testServer.setMode(statusModeDone)

	By("creating fake Datacenter service")
	fakeSvc = newFakeDatacenterService(testServer.server.URL)

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

	By("registering Datacenter controller with fake connector")
	name := managed.ControllerName(v1alpha1.DatacenterGroupKind)
	metricRecorder := managed.NewMRMetricRecorder()
	stateMetrics := statemetrics.NewMRStateMetrics()

	stateMetricsRecorder := statemetrics.NewMRStateRecorder(
		mgr.GetClient(), logging.NewLogrLogger(logger),
		stateMetrics, &v1alpha1.DatacenterList{}, 5*time.Minute,
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
		For(&v1alpha1.Datacenter{}).
		Complete(managed.NewReconciler(mgr,
			resource.ManagedKind(v1alpha1.DatacenterGroupVersionKind),
			managed.WithExternalConnecter(&testConnectorDatacenter{
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
// Helper: create a Datacenter CR
// ---------------------------------------------------------------------------

func newDatacenterCR(name string, dcName string) *v1alpha1.Datacenter {
	return &v1alpha1.Datacenter{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
		Spec: v1alpha1.DatacenterSpec{
			ResourceSpec: xpv1.ResourceSpec{
				DeletionPolicy:     xpv1.DeletionDelete,
				ManagementPolicies: xpv1.ManagementPolicies{xpv1.ManagementActionAll},
			},
			ForProvider: v1alpha1.DatacenterParameters{
				Name:     dcName,
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

	Describe("Scenario 1: Successful creation lifecycle", Ordered, func() {
		var crName string

		BeforeAll(func() {
			testServer.setMode(statusModeDone)
			fakeSvc.clearErrors()
			crName = "test-dc-create"
		})

		AfterAll(func() {
			cr, err := getDatacenterCR(context.Background(), crName)
			if err == nil {
				_ = k8sClient.Delete(context.Background(), cr)
			}
		})

		It("should create a Datacenter CR and reconcile to AVAILABLE", func() {
			ctx := context.Background()
			cr := newDatacenterCR(crName, "test-datacenter")
			Expect(k8sClient.Create(ctx, cr)).Should(Succeed())

			Eventually(func(g Gomega) {
				fetched, err := getDatacenterCR(ctx, crName)
				g.Expect(err).NotTo(HaveOccurred())
				g.Expect(meta.GetExternalName(fetched)).NotTo(BeEmpty())
				g.Expect(fetched.Status.AtProvider.DatacenterID).NotTo(BeEmpty())
				g.Expect(fetched.Status.AtProvider.State).To(Equal("AVAILABLE"))
				g.Expect(fetched.Status.GetCondition(xpv1.TypeReady).Equal(xpv1.Available())).To(BeTrue())
			}, timeout, interval).Should(Succeed())
		})
	})

	Describe("Scenario 2: Observe stability after create", Ordered, func() {
		var crName string

		BeforeAll(func() {
			testServer.setMode(statusModeDone)
			fakeSvc.clearErrors()
			crName = "test-dc-stable"
		})

		AfterAll(func() {
			cr, err := getDatacenterCR(context.Background(), crName)
			if err == nil {
				_ = k8sClient.Delete(context.Background(), cr)
			}
		})

		It("should stay stable after creation", func() {
			ctx := context.Background()
			cr := newDatacenterCR(crName, "stable-datacenter")
			Expect(k8sClient.Create(ctx, cr)).Should(Succeed())

			Eventually(func(g Gomega) {
				fetched, err := getDatacenterCR(ctx, crName)
				g.Expect(err).NotTo(HaveOccurred())
				g.Expect(fetched.Status.AtProvider.State).To(Equal("AVAILABLE"))
			}, timeout, interval).Should(Succeed())

			fetched, err := getDatacenterCR(ctx, crName)
			Expect(err).NotTo(HaveOccurred())
			gen := fetched.Generation

			Consistently(func(g Gomega) {
				fetched, err := getDatacenterCR(ctx, crName)
				g.Expect(err).NotTo(HaveOccurred())
				g.Expect(fetched.Status.AtProvider.State).To(Equal("AVAILABLE"))
				g.Expect(fetched.Generation).To(Equal(gen))
			}, 5*time.Second, 1*time.Second).Should(Succeed())
		})
	})

	Describe("Scenario 3: Update lifecycle", Ordered, func() {
		var crName string

		BeforeAll(func() {
			testServer.setMode(statusModeDone)
			fakeSvc.clearErrors()
			crName = "test-dc-update"
		})

		AfterAll(func() {
			cr, err := getDatacenterCR(context.Background(), crName)
			if err == nil {
				_ = k8sClient.Delete(context.Background(), cr)
			}
		})

		It("should update the Datacenter when spec changes", func() {
			ctx := context.Background()
			cr := newDatacenterCR(crName, "update-datacenter")
			Expect(k8sClient.Create(ctx, cr)).Should(Succeed())

			Eventually(func(g Gomega) {
				fetched, err := getDatacenterCR(ctx, crName)
				g.Expect(err).NotTo(HaveOccurred())
				g.Expect(fetched.Status.AtProvider.State).To(Equal("AVAILABLE"))
			}, timeout, interval).Should(Succeed())

			By("updating Datacenter name")
			Eventually(func(g Gomega) {
				fetched, err := getDatacenterCR(ctx, crName)
				g.Expect(err).NotTo(HaveOccurred())
				fetched.Spec.ForProvider.Name = "updated-datacenter"
				g.Expect(k8sClient.Update(ctx, fetched)).To(Succeed())
			}, timeout, interval).Should(Succeed())

			Eventually(func(g Gomega) {
				fetched, err := getDatacenterCR(ctx, crName)
				g.Expect(err).NotTo(HaveOccurred())
				g.Expect(fetched.Status.AtProvider.State).To(Equal("AVAILABLE"))
			}, timeout, interval).Should(Succeed())
		})
	})

	Describe("Scenario 4: Delete lifecycle", Ordered, func() {
		var crName string

		BeforeAll(func() {
			testServer.setMode(statusModeDone)
			fakeSvc.clearErrors()
			crName = "test-dc-delete"
		})

		It("should delete the Datacenter CR and remove it from K8s", func() {
			ctx := context.Background()
			cr := newDatacenterCR(crName, "delete-datacenter")
			Expect(k8sClient.Create(ctx, cr)).Should(Succeed())

			Eventually(func(g Gomega) {
				fetched, err := getDatacenterCR(ctx, crName)
				g.Expect(err).NotTo(HaveOccurred())
				g.Expect(fetched.Status.AtProvider.State).To(Equal("AVAILABLE"))
			}, timeout, interval).Should(Succeed())

			By("deleting the Datacenter CR")
			fetched, err := getDatacenterCR(ctx, crName)
			Expect(err).NotTo(HaveOccurred())
			Expect(k8sClient.Delete(ctx, fetched)).Should(Succeed())

			Eventually(func() bool {
				_, err := getDatacenterCR(ctx, crName)
				return err != nil
			}, timeout, interval).Should(BeTrue())

			deleteCalls := fakeSvc.getDeleteCalls()
			Expect(len(deleteCalls)).To(BeNumerically(">", 0))
		})
	})

	Describe("Scenario 5: Create error — API returns error", Ordered, func() {
		var crName string

		BeforeAll(func() {
			testServer.setMode(statusModeDone)
			crName = "test-dc-create-err"
		})

		AfterAll(func() {
			fakeSvc.clearErrors()
			cr, err := getDatacenterCR(context.Background(), crName)
			if err == nil {
				_ = k8sClient.Delete(context.Background(), cr)
				Eventually(func() bool {
					_, err := getDatacenterCR(context.Background(), crName)
					return err != nil
				}, timeout, interval).Should(BeTrue())
			}
		})

		It("should fail then recover when error is cleared", func() {
			ctx := context.Background()
			fakeSvc.setError("create", fmt.Errorf("simulated create error"))

			cr := newDatacenterCR(crName, "error-datacenter")
			Expect(k8sClient.Create(ctx, cr)).Should(Succeed())

			Consistently(func(g Gomega) {
				fetched, err := getDatacenterCR(ctx, crName)
				g.Expect(err).NotTo(HaveOccurred())
				g.Expect(fetched.Status.AtProvider.State).NotTo(Equal("AVAILABLE"))
			}, 5*time.Second, 1*time.Second).Should(Succeed())

			By("clearing the create error")
			fakeSvc.clearErrors()

			Eventually(func(g Gomega) {
				fetched, err := getDatacenterCR(ctx, crName)
				g.Expect(err).NotTo(HaveOccurred())
				g.Expect(fetched.Status.AtProvider.State).To(Equal("AVAILABLE"))
				g.Expect(fetched.Status.GetCondition(xpv1.TypeReady).Equal(xpv1.Available())).To(BeTrue())
			}, timeout, interval).Should(Succeed())
		})
	})

	Describe("Scenario 6: WaitForRequest error during Create", Ordered, func() {
		var crName string

		BeforeAll(func() {
			crName = "test-dc-waitreq-err"
			fakeSvc.clearErrors()
		})

		AfterAll(func() {
			testServer.setMode(statusModeDone)
			fakeSvc.clearErrors()
			cr, err := getDatacenterCR(context.Background(), crName)
			if err == nil {
				_ = k8sClient.Delete(context.Background(), cr)
				Eventually(func() bool {
					_, err := getDatacenterCR(context.Background(), crName)
					return err != nil
				}, timeout, interval).Should(BeTrue())
			}
		})

		It("should recover via Observe after WaitForRequest fails", func() {
			ctx := context.Background()
			testServer.setMode(statusModeRunning)

			cr := newDatacenterCR(crName, "waitreq-datacenter")
			Expect(k8sClient.Create(ctx, cr)).Should(Succeed())

			Eventually(func(g Gomega) {
				fetched, err := getDatacenterCR(ctx, crName)
				g.Expect(err).NotTo(HaveOccurred())
				g.Expect(meta.GetExternalName(fetched)).NotTo(BeEmpty())
			}, timeout, interval).Should(Succeed())

			By("switching HTTP server to DONE")
			testServer.setMode(statusModeDone)

			Eventually(func(g Gomega) {
				fetched, err := getDatacenterCR(ctx, crName)
				g.Expect(err).NotTo(HaveOccurred())
				g.Expect(fetched.Status.AtProvider.State).To(Equal("AVAILABLE"))
				g.Expect(fetched.Status.GetCondition(xpv1.TypeReady).Equal(xpv1.Available())).To(BeTrue())
			}, timeout, interval).Should(Succeed())
		})
	})

	Describe("Scenario 7: IsRequestDoneNEW — request still running", Ordered, func() {
		var crName string
		const dcID = "prereq-running-1"

		BeforeAll(func() {
			crName = "test-dc-isreqdone-running"
			fakeSvc.clearErrors()
		})

		AfterAll(func() {
			testServer.setMode(statusModeDone)
			fakeSvc.clearErrors()
			cr, err := getDatacenterCR(context.Background(), crName)
			if err == nil {
				_ = k8sClient.Delete(context.Background(), cr)
				Eventually(func() bool {
					_, err := getDatacenterCR(context.Background(), crName)
					return err != nil
				}, timeout, interval).Should(BeTrue())
			}
		})

		It("should wait for request then reconcile successfully", func() {
			ctx := context.Background()
			testServer.setMode(statusModeRunning)

			cr := newDatacenterCR(crName, "isreqdone-datacenter")
			cr.Annotations = map[string]string{
				compute.POSTRequestIDAnnotationKey: "simulated-post-req-1",
			}
			meta.SetExternalName(cr, dcID)
			Expect(k8sClient.Create(ctx, cr)).Should(Succeed())

			Consistently(func(g Gomega) {
				fetched, err := getDatacenterCR(ctx, crName)
				g.Expect(err).NotTo(HaveOccurred())
				g.Expect(fetched.Status.AtProvider.State).NotTo(Equal("AVAILABLE"))
			}, 5*time.Second, 500*time.Millisecond).Should(Succeed())

			By("switching HTTP server to DONE and materializing the Datacenter")
			testServer.setMode(statusModeDone)
			fakeSvc.storeDatacenter(dcID, sdkgo.Datacenter{
				Id: ptr.To(dcID),
				Properties: &sdkgo.DatacenterProperties{
					Name:     ptr.To("isreqdone-datacenter"),
					Location: ptr.To("de/fra"),
				},
				Metadata: &sdkgo.DatacenterElementMetadata{
					State: ptr.To("AVAILABLE"),
				},
			})

			Eventually(func(g Gomega) {
				fetched, err := getDatacenterCR(ctx, crName)
				g.Expect(err).NotTo(HaveOccurred())
				g.Expect(fetched.Status.AtProvider.State).To(Equal("AVAILABLE"))
				g.Expect(fetched.Status.GetCondition(xpv1.TypeReady).Equal(xpv1.Available())).To(BeTrue())
			}, timeout, interval).Should(Succeed())
		})
	})

	Describe("Scenario 8: IsRequestDoneNEW — request failed", Ordered, func() {
		var crName string

		BeforeAll(func() {
			crName = "test-dc-isreqdone-failed"
			fakeSvc.clearErrors()
		})

		AfterAll(func() {
			testServer.setMode(statusModeDone)
			fakeSvc.clearErrors()
			cr, err := getDatacenterCR(context.Background(), crName)
			if err == nil {
				_ = k8sClient.Delete(context.Background(), cr)
				Eventually(func() bool {
					_, err := getDatacenterCR(context.Background(), crName)
					return err != nil
				}, timeout, interval).Should(BeTrue())
			}
		})

		It("should propagate error when request status is FAILED", func() {
			ctx := context.Background()
			testServer.setMode(statusModeFailed)

			cr := newDatacenterCR(crName, "failed-datacenter")
			cr.Annotations = map[string]string{
				compute.POSTRequestIDAnnotationKey: "simulated-post-req-failed",
			}
			meta.SetExternalName(cr, "prereq-failed-1")
			Expect(k8sClient.Create(ctx, cr)).Should(Succeed())

			Eventually(func(g Gomega) {
				fetched, err := getDatacenterCR(ctx, crName)
				g.Expect(err).NotTo(HaveOccurred())
				syncedCond := fetched.Status.GetCondition(xpv1.TypeSynced)
				g.Expect(syncedCond.Status).To(Equal(corev1.ConditionFalse))
			}, timeout, interval).Should(Succeed())

			Consistently(func(g Gomega) {
				fetched, err := getDatacenterCR(ctx, crName)
				g.Expect(err).NotTo(HaveOccurred())
				readyCond := fetched.Status.GetCondition(xpv1.TypeReady)
				g.Expect(readyCond.Equal(xpv1.Available())).To(BeFalse())
			}, 3*time.Second, 500*time.Millisecond).Should(Succeed())
		})
	})

	Describe("Scenario 9: IsRequestDoneNEW — 404 lost request", Ordered, func() {
		var crName string

		BeforeAll(func() {
			crName = "test-dc-isreqdone-404"
			fakeSvc.clearErrors()
		})

		AfterAll(func() {
			testServer.setMode(statusModeDone)
			fakeSvc.clearErrors()
			cr, err := getDatacenterCR(context.Background(), crName)
			if err == nil {
				_ = k8sClient.Delete(context.Background(), cr)
				Eventually(func() bool {
					_, err := getDatacenterCR(context.Background(), crName)
					return err != nil
				}, timeout, interval).Should(BeTrue())
			}
		})

		It("should recover after annotation is manually removed", func() {
			ctx := context.Background()
			testServer.setMode(statusMode404)

			cr := newDatacenterCR(crName, "lost-req-datacenter")
			cr.Annotations = map[string]string{
				compute.POSTRequestIDAnnotationKey: "simulated-post-req-lost",
			}
			Expect(k8sClient.Create(ctx, cr)).Should(Succeed())

			Consistently(func(g Gomega) {
				fetched, err := getDatacenterCR(ctx, crName)
				g.Expect(err).NotTo(HaveOccurred())
				g.Expect(fetched.Status.AtProvider.State).NotTo(Equal("AVAILABLE"))
			}, 5*time.Second, 1*time.Second).Should(Succeed())

			By("removing the POST request ID annotation")
			testServer.setMode(statusModeDone)
			Eventually(func(g Gomega) {
				fetched, err := getDatacenterCR(ctx, crName)
				g.Expect(err).NotTo(HaveOccurred())
				annotations := fetched.GetAnnotations()
				delete(annotations, compute.POSTRequestIDAnnotationKey)
				fetched.SetAnnotations(annotations)
				g.Expect(k8sClient.Update(ctx, fetched)).To(Succeed())
			}, timeout, interval).Should(Succeed())

			Eventually(func(g Gomega) {
				fetched, err := getDatacenterCR(ctx, crName)
				g.Expect(err).NotTo(HaveOccurred())
				g.Expect(fetched.Status.AtProvider.State).To(Equal("AVAILABLE"))
				g.Expect(fetched.Status.GetCondition(xpv1.TypeReady).Equal(xpv1.Available())).To(BeTrue())
			}, timeout, interval).Should(Succeed())
		})
	})

	Describe("Scenario 10: Delete when API returns 404", Ordered, func() {
		var crName string

		BeforeAll(func() {
			testServer.setMode(statusModeDone)
			fakeSvc.clearErrors()
			crName = "test-dc-delete-404"
		})

		AfterAll(func() {
			fakeSvc.clearErrors()
		})

		It("should handle 404 gracefully on delete", func() {
			ctx := context.Background()
			cr := newDatacenterCR(crName, "delete-404-datacenter")
			Expect(k8sClient.Create(ctx, cr)).Should(Succeed())

			Eventually(func(g Gomega) {
				fetched, err := getDatacenterCR(ctx, crName)
				g.Expect(err).NotTo(HaveOccurred())
				g.Expect(fetched.Status.AtProvider.State).To(Equal("AVAILABLE"))
			}, timeout, interval).Should(Succeed())

			fetched, err := getDatacenterCR(ctx, crName)
			Expect(err).NotTo(HaveOccurred())
			dcID := meta.GetExternalName(fetched)
			fakeSvc.removeDatacenter(dcID)
			fakeSvc.setError("delete", fmt.Errorf("datacenter not found"))

			By("deleting the Datacenter CR")
			Expect(k8sClient.Delete(ctx, fetched)).Should(Succeed())

			Eventually(func() bool {
				_, err := getDatacenterCR(ctx, crName)
				return err != nil
			}, timeout, interval).Should(BeTrue())
		})
	})
})
