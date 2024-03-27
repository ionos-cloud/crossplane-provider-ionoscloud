---
description: Manages NetworkLoadBalancer Resource on IONOS Cloud.
---

# NetworkLoadBalancer Managed Resource

## Overview

* Resource Name: `NetworkLoadBalancer`
* Resource Group: `nlb.ionoscloud.crossplane.io`
* Resource Version: `v1alpha1`
* Resource Scope: `Cluster`

## Usage

In order to manage resources on IONOS Cloud using Crossplane Provider, you need to have Crossplane Provider for IONOS Cloud installed into a Kubernetes Cluster, as a prerequisite. For a step-by-step guide, check the following [link](https://github.com/ionos-cloud/crossplane-provider-ionoscloud/tree/master/examples/example.md).

It is recommended to clone the repository for easier access to the example files.

### Create

Use the following command to create a resource instance. Before applying the file, check the properties defined in the `spec.forProvider` fields:

```bash
kubectl apply -f examples/ionoscloud/nlb/networkloadbalancer.yaml
```

_Note_: The command should be run from the root of the `crossplane-provider-ionoscloud` directory.

### Update

Use the following command to update an instance. Before applying the file, update the properties defined in the `spec.forProvider` fields:

```bash
kubectl apply -f examples/ionoscloud/nlb/networkloadbalancer.yaml
```

_Note_: The command should be run from the root of the `crossplane-provider-ionoscloud` directory.

### Wait

Use the following commands to wait for resources to be ready and synced. Update the `<instance-name>` accordingly:

```bash
kubectl wait --for=condition=ready networkloadbalancers.nlb.ionoscloud.crossplane.io/<instance-name>
```

```bash
kubectl wait --for=condition=synced networkloadbalancers.nlb.ionoscloud.crossplane.io/<instance-name>
```

### Get

Use the following command to get a list of the existing instances:

```bash
kubectl get -f networkloadbalancers.nlb.ionoscloud.crossplane.io
```

_Note_: Use options `--output wide`, `--output json` to get more information about the resource instances.

### Delete

Use the following command to destroy the resources created by applying the file:

```bash
kubectl delete -f examples/ionoscloud/nlb/networkloadbalancer.yaml
```

_Note_: The command should be run from the root of the `crossplane-provider-ionoscloud` directory.

## Properties

In order to configure the IONOS Cloud Resource, the user can set the `spec.forProvider` fields into the specification file for the resource instance. The required fields that need to be set can be found [here](#required-properties). Following, there is a list of all the properties:

* `datacenterConfig` (object)
	* description: A Datacenter, to which the user has access, to provision the Network Load Balancer in.
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
* `ipsConfig` (object)
	* description: Collection of the Network Load Balancer IP addresses.
(Inbound and outbound) IPs of the listenerLan are customer-reserved public IPs for
the public Load Balancers, and private IPs for the private Load Balancers.
The IPs can be set directly or using reference to the existing IPBlocks and indexes.
	* properties:
		* `ips` (array)
			* description: IPs can be used to directly specify a list of ips to the resource
		* `ipsBlocksConfig` (array)
			* description: IPBlocks can be used to reference existing IPBlocks and assign ips by indexing
			* properties:
				* `indexes` (array)
					* description: Indexes can be used to retrieve multiple ips from an IPBlock
Starting index is 0. If no index is set, the entire IP set of the block will be assigned.
				* `ipBlockConfig` (object)
					* description: IPBlock  used to reference an existing IPBlock
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
* `lbPrivateIps` (array)
	* description: Collection of private IP addresses with the subnet mask of the Network Load Balancer.
IPs must contain valid a subnet mask.
If no IP is provided, the system will generate an IP with /24 subnet.
* `listenerLanConfig` (object)
	* description: ID of the listening (inbound) LAN.
Lan ID can be set directly or via reference.
	* properties:
		* `lanId` (string)
			* description: LanID is the ID of the Lan on which the resource will be created.
It needs to be provided directly or via reference.
		* `lanIdRef` (object)
			* description: LanIDRef references to a Lan to retrieve its ID.
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
		* `lanIdSelector` (object)
			* description: LanIDSelector selects reference to a Lan to retrieve its LanID.
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
* `name` (string)
	* description: The name of the Network Load Balancer.
* `targetLanConfig` (object)
	* description: ID of the balanced private target (outbound) LAN .
Lan ID can be set directly or via reference.
	* properties:
		* `lanId` (string)
			* description: LanID is the ID of the Lan on which the resource will be created.
It needs to be provided directly or via reference.
		* `lanIdRef` (object)
			* description: LanIDRef references to a Lan to retrieve its ID.
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
		* `lanIdSelector` (object)
			* description: LanIDSelector selects reference to a Lan to retrieve its LanID.
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

### Required Properties

The user needs to set the following properties in order to configure the IONOS Cloud Resource:

* `datacenterConfig`
* `listenerLanConfig`
* `name`
* `targetLanConfig`

## Resource Definition

The corresponding resource definition can be found [here](https://github.com/ionos-cloud/crossplane-provider-ionoscloud/tree/master/package/crds/nlb.ionoscloud.crossplane.io_networkloadbalancers.yaml).

## Resource Instance Example

An example of a resource instance can be found [here](https://github.com/ionos-cloud/crossplane-provider-ionoscloud/tree/master/examples/ionoscloud/nlb/networkloadbalancer.yaml).

