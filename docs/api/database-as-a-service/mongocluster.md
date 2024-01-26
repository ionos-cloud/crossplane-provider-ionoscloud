---
description: Manages MongoCluster Resource on IONOS Cloud.
---

# MongoCluster Managed Resource

## Overview

* Resource Name: `MongoCluster`
* Resource Group: `dbaas.mongo.ionoscloud.crossplane.io`
* Resource Version: `v1alpha1`
* Resource Scope: `Cluster`

## Usage

In order to manage resources on IONOS Cloud using Crossplane Provider, you need to have Crossplane Provider for IONOS Cloud installed into a Kubernetes Cluster, as a prerequisite. For a step-by-step guide, check the following [link](https://github.com/ionos-cloud/crossplane-provider-ionoscloud/tree/master/examples/example.md).

It is recommended to clone the repository for easier access to the example files.

### Create

Use the following command to create a resource instance. Before applying the file, check the properties defined in the `spec.forProvider` fields:

```bash
kubectl apply -f examples/ionoscloud/dbaas/mongo-cluster.yaml
```

_Note_: The command should be run from the root of the `crossplane-provider-ionoscloud` directory.

### Update

Use the following command to update an instance. Before applying the file, update the properties defined in the `spec.forProvider` fields:

```bash
kubectl apply -f examples/ionoscloud/dbaas/mongo-cluster.yaml
```

_Note_: The command should be run from the root of the `crossplane-provider-ionoscloud` directory.

### Wait

Use the following commands to wait for resources to be ready and synced. Update the `<instance-name>` accordingly:

```bash
kubectl wait --for=condition=ready mongoclusters.dbaas.mongo.ionoscloud.crossplane.io/<instance-name>
```

```bash
kubectl wait --for=condition=synced mongoclusters.dbaas.mongo.ionoscloud.crossplane.io/<instance-name>
```

### Get

Use the following command to get a list of the existing instances:

```bash
kubectl get -f mongoclusters.dbaas.mongo.ionoscloud.crossplane.io
```

_Note_: Use options `--output wide`, `--output json` to get more information about the resource instances.

### Delete

Use the following command to destroy the resources created by applying the file:

```bash
kubectl delete -f examples/ionoscloud/dbaas/mongo-cluster.yaml
```

_Note_: The command should be run from the root of the `crossplane-provider-ionoscloud` directory.

## Properties

In order to configure the IONOS Cloud Resource, the user can set the `spec.forProvider` fields into the specification file for the resource instance. The required fields that need to be set can be found [here](#required-properties). Following, there is a list of all the properties:

* `storageType` (string)
	* description: The storage type used in your cluster. Possible values: HDD, SSD Standard, SSD Premium
* `shards` (integer)
	* description: The total number of shards in the cluster.
	* format: int32
* `storageSize` (integer)
	* description: The amount of storage per instance in megabytes.
	* format: int32
* `synchronizationMode` (string)
	* description: SynchronizationMode Represents different modes of replication.
	* possible values: "ASYNCHRONOUS";"STRICTLY_SYNCHRONOUS";"SYNCHRONOUS"
* `type` (string)
	* description: The cluster type, either `replicaset` or `sharded-cluster`.
* `backup` (object)
	* description: The location where the cluster backups will be stored. If not set, the backup is stored in the nearest location of the cluster.
	* properties:
		* `location` (string)
			* description: The location where the cluster backups will be stored. If not set, the backup is stored in the nearest location of the cluster.
* `edition` (string)
	* description: The cluster edition.
* `instances` (integer)
	* description: The total number of instances in the cluster (one master and n-1 standbys).
	* format: int32
* `displayName` (string)
	* description: The friendly name of your cluster.
* `fromBackup` (object)
	* description: CreateRestoreRequest The restore request.
	* properties:
		* `backupId` (string)
			* description: The unique ID of the snapshot you want to restore.
		* `recoveryTargetTime` (string)
			* description: If this value is supplied as ISO 8601 timestamp, the backup will be replayed up until the given timestamp. If empty, the backup will be applied completely.
	* required properties:
		* `backupId`
* `ram` (integer)
	* description: The amount of memory per instance in megabytes. Has to be a multiple of 1024.
	* format: int32
	* multiple of: 1024.000000
* `templateID` (string)
	* description: The unique ID of the template, which specifies the number of cores, storage size, and memory. You cannot downgrade to a smaller template or minor edition (e.g. from business to playground). To get a list of all templates to confirm the changes use the /templates endpoint.
* `biConnector` (object)
	* description: The MongoDB Connector for Business Intelligence allows you to query a MongoDB database using SQL commands to aid in data analysis.
	* properties:
		* `enabled` (boolean)
			* description: The MongoDB Connector for Business Intelligence allows you to query a MongoDB database using SQL commands to aid in data analysis.
		* `host` (string)
			* description: The host where this new BI Connector is installed.
		* `port` (string)
			* description: Port number used when connecting to this new BI Connector.
* `connections` (array)
	* description: Connection - details about the network connection (datacenter, lan, CIDR) for your cluster.
	* properties:
		* `cidr` (array)
			* description: The IP and subnet for your cluster. Note: the following IP ranges are unavailable: 10.233.64.0/18 10.233.0.0/18 10.233.114.0/24.
		* `datacenterConfig` (object)
			* description: DatacenterConfig contains information about the datacenter resource.
			* properties:
				* `datacenterId` (string)
					* description: DatacenterID is the ID of the Datacenter on which the resource will be created. It needs to be provided via directly or via reference.
					* format: uuid
				* `datacenterIdRef` (object)
					* description: DatacenterIDRef references to a Datacenter to retrieve its ID.
					* properties:
						* `policy` (object)
							* description: Policies for referencing.
							* properties:
								* `resolution` (string)
									* description: Resolution specifies whether resolution of this reference is required. The default is 'Required', which means the reconcile will fail if the reference cannot be resolved. 'Optional' means this reference will be a no-op if it cannot be resolved.
									* default: "Required"
									* possible values: "Required";"Optional"
								* `resolve` (string)
									* description: Resolve specifies when this reference should be resolved. The default is 'IfNotPresent', which will attempt to resolve the reference only when the corresponding field is not present. Use 'Always' to resolve the reference on every reconcile.
									* possible values: "Always";"IfNotPresent"
						* `name` (string)
							* description: Name of the referenced object.
					* required properties:
						* `name`
				* `datacenterIdSelector` (object)
					* description: DatacenterIDSelector selects reference to a Datacenter to retrieve its DatacenterID.
					* properties:
						* `matchLabels` (object)
							* description: MatchLabels ensures an object with matching labels is selected.
						* `policy` (object)
							* description: Policies for selection.
							* properties:
								* `resolve` (string)
									* description: Resolve specifies when this reference should be resolved. The default is 'IfNotPresent', which will attempt to resolve the reference only when the corresponding field is not present. Use 'Always' to resolve the reference on every reconcile.
									* possible values: "Always";"IfNotPresent"
								* `resolution` (string)
									* description: Resolution specifies whether resolution of this reference is required. The default is 'Required', which means the reconcile will fail if the reference cannot be resolved. 'Optional' means this reference will be a no-op if it cannot be resolved.
									* default: "Required"
									* possible values: "Required";"Optional"
						* `matchControllerRef` (boolean)
							* description: MatchControllerRef ensures an object with the same controller reference as the selecting object is selected.
		* `lanConfig` (object)
			* description: LanConfig contains information about the lan resource.
			* properties:
				* `lanId` (string)
					* description: LanID is the ID of the Lan on which the cluster will connect to. It needs to be provided via directly or via reference.
				* `lanIdRef` (object)
					* description: LanIDRef references to a Lan to retrieve its ID.
					* properties:
						* `policy` (object)
							* description: Policies for referencing.
							* properties:
								* `resolution` (string)
									* description: Resolution specifies whether resolution of this reference is required. The default is 'Required', which means the reconcile will fail if the reference cannot be resolved. 'Optional' means this reference will be a no-op if it cannot be resolved.
									* default: "Required"
									* possible values: "Required";"Optional"
								* `resolve` (string)
									* description: Resolve specifies when this reference should be resolved. The default is 'IfNotPresent', which will attempt to resolve the reference only when the corresponding field is not present. Use 'Always' to resolve the reference on every reconcile.
									* possible values: "Always";"IfNotPresent"
						* `name` (string)
							* description: Name of the referenced object.
					* required properties:
						* `name`
				* `lanIdSelector` (object)
					* description: LanIDSelector selects reference to a Lan to retrieve its LanID.
					* properties:
						* `matchControllerRef` (boolean)
							* description: MatchControllerRef ensures an object with the same controller reference as the selecting object is selected.
						* `matchLabels` (object)
							* description: MatchLabels ensures an object with matching labels is selected.
						* `policy` (object)
							* description: Policies for selection.
							* properties:
								* `resolution` (string)
									* description: Resolution specifies whether resolution of this reference is required. The default is 'Required', which means the reconcile will fail if the reference cannot be resolved. 'Optional' means this reference will be a no-op if it cannot be resolved.
									* default: "Required"
									* possible values: "Required";"Optional"
								* `resolve` (string)
									* description: Resolve specifies when this reference should be resolved. The default is 'IfNotPresent', which will attempt to resolve the reference only when the corresponding field is not present. Use 'Always' to resolve the reference on every reconcile.
									* possible values: "Always";"IfNotPresent"
	* required properties:
		* `cidr`
		* `datacenterConfig`
		* `lanConfig`
* `cores` (integer)
	* description: The number of CPU cores per instance.
	* format: int32
* `location` (string)
	* description: Location - The physical location where the cluster will be created. This is the location where all your instances will be located. This property is immutable.
* `maintenanceWindow` (object)
	* description: MaintenanceWindow A weekly 4 hour-long window, during which maintenance might occur.
	* properties:
		* `dayOfTheWeek` (string)
			* description: DayOfTheWeek The name of the week day.
		* `time` (string)
* `mongoDBVersion` (string)
	* description: The MongoDB version of your cluster.

### Required Properties

The user needs to set the following properties in order to configure the IONOS Cloud Resource:

* `connections`
* `displayName`
* `instances`
* `location`
* `mongoDBVersion`
* `synchronizationMode`

## Resource Definition

The corresponding resource definition can be found [here](https://github.com/ionos-cloud/crossplane-provider-ionoscloud/tree/master/package/crds/dbaas.mongo.ionoscloud.crossplane.io_mongoclusters.yaml).

## Resource Instance Example

An example of a resource instance can be found [here](https://github.com/ionos-cloud/crossplane-provider-ionoscloud/tree/master/examples/ionoscloud/dbaas/mongo-cluster.yaml).

