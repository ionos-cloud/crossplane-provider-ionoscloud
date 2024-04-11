#!/usr/bin/env bash

set -e

## The purpose of this script is to have the tests for
## the dbaas mongo resources
## Please name the functions the following format:
## <resource_name>_tests() and <resource_name>_tests_cleanup().

## DBaaS Mongo Cluster CR Tests
function dbaas_mongo_cluster_tests() {
#  echo_step "add mongocreds and mongocreds2 secrets"
#  "${KUBECTL}" create secret generic mongocreds --namespace=crossplane-system --from-literal=credentials="{\"username\":\"testuser\",\"password\":\"thisshouldwork111\"}"
#  "${KUBECTL}" create secret generic mongocreds2 --namespace=crossplane-system --from-literal=credentials="{\"username\":\"testuser2\",\"password\":\"thisshouldwork111\"}"

  echo_step "deploy a dbaas mongo cluster CR"
  INSTALL_RESOURCE_YAML="$(
    cat <<EOF
apiVersion: compute.ionoscloud.crossplane.io/v1alpha1
kind: Datacenter
metadata:
  name: exampledbaasmongo
spec:
  managementPolicies:
    - "*"
  forProvider:
    name: exampleDatacenter
    location: es/vit
    description: test
  providerConfigRef:
    name: example
---
apiVersion: compute.ionoscloud.crossplane.io/v1alpha1
kind: Lan
metadata:
  name: examplelandbaasmongo
spec:
  managementPolicies:
    - "*"
  forProvider:
    name: exampleLan
    public: false
    datacenterConfig:
      datacenterIdRef:
        name: exampledbaasmongo
  providerConfigRef:
    name: example
---
apiVersion: dbaas.mongo.ionoscloud.crossplane.io/v1alpha1
kind: MongoCluster
metadata:
  name: example
spec:
  managementPolicies:
    - "*"
  forProvider:
    displayName: testDBaaSMongo
    mongoDBVersion: "5.0"
    connections:
      - datacenterConfig:
          datacenterIdRef:
            name: exampledbaasmongo
        lanConfig:
          lanIdRef:
            name: examplelandbaasmongo
        cidr:
          - 192.168.1.100/24
    location: es/vit
    instances: 1
    templateID: 6b78ea06-ee0e-4689-998c-fc9c46e781f6
    synchronizationMode: ASYNCHRONOUS
  providerConfigRef:
    name: example
EOF
  )"

  echo "${INSTALL_RESOURCE_YAML}" | "${KUBECTL}" apply -f -

  echo_step "waiting for dbaas mongo cluster CR to be ready & synced"
  kubectl wait --for=condition=ready mongoclusters.dbaas.mongo.ionoscloud.crossplane.io/example --timeout=3600s
  kubectl wait --for=condition=synced mongoclusters.dbaas.mongo.ionoscloud.crossplane.io/example --timeout=3600s

  echo_step "get dbaas mongo cluster CR"
  kubectl get mongocluster.dbaas.mongo.ionoscloud.crossplane.io -o wide

  echo_step "update dbaas mongo cluster CR"
  INSTALL_RESOURCE_YAML="$(
    cat <<EOF
apiVersion: dbaas.mongo.ionoscloud.crossplane.io/v1alpha1
kind: MongoCluster
metadata:
  name: example
spec:
  managementPolicies:
    - "*"
  forProvider:
    displayName: testDBaaSMongo
    mongoDBVersion: "5.0"
    connections:
      - datacenterConfig:
          datacenterIdRef:
            name: exampledbaasmongo
        lanConfig:
          lanIdRef:
            name: examplelandbaasmongo
        cidr:
          - 192.168.1.100/24
    location: es/vit
    instances: 1
    templateID: 6b78ea06-ee0e-4689-998c-fc9c46e781f6
    synchronizationMode: ASYNCHRONOUS
  providerConfigRef:
    name: example
EOF
  )"

  echo "${INSTALL_RESOURCE_YAML}" | "${KUBECTL}" apply -f -

  echo_step "waiting for dbaas mongo cluster CR to be ready & synced"
  kubectl wait --for=condition=ready mongoclusters.dbaas.mongo.ionoscloud.crossplane.io/example --timeout=1800s
  kubectl wait --for=condition=synced mongoclusters.dbaas.mongo.ionoscloud.crossplane.io/example --timeout=1800s

#  echo_step "deploy a dbaas mongo user CR"
#    INSTALL_RESOURCE_YAML="$(
#      cat <<EOF
#apiVersion: dbaas.ionoscloud.crossplane.io/v1alpha1
#kind: MongoUser
#metadata:
#  name: example
#managementPolicies:
#  - "*"
#spec:
#  forProvider:
#    credentials:
#      source: Secret
#      secretRef:
#        namespace: crossplane-system
#        name: psqlcreds2
#        key: credentials
#    clusterConfig:
#      ClusterIdRef:
#        name: testDBaaSMongo
#  providerConfigRef:
#    name: example
#EOF
#    )"
#
#  echo "${INSTALL_RESOURCE_YAML}" | "${KUBECTL}" apply -f -
#
#  echo_step "waiting for dbaas mongo cluster CR to be ready & synced after user creation"
#  kubectl wait --for=condition=ready mongoclusters.dbaas.mongo.ionoscloud.crossplane.io/example --timeout=1800s
#  kubectl wait --for=condition=synced mongoclusters.dbaas.mongo.ionoscloud.crossplane.io/example --timeout=1800s
}

function dbaas_mongo_cluster_tests_cleanup() {
  INSTALL_RESOURCE_YAML="$(
    cat <<EOF
apiVersion: compute.ionoscloud.crossplane.io/v1alpha1
kind: Datacenter
metadata:
  name: exampledbaasmongo
spec:
  managementPolicies:
    - "*"
  forProvider:
    name: exampleDatacenter
    location: es/vit
    description: test
  providerConfigRef:
    name: example
---
apiVersion: compute.ionoscloud.crossplane.io/v1alpha1
kind: Lan
metadata:
  name: examplelandbaasmongo
spec:
  managementPolicies:
    - "*"
  forProvider:
    name: exampleLan
    public: false
    datacenterConfig:
      datacenterIdRef:
        name: exampledbaasmongo
  providerConfigRef:
    name: example
---
apiVersion: dbaas.mongo.ionoscloud.crossplane.io/v1alpha1
kind: MongoCluster
metadata:
  name: example
spec:
  managementPolicies:
    - "*"
  forProvider:
    displayName: testDBaaSMongo
    mongoDBVersion: "5.0"
    connections:
      - datacenterConfig:
          datacenterIdRef:
            name: exampledbaasmongo
        lanConfig:
          lanIdRef:
            name: examplelandbaasmongo
        cidr:
          - 192.168.1.100/24
    location: es/vit
    instances: 1
    templateID: 6b78ea06-ee0e-4689-998c-fc9c46e781f6
    synchronizationMode: ASYNCHRONOUS
  providerConfigRef:
    name: example
EOF
  )"

  echo_step "uninstalling dbaas mongo cluster CR"
  echo "${INSTALL_RESOURCE_YAML}" | "${KUBECTL}" delete -f -

  echo_step "wait for deletion dbaas mongo cluster CR"
  kubectl wait --for=delete mongoclusters.dbaas.mongo.ionoscloud.crossplane.io/example --timeout=900s
}
