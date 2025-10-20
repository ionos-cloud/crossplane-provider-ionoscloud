---
description: Manages PostgresDatabase Resource on IONOS Cloud.
---

# PostgresDatabase Managed Resource

## Overview

* Description: A PostgresDatabase is an example API type.
* Resource Name: `PostgresDatabase`
* Resource Group: `dbaas.ionoscloud.crossplane.io`
* Resource Version: `v1alpha1`
* Resource Scope: `Cluster`

## Usage

In order to manage resources on IONOS Cloud using Crossplane Provider, you need to have Crossplane Provider for IONOS Cloud installed into a Kubernetes Cluster, as a prerequisite. For a step-by-step guide, check the following [link](https://github.com/ionos-cloud/crossplane-provider-ionoscloud/tree/master/examples/example.md).

It is recommended to clone the repository for easier access to the example files.

### Create

Use the following command to create a resource instance. Before applying the file, check the properties defined in the `spec.forProvider` fields:

```bash
kubectl apply -f examples/ionoscloud/dbaas/postgresdatabase.yaml
```

_Note_: The command should be run from the root of the `crossplane-provider-ionoscloud` directory.

### Update

Use the following command to update an instance. Before applying the file, update the properties defined in the `spec.forProvider` fields:

```bash
kubectl apply -f examples/ionoscloud/dbaas/postgresdatabase.yaml
```

_Note_: The command should be run from the root of the `crossplane-provider-ionoscloud` directory.

### Wait

Use the following commands to wait for resources to be ready and synced. Update the `<instance-name>` accordingly:

```bash
kubectl wait --for=condition=ready postgresdatabases.dbaas.ionoscloud.crossplane.io/<instance-name>
```

```bash
kubectl wait --for=condition=synced postgresdatabases.dbaas.ionoscloud.crossplane.io/<instance-name>
```

### Get

Use the following command to get a list of the existing instances:

```bash
kubectl get -f postgresdatabases.dbaas.ionoscloud.crossplane.io
```

_Note_: Use options `--output wide`, `--output json` to get more information about the resource instances.

### Delete

Use the following command to destroy the resources created by applying the file:

```bash
kubectl delete -f examples/ionoscloud/dbaas/postgresdatabase.yaml
```

_Note_: The command should be run from the root of the `crossplane-provider-ionoscloud` directory.

## Properties

In order to configure the IONOS Cloud Resource, the user can set the `spec.forProvider` fields into the specification file for the resource instance. The required fields that need to be set can be found [here](#required-properties). Following, there is a list of all the properties:

* `clusterConfig` (object)
	* description: ClusterConfig is used by resources that need to link psql clusters via id or via reference.
	* properties:
		* `ClusterId` (string)
			* description: ClusterID is the ID of the Cluster on which the resource will be created.
			  It needs to be provided via directly or via reference.
			* format: uuid
		* `ClusterIdRef` (object)
			* description: ClusterIDRef references to a Cluster to retrieve its ID.
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
		* `ClusterIdSelector` (object)
			* description: ClusterIDSelector selects reference to a Cluster to retrieve its ClusterID.
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
* `name` (string)
	* description: The database name.
* `owner` (object)
	* description: The name of the role owning a given database.
	* properties:
		* `UserNameRef` (object)
			* description: UserIDRef references to a User to retrieve its Name.
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
		* `UserNameSelector` (object)
			* description: UserNameSelector selects reference to a User to retrieve its UserName.
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
		* `userName` (string)
			* description: UserName is the Name of the User on which the resource will be created.
			  It needs to be provided directly or via reference.

### Required Properties

The user needs to set the following properties in order to configure the IONOS Cloud Resource:

* `clusterConfig`
* `name`
* `owner`

## Resource Definition

The corresponding resource definition can be found [here](https://github.com/ionos-cloud/crossplane-provider-ionoscloud/tree/master/package/crds/dbaas.ionoscloud.crossplane.io_postgresdatabases.yaml).

## Resource Instance Example

An example of a resource instance can be found [here](https://github.com/ionos-cloud/crossplane-provider-ionoscloud/tree/master/examples/ionoscloud/dbaas/postgresdatabase.yaml).

