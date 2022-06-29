# Server Managed Resource

## Overview

* Resource Name: `Server`
* Resource Group: `compute.ionoscloud.crossplane.io`
* Resource Version: `v1alpha1`
* Resource Scope: `Cluster`

## Usage

In order to manage resources on IONOS Cloud using Crossplane Provider, you need to have Crossplane Provider for IONOS Cloud installed into a Kubernetes Cluster, as a prerequisite. For a step-by-step guide, check the following [link](https://github.com/ionos-cloud/crossplane-provider-ionoscloud/tree/master/examples/example.md).

It is recommended to clone the repository for easier access to the example files.

### Create

Use the following command to create a resource instance. Before applying the file, check the properties defined in the `spec.forProvider` fields:

```
kubectl apply -f examples/ionoscloud/compute/server.yaml
```

_Note_: The command should be run from the root of the `crossplane-provider-ionoscloud` directory.
### Update

Use the following command to update an instance. Before applying the file, update the properties defined in the `spec.forProvider` fields:

```
kubectl apply -f examples/ionoscloud/compute/server.yaml
```

_Note_: The command should be run from the root of the `crossplane-provider-ionoscloud` directory.
### Wait

Use the following commands to wait for resources to be ready and synced. Update the `<instance-name>` accordingly:

```
kubectl wait --for=condition=ready servers.compute.ionoscloud.crossplane.io/<instance-name>
kubectl wait --for=condition=synced servers.compute.ionoscloud.crossplane.io/<instance-name>
```

### Get

Use the following command to get a list of the existing instances:

```
kubectl get servers.compute.ionoscloud.crossplane.io
```

Use the following command to get a list of the existing instances with more details displayed:

```
kubectl get servers.compute.ionoscloud.crossplane.io -o wide
```

Use the following command to get a list of the existing instances in JSON format:

```
kubectl get servers.compute.ionoscloud.crossplane.io -o json
```

### Delete

Use the following command to destroy the resources created by applying the file:

```
kubectl delete -f examples/ionoscloud/compute/server.yaml
```

_Note_: The command should be run from the root of the `crossplane-provider-ionoscloud` directory.

## Properties

In order to configure the IONOS Cloud Resource, the user can set the `spec.forProvider` fields into the specification file for the resource instance. The required fields that need to be set can be found [here](#required-properties). Following, there is a list of all the properties:

* `availabilityZone` (string)
	* description: The availability zone in which the server should be provisioned.
	* default: "AUTO"
	* possible values: "AUTO";"ZONE_1";"ZONE_2"
* `bootCdromId` (string)
* `cores` (integer)
	* description: The total number of cores for the server.
	* format: int32
* `cpuFamily` (string)
	* description: CPU architecture on which server gets provisioned; not all CPU architectures are available in all datacenter regions; available CPU architectures can be retrieved from the datacenter resource.
	* possible values: "AMD_OPTERON";"INTEL_SKYLAKE";"INTEL_XEON"
* `datacenterConfig` (object)
	* description: DatacenterConfig contains information about the datacenter resource on which the server will be created
	* properties:
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
		* `datacenterId` (string)
			* description: DatacenterID is the ID of the Datacenter on which the resource will be created. It needs to be provided via directly or via reference.
			* format: uuid
* `name` (string)
	* description: The name of the  resource.
* `ram` (integer)
	* description: The memory size for the server in MB, such as 2048. Size must be specified in multiples of 256 MB with a minimum of 256 MB. however, if you set ramHotPlug to TRUE then you must use a minimum of 1024 MB. If you set the RAM size more than 240GB, then ramHotPlug will be set to FALSE and can not be set to TRUE unless RAM size not set to less than 240GB.
	* format: int32
	* multiple of: 256.000000
* `volumeConfig` (object)
	* description: In order to attach a volume to the server, it is recommended to use VolumeConfig to set the existing volume (via id or via reference). To detach a volume from the server, update the CR spec by removing it. 
 VolumeConfig contains information about the existing volume resource which will be attached to the server and set as bootVolume
	* properties:
		* `volumeId` (string)
			* description: VolumeID is the ID of the Volume. It needs to be provided via directly or via reference.
			* format: uuid
		* `volumeIdRef` (object)
			* description: VolumeIDRef references to a Volume to retrieve its ID
			* properties:
				* `name` (string)
					* description: Name of the referenced object.
			* required properties:
				* `name`
		* `volumeIdSelector` (object)
			* description: VolumeIDSelector selects reference to a Volume to retrieve its volumeId
			* properties:
				* `matchControllerRef` (boolean)
					* description: MatchControllerRef ensures an object with the same controller reference as the selecting object is selected.
				* `matchLabels` (object)
					* description: MatchLabels ensures an object with matching labels is selected.

### Required Properties

The user needs to set the following properties in order to configure the IONOS Cloud Resource:

* `cores`
* `datacenterConfig`
* `ram`

## Resource Definition

The corresponding resource definition can be found [here](https://github.com/ionos-cloud/crossplane-provider-ionoscloud/tree/master/package/crds/compute.ionoscloud.crossplane.io_servers.yaml).

## Resource Instance Example

An example of a resource instance can be found [here](https://github.com/ionos-cloud/crossplane-provider-ionoscloud/tree/master/examples/ionoscloud/compute/server.yaml).
