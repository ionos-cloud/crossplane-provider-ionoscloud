#!/usr/bin/env bash

set -e

## The purpose of this script is to have the tests for
## the Dataplatform resources.
## Please name the functions the following format:
## <resource_name>_tests() and <resource_name>_tests_cleanup().

## dataplatform CR Tests
function dataplatform_tests() {
  echo_step "deploy a dataplatform CR"
  INSTALL_RESOURCE_YAML="$(
    cat <<EOF
apiVersion: compute.ionoscloud.crossplane.io/v1alpha1
kind: Datacenter
metadata:
  name: exampledataplatform
managementPolicies:
  - "*"
spec:
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
managementPolicies:
  - "*"
spec:
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

  echo_step "waiting for dataplatform CR to be ready & synced"
  kubectl wait --for=condition=ready dataplatformclusters/example --timeout=60m
  kubectl wait --for=condition=synced dataplatformclusters/example --timeout=60m

  echo_step "get dataplatform CR"
  kubectl get dataplatformclusters
}

function dataplatform_tests_cleanup() {
  INSTALL_RESOURCE_YAML="$(
    cat <<EOF
apiVersion: dataplatform.ionoscloud.crossplane.io/v1alpha1
kind: DataplatformCluster
metadata:
  name: example
managementPolicies:
  - "*"
spec:
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
