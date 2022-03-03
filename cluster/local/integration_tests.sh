#!/usr/bin/env bash
set -e

# add prints functions
source ./cluster/local/print.sh
# add integration tests for resources
source ./cluster/local/integration_tests_compute.sh

# ------------------------------
projectdir="$(cd "$(dirname "${BASH_SOURCE[0]}")"/../.. && pwd)"

# get the build environment variables from the special build.vars target in the main makefile
eval $(make --no-print-directory -C ${projectdir} build.vars)

# ------------------------------

REGISTRY=${REGISTRY:-ghcr.io}
ORG_NAME=${ORG_NAME:-ionos-cloud}
BUILD_IMAGE="${REGISTRY}/${ORG_NAME}/${PROJECT_NAME}"
CONTROLLER_IMAGE="${REGISTRY}/${ORG_NAME}/${PROJECT_NAME}-controller"

version_tag="$(cat ${projectdir}/_output/version)"
# tag as latest version to load into kind cluster
PACKAGE_CONTROLLER_IMAGE="${REGISTRY}/${ORG_NAME}/${PROJECT_NAME}-controller:${VERSION}"
PACKAGE_PROVIDER_IMAGE="${REGISTRY}/${ORG_NAME}/${PROJECT_NAME}:${VERSION}"
K8S_CLUSTER="${K8S_CLUSTER:-${BUILD_REGISTRY}-inttests}"
KIND_NODE_IMAGE_TAG="${KIND_NODE_IMAGE_TAG:-v1.21.1}"

CROSSPLANE_NAMESPACE="crossplane-system"
PACKAGE_NAME="provider-ionoscloud"
LOGS_FILE="${PACKAGE_NAME}.txt"

# cleanup on exit
if [ "$skipcleanup" != true ]; then
    function cleanup() {
        echo_step "Cleaning up..."
        echo_step "Checking is ${PACKAGE_NAME} pod exists..."
        POD_INFO_PROVIDER=$(kubectl get pods -n ${CROSSPLANE_NAMESPACE} | grep ${PACKAGE_NAME})
        if [ ! -z "${POD_INFO_PROVIDER}" ]; then
            POD_PROVIDER=($POD_INFO_PROVIDER)
            echo_step "Saving logs to ${LOGS_FILE} file..."
            echo "--- logs of the ${POD_PROVIDER} pod---" >>${LOGS_FILE}
            kubectl logs pod/${POD_PROVIDER} -n ${CROSSPLANE_NAMESPACE} >>${LOGS_FILE}
        fi
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
"${HELM3}" repo add crossplane-stable https://charts.crossplane.io/stable
"${HELM3}" repo update
chart_version="$("${HELM3}" search repo crossplane-stable/crossplane --devel | awk 'FNR == 2 {print $2}')"
echo_info "using crossplane version ${chart_version}"
echo
# we replace empty dir with our PVC so that the /cache dir in the kind node
# container is exposed to the crossplane pod
"${HELM3}" install crossplane --namespace crossplane-system crossplane-stable/crossplane --version ${chart_version} --devel --wait --set packageCache.pvc=package-cache

# ----------- integration tests
echo_step "--- INTEGRATION TESTS ---"

# install package
echo_step "installing ${PROJECT_NAME} into \"${CROSSPLANE_NAMESPACE}\" namespace"

INSTALL_YAML="$(
    cat <<EOF
apiVersion: pkg.crossplane.io/v1
kind: Provider
metadata:
  name: "${PACKAGE_NAME}"
spec:
  package: "${PACKAGE_NAME}"
  packagePullPolicy: Never
EOF
)"

echo "${INSTALL_YAML}" | "${KUBECTL}" apply -f -

# printing the cache dir contents can be useful for troubleshooting failures
echo_step "check kind node cache dir contents"
docker exec "${K8S_CLUSTER}-control-plane" ls -la /cache

echo_step "checking provider installation"

echo_step "checking provider"
kubectl get provider
echo "--- describe provider ${PACKAGE_NAME} ---" >>${LOGS_FILE}
kubectl describe provider ${PACKAGE_NAME} >>${LOGS_FILE}
sleep 5

echo_step "checking providerrevision"
kubectl get providerrevision
echo "--- describe providerrevision ${PACKAGE_NAME} ---" >>${LOGS_FILE}
kubectl describe providerrevision ${PACKAGE_NAME} >>${LOGS_FILE}
sleep 5

echo_step "checking deployments"
kubectl get deployments -n crossplane-system
echo "--- describe deployments ${PACKAGE_NAME} ---" >>${LOGS_FILE}
kubectl describe deployments provider-ionoscloud-provider-ion -n ${CROSSPLANE_NAMESPACE} >>${LOGS_FILE}
sleep 5

echo_step "waiting for provider to be installed"
kubectl wait "provider.pkg.crossplane.io/${PACKAGE_NAME}" --for=condition=healthy --timeout=60s

echo_step "waiting for all pods in ${CROSSPLANE_NAMESPACE} namespace to be ready"
kubectl wait --for=condition=ready pods --all -n ${CROSSPLANE_NAMESPACE}
kubectl get pods -n crossplane-system

echo_step "add secret credentials"
BASE64_PW=$(echo -n "${IONOS_PASSWORD}" | base64)
kubectl create secret generic --namespace ${CROSSPLANE_NAMESPACE} example-provider-secret --from-literal=credentials="{\"user\":\"${IONOS_USERNAME}\",\"password\":\"${BASE64_PW}\"}"
INSTALL_CRED_YAML="$(
    cat <<EOF
apiVersion: ionoscloud.crossplane.io/v1alpha1
kind: ProviderConfig
metadata:
  name: example
spec:
  credentials:
    source: Secret
    secretRef:
      namespace: crossplane-system
      name: example-provider-secret
      key: credentials
EOF
)"

echo "${INSTALL_CRED_YAML}" | "${KUBECTL}" apply -f -

# run Compute Resources Tests
echo_step "--- datacenter tests ---"
datacenter_tests
echo_step "--- lan tests ---"
lan_tests
echo_step "--- server tests ---"
server_tests
echo_step "--- volume tests ---"
volume_tests

# uninstalling Compute Resources
echo_step "cleanup lan tests"
lan_tests_cleanup
echo_step "cleanup server tests"
server_tests_cleanup
echo_step "cleanup volume tests"
volume_tests_cleanup
echo_step "cleanup datacenter tests"
datacenter_tests_cleanup

# uninstalling Crossplane Provider IONOS Cloud
echo_step "uninstalling ${PROJECT_NAME}"
echo "${INSTALL_YAML}" | "${KUBECTL}" delete -f -

# check pods deleted
timeout=60
current=0
step=3
while [[ $(kubectl get providerrevision.pkg.crossplane.io -o name | wc -l) != "0" ]]; do
    echo "waiting for provider to be deleted for another $step seconds"
    current=$current+$step
    if ! [[ $timeout > $current ]]; then
        echo_error "timeout of ${timeout}s has been reached"
    fi
    sleep $step
done

echo_success "Integration tests succeeded!"
