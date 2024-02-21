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
	"os"
	"path/filepath"

	"gopkg.in/alecthomas/kingpin.v2"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/cache"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"

	"github.com/crossplane/crossplane-runtime/pkg/logging"
	"github.com/crossplane/crossplane-runtime/pkg/ratelimiter"

	"github.com/ionos-cloud/crossplane-provider-ionoscloud/apis"
	"github.com/ionos-cloud/crossplane-provider-ionoscloud/internal/controller"
	"github.com/ionos-cloud/crossplane-provider-ionoscloud/internal/utils"
)

const (
	// defaultProviderRPS is the recommended default average requeues per
	// second tolerated by a Crossplane provider.
	defaultProviderRPS = 1
)

func main() {
	var (
		app               = kingpin.New(filepath.Base(os.Args[0]), "IONOS Cloud support for Crossplane.").DefaultEnvars()
		debug             = app.Flag("debug", "Run with debug logging.").Short('d').Bool()
		uniqueNames       = app.Flag("unique-names", "Enable uniqueness name support for IONOS Cloud resources").Short('u').Bool()
		syncInterval      = app.Flag("sync", "Controller manager sync interval such as 300ms, 1.5h, or 2h45m").Short('s').Default("1h").Duration()
		pollInterval      = app.Flag("poll", "Poll interval controls how often an individual resource should be checked for changes.").Default("1m").Duration()
		leaderElection    = app.Flag("leader-election", "Use leader election for the controller manager.").Short('l').Default("false").Envar("LEADER_ELECTION").Bool()
		createGracePeriod = app.Flag("create-grace-period", "Grace period for creation of IONOS Cloud resources.").Default("1m").Duration()
		timeout           = app.Flag("timeout", "Timeout duration cumulatively for all the calls happening in the reconciliation functions.").Default("30m").Duration()
	)
	kingpin.MustParse(app.Parse(os.Args[1:]))

	zl := zap.New(zap.UseDevMode(*debug))
	log := logging.NewLogrLogger(zl.WithName("provider-ionoscloud"))
	if *debug {
		// The controller-runtime runs with a no-op logger by default. It is
		// *very* verbose even at info level, so we only provide it a real
		// logger when we're running in debug mode.
		ctrl.SetLogger(zl)
	}

	log.Debug("Starting", "sync-period", syncInterval.String())

	cfg, err := ctrl.GetConfig()
	kingpin.FatalIfError(err, "Cannot get API server rest config")

	mgr, err := ctrl.NewManager(cfg, ctrl.Options{
		LeaderElection:   *leaderElection,
		LeaderElectionID: "crossplane-leader-election-provider-ionoscloud",
		Cache:            cache.Options{SyncPeriod: syncInterval},
	})
	kingpin.FatalIfError(err, "Cannot create controller manager")

	rl := ratelimiter.NewGlobal(defaultProviderRPS)
	kingpin.FatalIfError(apis.AddToScheme(mgr.GetScheme()), "Cannot add IONOS Cloud APIs to scheme")
	options := utils.NewConfigurationOptions(*pollInterval, *createGracePeriod, *timeout, *uniqueNames)
	kingpin.FatalIfError(controller.Setup(mgr, log, rl, options), "Cannot setup IONOS Cloud controllers")
	kingpin.FatalIfError(mgr.Start(ctrl.SetupSignalHandler()), "Cannot start controller manager")
}
