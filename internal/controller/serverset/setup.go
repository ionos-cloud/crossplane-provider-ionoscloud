package serverset

import (
	"github.com/crossplane/crossplane-runtime/pkg/event"
	"github.com/crossplane/crossplane-runtime/pkg/reconciler/managed"
	"github.com/crossplane/crossplane-runtime/pkg/resource"
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
			managed.WithRecorder(event.NewAPIRecorder(mgr.GetEventRecorderFor(name)))))
}
