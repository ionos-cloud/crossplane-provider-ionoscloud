//go:build volume_mock

package volume

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
	volumeClient "github.com/ionos-cloud/crossplane-provider-ionoscloud/internal/clients/compute/volume"
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
// Fake Volume service
// ---------------------------------------------------------------------------

type storedVolume struct {
	volume       sdkgo.Volume
	datacenterID string
}

type fakeVolumeService struct {
	mu      sync.Mutex
	volumes map[string]storedVolume
	nextID  int

	createErr, getErr, updateErr, deleteErr error
	createCalls                             []sdkgo.Volume
	updateCalls                             []sdkgo.VolumeProperties
	deleteCalls                             []string

	apiClient *sdkgo.APIClient
	serverURL string
}

func newFakeVolumeService(serverURL string) *fakeVolumeService {
	cfg := sdkgo.NewConfiguration("", "", "test-token", serverURL)
	cfg.PollInterval = 100 * time.Millisecond
	return &fakeVolumeService{
		volumes:   make(map[string]storedVolume),
		nextID:    1,
		apiClient: sdkgo.NewAPIClient(cfg),
		serverURL: serverURL,
	}
}

func (f *fakeVolumeService) key(dcID, volID string) string { return dcID + "/" + volID }

func (f *fakeVolumeService) CheckDuplicateVolume(_ context.Context, datacenterID, volumeName, _, _, _, _ string) (*sdkgo.Volume, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	for _, sv := range f.volumes {
		if sv.datacenterID == datacenterID && sv.volume.Properties != nil &&
			sv.volume.Properties.Name != nil && *sv.volume.Properties.Name == volumeName {
			v := sv.volume
			return &v, nil
		}
	}
	return nil, nil
}

func (f *fakeVolumeService) GetVolumeID(v *sdkgo.Volume) (string, error) {
	if v != nil {
		if id, ok := v.GetIdOk(); ok && id != nil {
			return *id, nil
		}
		return "", fmt.Errorf("error: getting volume id")
	}
	return "", nil
}

func (f *fakeVolumeService) GetVolume(_ context.Context, datacenterID, volumeID string) (sdkgo.Volume, *sdkgo.APIResponse, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.getErr != nil {
		return sdkgo.Volume{}, &sdkgo.APIResponse{Response: &http.Response{StatusCode: http.StatusInternalServerError}}, f.getErr
	}
	sv, ok := f.volumes[f.key(datacenterID, volumeID)]
	if !ok {
		return sdkgo.Volume{}, &sdkgo.APIResponse{Response: &http.Response{StatusCode: http.StatusNotFound}}, fmt.Errorf("volume %s not found", volumeID)
	}
	return sv.volume, &sdkgo.APIResponse{Response: &http.Response{StatusCode: http.StatusOK}}, nil
}

func (f *fakeVolumeService) CreateVolume(_ context.Context, datacenterID string, v sdkgo.Volume) (sdkgo.Volume, *sdkgo.APIResponse, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.createErr != nil {
		return sdkgo.Volume{}, &sdkgo.APIResponse{Response: &http.Response{StatusCode: http.StatusInternalServerError}}, f.createErr
	}
	f.createCalls = append(f.createCalls, v)
	volID := strconv.Itoa(f.nextID)
	f.nextID++

	props := &sdkgo.VolumeProperties{}
	if v.Properties != nil {
		if v.Properties.Name != nil {
			props.Name = v.Properties.Name
		}
		if v.Properties.Type != nil {
			props.Type = v.Properties.Type
		}
		if v.Properties.Size != nil {
			props.Size = v.Properties.Size
		}
		if v.Properties.LicenceType != nil {
			props.LicenceType = v.Properties.LicenceType
		}
	}
	newVol := sdkgo.Volume{
		Id:         &volID,
		Properties: props,
		Metadata:   &sdkgo.DatacenterElementMetadata{State: ptr.To("AVAILABLE")},
	}
	f.volumes[f.key(datacenterID, volID)] = storedVolume{volume: newVol, datacenterID: datacenterID}

	header := http.Header{}
	header.Set("Location", f.serverURL+"/cloudapi/v6/requests/create-req-"+volID+"/status")
	return newVol, &sdkgo.APIResponse{Response: &http.Response{StatusCode: http.StatusAccepted, Header: header}}, nil
}

func (f *fakeVolumeService) UpdateVolume(_ context.Context, datacenterID, volumeID string, props sdkgo.VolumeProperties) (sdkgo.Volume, *sdkgo.APIResponse, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.updateErr != nil {
		return sdkgo.Volume{}, &sdkgo.APIResponse{Response: &http.Response{StatusCode: http.StatusInternalServerError}}, f.updateErr
	}
	f.updateCalls = append(f.updateCalls, props)
	sv, ok := f.volumes[f.key(datacenterID, volumeID)]
	if !ok {
		return sdkgo.Volume{}, &sdkgo.APIResponse{Response: &http.Response{StatusCode: http.StatusNotFound}}, fmt.Errorf("volume %s not found", volumeID)
	}
	if props.Name != nil {
		sv.volume.Properties.Name = props.Name
	}
	if props.Size != nil {
		sv.volume.Properties.Size = props.Size
	}
	f.volumes[f.key(datacenterID, volumeID)] = sv

	header := http.Header{}
	header.Set("Location", f.serverURL+"/cloudapi/v6/requests/update-req-"+volumeID+"/status")
	return sv.volume, &sdkgo.APIResponse{Response: &http.Response{StatusCode: http.StatusAccepted, Header: header}}, nil
}

func (f *fakeVolumeService) DeleteVolume(_ context.Context, datacenterID, volumeID string) (*sdkgo.APIResponse, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.deleteErr != nil {
		return &sdkgo.APIResponse{Response: &http.Response{StatusCode: http.StatusNotFound}}, f.deleteErr
	}
	f.deleteCalls = append(f.deleteCalls, volumeID)
	delete(f.volumes, f.key(datacenterID, volumeID))

	header := http.Header{}
	header.Set("Location", f.serverURL+"/cloudapi/v6/requests/delete-req-"+volumeID+"/status")
	return &sdkgo.APIResponse{Response: &http.Response{StatusCode: http.StatusAccepted, Header: header}}, nil
}

func (f *fakeVolumeService) GetServerNameByID(_ context.Context, _, _ string) (string, error) {
	return "", nil
}

func (f *fakeVolumeService) GetAPIClient() *sdkgo.APIClient { return f.apiClient }

func (f *fakeVolumeService) setError(method string, err error) {
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

func (f *fakeVolumeService) clearErrors() {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.createErr = nil
	f.getErr = nil
	f.updateErr = nil
	f.deleteErr = nil
}

func (f *fakeVolumeService) storeVolume(dcID, volID string, v sdkgo.Volume) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.volumes[f.key(dcID, volID)] = storedVolume{volume: v, datacenterID: dcID}
}

func (f *fakeVolumeService) removeVolume(dcID, volID string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	delete(f.volumes, f.key(dcID, volID))
}

func (f *fakeVolumeService) getDeleteCalls() []string {
	f.mu.Lock()
	defer f.mu.Unlock()
	result := make([]string, len(f.deleteCalls))
	copy(result, f.deleteCalls)
	return result
}

// ---------------------------------------------------------------------------
// Test connector
// ---------------------------------------------------------------------------

type testConnectorVolume struct {
	service volumeClient.Client
	log     logging.Logger
}

func (c *testConnectorVolume) Connect(_ context.Context, _ resource.Managed) (managed.ExternalClient, error) {
	return &externalVolume{service: c.service, log: c.log}, nil
}

// ---------------------------------------------------------------------------
// Test globals
// ---------------------------------------------------------------------------

var (
	k8sClient  client.Client
	cancel     context.CancelFunc
	testEnv    *envtest.Environment
	fakeSvc    *fakeVolumeService
	testServer *testHTTPServer
)

var logger = zap.New(zap.UseDevMode(true))

const (
	timeout          = 60 * time.Second
	interval         = 500 * time.Millisecond
	testDatacenterID = "a1b2c3d4-e5f6-7890-abcd-ef1234567890"
)

func TestVolumeController_Mock(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Volume Controller Mock Suite")
}

var _ = BeforeSuite(func() {
	logf.SetLogger(logger)
	var ctx context.Context
	ctx, cancel = context.WithCancel(context.Background())

	By("starting test HTTP server")
	testServer = newTestHTTPServer()
	testServer.setMode(statusModeDone)

	By("creating fake Volume service")
	fakeSvc = newFakeVolumeService(testServer.server.URL)

	By("bootstrapping test environment")
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

	By("creating controller manager")
	ctrl.SetLogger(logger)
	mgr, err := ctrl.NewManager(cfg, ctrl.Options{Scheme: scheme.Scheme, Logger: logger})
	Expect(err).NotTo(HaveOccurred())

	By("registering Volume controller with fake connector")
	name := managed.ControllerName(v1alpha1.VolumeGroupKind)
	metricRecorder := managed.NewMRMetricRecorder()
	stateMetrics := statemetrics.NewMRStateMetrics()
	stateMetricsRecorder := statemetrics.NewMRStateRecorder(mgr.GetClient(), logging.NewLogrLogger(logger), stateMetrics, &v1alpha1.VolumeList{}, 5*time.Minute)
	err = mgr.Add(stateMetricsRecorder)
	Expect(err).NotTo(HaveOccurred())

	err = ctrl.NewControllerManagedBy(mgr).
		Named(name).
		WithOptions(controller.Options{MaxConcurrentReconciles: 1, RateLimiter: ratelimiter.NewController(), RecoverPanic: ptr.To(true)}).
		For(&v1alpha1.Volume{}).
		Complete(managed.NewReconciler(mgr,
			resource.ManagedKind(v1alpha1.VolumeGroupVersionKind),
			managed.WithExternalConnecter(&testConnectorVolume{service: fakeSvc, log: logging.NewLogrLogger(logger)}),
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
	Eventually(func() bool { return mgr.GetCache().WaitForCacheSync(ctx) }, timeout, interval).Should(BeTrue())
})

var _ = AfterSuite(func() {
	cancel()
	testServer.stop()
	_ = testEnv.Stop()
})

func newVolumeCR(name string, volumeName string, size float32) *v1alpha1.Volume {
	return &v1alpha1.Volume{
		ObjectMeta: metav1.ObjectMeta{Name: name},
		Spec: v1alpha1.VolumeSpec{
			ResourceSpec: xpv1.ResourceSpec{DeletionPolicy: xpv1.DeletionDelete, ManagementPolicies: xpv1.ManagementPolicies{xpv1.ManagementActionAll}},
			ForProvider: v1alpha1.VolumeParameters{
				DatacenterCfg: v1alpha1.DatacenterConfig{DatacenterID: testDatacenterID},
				Name:          volumeName,
				Type:          "HDD",
				Size:          size,
				LicenceType:   "LINUX",
			},
		},
	}
}

func getVolumeCR(ctx context.Context, name string) (*v1alpha1.Volume, error) {
	cr := &v1alpha1.Volume{}
	return cr, k8sClient.Get(ctx, types.NamespacedName{Name: name}, cr)
}

var _ = Describe("Volume Controller E2E Tests", func() {
	Describe("Scenario 1: Successful creation lifecycle", Ordered, func() {
		var crName string
		BeforeAll(func() { testServer.setMode(statusModeDone); fakeSvc.clearErrors(); crName = "test-vol-create" })
		AfterAll(func() {
			if cr, err := getVolumeCR(context.Background(), crName); err == nil {
				_ = k8sClient.Delete(context.Background(), cr)
			}
		})
		It("should create a Volume CR and reconcile to AVAILABLE", func() {
			ctx := context.Background()
			Expect(k8sClient.Create(ctx, newVolumeCR(crName, "test-volume", 10))).Should(Succeed())
			Eventually(func(g Gomega) {
				fetched, err := getVolumeCR(ctx, crName)
				g.Expect(err).NotTo(HaveOccurred())
				g.Expect(meta.GetExternalName(fetched)).NotTo(BeEmpty())
				g.Expect(fetched.Status.AtProvider.VolumeID).NotTo(BeEmpty())
				g.Expect(fetched.Status.AtProvider.State).To(Equal("AVAILABLE"))
				g.Expect(fetched.Status.GetCondition(xpv1.TypeReady).Equal(xpv1.Available())).To(BeTrue())
			}, timeout, interval).Should(Succeed())
		})
	})

	Describe("Scenario 2: Observe stability after create", Ordered, func() {
		var crName string
		BeforeAll(func() { testServer.setMode(statusModeDone); fakeSvc.clearErrors(); crName = "test-vol-stable" })
		AfterAll(func() {
			if cr, err := getVolumeCR(context.Background(), crName); err == nil {
				_ = k8sClient.Delete(context.Background(), cr)
			}
		})
		It("should stay stable after creation", func() {
			ctx := context.Background()
			Expect(k8sClient.Create(ctx, newVolumeCR(crName, "stable-volume", 10))).Should(Succeed())
			Eventually(func(g Gomega) {
				fetched, err := getVolumeCR(ctx, crName)
				g.Expect(err).NotTo(HaveOccurred())
				g.Expect(fetched.Status.AtProvider.State).To(Equal("AVAILABLE"))
			}, timeout, interval).Should(Succeed())
			fetched, _ := getVolumeCR(ctx, crName)
			gen := fetched.Generation
			Consistently(func(g Gomega) {
				fetched, err := getVolumeCR(ctx, crName)
				g.Expect(err).NotTo(HaveOccurred())
				g.Expect(fetched.Status.AtProvider.State).To(Equal("AVAILABLE"))
				g.Expect(fetched.Generation).To(Equal(gen))
			}, 5*time.Second, 1*time.Second).Should(Succeed())
		})
	})

	Describe("Scenario 3: Update lifecycle", Ordered, func() {
		var crName string
		BeforeAll(func() { testServer.setMode(statusModeDone); fakeSvc.clearErrors(); crName = "test-vol-update" })
		AfterAll(func() {
			if cr, err := getVolumeCR(context.Background(), crName); err == nil {
				_ = k8sClient.Delete(context.Background(), cr)
			}
		})
		It("should update the Volume when spec changes", func() {
			ctx := context.Background()
			Expect(k8sClient.Create(ctx, newVolumeCR(crName, "update-volume", 10))).Should(Succeed())
			Eventually(func(g Gomega) {
				fetched, err := getVolumeCR(ctx, crName)
				g.Expect(err).NotTo(HaveOccurred())
				g.Expect(fetched.Status.AtProvider.State).To(Equal("AVAILABLE"))
			}, timeout, interval).Should(Succeed())
			By("updating Volume name and size")
			Eventually(func(g Gomega) {
				fetched, err := getVolumeCR(ctx, crName)
				g.Expect(err).NotTo(HaveOccurred())
				fetched.Spec.ForProvider.Name = "updated-volume"
				fetched.Spec.ForProvider.Size = 20
				g.Expect(k8sClient.Update(ctx, fetched)).To(Succeed())
			}, timeout, interval).Should(Succeed())
			Eventually(func(g Gomega) {
				fetched, err := getVolumeCR(ctx, crName)
				g.Expect(err).NotTo(HaveOccurred())
				g.Expect(fetched.Status.AtProvider.State).To(Equal("AVAILABLE"))
			}, timeout, interval).Should(Succeed())
		})
	})

	Describe("Scenario 4: Delete lifecycle", Ordered, func() {
		var crName string
		BeforeAll(func() { testServer.setMode(statusModeDone); fakeSvc.clearErrors(); crName = "test-vol-delete" })
		It("should delete the Volume CR", func() {
			ctx := context.Background()
			Expect(k8sClient.Create(ctx, newVolumeCR(crName, "delete-volume", 10))).Should(Succeed())
			Eventually(func(g Gomega) {
				fetched, err := getVolumeCR(ctx, crName)
				g.Expect(err).NotTo(HaveOccurred())
				g.Expect(fetched.Status.AtProvider.State).To(Equal("AVAILABLE"))
			}, timeout, interval).Should(Succeed())
			fetched, _ := getVolumeCR(ctx, crName)
			Expect(k8sClient.Delete(ctx, fetched)).Should(Succeed())
			Eventually(func() bool { _, err := getVolumeCR(ctx, crName); return err != nil }, timeout, interval).Should(BeTrue())
			Expect(len(fakeSvc.getDeleteCalls())).To(BeNumerically(">", 0))
		})
	})

	Describe("Scenario 5: Create error — API returns error", Ordered, func() {
		var crName string
		BeforeAll(func() { testServer.setMode(statusModeDone); crName = "test-vol-create-err" })
		AfterAll(func() {
			fakeSvc.clearErrors()
			if cr, err := getVolumeCR(context.Background(), crName); err == nil {
				_ = k8sClient.Delete(context.Background(), cr)
				Eventually(func() bool { _, err := getVolumeCR(context.Background(), crName); return err != nil }, timeout, interval).Should(BeTrue())
			}
		})
		It("should fail then recover", func() {
			ctx := context.Background()
			fakeSvc.setError("create", fmt.Errorf("simulated create error"))
			Expect(k8sClient.Create(ctx, newVolumeCR(crName, "error-volume", 10))).Should(Succeed())
			Consistently(func(g Gomega) {
				fetched, err := getVolumeCR(ctx, crName)
				g.Expect(err).NotTo(HaveOccurred())
				g.Expect(fetched.Status.AtProvider.State).NotTo(Equal("AVAILABLE"))
			}, 5*time.Second, 1*time.Second).Should(Succeed())
			fakeSvc.clearErrors()
			Eventually(func(g Gomega) {
				fetched, err := getVolumeCR(ctx, crName)
				g.Expect(err).NotTo(HaveOccurred())
				g.Expect(fetched.Status.AtProvider.State).To(Equal("AVAILABLE"))
			}, timeout, interval).Should(Succeed())
		})
	})

	Describe("Scenario 6: WaitForRequest error during Create", Ordered, func() {
		var crName string
		BeforeAll(func() { crName = "test-vol-waitreq-err"; fakeSvc.clearErrors() })
		AfterAll(func() {
			testServer.setMode(statusModeDone); fakeSvc.clearErrors()
			if cr, err := getVolumeCR(context.Background(), crName); err == nil {
				_ = k8sClient.Delete(context.Background(), cr)
				Eventually(func() bool { _, err := getVolumeCR(context.Background(), crName); return err != nil }, timeout, interval).Should(BeTrue())
			}
		})
		It("should recover via Observe after WaitForRequest fails", func() {
			ctx := context.Background()
			testServer.setMode(statusModeRunning)
			Expect(k8sClient.Create(ctx, newVolumeCR(crName, "waitreq-volume", 10))).Should(Succeed())
			Eventually(func(g Gomega) {
				fetched, err := getVolumeCR(ctx, crName)
				g.Expect(err).NotTo(HaveOccurred())
				g.Expect(meta.GetExternalName(fetched)).NotTo(BeEmpty())
			}, timeout, interval).Should(Succeed())
			testServer.setMode(statusModeDone)
			Eventually(func(g Gomega) {
				fetched, err := getVolumeCR(ctx, crName)
				g.Expect(err).NotTo(HaveOccurred())
				g.Expect(fetched.Status.AtProvider.State).To(Equal("AVAILABLE"))
			}, timeout, interval).Should(Succeed())
		})
	})

	Describe("Scenario 7: IsRequestDoneNEW — request still running", Ordered, func() {
		var crName string
		const volID = "prereq-running-1"
		BeforeAll(func() { crName = "test-vol-isreqdone-running"; fakeSvc.clearErrors() })
		AfterAll(func() {
			testServer.setMode(statusModeDone); fakeSvc.clearErrors()
			if cr, err := getVolumeCR(context.Background(), crName); err == nil {
				_ = k8sClient.Delete(context.Background(), cr)
				Eventually(func() bool { _, err := getVolumeCR(context.Background(), crName); return err != nil }, timeout, interval).Should(BeTrue())
			}
		})
		It("should wait for request then reconcile successfully", func() {
			ctx := context.Background()
			testServer.setMode(statusModeRunning)
			cr := newVolumeCR(crName, "isreqdone-volume", 10)
			cr.Annotations = map[string]string{compute.POSTRequestIDAnnotationKey: "simulated-post-req-1"}
			meta.SetExternalName(cr, volID)
			Expect(k8sClient.Create(ctx, cr)).Should(Succeed())
			Consistently(func(g Gomega) {
				fetched, err := getVolumeCR(ctx, crName)
				g.Expect(err).NotTo(HaveOccurred())
				g.Expect(fetched.Status.AtProvider.State).NotTo(Equal("AVAILABLE"))
			}, 5*time.Second, 500*time.Millisecond).Should(Succeed())
			testServer.setMode(statusModeDone)
			fakeSvc.storeVolume(testDatacenterID, volID, sdkgo.Volume{
				Id:         ptr.To(volID),
				Properties: &sdkgo.VolumeProperties{Name: ptr.To("isreqdone-volume"), Type: ptr.To("HDD"), Size: ptr.To(float32(10))},
				Metadata:   &sdkgo.DatacenterElementMetadata{State: ptr.To("AVAILABLE")},
			})
			Eventually(func(g Gomega) {
				fetched, err := getVolumeCR(ctx, crName)
				g.Expect(err).NotTo(HaveOccurred())
				g.Expect(fetched.Status.AtProvider.State).To(Equal("AVAILABLE"))
			}, timeout, interval).Should(Succeed())
		})
	})

	Describe("Scenario 8: IsRequestDoneNEW — request failed", Ordered, func() {
		var crName string
		BeforeAll(func() { crName = "test-vol-isreqdone-failed"; fakeSvc.clearErrors() })
		AfterAll(func() {
			testServer.setMode(statusModeDone); fakeSvc.clearErrors()
			if cr, err := getVolumeCR(context.Background(), crName); err == nil {
				_ = k8sClient.Delete(context.Background(), cr)
				Eventually(func() bool { _, err := getVolumeCR(context.Background(), crName); return err != nil }, timeout, interval).Should(BeTrue())
			}
		})
		It("should propagate error when request status is FAILED", func() {
			ctx := context.Background()
			testServer.setMode(statusModeFailed)
			cr := newVolumeCR(crName, "failed-volume", 10)
			cr.Annotations = map[string]string{compute.POSTRequestIDAnnotationKey: "simulated-post-req-failed"}
			meta.SetExternalName(cr, "prereq-failed-1")
			Expect(k8sClient.Create(ctx, cr)).Should(Succeed())
			Eventually(func(g Gomega) {
				fetched, err := getVolumeCR(ctx, crName)
				g.Expect(err).NotTo(HaveOccurred())
				g.Expect(fetched.Status.GetCondition(xpv1.TypeSynced).Status).To(Equal(corev1.ConditionFalse))
			}, timeout, interval).Should(Succeed())
			Consistently(func(g Gomega) {
				fetched, err := getVolumeCR(ctx, crName)
				g.Expect(err).NotTo(HaveOccurred())
				g.Expect(fetched.Status.GetCondition(xpv1.TypeReady).Equal(xpv1.Available())).To(BeFalse())
			}, 3*time.Second, 500*time.Millisecond).Should(Succeed())
		})
	})

	Describe("Scenario 9: IsRequestDoneNEW — 404 lost request", Ordered, func() {
		var crName string
		BeforeAll(func() { crName = "test-vol-isreqdone-404"; fakeSvc.clearErrors() })
		AfterAll(func() {
			testServer.setMode(statusModeDone); fakeSvc.clearErrors()
			if cr, err := getVolumeCR(context.Background(), crName); err == nil {
				_ = k8sClient.Delete(context.Background(), cr)
				Eventually(func() bool { _, err := getVolumeCR(context.Background(), crName); return err != nil }, timeout, interval).Should(BeTrue())
			}
		})
		It("should recover after annotation is manually removed", func() {
			ctx := context.Background()
			testServer.setMode(statusMode404)
			cr := newVolumeCR(crName, "lost-req-volume", 10)
			cr.Annotations = map[string]string{compute.POSTRequestIDAnnotationKey: "simulated-post-req-lost"}
			Expect(k8sClient.Create(ctx, cr)).Should(Succeed())
			Consistently(func(g Gomega) {
				fetched, err := getVolumeCR(ctx, crName)
				g.Expect(err).NotTo(HaveOccurred())
				g.Expect(fetched.Status.AtProvider.State).NotTo(Equal("AVAILABLE"))
			}, 5*time.Second, 1*time.Second).Should(Succeed())
			testServer.setMode(statusModeDone)
			Eventually(func(g Gomega) {
				fetched, err := getVolumeCR(ctx, crName)
				g.Expect(err).NotTo(HaveOccurred())
				annotations := fetched.GetAnnotations()
				delete(annotations, compute.POSTRequestIDAnnotationKey)
				fetched.SetAnnotations(annotations)
				g.Expect(k8sClient.Update(ctx, fetched)).To(Succeed())
			}, timeout, interval).Should(Succeed())
			Eventually(func(g Gomega) {
				fetched, err := getVolumeCR(ctx, crName)
				g.Expect(err).NotTo(HaveOccurred())
				g.Expect(fetched.Status.AtProvider.State).To(Equal("AVAILABLE"))
			}, timeout, interval).Should(Succeed())
		})
	})

	Describe("Scenario 10: Delete when API returns 404", Ordered, func() {
		var crName string
		BeforeAll(func() { testServer.setMode(statusModeDone); fakeSvc.clearErrors(); crName = "test-vol-delete-404" })
		AfterAll(func() { fakeSvc.clearErrors() })
		It("should handle 404 gracefully on delete", func() {
			ctx := context.Background()
			Expect(k8sClient.Create(ctx, newVolumeCR(crName, "delete-404-volume", 10))).Should(Succeed())
			Eventually(func(g Gomega) {
				fetched, err := getVolumeCR(ctx, crName)
				g.Expect(err).NotTo(HaveOccurred())
				g.Expect(fetched.Status.AtProvider.State).To(Equal("AVAILABLE"))
			}, timeout, interval).Should(Succeed())
			fetched, _ := getVolumeCR(ctx, crName)
			volID := meta.GetExternalName(fetched)
			fakeSvc.removeVolume(testDatacenterID, volID)
			fakeSvc.setError("delete", fmt.Errorf("volume not found"))
			Expect(k8sClient.Delete(ctx, fetched)).Should(Succeed())
			Eventually(func() bool { _, err := getVolumeCR(ctx, crName); return err != nil }, timeout, interval).Should(BeTrue())
		})
	})
})
