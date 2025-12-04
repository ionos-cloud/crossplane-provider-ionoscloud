//go:build sss_e2e

package statefulserverset

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/crossplane/crossplane-runtime/pkg/ratelimiter"
	"github.com/crossplane/crossplane-runtime/pkg/reconciler/managed"
	"github.com/crossplane/crossplane-runtime/pkg/statemetrics"
	"k8s.io/utils/ptr"

	xpv1 "github.com/crossplane/crossplane-runtime/apis/common/v1"
	xpcontroller "github.com/crossplane/crossplane-runtime/pkg/controller"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"

	"github.com/crossplane/crossplane-runtime/pkg/controller"
	"github.com/crossplane/crossplane-runtime/pkg/logging"

	"github.com/ionos-cloud/crossplane-provider-ionoscloud/apis"
	"github.com/ionos-cloud/crossplane-provider-ionoscloud/apis/compute/v1alpha1"
	apisv1alpha1 "github.com/ionos-cloud/crossplane-provider-ionoscloud/apis/v1alpha1"
	"github.com/ionos-cloud/crossplane-provider-ionoscloud/internal/controller/compute/firewallrule"
	"github.com/ionos-cloud/crossplane-provider-ionoscloud/internal/controller/compute/nic"
	"github.com/ionos-cloud/crossplane-provider-ionoscloud/internal/controller/compute/server"
	"github.com/ionos-cloud/crossplane-provider-ionoscloud/internal/controller/serverset"
	"github.com/ionos-cloud/crossplane-provider-ionoscloud/internal/controller/volumeselector"

	"github.com/ionos-cloud/crossplane-provider-ionoscloud/internal/controller/compute/datacenter"
	"github.com/ionos-cloud/crossplane-provider-ionoscloud/internal/controller/compute/lan"
	"github.com/ionos-cloud/crossplane-provider-ionoscloud/internal/controller/compute/volume"
	"github.com/ionos-cloud/crossplane-provider-ionoscloud/internal/utils"

	corev1 "k8s.io/api/core/v1"
)

var (
	cfg       *rest.Config
	k8sClient client.Client
	testEnv   *envtest.Environment
	ctx       context.Context
	cancel    context.CancelFunc
	mgr       ctrl.Manager
)

var logger = zap.New(zap.UseDevMode(true))

const (
	timeout        = time.Minute * 20
	cleanupTimeout = 2 * time.Minute
	interval       = time.Second * 30 // Poll every 30 seconds
)

func TestSuccessfulCreation_E2E(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "StatefulServerSet Controller Integration Suite")
}

var _ = BeforeSuite(func() {

	// Setup logging with debug level, timestamps, and caller information
	logf.SetLogger(logger)
	ctx, cancel = context.WithCancel(context.Background())

	By("bootstrapping test environment")
	testEnv = &envtest.Environment{
		CRDDirectoryPaths: []string{
			filepath.Join("..", "..", "..", "..", "package", "crds"),
		},
		ErrorIfCRDPathMissing: true,
		BinaryAssetsDirectory: filepath.Join(os.TempDir(), "envtest-binaries"),
	}

	var err error
	cfg, err = testEnv.Start()
	Expect(err).NotTo(HaveOccurred())
	Expect(cfg).NotTo(BeNil())

	err = apis.AddToScheme(scheme.Scheme)
	Expect(err).NotTo(HaveOccurred())

	k8sClient, err = client.New(cfg, client.Options{Scheme: scheme.Scheme})
	Expect(err).NotTo(HaveOccurred())
	Expect(k8sClient).NotTo(BeNil())

	// Start the controller manager
	By("creating controller manager")
	ctrl.SetLogger(logger)
	mgr, err = ctrl.NewManager(cfg, ctrl.Options{
		Scheme: scheme.Scheme,
		Logger: logger,
	})
	Expect(err).NotTo(HaveOccurred())
	metricRecorder := managed.NewMRMetricRecorder()
	stateMetrics := statemetrics.NewMRStateMetrics()
	// Setup all the necessary controllers
	By("setting up controllers")
	reconciles := 4
	opts := &utils.ConfigurationOptions{
		CreationGracePeriod: 30 * time.Second,
		Timeout:             30 * time.Minute,
		CtrlOpts: controller.Options{
			PollInterval:            time.Minute,
			GlobalRateLimiter:       ratelimiter.NewGlobal(reconciles),
			Logger:                  logging.NewLogrLogger(logger),
			MaxConcurrentReconciles: reconciles,
			MetricOptions: &xpcontroller.MetricOptions{
				MRStateMetrics:          stateMetrics,
				PollStateMetricInterval: time.Minute * 5,
				MRMetrics:               metricRecorder,
			},
		},
	}

	By("setting up Datacenter controller")
	err = datacenter.Setup(mgr, opts)
	Expect(err).NotTo(HaveOccurred())

	By("setting up LAN controller")
	err = lan.Setup(mgr, opts)
	Expect(err).NotTo(HaveOccurred())

	By("setting up nic controller")
	err = nic.Setup(mgr, opts)
	Expect(err).NotTo(HaveOccurred())

	By("setting up Volume controller")
	err = volume.Setup(mgr, opts)
	Expect(err).NotTo(HaveOccurred())

	By("setting up server controller")
	err = server.Setup(mgr, opts)
	Expect(err).NotTo(HaveOccurred())

	By("setting up firewall rule controller")
	err = firewallrule.Setup(mgr, opts)
	Expect(err).NotTo(HaveOccurred())

	By("setting up volumeselector controller")
	err = volumeselector.Setup(mgr, opts)
	Expect(err).NotTo(HaveOccurred())

	By("setting up Serverset controller")
	err = serverset.SetupServerSet(mgr, opts)
	Expect(err).NotTo(HaveOccurred())

	By("setting up StatefulServerSet controller")
	err = Setup(mgr, opts)
	Expect(err).NotTo(HaveOccurred())

	// Start the manager in a goroutine
	By("starting controller manager")
	go func() {
		defer GinkgoRecover()
		err = mgr.Start(ctx)
		Expect(err).NotTo(HaveOccurred())
	}()

	// Wait for the manager to be ready
	Eventually(func() bool {
		return mgr.GetCache().WaitForCacheSync(ctx)
	}, timeout, interval).Should(BeTrue())
})

var _ = AfterSuite(func() {
	cancel()
	By("tearing down the test environment")
	_ = testEnv.Stop()
})

// createProviderConfigWithCredentials creates a ProviderConfig and its credentials Secret
// from environment variables: IONOS_USERNAME, IONOS_PASSWORD, IONOS_TOKEN
func createProviderConfigWithCredentials(ctx context.Context, name, namespace string) error {
	// Get credentials from environment variables
	token := os.Getenv("IONOS_TOKEN")

	// If no credentials are provided, panic
	if token == "" {
		GinkgoWriter.Printf("WARNING: No credentials found in environment variables (IONOS_USERNAME, IONOS_PASSWORD, IONOS_TOKEN)\n")
		GinkgoWriter.Printf("The controller will not be able to create cloud resources without credentials\n")
		Panic()
	}

	// Build the credentials JSON string
	// Format: {"token":"xxx","user":"xxx","password":"xxx","s3_access_key":"xxx","s3_secret_key":"xxx"}
	credentialsJSON := fmt.Sprintf(`{"token":"%s"}`, token)

	// Create the credentials Secret
	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Type: corev1.SecretTypeOpaque,
		Data: map[string][]byte{
			"credentials": []byte(credentialsJSON),
		},
	}

	if err := k8sClient.Create(ctx, secret); err != nil {
		return fmt.Errorf("failed to create credentials secret: %w", err)
	}

	DeferCleanup(func(ctx context.Context) {
		By("cleaning up provider config secret")
		Eventually(ctx, func() error {
			return client.IgnoreNotFound(k8sClient.Delete(ctx, secret))
		}).Should(Succeed())
	}, NodeTimeout(time.Minute))

	// Create the ProviderConfig
	providerConfig := &apisv1alpha1.ProviderConfig{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
		Spec: apisv1alpha1.ProviderConfigSpec{
			Credentials: apisv1alpha1.ProviderCredentials{
				Source: xpv1.CredentialsSourceSecret,
				CommonCredentialSelectors: xpv1.CommonCredentialSelectors{
					SecretRef: &xpv1.SecretKeySelector{
						Key: "credentials",
						SecretReference: xpv1.SecretReference{
							Name:      secret.Name,
							Namespace: namespace,
						},
					},
				},
			},
		},
	}

	if err := k8sClient.Create(ctx, providerConfig); err != nil {
		return fmt.Errorf("failed to create ProviderConfig: %w", err)
	}

	DeferCleanup(func(ctx context.Context) {
		By("cleaning up provider config")
		Eventually(ctx, func() error {
			return client.IgnoreNotFound(k8sClient.Delete(ctx, providerConfig))
		}).Should(Succeed())
	}, NodeTimeout(time.Minute))

	return nil
}

// This test verifies that a StatefulServerSet can be created on the API server
// and that the resource is properly validated and stored.
// The controller will create actual cloud resources if credentials are provided
// via environment variables: IONOS_TOKEN
var _ = Describe("StatefulServerSet Successful creation", func() {
	Context("When creating a StatefulServerSet", func() {
		It("should create the StatefulServerSet resource successfully", func() {
			ctx, cancel = context.WithCancel(context.Background())
			defer cancel()

			// 1. Create ProviderConfig with credentials from environment variables
			By("creating ProviderConfig with credentials from environment")
			err := createProviderConfigWithCredentials(ctx, "example", "default")
			Expect(err).NotTo(HaveOccurred())

			// 2. Create a Datacenter resource that the StatefulServerSet references
			datacenter := &v1alpha1.Datacenter{
				ObjectMeta: metav1.ObjectMeta{
					Name: "example",
				},
				Spec: v1alpha1.DatacenterSpec{
					ForProvider: v1alpha1.DatacenterParameters{
						Name:     "sss-test-datacenter",
						Location: "de/txl",
					},
					ResourceSpec: xpv1.ResourceSpec{
						ProviderConfigReference: &xpv1.Reference{
							Name: "example",
						},
					},
				},
			}
			Expect(k8sClient.Create(ctx, datacenter)).Should(Succeed())
			// 4. Verify that the datacenter was created successfully
			fetchedDC := &v1alpha1.Datacenter{}
			Eventually(func() error {
				return k8sClient.Get(ctx, types.NamespacedName{Name: datacenter.Name, Namespace: ""}, fetchedDC)
			}, timeout, interval).Should(Succeed())
			// 3. Define and create the StatefulServerSet resource
			crName := "sss-example"
			cr := &v1alpha1.StatefulServerSet{
				ObjectMeta: metav1.ObjectMeta{
					Name: crName,
				},
				Spec: v1alpha1.StatefulServerSetSpec{
					ForProvider: v1alpha1.StatefulServerSetParameters{
						Replicas:              1,
						RemovePendingOnReboot: false,
						DeploymentStrategy: v1alpha1.DeploymentStrategy{
							Type: "ZONES",
						},
						DatacenterCfg: v1alpha1.DatacenterConfig{
							DatacenterIDRef: &xpv1.Reference{
								Name: "example",
							},
						},
						Template: v1alpha1.ServerSetTemplate{
							Metadata: v1alpha1.ServerSetMetadata{
								Name: "server-name",
							},
							Spec: v1alpha1.ServerSetTemplateSpec{
								Cores: 1,
								RAM:   1024,
								NICs: []v1alpha1.ServerSetTemplateNIC{
									{
										Name:           "nic-customer",
										DHCP:           false,
										DHCPv6:         ptr.To(false),
										LanReference:   "customer",
										FirewallActive: true,
										FirewallRules: []v1alpha1.ServerSetTemplateFirewallRuleSpec{
											{
												Protocol: "TCP",
												Name:     "rule-tcp",
											},
											{
												Protocol: "ICMP",
												Name:     "rule-icmp",
											},
										},
									},
								},
							},
						},
						IdentityConfigMap: v1alpha1.IdentityConfigMap{
							Name:      "config-lease",
							Namespace: "default",
							KeyName:   "identity",
						},
						BootVolumeTemplate: v1alpha1.BootVolumeTemplate{
							Metadata: v1alpha1.ServerSetBootVolumeMetadata{
								Name: "boot-volume",
							},
							Spec: v1alpha1.ServerSetBootVolumeSpec{
								UpdateStrategy: v1alpha1.UpdateStrategy{
									Stype: "createBeforeDestroyBootVolume",
								},
								SetHotPlugsFromImage: false,
								Image:                "c38292f2-eeaa-11ef-8fa7-aee9942a25aa",
								Size:                 10,
								Type:                 "SSD",
								UserData:             "",
								ImagePassword:        "thisshouldwork11",
								Substitutions: []v1alpha1.Substitution{
									{
										Options: map[string]string{
											"cidr": "fd1d:15db:cf64:1337::/64",
										},
										Key:    "__ipv6Address",
										Type:   "ipv6Address",
										Unique: true,
									},
									{
										Options: map[string]string{
											"cidr": "192.168.42.0/24",
										},
										Key:    "ipv4Address",
										Type:   "ipv4Address",
										Unique: true,
									},
								},
							},
						},
						Lans: []v1alpha1.StatefulServerSetLan{
							{
								Metadata: v1alpha1.StatefulServerSetLanMetadata{
									Name: "data",
								},
								Spec: v1alpha1.StatefulServerSetLanSpec{
									Public: true,
								},
							},
							{
								Metadata: v1alpha1.StatefulServerSetLanMetadata{
									Name: "management",
								},
								Spec: v1alpha1.StatefulServerSetLanSpec{
									Public: false,
								},
							},
							{
								Metadata: v1alpha1.StatefulServerSetLanMetadata{
									Name: "customer",
								},
								Spec: v1alpha1.StatefulServerSetLanSpec{
									IPv6cidr: "AUTO",
									Public:   false,
								},
							},
						},
						Volumes: []v1alpha1.StatefulServerSetVolume{
							{
								Metadata: v1alpha1.StatefulServerSetVolumeMetadata{
									Name: "storage-disk",
								},
								Spec: v1alpha1.StatefulServerSetVolumeSpec{
									Size: 10,
									Type: "SSD",
								},
							},
							{
								Metadata: v1alpha1.StatefulServerSetVolumeMetadata{
									Name: "second-storage-disk",
								},
								Spec: v1alpha1.StatefulServerSetVolumeSpec{
									Size: 40,
									Type: "SSD",
								},
							},
						},
					},
					ResourceSpec: xpv1.ResourceSpec{
						ProviderConfigReference: &xpv1.Reference{
							Name: "example",
						},
					},
				},
			}

			Expect(k8sClient.Create(ctx, cr)).Should(Succeed())

			// 4. Verify that the StatefulServerSet was created successfully
			fetchedCR := &v1alpha1.StatefulServerSet{}
			Eventually(func() error {
				return k8sClient.Get(ctx, types.NamespacedName{Name: crName, Namespace: ""}, fetchedCR)
			}, timeout, interval).Should(Succeed())

			// 5. Verify the StatefulServerSet spec was correctly stored
			Expect(fetchedCR.Spec.ForProvider.Replicas).To(Equal(1))
			Expect(fetchedCR.Spec.ForProvider.Template.Metadata.Name).To(Equal("server-name"))
			Expect(fetchedCR.Spec.ForProvider.Template.Spec.Cores).To(Equal(int32(1)))
			Expect(fetchedCR.Spec.ForProvider.Template.Spec.RAM).To(Equal(int32(1024)))

			// 6. Verify the LANs are correctly defined
			Expect(fetchedCR.Spec.ForProvider.Lans).To(HaveLen(3))
			Expect(fetchedCR.Spec.ForProvider.Lans[0].Metadata.Name).To(Equal("data"))
			Expect(fetchedCR.Spec.ForProvider.Lans[0].Spec.Public).To(BeTrue())
			Expect(fetchedCR.Spec.ForProvider.Lans[1].Metadata.Name).To(Equal("management"))
			Expect(fetchedCR.Spec.ForProvider.Lans[1].Spec.Public).To(BeFalse())
			Expect(fetchedCR.Spec.ForProvider.Lans[2].Metadata.Name).To(Equal("customer"))
			Expect(fetchedCR.Spec.ForProvider.Lans[2].Spec.IPv6cidr).To(Equal("AUTO"))

			// 7. Verify the Volumes are correctly defined
			Expect(fetchedCR.Spec.ForProvider.Volumes).To(HaveLen(2))
			Expect(fetchedCR.Spec.ForProvider.Volumes[0].Metadata.Name).To(Equal("storage-disk"))
			Expect(fetchedCR.Spec.ForProvider.Volumes[0].Spec.Size).To(Equal(float32(10)))
			Expect(fetchedCR.Spec.ForProvider.Volumes[1].Metadata.Name).To(Equal("second-storage-disk"))
			Expect(fetchedCR.Spec.ForProvider.Volumes[1].Spec.Size).To(Equal(float32(40)))

			// 8. Verify boot volume configuration
			Expect(fetchedCR.Spec.ForProvider.BootVolumeTemplate).NotTo(BeNil())
			Expect(fetchedCR.Spec.ForProvider.BootVolumeTemplate.Metadata.Name).To(Equal("boot-volume"))
			Expect(fetchedCR.Spec.ForProvider.BootVolumeTemplate.Spec.Image).To(Equal("c38292f2-eeaa-11ef-8fa7-aee9942a25aa"))
			Expect(fetchedCR.Spec.ForProvider.BootVolumeTemplate.Spec.Size).To(Equal(float32(10)))
			Expect(fetchedCR.Spec.ForProvider.BootVolumeTemplate.Spec.Substitutions).To(HaveLen(2))

			// 9. Verify NICs configuration
			Expect(fetchedCR.Spec.ForProvider.Template.Spec.NICs).To(HaveLen(1))
			Expect(fetchedCR.Spec.ForProvider.Template.Spec.NICs[0].Name).To(Equal("nic-customer"))
			Expect(fetchedCR.Spec.ForProvider.Template.Spec.NICs[0].LanReference).To(Equal("customer"))
			Expect(fetchedCR.Spec.ForProvider.Template.Spec.NICs[0].FirewallRules).To(HaveLen(2))

			// 10. Wait for the controller to reconcile and create dependent resources
			By("waiting for controller to create dependent resources")

			// Check that LANs are created
			Eventually(func() int {
				lanList := &v1alpha1.LanList{}
				if err := k8sClient.List(ctx, lanList); err != nil {
					return 0
				}
				return len(lanList.Items)
			}, timeout, interval).Should(BeNumerically(">=", 3), "At least 3 LANs should be created")

			// Check that Volumes are created
			Eventually(func() int {
				volumeList := &v1alpha1.VolumeList{}
				if err := k8sClient.List(ctx, volumeList); err != nil {
					return 0
				}
				return len(volumeList.Items)
			}, timeout, interval).Should(BeNumerically(">=", 2), "At least 2 data volumes should be created")

			Eventually(func() int {
				list := &v1alpha1.NicList{}
				if err := k8sClient.List(ctx, list); err != nil {
					return 0
				}
				return len(list.Items)
			}, timeout, interval).Should(BeNumerically(">=", 1), "At least 1 nic should be created")

			Eventually(func() int {
				list := &v1alpha1.ServerSetList{}
				if err := k8sClient.List(ctx, list); err != nil {
					return 0
				}
				return len(list.Items)
			}, timeout, interval).Should(BeNumerically(">=", 1), "At least 1 server should be created")

			Eventually(func() int {
				list := &v1alpha1.FirewallRuleList{}
				if err := k8sClient.List(ctx, list); err != nil {
					return 0
				}
				return len(list.Items)
			}, timeout, interval).Should(BeNumerically(">=", 1), "At least 1 fw rule should be created")

			// check that serverset is in available state
			Eventually(func() bool {
				serverSetList := &v1alpha1.ServerSetList{}
				if err := k8sClient.List(ctx, serverSetList); err != nil {
					return false
				}
				if len(serverSetList.Items) == 0 {
					return false
				}
				return serverSetList.Items[0].Status.GetCondition(xpv1.TypeReady).Equal(xpv1.Available())
			}, timeout, interval).Should(BeTrue(), "ServerSet should be in available state")

			// check that vs is in available state
			Eventually(func() bool {
				volumeSelectorList := &v1alpha1.VolumeselectorList{}
				if err := k8sClient.List(ctx, volumeSelectorList); err != nil {
					return false
				}
				if len(volumeSelectorList.Items) == 0 {
					return false
				}
				return volumeSelectorList.Items[0].Status.GetCondition(xpv1.TypeReady).Equal(xpv1.Available())
			}, timeout, interval).Should(BeTrue(), "VolumeSelector should be created by the controller")

			// Verify the StatefulServerSet status is updated
			Eventually(func() bool {
				if err := k8sClient.Get(ctx, types.NamespacedName{Name: crName, Namespace: ""}, fetchedCR); err != nil {
					return false
				}
				return len(fetchedCR.Status.Conditions) > 0 && fetchedCR.Status.GetCondition(xpv1.TypeReady).Equal(xpv1.Available())
			}, timeout, interval).Should(BeTrue(), "StatefulServerSet should be in available state")

			By("cleaning up resources")
			DeferCleanup(func(ctx context.Context) {
				By("cleaning up StatefulServerSet")
				Expect(client.IgnoreNotFound(k8sClient.Delete(ctx, cr))).To(Succeed())
			}, NodeTimeout(cleanupTimeout))
			DeferCleanup(func(ctx context.Context) {
				By("cleaning up Datacenter")
				Expect(client.IgnoreNotFound(k8sClient.Delete(ctx, datacenter))).To(Succeed())
			}, NodeTimeout(cleanupTimeout))
		})
	})
})
