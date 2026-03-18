package mocktest

import (
	"context"
	"os"
	"path/filepath"
	"time"

	"github.com/crossplane/crossplane-runtime/pkg/event"
	"github.com/crossplane/crossplane-runtime/pkg/logging"
	"github.com/crossplane/crossplane-runtime/pkg/ratelimiter"
	"github.com/crossplane/crossplane-runtime/pkg/reconciler/managed"
	"github.com/crossplane/crossplane-runtime/pkg/resource"
	"github.com/crossplane/crossplane-runtime/pkg/statemetrics"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/utils/ptr"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"

	"github.com/ionos-cloud/crossplane-provider-ionoscloud/apis"
)

const (
	Timeout  = 60 * time.Second
	Interval = 500 * time.Millisecond

	TestDatacenterID = "a1b2c3d4-e5f6-7890-abcd-ef1234567890"
	TestServerID     = "b2c3d4e5-f6a7-8901-bcde-f12345678901"
	TestNicID        = "c3d4e5f6-a7b8-9012-cdef-123456789012"
	TestLanID        = "1"
)

// Logger is the shared test logger.
var Logger = zap.New(zap.UseDevMode(true))

// ControllerSetup holds the resource-specific config for setting up a controller under test.
type ControllerSetup struct {
	GroupKind        string
	GroupVersionKind schema.GroupVersionKind
	ManagedResource  client.Object
	ManagedList      resource.ManagedList
	Connector        managed.ExternalConnecter
}

// EnvTestHarness holds the initialized envtest infrastructure.
type EnvTestHarness struct {
	K8sClient client.Client
	Cancel    context.CancelFunc
	TestEnv   *envtest.Environment
}

// SetupEnvTest bootstraps envtest, creates a controller manager, registers the controller,
// and starts the manager. It returns the initialized harness.
func SetupEnvTest(setup ControllerSetup) *EnvTestHarness {
	logf.SetLogger(Logger)
	ctx, cancel := context.WithCancel(context.Background())

	By("bootstrapping test environment")
	testEnv := &envtest.Environment{
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

	k8sClient, err := client.New(cfg, client.Options{Scheme: scheme.Scheme})
	Expect(err).NotTo(HaveOccurred())
	Expect(k8sClient).NotTo(BeNil())

	By("creating controller manager")
	ctrl.SetLogger(Logger)
	mgr, err := ctrl.NewManager(cfg, ctrl.Options{
		Scheme: scheme.Scheme,
		Logger: Logger,
	})
	Expect(err).NotTo(HaveOccurred())

	By("registering controller with fake connector")
	name := managed.ControllerName(setup.GroupKind)
	metricRecorder := managed.NewMRMetricRecorder()
	stateMetrics := statemetrics.NewMRStateMetrics()

	stateMetricsRecorder := statemetrics.NewMRStateRecorder(
		mgr.GetClient(), logging.NewLogrLogger(Logger),
		stateMetrics, setup.ManagedList, 5*time.Minute,
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
		For(setup.ManagedResource).
		Complete(managed.NewReconciler(mgr,
			resource.ManagedKind(setup.GroupVersionKind),
			managed.WithExternalConnecter(setup.Connector),
			managed.WithReferenceResolver(managed.NewAPISimpleReferenceResolver(mgr.GetClient())),
			managed.WithInitializers(),
			managed.WithPollInterval(1*time.Second),
			managed.WithTimeout(30*time.Second),
			managed.WithCreationGracePeriod(5*time.Second),
			managed.WithLogger(logging.NewLogrLogger(Logger).WithValues("controller", name)),
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
	}, Timeout, Interval).Should(BeTrue())

	return &EnvTestHarness{
		K8sClient: k8sClient,
		Cancel:    cancel,
		TestEnv:   testEnv,
	}
}

// TeardownEnvTest cancels the context, stops the HTTP server, and tears down the envtest environment.
func TeardownEnvTest(harness *EnvTestHarness, testServer *TestHTTPServer) {
	harness.Cancel()
	if testServer != nil {
		By("stopping test HTTP server")
		testServer.Stop()
	}
	By("tearing down the test environment")
	_ = harness.TestEnv.Stop()
}
