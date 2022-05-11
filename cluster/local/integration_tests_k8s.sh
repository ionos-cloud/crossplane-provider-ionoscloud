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
    maintenanceWindow:
      dayOfTheWeek: Monday
      time: "23:40:58Z"
  writeConnectionSecretToRef:
    namespace: default
    name: kubeconfig
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
    maintenanceWindow:
      dayOfTheWeek: Friday
      time: "23:40:58Z"
  writeConnectionSecretToRef:
    namespace: default
    name: kubeconfig
  providerConfigRef:
    name: example
EOF
  )"

  echo "${INSTALL_RESOURCE_YAML}" | "${KUBECTL}" apply -f -

  echo_step "waiting for k8s cluster CR to be ready & synced"
  # sleep 10 seconds
  sleep 10
  kubectl wait --for=condition=ready clusters.k8s/examplek8s --timeout=10m
  kubectl wait --for=condition=synced clusters.k8s/examplek8s --timeout=10m
}

## K8s NodePool CR Tests
function k8s_nodepool_tests() {
  echo_step "deploy a k8s nodepool CR"
  INSTALL_RESOURCE_YAML="$(
    cat <<EOF
apiVersion: compute.ionoscloud.crossplane.io/v1alpha1
kind: Datacenter
metadata:
  name: exampledatacenterk8s
spec:
  forProvider:
    name: exampleDatacenterK8sNodepool
    location: us/las
    description: test
  providerConfigRef:
    name: example
---
apiVersion: compute.ionoscloud.crossplane.io/v1alpha1
kind: Lan
metadata:
  name: examplelank8s
spec:
  forProvider:
    name: exampleLanK8sNodepool
    public: false
    datacenterConfig:
      datacenterIdRef:
        name: exampledatacenterk8s
  providerConfigRef:
    name: example
---
apiVersion: k8s.ionoscloud.crossplane.io/v1alpha1
kind: NodePool
metadata:
  name: examplek8snodepool
spec:
  forProvider:
    name: exampleK8sNodepool
    nodeCount: 1
    cpuFamily: AMD_OPTERON
    coresCount: 1
    ramSize: 2048
    availabilityZone: AUTO
    storageType: HDD
    storageSize: 10
    lans:
      - lanConfig:
          lanIdRef:
            name: examplelank8s
        dhcp: true
    labels:
      testlabel: "valueLabelK8s"
    annotations:
      testannotation: "valueAnnotationK8s"
    datacenterConfig:
      datacenterIdRef:
        name: exampledatacenterk8s
    clusterConfig:
      clusterIdRef:
        name: examplek8s
  providerConfigRef:
    name: example
EOF
  )"

  echo "${INSTALL_RESOURCE_YAML}" | "${KUBECTL}" apply -f -

  echo_step "waiting for k8s nodepool CR to be ready & synced"
  kubectl wait --for=condition=ready nodepools.k8s/examplek8snodepool --timeout=15m
  kubectl wait --for=condition=synced nodepools.k8s/examplek8snodepool --timeout=10m

  echo_step "get k8s nodepool CR"
  kubectl get nodepools.k8s
}

function k8s_nodepool_tests_cleanup() {
  INSTALL_RESOURCE_YAML="$(
    cat <<EOF
apiVersion: compute.ionoscloud.crossplane.io/v1alpha1
kind: Datacenter
metadata:
  name: exampledatacenterk8s
spec:
  forProvider:
    name: exampleDatacenterK8sNodepool
    location: us/las
    description: test
  providerConfigRef:
    name: example
---
apiVersion: compute.ionoscloud.crossplane.io/v1alpha1
kind: Lan
metadata:
  name: examplelank8s
spec:
  forProvider:
    name: exampleLanK8sNodepool
    public: false
    datacenterConfig:
      datacenterIdRef:
        name: exampledatacenterk8s
  providerConfigRef:
    name: example
---
apiVersion: k8s.ionoscloud.crossplane.io/v1alpha1
kind: NodePool
metadata:
  name: examplek8snodepool
spec:
  forProvider:
    name: exampleK8sNodepool
    nodeCount: 1
    cpuFamily: AMD_OPTERON
    coresCount: 1
    ramSize: 2048
    availabilityZone: AUTO
    storageType: HDD
    storageSize: 10
    lans:
      - lanConfig:
          lanIdRef:
            name: examplelank8s
        dhcp: true
    labels:
      testlabel: "valueLabelK8s"
    annotations:
      testannotation: "valueAnnotationK8s"
    datacenterConfig:
      datacenterIdRef:
        name: exampledatacenterk8s
    clusterConfig:
      clusterIdRef:
        name: examplek8s
  providerConfigRef:
    name: example
EOF
  )"

  echo_step "uninstalling k8s nodepool CR"
  echo "${INSTALL_RESOURCE_YAML}" | "${KUBECTL}" delete -f -

  echo_step "wait for deletion k8s cluster CR"
  kubectl wait --for=delete nodepools.k8s/examplek8snodepool --timeout=15m
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
    maintenanceWindow:
      dayOfTheWeek: Friday
      time: "23:40:58Z"
  writeConnectionSecretToRef:
    namespace: default
    name: kubeconfig
  providerConfigRef:
    name: example
EOF
  )"

  echo_step "uninstalling k8s cluster CR"
  echo "${INSTALL_RESOURCE_YAML}" | "${KUBECTL}" delete -f -

  echo_step "wait for deletion k8s cluster CR"
  kubectl wait --for=delete clusters.k8s/examplek8s --timeout=10m
}
