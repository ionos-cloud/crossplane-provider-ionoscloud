#!/usr/bin/env bash

set -e

## The purpose of this script is to have the tests for
## the ApplicationLoadBalancer resources
## Please name the functions the following format:
## <resource_name>_tests() and <resource_name>_tests_cleanup().

## ApplicationLoadBalancer CR Tests
function alb_tests() {
  echo_step "deploy a application load balancer CR"
  INSTALL_RESOURCE_YAML="$(
    cat <<EOF
apiVersion: compute.ionoscloud.crossplane.io/v1alpha1
kind: IPBlock
metadata:
  name: example
spec:
  forProvider:
    name: exampleIpBlock
    size: 2
    location: de/txl
  providerConfigRef:
    name: example
EOF
  )"

  echo "${INSTALL_RESOURCE_YAML}" | "${KUBECTL}" apply -f -

  echo_step "waiting for application load balancer CR to be ready & synced"
  kubectl wait --for=condition=ready ipblocks/example
  kubectl wait --for=condition=synced ipblocks/example

  echo_step "get application load balancer CR"
  kubectl get ipblocks -o wide

  echo_step "update application load balancer CR"
  INSTALL_RESOURCE_YAML="$(
    cat <<EOF
apiVersion: compute.ionoscloud.crossplane.io/v1alpha1
kind: IPBlock
metadata:
  name: example
spec:
  forProvider:
    name: exampleIpBlockUpdate
    size: 2
    location: de/txl
  providerConfigRef:
    name: example
EOF
  )"

  echo "${INSTALL_RESOURCE_YAML}" | "${KUBECTL}" apply -f -

  echo_step "waiting for application load balancer CR to be ready & synced"
  kubectl wait --for=condition=ready ipblocks/example
  kubectl wait --for=condition=synced ipblocks/example

  echo_step "get updated application load balancer CR"
  kubectl get ipblocks -o wide
}

function alb_tests_cleanup() {
  INSTALL_RESOURCE_YAML="$(
    cat <<EOF
apiVersion: compute.ionoscloud.crossplane.io/v1alpha1
kind: IPBlock
metadata:
  name: example
spec:
  forProvider:
    name: exampleIpBlockUpdate
    size: 2
    location: de/txl
  providerConfigRef:
    name: example
EOF
  )"

  echo_step "uninstalling application load balancer CR"
  echo "${INSTALL_RESOURCE_YAML}" | "${KUBECTL}" delete -f -

  echo_step "wait for deletion application load balancer CR"
  kubectl wait --for=delete ipblocks/example
}

## ForwardingRule CR Tests
function forwardingrule_tests() {
  echo_step "deploy a forwarding rule CR"
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

  echo_step "waiting for forwarding rule CR to be ready & synced"
  kubectl wait --for=condition=ready datacenters/example
  kubectl wait --for=condition=synced datacenters/example

  echo_step "get forwarding rule CR"
  kubectl get datacenters

  echo_step "update forwarding rule CR"
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

  echo_step "waiting for forwarding rule CR to be ready & synced"
  kubectl wait --for=condition=ready datacenters/example
  kubectl wait --for=condition=synced datacenters/example
}

function forwardingrule_tests_cleanup() {
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

  echo_step "uninstalling forwarding rule CR"
  echo "${INSTALL_RESOURCE_YAML}" | "${KUBECTL}" delete -f -

  echo_step "wait for deletion forwarding rule CR"
  kubectl wait --for=delete datacenters/example
}

## TargetGroup CR Tests
function targetgroup_tests() {
  echo_step "deploy a target group CR"
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

  echo_step "waiting for target group CR to be ready & synced"
  kubectl wait --for=condition=ready volumes/example
  kubectl wait --for=condition=synced volumes/example

  echo_step "get target group CR"
  kubectl get volumes

  echo_step "update target group CR"
  INSTALL_RESOURCE_YAML="$(
    cat <<EOF
apiVersion: compute.ionoscloud.crossplane.io/v1alpha1
kind: Volume
metadata:
  name: example
spec:
  forProvider:
    name: exampleVolume
    size: 40
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

  echo_step "waiting for target group CR to be ready & synced"
  kubectl wait --for=condition=ready volumes/example
  kubectl wait --for=condition=synced volumes/example
}

function targetgroup_tests_cleanup() {
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

  echo_step "uninstalling target group CR"
  echo "${INSTALL_RESOURCE_YAML}" | "${KUBECTL}" delete -f -

  echo_step "wait for deletion target group CR"
  kubectl wait --for=delete volumes/example
}
