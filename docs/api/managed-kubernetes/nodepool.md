---
description: Manages NodePool Resource on IONOS Cloud.
---

# NodePool Managed Resource

## Overview

* Resource Name: `NodePool`
* Resource Group: `k8s.ionoscloud.crossplane.io`
* Resource Version: `v1alpha1`
* Resource Scope: `Cluster`

## Usage

In order to manage resources on IONOS Cloud using Crossplane Provider, you need to have Crossplane Provider for IONOS Cloud installed into a Kubernetes Cluster, as a prerequisite. For a step-by-step guide, check the following [link](https://github.com/ionos-cloud/crossplane-provider-ionoscloud/tree/master/examples/example.md).

It is recommended to clone the repository for easier access to the example files.

### Create

Use the following command to create a resource instance. Before applying the file, check the properties defined in the `spec.forProvider` fields:

```
kubectl apply -f examples/ionoscloud/k8s/k8s-nodepool.yaml
```

_Note_: The command should be run from the root of the `crossplane-provider-ionoscloud` directory.

### Update

Use the following command to update an instance. Before applying the file, update the properties defined in the `spec.forProvider` fields:

```
kubectl apply -f examples/ionoscloud/k8s/k8s-nodepool.yaml
```

_Note_: The command should be run from the root of the `crossplane-provider-ionoscloud` directory.

### Wait

Use the following commands to wait for resources to be ready and synced. Update the `<instance-name>` accordingly:

```
kubectl wait --for=condition=ready nodepools.k8s.ionoscloud.crossplane.io/<instance-name>
kubectl wait --for=condition=synced nodepools.k8s.ionoscloud.crossplane.io/<instance-name>
```

### Get

Use the following command to get a list of the existing instances:

```
kubectl get nodepools.k8s.ionoscloud.crossplane.io
```

Use the following command to get a list of the existing instances with more details displayed:

```
kubectl get nodepools.k8s.ionoscloud.crossplane.io -o wide
```

Use the following command to get a list of the existing instances in JSON format:

```
kubectl get nodepools.k8s.ionoscloud.crossplane.io -o json
```

### Delete

Use the following command to destroy the resources created by applying the file:

```
kubectl delete -f examples/ionoscloud/k8s/k8s-nodepool.yaml
```

_Note_: The command should be run from the root of the `crossplane-provider-ionoscloud` directory.

## Properties

In order to configure the IONOS Cloud Resource, the user can set the `spec.forProvider` fields into the specification file for the resource instance. The required fields that need to be set can be found [here](#required-properties). Following, there is a list of all the properties:

* `lans` (array)
	* description: Array of additional private LANs attached to worker nodes
* `nodeCount` (integer)
	* description: The number of nodes that make up the node pool.
	* format: int32
* `annotations` (object)
	* description: Map of annotations attached to NodePool.
* `clusterConfig` (object)
	* description: The K8s Cluster on which the NodePool will be created
	* properties:
		* `clusterId` (string)
			* description: ClusterID is the ID of the Cluster on which the resource will be created. It needs to be provided via directly or via reference.
			* format: uuid
		* `clusterIdRef` (object)
			* description: ClusterIDRef references to a Cluster to retrieve its ID
			* properties:
				* `name` (string)
					* description: Name of the referenced object.
			* required properties:
				* `name`
		* `clusterIdSelector` (object)
			* description: ClusterIDSelector selects reference to a Cluster to retrieve its clusterId
			* properties:
				* `matchControllerRef` (boolean)
					* description: MatchControllerRef ensures an object with the same controller reference as the selecting object is selected.
				* `matchLabels` (object)
					* description: MatchLabels ensures an object with matching labels is selected.
* `cpuFamily` (string)
	* description: A valid CPU family name. If no CPUFamily is provided, it will be set the first CPUFamily supported by the location.
	* possible values: "AMD_OPTERON";"INTEL_SKYLAKE";"INTEL_XEON"
* `name` (string)
	* description: A Kubernetes node pool name. Valid Kubernetes node pool name must be 63 characters or less and must be empty or begin and end with an alphanumeric character ([a-z0-9A-Z]) with dashes (-), underscores (_), dots (.), and alphanumerics between.
* `ramSize` (integer)
	* description: The RAM size for the node. Must be set in multiples of 1024 MB, with minimum size is of 2048 MB.
	* format: int32
	* minimum: 2048.000000
	* multiple of: 1024.000000
* `autoScaling` (object)
	* description: property to be set when auto-scaling needs to be enabled for the NodePool. By default, auto-scaling is not enabled.
	* properties:
		* `minNodeCount` (integer)
			* description: The minimum number of worker nodes that the managed node group can scale in. Should be set together with 'maxNodeCount'. Value for this attribute must be greater than equal to 1 and less than equal to maxNodeCount.
			* format: int32
			* minimum: 1.000000
		* `maxNodeCount` (integer)
			* description: The maximum number of worker nodes that the managed node pool can scale-out. Should be set together with 'minNodeCount'. Value for this attribute must be greater than equal to 1 and minNodeCount.
			* format: int32
			* minimum: 1.000000
* `availabilityZone` (string)
	* description: The availability zone in which the target VM should be provisioned.
	* possible values: "AUTO";"ZONE_1";"ZONE_2"
* `labels` (object)
	* description: Map of labels attached to NodePool.
* `storageType` (string)
	* description: The type of hardware for the volume.
	* possible values: "HDD";"SSD"
* `datacenterConfig` (object)
	* description: A Datacenter, to which the user has access.
	* properties:
		* `datacenterIdSelector` (object)
			* description: DatacenterIDSelector selects reference to a Datacenter to retrieve its datacenterId
			* properties:
				* `matchControllerRef` (boolean)
					* description: MatchControllerRef ensures an object with the same controller reference as the selecting object is selected.
				* `matchLabels` (object)
					* description: MatchLabels ensures an object with matching labels is selected.
		* `datacenterId` (string)
			* description: DatacenterID is the ID of the Datacenter on which the resource should have access. It needs to be provided via directly or via reference.
			* format: uuid
		* `datacenterIdRef` (object)
			* description: DatacenterIDRef references to a Datacenter to retrieve its ID
			* properties:
				* `name` (string)
					* description: Name of the referenced object.
			* required properties:
				* `name`
* `k8sVersion` (string)
	* description: The Kubernetes version the NodePool is running. This imposes restrictions on what Kubernetes versions can be run in a cluster's NodePools. Additionally, not all Kubernetes versions are viable upgrade targets for all prior versions.
* `maintenanceWindow` (object)
	* description: The maintenance window is used for updating the software on the NodePool's nodes and for upgrading the NodePool's K8s version. If no value is given, one is chosen dynamically, so there is no fixed default.
	* properties:
		* `dayOfTheWeek` (string)
			* description: DayOfTheWeek The name of the week day.
		* `time` (string)
* `coresCount` (integer)
	* description: The number of cores for the node.
	* format: int32
* `publicIpsConfigs` (object)
	* description: Optional array of reserved public IP addresses to be used by the nodes. IPs must be from same location as the Datacenter used for the NodePool. The array must contain one more IP than the maximum possible number of nodes (nodeCount+1 for fixed number of nodes or maxNodeCount+1 when auto-scaling is used). The extra IP is used when the nodes are rebuilt. IPs can be set directly or via reference and indexes.
	* properties:
		* `ips` (array)
		* `ipsBlockConfigs` (array)
* `storageSize` (integer)
	* description: The size of the volume in GB. The size should be greater than 10GB.
	* format: int32
	* minimum: 10.000000

### Required Properties

The user needs to set the following properties in order to configure the IONOS Cloud Resource:

* `availabilityZone`
* `clusterConfig`
* `coresCount`
* `datacenterConfig`
* `name`
* `nodeCount`
* `ramSize`
* `storageSize`
* `storageType`

## Resource Definition

The corresponding resource definition can be found [here](https://github.com/ionos-cloud/crossplane-provider-ionoscloud/tree/master/package/crds/k8s.ionoscloud.crossplane.io_nodepools.yaml).

## Resource Instance Example

An example of a resource instance can be found [here](https://github.com/ionos-cloud/crossplane-provider-ionoscloud/tree/master/examples/ionoscloud/k8s/k8s-nodepool.yaml).

