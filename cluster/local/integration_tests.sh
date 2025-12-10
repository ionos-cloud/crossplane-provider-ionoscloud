#!/usr/bin/env bash
set -e

# add prints functions
source ./cluster/local/print.sh
# add integration tests for resources
source ./cluster/local/integration_tests_provider.sh
source ./cluster/local/integration_tests_compute.sh
source ./cluster/local/integration_tests_alb.sh
source ./cluster/local/integration_tests_dbaas_postgres.sh
source ./cluster/local/integration_tests_dbaas_mongo.sh
source ./cluster/local/integration_tests_k8s.sh
source ./cluster/local/integration_tests_backup.sh
source ./cluster/local/integration_tests_serverset.sh
source ./cluster/local/integration_tests_statefulserverset.sh

# ------------------------------
projectdir="$(cd "$(dirname "${BASH_SOURCE[0]}")"/../.. && pwd)"

# get the build environment variables from the special build.vars target in the main makefile
eval $(make --no-print-directory -C ${projectdir} build.vars)

# ------------------------------
# You can select which tests to run.
# Pay attention to the default values that are set!
# To run specific tests, for example for dbaas resources,
# use: make e2e TEST_COMPUTE=false TEST_DBAAS=true
TEST_COMPUTE=${TEST_COMPUTE:-true}
# by default, do not test the following resources
# since it takes a lot of time
TEST_DBAAS=${TEST_DBAAS:-false}
TEST_MONGO=${TEST_MONGO:-false}
TEST_POSTGRES=${TEST_POSTGRES:-false}
TEST_K8S=${TEST_K8S:-false}
TEST_ALB=${TEST_ALB:-false}
TEST_NLB=${TEST_NLB:-false}
TEST_BACKUP=${TEST_BACKUP:-false}
TEST_SERVERSET=${TEST_SERVERSET:-false}
TEST_STATEFULSERVERSET=${TEST_STATEFULSERVERSET:-false}
skipcleanup=${skipcleanup:-false}
KIND_CLUSTER_NAME=${KIND_CLUSTER_NAME:-${PROJECT_NAME}-dev}
CROSSPLANE_NAMESPACE=${CROSSPLANE_NAMESPACE:-crossplane-system}

# ------------------------------

# cleanup on exit
if [ "$skipcleanup" != true ]; then
  function cleanup() {
    export KUBECONFIG=
    "${KIND}" delete cluster --name="${KIND_CLUSTER_NAME}"
  }

  trap cleanup EXIT
fi

# ----------- integration tests
echo_step "--- INTEGRATION TESTS ---"

# install package
echo_step "--- install Crossplane Provider IONOSCLOUD config ---"
install_provider

if [ "$TEST_COMPUTE" = true ]; then
  echo_step "--- COMPUTE ENGINE TESTS ---"
  echo_step "--- ipblock tests ---"
  ipblock_tests
  echo_step "--- datacenter tests ---"
  datacenter_tests
  echo_step "--- lan tests ---"
  lan_tests
  echo_step "--- volume tests ---"
  volume_tests
  echo_step "--- server tests ---"
  server_tests
  echo_step "--- nic tests ---"
  nic_tests
  echo_step "--- firewallrule tests ---"
  firewallrule_tests
  echo_step "--- ipfailover tests ---"
  ipfailover_tests
  echo_step "--- CLEANING UP COMPUTE ENGINE TESTS ---"
  echo_step "--- cleanup firewallrule tests ---"
  firewallrule_tests_cleanup
  echo_step "--- cleanup ipfailover tests ---"
  ipfailover_tests_cleanup
  echo_step "--- cleanup nic tests ---"
  nic_tests_cleanup
  echo_step "--- cleanup lan tests ---"
  lan_tests_cleanup
  echo_step "--- cleanup volume tests ---"
  volume_tests_cleanup
  echo_step "--- cleanup server tests ---"
  server_tests_cleanup
  echo_step "--- cleanup datacenter tests ---"
  datacenter_tests_cleanup
  echo_step "--- cleanup ipblock tests ---"
  ipblock_tests_cleanup
  echo_step "--- cleanup pcc resource part of lan tests ---"
  pcc_tests_cleanup
fi

if [ "$TEST_DBAAS" = true ]; then
  echo_step "--- DBAAS POSTGRES TESTS ---"
  echo_step "--- dbaas postgres cluster tests ---"
  dbaas_postgres_cluster_tests
  echo_step "--- CLEANING UP DBAAS POSTGRES TESTS ---"
  echo_step "--- dbaas postgres cluster tests ---"
  dbaas_postgres_cluster_tests_cleanup
  echo_step "--- DBAAS MONGO TESTS ---"
  echo_step "--- dbaas postgres cluster tests ---"
  dbaas_mongo_cluster_tests
  echo_step "--- CLEANING UP DBAAS MONGO TESTS ---"
  echo_step "--- dbaas postgres cluster tests ---"
  dbaas_mongo_cluster_tests_cleanup
fi

if [ "$TEST_POSTGRES" = true ]; then
  echo_step "--- DBAAS POSTGRES TESTS ---"
  echo_step "--- dbaas postgres cluster tests ---"
  dbaas_postgres_cluster_tests
  echo_step "--- CLEANING UP DBAAS POSTGRES TESTS ---"
  echo_step "--- dbaas postgres cluster tests ---"
  dbaas_postgres_cluster_tests_cleanup
fi

if [ "$TEST_MONGO" = true ]; then
  echo_step "--- DBAAS MONGO TESTS ---"
  echo_step "--- dbaas mongo cluster tests ---"
  dbaas_mongo_cluster_tests
  echo_step "--- CLEANING UP DBAAS MONGO TESTS ---"
  echo_step "--- dbaas mongo cluster tests ---"
  dbaas_mongo_cluster_tests_cleanup
fi

if [ "$TEST_K8S" = true ]; then
  echo_step "--- K8S TESTS ---"
  echo_step "--- k8s cluster tests ---"
  k8s_cluster_tests
  echo_step "--- k8s nodepool tests ---"
  k8s_nodepool_tests
  echo_step "--- CLEANING UP K8S TESTS ---"
  echo_step "--- k8s nodepool tests ---"
  k8s_nodepool_tests_cleanup
  echo_step "--- k8s cluster tests ---"
  k8s_cluster_tests_cleanup
fi

if [ "$TEST_ALB" = true ]; then
  echo_step "--- ALB TESTS ---"
  echo_step "--- target group tests ---"
  targetgroup_tests
  echo_step "--- application load balancer tests ---"
  alb_tests
  echo_step "--- forwarding rule tests ---"
  forwardingrule_tests
  echo_step "--- CLEANING UP ALB TESTS---"
  echo_step "--- forwarding rule tests ---"
  forwardingrule_tests_cleanup
  echo_step "--- application load balancer tests ---"
  alb_tests_cleanup
  echo_step "--- target group tests ---"
  targetgroup_tests_cleanup
fi

if [ "$TEST_NLB" = true ]; then
  echo_step "--- NLB TESTS ---"
  echo_step "--- network load balancer tests ---"
  nlb_tests
  echo_step "--- nlb forwarding rule tests ---"
  nlbforwardingrule_tests
  echo_step "--- nlb flow log tests ---"
  nlbflowlog_tests
  echo_step "--- CLEANING UP NLB TESTS---"
  echo_step "--- flow log tests ---"
  nlbflowlog_tests_cleanup
  echo_step "--- forwarding rule tests ---"
  nlbforwardingrule_tests_cleanup
  echo_step "--- network load balancer tests ---"
  nlb_tests_cleanup
fi

if [ "$TEST_BACKUP" = true ]; then
  echo_step "--- BACKUP TESTS ---"
  echo_step "--- backupunit tests ---"
  backupunit_tests
  echo_step "--- CLEANING UP BACKUP TESTS ---"
  echo_step "--- backupunit tests ---"
  backupunit_tests_cleanup
fi

if [ "$TEST_SERVERSET" = true ]; then
  echo_step "--- SERVERSET TESTS ---"
  serverset_tests
  echo_step "--- CLEANING UP SERVERSET TESTS ---"
  serverset_tests_cleanup
fi

if [ "$TEST_STATEFULSERVERSET" = true ]; then
  echo_step "--- STATEFULSERVERSET TESTS ---"
  statefulserverset_tests
  echo_step "--- CLEANING UP STATEFULSERVERSET TESTS ---"
  statefulserverset_tests_cleanup
fi

echo_step "-------------------"
echo_step "--- CLEANING UP ---"
echo_step "-------------------"

# uninstalling Crossplane Provider IONOS Cloud
echo_step "--- uninstalling ${PROJECT_NAME} ---"
uninstall_provider

echo_success "Integration tests succeeded!"
