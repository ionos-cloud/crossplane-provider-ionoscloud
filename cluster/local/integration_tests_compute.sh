#!/usr/bin/env bash

set -e

## The purpose of this script is to have the tests for
## the compute-engine resources
## Please name the functions the following format:
## <resource_name>_tests() and <resource_name>_tests_cleanup().

## IPBlock CR Tests
function ipblock_tests() {
  echo_step "deploy a ipblock CR"
  INSTALL_RESOURCE_YAML="$(
    cat <<EOF
apiVersion: compute.ionoscloud.crossplane.io/v1alpha1
kind: IPBlock
managementPolicies:
  - '*'
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

  echo_step "waiting for ipblock CR to be ready & synced"
  sleep 5
  kubectl describe ipblock
  kubectl wait --for=condition=ready ipblocks/example
  kubectl wait --for=condition=synced ipblocks/example

  echo_step "get ipblock CR"
  kubectl get ipblocks -o wide

  echo_step "update ipblock CR"
  INSTALL_RESOURCE_YAML="$(
    cat <<EOF
apiVersion: compute.ionoscloud.crossplane.io/v1alpha1
kind: IPBlock
metadata:
  name: example
managementPolicies:
  - '*'
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

  echo_step "waiting for ipblock CR to be ready & synced"
  kubectl wait --for=condition=ready ipblocks/example
  kubectl wait --for=condition=synced ipblocks/example

  echo_step "get updated ipblock CR"
  kubectl get ipblocks -o wide
}

function ipblock_tests_cleanup() {
  INSTALL_RESOURCE_YAML="$(
    cat <<EOF
apiVersion: compute.ionoscloud.crossplane.io/v1alpha1
kind: IPBlock
metadata:
  name: example
managementPolicies:
  - "*"
spec:
  forProvider:
    name: exampleIpBlockUpdate
    size: 2
    location: de/txl
  providerConfigRef:
    name: example
EOF
  )"

  echo_step "uninstalling ipblock CR"
  echo "${INSTALL_RESOURCE_YAML}" | "${KUBECTL}" delete -f -

  echo_step "wait for deletion ipblock CR"
  kubectl wait --for=delete ipblocks/example
}

## Datacenter CR Tests
function datacenter_tests() {
  echo_step "deploy a datacenter CR"
  INSTALL_RESOURCE_YAML="$(
    cat <<EOF
apiVersion: compute.ionoscloud.crossplane.io/v1alpha1
kind: Datacenter
metadata:
  name: example
managementPolicies: 
  - "*"
spec:
  forProvider:
    name: testdatacenter
    location: de/txl
  providerConfigRef:
    name: example
EOF
  )"

  echo "${INSTALL_RESOURCE_YAML}" | "${KUBECTL}" apply -f -

  echo_step "waiting for datacenter CR to be ready & synced"
  sleep 5
  kubectl describe datacenters
  kubectl wait --for=condition=ready datacenters/example --timeout=90s
  kubectl wait --for=condition=synced datacenters/example --timeout=90s

  echo_step "get datacenter CR"
  kubectl get datacenters

  echo_step "update datacenter CR"
  INSTALL_RESOURCE_YAML="$(
    cat <<EOF
apiVersion: compute.ionoscloud.crossplane.io/v1alpha1
kind: Datacenter
metadata:
  name: example
managementPolicies: 
  - "*"
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

  echo_step "waiting for datacenter CR to be ready & synced"
  sleep 5
  kubectl describe datacenters
  kubectl wait --for=condition=ready datacenters/example --timeout=90s
  kubectl wait --for=condition=synced datacenters/example --timeout=90s
}

function datacenter_tests_cleanup() {
  INSTALL_RESOURCE_YAML="$(
    cat <<EOF
apiVersion: compute.ionoscloud.crossplane.io/v1alpha1
kind: Datacenter
metadata:
  name: example
managementPolicies: 
  - "*"
spec:
  forProvider:
    name: Test Datacenter CR
    location: de/txl
    description: e2e crossplane testing
  providerConfigRef:
    name: example
EOF
  )"

  echo_step "uninstalling datacenter CR"
  echo "${INSTALL_RESOURCE_YAML}" | "${KUBECTL}" delete -f -

  echo_step "wait for deletion datacenter CR"
  kubectl wait --for=delete datacenters/example
}

## Volume CR Tests
function volume_tests() {
  echo_step "deploy a volume CR"
  INSTALL_RESOURCE_YAML="$(
    cat <<EOF
apiVersion: compute.ionoscloud.crossplane.io/v1alpha1
kind: Volume
metadata:
  name: example
managementPolicies: 
  - "*"
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

  echo_step "waiting for volume CR to be ready & synced"
  sleep 5
  kubectl describe volumes
  kubectl wait --for=condition=ready volumes/example --timeout=90s
  kubectl wait --for=condition=synced volumes/example --timeout=90s

  echo_step "get volume CR"
  kubectl get volumes

  echo_step "update volume CR"
  INSTALL_RESOURCE_YAML="$(
    cat <<EOF
apiVersion: compute.ionoscloud.crossplane.io/v1alpha1
kind: Volume
metadata:
  name: example
managementPolicies: 
  - "*"
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

  echo_step "waiting for updated volume CR to be ready & synced"
  kubectl wait --for=condition=ready volumes/example
  kubectl wait --for=condition=synced volumes/example
}

function volume_tests_cleanup() {
  INSTALL_RESOURCE_YAML="$(
    cat <<EOF
apiVersion: compute.ionoscloud.crossplane.io/v1alpha1
kind: Volume
metadata:
  name: example
managementPolicies: 
  - "*"
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

  echo_step "uninstalling volume CR"
  echo "${INSTALL_RESOURCE_YAML}" | "${KUBECTL}" delete -f -

  echo_step "wait for deletion volume CR"
  kubectl wait --for=delete volumes/example
}

## Server CR Tests
function server_tests() {
  echo_step "deploy a server CR and attach volume"
  INSTALL_RESOURCE_YAML="$(
    cat <<EOF
apiVersion: compute.ionoscloud.crossplane.io/v1alpha1
kind: Server
metadata:
  name: example
managementPolicies: 
  - "*"
spec:
  forProvider:
    name: exampletest
    cores: 4
    ram: 2048
    availabilityZone: AUTO
    cpuFamily: INTEL_SKYLAKE
    datacenterConfig:
      datacenterIdRef:
        name: example
    volumeConfig:
      volumeIdRef:
        name: example
  providerConfigRef:
    name: example
EOF
  )"

  echo "${INSTALL_RESOURCE_YAML}" | "${KUBECTL}" apply -f -

  echo_step "waiting for server CR to be ready & synced after creation"
  sleep 5
  kubectl describe servers
  kubectl wait --for=condition=ready servers/example --timeout=420s
  kubectl wait --for=condition=synced servers/example --timeout=420s

  echo_step "get server CR"
  kubectl get servers

  echo_step "update server CR"
  INSTALL_RESOURCE_YAML="$(
    cat <<EOF
apiVersion: compute.ionoscloud.crossplane.io/v1alpha1
kind: Server
metadata:
  name: example
managementPolicies: 
  - "*"
spec:
  forProvider:
    name: exampleServerUpdate
    cores: 4
    ram: 2048
    availabilityZone: AUTO
    cpuFamily: INTEL_SKYLAKE
    datacenterConfig:
      datacenterIdRef:
        name: example
    volumeConfig:
      volumeIdRef:
        name: example
  providerConfigRef:
    name: example
EOF
  )"

  echo "${INSTALL_RESOURCE_YAML}" | "${KUBECTL}" apply -f -

  echo_step "waiting for server CR to be ready & synced"
  sleep 5
  kubectl describe servers
  kubectl wait --for=condition=ready servers/example --timeout=90s
  kubectl wait --for=condition=synced servers/example --timeout=90s
}

function server_tests_cleanup() {
  INSTALL_RESOURCE_YAML="$(
    cat <<EOF
apiVersion: compute.ionoscloud.crossplane.io/v1alpha1
kind: Server
metadata:
  name: example
managementPolicies: 
  - "*"
spec:
  forProvider:
    name: exampleServerUpdate
    cores: 4
    ram: 2048
    availabilityZone: AUTO
    cpuFamily: INTEL_SKYLAKE
    datacenterConfig:
      datacenterIdRef:
        name: example
    volumeConfig:
      volumeIdRef:
        name: example
  providerConfigRef:
    name: example
EOF
  )"

  echo_step "uninstalling server CR"
  echo "${INSTALL_RESOURCE_YAML}" | "${KUBECTL}" delete -f -

  echo_step "wait for deletion server CR"
  kubectl wait --for=delete servers/example
}

## Lan CR Tests
function lan_tests() {
  echo_step "deploy a lan CR"
  INSTALL_RESOURCE_YAML="$(
    cat <<EOF
apiVersion: compute.ionoscloud.crossplane.io/v1alpha1
kind: Lan
metadata:
  name: example
managementPolicies: 
  - "*"
spec:
  forProvider:
    name: exampletest
    public: false
    datacenterConfig:
      datacenterIdRef:
        name: example
  providerConfigRef:
    name: example
EOF
  )"

  echo "${INSTALL_RESOURCE_YAML}" | "${KUBECTL}" apply -f -

  echo_step "waiting for lan CR to be ready & synced"
  sleep 5
  kubectl describe lans
  kubectl wait --for=condition=ready lans/example  --timeout=90s
  kubectl wait --for=condition=synced lans/example  --timeout=90s

  echo_step "get lan CR"
  kubectl get lans

  echo_step "update lan CR"
  INSTALL_RESOURCE_YAML="$(
    cat <<EOF
apiVersion: compute.ionoscloud.crossplane.io/v1alpha1
kind: Lan
metadata:
  name: example
managementPolicies: 
  - "*"
spec:
  forProvider:
    name: exampletestLan
    public: true
    datacenterConfig:
      datacenterIdRef:
        name: example
  providerConfigRef:
    name: example
EOF
  )"

  echo "${INSTALL_RESOURCE_YAML}" | "${KUBECTL}" apply -f -

  echo_step "waiting for update lan CR to be ready & synced"
  sleep 5
  kubectl describe lans
  kubectl wait --for=condition=ready lans/example
  kubectl wait --for=condition=synced lans/example
}

function lan_tests_cleanup() {
  INSTALL_RESOURCE_YAML="$(
    cat <<EOF
apiVersion: compute.ionoscloud.crossplane.io/v1alpha1
kind: Lan
metadata:
  name: example
managementPolicies: 
  - "*"
spec:
  forProvider:
    name: exampletestLan
    public: true
    datacenterConfig:
      datacenterIdRef:
        name: example
  providerConfigRef:
    name: example
EOF
  )"

  echo_step "uninstalling lan CR"
  echo "${INSTALL_RESOURCE_YAML}" | "${KUBECTL}" delete -f -

  echo_step "wait for deletion lan CR"
  kubectl wait --for=delete lans/example
}

## Nic CR Tests
function nic_tests() {
  echo_step "deploy a nic CR"
  INSTALL_RESOURCE_YAML="$(
    cat <<EOF
apiVersion: compute.ionoscloud.crossplane.io/v1alpha1
kind: Nic
metadata:
  name: example
managementPolicies: 
  - "*"
spec:
  forProvider:
    name: exampleNic
    dhcp: false
    ipsConfigs:
      ipsBlockConfigs:
        - ipBlockIdRef:
            name: example
    datacenterConfig:
      datacenterIdRef:
        name: example
    serverConfig:
      serverIdRef:
        name: example
    lanConfig:
      lanIdRef:
        name: example
  providerConfigRef:
    name: example
EOF
  )"

  echo "${INSTALL_RESOURCE_YAML}" | "${KUBECTL}" apply -f -

  echo_step "waiting for nic CR to be ready & synced"
  sleep 10s
  kubectl describe nics
  kubectl wait --for=condition=ready nics/example --timeout 120s
  kubectl wait --for=condition=synced nics/example --timeout 120s

  echo_step "get nic CR"
  kubectl get nics
}

function nic_tests_cleanup() {
  INSTALL_RESOURCE_YAML="$(
    cat <<EOF
apiVersion: compute.ionoscloud.crossplane.io/v1alpha1
kind: Nic
metadata:
  name: example
managementPolicies: 
  - "*"
spec:
  forProvider:
    name: exampleNic
    dhcp: false
    ipsConfigs:
      ipsBlockConfigs:
        - ipBlockIdRef:
            name: example
    datacenterConfig:
      datacenterIdRef:
        name: example
    serverConfig:
      serverIdRef:
        name: example
    lanConfig:
      lanIdRef:
        name: example
  providerConfigRef:
    name: example
EOF
  )"

  echo_step "uninstalling nic CR"
  echo "${INSTALL_RESOURCE_YAML}" | "${KUBECTL}" delete -f -

  echo_step "wait for deletion nic CR"
  kubectl wait --for=delete nics/example
}

## FirewallRule CR Tests
function firewallrule_tests() {
  echo_step "deploy a firewallrule CR"
  INSTALL_RESOURCE_YAML="$(
    cat <<EOF
apiVersion: compute.ionoscloud.crossplane.io/v1alpha1
kind: FirewallRule
metadata:
  name: example
managementPolicies: 
  - "*"
spec:
  forProvider:
    name: exampleFirewallRule
    protocol: ANY
    type: EGRESS
    sourceIpConfig:
      ip: 192.168.42.2/31
    targetIpConfig:
      ipBlockConfig:
        ipBlockIdRef:
          name: example
        index: 1
    datacenterConfig:
      datacenterIdRef:
        name: example
    serverConfig:
      serverIdRef:
        name: example
    nicConfig:
      nicIdRef:
        name: example
  providerConfigRef:
    name: example
EOF
  )"

  echo "${INSTALL_RESOURCE_YAML}" | "${KUBECTL}" apply -f -

  echo_step "waiting for firewallrule CR to be ready & synced"
  kubectl wait --for=condition=ready firewallrules/example --timeout 120s
  kubectl wait --for=condition=synced firewallrules/example --timeout 120s

  echo_step "get firewallrule CR"
  kubectl get firewallrules

  echo_step "update firewallrule CR"
  INSTALL_RESOURCE_YAML="$(
    cat <<EOF
apiVersion: compute.ionoscloud.crossplane.io/v1alpha1
kind: FirewallRule
metadata:
  name: example
managementPolicies: 
  - "*"
spec:
  forProvider:
    name: exampleFirewallRuleUpdated
    protocol: ANY
    type: EGRESS
    sourceIpConfig:
      ip: 192.168.42.2/31
    targetIpConfig:
      ip: 192.168.24.3
    datacenterConfig:
      datacenterIdRef:
        name: example
    serverConfig:
      serverIdRef:
        name: example
    nicConfig:
      nicIdRef:
        name: example
  providerConfigRef:
    name: example
EOF
  )"

  echo "${INSTALL_RESOURCE_YAML}" | "${KUBECTL}" apply -f -

  echo_step "waiting for firewallrule CR to be ready & synced"
  kubectl wait --for=condition=ready firewallrules/example --timeout 120s
  kubectl wait --for=condition=synced firewallrules/example --timeout 120s
}

function firewallrule_tests_cleanup() {
  INSTALL_RESOURCE_YAML="$(
    cat <<EOF
apiVersion: compute.ionoscloud.crossplane.io/v1alpha1
kind: FirewallRule
metadata:
  name: example
managementPolicies: 
  - "*"
spec:
  forProvider:
    name: exampleFirewallRuleUpdated
    protocol: ANY
    type: EGRESS
    datacenterConfig:
      datacenterIdRef:
        name: example
    serverConfig:
      serverIdRef:
        name: example
    nicConfig:
      nicIdRef:
        name: example
  providerConfigRef:
    name: example
EOF
  )"

  echo_step "uninstalling firewallrule CR"
  echo "${INSTALL_RESOURCE_YAML}" | "${KUBECTL}" delete -f -

  echo_step "wait for deletion firewallrule CR"
  kubectl wait --for=delete firewallrule/example
}

## IPFailover CR Tests
function ipfailover_tests() {
  echo_step "deploy a ipfailover CR"
  INSTALL_RESOURCE_YAML="$(
    cat <<EOF
apiVersion: compute.ionoscloud.crossplane.io/v1alpha1
kind: IPFailover
metadata:
  name: example
managementPolicies: 
  - "*"
spec:
  forProvider:
    ipConfig:
      ipBlockConfig:
        ipBlockIdRef:
          name: example
        index: 0
    datacenterConfig:
      datacenterIdRef:
        name: example
    lanConfig:
      lanIdRef:
        name: example
    nicConfig:
      nicIdRef:
        name: example
  providerConfigRef:
    name: example
EOF
  )"

  echo "${INSTALL_RESOURCE_YAML}" | "${KUBECTL}" apply -f -

  echo_step "waiting for ipfailover CR to be ready & synced"
  sleep 10
  kubectl describe ipfailovers
  kubectl wait --for=condition=ready ipfailovers/example --timeout 120s
  kubectl wait --for=condition=synced ipfailovers/example --timeout 120s

  echo_step "get ipfailover CR"
  kubectl get ipfailovers

  echo_step "update ipfailover CR"
  INSTALL_RESOURCE_YAML="$(
    cat <<EOF
apiVersion: compute.ionoscloud.crossplane.io/v1alpha1
kind: IPFailover
metadata:
  name: example
managementPolicies: 
  - "*"
spec:
  forProvider:
    ipConfig:
      ipBlockConfig:
        ipBlockIdRef:
          name: example
        index: 1
    datacenterConfig:
      datacenterIdRef:
        name: example
    lanConfig:
      lanIdRef:
        name: example
    nicConfig:
      nicIdRef:
        name: example
  providerConfigRef:
    name: example
EOF
  )"

  echo "${INSTALL_RESOURCE_YAML}" | "${KUBECTL}" apply -f -

  echo_step "waiting for ipfailover CR after update to be ready & synced"
  kubectl wait --for=condition=ready ipfailovers/example --timeout 120s
  kubectl wait --for=condition=synced ipfailovers/example --timeout 120s
}

function ipfailover_tests_cleanup() {
  INSTALL_RESOURCE_YAML="$(
    cat <<EOF
apiVersion: compute.ionoscloud.crossplane.io/v1alpha1
kind: IPFailover
metadata:
  name: example
managementPolicies: 
  - "*"
spec:
  forProvider:
    ipConfig:
      ipBlockConfig:
        ipBlockIdRef:
          name: example
        index: 1
    datacenterConfig:
      datacenterIdRef:
        name: example
    lanConfig:
      lanIdRef:
        name: example
    nicConfig:
      nicIdRef:
        name: example
  providerConfigRef:
    name: example
EOF
  )"

  echo_step "uninstalling ipfailover CR"
  echo "${INSTALL_RESOURCE_YAML}" | "${KUBECTL}" delete -f -

  echo_step "wait for deletion ipfailover CR"
  kubectl wait --for=delete ipfailovers/example
}
