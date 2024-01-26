---
description: Manages IPFailover Resource on IONOS Cloud.
---

# IPFailover Managed Resource

## Overview

* Resource Name: `IPFailover`
* Resource Group: `compute.ionoscloud.crossplane.io`
* Resource Version: `v1alpha1`
* Resource Scope: `Cluster`

## Usage

In order to manage resources on IONOS Cloud using Crossplane Provider, you need to have Crossplane Provider for IONOS Cloud installed into a Kubernetes Cluster, as a prerequisite. For a step-by-step guide, check the following [link](https://github.com/ionos-cloud/crossplane-provider-ionoscloud/tree/master/examples/example.md).

It is recommended to clone the repository for easier access to the example files.

### Create

Use the following command to create a resource instance. Before applying the file, check the properties defined in the `spec.forProvider` fields:

```bash
kubectl apply -f examples/ionoscloud/compute/ipfailover.yaml
```

_Note_: The command should be run from the root of the `crossplane-provider-ionoscloud` directory.

### Update

Use the following command to update an instance. Before applying the file, update the properties defined in the `spec.forProvider` fields:

```bash
kubectl apply -f examples/ionoscloud/compute/ipfailover.yaml
```

_Note_: The command should be run from the root of the `crossplane-provider-ionoscloud` directory.

### Wait

Use the following commands to wait for resources to be ready and synced. Update the `<instance-name>` accordingly:

```bash
kubectl wait --for=condition=ready ipfailovers.compute.ionoscloud.crossplane.io/<instance-name>
```

```bash
kubectl wait --for=condition=synced ipfailovers.compute.ionoscloud.crossplane.io/<instance-name>
```

### Get

Use the following command to get a list of the existing instances:

```bash
kubectl get -f ipfailovers.compute.ionoscloud.crossplane.io
```

_Note_: Use options `--output wide`, `--output json` to get more information about the resource instances.

### Delete

Use the following command to destroy the resources created by applying the file:

```bash
kubectl delete -f examples/ionoscloud/compute/ipfailover.yaml
```

_Note_: The command should be run from the root of the `crossplane-provider-ionoscloud` directory.

## Properties

In order to configure the IONOS Cloud Resource, the user can set the `spec.forProvider` fields into the specification file for the resource instance. The required fields that need to be set can be found [here](#required-properties). Following, there is a list of all the properties:

* `datacenterConfig` (object)
	* description: DatacenterConfig contains information about the datacenter resource on which the resource will be created.
* :
	* `datacenterId` (string)
		* description: DatacenterID is the ID of the Datacenter on which the resource will be created. It needs to be provided via directly or via reference.
		* format: uuid
	* `datacenterIdRef` (object)
		* description: DatacenterIDRef references to a Datacenter to retrieve its ID.
* :
		* `name` (string)
			* description: Name of the referenced object.
		* `policy` (object)
			* description: Policies for referencing.
* :
			* `resolution` (string)
				* description: Resolution specifies whether resolution of this reference is required. The default is 'Required', which means the reconcile will fail if the reference cannot be resolved. 'Optional' means this reference will be a no-op if it cannot be resolved.
				* default: "Required"
				* possible values: "Required";"Optional"
			* `resolve` (string)
				* description: Resolve specifies when this reference should be resolved. The default is 'IfNotPresent', which will attempt to resolve the reference only when the corresponding field is not present. Use 'Always' to resolve the reference on every reconcile.
				* possible values: "Always";"IfNotPresent"
	* `datacenterIdSelector` (object)
		* description: DatacenterIDSelector selects reference to a Datacenter to retrieve its DatacenterID.
* :
		* `matchControllerRef` (boolean)
			* description: MatchControllerRef ensures an object with the same controller reference as the selecting object is selected.
		* `matchLabels` (object)
			* description: MatchLabels ensures an object with matching labels is selected.
* :
		* `policy` (object)
			* description: Policies for selection.
* :
			* `resolution` (string)
				* description: Resolution specifies whether resolution of this reference is required. The default is 'Required', which means the reconcile will fail if the reference cannot be resolved. 'Optional' means this reference will be a no-op if it cannot be resolved.
				* default: "Required"
				* possible values: "Required";"Optional"
			* `resolve` (string)
				* description: Resolve specifies when this reference should be resolved. The default is 'IfNotPresent', which will attempt to resolve the reference only when the corresponding field is not present. Use 'Always' to resolve the reference on every reconcile.
				* possible values: "Always";"IfNotPresent"
* `ipConfig` (object)
	* description: IPConfig must have a public IP for which the group is responsible for. IP can be set directly, using ipConfig.ip or via reference and index. If both ip and ipBlockConfig is set, only ip field will be considered. It is recommended to use ip field instead of ipBlockConfig field if the ipBlock contains multiple ips.
* :
	* `ip` (string)
		* description: Use IP to set specific IP to the resource. If both IP and IPBlockConfig are set, only `ip` field will be considered.
		* pattern: ^(?:(?:25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?).){3}(?:25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)$
	* `ipBlockConfig` (object)
		* description: Use IpBlockConfig to reference existing IPBlock, and to mention the index for the IP. Index starts from 0 and it must be provided.
* :
		* `index` (integer)
			* description: Index is referring to the IP index retrieved from the IPBlock. Index is starting from 0.
		* `ipBlockId` (string)
			* description: IPBlockID is the ID of the IPBlock on which the resource will be created. It needs to be provided via directly or via reference.
			* format: uuid
		* `ipBlockIdRef` (object)
			* description: IPBlockIDRef references to a IPBlock to retrieve its ID.
* :
			* `name` (string)
				* description: Name of the referenced object.
			* `policy` (object)
				* description: Policies for referencing.
* :
				* `resolution` (string)
					* description: Resolution specifies whether resolution of this reference is required. The default is 'Required', which means the reconcile will fail if the reference cannot be resolved. 'Optional' means this reference will be a no-op if it cannot be resolved.
					* default: "Required"
					* possible values: "Required";"Optional"
				* `resolve` (string)
					* description: Resolve specifies when this reference should be resolved. The default is 'IfNotPresent', which will attempt to resolve the reference only when the corresponding field is not present. Use 'Always' to resolve the reference on every reconcile.
					* possible values: "Always";"IfNotPresent"
		* `ipBlockIdSelector` (object)
			* description: IPBlockIDSelector selects reference to a IPBlock to retrieve its IPBlockID.
* :
			* `matchControllerRef` (boolean)
				* description: MatchControllerRef ensures an object with the same controller reference as the selecting object is selected.
			* `matchLabels` (object)
				* description: MatchLabels ensures an object with matching labels is selected.
* :
			* `policy` (object)
				* description: Policies for selection.
* :
				* `resolution` (string)
					* description: Resolution specifies whether resolution of this reference is required. The default is 'Required', which means the reconcile will fail if the reference cannot be resolved. 'Optional' means this reference will be a no-op if it cannot be resolved.
					* default: "Required"
					* possible values: "Required";"Optional"
				* `resolve` (string)
					* description: Resolve specifies when this reference should be resolved. The default is 'IfNotPresent', which will attempt to resolve the reference only when the corresponding field is not present. Use 'Always' to resolve the reference on every reconcile.
					* possible values: "Always";"IfNotPresent"
* `lanConfig` (object)
	* description: LanConfig contains information about the lan resource on which the resource will be created.
* :
	* `lanId` (string)
		* description: LanID is the ID of the Lan on which the resource will be created. It needs to be provided via directly or via reference.
	* `lanIdRef` (object)
		* description: LanIDRef references to a Lan to retrieve its ID.
* :
		* `name` (string)
			* description: Name of the referenced object.
		* `policy` (object)
			* description: Policies for referencing.
* :
			* `resolution` (string)
				* description: Resolution specifies whether resolution of this reference is required. The default is 'Required', which means the reconcile will fail if the reference cannot be resolved. 'Optional' means this reference will be a no-op if it cannot be resolved.
				* default: "Required"
				* possible values: "Required";"Optional"
			* `resolve` (string)
				* description: Resolve specifies when this reference should be resolved. The default is 'IfNotPresent', which will attempt to resolve the reference only when the corresponding field is not present. Use 'Always' to resolve the reference on every reconcile.
				* possible values: "Always";"IfNotPresent"
	* `lanIdSelector` (object)
		* description: LanIDSelector selects reference to a Lan to retrieve its LanID.
* :
		* `matchControllerRef` (boolean)
			* description: MatchControllerRef ensures an object with the same controller reference as the selecting object is selected.
		* `matchLabels` (object)
			* description: MatchLabels ensures an object with matching labels is selected.
* :
		* `policy` (object)
			* description: Policies for selection.
* :
			* `resolution` (string)
				* description: Resolution specifies whether resolution of this reference is required. The default is 'Required', which means the reconcile will fail if the reference cannot be resolved. 'Optional' means this reference will be a no-op if it cannot be resolved.
				* default: "Required"
				* possible values: "Required";"Optional"
			* `resolve` (string)
				* description: Resolve specifies when this reference should be resolved. The default is 'IfNotPresent', which will attempt to resolve the reference only when the corresponding field is not present. Use 'Always' to resolve the reference on every reconcile.
				* possible values: "Always";"IfNotPresent"
* `nicConfig` (object)
	* description: NicConfig contains information about the nic resource on which the resource will be created.
* :
	* `nicId` (string)
		* description: NicID is the ID of the Nic on which the resource will be created. It needs to be provided via directly or via reference.
		* format: uuid
	* `nicIdRef` (object)
		* description: NicIDRef references to a Nic to retrieve its ID.
* :
		* `name` (string)
			* description: Name of the referenced object.
		* `policy` (object)
			* description: Policies for referencing.
* :
			* `resolution` (string)
				* description: Resolution specifies whether resolution of this reference is required. The default is 'Required', which means the reconcile will fail if the reference cannot be resolved. 'Optional' means this reference will be a no-op if it cannot be resolved.
				* default: "Required"
				* possible values: "Required";"Optional"
			* `resolve` (string)
				* description: Resolve specifies when this reference should be resolved. The default is 'IfNotPresent', which will attempt to resolve the reference only when the corresponding field is not present. Use 'Always' to resolve the reference on every reconcile.
				* possible values: "Always";"IfNotPresent"
	* `nicIdSelector` (object)
		* description: NicIDSelector selects reference to a Nic to retrieve its NicID.
* :
		* `matchControllerRef` (boolean)
			* description: MatchControllerRef ensures an object with the same controller reference as the selecting object is selected.
		* `matchLabels` (object)
			* description: MatchLabels ensures an object with matching labels is selected.
* :
		* `policy` (object)
			* description: Policies for selection.
* :
			* `resolution` (string)
				* description: Resolution specifies whether resolution of this reference is required. The default is 'Required', which means the reconcile will fail if the reference cannot be resolved. 'Optional' means this reference will be a no-op if it cannot be resolved.
				* default: "Required"
				* possible values: "Required";"Optional"
			* `resolve` (string)
				* description: Resolve specifies when this reference should be resolved. The default is 'IfNotPresent', which will attempt to resolve the reference only when the corresponding field is not present. Use 'Always' to resolve the reference on every reconcile.
				* possible values: "Always";"IfNotPresent"

### Required Properties

The user needs to set the following properties in order to configure the IONOS Cloud Resource:

* `datacenterConfig`
* `ipConfig`
* `lanConfig`
* `nicConfig`

## Resource Definition

The corresponding resource definition can be found [here](https://github.com/ionos-cloud/crossplane-provider-ionoscloud/tree/master/package/crds/compute.ionoscloud.crossplane.io_ipfailovers.yaml).

## Resource Instance Example

An example of a resource instance can be found [here](https://github.com/ionos-cloud/crossplane-provider-ionoscloud/tree/master/examples/ionoscloud/compute/ipfailover.yaml).

