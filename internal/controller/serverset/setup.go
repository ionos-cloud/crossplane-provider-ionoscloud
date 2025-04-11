package serverset

import (
	"time"

	xpv1 "github.com/crossplane/crossplane-runtime/apis/common/v1"
	"github.com/crossplane/crossplane-runtime/pkg/event"
	"github.com/crossplane/crossplane-runtime/pkg/reconciler/managed"
	"github.com/crossplane/crossplane-runtime/pkg/resource"
	v1 "k8s.io/api/core/v1"
	"math/rand/v2"
	ctrl "sigs.k8s.io/controller-runtime"

	"github.com/ionos-cloud/crossplane-provider-ionoscloud/apis/compute/v1alpha1"
	apisv1alpha1 "github.com/ionos-cloud/crossplane-provider-ionoscloud/apis/v1alpha1"
	"github.com/ionos-cloud/crossplane-provider-ionoscloud/internal/utils"
)

// SetupServerSet adds a controller that reconciles ServerSet managed resources.
func SetupServerSet(mgr ctrl.Manager, opts *utils.ConfigurationOptions) error {
	name := managed.ControllerName(v1alpha1.ServerSetGroupKind)
	logger := opts.CtrlOpts.Logger
	mapController := kubeConfigmapController{
		kube: mgr.GetClient(),
		log:  logger,
	}
	return ctrl.NewControllerManagedBy(mgr).
		Named(name).
		WithOptions(opts.CtrlOpts.ForControllerRuntime()).
		WithEventFilter(resource.DesiredStateChanged()).
		For(&v1alpha1.ServerSet{}).
		Complete(managed.NewReconciler(mgr,
			resource.ManagedKind(v1alpha1.ServerSetGroupVersionKind),
			managed.WithExternalConnecter(&connector{
				kube:                    mgr.GetClient(),
				kubeConfigmapController: &mapController,
				bootVolumeController: &kubeBootVolumeController{
					kube:          mgr.GetClient(),
					log:           logger,
					mapController: &mapController,
				},
				nicController: &kubeNicController{
					kube: mgr.GetClient(),
					log:  logger,
				},
				serverController: &kubeServerController{
					kube: mgr.GetClient(),
					log:  logger,
				},
				firewallRuleController: &kubeFirewallRuleController{
					kube: mgr.GetClient(),
					log:  logger,
				},

				usage: resource.NewProviderConfigUsageTracker(mgr.GetClient(), &apisv1alpha1.ProviderConfigUsage{}),
				log:   logger,
			}),
			managed.WithReferenceResolver(managed.NewAPISimpleReferenceResolver(mgr.GetClient())),
			managed.WithInitializers(),
			managed.WithPollInterval(opts.GetPollInterval()),
			managed.WithTimeout(opts.GetTimeout()),
			managed.WithCreationGracePeriod(opts.GetCreationGracePeriod()),
			managed.WithLogger(logger.WithValues("controller", name)),
			managed.WithPollIntervalHook(func(mg resource.Managed, pollInterval time.Duration) time.Duration {
				if mg.GetCondition(xpv1.TypeReady).Status != v1.ConditionTrue {
					// If the resource is not ready, we should poll more frequently not to delay time to readiness.
					pollInterval = 30 * time.Second
				}
				// This is the same as runtime default poll interval with jitter, see:
				// https://github.com/crossplane/crossplane-runtime/blob/7fcb8c5cad6fc4abb6649813b92ab92e1832d368/pkg/reconciler/managed/reconciler.go#L573
				return pollInterval + time.Duration((rand.Float64()-0.5)*2*float64(opts.PollJitter)) //nolint G404 // No need for secure randomness
			}),
			managed.WithRecorder(event.NewAPIRecorder(mgr.GetEventRecorderFor(name)))))
}
