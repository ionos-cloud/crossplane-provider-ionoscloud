---
description: Manages Group Resource on IONOS Cloud.
---

# Group Managed Resource

## Overview

* Description: Group is the Schema for the Group resource API
* Resource Name: `Group`
* Resource Group: `compute.ionoscloud.crossplane.io`
* Resource Version: `v1alpha1`
* Resource Scope: `Cluster`

## Usage

In order to manage resources on IONOS Cloud using Crossplane Provider, you need to have Crossplane Provider for IONOS Cloud installed into a Kubernetes Cluster, as a prerequisite. For a step-by-step guide, check the following [link](https://github.com/ionos-cloud/crossplane-provider-ionoscloud/tree/master/examples/example.md).

It is recommended to clone the repository for easier access to the example files.

### Create

Use the following command to create a resource instance. Before applying the file, check the properties defined in the `spec.forProvider` fields:

```bash
kubectl apply -f examples/ionoscloud/compute/group.yaml
```

_Note_: The command should be run from the root of the `crossplane-provider-ionoscloud` directory.

### Update

Use the following command to update an instance. Before applying the file, update the properties defined in the `spec.forProvider` fields:

```bash
kubectl apply -f examples/ionoscloud/compute/group.yaml
```

_Note_: The command should be run from the root of the `crossplane-provider-ionoscloud` directory.

### Wait

Use the following commands to wait for resources to be ready and synced. Update the `<instance-name>` accordingly:

```bash
kubectl wait --for=condition=ready groups.compute.ionoscloud.crossplane.io/<instance-name>
```

```bash
kubectl wait --for=condition=synced groups.compute.ionoscloud.crossplane.io/<instance-name>
```

### Get

Use the following command to get a list of the existing instances:

```bash
kubectl get -f groups.compute.ionoscloud.crossplane.io
```

_Note_: Use options `--output wide`, `--output json` to get more information about the resource instances.

### Delete

Use the following command to destroy the resources created by applying the file:

```bash
kubectl delete -f examples/ionoscloud/compute/group.yaml
```

_Note_: The command should be run from the root of the `crossplane-provider-ionoscloud` directory.

## Properties

In order to configure the IONOS Cloud Resource, the user can set the `spec.forProvider` fields into the specification file for the resource instance. The required fields that need to be set can be found [here](#required-properties). Following, there is a list of all the properties:

* `accessActivityLog` (boolean)
	* description: AccessActivityLog privilege for a group to access activity logs.
	* default: false
* `accessAndManageCertificates` (boolean)
	* description: AccessAndManageCertificates privilege for a group to access and manage certificates.
	* default: false
* `accessAndManageDns` (boolean)
	* description: AccessAndManageDNS privilege for a group to access and manage dns records.
	* default: false
* `accessAndManageMonitoring` (boolean)
	* description: AccessAndManageMonitoring privilege for a group to access and manage monitoring related functionality
	* default: false
* `createBackupUnit` (boolean)
	* description: CreateBackupUnit privilege to create backup unit resource
	* default: false
* `createDataCenter` (boolean)
	* description: CreateDataCenter privilege to create datacenter resource
	* default: false
* `createFlowLog` (boolean)
	* description: CreateFlowLog privilege to create flow log resource
	* default: false
* `createInternetAccess` (boolean)
	* description: CreateInternetAccess privilege to create internet access
	* default: false
* `createK8sCluster` (boolean)
	* description: CreateK8sCluster privilege to create kubernetes cluster
	* default: false
* `createPcc` (boolean)
	* description: CreatePcc privilege to create private cross connect
	* default: false
* `createSnapshot` (boolean)
	* description: CreateSnapshot privilege to create snapshot
	* default: false
* `manageDBaaS` (boolean)
	* description: ManageDBaaS privilege to manage DBaaS related functionality
	* default: false
* `manageDataplatform` (boolean)
	* description: ManageDataPlatform privilege to access and manage the Data Platform
	* default: false
* `manageRegistry` (boolean)
	* description: ManageRegistry privilege to access container registry related functionality
	* default: false
* `name` (string)
	* description: Name of the resource.
* `reserveIp` (boolean)
	* description: ReserveIp privilege to reserve ip block
	* default: false
* `s3Privilege` (boolean)
	* description: S3Privilege privilege to access S3 functionality
	* default: false
* `sharedResourcesConfig` (array)
	* description: SharedResources allows sharing privilege to resources between the members of the group
In order to share a resource within a group, it must be referenced either by providing its ID directly
or by specifying a set of values by which its K8s object can be identified
	* properties:
		* `kind` (string)
			* description: Kind of the Custom Resource
		* `name` (string)
			* description: If ResourceID is not provided directly, the resource can be referenced through other attributes
These attributes mut all be provided for the Resource to be resolved successfully
Name of the kubernetes object instance of the Custom Resource
		* `resourceShare` (object)
			* description: ResourceShare
			* properties:
				* `editPrivilege` (boolean)
					* description: EditPrivilege for the Resource
					* default: false
				* `resourceId` (string)
					* description: ResourceID is the ID of the Resource to which Group members gain privileges
It can only be provided directly
					* format: uuid
				* `sharePrivilege` (boolean)
					* description: SharePrivilege for the Resource
					* default: false
		* `version` (string)
			* description: Version of the Custom Resource
* `userConfig` (array)
	* description: In order to add a User as member to the Group, it is recommended to use UserCfg
to add an existing User as a member (via id or via reference).
To remove a User from the Group, update the CR spec by removing it.

UserCfg contains information about an existing User resource
which will be added to the Group
	* properties:
		* `userId` (string)
			* description: UserID is the ID of the User on which the resource should have access.
It needs to be provided directly or via reference.
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

The corresponding resource definition can be found [here](https://github.com/ionos-cloud/crossplane-provider-ionoscloud/tree/master/package/crds/compute.ionoscloud.crossplane.io_groups.yaml).

## Resource Instance Example

An example of a resource instance can be found [here](https://github.com/ionos-cloud/crossplane-provider-ionoscloud/tree/master/examples/ionoscloud/compute/group.yaml).

