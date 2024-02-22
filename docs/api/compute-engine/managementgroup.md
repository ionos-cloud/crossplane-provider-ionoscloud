---
description: Manages ManagementGroup Resource on IONOS Cloud.
---

# ManagementGroup Managed Resource

## Overview

* Resource Name: `ManagementGroup`
* Resource Group: `compute.ionoscloud.crossplane.io`
* Resource Version: `v1alpha1`
* Resource Scope: `Cluster`

## Usage

In order to manage resources on IONOS Cloud using Crossplane Provider, you need to have Crossplane Provider for IONOS Cloud installed into a Kubernetes Cluster, as a prerequisite. For a step-by-step guide, check the following [link](https://github.com/ionos-cloud/crossplane-provider-ionoscloud/tree/master/examples/example.md).

It is recommended to clone the repository for easier access to the example files.

### Create

Use the following command to create a resource instance. Before applying the file, check the properties defined in the `spec.forProvider` fields:

```bash
kubectl apply -f examples/ionoscloud/compute/managementgroup.yaml
```

_Note_: The command should be run from the root of the `crossplane-provider-ionoscloud` directory.

### Update

Use the following command to update an instance. Before applying the file, update the properties defined in the `spec.forProvider` fields:

```bash
kubectl apply -f examples/ionoscloud/compute/managementgroup.yaml
```

_Note_: The command should be run from the root of the `crossplane-provider-ionoscloud` directory.

### Wait

Use the following commands to wait for resources to be ready and synced. Update the `<instance-name>` accordingly:

```bash
kubectl wait --for=condition=ready managementgroups.compute.ionoscloud.crossplane.io/<instance-name>
```

```bash
kubectl wait --for=condition=synced managementgroups.compute.ionoscloud.crossplane.io/<instance-name>
```

### Get

Use the following command to get a list of the existing instances:

```bash
kubectl get -f managementgroups.compute.ionoscloud.crossplane.io
```

_Note_: Use options `--output wide`, `--output json` to get more information about the resource instances.

### Delete

Use the following command to destroy the resources created by applying the file:

```bash
kubectl delete -f examples/ionoscloud/compute/managementgroup.yaml
```

_Note_: The command should be run from the root of the `crossplane-provider-ionoscloud` directory.

## Properties

In order to configure the IONOS Cloud Resource, the user can set the `spec.forProvider` fields into the specification file for the resource instance. The required fields that need to be set can be found [here](#required-properties). Following, there is a list of all the properties:

* `accessActivityLog` (boolean)
	* description: AccessActivityLog privilege for a group to access activity logs.
* `accessAndManageCertificates` (boolean)
	* description: AccessAndManageCertificates privilege for a group to access and manage certificates.
* `accessAndManageDns` (boolean)
	* description: AccessAndManageDNS privilege for a group to access and manage dns records.
* `accessAndManageMonitoring` (boolean)
	* description: AccessAndManageMonitoring privilege for a group to access and manage monitoring related functionality
* `createBackupUnit` (boolean)
	* description: CreateBackupUnit privilege to create backup unit resource
* `createDataCenter` (boolean)
	* description: CreateDataCenter privilege to create datacenter resource
* `createFlowLog` (boolean)
	* description: CreateFlowLog privilege to create flow log resource
* `createInternetAccess` (boolean)
	* description: CreateInternetAccess privilege to create internet access
* `createK8sCluster` (boolean)
	* description: CreateK8sCluster privilege to create kubernetes cluster
* `createPcc` (boolean)
	* description: CreatePcc privilege to create private cross connect
* `createSnapshot` (boolean)
	* description: CreateSnapshot privilege to create snapshot
* `manageDBaaS` (boolean)
	* description: ManageDBaaS privilege to manage DBaaS related functionality
* `manageDataplatform` (boolean)
	* description: ManageDataPlatform privilege to access and manage the Data Platform
* `manageRegistry` (boolean)
	* description: ManageRegistry privilege to access container registry related functionality
* `name` (string)
	* description: Name of the resource.
* `reserveIp` (boolean)
	* description: ReserveIp privilege to reserve ip block
* `s3Privilege` (boolean)
	* description: S3Privilege privilege to access S3 functionality
* `userConfig` (array)
	* description: In order to add a User as member to the ManagementGroup, it is recommended to use UserCfg
to add an existing User as a member (via id or via reference).
To remove a User from the Group, update the CR spec by removing it.


UserCfg contains information about an existing User resource
which will be added to the Group
	* properties:
		* `userId` (string)
			* description: UserID is the ID of the User on which the resource should have access.
It needs to be provided via directly or via reference.
			* format: uuid
		* `userIdRef` (object)
			* description: UserIDRef references to a User to retrieve its ID.
			* properties:
				* `name` (string)
					* description: Name of the referenced object.
				* `policy` (object)
					* description: Policies for referencing.
					* properties:
						* `resolution` (string)
							* description: Resolution specifies whether resolution of this reference is required.
The default is 'Required', which means the reconcile will fail if the
reference cannot be resolved. 'Optional' means this reference will be
a no-op if it cannot be resolved.
							* default: "Required"
							* possible values: "Required";"Optional"
						* `resolve` (string)
							* description: Resolve specifies when this reference should be resolved. The default
is 'IfNotPresent', which will attempt to resolve the reference only when
the corresponding field is not present. Use 'Always' to resolve the
reference on every reconcile.
							* possible values: "Always";"IfNotPresent"
			* required properties:
				* `name`
		* `userIdSelector` (object)
			* description: UserIDSelector selects reference to a User to retrieve its UserID.
			* properties:
				* `matchControllerRef` (boolean)
					* description: MatchControllerRef ensures an object with the same controller reference
as the selecting object is selected.
				* `matchLabels` (object)
					* description: MatchLabels ensures an object with matching labels is selected.
				* `policy` (object)
					* description: Policies for selection.
					* properties:
						* `resolution` (string)
							* description: Resolution specifies whether resolution of this reference is required.
The default is 'Required', which means the reconcile will fail if the
reference cannot be resolved. 'Optional' means this reference will be
a no-op if it cannot be resolved.
							* default: "Required"
							* possible values: "Required";"Optional"
						* `resolve` (string)
							* description: Resolve specifies when this reference should be resolved. The default
is 'IfNotPresent', which will attempt to resolve the reference only when
the corresponding field is not present. Use 'Always' to resolve the
reference on every reconcile.
							* possible values: "Always";"IfNotPresent"

### Required Properties

The user needs to set the following properties in order to configure the IONOS Cloud Resource:

* `name`

## Resource Definition

The corresponding resource definition can be found [here](https://github.com/ionos-cloud/crossplane-provider-ionoscloud/tree/master/package/crds/compute.ionoscloud.crossplane.io_managementgroups.yaml).

## Resource Instance Example

An example of a resource instance can be found [here](https://github.com/ionos-cloud/crossplane-provider-ionoscloud/tree/master/examples/ionoscloud/compute/managementgroup.yaml).

