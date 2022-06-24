#!/usr/bin/env bash

set -e

## The purpose of this script is to have the tests for
## the ApplicationLoadBalancer resources
## Please name the functions the following format:
## <resource_name>_tests() and <resource_name>_tests_cleanup().

## TargetGroup CR Tests
function targetgroup_tests() {
  echo_step "deploy a target group CR"
  INSTALL_RESOURCE_YAML="$(
    cat <<EOF
apiVersion: alb.ionoscloud.crossplane.io/v1alpha1
kind: TargetGroup
metadata:
  name: example
spec:
  forProvider:
    name: exampleTargetGroup
    algorithm: ROUND_ROBIN
    protocol: HTTP
    targets:
      - ip: 10.0.2.19
        port: 80
        weight: 1
      - ip: 10.0.2.20
        port: 80
        weight: 1
  providerConfigRef:
    name: example
EOF
  )"

  echo "${INSTALL_RESOURCE_YAML}" | "${KUBECTL}" apply -f -

  echo_step "waiting for target group CR to be ready & synced"
  kubectl wait --for=condition=ready targetgroups.alb.ionoscloud.crossplane.io/example
  kubectl wait --for=condition=synced targetgroups.alb.ionoscloud.crossplane.io/example

  echo_step "get target group CR"
  kubectl get targetgroups.alb.ionoscloud.crossplane.io

  echo_step "update target group CR"
  INSTALL_RESOURCE_YAML="$(
    cat <<EOF
apiVersion: compute.ionoscloud.crossplane.io/v1alpha1
apiVersion: alb.ionoscloud.crossplane.io/v1alpha1
kind: TargetGroup
metadata:
  name: example
spec:
  forProvider:
    name: exampleTargetGroupUpdated
    algorithm: ROUND_ROBIN
    protocol: HTTP
    targets:
      - ip: 10.0.2.19
        port: 80
        weight: 1
  providerConfigRef:
    name: example
EOF
  )"

  echo "${INSTALL_RESOURCE_YAML}" | "${KUBECTL}" apply -f -

  echo_step "waiting for target group CR to be ready & synced"
  kubectl wait --for=condition=ready targetgroups.alb.ionoscloud.crossplane.io/example
  kubectl wait --for=condition=synced targetgroups.alb.ionoscloud.crossplane.io/example
}

function targetgroup_tests_cleanup() {
  INSTALL_RESOURCE_YAML="$(
    cat <<EOF
apiVersion: alb.ionoscloud.crossplane.io/v1alpha1
kind: TargetGroup
metadata:
  name: example
spec:
  forProvider:
    name: exampleTargetGroupUpdated
    algorithm: ROUND_ROBIN
    protocol: HTTP
    targets:
      - ip: 10.0.2.19
        port: 80
        weight: 1
  providerConfigRef:
    name: example
EOF
  )"

  echo_step "uninstalling target group CR"
  echo "${INSTALL_RESOURCE_YAML}" | "${KUBECTL}" delete -f -

  echo_step "wait for deletion target group CR"
  kubectl wait --for=delete targetgroups.alb.ionoscloud.crossplane.io/example
}

## ApplicationLoadBalancer CR Tests
function alb_tests() {
  echo_step "deploy a application load balancer CR"
  INSTALL_RESOURCE_YAML="$(
    cat <<EOF
apiVersion: compute.ionoscloud.crossplane.io/v1alpha1
kind: IPBlock
metadata:
  name: example
spec:
  forProvider:
    name: exampleIpBlock
    size: 2
    location: de/txl
  providerConfigRef:
    name: example
---
apiVersion: compute.ionoscloud.crossplane.io/v1alpha1
kind: Datacenter
metadata:
  name: example
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
  name: examplelan
spec:
  forProvider:
    name: exampleLan
    public: true
    datacenterConfig:
      datacenterIdRef:
        name: example
  providerConfigRef:
    name: example
---
apiVersion: compute.ionoscloud.crossplane.io/v1alpha1
kind: Lan
metadata:
  name: example
spec:
  forProvider:
    name: exampleLan
    public: false
    datacenterConfig:
      datacenterIdRef:
        name: example
  providerConfigRef:
    name: example
---
apiVersion: alb.ionoscloud.crossplane.io/v1alpha1
kind: ApplicationLoadBalancer
metadata:
  name: example
spec:
  forProvider:
    datacenterConfig:
      datacenterIdRef:
        name: example
    name: exampleApplicationLoadBalancer
    targetLanConfig:
      lanIdRef:
        name: example
    listenerLanConfig:
      lanIdRef:
        name: examplelan
    ipsConfig:
      ipsBlockConfigs:
        - ipBlockIdRef:
            name: example
          indexes: [ 0 ]
  providerConfigRef:
    name: example
EOF
  )"

  echo "${INSTALL_RESOURCE_YAML}" | "${KUBECTL}" apply -f -

  echo_step "waiting for prerequisities to be ready & synced"
  kubectl wait --for=condition=ready ipblocks.compute.ionoscloud.crossplane.io/example datacenters.compute.ionoscloud.crossplane.io/example lans.compute.ionoscloud.crossplane.io/example lans.compute.ionoscloud.crossplane.io/examplelan
  kubectl wait --for=condition=synced ipblocks.compute.ionoscloud.crossplane.io/example datacenters.compute.ionoscloud.crossplane.io/example lans.compute.ionoscloud.crossplane.io/example lans.compute.ionoscloud.crossplane.io/examplelan

  echo_step "waiting for application load balancer CR to be ready & synced"
  kubectl wait --for=condition=ready applicationloadbalancers.alb.ionoscloud.crossplane.io/example --timeout=30m
  kubectl wait --for=condition=synced applicationloadbalancers.alb.ionoscloud.crossplane.io/example --timeout=30m

  echo_step "get application load balancer CR"
  kubectl get applicationloadbalancers.alb.ionoscloud.crossplane.io -o wide

  echo_step "update application load balancer CR"
  INSTALL_RESOURCE_YAML="$(
    cat <<EOF
apiVersion: alb.ionoscloud.crossplane.io/v1alpha1
kind: ApplicationLoadBalancer
metadata:
  name: example
spec:
  forProvider:
    datacenterConfig:
      datacenterIdRef:
        name: example
    name: exampleApplicationLoadBalancerUpdated
    targetLanConfig:
      lanIdRef:
        name: example
    listenerLanConfig:
      lanIdRef:
        name: examplelan
    ipsConfig:
      ipsBlockConfigs:
        - ipBlockIdRef:
            name: example
          indexes: [ 0 ]
  providerConfigRef:
    name: example
EOF
  )"

  echo "${INSTALL_RESOURCE_YAML}" | "${KUBECTL}" apply -f -

  echo_step "waiting for application load balancer CR to be ready & synced"
  kubectl wait --for=condition=ready applicationloadbalancers.alb.ionoscloud.crossplane.io/example --timeout=30m
  kubectl wait --for=condition=synced applicationloadbalancers.alb.ionoscloud.crossplane.io/example --timeout=30m

  echo_step "get updated application load balancer CR"
  kubectl get applicationloadbalancers.alb.ionoscloud.crossplane.io -o wide
}

function alb_tests_cleanup() {
  INSTALL_RESOURCE_YAML="$(
    cat <<EOF
apiVersion: alb.ionoscloud.crossplane.io/v1alpha1
kind: ApplicationLoadBalancer
metadata:
  name: example
spec:
  forProvider:
    datacenterConfig:
      datacenterIdRef:
        name: example
    name: exampleApplicationLoadBalancer
    targetLanConfig:
      lanIdRef:
        name: example2
    listenerLanConfig:
      lanIdRef:
        name: example1
    ipsConfig:
      ipsBlockConfigs:
        - ipBlockIdRef:
            name: example
          indexes: [ 0 ]
  providerConfigRef:
    name: example
EOF
  )"

  echo_step "uninstalling application load balancer CR"
  echo "${INSTALL_RESOURCE_YAML}" | "${KUBECTL}" delete -f -

  echo_step "wait for deletion application load balancer CR"
  kubectl wait --for=delete applicationloadbalancers.alb.ionoscloud.crossplane.io/example --timeout=30m

  INSTALL_RESOURCE_YAML="$(
    cat <<EOF
apiVersion: compute.ionoscloud.crossplane.io/v1alpha1
kind: Lan
metadata:
  name: examplelan
spec:
  forProvider:
    name: exampleLan
    public: true
    datacenterConfig:
      datacenterIdRef:
        name: example
  providerConfigRef:
    name: example
---
apiVersion: compute.ionoscloud.crossplane.io/v1alpha1
kind: Lan
metadata:
  name: example
spec:
  forProvider:
    name: exampleLan
    public: false
    datacenterConfig:
      datacenterIdRef:
        name: example
  providerConfigRef:
    name: example
---
apiVersion: compute.ionoscloud.crossplane.io/v1alpha1
kind: Datacenter
metadata:
  name: example
spec:
  forProvider:
    name: exampleDatacenter
    location: de/txl
    description: test
  providerConfigRef:
    name: example
---
apiVersion: compute.ionoscloud.crossplane.io/v1alpha1
kind: IPBlock
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

  echo_step "uninstalling prerequisites for application load balancer CR"
  echo "${INSTALL_RESOURCE_YAML}" | "${KUBECTL}" delete -f -

  echo_step "wait for deletion of prerequisites for application load balancer CR"
  kubectl wait --for=delete lans.compute.ionoscloud.crossplane.io/example lans.compute.ionoscloud.crossplane.io/examplelan
  kubectl wait --for=delete datacenters.compute.ionoscloud.crossplane.io/example
  kubectl wait --for=delete ipblocks.compute.ionoscloud.crossplane.io/example
}

## ForwardingRule CR Tests
function forwardingrule_tests() {
  echo_step "deploy a forwarding rule CR"
  INSTALL_RESOURCE_YAML="$(
    cat <<EOF
apiVersion: alb.ionoscloud.crossplane.io/v1alpha1
kind: ForwardingRule
metadata:
  name: example
spec:
  forProvider:
    datacenterConfig:
      datacenterIdRef:
        name: example
    applicationLoadBalancerConfig:
      applicationLoadBalancerIdRef:
        name: example
    name: exampleForwardingRuleALB
    protocol: HTTP
    listenerIpConfig:
      ipBlockConfig:
        ipBlockIdRef:
          name: example
        index: 0
    listenerPort: 80
    httpRules:
      - name: exampleForwardHTTPRuleALB
        type: FORWARD
        targetGroupConfig:
          targetGroupIdRef:
            name: example
        conditions:
          - type: QUERY
            condition: ENDS_WITH
            negate: true
            key: goto
            value: onos
      - name: exampleRedirectHTTPRuleALB
        type: REDIRECT
        dropQuery: true
        location: "https://ionos.com"
        statusCode: 301
        conditions:
          - type: QUERY
            condition: ENDS_WITH
            negate: false
            key: goto
            value: onos
      - name: exampleStaticHTTPRuleALB
        type: STATIC
        responseMessage: "IONOS CLOUD"
        contentType: "text/html"
        statusCode: 503
        conditions:
          - type: PATH
            condition: CONTAINS
            negate: false
            value: "example"
  providerConfigRef:
    name: example
EOF
  )"

  echo "${INSTALL_RESOURCE_YAML}" | "${KUBECTL}" apply -f -

  echo_step "waiting for forwarding rule CR to be ready & synced"
  kubectl wait --for=condition=ready forwardingrules.alb.ionoscloud.crossplane.io/example --timeout=30m
  kubectl wait --for=condition=synced forwardingrules.alb.ionoscloud.crossplane.io/example --timeout=30m

  echo_step "get forwarding rule CR"
  kubectl get forwardingrules.alb.ionoscloud.crossplane.io

  echo_step "update forwarding rule CR"
  INSTALL_RESOURCE_YAML="$(
    cat <<EOF
apiVersion: alb.ionoscloud.crossplane.io/v1alpha1
kind: ForwardingRule
metadata:
  name: example
spec:
  forProvider:
    datacenterConfig:
      datacenterIdRef:
        name: example
    applicationLoadBalancerConfig:
      applicationLoadBalancerIdRef:
        name: example
    name: exampleForwardingRuleALB
    protocol: HTTP
    listenerIpConfig:
      ipBlockConfig:
        ipBlockIdRef:
          name: example
        index: 0
    listenerPort: 80
    httpRules:
      - name: exampleRedirectHTTPRuleALB
        type: REDIRECT
        dropQuery: true
        location: "https://ionos.com"
        statusCode: 301
        conditions:
          - type: QUERY
            condition: ENDS_WITH
            negate: false
            key: goto
            value: onos
  providerConfigRef:
    name: example
EOF
  )"

  echo "${INSTALL_RESOURCE_YAML}" | "${KUBECTL}" apply -f -

  echo_step "waiting for forwarding rule CR to be ready & synced"
  kubectl wait --for=condition=ready forwardingrules.alb.ionoscloud.crossplane.io/example --timeout=30m
  kubectl wait --for=condition=synced forwardingrules.alb.ionoscloud.crossplane.io/example --timeout=30m
}

function forwardingrule_tests_cleanup() {
  INSTALL_RESOURCE_YAML="$(
    cat <<EOF
apiVersion: alb.ionoscloud.crossplane.io/v1alpha1
kind: ForwardingRule
metadata:
  name: example
spec:
  forProvider:
    datacenterConfig:
      datacenterIdRef:
        name: example
    applicationLoadBalancerConfig:
      applicationLoadBalancerIdRef:
        name: example
    name: exampleForwardingRuleALB
    protocol: HTTP
    listenerIpConfig:
      ipBlockConfig:
        ipBlockIdRef:
          name: example
        index: 0
    listenerPort: 80
    httpRules:
      - name: exampleRedirectHTTPRuleALB
        type: REDIRECT
        dropQuery: true
        location: "https://ionos.com"
        statusCode: 301
        conditions:
          - type: QUERY
            condition: ENDS_WITH
            negate: false
            key: goto
            value: onos
  providerConfigRef:
    name: example
EOF
  )"

  echo_step "uninstalling forwarding rule CR"
  echo "${INSTALL_RESOURCE_YAML}" | "${KUBECTL}" delete -f -

  echo_step "wait for deletion forwarding rule CR"
  kubectl wait --for=delete forwardingrules.alb.ionoscloud.crossplane.io/example --timeout=30m
}
