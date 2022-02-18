#!/usr/bin/env bash

set -e

## The purpose of this script is to have the tests for
## the compute resources

## Datacenter CR Tests
function datacenter_tests() {
  echo_step "deploy a datacenter CR"
  INSTALL_DC_YAML="$(
    cat <<EOF
apiVersion: compute.ionoscloud.crossplane.io/v1alpha1
kind: Datacenter
metadata:
  name: example
spec:
  forProvider:
    name: testdatacenter
    location: de/txl
  providerConfigRef:
    name: example
EOF
  )"

  echo "${INSTALL_DC_YAML}" | "${KUBECTL}" apply -f -

  echo_step "waiting for datacenter CR to be ready"
  kubectl wait --for=condition=ready datacenters/example

  echo_step "get datacenters and describe datacenter CR"
  kubectl get datacenters
  kubectl describe datacenters example

  echo_step "update datacenter CR"
  INSTALL_DC_YAML="$(
    cat <<EOF
apiVersion: compute.ionoscloud.crossplane.io/v1alpha1
kind: Datacenter
metadata:
  name: example
spec:
  forProvider:
    name: Test Datacenter CR
    location: de/txl
    description: e2e crossplane testing
  providerConfigRef:
    name: example
EOF
  )"

  echo "${INSTALL_DC_YAML}" | "${KUBECTL}" apply -f -

  echo_step "waiting for datacenter CR to be ready"
  kubectl wait --for=condition=ready datacenters/example

  echo_step "uninstalling datacenter CR"
  echo "${INSTALL_DC_YAML}" | "${KUBECTL}" delete -f -

  echo_step "wait for deletion datacenter CR"
  kubectl wait --for=delete datacenters/example
}
