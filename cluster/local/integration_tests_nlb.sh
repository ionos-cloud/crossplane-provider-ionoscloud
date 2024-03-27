#!/usr/bin/env bash

set -e

## The purpose of this script is to have the tests for
## the NetworkLoadBalancer resources
## Please name the functions the following format:
## <resource_name>_tests() and <resource_name>_tests_cleanup().


## NetworkLoadBalancer CR Tests
function nlb_tests() {
  echo_step "deploy a network load balancer CR"
  INSTALL_RESOURCE_YAML="$(
    cat <<EOF
apiVersion: compute.ionoscloud.crossplane.io/v1alpha1
kind: IPBlock
metadata:
  name: examplenlb
spec:
  managementPolicies:
    - "*"
  forProvider:
    name: exampleIpBlockNLB
    size: 3
    location: de/txl
  providerConfigRef:
    name: example
---
apiVersion: compute.ionoscloud.crossplane.io/v1alpha1
kind: Datacenter
metadata:
  name: examplenlb
spec:
  managementPolicies:
    - "*"
  forProvider:
    name: exampleDatacenterNLB
    location: de/txl
    description: test
  providerConfigRef:
    name: example
---
apiVersion: compute.ionoscloud.crossplane.io/v1alpha1
kind: Lan
metadata:
  name: listenernlb
spec:
  forProvider:
    name: exampleListenerLanNLB
    public: true
    datacenterConfig:
      datacenterIdRef:
        name: examplenlb
  providerConfigRef:
    name: example
---
apiVersion: compute.ionoscloud.crossplane.io/v1alpha1
kind: Lan
metadata:
  name: targetnlb
spec:
  managementPolicies:
    - "*"
  forProvider:
    name: exampleTargetLanNLB
    public: false
    datacenterConfig:
      datacenterIdRef:
        name: examplenlb
  providerConfigRef:
    name: example
---
apiVersion: nlb.ionoscloud.crossplane.io/v1alpha1
kind: NetworkLoadBalancer
metadata:
  name: examplenlb
spec:
  managementPolicies:
    - "*"
  forProvider:
    datacenterConfig:
      datacenterIdRef:
        name: examplenlb
    name: exampleNetworkLoadBalancer
    targetLanConfig:
      lanIdRef:
        name: targetnlb
    listenerLanConfig:
      lanIdRef:
        name: listenernlb
    ipsConfig:
      ipsBlocksConfig:
        - ipBlockConfig:
            ipBlockIdRef:
              name: nlbipblock
          indexes: [ 0 ]
  providerConfigRef:
    name: example
EOF
  )"

  echo "${INSTALL_RESOURCE_YAML}" | "${KUBECTL}" apply -f -

  echo_step "waiting for prerequisities to be ready & synced"
  kubectl wait --for=condition=ready ipblocks.compute.ionoscloud.crossplane.io/examplenlb datacenters.compute.ionoscloud.crossplane.io/examplenlb lans.compute.ionoscloud.crossplane.io/targetnlb lans.compute.ionoscloud.crossplane.io/listenernlb
  kubectl wait --for=condition=synced ipblocks.compute.ionoscloud.crossplane.io/examplenlb datacenters.compute.ionoscloud.crossplane.io/examplenlb lans.compute.ionoscloud.crossplane.io/targetnlb lans.compute.ionoscloud.crossplane.io/listenernlb

  echo_step "waiting for application load balancer CR to be ready & synced"
  kubectl wait --for=condition=ready networkloadbalancers.nlb.ionoscloud.crossplane.io/examplenlb --timeout=30m
  kubectl wait --for=condition=synced networkloadbalancers.nlb.ionoscloud.crossplane.io/examplenlb --timeout=30m

  echo_step "get application load balancer CR"
  kubectl get networkloadbalancers.nlb.ionoscloud.crossplane.io -o wide

  echo_step "update application load balancer CR"
  INSTALL_RESOURCE_YAML="$(
    cat <<EOF
apiVersion: nlb.ionoscloud.crossplane.io/v1alpha1
kind: NetworkLoadBalancer
metadata:
  name: examplenlb
spec:
  managementPolicies:
    - "*"
  forProvider:
    datacenterConfig:
      datacenterIdRef:
        name: examplenlb
    name: exampleNetworkLoadBalancerUpdated
    targetLanConfig:
      lanIdRef:
        name: targetnlb
    listenerLanConfig:
      lanIdRef:
        name: listenernlb
    ipsConfig:
      ipsBlockConfigs:
        - ipBlockIdRef:
            name: examplenlb
          indexes: [ 1 ]
  providerConfigRef:
    name: example
EOF
  )"

  echo "${INSTALL_RESOURCE_YAML}" | "${KUBECTL}" apply -f -

  echo_step "waiting for application load balancer CR to be ready & synced"
  kubectl wait --for=condition=ready networkloadbalancers.nlb.ionoscloud.crossplane.io/examplenlb --timeout=30m
  kubectl wait --for=condition=synced networkloadbalancers.nlb.ionoscloud.crossplane.io/examplenlb --timeout=30m

  echo_step "get updated application load balancer CR"
  kubectl get networkloadbalancers.nlb.ionoscloud.crossplane.io -o wide
}

function nlb_tests_cleanup() {
  INSTALL_RESOURCE_YAML="$(
    cat <<EOF
apiVersion: nlb.ionoscloud.crossplane.io/v1alpha1
kind: NetworkLoadBalancer
metadata:
  name: examplenlb
spec:
  managementPolicies:
    - "*"
  forProvider:
    datacenterConfig:
      datacenterIdRef:
        name: examplenlb
    name: exampleNetworkLoadBalancerUpdated
    targetLanConfig:
      lanIdRef:
        name: targetnlb
    listenerLanConfig:
      lanIdRef:
        name: listenernlb
    ipsConfig:
      ipsBlockConfigs:
        - ipBlockIdRef:
            name: examplenlb
          indexes: [ 0 ]
  providerConfigRef:
    name: example
EOF
  )"

  echo_step "uninstalling application load balancer CR"
  echo "${INSTALL_RESOURCE_YAML}" | "${KUBECTL}" delete -f -

  echo_step "wait for deletion application load balancer CR"
  kubectl wait --for=delete networkloadbalancers.nlb.ionoscloud.crossplane.io/examplenlb --timeout=30m

  INSTALL_RESOURCE_YAML="$(
    cat <<EOF
apiVersion: compute.ionoscloud.crossplane.io/v1alpha1
kind: Lan
metadata:
  name: listenernlb
spec:
  managementPolicies:
    - "*"
  forProvider:
    name: exampleListenerLanNLB
    public: true
    datacenterConfig:
      datacenterIdRef:
        name: examplenlb
  providerConfigRef:
    name: example
---
apiVersion: compute.ionoscloud.crossplane.io/v1alpha1
kind: Lan
metadata:
  name: targetnlb
spec:
  forProvider:
    name: exampleTargetLanNLB
    public: false
    datacenterConfig:
      datacenterIdRef:
        name: examplenlb
  providerConfigRef:
    name: example
---
apiVersion: compute.ionoscloud.crossplane.io/v1alpha1
kind: Datacenter
metadata:
  name: examplenlb
spec:
  managementPolicies:
    - "*"
  forProvider:
    name: exampleDatacenterNLB
    location: de/txl
    description: test
  providerConfigRef:
    name: example
---
apiVersion: compute.ionoscloud.crossplane.io/v1alpha1
kind: IPBlock
metadata:
  name: examplenlb
spec:
  forProvider:
    name: exampleIpBlockNLB
    size: 2
    location: de/txl
  providerConfigRef:
    name: example
EOF
  )"

  echo_step "uninstalling prerequisites for application load balancer CR"
  echo "${INSTALL_RESOURCE_YAML}" | "${KUBECTL}" delete -f -

  echo_step "wait for deletion of prerequisites for application load balancer CR"
  kubectl wait --for=delete lans.compute.ionoscloud.crossplane.io/examplenlb lans.compute.ionoscloud.crossplane.io/examplelannlb
  kubectl wait --for=delete datacenters.compute.ionoscloud.crossplane.io/examplenlb
  kubectl wait --for=delete ipblocks.compute.ionoscloud.crossplane.io/examplenlb
}

## ForwardingRule CR Tests
function nlbforwardingrule_tests() {
  echo_step "deploy a forwarding rule CR"
  INSTALL_RESOURCE_YAML="$(
    cat <<EOF
apiVersion: nlb.ionoscloud.crossplane.io/v1alpha1
kind: ForwardingRule
metadata:
  name: examplenlb
spec:
  managementPolicies:
    - "*"
  forProvider:
    datacenterConfig:
      datacenterIdRef:
        name: examplenlb
    networkLoadBalancerConfig:
      networkLoadBalancerIdRef:
        name: examplenlb
    listenerIpConfig:
      ipBlockConfig:
        ipBlockIdRef:
          name: examplenlb
        index: 2
    listenerPort: 8081
    name: exampleForwardingRuleNLB
    listenerPort: 8081
    algorithm: RANDOM
    protocol: TCP
    targets:
      - ip:
          ip: "93.93.119.108"
        port: 31234
        weight: 10
    healthCheck:
      targetTimeout: 30000
      retries: 3
      clientTimeout: 70000
      connectTimeout: 60000
  providerConfigRef:
    name: example
EOF
  )"

  echo "${INSTALL_RESOURCE_YAML}" | "${KUBECTL}" apply -f -

  echo_step "waiting for forwarding rule CR to be ready & synced"
  kubectl wait --for=condition=ready forwardingrules.nlb.ionoscloud.crossplane.io/examplenlb --timeout=30m
  kubectl wait --for=condition=synced forwardingrules.nlb.ionoscloud.crossplane.io/examplenlb --timeout=30m

  echo_step "get forwarding rule CR"
  kubectl get forwardingrules.nlb.ionoscloud.crossplane.io

  echo_step "update forwarding rule CR"
  INSTALL_RESOURCE_YAML="$(
    cat <<EOF
apiVersion: nlb.ionoscloud.crossplane.io/v1alpha1
kind: ForwardingRule
metadata:
  name: examplenlb
spec:
  managementPolicies:
    - "*"
  forProvider:
    datacenterConfig:
      datacenterIdRef:
        name: examplenlb
    networkLoadBalancerConfig:
      networkLoadBalancerIdRef:
        name: examplenlb
    listenerIpConfig:
      ipBlockConfig:
        ipBlockIdRef:
          name: examplenlb
        index: 2
    listenerPort: 8081
    name: exampleForwardingRuleNLB
    listenerPort: 8081
    algorithm: ROUND_ROBIN
    protocol: TCP
    targets:
      - ip:
          ip: "93.93.119.108"
        port: 31234
        weight: 10
      - ip:
          ip: "93.93.119.110"
        port: 31234
        weight: 10
        healthCheck:
          checkInterval: 8000
          check: true
          maintenance: true
    healthCheck:
      targetTimeout: 30000
      retries: 5
      clientTimeout: 70000
      connectTimeout: 60000
  providerConfigRef:
    name: example
EOF
  )"

  echo "${INSTALL_RESOURCE_YAML}" | "${KUBECTL}" apply -f -

  echo_step "waiting for forwarding rule CR to be ready & synced"
  kubectl wait --for=condition=ready forwardingrules.nlb.ionoscloud.crossplane.io/examplenlb --timeout=30m
  kubectl wait --for=condition=synced forwardingrules.nlb.ionoscloud.crossplane.io/examplenlb --timeout=30m
}

function nlbforwardingrule_tests_cleanup() {
  INSTALL_RESOURCE_YAML="$(
    cat <<EOF
apiVersion: nlb.ionoscloud.crossplane.io/v1alpha1
kind: ForwardingRule
metadata:
  name: examplenlb
spec:
  managementPolicies:
    - "*"
  forProvider:
    datacenterConfig:
      datacenterIdRef:
        name: examplenlb
    networkLoadBalancerConfig:
      networkLoadBalancerIdRef:
        name: examplenlb
    listenerIpConfig:
      ipBlockConfig:
        ipBlockIdRef:
          name: examplenlb
        index: 2
    listenerPort: 8081
    name: exampleForwardingRuleNLB
    listenerPort: 8081
    algorithm: ROUND_ROBIN
    protocol: TCP
    targets:
      - ip:
          ip: "93.93.119.108"
        port: 31234
        weight: 10
      - ip:
          ip: "93.93.119.110"
        port: 31234
        weight: 10
        healthCheck:
          checkInterval: 8000
          check: true
          maintenance: true
    healthCheck:
      targetTimeout: 30000
      retries: 5
      clientTimeout: 70000
      connectTimeout: 60000
  providerConfigRef:
    name: example
EOF
  )"

  echo_step "uninstalling forwarding rule CR"
  echo "${INSTALL_RESOURCE_YAML}" | "${KUBECTL}" delete -f -

  echo_step "wait for deletion forwarding rule CR"
  kubectl wait --for=delete forwardingrules.nlb.ionoscloud.crossplane.io/examplenlb --timeout=30m
}

## FlowLog CR Tests
function nlbflowlog_tests() {
  echo_step "deploy a flow log CR"
  INSTALL_RESOURCE_YAML="$(
    cat <<EOF
apiVersion: nlb.ionoscloud.crossplane.io/v1alpha1
kind: FlowLog
metadata:
  name: examplenlb
spec:
  managementPolicies:
    - "*"
  forProvider:
    datacenterConfig:
      datacenterIdRef:
        name: examplenlb
    networkLoadBalancerConfig:
      networkLoadBalancerIdRef:
        name: examplenlb
    name: example
    action: ACCEPTED
    direction: INGRESS
    bucket: flowlog-acceptance-test
  providerConfigRef:
    name: example
EOF
  )"

  echo "${INSTALL_RESOURCE_YAML}" | "${KUBECTL}" apply -f -

  echo_step "waiting for flow log CR to be ready & synced"
  kubectl wait --for=condition=ready flowlogs.nlb.ionoscloud.crossplane.io/examplenlb
  kubectl wait --for=condition=synced flowlogs.nlb.ionoscloud.crossplane.io/examplenlb

  echo_step "get flow log CR"
  kubectl get flowlogs.nlb.ionoscloud.crossplane.io

  echo_step "update flow log CR"
  INSTALL_RESOURCE_YAML="$(
    cat <<EOF
apiVersion: compute.ionoscloud.crossplane.io/v1alpha1
apiVersion: nlb.ionoscloud.crossplane.io/v1alpha1
kind: FlowLog
metadata:
  name: examplenlb
spec:
  managementPolicies:
    - "*"
  forProvider:
    datacenterConfig:
      datacenterIdRef:
        name: examplenlb
    networkLoadBalancerConfig:
      networkLoadBalancerIdRef:
        name: examplenlb
    name: exampleUpdated
    action: ACCEPTED
    direction: BIDIRECTIONAL
    bucket: flowlog-acceptance-test
  providerConfigRef:
    name: example
EOF
  )"

  echo "${INSTALL_RESOURCE_YAML}" | "${KUBECTL}" apply -f -

  echo_step "waiting for flow log CR to be ready & synced"
  kubectl wait --for=condition=ready flowlogs.nlb.ionoscloud.crossplane.io/examplenlb
  kubectl wait --for=condition=synced flowlogs.nlb.ionoscloud.crossplane.io/examplenlb
}

function nlbflowlog_tests_cleanup() {
  INSTALL_RESOURCE_YAML="$(
    cat <<EOF
apiVersion: nlb.ionoscloud.crossplane.io/v1alpha1
kind: FlowLog
metadata:
  name: examplenlb
spec:
  managementPolicies:
    - "*"
  forProvider:
    datacenterConfig:
      datacenterIdRef:
        name: examplenlb
    networkLoadBalancerConfig:
      networkLoadBalancerIdRef:
        name: examplenlb
    name: exampleUpdated
    action: ACCEPTED
    direction: BIDIRECTIONAL
    bucket: flowlog-acceptance-test
  providerConfigRef:
    name: example
EOF
  )"

  echo_step "uninstalling flow log CR"
  echo "${INSTALL_RESOURCE_YAML}" | "${KUBECTL}" delete -f -

  echo_step "wait for deletion flow log CR"
  kubectl wait --for=delete flowlogs.nlb.ionoscloud.crossplane.io/examplenlb
}
