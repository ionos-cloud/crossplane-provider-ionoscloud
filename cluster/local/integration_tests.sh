#!/usr/bin/env bash
set -e

# add prints functions
source ./cluster/local/print.sh
# add integration tests for resources
source ./cluster/local/integration_tests_provider.sh
source ./cluster/local/integration_tests_compute.sh
source ./cluster/local/integration_tests_alb.sh
source ./cluster/local/integration_tests_dbaas_postgres.sh
source ./cluster/local/integration_tests_k8s.sh

# ------------------------------
projectdir="$(cd "$(dirname "${BASH_SOURCE[0]}")"/../.. && pwd)"

# get the build environment variables from the special build.vars target in the main makefile
eval $(make --no-print-directory -C ${projectdir} build.vars)

# ------------------------------

REGISTRY=${REGISTRY:-ghcr.io}
ORG_NAME=${ORG_NAME:-ionos-cloud}
BUILD_IMAGE="${REGISTRY}/${ORG_NAME}/${PROJECT_NAME}"
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
TEST_K8S=${TEST_K8S:-false}
TEST_ALB=${TEST_ALB:-false}

version_tag="$(cat ${projectdir}/_output/version)"
# tag as latest version to load into kind cluster
PACKAGE_CONTROLLER_IMAGE="${REGISTRY}/${ORG_NAME}/${PROJECT_NAME}-controller:${VERSION}"
PACKAGE_PROVIDER_IMAGE="${REGISTRY}/${ORG_NAME}/${PROJECT_NAME}:${VERSION}"
K8S_CLUSTER="${K8S_CLUSTER:-${BUILD_REGISTRY}-inttests}"
KIND_NODE_IMAGE_TAG="${KIND_NODE_IMAGE_TAG:-v1.21.1}"

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
docker save "${BUILD_IMAGE}" -o "${CACHE_PATH}/${PACKAGE_NAME}.xpkg" && chmod 644 "${CACHE_PATH}/${PACKAGE_NAME}.xpkg"

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

# files are not synced properly from host to kind node container on Jenkins, so
# we must manually copy image from host to node
echo_step "pre-cache package by copying to kind node"
docker cp "${CACHE_PATH}/${PACKAGE_NAME}.xpkg" "${K8S_CLUSTER}-control-plane":"/cache/${PACKAGE_NAME}.xpkg"

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
#chart_version="$("${HELM3}" search repo crossplane-stable/crossplane --devel | awk 'FNR == 2 {print $2}')"
chart_version="1.6.4"
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
fi

if [ "$TEST_DBAAS" = true ]; then
  echo_step "--- dbaas postgres cluster tests ---"
  dbaas_postgres_cluster_tests
fi

if [ "$TEST_K8S" = true ]; then
  echo_step "--- k8s cluster tests ---"
  k8s_cluster_tests
  echo_step "--- k8s nodepool tests ---"
  k8s_nodepool_tests
fi

if [ "$TEST_ALB" = true ]; then
  echo_step "--- target group tests ---"
  targetgroup_tests
  echo_step "--- application load balancer tests ---"
  alb_tests
  echo_step "--- forwarding rule tests ---"
  forwardingrule_tests
fi

echo_step "-------------------"
echo_step "--- CLEANING UP ---"
echo_step "-------------------"

if [ "$TEST_COMPUTE" = true ]; then
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
fi

if [ "$TEST_DBAAS" = true ]; then
  echo_step "--- dbaas postgres cluster tests ---"
  dbaas_postgres_cluster_tests_cleanup
fi

if [ "$TEST_K8S" = true ]; then
  echo_step "--- k8s nodepool tests ---"
  k8s_nodepool_tests_cleanup
  echo_step "--- k8s cluster tests ---"
  k8s_cluster_tests_cleanup
fi

if [ "$TEST_ALB" = true ]; then
  echo_step "--- forwarding rule tests ---"
  forwardingrule_tests_cleanup
  echo_step "--- application load balancer tests ---"
  alb_tests_cleanup
  echo_step "--- target group tests ---"
  targetgroup_tests_cleanup
fi

# uninstalling Crossplane Provider IONOS Cloud
echo_step "--- uninstalling ${PROJECT_NAME} ---"
uninstall_provider

echo_success "Integration tests succeeded!"
