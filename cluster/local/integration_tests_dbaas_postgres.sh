#!/usr/bin/env bash

set -e

## The purpose of this script is to have the tests for
## the dbaas postgres resources
## Please name the functions the following format:
## <resource_name>_tests() and <resource_name>_tests_cleanup().

## DBaaS Postgres Cluster CR Tests
function dbaas_postgres_cluster_tests() {
  echo_step "add psqlcreds and psqlcreds2 secrets"
  "${KUBECTL}" create secret generic psqlcreds --namespace=crossplane-system --from-literal=credentials="{\"username\":\"testuser\",\"password\":\"thisshouldwork111\"}"
  "${KUBECTL}" create secret generic psqlcreds2 --namespace=crossplane-system --from-literal=credentials="{\"username\":\"testuser2\",\"password\":\"thisshouldwork111\"}"

  echo_step "deploy a dbaas postgres cluster CR"
  INSTALL_RESOURCE_YAML="$(
    cat <<EOF
apiVersion: compute.ionoscloud.crossplane.io/v1alpha1
kind: Datacenter
metadata:
  name: exampledbaas
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
apiVersion: compute.ionoscloud.crossplane.io/v1alpha1
kind: Lan
metadata:
  name: examplelandbaas
spec:
  managementPolicies:
    - "*"
  forProvider:
    name: exampleLan
    public: false
    datacenterConfig:
      datacenterIdRef:
        name: exampledbaas
  providerConfigRef:
    name: example
---
apiVersion: dbaas.ionoscloud.crossplane.io/v1alpha1
kind: PostgresCluster
metadata:
  name: example
spec:
  managementPolicies:
    - "*"
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
      source : Secret
      secretRef:
        namespace: crossplane-system
        name: psqlcreds
        key: credentials
    location: de/txl
    backupLocation: de
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
  kubectl wait --for=condition=ready postgresclusters.dbaas.ionoscloud.crossplane.io/example --timeout=3600s
  kubectl wait --for=condition=synced postgresclusters.dbaas.ionoscloud.crossplane.io/example --timeout=3600s

  echo_step "get dbaas postgres cluster CR"
  kubectl get postgresclusters.dbaas.ionoscloud.crossplane.io -o wide

  echo_step "update dbaas postgres cluster CR"
  INSTALL_RESOURCE_YAML="$(
    cat <<EOF
apiVersion: dbaas.ionoscloud.crossplane.io/v1alpha1
kind: PostgresCluster
metadata:
  name: example
spec:
  managementPolicies:
    - "*"
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
      source : Secret
      secretRef:
        namespace: crossplane-system
        name: psqlcreds
        key: credentials
    location: de/txl
    backupLocation: de
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
  kubectl wait --for=condition=ready postgresclusters.dbaas.ionoscloud.crossplane.io/example --timeout=1800s
  kubectl wait --for=condition=synced postgresclusters.dbaas.ionoscloud.crossplane.io/example --timeout=1800s

  echo_step "deploy a dbaas postgres cluster CR"
    INSTALL_RESOURCE_YAML="$(
      cat <<EOF
apiVersion: dbaas.ionoscloud.crossplane.io/v1alpha1
kind: PostgresUser
metadata:
  name: example
spec:
  managementPolicies:
    - "*"
  forProvider:
    credentials:
      source: Secret
      secretRef:
        namespace: crossplane-system
        name: psqlcreds2
        key: credentials
    clusterConfig:
      ClusterIdRef:
        name: testDBaaSPostgres
  providerConfigRef:
    name: example
EOF
    )"

  echo "${INSTALL_RESOURCE_YAML}" | "${KUBECTL}" apply -f -

  echo_step "waiting for dbaas postgres cluster CR to be ready & synced after user creation"
  kubectl wait --for=condition=ready postgresclusters.dbaas.ionoscloud.crossplane.io/example --timeout=1800s
  kubectl wait --for=condition=synced postgresclusters.dbaas.ionoscloud.crossplane.io/example --timeout=1800s
}

function dbaas_postgres_cluster_tests_cleanup() {
  INSTALL_RESOURCE_YAML="$(
    cat <<EOF
apiVersion: compute.ionoscloud.crossplane.io/v1alpha1
kind: Datacenter
metadata:
  name: exampledbaas
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
apiVersion: compute.ionoscloud.crossplane.io/v1alpha1
kind: Lan
metadata:
  name: examplelandbaas
spec:
  managementPolicies:
    - "*"
  forProvider:
    name: exampleLan
    public: false
    datacenterConfig:
      datacenterIdRef:
        name: exampledbaas
  providerConfigRef:
    name: example
---
apiVersion: dbaas.ionoscloud.crossplane.io/v1alpha1
kind: PostgresCluster
metadata:
  name: example
spec:
  managementPolicies:
    - "*"
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
      source : Secret
      secretRef:
        namespace: crossplane-system
        name: psqlcreds
        key: credentials
    location: de/txl
    backupLocation: de
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
  kubectl wait --for=delete postgresclusters.dbaas.ionoscloud.crossplane.io/example --timeout=900s
}
