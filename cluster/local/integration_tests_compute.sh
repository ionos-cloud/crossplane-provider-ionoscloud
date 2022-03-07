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

  echo_step "get datacenter CR"
  kubectl get datacenters

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

## Volume CR Tests
function volume_tests() {
  echo_step "deploy a volume CR"
  INSTALL_RESOURCE_YAML="$(
    cat <<EOF
apiVersion: compute.ionoscloud.crossplane.io/v1alpha1
kind: Volume
metadata:
  name: example
spec:
  forProvider:
    name: exampletest
    size: 30
    type: HDD
    bus: VIRTIO
    licenceType: LINUX
    datacenterConfig:
      datacenterIdRef:
        name: example
  providerConfigRef:
    name: example
EOF
  )"

  echo "${INSTALL_RESOURCE_YAML}" | "${KUBECTL}" apply -f -

  echo_step "waiting for volume CR to be ready"
  kubectl wait --for=condition=ready volumes/example

  echo_step "get volume CR"
  kubectl get volumes

  echo_step "update volume CR"
  INSTALL_RESOURCE_YAML="$(
    cat <<EOF
apiVersion: compute.ionoscloud.crossplane.io/v1alpha1
kind: Volume
metadata:
  name: example
spec:
  forProvider:
    name: exampleVolume
    size: 30
    type: HDD
    bus: VIRTIO
    licenceType: LINUX
    datacenterConfig:
      datacenterIdRef:
        name: example
  providerConfigRef:
    name: example
EOF
  )"

  echo "${INSTALL_RESOURCE_YAML}" | "${KUBECTL}" apply -f -

  echo_step "waiting for volume CR to be ready"
  kubectl wait --for=condition=ready volumes/example
}

function volume_tests_cleanup() {
  INSTALL_RESOURCE_YAML="$(
    cat <<EOF
apiVersion: compute.ionoscloud.crossplane.io/v1alpha1
kind: Volume
metadata:
  name: example
spec:
  forProvider:
    name: exampleVolume
    size: 30
    type: HDD
    bus: VIRTIO
    licenceType: LINUX
    datacenterConfig:
      datacenterIdRef:
        name: example
  providerConfigRef:
    name: example
EOF
  )"

  echo_step "uninstalling volume CR"
  echo "${INSTALL_RESOURCE_YAML}" | "${KUBECTL}" delete -f -

  echo_step "wait for deletion volume CR"
  kubectl wait --for=delete volumes/example
}

## Server CR Tests
function server_tests() {
  echo_step "deploy a server CR and attach volume"
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
    availabilityZone: AUTO
    cpuFamily: INTEL_SKYLAKE
    datacenterConfig:
      datacenterIdRef:
        name: example
    volumeConfig:
      volumeIdRef:
        name: example
  providerConfigRef:
    name: example
EOF
  )"

  echo "${INSTALL_RESOURCE_YAML}" | "${KUBECTL}" apply -f -

  echo_step "waiting for server CR to be ready"
  kubectl wait --for=condition=ready servers/example

  echo_step "get server CR"
  kubectl get servers

  echo_step "update server CR and detach volume"
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
    availabilityZone: AUTO
    cpuFamily: INTEL_SKYLAKE
    datacenterConfig:
      datacenterIdRef:
        name: example
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
    availabilityZone: AUTO
    cpuFamily: INTEL_SKYLAKE
    datacenterConfig:
      datacenterIdRef:
        name: example
  providerConfigRef:
    name: example
EOF
  )"

  echo_step "uninstalling server CR"
  echo "${INSTALL_RESOURCE_YAML}" | "${KUBECTL}" delete -f -

  echo_step "wait for deletion server CR"
  kubectl wait --for=delete servers/example
}

## Lan CR Tests
function lan_tests() {
  echo_step "deploy a lan CR"
  INSTALL_RESOURCE_YAML="$(
    cat <<EOF
apiVersion: compute.ionoscloud.crossplane.io/v1alpha1
kind: Lan
metadata:
  name: example
spec:
  forProvider:
    name: exampletest
    public: false
    datacenterConfig:
      datacenterIdRef:
        name: example
  providerConfigRef:
    name: example
EOF
  )"

  echo "${INSTALL_RESOURCE_YAML}" | "${KUBECTL}" apply -f -

  echo_step "waiting for lan CR to be ready"
  kubectl wait --for=condition=ready lans/example

  echo_step "get lan CR"
  kubectl get lans

  echo_step "update lan CR"
  INSTALL_RESOURCE_YAML="$(
    cat <<EOF
apiVersion: compute.ionoscloud.crossplane.io/v1alpha1
kind: Lan
metadata:
  name: example
spec:
  forProvider:
    name: exampletestLan
    public: true
    datacenterConfig:
      datacenterIdRef:
        name: example
  providerConfigRef:
    name: example
EOF
  )"

  echo "${INSTALL_RESOURCE_YAML}" | "${KUBECTL}" apply -f -

  echo_step "waiting for lan CR to be ready"
  kubectl wait --for=condition=ready lans/example
}

function lan_tests_cleanup() {
  INSTALL_RESOURCE_YAML="$(
    cat <<EOF
apiVersion: compute.ionoscloud.crossplane.io/v1alpha1
kind: Lan
metadata:
  name: example
spec:
  forProvider:
    name: exampletestLan
    public: true
    datacenterConfig:
      datacenterIdRef:
        name: example
  providerConfigRef:
    name: example
EOF
  )"

  echo_step "uninstalling lan CR"
  echo "${INSTALL_RESOURCE_YAML}" | "${KUBECTL}" delete -f -

  echo_step "wait for deletion lan CR"
  kubectl wait --for=delete lans/example
}

## Nic CR Tests
function nic_tests() {
  echo_step "deploy a nic CR"
  INSTALL_RESOURCE_YAML="$(
    cat <<EOF
apiVersion: compute.ionoscloud.crossplane.io/v1alpha1
kind: Nic
metadata:
  name: example
spec:
  forProvider:
    name: exampleNic
    dhcp: false
    datacenterConfig:
      datacenterIdRef:
        name: example
    serverConfig:
      serverIdRef:
        name: example
    lanConfig:
      lanIdRef:
        name: example
  providerConfigRef:
    name: example
EOF
  )"

  echo "${INSTALL_RESOURCE_YAML}" | "${KUBECTL}" apply -f -

  echo_step "waiting for nic CR to be ready"
  kubectl wait --for=condition=ready nics/example --timeout 120s

  echo_step "get nic CR"
  kubectl get nics

  echo_step "update nic CR"
  INSTALL_RESOURCE_YAML="$(
    cat <<EOF
apiVersion: compute.ionoscloud.crossplane.io/v1alpha1
kind: Nic
metadata:
  name: example
spec:
  forProvider:
    name: exampleNic
    dhcp: true
    firewallActive: true
    datacenterConfig:
      datacenterIdRef:
        name: example
    serverConfig:
      serverIdRef:
        name: example
    lanConfig:
      lanIdRef:
        name: example
  providerConfigRef:
    name: example
EOF
  )"

  echo "${INSTALL_RESOURCE_YAML}" | "${KUBECTL}" apply -f -

  echo_step "waiting for nic CR to be ready"
  kubectl wait --for=condition=ready nics/example --timeout 120s
}

function nic_tests_cleanup() {
  INSTALL_RESOURCE_YAML="$(
    cat <<EOF
apiVersion: compute.ionoscloud.crossplane.io/v1alpha1
kind: Nic
metadata:
  name: example
spec:
  forProvider:
    name: exampleNic
    dhcp: false
    datacenterConfig:
      datacenterIdRef:
        name: example
    serverConfig:
      serverIdRef:
        name: example
    lanConfig:
      lanIdRef:
        name: example
  providerConfigRef:
    name: example
EOF
  )"

  echo_step "uninstalling nic CR"
  echo "${INSTALL_RESOURCE_YAML}" | "${KUBECTL}" delete -f -

  echo_step "wait for deletion nic CR"
  kubectl wait --for=delete nics/example
}
