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
source ./cluster/local/integration_tests_dataplatform.sh
source ./cluster/local/integration_tests_serverset.sh
source ./cluster/local/integration_tests_statefulserverset.sh

# ------------------------------
projectdir="$(cd "$(dirname "${BASH_SOURCE[0]}")"/../.. && pwd)"

# get the build environment variables from the special build.vars target in the main makefile
eval $(make --no-print-directory -C ${projectdir} build.vars)
eval $(make --no-print-directory -C ${projectdir} build.vars)

# ------------------------------

REGISTRY=${REGISTRY:-ghcr.io}
ORG_NAME=${ORG_NAME:-ionos-cloud}
BUILD_IMAGE="${REGISTRY}/${ORG_NAME}/${PROJECT_NAME}"
PACKAGE_IMAGE="crossplane.io/inttests/${PROJECT_NAME}:${VERSION}"
CONTROLLER_IMAGE="${REGISTRY}/${ORG_NAME}/${PROJECT_NAME}-controller"

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
TEST_DATAPLATFORM=${TEST_DATAPLATFORM:-false}
TEST_SERVERSET=${TEST_SERVERSET:-false}
TEST_STATEFULSERVERSET=${TEST_STATEFULSERVERSET:-true}
skipcleanup=${skipcleanup:-false}

version_tag="$(cat ${projectdir}/_output/version)"
# tag as latest version to load into kind cluster
PACKAGE_CONTROLLER_IMAGE="${REGISTRY}/${ORG_NAME}/${PROJECT_NAME}-controller:${VERSION}"
PACKAGE_PROVIDER_IMAGE="${REGISTRY}/${ORG_NAME}/${PROJECT_NAME}:${VERSION}"
K8S_CLUSTER="${K8S_CLUSTER:-${BUILD_REGISTRY}-inttests}"
KIND_NODE_IMAGE_TAG="${KIND_NODE_IMAGE_TAG:-v1.31.1}"

CROSSPLANE_NAMESPACE="crossplane-system"
PACKAGE_NAME="provider-ionoscloud"

# cleanup on exit
if [ "$skipcleanup" != true ]; then
  function cleanup() {
    export KUBECONFIG=
    "${KIND}" delete cluster --name="${K8S_CLUSTER}"
  }

  trap cleanup EXIT
fi

# setup package cache
echo_step "setting up local package cache"
CACHE_PATH="${projectdir}/.work/inttest-package-cache"
mkdir -p "${CACHE_PATH}"
echo "created cache dir at ${CACHE_PATH}"
docker tag "${BUILD_IMAGE}" "${PACKAGE_IMAGE}"
"${UP}" xpkg xp-extract --from-daemon "${PACKAGE_IMAGE}" -o "${CACHE_PATH}/${PACKAGE_NAME}.gz" && chmod 644 "${CACHE_PATH}/${PACKAGE_NAME}.gz"

# create kind cluster with extra mounts
KIND_NODE_IMAGE="kindest/node:${KIND_NODE_IMAGE_TAG}"
echo_step "creating k8s cluster using kind ${KIND_VERSION} and node image ${KIND_NODE_IMAGE}"
KIND_CONFIG="$(
  cat <<EOF
kind: Cluster
apiVersion: kind.x-k8s.io/v1alpha4
nodes:
- role: control-plane
  extraMounts:
  - hostPath: "${CACHE_PATH}/"
    containerPath: /cache
EOF
)"
echo "${KIND_CONFIG}" | "${KIND}" create cluster --name="${K8S_CLUSTER}" --wait=5m --image="${KIND_NODE_IMAGE}" --config=-

# tag controller images and load them into the kind cluster
#docker tag "${CONTROLLER_IMAGE}" "${PACKAGE_CONTROLLER_IMAGE}"
#docker tag "${BUILD_IMAGE}" "${PACKAGE_PROVIDER_IMAGE}"
sleep 5
"${KIND}" load docker-image "${PACKAGE_CONTROLLER_IMAGE}" --name="${K8S_CLUSTER}"
"${KIND}" load docker-image "${PACKAGE_PROVIDER_IMAGE}" --name="${K8S_CLUSTER}"

echo_step "create crossplane-system namespace"
"${KUBECTL}" create ns crossplane-system

echo_step "create persistent volume and claim for mounting package-cache"
PV_YAML="$(
  cat <<EOF
apiVersion: v1
kind: PersistentVolume
metadata:
  name: package-cache
  labels:
    type: local
spec:
  storageClassName: manual
  capacity:
    storage: 5Mi
  accessModes:
    - ReadWriteOnce
  hostPath:
    path: "/cache"
EOF
)"
echo "${PV_YAML}" | "${KUBECTL}" create -f -

PVC_YAML="$(
  cat <<EOF
apiVersion: v1
kind: PersistentVolumeClaim
metadata:
  name: package-cache
  namespace: crossplane-system
spec:
  accessModes:
    - ReadWriteOnce
  volumeName: package-cache
  storageClassName: manual
  resources:
    requests:
      storage: 1Mi
EOF
)"
echo "${PVC_YAML}" | "${KUBECTL}" create -f -

# install crossplane from stable channel
echo_step "installing crossplane from stable channel"
"${HELM3}" version
"${HELM3}" repo add crossplane-stable https://charts.crossplane.io/stable --force-update
# TODO: this is a hotfix until the latest stable version is supported
chart_version="$("${HELM3}" search repo crossplane-stable/crossplane --devel | awk 'FNR == 2 {print $2}')"
echo_info "using crossplane version ${chart_version}"
echo
# we replace empty dir with our PVC so that the /cache dir in the kind node
# container is exposed to the crossplane pod
"${HELM3}" install crossplane --namespace crossplane-system crossplane-stable/crossplane --version ${chart_version} --wait --set packageCache.pvc=package-cache

# ----------- integration tests
echo_step "--- INTEGRATION TESTS ---"

# install package
echo_step "--- install Crossplane Provider IONOSCLOUD ---"
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

if [ "$TEST_DATAPLATFORM" = true ]; then
  echo_step "--- DATAPLATFORM TESTS ---"
  echo_step "--- DATAPLATFORM cluster tests ---"
  dataplatform_tests
  echo_step "--- CLEANING UP DATAPLATFORM TESTS ---"
  echo_step "--- DATAPLATFORM cluster tests ---"
  dataplatform_tests_cleanup
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
