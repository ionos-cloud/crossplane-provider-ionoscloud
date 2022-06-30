---
description: Manages Lan Resource on IONOS Cloud.
---

# Lan Managed Resource

## Overview

* Resource Name: `Lan`
* Resource Group: `compute.ionoscloud.crossplane.io`
* Resource Version: `v1alpha1`
* Resource Scope: `Cluster`

## Usage

In order to manage resources on IONOS Cloud using Crossplane Provider, you need to have Crossplane Provider for IONOS Cloud installed into a Kubernetes Cluster, as a prerequisite. For a step-by-step guide, check the following [link](https://github.com/ionos-cloud/crossplane-provider-ionoscloud/tree/master/examples/example.md).

It is recommended to clone the repository for easier access to the example files.

### Create

Use the following command to create a resource instance. Before applying the file, check the properties defined in the `spec.forProvider` fields:

```
kubectl apply -f examples/ionoscloud/compute/lan.yaml
```

_Note_: The command should be run from the root of the `crossplane-provider-ionoscloud` directory.

### Update

Use the following command to update an instance. Before applying the file, update the properties defined in the `spec.forProvider` fields:

```
kubectl apply -f examples/ionoscloud/compute/lan.yaml
```

_Note_: The command should be run from the root of the `crossplane-provider-ionoscloud` directory.

### Wait

Use the following commands to wait for resources to be ready and synced. Update the `<instance-name>` accordingly:

```
kubectl wait --for=condition=ready lans.compute.ionoscloud.crossplane.io/<instance-name>
kubectl wait --for=condition=synced lans.compute.ionoscloud.crossplane.io/<instance-name>
```

### Get

Use the following command to get a list of the existing instances:

```
kubectl get lans.compute.ionoscloud.crossplane.io
```

Use the following command to get a list of the existing instances with more details displayed:

```
kubectl get lans.compute.ionoscloud.crossplane.io -o wide
```

Use the following command to get a list of the existing instances in JSON format:

```
kubectl get lans.compute.ionoscloud.crossplane.io -o json
```

### Delete

Use the following command to destroy the resources created by applying the file:

```
kubectl delete -f examples/ionoscloud/compute/lan.yaml
```

_Note_: The command should be run from the root of the `crossplane-provider-ionoscloud` directory.

## Properties

In order to configure the IONOS Cloud Resource, the user can set the `spec.forProvider` fields into the specification file for the resource instance. The required fields that need to be set can be found [here](#required-properties). Following, there is a list of all the properties:

* `public` (boolean)
	* description: This LAN faces the public Internet.
* `datacenterConfig` (object)
	* description: DatacenterConfig contains information about the datacenter resource on which the lan will be created
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
* `name` (string)
	* description: The name of the  resource.
* `pcc` (string)
	* description: The unique identifier of the private Cross-Connect the LAN is connected to, if any.

### Required Properties

The user needs to set the following properties in order to configure the IONOS Cloud Resource:

* `datacenterConfig`
* `public`

## Resource Definition

The corresponding resource definition can be found [here](https://github.com/ionos-cloud/crossplane-provider-ionoscloud/tree/master/package/crds/compute.ionoscloud.crossplane.io_lans.yaml).

## Resource Instance Example

An example of a resource instance can be found [here](https://github.com/ionos-cloud/crossplane-provider-ionoscloud/tree/master/examples/ionoscloud/compute/lan.yaml).

