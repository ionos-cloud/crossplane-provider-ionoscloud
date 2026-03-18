//go:build firewallrule_mock

package firewallrule

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
	fwClient "github.com/ionos-cloud/crossplane-provider-ionoscloud/internal/clients/compute/firewallrule"
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
// Fake FirewallRule service
// ---------------------------------------------------------------------------

type storedFirewallRule struct {
	rule         sdkgo.FirewallRule
	datacenterID string
	serverID     string
	nicID        string
}

type fakeFirewallRuleService struct {
	mu    sync.Mutex
	rules map[string]storedFirewallRule
	nextID int

	createErr, getErr, updateErr, deleteErr error
	deleteCalls                             []string

	apiClient *sdkgo.APIClient
	serverURL string
}

func newFakeFirewallRuleService(serverURL string) *fakeFirewallRuleService {
	cfg := sdkgo.NewConfiguration("", "", "test-token", serverURL)
	cfg.PollInterval = 100 * time.Millisecond
	return &fakeFirewallRuleService{
		rules:     make(map[string]storedFirewallRule),
		nextID:    1,
		apiClient: sdkgo.NewAPIClient(cfg),
		serverURL: serverURL,
	}
}

func (f *fakeFirewallRuleService) key(dcID, srvID, nicID, fwID string) string {
	return dcID + "/" + srvID + "/" + nicID + "/" + fwID
}

func (f *fakeFirewallRuleService) CheckDuplicateFirewallRule(_ context.Context, datacenterID, serverID, nicID, fwName, protocol string) (*sdkgo.FirewallRule, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	for _, sr := range f.rules {
		if sr.datacenterID == datacenterID && sr.serverID == serverID && sr.nicID == nicID &&
			sr.rule.Properties != nil && sr.rule.Properties.Name != nil && *sr.rule.Properties.Name == fwName {
			r := sr.rule
			return &r, nil
		}
	}
	return nil, nil
}

func (f *fakeFirewallRuleService) GetFirewallRuleID(r *sdkgo.FirewallRule) (string, error) {
	if r != nil {
		if id, ok := r.GetIdOk(); ok && id != nil {
			return *id, nil
		}
		return "", fmt.Errorf("error: getting firewallrule id")
	}
	return "", nil
}

func (f *fakeFirewallRuleService) GetFirewallRule(_ context.Context, datacenterID, serverID, nicID, fwID string) (sdkgo.FirewallRule, *sdkgo.APIResponse, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.getErr != nil {
		return sdkgo.FirewallRule{}, &sdkgo.APIResponse{Response: &http.Response{StatusCode: http.StatusInternalServerError}}, f.getErr
	}
	sr, ok := f.rules[f.key(datacenterID, serverID, nicID, fwID)]
	if !ok {
		return sdkgo.FirewallRule{}, &sdkgo.APIResponse{Response: &http.Response{StatusCode: http.StatusNotFound}}, fmt.Errorf("firewallrule %s not found", fwID)
	}
	return sr.rule, &sdkgo.APIResponse{Response: &http.Response{StatusCode: http.StatusOK}}, nil
}

func (f *fakeFirewallRuleService) CreateFirewallRule(_ context.Context, datacenterID, serverID, nicID string, r sdkgo.FirewallRule) (sdkgo.FirewallRule, *sdkgo.APIResponse, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.createErr != nil {
		return sdkgo.FirewallRule{}, &sdkgo.APIResponse{Response: &http.Response{StatusCode: http.StatusInternalServerError}}, f.createErr
	}
	fwID := strconv.Itoa(f.nextID)
	f.nextID++

	props := &sdkgo.FirewallruleProperties{}
	if r.Properties != nil {
		if r.Properties.Name != nil {
			props.Name = r.Properties.Name
		}
		if r.Properties.Protocol != nil {
			props.Protocol = r.Properties.Protocol
		}
	}
	newRule := sdkgo.FirewallRule{
		Id:         &fwID,
		Properties: props,
		Metadata:   &sdkgo.DatacenterElementMetadata{State: ptr.To("AVAILABLE")},
	}
	f.rules[f.key(datacenterID, serverID, nicID, fwID)] = storedFirewallRule{rule: newRule, datacenterID: datacenterID, serverID: serverID, nicID: nicID}

	header := http.Header{}
	header.Set("Location", f.serverURL+"/cloudapi/v6/requests/create-req-"+fwID+"/status")
	return newRule, &sdkgo.APIResponse{Response: &http.Response{StatusCode: http.StatusAccepted, Header: header}}, nil
}

func (f *fakeFirewallRuleService) UpdateFirewallRule(_ context.Context, datacenterID, serverID, nicID, fwID string, props sdkgo.FirewallruleProperties) (sdkgo.FirewallRule, *sdkgo.APIResponse, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.updateErr != nil {
		return sdkgo.FirewallRule{}, &sdkgo.APIResponse{Response: &http.Response{StatusCode: http.StatusInternalServerError}}, f.updateErr
	}
	sr, ok := f.rules[f.key(datacenterID, serverID, nicID, fwID)]
	if !ok {
		return sdkgo.FirewallRule{}, &sdkgo.APIResponse{Response: &http.Response{StatusCode: http.StatusNotFound}}, fmt.Errorf("firewallrule %s not found", fwID)
	}
	if props.Name != nil {
		sr.rule.Properties.Name = props.Name
	}
	if props.Protocol != nil {
		sr.rule.Properties.Protocol = props.Protocol
	}
	f.rules[f.key(datacenterID, serverID, nicID, fwID)] = sr

	header := http.Header{}
	header.Set("Location", f.serverURL+"/cloudapi/v6/requests/update-req-"+fwID+"/status")
	return sr.rule, &sdkgo.APIResponse{Response: &http.Response{StatusCode: http.StatusAccepted, Header: header}}, nil
}

func (f *fakeFirewallRuleService) DeleteFirewallRule(_ context.Context, datacenterID, serverID, nicID, fwID string) (*sdkgo.APIResponse, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.deleteErr != nil {
		return &sdkgo.APIResponse{Response: &http.Response{StatusCode: http.StatusNotFound}}, f.deleteErr
	}
	f.deleteCalls = append(f.deleteCalls, fwID)
	delete(f.rules, f.key(datacenterID, serverID, nicID, fwID))

	header := http.Header{}
	header.Set("Location", f.serverURL+"/cloudapi/v6/requests/delete-req-"+fwID+"/status")
	return &sdkgo.APIResponse{Response: &http.Response{StatusCode: http.StatusAccepted, Header: header}}, nil
}

func (f *fakeFirewallRuleService) GetAPIClient() *sdkgo.APIClient { return f.apiClient }

func (f *fakeFirewallRuleService) setError(method string, err error) {
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

func (f *fakeFirewallRuleService) clearErrors() {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.createErr = nil
	f.getErr = nil
	f.updateErr = nil
	f.deleteErr = nil
}

func (f *fakeFirewallRuleService) storeRule(dcID, srvID, nicID, fwID string, r sdkgo.FirewallRule) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.rules[f.key(dcID, srvID, nicID, fwID)] = storedFirewallRule{rule: r, datacenterID: dcID, serverID: srvID, nicID: nicID}
}

func (f *fakeFirewallRuleService) removeRule(dcID, srvID, nicID, fwID string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	delete(f.rules, f.key(dcID, srvID, nicID, fwID))
}

func (f *fakeFirewallRuleService) getDeleteCalls() []string {
	f.mu.Lock()
	defer f.mu.Unlock()
	result := make([]string, len(f.deleteCalls))
	copy(result, f.deleteCalls)
	return result
}

// ---------------------------------------------------------------------------
// Fake IPBlock service (minimal)
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

type testConnectorFirewallRule struct {
	service        fwClient.Client
	ipBlockService ipblockClient.Client
	log            logging.Logger
}

func (c *testConnectorFirewallRule) Connect(_ context.Context, _ resource.Managed) (managed.ExternalClient, error) {
	return &externalFirewallRule{service: c.service, ipBlockService: c.ipBlockService, log: c.log}, nil
}

// ---------------------------------------------------------------------------
// Test globals
// ---------------------------------------------------------------------------

var (
	k8sClient  client.Client
	cancel     context.CancelFunc
	testEnv    *envtest.Environment
	fakeSvc    *fakeFirewallRuleService
	testServer *testHTTPServer
)

var logger = zap.New(zap.UseDevMode(true))

const (
	timeout          = 60 * time.Second
	interval         = 500 * time.Millisecond
	testDatacenterID = "a1b2c3d4-e5f6-7890-abcd-ef1234567890"
	testServerID     = "b2c3d4e5-f6a7-8901-bcde-f12345678901"
	testNicID        = "c3d4e5f6-a7b8-9012-cdef-123456789012"
)

func TestFirewallRuleController_Mock(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "FirewallRule Controller Mock Suite")
}

var _ = BeforeSuite(func() {
	logf.SetLogger(logger)
	var ctx context.Context
	ctx, cancel = context.WithCancel(context.Background())

	testServer = newTestHTTPServer()
	testServer.setMode(statusModeDone)
	fakeSvc = newFakeFirewallRuleService(testServer.server.URL)
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

	name := managed.ControllerName(v1alpha1.FirewallRuleGroupKind)
	metricRecorder := managed.NewMRMetricRecorder()
	stateMetrics := statemetrics.NewMRStateMetrics()
	stateMetricsRecorder := statemetrics.NewMRStateRecorder(mgr.GetClient(), logging.NewLogrLogger(logger), stateMetrics, &v1alpha1.FirewallRuleList{}, 5*time.Minute)
	err = mgr.Add(stateMetricsRecorder)
	Expect(err).NotTo(HaveOccurred())

	err = ctrl.NewControllerManagedBy(mgr).
		Named(name).
		WithOptions(controller.Options{MaxConcurrentReconciles: 1, RateLimiter: ratelimiter.NewController(), RecoverPanic: ptr.To(true)}).
		For(&v1alpha1.FirewallRule{}).
		Complete(managed.NewReconciler(mgr,
			resource.ManagedKind(v1alpha1.FirewallRuleGroupVersionKind),
			managed.WithExternalConnecter(&testConnectorFirewallRule{service: fakeSvc, ipBlockService: fakeIPSvc, log: logging.NewLogrLogger(logger)}),
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

func newFirewallRuleCR(name string, fwName string, protocol string) *v1alpha1.FirewallRule {
	return &v1alpha1.FirewallRule{
		ObjectMeta: metav1.ObjectMeta{Name: name},
		Spec: v1alpha1.FirewallRuleSpec{
			ResourceSpec: xpv1.ResourceSpec{DeletionPolicy: xpv1.DeletionDelete, ManagementPolicies: xpv1.ManagementPolicies{xpv1.ManagementActionAll}},
			ForProvider: v1alpha1.FirewallRuleParameters{
				DatacenterCfg: v1alpha1.DatacenterConfig{DatacenterID: testDatacenterID},
				ServerCfg:     v1alpha1.ServerConfig{ServerID: testServerID},
				NicCfg:        v1alpha1.NicConfig{NicID: testNicID},
				Protocol:      protocol,
				Name:          fwName,
			},
		},
	}
}

func getFirewallRuleCR(ctx context.Context, name string) (*v1alpha1.FirewallRule, error) {
	cr := &v1alpha1.FirewallRule{}
	return cr, k8sClient.Get(ctx, types.NamespacedName{Name: name}, cr)
}

var _ = Describe("FirewallRule Controller E2E Tests", func() {
	Describe("Scenario 1: Successful creation lifecycle", Ordered, func() {
		var crName string
		BeforeAll(func() { testServer.setMode(statusModeDone); fakeSvc.clearErrors(); crName = "test-fw-create" })
		AfterAll(func() {
			if cr, err := getFirewallRuleCR(context.Background(), crName); err == nil {
				_ = k8sClient.Delete(context.Background(), cr)
			}
		})
		It("should create a FirewallRule CR and reconcile to AVAILABLE", func() {
			ctx := context.Background()
			Expect(k8sClient.Create(ctx, newFirewallRuleCR(crName, "test-fw", "TCP"))).Should(Succeed())
			Eventually(func(g Gomega) {
				fetched, err := getFirewallRuleCR(ctx, crName)
				g.Expect(err).NotTo(HaveOccurred())
				g.Expect(meta.GetExternalName(fetched)).NotTo(BeEmpty())
				g.Expect(fetched.Status.AtProvider.FirewallRuleID).NotTo(BeEmpty())
				g.Expect(fetched.Status.AtProvider.State).To(Equal("AVAILABLE"))
				g.Expect(fetched.Status.GetCondition(xpv1.TypeReady).Equal(xpv1.Available())).To(BeTrue())
			}, timeout, interval).Should(Succeed())
		})
	})

	Describe("Scenario 2: Observe stability after create", Ordered, func() {
		var crName string
		BeforeAll(func() { testServer.setMode(statusModeDone); fakeSvc.clearErrors(); crName = "test-fw-stable" })
		AfterAll(func() {
			if cr, err := getFirewallRuleCR(context.Background(), crName); err == nil {
				_ = k8sClient.Delete(context.Background(), cr)
			}
		})
		It("should stay stable", func() {
			ctx := context.Background()
			Expect(k8sClient.Create(ctx, newFirewallRuleCR(crName, "stable-fw", "TCP"))).Should(Succeed())
			Eventually(func(g Gomega) {
				fetched, err := getFirewallRuleCR(ctx, crName)
				g.Expect(err).NotTo(HaveOccurred())
				g.Expect(fetched.Status.AtProvider.State).To(Equal("AVAILABLE"))
			}, timeout, interval).Should(Succeed())
			fetched, _ := getFirewallRuleCR(ctx, crName)
			gen := fetched.Generation
			Consistently(func(g Gomega) {
				fetched, err := getFirewallRuleCR(ctx, crName)
				g.Expect(err).NotTo(HaveOccurred())
				g.Expect(fetched.Status.AtProvider.State).To(Equal("AVAILABLE"))
				g.Expect(fetched.Generation).To(Equal(gen))
			}, 5*time.Second, 1*time.Second).Should(Succeed())
		})
	})

	Describe("Scenario 3: Update lifecycle", Ordered, func() {
		var crName string
		BeforeAll(func() { testServer.setMode(statusModeDone); fakeSvc.clearErrors(); crName = "test-fw-update" })
		AfterAll(func() {
			if cr, err := getFirewallRuleCR(context.Background(), crName); err == nil {
				_ = k8sClient.Delete(context.Background(), cr)
			}
		})
		It("should update the FirewallRule", func() {
			ctx := context.Background()
			Expect(k8sClient.Create(ctx, newFirewallRuleCR(crName, "update-fw", "TCP"))).Should(Succeed())
			Eventually(func(g Gomega) {
				fetched, err := getFirewallRuleCR(ctx, crName)
				g.Expect(err).NotTo(HaveOccurred())
				g.Expect(fetched.Status.AtProvider.State).To(Equal("AVAILABLE"))
			}, timeout, interval).Should(Succeed())
			Eventually(func(g Gomega) {
				fetched, err := getFirewallRuleCR(ctx, crName)
				g.Expect(err).NotTo(HaveOccurred())
				fetched.Spec.ForProvider.Name = "updated-fw"
				g.Expect(k8sClient.Update(ctx, fetched)).To(Succeed())
			}, timeout, interval).Should(Succeed())
			Eventually(func(g Gomega) {
				fetched, err := getFirewallRuleCR(ctx, crName)
				g.Expect(err).NotTo(HaveOccurred())
				g.Expect(fetched.Status.AtProvider.State).To(Equal("AVAILABLE"))
			}, timeout, interval).Should(Succeed())
		})
	})

	Describe("Scenario 4: Delete lifecycle", Ordered, func() {
		var crName string
		BeforeAll(func() { testServer.setMode(statusModeDone); fakeSvc.clearErrors(); crName = "test-fw-delete" })
		It("should delete the FirewallRule CR", func() {
			ctx := context.Background()
			Expect(k8sClient.Create(ctx, newFirewallRuleCR(crName, "delete-fw", "TCP"))).Should(Succeed())
			Eventually(func(g Gomega) {
				fetched, err := getFirewallRuleCR(ctx, crName)
				g.Expect(err).NotTo(HaveOccurred())
				g.Expect(fetched.Status.AtProvider.State).To(Equal("AVAILABLE"))
			}, timeout, interval).Should(Succeed())
			fetched, _ := getFirewallRuleCR(ctx, crName)
			Expect(k8sClient.Delete(ctx, fetched)).Should(Succeed())
			Eventually(func() bool { _, err := getFirewallRuleCR(ctx, crName); return err != nil }, timeout, interval).Should(BeTrue())
			Expect(len(fakeSvc.getDeleteCalls())).To(BeNumerically(">", 0))
		})
	})

	Describe("Scenario 5: Create error — API returns error", Ordered, func() {
		var crName string
		BeforeAll(func() { testServer.setMode(statusModeDone); crName = "test-fw-create-err" })
		AfterAll(func() {
			fakeSvc.clearErrors()
			if cr, err := getFirewallRuleCR(context.Background(), crName); err == nil {
				_ = k8sClient.Delete(context.Background(), cr)
				Eventually(func() bool { _, err := getFirewallRuleCR(context.Background(), crName); return err != nil }, timeout, interval).Should(BeTrue())
			}
		})
		It("should fail then recover", func() {
			ctx := context.Background()
			fakeSvc.setError("create", fmt.Errorf("simulated create error"))
			Expect(k8sClient.Create(ctx, newFirewallRuleCR(crName, "error-fw", "TCP"))).Should(Succeed())
			Consistently(func(g Gomega) {
				fetched, err := getFirewallRuleCR(ctx, crName)
				g.Expect(err).NotTo(HaveOccurred())
				g.Expect(fetched.Status.AtProvider.State).NotTo(Equal("AVAILABLE"))
			}, 5*time.Second, 1*time.Second).Should(Succeed())
			fakeSvc.clearErrors()
			Eventually(func(g Gomega) {
				fetched, err := getFirewallRuleCR(ctx, crName)
				g.Expect(err).NotTo(HaveOccurred())
				g.Expect(fetched.Status.AtProvider.State).To(Equal("AVAILABLE"))
			}, timeout, interval).Should(Succeed())
		})
	})

	Describe("Scenario 6: WaitForRequest error during Create", Ordered, func() {
		var crName string
		BeforeAll(func() { crName = "test-fw-waitreq-err"; fakeSvc.clearErrors() })
		AfterAll(func() {
			testServer.setMode(statusModeDone); fakeSvc.clearErrors()
			if cr, err := getFirewallRuleCR(context.Background(), crName); err == nil {
				_ = k8sClient.Delete(context.Background(), cr)
				Eventually(func() bool { _, err := getFirewallRuleCR(context.Background(), crName); return err != nil }, timeout, interval).Should(BeTrue())
			}
		})
		It("should recover via Observe", func() {
			ctx := context.Background()
			testServer.setMode(statusModeRunning)
			Expect(k8sClient.Create(ctx, newFirewallRuleCR(crName, "waitreq-fw", "TCP"))).Should(Succeed())
			Eventually(func(g Gomega) {
				fetched, err := getFirewallRuleCR(ctx, crName)
				g.Expect(err).NotTo(HaveOccurred())
				g.Expect(meta.GetExternalName(fetched)).NotTo(BeEmpty())
			}, timeout, interval).Should(Succeed())
			testServer.setMode(statusModeDone)
			Eventually(func(g Gomega) {
				fetched, err := getFirewallRuleCR(ctx, crName)
				g.Expect(err).NotTo(HaveOccurred())
				g.Expect(fetched.Status.AtProvider.State).To(Equal("AVAILABLE"))
			}, timeout, interval).Should(Succeed())
		})
	})

	Describe("Scenario 7: IsRequestDoneNEW — request still running", Ordered, func() {
		var crName string
		const fwID = "prereq-running-1"
		BeforeAll(func() { crName = "test-fw-isreqdone-running"; fakeSvc.clearErrors() })
		AfterAll(func() {
			testServer.setMode(statusModeDone); fakeSvc.clearErrors()
			if cr, err := getFirewallRuleCR(context.Background(), crName); err == nil {
				_ = k8sClient.Delete(context.Background(), cr)
				Eventually(func() bool { _, err := getFirewallRuleCR(context.Background(), crName); return err != nil }, timeout, interval).Should(BeTrue())
			}
		})
		It("should wait for request then reconcile", func() {
			ctx := context.Background()
			testServer.setMode(statusModeRunning)
			cr := newFirewallRuleCR(crName, "isreqdone-fw", "TCP")
			cr.Annotations = map[string]string{compute.POSTRequestIDAnnotationKey: "simulated-post-req-1"}
			meta.SetExternalName(cr, fwID)
			Expect(k8sClient.Create(ctx, cr)).Should(Succeed())
			Consistently(func(g Gomega) {
				fetched, err := getFirewallRuleCR(ctx, crName)
				g.Expect(err).NotTo(HaveOccurred())
				g.Expect(fetched.Status.AtProvider.State).NotTo(Equal("AVAILABLE"))
			}, 5*time.Second, 500*time.Millisecond).Should(Succeed())
			testServer.setMode(statusModeDone)
			fakeSvc.storeRule(testDatacenterID, testServerID, testNicID, fwID, sdkgo.FirewallRule{
				Id:         ptr.To(fwID),
				Properties: &sdkgo.FirewallruleProperties{Name: ptr.To("isreqdone-fw"), Protocol: ptr.To("TCP")},
				Metadata:   &sdkgo.DatacenterElementMetadata{State: ptr.To("AVAILABLE")},
			})
			Eventually(func(g Gomega) {
				fetched, err := getFirewallRuleCR(ctx, crName)
				g.Expect(err).NotTo(HaveOccurred())
				g.Expect(fetched.Status.AtProvider.State).To(Equal("AVAILABLE"))
			}, timeout, interval).Should(Succeed())
		})
	})

	Describe("Scenario 8: IsRequestDoneNEW — request failed", Ordered, func() {
		var crName string
		BeforeAll(func() { crName = "test-fw-isreqdone-failed"; fakeSvc.clearErrors() })
		AfterAll(func() {
			testServer.setMode(statusModeDone); fakeSvc.clearErrors()
			if cr, err := getFirewallRuleCR(context.Background(), crName); err == nil {
				_ = k8sClient.Delete(context.Background(), cr)
				Eventually(func() bool { _, err := getFirewallRuleCR(context.Background(), crName); return err != nil }, timeout, interval).Should(BeTrue())
			}
		})
		It("should propagate error when FAILED", func() {
			ctx := context.Background()
			testServer.setMode(statusModeFailed)
			cr := newFirewallRuleCR(crName, "failed-fw", "TCP")
			cr.Annotations = map[string]string{compute.POSTRequestIDAnnotationKey: "simulated-post-req-failed"}
			meta.SetExternalName(cr, "prereq-failed-1")
			Expect(k8sClient.Create(ctx, cr)).Should(Succeed())
			Eventually(func(g Gomega) {
				fetched, err := getFirewallRuleCR(ctx, crName)
				g.Expect(err).NotTo(HaveOccurred())
				g.Expect(fetched.Status.GetCondition(xpv1.TypeSynced).Status).To(Equal(corev1.ConditionFalse))
			}, timeout, interval).Should(Succeed())
			Consistently(func(g Gomega) {
				fetched, err := getFirewallRuleCR(ctx, crName)
				g.Expect(err).NotTo(HaveOccurred())
				g.Expect(fetched.Status.GetCondition(xpv1.TypeReady).Equal(xpv1.Available())).To(BeFalse())
			}, 3*time.Second, 500*time.Millisecond).Should(Succeed())
		})
	})

	Describe("Scenario 9: IsRequestDoneNEW — 404 lost request", Ordered, func() {
		var crName string
		BeforeAll(func() { crName = "test-fw-isreqdone-404"; fakeSvc.clearErrors() })
		AfterAll(func() {
			testServer.setMode(statusModeDone); fakeSvc.clearErrors()
			if cr, err := getFirewallRuleCR(context.Background(), crName); err == nil {
				_ = k8sClient.Delete(context.Background(), cr)
				Eventually(func() bool { _, err := getFirewallRuleCR(context.Background(), crName); return err != nil }, timeout, interval).Should(BeTrue())
			}
		})
		It("should recover after annotation removal", func() {
			ctx := context.Background()
			testServer.setMode(statusMode404)
			cr := newFirewallRuleCR(crName, "lost-req-fw", "TCP")
			cr.Annotations = map[string]string{compute.POSTRequestIDAnnotationKey: "simulated-post-req-lost"}
			Expect(k8sClient.Create(ctx, cr)).Should(Succeed())
			Consistently(func(g Gomega) {
				fetched, err := getFirewallRuleCR(ctx, crName)
				g.Expect(err).NotTo(HaveOccurred())
				g.Expect(fetched.Status.AtProvider.State).NotTo(Equal("AVAILABLE"))
			}, 5*time.Second, 1*time.Second).Should(Succeed())
			testServer.setMode(statusModeDone)
			Eventually(func(g Gomega) {
				fetched, err := getFirewallRuleCR(ctx, crName)
				g.Expect(err).NotTo(HaveOccurred())
				annotations := fetched.GetAnnotations()
				delete(annotations, compute.POSTRequestIDAnnotationKey)
				fetched.SetAnnotations(annotations)
				g.Expect(k8sClient.Update(ctx, fetched)).To(Succeed())
			}, timeout, interval).Should(Succeed())
			Eventually(func(g Gomega) {
				fetched, err := getFirewallRuleCR(ctx, crName)
				g.Expect(err).NotTo(HaveOccurred())
				g.Expect(fetched.Status.AtProvider.State).To(Equal("AVAILABLE"))
			}, timeout, interval).Should(Succeed())
		})
	})

	Describe("Scenario 10: Delete when API returns 404", Ordered, func() {
		var crName string
		BeforeAll(func() { testServer.setMode(statusModeDone); fakeSvc.clearErrors(); crName = "test-fw-delete-404" })
		AfterAll(func() { fakeSvc.clearErrors() })
		It("should handle 404 gracefully on delete", func() {
			ctx := context.Background()
			Expect(k8sClient.Create(ctx, newFirewallRuleCR(crName, "delete-404-fw", "TCP"))).Should(Succeed())
			Eventually(func(g Gomega) {
				fetched, err := getFirewallRuleCR(ctx, crName)
				g.Expect(err).NotTo(HaveOccurred())
				g.Expect(fetched.Status.AtProvider.State).To(Equal("AVAILABLE"))
			}, timeout, interval).Should(Succeed())
			fetched, _ := getFirewallRuleCR(ctx, crName)
			fwID := meta.GetExternalName(fetched)
			fakeSvc.removeRule(testDatacenterID, testServerID, testNicID, fwID)
			fakeSvc.setError("delete", fmt.Errorf("firewallrule not found"))
			Expect(k8sClient.Delete(ctx, fetched)).Should(Succeed())
			Eventually(func() bool { _, err := getFirewallRuleCR(ctx, crName); return err != nil }, timeout, interval).Should(BeTrue())
		})
	})
})
