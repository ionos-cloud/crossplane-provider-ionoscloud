#!/usr/bin/env bash

set -e

## The purpose of this script is to have the tests for
## the serverset resource.
## Please name the functions the following format:
## <resource_name>_tests() and <resource_name>_tests_cleanup().

## serverset CR Tests
function serverset_tests() {
  echo_step "deploy a serverset CR"
  INSTALL_RESOURCE_YAML="$(
    cat <<EOF
apiVersion: compute.ionoscloud.crossplane.io/v1alpha1
kind: Datacenter
metadata:
  name: exampleserverset
spec:
  managementPolicies:
    - "*"
  forProvider:
    name: exampleDatacenter
    location: us/las
    description: serverset datacenter
  providerConfigRef:
    name: example
---
apiVersion: compute.ionoscloud.crossplane.io/v1alpha1
kind: Lan
metadata:
  name: examplelan
spec:
  managementPolicies:
    - "*"
  forProvider:
    name: exampleLan
    public: false
    datacenterConfig:
      datacenterIdRef:
        name: exampleserverset
  providerConfigRef:
    name: example
---
apiVersion: compute.ionoscloud.crossplane.io/v1alpha1
kind: ServerSet
metadata:
  name: serverset
spec:
  managementPolicies:
    - "*"
  providerConfigRef:
    name: example
  forProvider:
    replicas: 1
    datacenterConfig:
      datacenterIdRef:
        name: exampleserverset
    template:
      metadata:
        name: server-sample
        labels:
          key: value
      spec:
        cpuFamily: INTEL_XEON
        cores: 1
        ram: 1024
        nics:
          - name: nic-sample
            ipv4: "10.0.0.1/24"
            reference: examplelan
        volumeMounts:
          - reference: "volume-mount-id"
        bootStorageVolumeRef: "volume-id"
    bootVolumeTemplate:
      spec:
        updateStrategy:
#          createBeforeDestroyBootVolume createAllBeforeDestroy
          type: "createBeforeDestroyBootVolume"
        image: "28d0fa34-927f-11ee-8008-6202af74e858"
        size: 10
        type: HDD
#        todo - add sshKeys or imagePassword
#        todo - add userdata in volume creation
        userData: "" #cloud-config
EOF
  )"

  echo "${INSTALL_RESOURCE_YAML}" | "${KUBECTL}" apply -f -

  echo_step "describe serverset CR with resources"
  sleep 30
  kubectl describe volume
  sleep 120
  kubectl describe server
  sleep 120
  kubectl describe serverset/serverset
  echo_step "waiting for serverset CR to be ready & synced"
  kubectl wait --for=condition=ready serverset/serverset --timeout=10m
  kubectl wait --for=condition=synced serverset/serverset --timeout=10m

  echo_step "get serverset CR"
  kubectl get serversets

  echo_step "update a serverset CR"
  INSTALL_RESOURCE_YAML="$(
    cat <<EOF
apiVersion: compute.ionoscloud.crossplane.io/v1alpha1
kind: ServerSet
metadata:
  name: serverset
spec:
  managementPolicies:
    - "*"
  providerConfigRef:
    name: example
  forProvider:
    replicas: 1
    datacenterConfig:
      datacenterIdRef:
        name: example
    template:
      metadata:
        name: server-sample
        labels:
          key: value
      spec:
        cpuFamily: INTEL_XEON
        cores: 1
        ram: 1024
        nics:
          - name: nic-sample
            ipv4: "10.0.0.1/24"
            reference: examplelan
        volumeMounts:
          - reference: "volume-mount-id"
        bootStorageVolumeRef: "volume-id"
    bootVolumeTemplate:
      spec:
        updateStrategy:
#          createBeforeDestroyBootVolume createAllBeforeDestroy
          type: "createBeforeDestroyBootVolume"
        image: "28d0fa34-927f-11ee-8008-6202af74e858"
        size: 10
        type: SSD
#        todo - add sshKeys or imagePassword
#        todo - add userdata in volume creation
        userData: "" #cloud-config
EOF
  )"

  echo "${INSTALL_RESOURCE_YAML}" | "${KUBECTL}" apply -f -

  echo_step "waiting for updated serverset CR to be ready & synced"
  kubectl wait --for=condition=ready serverset/serverset --timeout=5m
  kubectl wait --for=condition=synced serverset/serverset --timeout=5m

  echo_step "get serverset CR"
  kubectl get serverset
}

function serverset_tests_cleanup() {
  INSTALL_RESOURCE_YAML="$(
    cat <<EOF
apiVersion: compute.ionoscloud.crossplane.io/v1alpha1
kind: ServerSet
metadata:
  name: serverset
spec:
  managementPolicies:
    - "*"
  providerConfigRef:
    name: example
  forProvider:
    replicas: 1
    datacenterConfig:
      datacenterIdRef:
        name: example
    template:
      metadata:
        name: server-sample
        labels:
          key: value
      spec:
        cpuFamily: INTEL_XEON
        cores: 1
        ram: 1024
        nics:
          - name: nic-sample
            ipv4: "10.0.0.1/24"
            reference: examplelan
        volumeMounts:
          - reference: "volume-mount-id"
        bootStorageVolumeRef: "volume-id"
    bootVolumeTemplate:
      spec:
        updateStrategy:
#          createBeforeDestroyBootVolume createAllBeforeDestroy
          type: "createBeforeDestroyBootVolume"
        image: "28d0fa34-927f-11ee-8008-6202af74e858"
        size: 10
        type: SSD
#        todo - add sshKeys or imagePassword
#        todo - add userdata in volume creation
        userData: "" #cloud-config
---
apiVersion: compute.ionoscloud.crossplane.io/v1alpha1
kind: Lan
metadata:
  name: examplelan
spec:
  managementPolicies:
    - "*"
  forProvider:
    name: exampleLan
    public: false
    datacenterConfig:
      datacenterIdRef:
        name: exampleserverset
  providerConfigRef:
    name: example
---
apiVersion: compute.ionoscloud.crossplane.io/v1alpha1
kind: Datacenter
metadata:
  name: exampleserverset
spec:
  managementPolicies:
    - "*"
  forProvider:
    name: exampleDatacenter
    location: us/las
    description: serverset datacenter
  providerConfigRef:
    name: example
EOF
  )"

  sleep 120
  echo_step "uninstalling serverset CR"
  echo "${INSTALL_RESOURCE_YAML}" | "${KUBECTL}" delete -f -

  echo_step "wait for deletion serverset CR"
  kubectl wait --for=delete serverset/serverset --timeout=5m
}
