package serverset

import (
	"github.com/crossplane/crossplane-runtime/pkg/errors"
	"github.com/crossplane/crossplane-runtime/pkg/event"
	"github.com/crossplane/crossplane-runtime/pkg/ratelimiter"
	"github.com/crossplane/crossplane-runtime/pkg/reconciler/managed"
	"github.com/crossplane/crossplane-runtime/pkg/resource"
	"github.com/crossplane/crossplane-runtime/pkg/statemetrics"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/controller"

	"github.com/ionos-cloud/crossplane-provider-ionoscloud/apis/compute/v1alpha1"
	apisv1alpha1 "github.com/ionos-cloud/crossplane-provider-ionoscloud/apis/v1alpha1"
	"github.com/ionos-cloud/crossplane-provider-ionoscloud/internal/utils"
)

// SetupServerSet adds a controller that reconciles ServerSet managed resources.
func SetupServerSet(mgr ctrl.Manager, opts *utils.ConfigurationOptions) error {
	name := managed.ControllerName(v1alpha1.ServerSetGroupKind)
	logger := opts.CtrlOpts.Logger
	if opts.CtrlOpts.MetricOptions != nil && opts.CtrlOpts.MetricOptions.MRStateMetrics != nil {
		stateMetricsRecorder := statemetrics.NewMRStateRecorder(
			mgr.GetClient(), opts.CtrlOpts.Logger, opts.CtrlOpts.MetricOptions.MRStateMetrics, &v1alpha1.ServerSetList{}, opts.CtrlOpts.MetricOptions.PollStateMetricInterval,
		)
		if err := mgr.Add(stateMetricsRecorder); err != nil {
			return errors.Wrap(err, "cannot register MR state metrics recorder for kind"+name)
		}
	}
	mapController := kubeConfigmapController{
		kube: mgr.GetClient(),
		log:  logger,
	}
	reconcileTimeout := opts.GetTimeout()
	if opts.GetExtendServerSetTimeoutForVMReboot() {
		reconcileTimeout = max(reconcileTimeout, opts.GetVMRebootTimeout())
	}
	return ctrl.NewControllerManagedBy(mgr).
		Named(name).
		WithEventFilter(utils.DesiredStateChanged()).
		WithOptions(controller.Options{
			MaxConcurrentReconciles: opts.GetMaxConcurrentReconcileRate(v1alpha1.ServerSetKind),
			RateLimiter:             ratelimiter.NewController(),
			RecoverPanic:            new(true),
		}).
		For(&v1alpha1.ServerSet{}).
		Complete(managed.NewReconciler(mgr,
			resource.ManagedKind(v1alpha1.ServerSetGroupVersionKind),
			managed.WithExternalConnecter(&connector{
				kube:                    mgr.GetClient(),
				kubeConfigmapController: &mapController,
				vmRebootTimeout:         opts.GetVMRebootTimeout(),
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
			managed.WithTimeout(reconcileTimeout),
			managed.WithCreationGracePeriod(opts.GetCreationGracePeriod()),
			managed.WithLogger(logger.WithValues("controller", name)),
			managed.WithPollIntervalHook(opts.PollIntervalHook()),
			managed.WithMetricRecorder(opts.CtrlOpts.MetricOptions.MRMetrics),
			managed.WithRecorder(event.NewAPIRecorder(mgr.GetEventRecorderFor(name)))))
}
