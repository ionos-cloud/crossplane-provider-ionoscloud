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

package controller

import (
	"time"

	"k8s.io/client-go/util/workqueue"
	ctrl "sigs.k8s.io/controller-runtime"

	"github.com/crossplane/crossplane-runtime/pkg/logging"

	"github.com/ionos-cloud/crossplane-provider-ionoscloud/internal/controller/compute/cubeserver"
	"github.com/ionos-cloud/crossplane-provider-ionoscloud/internal/controller/compute/datacenter"
	"github.com/ionos-cloud/crossplane-provider-ionoscloud/internal/controller/compute/firewallrule"
	"github.com/ionos-cloud/crossplane-provider-ionoscloud/internal/controller/compute/ipblock"
	"github.com/ionos-cloud/crossplane-provider-ionoscloud/internal/controller/compute/ipfailover"
	"github.com/ionos-cloud/crossplane-provider-ionoscloud/internal/controller/compute/lan"
	"github.com/ionos-cloud/crossplane-provider-ionoscloud/internal/controller/compute/nic"
	"github.com/ionos-cloud/crossplane-provider-ionoscloud/internal/controller/compute/server"
	"github.com/ionos-cloud/crossplane-provider-ionoscloud/internal/controller/compute/volume"
	"github.com/ionos-cloud/crossplane-provider-ionoscloud/internal/controller/config"
	"github.com/ionos-cloud/crossplane-provider-ionoscloud/internal/controller/dbaas/postgrescluster"
	"github.com/ionos-cloud/crossplane-provider-ionoscloud/internal/controller/k8s/k8scluster"
	"github.com/ionos-cloud/crossplane-provider-ionoscloud/internal/controller/k8s/k8snodepool"
)

// Setup creates all IONOS Cloud controllers with the supplied logger
// and adds them to the supplied manager.
func Setup(mgr ctrl.Manager, l logging.Logger, wl workqueue.RateLimiter, poll time.Duration) error {
	for _, setup := range []func(ctrl.Manager, logging.Logger, workqueue.RateLimiter, time.Duration) error{
		datacenter.Setup,
		server.Setup,
		cubeserver.Setup,
		volume.Setup,
		lan.Setup,
		nic.Setup,
		firewallrule.Setup,
		ipblock.Setup,
		ipfailover.Setup,
		k8scluster.Setup,
		k8snodepool.Setup,
		postgrescluster.Setup,
	} {
		if err := setup(mgr, l, wl, poll); err != nil {
			return err
		}
	}
	return config.Setup(mgr, l, wl)
}
