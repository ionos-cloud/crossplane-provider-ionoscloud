---
description: Manages PostgresCluster Resource on IONOS Cloud.
---

# PostgresCluster Managed Resource

## Overview

* Resource Name: `PostgresCluster`
* Resource Group: `dbaas.ionoscloud.crossplane.io`
* Resource Version: `v1alpha1`
* Resource Scope: `Cluster`

## Usage

In order to manage resources on IONOS Cloud using Crossplane Provider, you need to have Crossplane Provider for IONOS Cloud installed into a Kubernetes Cluster, as a prerequisite. For a step-by-step guide, check the following [link](https://github.com/ionos-cloud/crossplane-provider-ionoscloud/tree/master/examples/example.md).

It is recommended to clone the repository for easier access to the example files.

### Create

Use the following command to create a resource instance. Before applying the file, check the properties defined in the `spec.forProvider` fields:

```
kubectl apply -f examples/ionoscloud/dbaas/postgres-cluster.yaml
```

_Note_: The command should be run from the root of the `crossplane-provider-ionoscloud` directory.

### Update

Use the following command to update an instance. Before applying the file, update the properties defined in the `spec.forProvider` fields:

```
kubectl apply -f examples/ionoscloud/dbaas/postgres-cluster.yaml
```

_Note_: The command should be run from the root of the `crossplane-provider-ionoscloud` directory.

### Wait

Use the following commands to wait for resources to be ready and synced. Update the `<instance-name>` accordingly:

```
kubectl wait --for=condition=ready postgresclusters.dbaas.ionoscloud.crossplane.io/<instance-name>
kubectl wait --for=condition=synced postgresclusters.dbaas.ionoscloud.crossplane.io/<instance-name>
```

### Get

Use the following command to get a list of the existing instances:

```
kubectl get postgresclusters.dbaas.ionoscloud.crossplane.io
```

Use the following command to get a list of the existing instances with more details displayed:

```
kubectl get postgresclusters.dbaas.ionoscloud.crossplane.io -o wide
```

Use the following command to get a list of the existing instances in JSON format:

```
kubectl get postgresclusters.dbaas.ionoscloud.crossplane.io -o json
```

### Delete

Use the following command to destroy the resources created by applying the file:

```
kubectl delete -f examples/ionoscloud/dbaas/postgres-cluster.yaml
```

_Note_: The command should be run from the root of the `crossplane-provider-ionoscloud` directory.

## Properties

In order to configure the IONOS Cloud Resource, the user can set the `spec.forProvider` fields into the specification file for the resource instance. The required fields that need to be set can be found [here](#required-properties). Following, there is a list of all the properties:

* `credentials` (object)
	* description: DBUser Credentials for the database user to be created.
	* properties:
		* `password` (string)
		* `username` (string)
			* description: The username for the initial postgres user. Some system usernames are restricted (e.g. \"postgres\", \"admin\", \"standby\").
	* required properties:
		* `password`
		* `username`
* `maintenanceWindow` (object)
	* description: MaintenanceWindow A weekly 4 hour-long window, during which maintenance might occur
	* properties:
		* `dayOfTheWeek` (string)
			* description: DayOfTheWeek The name of the week day.
		* `time` (string)
* `postgresVersion` (string)
	* description: The PostgreSQL version of your cluster.
* `connections` (array)
* `instances` (integer)
	* description: The total number of instances in the cluster (one master and n-1 standbys).
	* format: int32
* `backupLocation` (string)
	* description: The S3 location where the backups will be stored.
	* possible values: "de";"eu-south-2";"eu-central-2"
* `displayName` (string)
	* description: The friendly name of your cluster.
* `fromBackup` (object)
	* description: CreateRestoreRequest The restore request.
	* properties:
		* `backupId` (string)
			* description: The unique ID of the backup you want to restore.
		* `recoveryTargetTime` (string)
			* description: If this value is supplied as ISO 8601 timestamp, the backup will be replayed up until the given timestamp. If empty, the backup will be applied completely.
	* required properties:
		* `backupId`
* `ram` (integer)
	* description: The amount of memory per instance in megabytes. Has to be a multiple of 1024.
	* format: int32
	* multiple of: 1024.000000
* `cores` (integer)
	* description: The number of CPU cores per instance.
	* format: int32
* `storageSize` (integer)
	* description: The amount of storage per instance in megabytes.
	* format: int32
* `storageType` (string)
	* description: The storage type used in your cluster. Value "SSD" is deprecated. Use the equivalent "SSD Premium" instead.
	* possible values: "HDD";"SSD";"SSD Standard";"SSD Premium"
* `synchronizationMode` (string)
	* description: SynchronizationMode Represents different modes of replication.
	* possible values: "ASYNCHRONOUS";"STRICTLY_SYNCHRONOUS";"SYNCHRONOUS"
* `location` (string)
	* description: Location The physical location where the cluster will be created. This will be where all of your instances live. Property cannot be modified after datacenter creation. Location can have the following values: de/fra, us/las, us/ewr, de/txl, gb/lhr, es/vit.

### Required Properties

The user needs to set the following properties in order to configure the IONOS Cloud Resource:

* `connections`
* `cores`
* `credentials`
* `displayName`
* `instances`
* `location`
* `postgresVersion`
* `ram`
* `storageSize`
* `storageType`
* `synchronizationMode`

## Resource Definition

The corresponding resource definition can be found [here](https://github.com/ionos-cloud/crossplane-provider-ionoscloud/tree/master/package/crds/dbaas.ionoscloud.crossplane.io_postgresclusters.yaml).

## Resource Instance Example

An example of a resource instance can be found [here](https://github.com/ionos-cloud/crossplane-provider-ionoscloud/tree/master/examples/ionoscloud/dbaas/postgres-cluster.yaml).

