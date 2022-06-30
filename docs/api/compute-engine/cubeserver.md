---
description: Manages CubeServer Resource on IONOS Cloud.
---

# CubeServer Managed Resource

## Overview

* Resource Name: `CubeServer`
* Resource Group: `compute.ionoscloud.crossplane.io`
* Resource Version: `v1alpha1`
* Resource Scope: `Cluster`

## Usage

In order to manage resources on IONOS Cloud using Crossplane Provider, you need to have Crossplane Provider for IONOS Cloud installed into a Kubernetes Cluster, as a prerequisite. For a step-by-step guide, check the following [link](https://github.com/ionos-cloud/crossplane-provider-ionoscloud/tree/master/examples/example.md).

It is recommended to clone the repository for easier access to the example files.

### Create

Use the following command to create a resource instance. Before applying the file, check the properties defined in the `spec.forProvider` fields:

```bash
kubectl apply -f examples/ionoscloud/compute/cubeserver.yaml
```

_Note_: The command should be run from the root of the `crossplane-provider-ionoscloud` directory.

### Update

Use the following command to update an instance. Before applying the file, update the properties defined in the `spec.forProvider` fields:

```bash
kubectl apply -f examples/ionoscloud/compute/cubeserver.yaml
```

_Note_: The command should be run from the root of the `crossplane-provider-ionoscloud` directory.

### Wait

Use the following commands to wait for resources to be ready and synced. Update the `<instance-name>` accordingly:

```bash
kubectl wait --for=condition=ready cubeservers.compute.ionoscloud.crossplane.io/<instance-name>
```

```bash
kubectl wait --for=condition=synced cubeservers.compute.ionoscloud.crossplane.io/<instance-name>
```

### Get

Use the following command to get a list of the existing instances:

```bash
kubectl get -f cubeservers.compute.ionoscloud.crossplane.io
```

_Note_: Use options `--output wide`, `--output json` to get more information about the resource instances.

### Delete

Use the following command to destroy the resources created by applying the file:

```bash
kubectl delete -f examples/ionoscloud/compute/cubeserver.yaml
```

_Note_: The command should be run from the root of the `crossplane-provider-ionoscloud` directory.

## Properties

In order to configure the IONOS Cloud Resource, the user can set the `spec.forProvider` fields into the specification file for the resource instance. The required fields that need to be set can be found [here](#required-properties). Following, there is a list of all the properties:

* `name` (string)
	* description: The name of the  resource.
* `template` (object)
	* description: The ID or the name of the template for creating a CUBE server.
	* properties:
		* `name` (string)
			* description: The name of the  resource.
		* `templateId` (string)
			* description: The ID of the  template.
			* format: uuid
* `volume` (object)
	* description: DasVolumeProperties contains properties for the DAS volume attached to the Cube Server.
	* properties:
		* `licenceType` (string)
			* description: OS type for this volume. Note: when creating a volume - set image, image alias, or licence type.
			* possible values: "UNKNOWN";"WINDOWS";"WINDOWS2016";"WINDOWS2022";"LINUX";"OTHER"
		* `name` (string)
			* description: The name of the DAS Volume.
		* `sshKeys` (array)
			* description: Public SSH keys are set on the image as authorized keys for appropriate SSH login to the instance using the corresponding private key. This field may only be set in creation requests. When reading, it always returns null. SSH keys are only supported if a public Linux image is used for the volume creation.
		* `bus` (string)
			* description: The bus type of the volume.
			* possible values: "VIRTIO";"IDE";"UNKNOWN"
		* `image` (string)
			* description: Image or snapshot ID to be used as template for this volume. Make sure the image selected is compatible with the datacenter's location. Note: when creating a volume - set image, image alias, or licence type.
		* `imageAlias` (string)
			* description: Image Alias to be used for this volume. Note: when creating a volume - set image, image alias, or licence type.
		* `imagePassword` (string)
			* description: Initial password to be set for installed OS. Works with public images only. Not modifiable, forbidden in update requests. Password rules allows all characters from a-z, A-Z, 0-9.
	* required properties:
		* `bus`
		* `name`
* `availabilityZone` (string)
	* description: The availability zone in which the server should be provisioned.
	* default: "AUTO"
	* possible values: "AUTO";"ZONE_1";"ZONE_2"
* `cpuFamily` (string)
	* description: CPU architecture on which server gets provisioned; not all CPU architectures are available in all datacenter regions; available CPU architectures can be retrieved from the datacenter resource.
	* possible values: "AMD_OPTERON";"INTEL_SKYLAKE";"INTEL_XEON"
* `datacenterConfig` (object)
	* description: DatacenterConfig contains information about the datacenter resource on which the server will be created.
	* properties:
		* `datacenterIdSelector` (object)
			* description: DatacenterIDSelector selects reference to a Datacenter to retrieve its DatacenterID.
			* properties:
				* `matchControllerRef` (boolean)
					* description: MatchControllerRef ensures an object with the same controller reference as the selecting object is selected.
				* `matchLabels` (object)
					* description: MatchLabels ensures an object with matching labels is selected.
		* `datacenterId` (string)
			* description: DatacenterID is the ID of the Datacenter on which the resource will be created. It needs to be provided via directly or via reference.
			* format: uuid
		* `datacenterIdRef` (object)
			* description: DatacenterIDRef references to a Datacenter to retrieve its ID.
			* properties:
				* `name` (string)
					* description: Name of the referenced object.
			* required properties:
				* `name`

### Required Properties

The user needs to set the following properties in order to configure the IONOS Cloud Resource:

* `datacenterConfig`
* `template`
* `volume`

## Resource Definition

The corresponding resource definition can be found [here](https://github.com/ionos-cloud/crossplane-provider-ionoscloud/tree/master/package/crds/compute.ionoscloud.crossplane.io_cubeservers.yaml).

## Resource Instance Example

An example of a resource instance can be found [here](https://github.com/ionos-cloud/crossplane-provider-ionoscloud/tree/master/examples/ionoscloud/compute/cubeserver.yaml).

