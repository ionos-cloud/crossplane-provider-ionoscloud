# CubeServer Managed Resource

## Overview

* Resource Name: CubeServer
* Resource Group: compute.ionoscloud.crossplane.io
* Resource Version: v1alpha1
* Resource Scope: Cluster

## Properties

The user can set the following properties in order to configure the IONOS Cloud Resource:

* `name`
	* description: The name of the  resource.
	* type: string
* `template`
	* description: The ID or the name of the template for creating a CUBE server.
	* type: object
	* properties:
		* `name`
			* description: The name of the  resource.
		* `templateId`
			* description: The ID of the  template.
* `volume`
	* description: DasVolumeProperties contains properties for the DAS volume attached to the Cube Server
	* type: object
	* properties:
		* `imageAlias`
			* description: Note: when creating a volume, set image, image alias, or licence type
		* `imagePassword`
			* description: Initial password to be set for installed OS. Works with public images only. Not modifiable, forbidden in update requests. Password rules allows all characters from a-z, A-Z, 0-9.
		* `licenceType`
			* description: OS type for this volume. Note: when creating a volume, set image, image alias, or licence type
		* `name`
			* description: The name of the DAS Volume.
		* `sshKeys`
			* description: Public SSH keys are set on the image as authorized keys for appropriate SSH login to the instance using the corresponding private key. This field may only be set in creation requests. When reading, it always returns null. SSH keys are only supported if a public Linux image is used for the volume creation.
		* `bus`
			* description: The bus type of the volume.
		* `image`
			* description: Image or snapshot ID to be used as template for this volume. Make sure the image selected is compatible with the datacenter's location. Note: when creating a volume, set image, image alias, or licence type
	* required properties:
		* `bus`
		* `name`
* `availabilityZone`
	* description: The availability zone in which the server should be provisioned.
	* type: string
	* default: &JSON{Raw:*[34 65 85 84 79 34],}
* `cpuFamily`
	* description: CPU architecture on which server gets provisioned; not all CPU architectures are available in all datacenter regions; available CPU architectures can be retrieved from the datacenter resource.
	* type: string
* `datacenterConfig`
	* description: DatacenterConfig contains information about the datacenter resource on which the server will be created
	* type: object
	* properties:
		* `datacenterId`
			* description: DatacenterID is the ID of the Datacenter on which the resource will be created. It needs to be provided via directly or via reference.
		* `datacenterIdRef`
			* description: DatacenterIDRef references to a Datacenter to retrieve its ID
		* `datacenterIdSelector`
			* description: DatacenterIDSelector selects reference to a Datacenter to retrieve its datacenterId

### Required Properties
The user needs to set the following properties in order to configure the IONOS Cloud Resource:

* `datacenterConfig`
* `template`
* `volume`

## Resource Definition

The corresponding resource definition can be found [here](https://github.com/ionos-cloud/crossplane-provider-ionoscloud/tree/master/package/crds/compute.ionoscloud.crossplane.io_cubeservers.yaml).

## Resource

An example for a resource instance can be found [here](https://github.com/ionos-cloud/crossplane-provider-ionoscloud/tree/master/examples/ionoscloud/compute/cubeserver.yaml).

## Usage

### Create/Update

The following command should be run from the root of the `crossplane-provider-ionoscloud` directory. Before applying the file, make sure to check the properties defined in the `spec.forProvider` fields:

```
kubectl apply -f examples/ionoscloud/compute/cubeserver.yaml
```

### Get

```
kubectl get cubeservers.compute.ionoscloud.crossplane.io
```

### Delete

```
kubectl delete -f examples/ionoscloud/compute/cubeserver.yaml
```

**Note**: the commands presented should be run from the root of the `crossplane-provider-ionoscloud` directory. Please clone the repository for easier access.
