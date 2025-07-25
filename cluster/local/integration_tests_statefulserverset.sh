#!/usr/bin/env bash

set -e

## The purpose of this script is to have the tests for
## the statefulserverset resource.
## Please name the functions the following format:
## <resource_name>_tests() and <resource_name>_tests_cleanup().

## statefulserverset CR Tests
function statefulserverset_tests() {
  echo_step "deploy a statefulserverset CR"
  INSTALL_RESOURCE_YAML="$(
    cat <<EOF
apiVersion: compute.ionoscloud.crossplane.io/v1alpha1
kind: Datacenter
metadata:
  name: examplestatefulserverset
spec:
  managementPolicies:
    - "*"
  forProvider:
    name: examplestatefulserversetDC
    location: de/txl
    description: statefulserverset datacenter
  providerConfigRef:
    name: example
---
apiVersion: compute.ionoscloud.crossplane.io/v1alpha1
kind: StatefulServerSet
metadata:
  name: sss-example
spec:
  forProvider:
    replicas: 2
    removePendingOnReboot: true
    deploymentStrategy:
      type: ZONES
    datacenterConfig:
      datacenterIdRef:
        name: examplestatefulserverset
    template:
      metadata:
        name: server-name
      spec:
        cores: 1
        ram: 1024
        nics:
          - name: nic-customer
            dhcp: false
            dhcpv6: false
            lanReference: customer
            firewallActive: true
            firewallType: INGRESS
            firewallRules:
              - protocol: "TCP"
                name: "rule-tcp"
              - protocol: "ICMP"
                name: "rule-icmp"
    identityConfigMap:
        name: "config-lease"
        namespace: "default"
        keyName: "identity"
    bootVolumeTemplate:
      metadata:
          name: boot-volume
      spec:
        updateStrategy:
          type: "createBeforeDestroyBootVolume"
        image: "c38292f2-eeaa-11ef-8fa7-aee9942a25aa"
        size: 10
        type: HDD
        userData: ""
        imagePassword: "${TEST_IMAGE_PASSWORD}"
        substitutions:
          - options:
              cidr: "fd1d:15db:cf64:1337::/64"
            key: __ipv6Address
            type: ipv6Address
            unique: true
          - options:
              cidr: "192.168.42.0/24"
            key: ipv4Address
            type: ipv4Address
            unique: true
    lans:
      - metadata:
          name: data
        spec:
          public: true
      - metadata:
          name: management
        spec:
          public: false
      - metadata:
          name: customer
        spec:
          ipv6cidr: "AUTO"
          public: false
    volumes:
      - metadata:
          name: storage-disk
        spec:
          size: 10
          type: SSD
      - metadata:
          name: second-storage-disk
        spec:
          size: 20
          type: SSD
  providerConfigRef:
    name: example
EOF
  )"

  echo "${INSTALL_RESOURCE_YAML}" | "${KUBECTL}" apply -f -

  echo_step "describe statefulserverset CR with resources"
  echo_step "waiting for Datacenter resource to be ready"
  kubectl wait --for=condition=ready dc/examplestatefulserverset --timeout=30m
  kubectl get dc
  echo_step "waiting for statefulserverset CR to be ready & synced"
  kubectl wait --for=condition=ready statefulserverset/sss-example --timeout=30m
  kubectl wait --for=condition=synced statefulserverset/sss-example --timeout=30m

  echo_step "get statefulserverset CR"
  kubectl get sss

  echo_step "update a statefulserverset CR"
  INSTALL_RESOURCE_YAML="$(
    cat <<EOF
apiVersion: compute.ionoscloud.crossplane.io/v1alpha1
kind: StatefulServerSet
metadata:
  name: sss-example
spec:
  forProvider:
    replicas: 2
    removePendingOnReboot: true
    deploymentStrategy:
      type: ZONES
    datacenterConfig:
      datacenterIdRef:
        name: examplestatefulserverset
    template:
      metadata:
        name: server-name
      spec:
        cores: 1
        ram: 1024
        nics:
          - name: nic-customer
            dhcp: false
            dhcpv6: false
            lanReference: customer
            firewallActive: true
            firewallType: INGRESS
            firewallRules:
              - protocol: "TCP"
                name: "rule-tcp"
              - protocol: "ICMP"
                name: "rule-icmp"
    identityConfigMap:
        name: "config-lease"
        namespace: "default"
        keyName: "identity"
    bootVolumeTemplate:
      metadata:
          name: boot-volume
      spec:
        updateStrategy:
          type: "createBeforeDestroyBootVolume"
        image: "c38292f2-eeaa-11ef-8fa7-aee9942a25aa"
        size: 10
        type: SSD
        userData: ""
        imagePassword: "${TEST_IMAGE_PASSWORD}"
        substitutions:
          - options:
              cidr: "fd1d:15db:cf64:1337::/64"
            key: __ipv6Address
            type: ipv6Address
            unique: true
          - options:
              cidr: "192.168.42.0/24"
            key: ipv4Address
            type: ipv4Address
            unique: true
    lans:
      - metadata:
          name: data
        spec:
          public: true
      - metadata:
          name: management
        spec:
          public: false
      - metadata:
          name: customer
        spec:
          ipv6cidr: "AUTO"
          public: false
    volumes:
      - metadata:
          name: storage-disk
        spec:
          size: 10
          type: SSD
      - metadata:
          name: second-storage-disk
        spec:
          size: 20
          type: SSD
  providerConfigRef:
    name: example
EOF
  )"

  echo "${INSTALL_RESOURCE_YAML}" | "${KUBECTL}" apply -f -

  echo_step "waiting for updated statefulserverset sss-example CR to be ready & synced"
  kubectl wait --for=condition=ready statefulserverset/sss-example --timeout=20m
  kubectl wait --for=condition=synced statefulserverset/sss-example --timeout=20m

  echo_step "get statefulserverset CR"
  kubectl get sss
}

function statefulserverset_tests_cleanup() {
  INSTALL_RESOURCE_YAML="$(
    cat <<EOF
apiVersion: compute.ionoscloud.crossplane.io/v1alpha1
kind: Datacenter
metadata:
  name: examplestatefulserverset
spec:
  managementPolicies:
    - "*"
  forProvider:
    name: examplestatefulserversetDC
    location: de/txl
    description: statefulserverset datacenter
  providerConfigRef:
    name: example
---
apiVersion: compute.ionoscloud.crossplane.io/v1alpha1
kind: StatefulServerSet
metadata:
  name: sss-example
spec:
  forProvider:
    replicas: 2
    removePendingOnReboot: true
    deploymentStrategy:
      type: ZONES
    datacenterConfig:
      datacenterIdRef:
        name: examplestatefulserverset
    template:
      metadata:
        name: server-name
      spec:
        cores: 1
        ram: 1024
        nics:
          - name: nic-customer
            dhcp: false
            dhcpv6: false
            lanReference: customer
            firewallActive: true
            firewallType: INGRESS
            firewallRules:
              - protocol: "TCP"
                name: "rule-tcp"
              - protocol: "ICMP"
                name: "rule-icmp"
    identityConfigMap:
        name: "config-lease"
        namespace: "default"
        keyName: "identity"
    bootVolumeTemplate:
      metadata:
          name: boot-volume
      spec:
        updateStrategy:
          type: "createBeforeDestroyBootVolume"
        image: "c38292f2-eeaa-11ef-8fa7-aee9942a25aa"
        size: 10
        type: SSD
        userData: ""
        imagePassword: "${TEST_IMAGE_PASSWORD}"
        substitutions:
          - options:
              cidr: "fd1d:15db:cf64:1337::/64"
            key: __ipv6Address
            type: ipv6Address
            unique: true
          - options:
              cidr: "192.168.42.0/24"
            key: ipv4Address
            type: ipv4Address
            unique: true

    lans:
      - metadata:
          name: data
        spec:
          public: true
      - metadata:
          name: management
        spec:
          public: false
      - metadata:
          name: customer
        spec:
          ipv6cidr: "AUTO"
          public: false
    volumes:
      - metadata:
          name: storage-disk
        spec:
          size: 10
          type: SSD
      - metadata:
          name: second-storage-disk
        spec:
          size: 20
          type: SSD
  providerConfigRef:
    name: example
EOF
  )"

  sleep 120
  echo_step "uninstalling statefulserverset CR"
  echo "${INSTALL_RESOURCE_YAML}" | "${KUBECTL}" delete -f -

  echo_step "wait for deletion statefulserverset CR"
  kubectl wait --for=delete statefulserverset/sss-example --timeout=5m
}
