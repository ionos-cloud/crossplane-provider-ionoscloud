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
	"k8s.io/client-go/util/workqueue"
	ctrl "sigs.k8s.io/controller-runtime"

	"github.com/ionos-cloud/crossplane-provider-ionoscloud/internal/controller/alb/applicationloadbalancer"
	albforwardingrule "github.com/ionos-cloud/crossplane-provider-ionoscloud/internal/controller/alb/forwardingrule"
	"github.com/ionos-cloud/crossplane-provider-ionoscloud/internal/controller/alb/targetgroup"
	"github.com/ionos-cloud/crossplane-provider-ionoscloud/internal/controller/backup/backupunit"
	"github.com/ionos-cloud/crossplane-provider-ionoscloud/internal/controller/compute/cubeserver"
	"github.com/ionos-cloud/crossplane-provider-ionoscloud/internal/controller/compute/datacenter"
	"github.com/ionos-cloud/crossplane-provider-ionoscloud/internal/controller/compute/firewallrule"
	"github.com/ionos-cloud/crossplane-provider-ionoscloud/internal/controller/compute/group"
	"github.com/ionos-cloud/crossplane-provider-ionoscloud/internal/controller/compute/ipblock"
	"github.com/ionos-cloud/crossplane-provider-ionoscloud/internal/controller/compute/ipfailover"
	"github.com/ionos-cloud/crossplane-provider-ionoscloud/internal/controller/compute/lan"
	"github.com/ionos-cloud/crossplane-provider-ionoscloud/internal/controller/compute/nic"
	pcc "github.com/ionos-cloud/crossplane-provider-ionoscloud/internal/controller/compute/privatecrossconnect"
	"github.com/ionos-cloud/crossplane-provider-ionoscloud/internal/controller/compute/s3key"
	"github.com/ionos-cloud/crossplane-provider-ionoscloud/internal/controller/compute/server"
	"github.com/ionos-cloud/crossplane-provider-ionoscloud/internal/controller/compute/user"
	"github.com/ionos-cloud/crossplane-provider-ionoscloud/internal/controller/compute/volume"
	"github.com/ionos-cloud/crossplane-provider-ionoscloud/internal/controller/dataplatform/dataplatformcluster"
	"github.com/ionos-cloud/crossplane-provider-ionoscloud/internal/controller/dataplatform/dataplatformnodepool"
	"github.com/ionos-cloud/crossplane-provider-ionoscloud/internal/controller/dbaas/mongocluster"
	"github.com/ionos-cloud/crossplane-provider-ionoscloud/internal/controller/dbaas/mongouser"
	"github.com/ionos-cloud/crossplane-provider-ionoscloud/internal/controller/dbaas/postgrescluster"
	"github.com/ionos-cloud/crossplane-provider-ionoscloud/internal/controller/dbaas/postgresuser"
	"github.com/ionos-cloud/crossplane-provider-ionoscloud/internal/controller/k8s/k8scluster"
	"github.com/ionos-cloud/crossplane-provider-ionoscloud/internal/controller/k8s/k8snodepool"
	"github.com/ionos-cloud/crossplane-provider-ionoscloud/internal/controller/nlb/flowlog"
	nlbforwardingrule "github.com/ionos-cloud/crossplane-provider-ionoscloud/internal/controller/nlb/forwardingrule"
	"github.com/ionos-cloud/crossplane-provider-ionoscloud/internal/controller/nlb/networkloadbalancer"

	"github.com/crossplane/crossplane-runtime/pkg/logging"

	"github.com/ionos-cloud/crossplane-provider-ionoscloud/internal/controller/config"
	"github.com/ionos-cloud/crossplane-provider-ionoscloud/internal/utils"
)

// controllerSetup is the func signature of the setup method of a controller.
type controllerSetup func(ctrl.Manager, logging.Logger, workqueue.RateLimiter, *utils.ConfigurationOptions) error

// controllers is a list of controllers which must be setup and initialized.
var controllers = []controllerSetup{
	datacenter.Setup,
	pcc.Setup,
	server.Setup,
	cubeserver.Setup,
	volume.Setup,
	nic.Setup,
	firewallrule.Setup,
	ipblock.Setup,
	ipfailover.Setup,
	k8scluster.Setup,
	k8snodepool.Setup,
	postgrescluster.Setup,
	lan.Setup,
	applicationloadbalancer.Setup,
	albforwardingrule.Setup,
	targetgroup.Setup,
	backupunit.Setup,
	s3key.Setup,
	user.Setup,
	postgresuser.Setup,
	mongocluster.Setup,
	mongouser.Setup,
	dataplatformcluster.Setup,
	dataplatformnodepool.Setup,
	group.Setup,
	networkloadbalancer.Setup,
	flowlog.Setup,
	nlbforwardingrule.Setup,
}

// Setup creates all IONOS Cloud controllers with the supplied logger
// and adds them to the supplied manager.
func Setup(mgr ctrl.Manager, l logging.Logger, wl workqueue.RateLimiter, options *utils.ConfigurationOptions) error {
	for _, setup := range controllers {
		if err := setup(mgr, l, wl, options); err != nil {
			return err
		}
	}
	return config.Setup(mgr, l, wl)
}
