//go:build ipblock_mock

package ipblock

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
// Fake IPBlock service
// ---------------------------------------------------------------------------

type storedIPBlock struct {
	ipBlock sdkgo.IpBlock
}

type fakeIPBlockService struct {
	mu       sync.Mutex
	ipBlocks map[string]storedIPBlock
	nextID   int

	createErr, getErr, updateErr, deleteErr error
	deleteCalls                             []string

	apiClient *sdkgo.APIClient
	serverURL string
}

func newFakeIPBlockService(serverURL string) *fakeIPBlockService {
	cfg := sdkgo.NewConfiguration("", "", "test-token", serverURL)
	cfg.PollInterval = 100 * time.Millisecond
	return &fakeIPBlockService{
		ipBlocks:  make(map[string]storedIPBlock),
		nextID:    1,
		apiClient: sdkgo.NewAPIClient(cfg),
		serverURL: serverURL,
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
	f.mu.Lock()
	defer f.mu.Unlock()
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
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.getErr != nil {
		return sdkgo.IpBlock{}, &sdkgo.APIResponse{Response: &http.Response{StatusCode: http.StatusInternalServerError}}, f.getErr
	}
	sb, ok := f.ipBlocks[ipBlockID]
	if !ok {
		return sdkgo.IpBlock{}, &sdkgo.APIResponse{Response: &http.Response{StatusCode: http.StatusNotFound}}, fmt.Errorf("ipblock %s not found", ipBlockID)
	}
	return sb.ipBlock, &sdkgo.APIResponse{Response: &http.Response{StatusCode: http.StatusOK}}, nil
}

func (f *fakeIPBlockService) CreateIPBlock(_ context.Context, b sdkgo.IpBlock) (sdkgo.IpBlock, *sdkgo.APIResponse, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.createErr != nil {
		return sdkgo.IpBlock{}, &sdkgo.APIResponse{Response: &http.Response{StatusCode: http.StatusInternalServerError}}, f.createErr
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

	header := http.Header{}
	header.Set("Location", f.serverURL+"/cloudapi/v6/requests/create-req-"+blockID+"/status")
	return newBlock, &sdkgo.APIResponse{Response: &http.Response{StatusCode: http.StatusAccepted, Header: header}}, nil
}

func (f *fakeIPBlockService) UpdateIPBlock(_ context.Context, ipBlockID string, props sdkgo.IpBlockProperties) (sdkgo.IpBlock, *sdkgo.APIResponse, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.updateErr != nil {
		return sdkgo.IpBlock{}, &sdkgo.APIResponse{Response: &http.Response{StatusCode: http.StatusInternalServerError}}, f.updateErr
	}
	sb, ok := f.ipBlocks[ipBlockID]
	if !ok {
		return sdkgo.IpBlock{}, &sdkgo.APIResponse{Response: &http.Response{StatusCode: http.StatusNotFound}}, fmt.Errorf("ipblock %s not found", ipBlockID)
	}
	if props.Name != nil {
		sb.ipBlock.Properties.Name = props.Name
	}
	f.ipBlocks[ipBlockID] = sb

	header := http.Header{}
	header.Set("Location", f.serverURL+"/cloudapi/v6/requests/update-req-"+ipBlockID+"/status")
	return sb.ipBlock, &sdkgo.APIResponse{Response: &http.Response{StatusCode: http.StatusAccepted, Header: header}}, nil
}

func (f *fakeIPBlockService) DeleteIPBlock(_ context.Context, ipBlockID string) (*sdkgo.APIResponse, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.deleteErr != nil {
		return &sdkgo.APIResponse{Response: &http.Response{StatusCode: http.StatusNotFound}}, f.deleteErr
	}
	f.deleteCalls = append(f.deleteCalls, ipBlockID)
	delete(f.ipBlocks, ipBlockID)

	header := http.Header{}
	header.Set("Location", f.serverURL+"/cloudapi/v6/requests/delete-req-"+ipBlockID+"/status")
	return &sdkgo.APIResponse{Response: &http.Response{StatusCode: http.StatusAccepted, Header: header}}, nil
}

func (f *fakeIPBlockService) GetIPs(_ context.Context, ipBlockID string, indexes ...int) ([]string, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
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

func (f *fakeIPBlockService) GetAPIClient() *sdkgo.APIClient { return f.apiClient }

func (f *fakeIPBlockService) setError(method string, err error) {
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

func (f *fakeIPBlockService) clearErrors() {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.createErr = nil
	f.getErr = nil
	f.updateErr = nil
	f.deleteErr = nil
}

func (f *fakeIPBlockService) storeIPBlock(id string, b sdkgo.IpBlock) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.ipBlocks[id] = storedIPBlock{ipBlock: b}
}

func (f *fakeIPBlockService) removeIPBlock(id string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	delete(f.ipBlocks, id)
}

func (f *fakeIPBlockService) getDeleteCalls() []string {
	f.mu.Lock()
	defer f.mu.Unlock()
	result := make([]string, len(f.deleteCalls))
	copy(result, f.deleteCalls)
	return result
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
	k8sClient    client.Client
	cancel       context.CancelFunc
	testEnv      *envtest.Environment
	fakeSvc      *fakeIPBlockService
	testSrvHTTP  *testHTTPServer
)

var logger = zap.New(zap.UseDevMode(true))

const (
	timeout  = 60 * time.Second
	interval = 500 * time.Millisecond
)

func TestIPBlockController_Mock(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "IPBlock Controller Mock Suite")
}

var _ = BeforeSuite(func() {
	logf.SetLogger(logger)
	var ctx context.Context
	ctx, cancel = context.WithCancel(context.Background())

	testSrvHTTP = newTestHTTPServer()
	testSrvHTTP.setMode(statusModeDone)
	fakeSvc = newFakeIPBlockService(testSrvHTTP.server.URL)

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

	name := managed.ControllerName(v1alpha1.IPBlockGroupKind)
	metricRecorder := managed.NewMRMetricRecorder()
	stateMetrics := statemetrics.NewMRStateMetrics()
	stateMetricsRecorder := statemetrics.NewMRStateRecorder(mgr.GetClient(), logging.NewLogrLogger(logger), stateMetrics, &v1alpha1.IPBlockList{}, 5*time.Minute)
	err = mgr.Add(stateMetricsRecorder)
	Expect(err).NotTo(HaveOccurred())

	err = ctrl.NewControllerManagedBy(mgr).
		Named(name).
		WithOptions(controller.Options{MaxConcurrentReconciles: 1, RateLimiter: ratelimiter.NewController(), RecoverPanic: ptr.To(true)}).
		For(&v1alpha1.IPBlock{}).
		Complete(managed.NewReconciler(mgr,
			resource.ManagedKind(v1alpha1.IPBlockGroupVersionKind),
			managed.WithExternalConnecter(&testConnectorIPBlock{service: fakeSvc, log: logging.NewLogrLogger(logger)}),
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
	testSrvHTTP.stop()
	_ = testEnv.Stop()
})

func newIPBlockCR(name string, ipBlockName string, size int32) *v1alpha1.IPBlock {
	return &v1alpha1.IPBlock{
		ObjectMeta: metav1.ObjectMeta{Name: name},
		Spec: v1alpha1.IPBlockSpec{
			ResourceSpec: xpv1.ResourceSpec{DeletionPolicy: xpv1.DeletionDelete, ManagementPolicies: xpv1.ManagementPolicies{xpv1.ManagementActionAll}},
			ForProvider: v1alpha1.IPBlockParameters{
				Name:     ipBlockName,
				Location: "de/fra",
				Size:     size,
			},
		},
	}
}

func getIPBlockCR(ctx context.Context, name string) (*v1alpha1.IPBlock, error) {
	cr := &v1alpha1.IPBlock{}
	return cr, k8sClient.Get(ctx, types.NamespacedName{Name: name}, cr)
}

var _ = Describe("IPBlock Controller E2E Tests", func() {
	Describe("Scenario 1: Successful creation lifecycle", Ordered, func() {
		var crName string
		BeforeAll(func() { testSrvHTTP.setMode(statusModeDone); fakeSvc.clearErrors(); crName = "test-ipb-create" })
		AfterAll(func() {
			if cr, err := getIPBlockCR(context.Background(), crName); err == nil {
				_ = k8sClient.Delete(context.Background(), cr)
			}
		})
		It("should create an IPBlock CR and reconcile to AVAILABLE", func() {
			ctx := context.Background()
			Expect(k8sClient.Create(ctx, newIPBlockCR(crName, "test-ipblock", 2))).Should(Succeed())
			Eventually(func(g Gomega) {
				fetched, err := getIPBlockCR(ctx, crName)
				g.Expect(err).NotTo(HaveOccurred())
				g.Expect(meta.GetExternalName(fetched)).NotTo(BeEmpty())
				g.Expect(fetched.Status.AtProvider.IPBlockID).NotTo(BeEmpty())
				g.Expect(fetched.Status.AtProvider.State).To(Equal("AVAILABLE"))
				g.Expect(fetched.Status.GetCondition(xpv1.TypeReady).Equal(xpv1.Available())).To(BeTrue())
			}, timeout, interval).Should(Succeed())
		})
	})

	Describe("Scenario 2: Observe stability after create", Ordered, func() {
		var crName string
		BeforeAll(func() { testSrvHTTP.setMode(statusModeDone); fakeSvc.clearErrors(); crName = "test-ipb-stable" })
		AfterAll(func() {
			if cr, err := getIPBlockCR(context.Background(), crName); err == nil {
				_ = k8sClient.Delete(context.Background(), cr)
			}
		})
		It("should stay stable after creation", func() {
			ctx := context.Background()
			Expect(k8sClient.Create(ctx, newIPBlockCR(crName, "stable-ipblock", 2))).Should(Succeed())
			Eventually(func(g Gomega) {
				fetched, err := getIPBlockCR(ctx, crName)
				g.Expect(err).NotTo(HaveOccurred())
				g.Expect(fetched.Status.AtProvider.State).To(Equal("AVAILABLE"))
			}, timeout, interval).Should(Succeed())
			fetched, _ := getIPBlockCR(ctx, crName)
			gen := fetched.Generation
			Consistently(func(g Gomega) {
				fetched, err := getIPBlockCR(ctx, crName)
				g.Expect(err).NotTo(HaveOccurred())
				g.Expect(fetched.Status.AtProvider.State).To(Equal("AVAILABLE"))
				g.Expect(fetched.Generation).To(Equal(gen))
			}, 5*time.Second, 1*time.Second).Should(Succeed())
		})
	})

	Describe("Scenario 3: Update lifecycle", Ordered, func() {
		var crName string
		BeforeAll(func() { testSrvHTTP.setMode(statusModeDone); fakeSvc.clearErrors(); crName = "test-ipb-update" })
		AfterAll(func() {
			if cr, err := getIPBlockCR(context.Background(), crName); err == nil {
				_ = k8sClient.Delete(context.Background(), cr)
			}
		})
		It("should update the IPBlock when spec changes", func() {
			ctx := context.Background()
			Expect(k8sClient.Create(ctx, newIPBlockCR(crName, "update-ipblock", 2))).Should(Succeed())
			Eventually(func(g Gomega) {
				fetched, err := getIPBlockCR(ctx, crName)
				g.Expect(err).NotTo(HaveOccurred())
				g.Expect(fetched.Status.AtProvider.State).To(Equal("AVAILABLE"))
			}, timeout, interval).Should(Succeed())
			Eventually(func(g Gomega) {
				fetched, err := getIPBlockCR(ctx, crName)
				g.Expect(err).NotTo(HaveOccurred())
				fetched.Spec.ForProvider.Name = "updated-ipblock"
				g.Expect(k8sClient.Update(ctx, fetched)).To(Succeed())
			}, timeout, interval).Should(Succeed())
			Eventually(func(g Gomega) {
				fetched, err := getIPBlockCR(ctx, crName)
				g.Expect(err).NotTo(HaveOccurred())
				g.Expect(fetched.Status.AtProvider.State).To(Equal("AVAILABLE"))
			}, timeout, interval).Should(Succeed())
		})
	})

	Describe("Scenario 4: Delete lifecycle", Ordered, func() {
		var crName string
		BeforeAll(func() { testSrvHTTP.setMode(statusModeDone); fakeSvc.clearErrors(); crName = "test-ipb-delete" })
		It("should delete the IPBlock CR", func() {
			ctx := context.Background()
			Expect(k8sClient.Create(ctx, newIPBlockCR(crName, "delete-ipblock", 2))).Should(Succeed())
			Eventually(func(g Gomega) {
				fetched, err := getIPBlockCR(ctx, crName)
				g.Expect(err).NotTo(HaveOccurred())
				g.Expect(fetched.Status.AtProvider.State).To(Equal("AVAILABLE"))
			}, timeout, interval).Should(Succeed())
			fetched, _ := getIPBlockCR(ctx, crName)
			Expect(k8sClient.Delete(ctx, fetched)).Should(Succeed())
			Eventually(func() bool { _, err := getIPBlockCR(ctx, crName); return err != nil }, timeout, interval).Should(BeTrue())
			Expect(len(fakeSvc.getDeleteCalls())).To(BeNumerically(">", 0))
		})
	})

	Describe("Scenario 5: Create error — API returns error", Ordered, func() {
		var crName string
		BeforeAll(func() { testSrvHTTP.setMode(statusModeDone); crName = "test-ipb-create-err" })
		AfterAll(func() {
			fakeSvc.clearErrors()
			if cr, err := getIPBlockCR(context.Background(), crName); err == nil {
				_ = k8sClient.Delete(context.Background(), cr)
				Eventually(func() bool { _, err := getIPBlockCR(context.Background(), crName); return err != nil }, timeout, interval).Should(BeTrue())
			}
		})
		It("should fail then recover", func() {
			ctx := context.Background()
			fakeSvc.setError("create", fmt.Errorf("simulated create error"))
			Expect(k8sClient.Create(ctx, newIPBlockCR(crName, "error-ipblock", 2))).Should(Succeed())
			Consistently(func(g Gomega) {
				fetched, err := getIPBlockCR(ctx, crName)
				g.Expect(err).NotTo(HaveOccurred())
				g.Expect(fetched.Status.AtProvider.State).NotTo(Equal("AVAILABLE"))
			}, 5*time.Second, 1*time.Second).Should(Succeed())
			fakeSvc.clearErrors()
			Eventually(func(g Gomega) {
				fetched, err := getIPBlockCR(ctx, crName)
				g.Expect(err).NotTo(HaveOccurred())
				g.Expect(fetched.Status.AtProvider.State).To(Equal("AVAILABLE"))
			}, timeout, interval).Should(Succeed())
		})
	})

	Describe("Scenario 6: WaitForRequest error during Create", Ordered, func() {
		var crName string
		BeforeAll(func() { crName = "test-ipb-waitreq-err"; fakeSvc.clearErrors() })
		AfterAll(func() {
			testSrvHTTP.setMode(statusModeDone); fakeSvc.clearErrors()
			if cr, err := getIPBlockCR(context.Background(), crName); err == nil {
				_ = k8sClient.Delete(context.Background(), cr)
				Eventually(func() bool { _, err := getIPBlockCR(context.Background(), crName); return err != nil }, timeout, interval).Should(BeTrue())
			}
		})
		It("should recover via Observe", func() {
			ctx := context.Background()
			testSrvHTTP.setMode(statusModeRunning)
			Expect(k8sClient.Create(ctx, newIPBlockCR(crName, "waitreq-ipblock", 2))).Should(Succeed())
			Eventually(func(g Gomega) {
				fetched, err := getIPBlockCR(ctx, crName)
				g.Expect(err).NotTo(HaveOccurred())
				g.Expect(meta.GetExternalName(fetched)).NotTo(BeEmpty())
			}, timeout, interval).Should(Succeed())
			testSrvHTTP.setMode(statusModeDone)
			Eventually(func(g Gomega) {
				fetched, err := getIPBlockCR(ctx, crName)
				g.Expect(err).NotTo(HaveOccurred())
				g.Expect(fetched.Status.AtProvider.State).To(Equal("AVAILABLE"))
			}, timeout, interval).Should(Succeed())
		})
	})

	Describe("Scenario 7: IsRequestDone — request still running", Ordered, func() {
		var crName string
		const blockID = "prereq-running-1"
		BeforeAll(func() { crName = "test-ipb-isreqdone-running"; fakeSvc.clearErrors() })
		AfterAll(func() {
			testSrvHTTP.setMode(statusModeDone); fakeSvc.clearErrors()
			if cr, err := getIPBlockCR(context.Background(), crName); err == nil {
				_ = k8sClient.Delete(context.Background(), cr)
				Eventually(func() bool { _, err := getIPBlockCR(context.Background(), crName); return err != nil }, timeout, interval).Should(BeTrue())
			}
		})
		It("should wait for request then reconcile", func() {
			ctx := context.Background()
			testSrvHTTP.setMode(statusModeRunning)
			cr := newIPBlockCR(crName, "isreqdone-ipblock", 2)
			cr.Annotations = map[string]string{compute.POSTRequestIDAnnotationKey: "simulated-post-req-1"}
			meta.SetExternalName(cr, blockID)
			Expect(k8sClient.Create(ctx, cr)).Should(Succeed())
			Consistently(func(g Gomega) {
				fetched, err := getIPBlockCR(ctx, crName)
				g.Expect(err).NotTo(HaveOccurred())
				g.Expect(fetched.Status.AtProvider.State).NotTo(Equal("AVAILABLE"))
			}, 5*time.Second, 500*time.Millisecond).Should(Succeed())
			testSrvHTTP.setMode(statusModeDone)
			ips := generateIPs(2)
			fakeSvc.storeIPBlock(blockID, sdkgo.IpBlock{
				Id:         ptr.To(blockID),
				Properties: &sdkgo.IpBlockProperties{Name: ptr.To("isreqdone-ipblock"), Location: ptr.To("de/fra"), Size: ptr.To(int32(2)), Ips: ips},
				Metadata:   &sdkgo.DatacenterElementMetadata{State: ptr.To("AVAILABLE")},
			})
			Eventually(func(g Gomega) {
				fetched, err := getIPBlockCR(ctx, crName)
				g.Expect(err).NotTo(HaveOccurred())
				g.Expect(fetched.Status.AtProvider.State).To(Equal("AVAILABLE"))
			}, timeout, interval).Should(Succeed())
		})
	})

	Describe("Scenario 8: IsRequestDone — request failed", Ordered, func() {
		var crName string
		BeforeAll(func() { crName = "test-ipb-isreqdone-failed"; fakeSvc.clearErrors() })
		AfterAll(func() {
			testSrvHTTP.setMode(statusModeDone); fakeSvc.clearErrors()
			if cr, err := getIPBlockCR(context.Background(), crName); err == nil {
				_ = k8sClient.Delete(context.Background(), cr)
				Eventually(func() bool { _, err := getIPBlockCR(context.Background(), crName); return err != nil }, timeout, interval).Should(BeTrue())
			}
		})
		It("should propagate error when FAILED", func() {
			ctx := context.Background()
			testSrvHTTP.setMode(statusModeFailed)
			cr := newIPBlockCR(crName, "failed-ipblock", 2)
			cr.Annotations = map[string]string{compute.POSTRequestIDAnnotationKey: "simulated-post-req-failed"}
			meta.SetExternalName(cr, "prereq-failed-1")
			Expect(k8sClient.Create(ctx, cr)).Should(Succeed())
			Eventually(func(g Gomega) {
				fetched, err := getIPBlockCR(ctx, crName)
				g.Expect(err).NotTo(HaveOccurred())
				g.Expect(fetched.Status.GetCondition(xpv1.TypeSynced).Status).To(Equal(corev1.ConditionFalse))
			}, timeout, interval).Should(Succeed())
			Consistently(func(g Gomega) {
				fetched, err := getIPBlockCR(ctx, crName)
				g.Expect(err).NotTo(HaveOccurred())
				g.Expect(fetched.Status.GetCondition(xpv1.TypeReady).Equal(xpv1.Available())).To(BeFalse())
			}, 3*time.Second, 500*time.Millisecond).Should(Succeed())
		})
	})

	Describe("Scenario 9: IsRequestDone — 404 lost request", Ordered, func() {
		var crName string
		BeforeAll(func() { crName = "test-ipb-isreqdone-404"; fakeSvc.clearErrors() })
		AfterAll(func() {
			testSrvHTTP.setMode(statusModeDone); fakeSvc.clearErrors()
			if cr, err := getIPBlockCR(context.Background(), crName); err == nil {
				_ = k8sClient.Delete(context.Background(), cr)
				Eventually(func() bool { _, err := getIPBlockCR(context.Background(), crName); return err != nil }, timeout, interval).Should(BeTrue())
			}
		})
		It("should recover after annotation removal", func() {
			ctx := context.Background()
			testSrvHTTP.setMode(statusMode404)
			cr := newIPBlockCR(crName, "lost-req-ipblock", 2)
			cr.Annotations = map[string]string{compute.POSTRequestIDAnnotationKey: "simulated-post-req-lost"}
			Expect(k8sClient.Create(ctx, cr)).Should(Succeed())
			Consistently(func(g Gomega) {
				fetched, err := getIPBlockCR(ctx, crName)
				g.Expect(err).NotTo(HaveOccurred())
				g.Expect(fetched.Status.AtProvider.State).NotTo(Equal("AVAILABLE"))
			}, 5*time.Second, 1*time.Second).Should(Succeed())
			testSrvHTTP.setMode(statusModeDone)
			Eventually(func(g Gomega) {
				fetched, err := getIPBlockCR(ctx, crName)
				g.Expect(err).NotTo(HaveOccurred())
				annotations := fetched.GetAnnotations()
				delete(annotations, compute.POSTRequestIDAnnotationKey)
				fetched.SetAnnotations(annotations)
				g.Expect(k8sClient.Update(ctx, fetched)).To(Succeed())
			}, timeout, interval).Should(Succeed())
			Eventually(func(g Gomega) {
				fetched, err := getIPBlockCR(ctx, crName)
				g.Expect(err).NotTo(HaveOccurred())
				g.Expect(fetched.Status.AtProvider.State).To(Equal("AVAILABLE"))
			}, timeout, interval).Should(Succeed())
		})
	})

	Describe("Scenario 10: Delete when API returns 404", Ordered, func() {
		var crName string
		BeforeAll(func() { testSrvHTTP.setMode(statusModeDone); fakeSvc.clearErrors(); crName = "test-ipb-delete-404" })
		AfterAll(func() { fakeSvc.clearErrors() })
		It("should handle 404 gracefully on delete", func() {
			ctx := context.Background()
			Expect(k8sClient.Create(ctx, newIPBlockCR(crName, "delete-404-ipblock", 2))).Should(Succeed())
			Eventually(func(g Gomega) {
				fetched, err := getIPBlockCR(ctx, crName)
				g.Expect(err).NotTo(HaveOccurred())
				g.Expect(fetched.Status.AtProvider.State).To(Equal("AVAILABLE"))
			}, timeout, interval).Should(Succeed())
			fetched, _ := getIPBlockCR(ctx, crName)
			blockID := meta.GetExternalName(fetched)
			fakeSvc.removeIPBlock(blockID)
			fakeSvc.setError("delete", fmt.Errorf("ipblock not found"))
			Expect(k8sClient.Delete(ctx, fetched)).Should(Succeed())
			Eventually(func() bool { _, err := getIPBlockCR(ctx, crName); return err != nil }, timeout, interval).Should(BeTrue())
		})
	})
})
