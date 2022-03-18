#!/usr/bin/env bash

set -e

## The purpose of this script is to have the tests for
## the dbaas postgres resources
## Please name the functions the following format:
## <resource_name>_tests() and <resource_name>_tests_cleanup().

## DBaaS Postgres Cluster CR Tests
function dbaas_postgres_cluster_tests() {
  echo_step "deploy a dbaas postgres cluster CR"
  INSTALL_RESOURCE_YAML="$(
    cat <<EOF
apiVersion: compute.ionoscloud.crossplane.io/v1alpha1
kind: Datacenter
metadata:
  name: exampledbaas
spec:
  forProvider:
    name: exampleDatacenter
    location: de/txl
    description: test
  providerConfigRef:
    name: example
---
apiVersion: compute.ionoscloud.crossplane.io/v1alpha1
kind: Lan
metadata:
  name: examplelandbaas
spec:
  forProvider:
    name: exampleLan
    public: false
    datacenterConfig:
      datacenterIdRef:
        name: exampledbaas
  providerConfigRef:
    name: example
---
apiVersion: dbaas.postgres.ionoscloud.crossplane.io/v1alpha1
kind: Cluster
metadata:
  name: example
spec:
  forProvider:
    displayName: testDBaaS
    postgresVersion: "13"
    connections:
      - datacenterConfig:
          datacenterIdRef:
            name: exampledbaas
        lanConfig:
          lanIdRef:
            name: examplelandbaas
        cidr: 192.168.1.100/24
    credentials:
      username: test
      password: test12345
    location: de/txl
    instances: 1
    cores: 2
    ram: 2048
    storageSize: 20480
    storageType: HDD
    synchronizationMode: ASYNCHRONOUS
  providerConfigRef:
    name: example
EOF
  )"

  echo "${INSTALL_RESOURCE_YAML}" | "${KUBECTL}" apply -f -

  echo_step "waiting for dbaas postgres cluster CR to be ready & synced"
  kubectl wait --for=condition=ready clusters.dbaas.postgres.ionoscloud.crossplane.io/example --timeout=600s
  kubectl wait --for=condition=synced clusters.dbaas.postgres.ionoscloud.crossplane.io/example --timeout=600s

  echo_step "get dbaas postgres cluster CR"
  kubectl get clusters.dbaas.postgres.ionoscloud.crossplane.io -o wide

  echo_step "update dbaas postgres cluster CR"
  INSTALL_RESOURCE_YAML="$(
    cat <<EOF
apiVersion: dbaas.postgres.ionoscloud.crossplane.io/v1alpha1
kind: Cluster
metadata:
  name: example
spec:
  forProvider:
    displayName: testDBaaSPostgres
    postgresVersion: "13"
    connections:
      - datacenterConfig:
          datacenterIdRef:
            name: exampledbaas
        lanConfig:
          lanIdRef:
            name: examplelandbaas
        cidr: 192.168.1.100/24
    credentials:
      username: test
      password: test12345
    location: de/txl
    instances: 1
    cores: 2
    ram: 2048
    storageSize: 20480
    storageType: HDD
    synchronizationMode: ASYNCHRONOUS
  providerConfigRef:
    name: example
EOF
  )"

  echo "${INSTALL_RESOURCE_YAML}" | "${KUBECTL}" apply -f -

  echo_step "waiting for dbaas postgres cluster CR to be ready & synced"
  kubectl wait --for=condition=ready clusters.dbaas.postgres.ionoscloud.crossplane.io/example --timeout=300s
  kubectl wait --for=condition=synced clusters.dbaas.postgres.ionoscloud.crossplane.io/example --timeout=300s
}

function dbaas_postgres_cluster_tests_cleanup() {
  INSTALL_RESOURCE_YAML="$(
    cat <<EOF
apiVersion: compute.ionoscloud.crossplane.io/v1alpha1
kind: Datacenter
metadata:
  name: exampledbaas
spec:
  forProvider:
    name: exampleDatacenter
    location: de/txl
    description: test
  providerConfigRef:
    name: example
---
apiVersion: compute.ionoscloud.crossplane.io/v1alpha1
kind: Lan
metadata:
  name: examplelandbaas
spec:
  forProvider:
    name: exampleLan
    public: false
    datacenterConfig:
      datacenterIdRef:
        name: exampledbaas
  providerConfigRef:
    name: example
---
apiVersion: dbaas.postgres.ionoscloud.crossplane.io/v1alpha1
kind: Cluster
metadata:
  name: example
spec:
  forProvider:
    displayName: testDBaaS
    postgresVersion: "13"
    connections:
      - datacenterConfig:
          datacenterIdRef:
            name: exampledbaas
        lanConfig:
          lanIdRef:
            name: examplelandbaas
        cidr: 192.168.1.100/24
    credentials:
      username: test
      password: test12345
    location: de/txl
    instances: 1
    cores: 2
    ram: 2048
    storageSize: 20480
    storageType: HDD
    synchronizationMode: ASYNCHRONOUS
  providerConfigRef:
    name: example
EOF
  )"

  echo_step "uninstalling dbaas postgres cluster CR"
  echo "${INSTALL_RESOURCE_YAML}" | "${KUBECTL}" delete -f -

  echo_step "wait for deletion dbaas postgres cluster CR"
  kubectl wait --for=delete clusters.dbaas.postgres.ionoscloud.crossplane.io/example --timeout=300s
}
