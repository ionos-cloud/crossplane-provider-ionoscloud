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

```
kubectl apply -f examples/ionoscloud/compute/ipfailover.yaml
```

_Note_: The command should be run from the root of the `crossplane-provider-ionoscloud` directory.

### Update

Use the following command to update an instance. Before applying the file, update the properties defined in the `spec.forProvider` fields:

```
kubectl apply -f examples/ionoscloud/compute/ipfailover.yaml
```

_Note_: The command should be run from the root of the `crossplane-provider-ionoscloud` directory.

### Wait

Use the following commands to wait for resources to be ready and synced. Update the `<instance-name>` accordingly:

```
kubectl wait --for=condition=ready ipfailovers.compute.ionoscloud.crossplane.io/<instance-name>
kubectl wait --for=condition=synced ipfailovers.compute.ionoscloud.crossplane.io/<instance-name>
```

### Get

Use the following command to get a list of the existing instances:

```
kubectl get ipfailovers.compute.ionoscloud.crossplane.io
```

Use the following command to get a list of the existing instances with more details displayed:

```
kubectl get ipfailovers.compute.ionoscloud.crossplane.io -o wide
```

Use the following command to get a list of the existing instances in JSON format:

```
kubectl get ipfailovers.compute.ionoscloud.crossplane.io -o json
```

### Delete

Use the following command to destroy the resources created by applying the file:

```
kubectl delete -f examples/ionoscloud/compute/ipfailover.yaml
```

_Note_: The command should be run from the root of the `crossplane-provider-ionoscloud` directory.

## Properties

In order to configure the IONOS Cloud Resource, the user can set the `spec.forProvider` fields into the specification file for the resource instance. The required fields that need to be set can be found [here](#required-properties). Following, there is a list of all the properties:

* `datacenterConfig` (object)
	* description: DatacenterConfig contains information about the datacenter resource on which the resource will be created
	* properties:
		* `datacenterId` (string)
			* description: DatacenterID is the ID of the Datacenter on which the resource will be created. It needs to be provided via directly or via reference.
			* format: uuid
		* `datacenterIdRef` (object)
			* description: DatacenterIDRef references to a Datacenter to retrieve its ID
			* properties:
				* `name` (string)
					* description: Name of the referenced object.
			* required properties:
				* `name`
		* `datacenterIdSelector` (object)
			* description: DatacenterIDSelector selects reference to a Datacenter to retrieve its datacenterId
			* properties:
				* `matchControllerRef` (boolean)
					* description: MatchControllerRef ensures an object with the same controller reference as the selecting object is selected.
				* `matchLabels` (object)
					* description: MatchLabels ensures an object with matching labels is selected.
* `ipConfig` (object)
	* description: IPConfig must have a public IP for which the group is responsible for. IP can be set directly, using ipConfig.ip or via reference and index. If both ip and ipBlockConfig is set, only ip field will be considered. It is recommended to use ip field instead of ipBlockConfig field if the ipBlock contains multiple ips.
	* properties:
		* `ip` (string)
			* pattern: ^(?:(?:25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?).){3}(?:25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)$
		* `ipBlockConfig` (object)
			* description: IPBlockConfig - used by resources that need to link IPBlock via id or via reference to get one single IP.
			* properties:
				* `index` (integer)
					* description: Index is referring to the IP index retrieved from the IPBlock. Index is starting from 0.
				* `ipBlockId` (string)
					* description: NicID is the ID of the IPBlock on which the resource will be created. It needs to be provided via directly or via reference.
					* format: uuid
				* `ipBlockIdRef` (object)
					* description: IPBlockIDRef references to a IPBlock to retrieve its ID
					* properties:
						* `name` (string)
							* description: Name of the referenced object.
					* required properties:
						* `name`
				* `ipBlockIdSelector` (object)
					* description: IPBlockIDSelector selects reference to a IPBlock to retrieve its nicId
					* properties:
						* `matchControllerRef` (boolean)
							* description: MatchControllerRef ensures an object with the same controller reference as the selecting object is selected.
						* `matchLabels` (object)
							* description: MatchLabels ensures an object with matching labels is selected.
			* required properties:
				* `index`
* `lanConfig` (object)
	* description: LanConfig contains information about the lan resource on which the resource will be created
	* properties:
		* `lanIdSelector` (object)
			* description: LanIDSelector selects reference to a Lan to retrieve its lanId
			* properties:
				* `matchControllerRef` (boolean)
					* description: MatchControllerRef ensures an object with the same controller reference as the selecting object is selected.
				* `matchLabels` (object)
					* description: MatchLabels ensures an object with matching labels is selected.
		* `lanId` (string)
			* description: LanID is the ID of the Lan on which the resource will be created. It needs to be provided via directly or via reference.
		* `lanIdRef` (object)
			* description: LanIDRef references to a Lan to retrieve its ID
			* properties:
				* `name` (string)
					* description: Name of the referenced object.
			* required properties:
				* `name`
* `nicConfig` (object)
	* description: NicConfig contains information about the nic resource on which the resource will be created
	* properties:
		* `nicId` (string)
			* description: NicID is the ID of the Nic on which the resource will be created. It needs to be provided via directly or via reference.
			* format: uuid
		* `nicIdRef` (object)
			* description: NicIDRef references to a Nic to retrieve its ID
			* properties:
				* `name` (string)
					* description: Name of the referenced object.
			* required properties:
				* `name`
		* `nicIdSelector` (object)
			* description: NicIDSelector selects reference to a Nic to retrieve its nicId
			* properties:
				* `matchLabels` (object)
					* description: MatchLabels ensures an object with matching labels is selected.
				* `matchControllerRef` (boolean)
					* description: MatchControllerRef ensures an object with the same controller reference as the selecting object is selected.

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

