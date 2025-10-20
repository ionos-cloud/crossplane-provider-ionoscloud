---
description: Manages ForwardingRule Resource on IONOS Cloud.
---

# ForwardingRule Managed Resource

## Overview

* Description: An ForwardingRule is an example API type.
* Resource Name: `ForwardingRule`
* Resource Group: `nlb.ionoscloud.crossplane.io`
* Resource Version: `v1alpha1`
* Resource Scope: `Cluster`

## Usage

In order to manage resources on IONOS Cloud using Crossplane Provider, you need to have Crossplane Provider for IONOS Cloud installed into a Kubernetes Cluster, as a prerequisite. For a step-by-step guide, check the following [link](https://github.com/ionos-cloud/crossplane-provider-ionoscloud/tree/master/examples/example.md).

It is recommended to clone the repository for easier access to the example files.

### Create

Use the following command to create a resource instance. Before applying the file, check the properties defined in the `spec.forProvider` fields:

```bash
kubectl apply -f examples/ionoscloud/nlb/forwardingrule.yaml
```

_Note_: The command should be run from the root of the `crossplane-provider-ionoscloud` directory.

### Update

Use the following command to update an instance. Before applying the file, update the properties defined in the `spec.forProvider` fields:

```bash
kubectl apply -f examples/ionoscloud/nlb/forwardingrule.yaml
```

_Note_: The command should be run from the root of the `crossplane-provider-ionoscloud` directory.

### Wait

Use the following commands to wait for resources to be ready and synced. Update the `<instance-name>` accordingly:

```bash
kubectl wait --for=condition=ready forwardingrules.nlb.ionoscloud.crossplane.io/<instance-name>
```

```bash
kubectl wait --for=condition=synced forwardingrules.nlb.ionoscloud.crossplane.io/<instance-name>
```

### Get

Use the following command to get a list of the existing instances:

```bash
kubectl get -f forwardingrules.nlb.ionoscloud.crossplane.io
```

_Note_: Use options `--output wide`, `--output json` to get more information about the resource instances.

### Delete

Use the following command to destroy the resources created by applying the file:

```bash
kubectl delete -f examples/ionoscloud/nlb/forwardingrule.yaml
```

_Note_: The command should be run from the root of the `crossplane-provider-ionoscloud` directory.

## Properties

In order to configure the IONOS Cloud Resource, the user can set the `spec.forProvider` fields into the specification file for the resource instance. The required fields that need to be set can be found [here](#required-properties). Following, there is a list of all the properties:

* `algorithm` (string)
	* description: Algorithm used in load balancing
	* possible values: "ROUND_ROBIN";"LEAST_CONNECTION";"RANDOM";"SOURCE_IP"
* `datacenterConfig` (object)
	* description: Datacenter in which the Network Load Balancer that this Forwarding Rule applies to is provisioned in.
	* properties:
		* `datacenterId` (string)
			* description: DatacenterID is the ID of the Datacenter on which the resource should have access.
			  It needs to be provided directly or via reference.
			* format: uuid
		* `datacenterIdRef` (object)
			* description: DatacenterIDRef references to a Datacenter to retrieve its ID.
			* properties:
				* `name` (string)
					* description: Name of the referenced object.
				* `policy` (object)
					* description: Policies for referencing.
					* properties:
						* `resolution` (string)
							* description: Resolution specifies whether resolution of this reference is required.
							  The default is 'Required', which means the reconcile will fail if the
							  reference cannot be resolved. 'Optional' means this reference will be
							  a no-op if it cannot be resolved.
							* default: "Required"
							* possible values: "Required";"Optional"
						* `resolve` (string)
							* description: Resolve specifies when this reference should be resolved. The default
							  is 'IfNotPresent', which will attempt to resolve the reference only when
							  the corresponding field is not present. Use 'Always' to resolve the
							  reference on every reconcile.
							* possible values: "Always";"IfNotPresent"
			* required properties:
				* `name`
		* `datacenterIdSelector` (object)
			* description: DatacenterIDSelector selects reference to a Datacenter to retrieve its DatacenterID.
			* properties:
				* `matchControllerRef` (boolean)
					* description: MatchControllerRef ensures an object with the same controller reference
					  as the selecting object is selected.
				* `matchLabels` (object)
					* description: MatchLabels ensures an object with matching labels is selected.
				* `policy` (object)
					* description: Policies for selection.
					* properties:
						* `resolution` (string)
							* description: Resolution specifies whether resolution of this reference is required.
							  The default is 'Required', which means the reconcile will fail if the
							  reference cannot be resolved. 'Optional' means this reference will be
							  a no-op if it cannot be resolved.
							* default: "Required"
							* possible values: "Required";"Optional"
						* `resolve` (string)
							* description: Resolve specifies when this reference should be resolved. The default
							  is 'IfNotPresent', which will attempt to resolve the reference only when
							  the corresponding field is not present. Use 'Always' to resolve the
							  reference on every reconcile.
							* possible values: "Always";"IfNotPresent"
* `healthCheck` (object)
	* description: HealthCheck options for the forwarding rule health check
	* default: {}
	* properties:
		* `clientTimeout` (integer)
			* description: ClientTimeout the maximum time in milliseconds to wait for the client to acknowledge or send data; default is 50,000 (50 seconds).
			* default: 50000
			* format: int32
		* `connectTimeout` (integer)
			* description: ConnectTimeout the maximum time in milliseconds to wait for a connection attempt to a target to succeed; default is 5000 (five seconds).
			* default: 5000
			* format: int32
		* `retries` (integer)
			* description: Retries the maximum number of attempts to reconnect to a target after a connection failure. Valid range is 0 to 65535 and default is three reconnection attempts.
			* default: 3
			* format: int32
		* `targetTimeout` (integer)
			* description: TargetTimeout the maximum time in milliseconds that a target can remain inactive; default is 50,000 (50 seconds).
			* default: 50000
			* format: int32
* `listenerIpConfig` (object)
	* description: Listening (inbound) IP. IP must be assigned to the listener NIC of the Network Load Balancer.
	* properties:
		* `index` (integer)
			* description: Index can be used to retrieve an ip from the referenced IPBlock
			  Starting index is 0.
		* `ip` (string)
			* description: IP can be used to directly specify a single ip to the resource
			* pattern: ^(?:(?:25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?).){3}(?:25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)$
		* `ipBlock` (object)
			* description: IPBlockConfig can be used to reference an existing IPBlock and assign an ip by indexing
			  For Network Load Balancer Forwarding Rules, only a single index can be specified
			* properties:
				* `ipBlockId` (string)
					* description: IPBlockID is the ID of the IPBlock on which the resource will be created.
					  It needs to be provided directly or via reference.
					* format: uuid
				* `ipBlockIdRef` (object)
					* description: IPBlockIDRef references to a IPBlock to retrieve its ID.
					* properties:
						* `name` (string)
							* description: Name of the referenced object.
						* `policy` (object)
							* description: Policies for referencing.
							* properties:
								* `resolution` (string)
									* description: Resolution specifies whether resolution of this reference is required.
									  The default is 'Required', which means the reconcile will fail if the
									  reference cannot be resolved. 'Optional' means this reference will be
									  a no-op if it cannot be resolved.
									* default: "Required"
									* possible values: "Required";"Optional"
								* `resolve` (string)
									* description: Resolve specifies when this reference should be resolved. The default
									  is 'IfNotPresent', which will attempt to resolve the reference only when
									  the corresponding field is not present. Use 'Always' to resolve the
									  reference on every reconcile.
									* possible values: "Always";"IfNotPresent"
					* required properties:
						* `name`
				* `ipBlockIdSelector` (object)
					* description: IPBlockIDSelector selects reference to a IPBlock to retrieve its IPBlockID.
					* properties:
						* `matchControllerRef` (boolean)
							* description: MatchControllerRef ensures an object with the same controller reference
							  as the selecting object is selected.
						* `matchLabels` (object)
							* description: MatchLabels ensures an object with matching labels is selected.
						* `policy` (object)
							* description: Policies for selection.
							* properties:
								* `resolution` (string)
									* description: Resolution specifies whether resolution of this reference is required.
									  The default is 'Required', which means the reconcile will fail if the
									  reference cannot be resolved. 'Optional' means this reference will be
									  a no-op if it cannot be resolved.
									* default: "Required"
									* possible values: "Required";"Optional"
								* `resolve` (string)
									* description: Resolve specifies when this reference should be resolved. The default
									  is 'IfNotPresent', which will attempt to resolve the reference only when
									  the corresponding field is not present. Use 'Always' to resolve the
									  reference on every reconcile.
									* possible values: "Always";"IfNotPresent"
* `listenerPort` (integer)
	* description: Listening (inbound) port number; valid range is 1 to 65535.
	* format: int32
	* minimum: 1.000000
	* maximum: 65535.000000
* `name` (string)
	* description: The name of the Network Load Balancer Forwarding Rule.
* `networkLoadBalancerConfig` (object)
	* description: NetworkLoadBalancer to which this Forwarding Rule will apply.
	* properties:
		* `networkLoadBalancerId` (string)
			* description: NetworkLoadBalancerID is the ID of the NetworkLoadBalancer on which the resource should have access.
			  It needs to be provided directly or via reference.
			* format: uuid
		* `networkLoadBalancerIdRef` (object)
			* description: NetworkLoadBalancerIDRef references to a Datacenter to retrieve its ID.
			* properties:
				* `name` (string)
					* description: Name of the referenced object.
				* `policy` (object)
					* description: Policies for referencing.
					* properties:
						* `resolution` (string)
							* description: Resolution specifies whether resolution of this reference is required.
							  The default is 'Required', which means the reconcile will fail if the
							  reference cannot be resolved. 'Optional' means this reference will be
							  a no-op if it cannot be resolved.
							* default: "Required"
							* possible values: "Required";"Optional"
						* `resolve` (string)
							* description: Resolve specifies when this reference should be resolved. The default
							  is 'IfNotPresent', which will attempt to resolve the reference only when
							  the corresponding field is not present. Use 'Always' to resolve the
							  reference on every reconcile.
							* possible values: "Always";"IfNotPresent"
			* required properties:
				* `name`
		* `networkLoadBalancerIdSelector` (object)
			* description: NetworkLoadBalancerIDSelector selects reference to a Datacenter to retrieve its DatacenterID.
			* properties:
				* `matchControllerRef` (boolean)
					* description: MatchControllerRef ensures an object with the same controller reference
					  as the selecting object is selected.
				* `matchLabels` (object)
					* description: MatchLabels ensures an object with matching labels is selected.
				* `policy` (object)
					* description: Policies for selection.
					* properties:
						* `resolution` (string)
							* description: Resolution specifies whether resolution of this reference is required.
							  The default is 'Required', which means the reconcile will fail if the
							  reference cannot be resolved. 'Optional' means this reference will be
							  a no-op if it cannot be resolved.
							* default: "Required"
							* possible values: "Required";"Optional"
						* `resolve` (string)
							* description: Resolve specifies when this reference should be resolved. The default
							  is 'IfNotPresent', which will attempt to resolve the reference only when
							  the corresponding field is not present. Use 'Always' to resolve the
							  reference on every reconcile.
							* possible values: "Always";"IfNotPresent"
* `protocol` (string)
	* description: Balancing protocol
	* possible values: "TCP";"HTTP"
* `targets` (array)
	* description: Targets is the list of load balanced targets
	* properties:
		* `healthCheck` (object)
			* description: HealthCheck options of the balanced target health check
			* default: {}
			* properties:
				* `check` (boolean)
					* description: Check makes the target available only if it accepts periodic health check TCP connection attempts.
					  When turned off, the target is considered always available.
					  The health check only consists of a connection attempt to the address and port of the target.
					* default: true
				* `checkInterval` (integer)
					* description: CheckInterval the interval in milliseconds between consecutive health checks; default is 2000.
					* default: 2000
					* format: int32
				* `maintenance` (boolean)
					* description: Maintenance mode prevents the target from receiving balanced traffic.
		* `ipConfig` (object)
			* description: IP of the balanced target
			* properties:
				* `index` (integer)
					* description: Index can be used to retrieve an ip from the referenced IPBlock
					  Starting index is 0.
				* `ip` (string)
					* description: IP can be used to directly specify a single ip to the resource
					* pattern: ^(?:(?:25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?).){3}(?:25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)$
				* `ipBlock` (object)
					* description: IPBlockConfig can be used to reference an existing IPBlock and assign an ip by indexing
					  For Network Load Balancer Forwarding Rules, only a single index can be specified
					* properties:
						* `ipBlockId` (string)
							* description: IPBlockID is the ID of the IPBlock on which the resource will be created.
							  It needs to be provided directly or via reference.
							* format: uuid
						* `ipBlockIdRef` (object)
							* description: IPBlockIDRef references to a IPBlock to retrieve its ID.
							* properties:
								* `name` (string)
									* description: Name of the referenced object.
								* `policy` (object)
									* description: Policies for referencing.
									* properties:
										* `resolution` (string)
											* description: Resolution specifies whether resolution of this reference is required.
											  The default is 'Required', which means the reconcile will fail if the
											  reference cannot be resolved. 'Optional' means this reference will be
											  a no-op if it cannot be resolved.
											* default: "Required"
											* possible values: "Required";"Optional"
										* `resolve` (string)
											* description: Resolve specifies when this reference should be resolved. The default
											  is 'IfNotPresent', which will attempt to resolve the reference only when
											  the corresponding field is not present. Use 'Always' to resolve the
											  reference on every reconcile.
											* possible values: "Always";"IfNotPresent"
							* required properties:
								* `name`
						* `ipBlockIdSelector` (object)
							* description: IPBlockIDSelector selects reference to a IPBlock to retrieve its IPBlockID.
							* properties:
								* `matchControllerRef` (boolean)
									* description: MatchControllerRef ensures an object with the same controller reference
									  as the selecting object is selected.
								* `matchLabels` (object)
									* description: MatchLabels ensures an object with matching labels is selected.
								* `policy` (object)
									* description: Policies for selection.
									* properties:
										* `resolution` (string)
											* description: Resolution specifies whether resolution of this reference is required.
											  The default is 'Required', which means the reconcile will fail if the
											  reference cannot be resolved. 'Optional' means this reference will be
											  a no-op if it cannot be resolved.
											* default: "Required"
											* possible values: "Required";"Optional"
										* `resolve` (string)
											* description: Resolve specifies when this reference should be resolved. The default
											  is 'IfNotPresent', which will attempt to resolve the reference only when
											  the corresponding field is not present. Use 'Always' to resolve the
											  reference on every reconcile.
											* possible values: "Always";"IfNotPresent"
		* `port` (integer)
			* description: Port of the balanced target
			* format: int32
		* `proxyProtocol` (string)
			* description: ProxyProtocol version of the proxy protocol
			* default: "none"
			* possible values: "none";"v1";"v2";"v2ssl"
		* `weight` (integer)
			* description: Weight of the balanced target Traffic is distributed in proportion to target weight, relative to the combined weight of all targets.
			  A target with higher weight receives a greater share of traffic. Valid range is 0 to 256 and default is 1.
			  Targets with weight of 0 do not participate in load balancing but still accept persistent connections.
			  It is best to assign weights in the middle of the range to leave room for later adjustments.
			* format: int32
	* required properties:
		* `ipConfig`
		* `port`
		* `weight`

### Required Properties

The user needs to set the following properties in order to configure the IONOS Cloud Resource:

* `algorithm`
* `datacenterConfig`
* `listenerIpConfig`
* `listenerPort`
* `name`
* `networkLoadBalancerConfig`
* `protocol`

## Resource Definition

The corresponding resource definition can be found [here](https://github.com/ionos-cloud/crossplane-provider-ionoscloud/tree/master/package/crds/nlb.ionoscloud.crossplane.io_forwardingrules.yaml).

## Resource Instance Example

An example of a resource instance can be found [here](https://github.com/ionos-cloud/crossplane-provider-ionoscloud/tree/master/examples/ionoscloud/nlb/forwardingrule.yaml).

