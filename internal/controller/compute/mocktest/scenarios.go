package mocktest

import (
	"context"
	"fmt"
	"time"

	xpv1 "github.com/crossplane/crossplane-runtime/apis/common/v1"
	"github.com/crossplane/crossplane-runtime/pkg/meta"
	. "github.com/onsi/ginkgo/v2" //nolint:staticcheck
	. "github.com/onsi/gomega"    //nolint:staticcheck
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/ionos-cloud/crossplane-provider-ionoscloud/internal/clients/compute"
)

// ScenarioConfig provides the resource-specific callbacks needed to run the 10 standard test scenarios.
// K8sClient and TestServer are functions (not values) because Ginkgo evaluates the top-level
// Describe body before BeforeSuite runs, so these package-level vars are still nil at registration time.
type ScenarioConfig struct {
	ResourceName string // e.g. "LAN", "Server"
	CRPrefix     string // e.g. "test-lan", "test-srv"
	K8sClient    func() client.Client
	TestServer   func() *TestHTTPServer

	// CR operations (resource-specific via closures)
	CreateCR               func(name string) client.Object
	CreateCRWithAnnotation func(name, externalName, annotationValue string) client.Object
	GetCR                  func(ctx context.Context, name string) (client.Object, error)
	GetState               func(obj client.Object) string
	GetResourceID          func(obj client.Object) string
	MutateForUpdate        func(obj client.Object)

	// Fake service operations
	StoreResource  func(externalID string)
	RemoveResource func(externalID string)
	SetError       func(method string, err error)
	ClearErrors    func()
	GetDeleteCalls func() []string
}

// RunStandardScenarios registers 10 Ginkgo Describe blocks covering the standard
// lifecycle test scenarios shared across all controller mock tests.
func RunStandardScenarios(cfg ScenarioConfig) {
	scenario1(cfg)
	scenario2(cfg)
	scenario3(cfg)
	scenario4(cfg)
	scenario5(cfg)
	scenario6(cfg)
	scenario7(cfg)
	scenario8(cfg)
	scenario9(cfg)
	scenario10(cfg)
}

func scenario1(cfg ScenarioConfig) {
	Describe("Scenario 1: Successful creation lifecycle", Ordered, func() {
		var crName string

		BeforeAll(func() {
			cfg.TestServer().SetMode(StatusModeDone)
			cfg.ClearErrors()
			crName = cfg.CRPrefix + "-create"
		})

		AfterAll(func() {
			obj, err := cfg.GetCR(context.Background(), crName)
			if err == nil {
				_ = cfg.K8sClient().Delete(context.Background(), obj)
			}
		})

		It(fmt.Sprintf("should create a %s CR and reconcile to AVAILABLE", cfg.ResourceName), func() {
			ctx := context.Background()
			cr := cfg.CreateCR(crName)
			Expect(cfg.K8sClient().Create(ctx, cr)).Should(Succeed())

			Eventually(func(g Gomega) {
				fetched, err := cfg.GetCR(ctx, crName)
				g.Expect(err).NotTo(HaveOccurred())
				g.Expect(meta.GetExternalName(fetched)).NotTo(BeEmpty())
				g.Expect(cfg.GetResourceID(fetched)).NotTo(BeEmpty())
				g.Expect(cfg.GetState(fetched)).To(Equal("AVAILABLE"))
				g.Expect(fetched.(ConditionedObject).GetCondition(xpv1.TypeReady).Equal(xpv1.Available())).To(BeTrue())
			}, Timeout, Interval).Should(Succeed())
		})
	})
}

func scenario2(cfg ScenarioConfig) {
	Describe("Scenario 2: Observe stability after create", Ordered, func() {
		var crName string

		BeforeAll(func() {
			cfg.TestServer().SetMode(StatusModeDone)
			cfg.ClearErrors()
			crName = cfg.CRPrefix + "-stable"
		})

		AfterAll(func() {
			obj, err := cfg.GetCR(context.Background(), crName)
			if err == nil {
				_ = cfg.K8sClient().Delete(context.Background(), obj)
			}
		})

		It("should stay stable after creation", func() {
			ctx := context.Background()
			cr := cfg.CreateCR(crName)
			Expect(cfg.K8sClient().Create(ctx, cr)).Should(Succeed())

			Eventually(func(g Gomega) {
				fetched, err := cfg.GetCR(ctx, crName)
				g.Expect(err).NotTo(HaveOccurred())
				g.Expect(cfg.GetState(fetched)).To(Equal("AVAILABLE"))
			}, Timeout, Interval).Should(Succeed())

			fetched, err := cfg.GetCR(ctx, crName)
			Expect(err).NotTo(HaveOccurred())
			gen := fetched.(GenerationObject).GetGeneration()

			Consistently(func(g Gomega) {
				fetched, err := cfg.GetCR(ctx, crName)
				g.Expect(err).NotTo(HaveOccurred())
				g.Expect(cfg.GetState(fetched)).To(Equal("AVAILABLE"))
				g.Expect(fetched.(GenerationObject).GetGeneration()).To(Equal(gen))
			}, 5*time.Second, 1*time.Second).Should(Succeed())
		})
	})
}

func scenario3(cfg ScenarioConfig) {
	Describe("Scenario 3: Update lifecycle", Ordered, func() {
		var crName string

		BeforeAll(func() {
			cfg.TestServer().SetMode(StatusModeDone)
			cfg.ClearErrors()
			crName = cfg.CRPrefix + "-update"
		})

		AfterAll(func() {
			obj, err := cfg.GetCR(context.Background(), crName)
			if err == nil {
				_ = cfg.K8sClient().Delete(context.Background(), obj)
			}
		})

		It(fmt.Sprintf("should update the %s when spec changes", cfg.ResourceName), func() {
			ctx := context.Background()
			cr := cfg.CreateCR(crName)
			Expect(cfg.K8sClient().Create(ctx, cr)).Should(Succeed())

			Eventually(func(g Gomega) {
				fetched, err := cfg.GetCR(ctx, crName)
				g.Expect(err).NotTo(HaveOccurred())
				g.Expect(cfg.GetState(fetched)).To(Equal("AVAILABLE"))
			}, Timeout, Interval).Should(Succeed())

			By(fmt.Sprintf("updating %s spec", cfg.ResourceName))
			Eventually(func(g Gomega) {
				fetched, err := cfg.GetCR(ctx, crName)
				g.Expect(err).NotTo(HaveOccurred())
				cfg.MutateForUpdate(fetched)
				g.Expect(cfg.K8sClient().Update(ctx, fetched)).To(Succeed())
			}, Timeout, Interval).Should(Succeed())

			Eventually(func(g Gomega) {
				fetched, err := cfg.GetCR(ctx, crName)
				g.Expect(err).NotTo(HaveOccurred())
				g.Expect(cfg.GetState(fetched)).To(Equal("AVAILABLE"))
			}, Timeout, Interval).Should(Succeed())
		})
	})
}

func scenario4(cfg ScenarioConfig) {
	Describe("Scenario 4: Delete lifecycle", Ordered, func() {
		var crName string

		BeforeAll(func() {
			cfg.TestServer().SetMode(StatusModeDone)
			cfg.ClearErrors()
			crName = cfg.CRPrefix + "-delete"
		})

		It(fmt.Sprintf("should delete the %s CR and remove it from K8s", cfg.ResourceName), func() {
			ctx := context.Background()
			cr := cfg.CreateCR(crName)
			Expect(cfg.K8sClient().Create(ctx, cr)).Should(Succeed())

			Eventually(func(g Gomega) {
				fetched, err := cfg.GetCR(ctx, crName)
				g.Expect(err).NotTo(HaveOccurred())
				g.Expect(cfg.GetState(fetched)).To(Equal("AVAILABLE"))
			}, Timeout, Interval).Should(Succeed())

			By(fmt.Sprintf("deleting the %s CR", cfg.ResourceName))
			fetched, err := cfg.GetCR(ctx, crName)
			Expect(err).NotTo(HaveOccurred())
			Expect(cfg.K8sClient().Delete(ctx, fetched)).Should(Succeed())

			Eventually(func() bool {
				_, err := cfg.GetCR(ctx, crName)
				return err != nil
			}, Timeout, Interval).Should(BeTrue())

			deleteCalls := cfg.GetDeleteCalls()
			Expect(len(deleteCalls)).To(BeNumerically(">", 0))
		})
	})
}

func scenario5(cfg ScenarioConfig) {
	Describe("Scenario 5: Create error — API returns error", Ordered, func() {
		var crName string

		BeforeAll(func() {
			cfg.TestServer().SetMode(StatusModeDone)
			crName = cfg.CRPrefix + "-create-err"
		})

		AfterAll(func() {
			cfg.ClearErrors()
			obj, err := cfg.GetCR(context.Background(), crName)
			if err == nil {
				_ = cfg.K8sClient().Delete(context.Background(), obj)
				Eventually(func() bool {
					_, err := cfg.GetCR(context.Background(), crName)
					return err != nil
				}, Timeout, Interval).Should(BeTrue())
			}
		})

		It("should fail then recover when error is cleared", func() {
			ctx := context.Background()
			cfg.SetError(MethodCreate, fmt.Errorf("simulated create error"))

			cr := cfg.CreateCR(crName)
			Expect(cfg.K8sClient().Create(ctx, cr)).Should(Succeed())

			Consistently(func(g Gomega) {
				fetched, err := cfg.GetCR(ctx, crName)
				g.Expect(err).NotTo(HaveOccurred())
				g.Expect(cfg.GetState(fetched)).NotTo(Equal("AVAILABLE"))
			}, 5*time.Second, 1*time.Second).Should(Succeed())

			By("clearing the create error")
			cfg.ClearErrors()

			Eventually(func(g Gomega) {
				fetched, err := cfg.GetCR(ctx, crName)
				g.Expect(err).NotTo(HaveOccurred())
				g.Expect(cfg.GetState(fetched)).To(Equal("AVAILABLE"))
			}, Timeout, Interval).Should(Succeed())
		})
	})
}

func scenario6(cfg ScenarioConfig) {
	Describe("Scenario 6: WaitForRequest error during Create", Ordered, func() {
		var crName string

		BeforeAll(func() {
			crName = cfg.CRPrefix + "-waitreq-err"
			cfg.ClearErrors()
		})

		AfterAll(func() {
			cfg.TestServer().SetMode(StatusModeDone)
			cfg.ClearErrors()
			obj, err := cfg.GetCR(context.Background(), crName)
			if err == nil {
				_ = cfg.K8sClient().Delete(context.Background(), obj)
				Eventually(func() bool {
					_, err := cfg.GetCR(context.Background(), crName)
					return err != nil
				}, Timeout, Interval).Should(BeTrue())
			}
		})

		It("should recover via Observe after WaitForRequest fails", func() {
			ctx := context.Background()
			cfg.TestServer().SetMode(StatusModeRunning)

			cr := cfg.CreateCR(crName)
			Expect(cfg.K8sClient().Create(ctx, cr)).Should(Succeed())

			Eventually(func(g Gomega) {
				fetched, err := cfg.GetCR(ctx, crName)
				g.Expect(err).NotTo(HaveOccurred())
				g.Expect(meta.GetExternalName(fetched)).NotTo(BeEmpty())
			}, Timeout, Interval).Should(Succeed())

			By("switching HTTP server to DONE")
			cfg.TestServer().SetMode(StatusModeDone)

			Eventually(func(g Gomega) {
				fetched, err := cfg.GetCR(ctx, crName)
				g.Expect(err).NotTo(HaveOccurred())
				g.Expect(cfg.GetState(fetched)).To(Equal("AVAILABLE"))
			}, Timeout, Interval).Should(Succeed())
		})
	})
}

func scenario7(cfg ScenarioConfig) {
	Describe("Scenario 7: IsRequestDone — request still running", Ordered, func() {
		var crName string
		const resourceID = "prereq-running-1"

		BeforeAll(func() {
			crName = cfg.CRPrefix + "-isreqdone-running"
			cfg.ClearErrors()
		})

		AfterAll(func() {
			cfg.TestServer().SetMode(StatusModeDone)
			cfg.ClearErrors()
			obj, err := cfg.GetCR(context.Background(), crName)
			if err == nil {
				_ = cfg.K8sClient().Delete(context.Background(), obj)
				Eventually(func() bool {
					_, err := cfg.GetCR(context.Background(), crName)
					return err != nil
				}, Timeout, Interval).Should(BeTrue())
			}
		})

		It("should wait for request then reconcile successfully", func() {
			ctx := context.Background()
			cfg.TestServer().SetMode(StatusModeRunning)

			cr := cfg.CreateCRWithAnnotation(crName, resourceID, "simulated-post-req-1")
			Expect(cfg.K8sClient().Create(ctx, cr)).Should(Succeed())

			Consistently(func(g Gomega) {
				fetched, err := cfg.GetCR(ctx, crName)
				g.Expect(err).NotTo(HaveOccurred())
				g.Expect(cfg.GetState(fetched)).NotTo(Equal("AVAILABLE"))
			}, 5*time.Second, 500*time.Millisecond).Should(Succeed())

			By(fmt.Sprintf("switching HTTP server to DONE and materializing the %s", cfg.ResourceName))
			cfg.TestServer().SetMode(StatusModeDone)
			cfg.StoreResource(resourceID)

			Eventually(func(g Gomega) {
				fetched, err := cfg.GetCR(ctx, crName)
				g.Expect(err).NotTo(HaveOccurred())
				g.Expect(cfg.GetState(fetched)).To(Equal("AVAILABLE"))
			}, Timeout, Interval).Should(Succeed())
		})
	})
}

func scenario8(cfg ScenarioConfig) {
	Describe("Scenario 8: IsRequestDone — request failed", Ordered, func() {
		var crName string

		BeforeAll(func() {
			crName = cfg.CRPrefix + "-isreqdone-failed"
			cfg.ClearErrors()
		})

		AfterAll(func() {
			cfg.TestServer().SetMode(StatusModeDone)
			cfg.ClearErrors()
			obj, err := cfg.GetCR(context.Background(), crName)
			if err == nil {
				_ = cfg.K8sClient().Delete(context.Background(), obj)
				Eventually(func() bool {
					_, err := cfg.GetCR(context.Background(), crName)
					return err != nil
				}, Timeout, Interval).Should(BeTrue())
			}
		})

		It("should propagate error when request status is FAILED", func() {
			ctx := context.Background()
			cfg.TestServer().SetMode(StatusModeFailed)

			cr := cfg.CreateCRWithAnnotation(crName, "prereq-failed-1", "simulated-post-req-failed")
			Expect(cfg.K8sClient().Create(ctx, cr)).Should(Succeed())

			Eventually(func(g Gomega) {
				fetched, err := cfg.GetCR(ctx, crName)
				g.Expect(err).NotTo(HaveOccurred())
				syncedCond := fetched.(ConditionedObject).GetCondition(xpv1.TypeSynced)
				g.Expect(syncedCond.Status).To(Equal(corev1.ConditionFalse))
			}, Timeout, Interval).Should(Succeed())

			Consistently(func(g Gomega) {
				fetched, err := cfg.GetCR(ctx, crName)
				g.Expect(err).NotTo(HaveOccurred())
				readyCond := fetched.(ConditionedObject).GetCondition(xpv1.TypeReady)
				g.Expect(readyCond.Equal(xpv1.Available())).To(BeFalse())
			}, 3*time.Second, 500*time.Millisecond).Should(Succeed())
		})
	})
}

func scenario9(cfg ScenarioConfig) {
	Describe("Scenario 9: IsRequestDone — 404 lost request", Ordered, func() {
		var crName string

		BeforeAll(func() {
			crName = cfg.CRPrefix + "-isreqdone-404"
			cfg.ClearErrors()
		})

		AfterAll(func() {
			cfg.TestServer().SetMode(StatusModeDone)
			cfg.ClearErrors()
			obj, err := cfg.GetCR(context.Background(), crName)
			if err == nil {
				_ = cfg.K8sClient().Delete(context.Background(), obj)
				Eventually(func() bool {
					_, err := cfg.GetCR(context.Background(), crName)
					return err != nil
				}, Timeout, Interval).Should(BeTrue())
			}
		})

		It("should recover after annotation is manually removed", func() {
			ctx := context.Background()
			cfg.TestServer().SetMode(StatusMode404)

			cr := cfg.CreateCRWithAnnotation(crName, "", "simulated-post-req-lost")
			Expect(cfg.K8sClient().Create(ctx, cr)).Should(Succeed())

			Consistently(func(g Gomega) {
				fetched, err := cfg.GetCR(ctx, crName)
				g.Expect(err).NotTo(HaveOccurred())
				g.Expect(cfg.GetState(fetched)).NotTo(Equal("AVAILABLE"))
			}, 5*time.Second, 1*time.Second).Should(Succeed())

			By("removing the POST request ID annotation")
			cfg.TestServer().SetMode(StatusModeDone)
			Eventually(func(g Gomega) {
				fetched, err := cfg.GetCR(ctx, crName)
				g.Expect(err).NotTo(HaveOccurred())
				annotations := fetched.(AnnotatedObject).GetAnnotations()
				delete(annotations, compute.POSTRequestIDAnnotationKey)
				fetched.(AnnotatedObject).SetAnnotations(annotations)
				g.Expect(cfg.K8sClient().Update(ctx, fetched)).To(Succeed())
			}, Timeout, Interval).Should(Succeed())

			Eventually(func(g Gomega) {
				fetched, err := cfg.GetCR(ctx, crName)
				g.Expect(err).NotTo(HaveOccurred())
				g.Expect(cfg.GetState(fetched)).To(Equal("AVAILABLE"))
			}, Timeout, Interval).Should(Succeed())
		})
	})
}

func scenario10(cfg ScenarioConfig) {
	Describe("Scenario 10: Delete when API returns 404", Ordered, func() {
		var crName string

		BeforeAll(func() {
			cfg.TestServer().SetMode(StatusModeDone)
			cfg.ClearErrors()
			crName = cfg.CRPrefix + "-delete-404"
		})

		AfterAll(func() {
			cfg.ClearErrors()
		})

		It("should handle 404 gracefully on delete", func() {
			ctx := context.Background()
			cr := cfg.CreateCR(crName)
			Expect(cfg.K8sClient().Create(ctx, cr)).Should(Succeed())

			Eventually(func(g Gomega) {
				fetched, err := cfg.GetCR(ctx, crName)
				g.Expect(err).NotTo(HaveOccurred())
				g.Expect(cfg.GetState(fetched)).To(Equal("AVAILABLE"))
			}, Timeout, Interval).Should(Succeed())

			fetched, err := cfg.GetCR(ctx, crName)
			Expect(err).NotTo(HaveOccurred())
			externalID := meta.GetExternalName(fetched)
			cfg.RemoveResource(externalID)
			cfg.SetError(MethodDelete, fmt.Errorf("%s not found", cfg.ResourceName))

			By(fmt.Sprintf("deleting the %s CR", cfg.ResourceName))
			Expect(cfg.K8sClient().Delete(ctx, fetched)).Should(Succeed())

			Eventually(func() bool {
				_, err := cfg.GetCR(ctx, crName)
				return err != nil
			}, Timeout, Interval).Should(BeTrue())
		})
	})
}

// ConditionedObject allows access to crossplane conditions on a client.Object.
type ConditionedObject interface {
	GetCondition(ct xpv1.ConditionType) xpv1.Condition
}

// GenerationObject allows access to the generation field.
type GenerationObject interface {
	GetGeneration() int64
}

// AnnotatedObject allows access to annotations.
type AnnotatedObject interface {
	GetAnnotations() map[string]string
	SetAnnotations(annotations map[string]string)
}
