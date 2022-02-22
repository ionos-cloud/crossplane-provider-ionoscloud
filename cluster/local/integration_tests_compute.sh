#!/usr/bin/env bash

set -e

## The purpose of this script is to have the tests for
## the compute resources

## Datacenter CR Tests
function datacenter_tests() {
  echo_step "deploy a datacenter CR"
  INSTALL_RESOURCE_YAML="$(
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

  echo "${INSTALL_RESOURCE_YAML}" | "${KUBECTL}" apply -f -

  echo_step "waiting for datacenter CR to be ready"
  kubectl wait --for=condition=ready datacenters/example

  echo_step "get datacenters and describe datacenter CR"
  kubectl get datacenters
  kubectl describe datacenters example

  echo_step "update datacenter CR"
  INSTALL_RESOURCE_YAML="$(
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

  echo "${INSTALL_RESOURCE_YAML}" | "${KUBECTL}" apply -f -

  echo_step "waiting for datacenter CR to be ready"
  kubectl wait --for=condition=ready datacenters/example
}

function datacenter_tests_cleanup() {
  INSTALL_RESOURCE_YAML="$(
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

  echo_step "uninstalling datacenter CR"
  echo "${INSTALL_RESOURCE_YAML}" | "${KUBECTL}" delete -f -

  echo_step "wait for deletion datacenter CR"
  kubectl wait --for=delete datacenters/example
}

## Server CR Tests
function server_tests() {
  echo_step "deploy a server CR"
  INSTALL_RESOURCE_YAML="$(
    cat <<EOF
apiVersion: compute.ionoscloud.crossplane.io/v1alpha1
kind: Server
metadata:
  name: example
spec:
  forProvider:
    name: exampletest
    cores: 4
    ram: 2048
    datacenterIDRef:
      name: example
    availabilityZone: AUTO
    cpuFamily: INTEL_SKYLAKE
  providerConfigRef:
    name: example
EOF
  )"

  echo "${INSTALL_RESOURCE_YAML}" | "${KUBECTL}" apply -f -

  echo_step "waiting for server CR to be ready"
  kubectl wait --for=condition=ready servers/example

  echo_step "get server and describe server CR"
  kubectl get servers
  kubectl describe servers example

  echo_step "update server CR"
  INSTALL_RESOURCE_YAML="$(
    cat <<EOF
apiVersion: compute.ionoscloud.crossplane.io/v1alpha1
kind: Server
metadata:
  name: example
spec:
  forProvider:
    name: exampleServer
    cores: 4
    ram: 2048
    datacenterIDRef:
      name: example
    availabilityZone: AUTO
    cpuFamily: INTEL_SKYLAKE
  providerConfigRef:
    name: example
EOF
  )"

  echo "${INSTALL_RESOURCE_YAML}" | "${KUBECTL}" apply -f -

  echo_step "waiting for server CR to be ready"
  kubectl wait --for=condition=ready servers/example
}

function server_tests_cleanup() {
  INSTALL_RESOURCE_YAML="$(
    cat <<EOF
apiVersion: compute.ionoscloud.crossplane.io/v1alpha1
kind: Server
metadata:
  name: example
spec:
  forProvider:
    name: exampleServer
    cores: 4
    ram: 2048
    datacenterIDRef:
      name: example
    availabilityZone: AUTO
    cpuFamily: INTEL_SKYLAKE
  providerConfigRef:
    name: example
EOF
  )"

  echo_step "uninstalling server CR"
  echo "${INSTALL_RESOURCE_YAML}" | "${KUBECTL}" delete -f -

  echo_step "wait for deletion server CR"
  kubectl wait --for=delete servers/example
}
