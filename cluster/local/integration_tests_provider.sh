#!/usr/bin/env bash

set -e

function install_provider() {
  echo_step "checking provider installation"
  echo_step "checking provider"
  kubectl get provider
  sleep 5

  echo_step "checking providerrevision"
  kubectl get providerrevision
  sleep 10

  echo_step "checking deployments"
  kubectl get deployments -n crossplane-system
  sleep 5

  echo_step "waiting for provider.pkg.crossplane.io/${PROJECT_NAME} to be installed"
  kubectl wait "provider.pkg.crossplane.io/${PROJECT_NAME}" --for=condition=healthy --timeout=120s

  echo_step "waiting for all pods in ${CROSSPLANE_NAMESPACE} namespace to be ready"
  kubectl wait --for=condition=ready pods --all -n ${CROSSPLANE_NAMESPACE}
  kubectl get pods -n crossplane-system

  echo_step "add secret credentials"
#  BASE64_PW=$(echo -n "${IONOS_PASSWORD}" | base64)
#  kubectl create secret generic --namespace ${CROSSPLANE_NAMESPACE} example-provider-secret --from-literal=credentials="{\"user\":\"${IONOS_USERNAME}\",\"password\":\"${BASE64_PW}\"}"
  # Use Token
  kubectl create secret generic --namespace ${CROSSPLANE_NAMESPACE} example-provider-secret --from-literal=credentials="{\"token\":\"${IONOS_TOKEN}\"}"
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
}

function uninstall_provider() {
  echo_step "uninstalling ${PROJECT_NAME} from \"${CROSSPLANE_NAMESPACE}\" namespace"
  # after deleting the ProviderConfig, it is safe to
  # also delete the Provider IONOS Cloud
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
  echo "${INSTALL_CRED_YAML}" | "${KUBECTL}" delete -f -
}
