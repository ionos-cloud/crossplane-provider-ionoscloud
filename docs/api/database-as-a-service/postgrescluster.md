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

```bash
kubectl apply -f examples/ionoscloud/dbaas/postgres-cluster.yaml
```

_Note_: The command should be run from the root of the `crossplane-provider-ionoscloud` directory.

### Update

Use the following command to update an instance. Before applying the file, update the properties defined in the `spec.forProvider` fields:

```bash
kubectl apply -f examples/ionoscloud/dbaas/postgres-cluster.yaml
```

_Note_: The command should be run from the root of the `crossplane-provider-ionoscloud` directory.

### Wait

Use the following commands to wait for resources to be ready and synced. Update the `<instance-name>` accordingly:

```bash
kubectl wait --for=condition=ready postgresclusters.dbaas.ionoscloud.crossplane.io/<instance-name>
```

```bash
kubectl wait --for=condition=synced postgresclusters.dbaas.ionoscloud.crossplane.io/<instance-name>
```

### Get

Use the following command to get a list of the existing instances:

```bash
kubectl get -f postgresclusters.dbaas.ionoscloud.crossplane.io
```

_Note_: Use options `--output wide`, `--output json` to get more information about the resource instances.

### Delete

Use the following command to destroy the resources created by applying the file:

```bash
kubectl delete -f examples/ionoscloud/dbaas/postgres-cluster.yaml
```

_Note_: The command should be run from the root of the `crossplane-provider-ionoscloud` directory.

## Properties

In order to configure the IONOS Cloud Resource, the user can set the `spec.forProvider` fields into the specification file for the resource instance. The required fields that need to be set can be found [here](#required-properties). Following, there is a list of all the properties:

* `backupLocation` (string)
	* description: The S3 location where the backups will be stored.
	* possible values: "de";"eu-south-2";"eu-central-2"
* `connections` (array)
	* description: Connection - details about the network connection (datacenter, lan, CIDR) for your cluster.
	* properties:
		* `cidr` (string)
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
						* `name` (string)
							* description: Name of the referenced object.
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
					* required properties:
						* `name`
				* `datacenterIdSelector` (object)
					* description: DatacenterIDSelector selects reference to a Datacenter to retrieve its DatacenterID.
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
		* `lanConfig` (object)
			* description: LanConfig contains information about the lan resource.
			* properties:
				* `lanId` (string)
					* description: LanID is the ID of the Lan on which the cluster will connect to. It needs to be provided via directly or via reference.
				* `lanIdRef` (object)
					* description: LanIDRef references to a Lan to retrieve its ID.
					* properties:
						* `name` (string)
							* description: Name of the referenced object.
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
* `credentials` (object)
	* description: Database credentials - either set directly, or as secret/path/env
	* properties:
		* `env` (object)
			* description: Env is a reference to an environment variable that contains credentials that must be used to connect to the provider.
			* properties:
				* `name` (string)
					* description: Name is the name of an environment variable.
			* required properties:
				* `name`
		* `fs` (object)
			* description: Fs is a reference to a filesystem location that contains credentials that must be used to connect to the provider.
			* properties:
				* `path` (string)
					* description: Path is a filesystem path.
			* required properties:
				* `path`
		* `password` (string)
		* `secretRef` (object)
			* description: A SecretRef is a reference to a secret key that contains the credentials that must be used to connect to the provider.
			* properties:
				* `key` (string)
					* description: The key to select.
				* `name` (string)
					* description: Name of the secret.
				* `namespace` (string)
					* description: Namespace of the secret.
			* required properties:
				* `key`
				* `name`
				* `namespace`
		* `source` (string)
			* description: Source of the provider credentials.
			* possible values: "None";"Secret";"InjectedIdentity";"Environment";"Filesystem"
		* `username` (string)
			* description: The username for the postgres user. Some system usernames are restricted (e.g. \"postgres\", \"admin\", \"standby\"). Password must have a minimum length o 10
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
* `instances` (integer)
	* description: The total number of instances in the cluster (one master and n-1 standbys).
	* format: int32
* `location` (string)
	* description: Location The physical location where the cluster will be created. This will be where all of your instances live. Property cannot be modified after datacenter creation. Location can have the following values: de/fra, us/las, us/ewr, de/txl, gb/lhr, es/vit.
* `maintenanceWindow` (object)
	* description: MaintenanceWindow A weekly 4 hour-long window, during which maintenance might occur.
	* properties:
		* `dayOfTheWeek` (string)
			* description: DayOfTheWeek The name of the week day.
		* `time` (string)
* `postgresVersion` (string)
	* description: The PostgreSQL version of your cluster.
* `ram` (integer)
	* description: The amount of memory per instance in megabytes. Has to be a multiple of 1024.
	* format: int32
	* multiple of: 1024.000000
* `storageSize` (integer)
	* description: The amount of storage per instance in megabytes.
	* format: int32
* `storageType` (string)
	* description: The storage type used in your cluster. Value "SSD" is deprecated. Use the equivalent "SSD Premium" instead.
	* possible values: "HDD";"SSD";"SSD Standard";"SSD Premium"
* `synchronizationMode` (string)
	* description: SynchronizationMode Represents different modes of replication.
	* possible values: "ASYNCHRONOUS";"STRICTLY_SYNCHRONOUS";"SYNCHRONOUS"

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

