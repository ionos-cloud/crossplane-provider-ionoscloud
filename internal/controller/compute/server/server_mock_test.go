//go:build server_mock

package server

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
	serverClient "github.com/ionos-cloud/crossplane-provider-ionoscloud/internal/clients/compute/server"
)

// ---------------------------------------------------------------------------
// Test HTTP server behavior modes
// ---------------------------------------------------------------------------

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
// Fake Server service implementing server.Client
// ---------------------------------------------------------------------------

type storedServer struct {
	server       sdkgo.Server
	datacenterID string
}

type fakeServerService struct {
	mu      sync.Mutex
	servers map[string]storedServer // key: "datacenterID/serverID"
	nextID  int

	createErr error
	getErr    error
	updateErr error
	deleteErr error

	createCalls []sdkgo.Server
	updateCalls []sdkgo.ServerProperties
	deleteCalls []string

	apiClient *sdkgo.APIClient
	serverURL string
}

func newFakeServerService(serverURL string) *fakeServerService {
	cfg := sdkgo.NewConfiguration("", "", "test-token", serverURL)
	cfg.PollInterval = 100 * time.Millisecond
	apiClient := sdkgo.NewAPIClient(cfg)

	return &fakeServerService{
		servers:   make(map[string]storedServer),
		nextID:    1,
		apiClient: apiClient,
		serverURL: serverURL,
	}
}

func (f *fakeServerService) key(datacenterID, serverID string) string {
	return datacenterID + "/" + serverID
}

func (f *fakeServerService) CheckDuplicateServer(_ context.Context, datacenterID, serverName, cpuFamily string) (*sdkgo.Server, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
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
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.getErr != nil {
		return sdkgo.Server{}, &sdkgo.APIResponse{
			Response: &http.Response{StatusCode: http.StatusInternalServerError},
		}, f.getErr
	}
	ss, ok := f.servers[f.key(datacenterID, serverID)]
	if !ok {
		return sdkgo.Server{}, &sdkgo.APIResponse{
			Response: &http.Response{StatusCode: http.StatusNotFound},
		}, fmt.Errorf("server %s not found", serverID)
	}
	return ss.server, &sdkgo.APIResponse{
		Response: &http.Response{StatusCode: http.StatusOK},
	}, nil
}

func (f *fakeServerService) CreateServer(_ context.Context, datacenterID string, s sdkgo.Server) (sdkgo.Server, *sdkgo.APIResponse, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.createErr != nil {
		return sdkgo.Server{}, &sdkgo.APIResponse{
			Response: &http.Response{StatusCode: http.StatusInternalServerError},
		}, f.createErr
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
		Metadata: &sdkgo.DatacenterElementMetadata{
			State: ptr.To("AVAILABLE"),
		},
	}
	f.servers[f.key(datacenterID, srvID)] = storedServer{server: newSrv, datacenterID: datacenterID}

	header := http.Header{}
	requestURL := f.serverURL + "/cloudapi/v6/requests/create-req-" + srvID + "/status"
	header.Set("Location", requestURL)
	resp := &http.Response{
		StatusCode: http.StatusAccepted,
		Header:     header,
	}

	return newSrv, &sdkgo.APIResponse{Response: resp}, nil
}

func (f *fakeServerService) UpdateServer(_ context.Context, datacenterID, serverID string, props sdkgo.ServerProperties) (sdkgo.Server, *sdkgo.APIResponse, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.updateErr != nil {
		return sdkgo.Server{}, &sdkgo.APIResponse{
			Response: &http.Response{StatusCode: http.StatusInternalServerError},
		}, f.updateErr
	}

	f.updateCalls = append(f.updateCalls, props)

	ss, ok := f.servers[f.key(datacenterID, serverID)]
	if !ok {
		return sdkgo.Server{}, &sdkgo.APIResponse{
			Response: &http.Response{StatusCode: http.StatusNotFound},
		}, fmt.Errorf("server %s not found", serverID)
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

	header := http.Header{}
	requestURL := f.serverURL + "/cloudapi/v6/requests/update-req-" + serverID + "/status"
	header.Set("Location", requestURL)
	resp := &http.Response{
		StatusCode: http.StatusAccepted,
		Header:     header,
	}

	return ss.server, &sdkgo.APIResponse{Response: resp}, nil
}

func (f *fakeServerService) DeleteServer(_ context.Context, datacenterID, serverID string) (*sdkgo.APIResponse, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.deleteErr != nil {
		resp := &sdkgo.APIResponse{
			Response: &http.Response{StatusCode: http.StatusNotFound},
		}
		return resp, f.deleteErr
	}

	f.deleteCalls = append(f.deleteCalls, serverID)
	delete(f.servers, f.key(datacenterID, serverID))

	header := http.Header{}
	requestURL := f.serverURL + "/cloudapi/v6/requests/delete-req-" + serverID + "/status"
	header.Set("Location", requestURL)
	resp := &http.Response{
		StatusCode: http.StatusAccepted,
		Header:     header,
	}

	return &sdkgo.APIResponse{Response: resp}, nil
}

func (f *fakeServerService) IsVolumeAttached(_ context.Context, _, _, _ string) (bool, error) {
	return false, nil
}

func (f *fakeServerService) AttachVolume(_ context.Context, datacenterID, serverID string, vol sdkgo.Volume) (sdkgo.Volume, *sdkgo.APIResponse, error) {
	header := http.Header{}
	requestURL := f.serverURL + "/cloudapi/v6/requests/attach-vol-req/status"
	header.Set("Location", requestURL)
	resp := &http.Response{
		StatusCode: http.StatusAccepted,
		Header:     header,
	}
	return vol, &sdkgo.APIResponse{Response: resp}, nil
}

func (f *fakeServerService) DetachVolume(_ context.Context, _, _, _ string) (*sdkgo.APIResponse, error) {
	header := http.Header{}
	requestURL := f.serverURL + "/cloudapi/v6/requests/detach-vol-req/status"
	header.Set("Location", requestURL)
	resp := &http.Response{
		StatusCode: http.StatusAccepted,
		Header:     header,
	}
	return &sdkgo.APIResponse{Response: resp}, nil
}

func (f *fakeServerService) AttachCdrom(_ context.Context, _, _ string, img sdkgo.Image) (sdkgo.Image, *sdkgo.APIResponse, error) {
	header := http.Header{}
	requestURL := f.serverURL + "/cloudapi/v6/requests/attach-cdrom-req/status"
	header.Set("Location", requestURL)
	resp := &http.Response{
		StatusCode: http.StatusAccepted,
		Header:     header,
	}
	return img, &sdkgo.APIResponse{Response: resp}, nil
}

func (f *fakeServerService) DetachCdrom(_ context.Context, _, _, _ string) (*sdkgo.APIResponse, error) {
	header := http.Header{}
	requestURL := f.serverURL + "/cloudapi/v6/requests/detach-cdrom-req/status"
	header.Set("Location", requestURL)
	resp := &http.Response{
		StatusCode: http.StatusAccepted,
		Header:     header,
	}
	return &sdkgo.APIResponse{Response: resp}, nil
}

func (f *fakeServerService) StartServer(_ context.Context, _, _ string) (*sdkgo.APIResponse, error) {
	header := http.Header{}
	requestURL := f.serverURL + "/cloudapi/v6/requests/start-srv-req/status"
	header.Set("Location", requestURL)
	resp := &http.Response{
		StatusCode: http.StatusAccepted,
		Header:     header,
	}
	return &sdkgo.APIResponse{Response: resp}, nil
}

func (f *fakeServerService) StopServer(_ context.Context, _, _ string) (*sdkgo.APIResponse, error) {
	header := http.Header{}
	requestURL := f.serverURL + "/cloudapi/v6/requests/stop-srv-req/status"
	header.Set("Location", requestURL)
	resp := &http.Response{
		StatusCode: http.StatusAccepted,
		Header:     header,
	}
	return &sdkgo.APIResponse{Response: resp}, nil
}

func (f *fakeServerService) SuspendServer(_ context.Context, _, _ string) (*sdkgo.APIResponse, error) {
	header := http.Header{}
	requestURL := f.serverURL + "/cloudapi/v6/requests/suspend-srv-req/status"
	header.Set("Location", requestURL)
	resp := &http.Response{
		StatusCode: http.StatusAccepted,
		Header:     header,
	}
	return &sdkgo.APIResponse{Response: resp}, nil
}

func (f *fakeServerService) ResumeServer(_ context.Context, _, _ string) (*sdkgo.APIResponse, error) {
	header := http.Header{}
	requestURL := f.serverURL + "/cloudapi/v6/requests/resume-srv-req/status"
	header.Set("Location", requestURL)
	resp := &http.Response{
		StatusCode: http.StatusAccepted,
		Header:     header,
	}
	return &sdkgo.APIResponse{Response: resp}, nil
}

func (f *fakeServerService) GetAPIClient() *sdkgo.APIClient {
	return f.apiClient
}

// Helper methods for tests

func (f *fakeServerService) setError(method string, err error) {
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

func (f *fakeServerService) clearErrors() {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.createErr = nil
	f.getErr = nil
	f.updateErr = nil
	f.deleteErr = nil
}

func (f *fakeServerService) getServer(datacenterID, serverID string) (sdkgo.Server, bool) {
	f.mu.Lock()
	defer f.mu.Unlock()
	ss, ok := f.servers[f.key(datacenterID, serverID)]
	return ss.server, ok
}

func (f *fakeServerService) storeServer(datacenterID, serverID string, s sdkgo.Server) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.servers[f.key(datacenterID, serverID)] = storedServer{server: s, datacenterID: datacenterID}
}

func (f *fakeServerService) removeServer(datacenterID, serverID string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	delete(f.servers, f.key(datacenterID, serverID))
}

func (f *fakeServerService) getDeleteCalls() []string {
	f.mu.Lock()
	defer f.mu.Unlock()
	result := make([]string, len(f.deleteCalls))
	copy(result, f.deleteCalls)
	return result
}

// ---------------------------------------------------------------------------
// Test connector
// ---------------------------------------------------------------------------

type testConnectorServer struct {
	service serverClient.Client
	log     logging.Logger
}

func (c *testConnectorServer) Connect(_ context.Context, _ resource.Managed) (managed.ExternalClient, error) {
	return &externalServer{
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
	fakeSvc    *fakeServerService
	testServer *testHTTPServer
)

var logger = zap.New(zap.UseDevMode(true))

const (
	timeout  = 60 * time.Second
	interval = 500 * time.Millisecond

	testDatacenterID = "a1b2c3d4-e5f6-7890-abcd-ef1234567890"
)

func TestServerController_Mock(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Server Controller Mock Suite")
}

var _ = BeforeSuite(func() {
	logf.SetLogger(logger)
	var ctx context.Context
	ctx, cancel = context.WithCancel(context.Background())

	By("starting test HTTP server")
	testServer = newTestHTTPServer()
	testServer.setMode(statusModeDone)

	By("creating fake Server service")
	fakeSvc = newFakeServerService(testServer.server.URL)

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

	By("registering Server controller with fake connector")
	name := managed.ControllerName(v1alpha1.ServerGroupKind)
	metricRecorder := managed.NewMRMetricRecorder()
	stateMetrics := statemetrics.NewMRStateMetrics()

	stateMetricsRecorder := statemetrics.NewMRStateRecorder(
		mgr.GetClient(), logging.NewLogrLogger(logger),
		stateMetrics, &v1alpha1.ServerList{}, 5*time.Minute,
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
		For(&v1alpha1.Server{}).
		Complete(managed.NewReconciler(mgr,
			resource.ManagedKind(v1alpha1.ServerGroupVersionKind),
			managed.WithExternalConnecter(&testConnectorServer{
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
// Helper: create a Server CR
// ---------------------------------------------------------------------------

func newServerCR(name string, serverName string, cores int32, ram int32) *v1alpha1.Server {
	return &v1alpha1.Server{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
		Spec: v1alpha1.ServerSpec{
			ResourceSpec: xpv1.ResourceSpec{
				DeletionPolicy:     xpv1.DeletionDelete,
				ManagementPolicies: xpv1.ManagementPolicies{xpv1.ManagementActionAll},
			},
			ForProvider: v1alpha1.ServerParameters{
				DatacenterCfg: v1alpha1.DatacenterConfig{
					DatacenterID: testDatacenterID,
				},
				Name:  serverName,
				Cores: cores,
				RAM:   ram,
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

	Describe("Scenario 1: Successful creation lifecycle", Ordered, func() {
		var crName string

		BeforeAll(func() {
			testServer.setMode(statusModeDone)
			fakeSvc.clearErrors()
			crName = "test-srv-create"
		})

		AfterAll(func() {
			cr, err := getServerCR(context.Background(), crName)
			if err == nil {
				_ = k8sClient.Delete(context.Background(), cr)
			}
		})

		It("should create a Server CR and reconcile to AVAILABLE", func() {
			ctx := context.Background()
			cr := newServerCR(crName, "test-server", 2, 2048)
			Expect(k8sClient.Create(ctx, cr)).Should(Succeed())

			Eventually(func(g Gomega) {
				fetched, err := getServerCR(ctx, crName)
				g.Expect(err).NotTo(HaveOccurred())
				g.Expect(meta.GetExternalName(fetched)).NotTo(BeEmpty())
				g.Expect(fetched.Status.AtProvider.ServerID).NotTo(BeEmpty())
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
			crName = "test-srv-stable"
		})

		AfterAll(func() {
			cr, err := getServerCR(context.Background(), crName)
			if err == nil {
				_ = k8sClient.Delete(context.Background(), cr)
			}
		})

		It("should stay stable after creation", func() {
			ctx := context.Background()
			cr := newServerCR(crName, "stable-server", 2, 2048)
			Expect(k8sClient.Create(ctx, cr)).Should(Succeed())

			Eventually(func(g Gomega) {
				fetched, err := getServerCR(ctx, crName)
				g.Expect(err).NotTo(HaveOccurred())
				g.Expect(fetched.Status.AtProvider.State).To(Equal("AVAILABLE"))
			}, timeout, interval).Should(Succeed())

			fetched, err := getServerCR(ctx, crName)
			Expect(err).NotTo(HaveOccurred())
			gen := fetched.Generation

			Consistently(func(g Gomega) {
				fetched, err := getServerCR(ctx, crName)
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
			crName = "test-srv-update"
		})

		AfterAll(func() {
			cr, err := getServerCR(context.Background(), crName)
			if err == nil {
				_ = k8sClient.Delete(context.Background(), cr)
			}
		})

		It("should update the Server when spec changes", func() {
			ctx := context.Background()
			cr := newServerCR(crName, "update-server", 2, 2048)
			Expect(k8sClient.Create(ctx, cr)).Should(Succeed())

			Eventually(func(g Gomega) {
				fetched, err := getServerCR(ctx, crName)
				g.Expect(err).NotTo(HaveOccurred())
				g.Expect(fetched.Status.AtProvider.State).To(Equal("AVAILABLE"))
			}, timeout, interval).Should(Succeed())

			By("updating Server cores and RAM")
			Eventually(func(g Gomega) {
				fetched, err := getServerCR(ctx, crName)
				g.Expect(err).NotTo(HaveOccurred())
				fetched.Spec.ForProvider.Name = "updated-server"
				fetched.Spec.ForProvider.Cores = 4
				fetched.Spec.ForProvider.RAM = 4096
				g.Expect(k8sClient.Update(ctx, fetched)).To(Succeed())
			}, timeout, interval).Should(Succeed())

			Eventually(func(g Gomega) {
				fetched, err := getServerCR(ctx, crName)
				g.Expect(err).NotTo(HaveOccurred())
				g.Expect(fetched.Status.AtProvider.State).To(Equal("AVAILABLE"))
				g.Expect(fetched.Status.AtProvider.Name).To(Equal("updated-server"))
			}, timeout, interval).Should(Succeed())
		})
	})

	Describe("Scenario 4: Delete lifecycle", Ordered, func() {
		var crName string

		BeforeAll(func() {
			testServer.setMode(statusModeDone)
			fakeSvc.clearErrors()
			crName = "test-srv-delete"
		})

		It("should delete the Server CR and remove it from K8s", func() {
			ctx := context.Background()
			cr := newServerCR(crName, "delete-server", 2, 2048)
			Expect(k8sClient.Create(ctx, cr)).Should(Succeed())

			Eventually(func(g Gomega) {
				fetched, err := getServerCR(ctx, crName)
				g.Expect(err).NotTo(HaveOccurred())
				g.Expect(fetched.Status.AtProvider.State).To(Equal("AVAILABLE"))
			}, timeout, interval).Should(Succeed())

			By("deleting the Server CR")
			fetched, err := getServerCR(ctx, crName)
			Expect(err).NotTo(HaveOccurred())
			Expect(k8sClient.Delete(ctx, fetched)).Should(Succeed())

			Eventually(func() bool {
				_, err := getServerCR(ctx, crName)
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
			crName = "test-srv-create-err"
		})

		AfterAll(func() {
			fakeSvc.clearErrors()
			cr, err := getServerCR(context.Background(), crName)
			if err == nil {
				_ = k8sClient.Delete(context.Background(), cr)
				Eventually(func() bool {
					_, err := getServerCR(context.Background(), crName)
					return err != nil
				}, timeout, interval).Should(BeTrue())
			}
		})

		It("should fail then recover when error is cleared", func() {
			ctx := context.Background()
			fakeSvc.setError("create", fmt.Errorf("simulated create error"))

			cr := newServerCR(crName, "error-server", 2, 2048)
			Expect(k8sClient.Create(ctx, cr)).Should(Succeed())

			Consistently(func(g Gomega) {
				fetched, err := getServerCR(ctx, crName)
				g.Expect(err).NotTo(HaveOccurred())
				g.Expect(fetched.Status.AtProvider.State).NotTo(Equal("AVAILABLE"))
			}, 5*time.Second, 1*time.Second).Should(Succeed())

			By("clearing the create error")
			fakeSvc.clearErrors()

			Eventually(func(g Gomega) {
				fetched, err := getServerCR(ctx, crName)
				g.Expect(err).NotTo(HaveOccurred())
				g.Expect(fetched.Status.AtProvider.State).To(Equal("AVAILABLE"))
				g.Expect(fetched.Status.GetCondition(xpv1.TypeReady).Equal(xpv1.Available())).To(BeTrue())
			}, timeout, interval).Should(Succeed())
		})
	})

	Describe("Scenario 6: WaitForRequest error during Create", Ordered, func() {
		var crName string

		BeforeAll(func() {
			crName = "test-srv-waitreq-err"
			fakeSvc.clearErrors()
		})

		AfterAll(func() {
			testServer.setMode(statusModeDone)
			fakeSvc.clearErrors()
			cr, err := getServerCR(context.Background(), crName)
			if err == nil {
				_ = k8sClient.Delete(context.Background(), cr)
				Eventually(func() bool {
					_, err := getServerCR(context.Background(), crName)
					return err != nil
				}, timeout, interval).Should(BeTrue())
			}
		})

		It("should recover via Observe after WaitForRequest fails", func() {
			ctx := context.Background()
			testServer.setMode(statusModeRunning)

			cr := newServerCR(crName, "waitreq-server", 2, 2048)
			Expect(k8sClient.Create(ctx, cr)).Should(Succeed())

			Eventually(func(g Gomega) {
				fetched, err := getServerCR(ctx, crName)
				g.Expect(err).NotTo(HaveOccurred())
				g.Expect(meta.GetExternalName(fetched)).NotTo(BeEmpty())
			}, timeout, interval).Should(Succeed())

			By("switching HTTP server to DONE")
			testServer.setMode(statusModeDone)

			Eventually(func(g Gomega) {
				fetched, err := getServerCR(ctx, crName)
				g.Expect(err).NotTo(HaveOccurred())
				g.Expect(fetched.Status.AtProvider.State).To(Equal("AVAILABLE"))
				g.Expect(fetched.Status.GetCondition(xpv1.TypeReady).Equal(xpv1.Available())).To(BeTrue())
			}, timeout, interval).Should(Succeed())
		})
	})

	Describe("Scenario 7: IsRequestDone — request still running", Ordered, func() {
		var crName string
		const srvID = "prereq-running-1"

		BeforeAll(func() {
			crName = "test-srv-isreqdone-running"
			fakeSvc.clearErrors()
		})

		AfterAll(func() {
			testServer.setMode(statusModeDone)
			fakeSvc.clearErrors()
			cr, err := getServerCR(context.Background(), crName)
			if err == nil {
				_ = k8sClient.Delete(context.Background(), cr)
				Eventually(func() bool {
					_, err := getServerCR(context.Background(), crName)
					return err != nil
				}, timeout, interval).Should(BeTrue())
			}
		})

		It("should wait for request then reconcile successfully", func() {
			ctx := context.Background()
			testServer.setMode(statusModeRunning)

			cr := newServerCR(crName, "isreqdone-server", 2, 2048)
			cr.Annotations = map[string]string{
				compute.POSTRequestIDAnnotationKey: "simulated-post-req-1",
			}
			meta.SetExternalName(cr, srvID)
			Expect(k8sClient.Create(ctx, cr)).Should(Succeed())

			Consistently(func(g Gomega) {
				fetched, err := getServerCR(ctx, crName)
				g.Expect(err).NotTo(HaveOccurred())
				g.Expect(fetched.Status.AtProvider.State).NotTo(Equal("AVAILABLE"))
			}, 5*time.Second, 500*time.Millisecond).Should(Succeed())

			By("switching HTTP server to DONE and materializing the Server")
			testServer.setMode(statusModeDone)
			fakeSvc.storeServer(testDatacenterID, srvID, sdkgo.Server{
				Id: ptr.To(srvID),
				Properties: &sdkgo.ServerProperties{
					Name:  ptr.To("isreqdone-server"),
					Cores: ptr.To(int32(2)),
					Ram:   ptr.To(int32(2048)),
				},
				Metadata: &sdkgo.DatacenterElementMetadata{
					State: ptr.To("AVAILABLE"),
				},
			})

			Eventually(func(g Gomega) {
				fetched, err := getServerCR(ctx, crName)
				g.Expect(err).NotTo(HaveOccurred())
				g.Expect(fetched.Status.AtProvider.State).To(Equal("AVAILABLE"))
				g.Expect(fetched.Status.GetCondition(xpv1.TypeReady).Equal(xpv1.Available())).To(BeTrue())
			}, timeout, interval).Should(Succeed())
		})
	})

	Describe("Scenario 8: IsRequestDone — request failed", Ordered, func() {
		var crName string

		BeforeAll(func() {
			crName = "test-srv-isreqdone-failed"
			fakeSvc.clearErrors()
		})

		AfterAll(func() {
			testServer.setMode(statusModeDone)
			fakeSvc.clearErrors()
			cr, err := getServerCR(context.Background(), crName)
			if err == nil {
				_ = k8sClient.Delete(context.Background(), cr)
				Eventually(func() bool {
					_, err := getServerCR(context.Background(), crName)
					return err != nil
				}, timeout, interval).Should(BeTrue())
			}
		})

		It("should propagate error when request status is FAILED", func() {
			ctx := context.Background()
			testServer.setMode(statusModeFailed)

			cr := newServerCR(crName, "failed-server", 2, 2048)
			cr.Annotations = map[string]string{
				compute.POSTRequestIDAnnotationKey: "simulated-post-req-failed",
			}
			meta.SetExternalName(cr, "prereq-failed-1")
			Expect(k8sClient.Create(ctx, cr)).Should(Succeed())

			Eventually(func(g Gomega) {
				fetched, err := getServerCR(ctx, crName)
				g.Expect(err).NotTo(HaveOccurred())
				syncedCond := fetched.Status.GetCondition(xpv1.TypeSynced)
				g.Expect(syncedCond.Status).To(Equal(corev1.ConditionFalse))
			}, timeout, interval).Should(Succeed())

			Consistently(func(g Gomega) {
				fetched, err := getServerCR(ctx, crName)
				g.Expect(err).NotTo(HaveOccurred())
				readyCond := fetched.Status.GetCondition(xpv1.TypeReady)
				g.Expect(readyCond.Equal(xpv1.Available())).To(BeFalse())
			}, 3*time.Second, 500*time.Millisecond).Should(Succeed())
		})
	})

	Describe("Scenario 9: IsRequestDone — 404 lost request", Ordered, func() {
		var crName string

		BeforeAll(func() {
			crName = "test-srv-isreqdone-404"
			fakeSvc.clearErrors()
		})

		AfterAll(func() {
			testServer.setMode(statusModeDone)
			fakeSvc.clearErrors()
			cr, err := getServerCR(context.Background(), crName)
			if err == nil {
				_ = k8sClient.Delete(context.Background(), cr)
				Eventually(func() bool {
					_, err := getServerCR(context.Background(), crName)
					return err != nil
				}, timeout, interval).Should(BeTrue())
			}
		})

		It("should recover after annotation is manually removed", func() {
			ctx := context.Background()
			testServer.setMode(statusMode404)

			cr := newServerCR(crName, "lost-req-server", 2, 2048)
			cr.Annotations = map[string]string{
				compute.POSTRequestIDAnnotationKey: "simulated-post-req-lost",
			}
			Expect(k8sClient.Create(ctx, cr)).Should(Succeed())

			Consistently(func(g Gomega) {
				fetched, err := getServerCR(ctx, crName)
				g.Expect(err).NotTo(HaveOccurred())
				g.Expect(fetched.Status.AtProvider.State).NotTo(Equal("AVAILABLE"))
			}, 5*time.Second, 1*time.Second).Should(Succeed())

			By("removing the POST request ID annotation")
			testServer.setMode(statusModeDone)
			Eventually(func(g Gomega) {
				fetched, err := getServerCR(ctx, crName)
				g.Expect(err).NotTo(HaveOccurred())
				annotations := fetched.GetAnnotations()
				delete(annotations, compute.POSTRequestIDAnnotationKey)
				fetched.SetAnnotations(annotations)
				g.Expect(k8sClient.Update(ctx, fetched)).To(Succeed())
			}, timeout, interval).Should(Succeed())

			Eventually(func(g Gomega) {
				fetched, err := getServerCR(ctx, crName)
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
			crName = "test-srv-delete-404"
		})

		AfterAll(func() {
			fakeSvc.clearErrors()
		})

		It("should handle 404 gracefully on delete", func() {
			ctx := context.Background()
			cr := newServerCR(crName, "delete-404-server", 2, 2048)
			Expect(k8sClient.Create(ctx, cr)).Should(Succeed())

			Eventually(func(g Gomega) {
				fetched, err := getServerCR(ctx, crName)
				g.Expect(err).NotTo(HaveOccurred())
				g.Expect(fetched.Status.AtProvider.State).To(Equal("AVAILABLE"))
			}, timeout, interval).Should(Succeed())

			fetched, err := getServerCR(ctx, crName)
			Expect(err).NotTo(HaveOccurred())
			srvID := meta.GetExternalName(fetched)
			fakeSvc.removeServer(testDatacenterID, srvID)
			fakeSvc.setError("delete", fmt.Errorf("server not found"))

			By("deleting the Server CR")
			Expect(k8sClient.Delete(ctx, fetched)).Should(Succeed())

			Eventually(func() bool {
				_, err := getServerCR(ctx, crName)
				return err != nil
			}, timeout, interval).Should(BeTrue())
		})
	})
})
