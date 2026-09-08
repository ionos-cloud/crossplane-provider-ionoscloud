/*
Copyright 2020 The Crossplane Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package main

import (
	"context"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	xpv1 "github.com/crossplane/crossplane-runtime/apis/common/v1"
	xpcontroller "github.com/crossplane/crossplane-runtime/pkg/controller"
	"github.com/crossplane/crossplane-runtime/pkg/feature"
	"github.com/crossplane/crossplane-runtime/pkg/logging"
	"github.com/crossplane/crossplane-runtime/pkg/ratelimiter"
	"github.com/crossplane/crossplane-runtime/pkg/reconciler/managed"
	"github.com/crossplane/crossplane-runtime/pkg/resource"
	"github.com/crossplane/crossplane-runtime/pkg/statemetrics"
	"gopkg.in/alecthomas/kingpin.v2"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/cache"
	"sigs.k8s.io/controller-runtime/pkg/config"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	"sigs.k8s.io/controller-runtime/pkg/metrics"

	"github.com/ionos-cloud/crossplane-provider-ionoscloud/apis"
	"github.com/ionos-cloud/crossplane-provider-ionoscloud/apis/v1alpha1"
	"github.com/ionos-cloud/crossplane-provider-ionoscloud/internal/controller"
	"github.com/ionos-cloud/crossplane-provider-ionoscloud/internal/features"
	"github.com/ionos-cloud/crossplane-provider-ionoscloud/internal/utils"
)

func main() {
	var (
		app                               = kingpin.New(filepath.Base(os.Args[0]), "IONOS Cloud support for Crossplane.").DefaultEnvars()
		debug                             = app.Flag("debug", "Run with debug logging.").Short('d').Bool()
		uniqueNames                       = app.Flag("unique-names", "Enable uniqueness name support for IONOS Cloud resources").Short('u').Default("false").Bool()
		syncInterval                      = app.Flag("sync", "Controller manager sync interval such as 300ms, 1.5h, or 2h45m").Short('s').Default("1h").Duration()
		pollInterval                      = app.Flag("poll", "Poll interval controls how often an individual resource should be checked for changes.").Default("1m").Duration()
		leaderElection                    = app.Flag("leader-election", "Use leader election for the controller manager.").Short('l').Default("false").Envar("LEADER_ELECTION").Bool()
		createGracePeriod                 = app.Flag("create-grace-period", "Grace period for creation of IONOS Cloud resources.").Default("1m").Duration()
		maxReconcileRate                  = app.Flag("max-reconcile-rate", "The global maximum rate per second at which resources may checked for drift from the desired state.").Default("10").Int()
		timeout                           = app.Flag("timeout", "Timeout duration cumulatively for all the calls happening in the reconciliation functions.").Default("1h").Duration()
		vmRebootTimeoutFlag               = app.Flag("vm-reboot-timeout", "Timeout to wait for a VM to reboot and report as ready again after a failover, for ServerSets using a custom state map. This budget is shared across every replica that needs to reboot within the same reconcile call (e.g. an image update touching all replicas) - it is not a fresh window per replica. If set higher than --timeout, --extend-serverset-timeout-for-vm-reboot must also be set, or the provider will refuse to start.")
		vmRebootTimeout                   = vmRebootTimeoutFlag.Default("120m").Duration()
		extendServerSetTimeoutForVMReboot = app.Flag("extend-serverset-timeout-for-vm-reboot", "If true, widens the ServerSet controller's own reconcile timeout to be at least --vm-reboot-timeout, so a long VM reboot wait is not cut short by --timeout. Only affects ServerSet reconciles; every other resource kind keeps using --timeout unchanged.").Default("false").Bool()
		pollStateMetricInterval           = app.Flag("poll-state-metric", "State metric recording interval").Default("5s").Duration()
		pollJitterPercentage              = app.Flag("poll-jitter-percentage", "Percentage of the poll interval by which an individual resource's poll interval may be randomly shifted, in either direction. 0 disables jitter, and it must be less than 100.").Default("0").Uint()
		pollNotReady                      = app.Flag("poll-not-ready", "Poll interval for a resource that is not ready yet, used in place of --poll while it is not ready. 0 leaves resources that are not ready on the --poll interval.").Default("0").Duration()
		namespace                         = app.Flag("namespace", "Namespace used to set as default scope in default secret store config.").Default("crossplane-system").Envar("POD_NAMESPACE").String()
		enableExternalSecretStores        = app.Flag("enable-external-secret-stores", "Enable support for ExternalSecretStores.").Default("false").Envar("ENABLE_EXTERNAL_SECRET_STORES").Bool()
		reconcileMap                      = app.Flag("max-reconcile-rate-per-resource", "Overrides the max-reconcile-rate on a per resource basis. Use the Kind of the resource as the key.").PlaceHolder("nic:2").StringMap()
	)
	kingpin.MustParse(app.Parse(os.Args[1:]))
	zl := zap.New(zap.UseDevMode(*debug))
	log := logging.NewLogrLogger(zl.WithName("provider-ionoscloud"))
	if *debug {
		// The controller-runtime runs with a no-op logger by default. It is
		// *very* verbose even at info level, so we only provide it a real
		// logger when we're running in debug mode.
		ctrl.SetLogger(zl)
	} else {
		// explicitly provide a no-op logger by default, otherwise controller-runtime gives a warning
		ctrl.SetLogger(zap.New(zap.WriteTo(io.Discard)))
	}

	log.Debug("Starting", "sync-period", syncInterval.String(), "poll-interval", pollInterval.String(), "poll-jitter-percentage", *pollJitterPercentage, "poll-not-ready", pollNotReady.String(), "max-reconcile-rate", *maxReconcileRate, "debug", *debug)

	// vmRebootTimeout's default (120m) is intentionally larger than timeout's default (1h) to
	// preserve the pre-existing hardcoded behavior, so only validate when the user actually
	// chose the value themselves - otherwise every deployment that never touches this flag
	// (most of them, since it only matters for ServerSets using a custom state map) would fail
	// to start. There's no legitimate reason to explicitly set vmRebootTimeout above timeout
	// without also enabling extendServerSetTimeoutForVMReboot: without it, the value is
	// silently capped and has no effect at all, so this can only ever be a mistake.
	if isFlagSetByUser(app, vmRebootTimeoutFlag, os.Args[1:]) && *vmRebootTimeout > *timeout && !*extendServerSetTimeoutForVMReboot {
		kingpin.Fatalf("--vm-reboot-timeout (%s) exceeds --timeout (%s) but --extend-serverset-timeout-for-vm-reboot is not set; "+
			"the VM reboot wait would be silently capped at ~--timeout and have no effect. "+
			"Either lower --vm-reboot-timeout or pass --extend-serverset-timeout-for-vm-reboot",
			vmRebootTimeout, timeout)
	}

	cfg, err := ctrl.GetConfig()
	kingpin.FatalIfError(err, "Cannot get API server rest config")
	skipControllerNameValidation := true
	mgr, err := ctrl.NewManager(cfg, ctrl.Options{
		LeaderElection: *leaderElection,
		Controller: config.Controller{
			SkipNameValidation: &skipControllerNameValidation,
		},
		LeaderElectionID: "crossplane-leader-election-provider-ionoscloud",
		Cache:            cache.Options{SyncPeriod: syncInterval},
		LeaseDuration:    func() *time.Duration { d := 60 * time.Second; return &d }(),
		RenewDeadline:    func() *time.Duration { d := 50 * time.Second; return &d }(),
	})
	kingpin.FatalIfError(err, "Cannot create controller manager")
	kingpin.FatalIfError(apis.AddToScheme(mgr.GetScheme()), "Cannot add IONOS Cloud APIs to scheme")
	metricRecorder := managed.NewMRMetricRecorder()
	stateMetrics := statemetrics.NewMRStateMetrics()

	metrics.Registry.MustRegister(metricRecorder)
	metrics.Registry.MustRegister(stateMetrics)
	mo := xpcontroller.MetricOptions{
		PollStateMetricInterval: *pollStateMetricInterval,
		MRMetrics:               metricRecorder,
		MRStateMetrics:          stateMetrics,
	}

	ctrlOpts := xpcontroller.Options{
		Logger:                  log,
		MaxConcurrentReconciles: *maxReconcileRate,
		PollInterval:            *pollInterval,
		GlobalRateLimiter:       ratelimiter.NewGlobal(*maxReconcileRate),
		Features:                &feature.Flags{},
		MetricOptions:           &mo,
	}

	if *enableExternalSecretStores {
		ctrlOpts.Features.Enable(features.EnableAlphaExternalSecretStores)
		log.Info("Alpha feature enabled", "flag", features.EnableAlphaExternalSecretStores)

		kingpin.FatalIfError(resource.Ignore(kerrors.IsAlreadyExists, mgr.GetClient().Create(context.Background(), &v1alpha1.StoreConfig{
			Name: "default",
			Spec: v1alpha1.StoreConfigSpec{
				SecretStoreConfig: xpv1.SecretStoreConfig{
					DefaultScope: *namespace,
				},
			},
		})), "cannot create default store config")
	}

	kingpin.FatalIfError(utils.ValidatePollJitterPercentage(*pollJitterPercentage), "Invalid --poll-jitter-percentage")

	options := utils.NewConfigurationOptions(*timeout, *createGracePeriod, *uniqueNames, ctrlOpts)
	options.PollJitterPercentage = *pollJitterPercentage
	options.NotReadyPollInterval = *pollNotReady
	options.VMRebootTimeout = *vmRebootTimeout
	options.ExtendServerSetTimeoutForVMReboot = *extendServerSetTimeoutForVMReboot
	if len(*reconcileMap) > 0 {
		options.MaxReconcilesPerResource = make(map[string]int, len(*reconcileMap))
		// convert to lowercase and convert string to int
		for k, v := range *reconcileMap {
			reconcileRate, err := strconv.Atoi(v)
			kingpin.FatalIfError(err, "Cannot convert maxReconcileRate for %s, value (%s) from string to int", k, v)
			options.MaxReconcilesPerResource[strings.ToLower(k)] = reconcileRate
		}
	}

	kingpin.FatalIfError(controller.Setup(mgr, options), "Cannot setup IONOS Cloud controllers")
	kingpin.FatalIfError(mgr.Start(ctrl.SetupSignalHandler()), "Cannot start controller manager")
}

// isFlagSetByUser reports whether flag was explicitly provided in args or via its envar, as
// opposed to just carrying its default value.
func isFlagSetByUser(app *kingpin.Application, flag *kingpin.FlagClause, args []string) bool {
	pctx, err := app.ParseContext(args)
	if err != nil {
		return false
	}
	// HasEnvarValue is only reliable once ParseContext has resolved the flag's default-envar
	// name as a side effect of Application.init, hence the ordering above.
	if flag.HasEnvarValue() {
		return true
	}
	for _, e := range pctx.Elements {
		if e.Clause == flag {
			return true
		}
	}
	return false
}
