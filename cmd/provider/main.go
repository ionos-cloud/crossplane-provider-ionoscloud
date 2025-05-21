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
	"fmt"
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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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
		app                        = kingpin.New(filepath.Base(os.Args[0]), "IONOS Cloud support for Crossplane.").DefaultEnvars()
		debug                      = app.Flag("debug", "Run with debug logging.").Short('d').Bool()
		uniqueNames                = app.Flag("unique-names", "Enable uniqueness name support for IONOS Cloud resources").Short('u').Default("false").Bool()
		syncInterval               = app.Flag("sync", "Controller manager sync interval such as 300ms, 1.5h, or 2h45m").Short('s').Default("1h").Duration()
		pollInterval               = app.Flag("poll", "Poll interval controls how often an individual resource should be checked for changes.").Default("1m").Duration()
		leaderElection             = app.Flag("leader-election", "Use leader election for the controller manager.").Short('l').Default("false").Envar("LEADER_ELECTION").Bool()
		createGracePeriod          = app.Flag("create-grace-period", "Grace period for creation of IONOS Cloud resources.").Default("1m").Duration()
		maxReconcileRate           = app.Flag("max-reconcile-rate", "The global maximum rate per second at which resources may checked for drift from the desired state.").Default("1").Int()
		timeout                    = app.Flag("timeout", "Timeout duration cumulatively for all the calls happening in the reconciliation functions.").Default("1h").Duration()
		pollStateMetricInterval    = app.Flag("poll-state-metric", "State metric recording interval").Default("5s").Duration()
		namespace                  = app.Flag("namespace", "Namespace used to set as default scope in default secret store config.").Default("crossplane-system").Envar("POD_NAMESPACE").String()
		enableExternalSecretStores = app.Flag("enable-external-secret-stores", "Enable support for ExternalSecretStores.").Default("false").Envar("ENABLE_EXTERNAL_SECRET_STORES").Bool()
		reconcileMap               = app.Flag("max-reconcile-rate-per-resource", "Overrides the max-reconcile-rate on a per resource basis. Use the Kind of the resource as the key.").PlaceHolder("nic:2").StringMap()
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

	log.Debug("Starting", "sync-period", syncInterval.String(), "poll-interval", pollInterval.String(), "max-reconcile-rate", *maxReconcileRate, "debug", *debug)

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
			ObjectMeta: metav1.ObjectMeta{
				Name: "default",
			},
			Spec: v1alpha1.StoreConfigSpec{
				SecretStoreConfig: xpv1.SecretStoreConfig{
					DefaultScope: *namespace,
				},
			},
		})), "cannot create default store config")
	}

	options := utils.NewConfigurationOptions(*timeout, *createGracePeriod, *uniqueNames, ctrlOpts)
	if len(*reconcileMap) > 0 {
		options.MaxReconcilesPerResource = make(map[string]int, len(*reconcileMap))
		// convert to lowercase and convert string to int
		for k, v := range *reconcileMap {
			reconcileRate, err := strconv.Atoi(v)
			kingpin.FatalIfError(err, fmt.Sprintf("Cannot convert maxReconcileRate for %s, value (%s) from string to int", k, v))
			options.MaxReconcilesPerResource[strings.ToLower(k)] = reconcileRate
		}
	}

	kingpin.FatalIfError(controller.Setup(mgr, options), "Cannot setup IONOS Cloud controllers")
	kingpin.FatalIfError(mgr.Start(ctrl.SetupSignalHandler()), "Cannot start controller manager")
}
