#!/usr/bin/env bash

set -e

## The purpose of this script is to have the tests for
## the k8s resources
## Please name the functions the following format:
## <resource_name>_tests() and <resource_name>_tests_cleanup().

## K8s Cluster CR Tests
function k8s_cluster_tests() {
  echo_step "deploy a k8s cluster CR"
  INSTALL_RESOURCE_YAML="$(
    cat <<EOF
apiVersion: k8s.ionoscloud.crossplane.io/v1alpha1
kind: Cluster
metadata:
  name: examplek8s
spec:
  forProvider:
    name: exampleK8sCluster
    public: true
    maintenanceWindow:
      dayOfTheWeek: Monday
      time: "23:40:58Z"
  providerConfigRef:
    name: example
EOF
  )"

  echo "${INSTALL_RESOURCE_YAML}" | "${KUBECTL}" apply -f -

  echo_step "waiting for k8s cluster CR to be ready & synced"
  kubectl wait --for=condition=ready clusters.k8s/examplek8s --timeout=15m
  kubectl wait --for=condition=synced clusters.k8s/examplek8s --timeout=10m

  echo_step "get k8s cluster CR"
  kubectl get clusters.k8s

  echo_step "update k8s cluster CR"
  INSTALL_RESOURCE_YAML="$(
    cat <<EOF
apiVersion: k8s.ionoscloud.crossplane.io/v1alpha1
kind: Cluster
metadata:
  name: examplek8s
spec:
  forProvider:
    name: exampleK8sClusterUpdate
    public: true
    maintenanceWindow:
      dayOfTheWeek: Friday
      time: "23:40:58Z"
  providerConfigRef:
    name: example
EOF
  )"

  echo "${INSTALL_RESOURCE_YAML}" | "${KUBECTL}" apply -f -

  echo_step "waiting for k8s cluster CR to be ready & synced"
  kubectl wait --for=condition=ready clusters.k8s/examplek8s --timeout=10m
  kubectl wait --for=condition=synced clusters.k8s/examplek8s --timeout=10m
}

function k8s_cluster_tests_cleanup() {
  INSTALL_RESOURCE_YAML="$(
    cat <<EOF
apiVersion: k8s.ionoscloud.crossplane.io/v1alpha1
kind: Cluster
metadata:
  name: examplek8s
spec:
  forProvider:
    name: exampleK8sCluster
    public: true
    maintenanceWindow:
      dayOfTheWeek: Friday
      time: "23:40:58Z"
  providerConfigRef:
    name: example
EOF
  )"

  echo_step "uninstalling k8s cluster CR"
  echo "${INSTALL_RESOURCE_YAML}" | "${KUBECTL}" delete -f -

  echo_step "wait for deletion k8s cluster CR"
  kubectl wait --for=delete clusters.k8s/examplek8s --timeout=10m
}
