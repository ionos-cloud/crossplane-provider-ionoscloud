#!/usr/bin/env bash

set -e

## The purpose of this script is to have the tests for
## the Dataplatform resources.
## Please name the functions the following format:
## <resource_name>_tests() and <resource_name>_tests_cleanup().

## dataplatform CR Tests
function dataplatform_tests() {
  echo_step "deploy a dataplatform cluster CR"
  INSTALL_RESOURCE_YAML="$(
    cat <<EOF
apiVersion: compute.ionoscloud.crossplane.io/v1alpha1
kind: Datacenter
metadata:
  name: exampledataplatform
spec:
  managementPolicies:
    - "*"
  forProvider:
    name: exampleDatacenter
    location: de/txl
    description: test
  providerConfigRef:
    name: example
---
apiVersion: dataplatform.ionoscloud.crossplane.io/v1alpha1
kind: DataplatformCluster
metadata:
  name: example
spec:
  managementPolicies:
    - "*"
  forProvider:
    datacenterConfig:
      datacenterIdRef:
        name: exampledataplatform
    name: exampleCluster
    version: "23.11"
  providerConfigRef:
    name: example
EOF
  )"

  echo "${INSTALL_RESOURCE_YAML}" | "${KUBECTL}" apply -f -

  echo_step "waiting for dataplatform cluster CR to be ready & synced"
  kubectl wait --for=condition=ready dataplatformclusters/example --timeout=60m
  kubectl wait --for=condition=synced dataplatformclusters/example --timeout=60m

  echo_step "get dataplatform CR"
  kubectl get dataplatformclusters

  echo_step "deploy a dataplatform nodepool CR"
  INSTALL_RESOURCE_YAML="$(
    cat <<EOF
apiVersion: dataplatform.ionoscloud.crossplane.io/v1alpha1
kind: DataplatformNodepool
metadata:
  name: example
spec:
  managementPolicies:
    - "*"
  forProvider:
    name: exampleDataplatformNodepool
    nodeCount: 2
    datacenterConfig:
      datacenterIdRef:
        name: exampledataplatform
    clusterConfig:
      ClusterIdRef:
        name: example
    storageType: HDD
  providerConfigRef:
    name: example
EOF
  )"

  echo "${INSTALL_RESOURCE_YAML}" | "${KUBECTL}" apply -f -

  echo_step "waiting for dataplatform nodepool CR to be ready & synced"
  kubectl wait --for=condition=ready dataplatformnodepools/example --timeout=60m
  kubectl wait --for=condition=synced dataplatformnodepools/example --timeout=60m

  echo_step "get dataplatform nodepool CR"
  kubectl get dataplatformnodepools
}

function dataplatform_tests_cleanup() {
  INSTALL_RESOURCE_YAML="$(
    cat <<EOF
apiVersion: dataplatform.ionoscloud.crossplane.io/v1alpha1
kind: DataplatformNodepool
metadata:
  name: example
spec:
  managementPolicies:
    - "*"
  forProvider:
    name: exampleDataplatformNodepool
    nodeCount: 2
    datacenterConfig:
      datacenterIdRef:
        name: exampledataplatform
    clusterConfig:
      ClusterIdRef:
        name: example
    storageType: HDD
  providerConfigRef:
    name: example
EOF
  )"

  echo_step "uninstalling dataplatform nodepool CR"
  echo "${INSTALL_RESOURCE_YAML}" | "${KUBECTL}" delete -f -

  echo_step "wait for deletion dataplatform nodepool CR"
  kubectl wait --for=delete dataplatformnodepools/example --timeout=30m

  INSTALL_RESOURCE_YAML="$(
    cat <<EOF
apiVersion: dataplatform.ionoscloud.crossplane.io/v1alpha1
kind: DataplatformCluster
metadata:
  name: example
spec:
  managementPolicies:
    - "*"
  forProvider:
    datacenterConfig:
      datacenterIdRef:
        name: exampledataplatform
    name: exampleCluster
    version: "23.11"
  providerConfigRef:
    name: example
EOF
  )"

  echo_step "uninstalling dataplatform CR"
  echo "${INSTALL_RESOURCE_YAML}" | "${KUBECTL}" delete -f -

  echo_step "wait for deletion dataplatform CR"
  kubectl wait --for=delete dataplatformclusters/example --timeout=30m

}
