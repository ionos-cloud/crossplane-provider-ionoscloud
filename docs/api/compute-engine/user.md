---
description: Manages User Resource on IONOS Cloud.
---

# User Managed Resource

## Overview

* Description: User is our managed resource.
* Resource Name: `User`
* Resource Group: `compute.ionoscloud.crossplane.io`
* Resource Version: `v1alpha1`
* Resource Scope: `Cluster`

## Usage

In order to manage resources on IONOS Cloud using Crossplane Provider, you need to have Crossplane Provider for IONOS Cloud installed into a Kubernetes Cluster, as a prerequisite. For a step-by-step guide, check the following [link](https://github.com/ionos-cloud/crossplane-provider-ionoscloud/tree/master/examples/example.md).

It is recommended to clone the repository for easier access to the example files.

### Create

Use the following command to create a resource instance. Before applying the file, check the properties defined in the `spec.forProvider` fields:

```bash
kubectl apply -f examples/ionoscloud/compute/user.yaml
```

_Note_: The command should be run from the root of the `crossplane-provider-ionoscloud` directory.

### Update

Use the following command to update an instance. Before applying the file, update the properties defined in the `spec.forProvider` fields:

```bash
kubectl apply -f examples/ionoscloud/compute/user.yaml
```

_Note_: The command should be run from the root of the `crossplane-provider-ionoscloud` directory.

### Wait

Use the following commands to wait for resources to be ready and synced. Update the `<instance-name>` accordingly:

```bash
kubectl wait --for=condition=ready users.compute.ionoscloud.crossplane.io/<instance-name>
```

```bash
kubectl wait --for=condition=synced users.compute.ionoscloud.crossplane.io/<instance-name>
```

### Get

Use the following command to get a list of the existing instances:

```bash
kubectl get -f users.compute.ionoscloud.crossplane.io
```

_Note_: Use options `--output wide`, `--output json` to get more information about the resource instances.

### Delete

Use the following command to destroy the resources created by applying the file:

```bash
kubectl delete -f examples/ionoscloud/compute/user.yaml
```

_Note_: The command should be run from the root of the `crossplane-provider-ionoscloud` directory.

## Properties

In order to configure the IONOS Cloud Resource, the user can set the `spec.forProvider` fields into the specification file for the resource instance. The required fields that need to be set can be found [here](#required-properties). Following, there is a list of all the properties:

* `active` (boolean)
	* description: Active Indicates if the user is active. Default: true.
	* default: true
* `administrator` (boolean)
	* description: Administrator The group has permission to edit privileges on this resource.
* `email` (string)
	* description: Email An e-mail address for the user.
* `firstName` (string)
	* description: FirstName A first name for the user.
* `forceSecAuth` (boolean)
	* description: ForceSecAuth Indicates if secure (two-factor) authentication should be enabled for the user (true) or not (false).
* `groupIDs` (array)
	* description: GroupIDs that this user will be a member of. If not provided at all (null value), this field will be completely
ignored and will not trigger an update if a user is added to a group externally. If provided, this field will
need to match the crossplane managed groups that the user is a member of, otherwise a conflict will occur. In
order to remove a user from all groups that he is a member of, set this field to an empty array, **NOT** null value.
NOTE: This conflicts with UserConfig slice from Group resource, only use that one.
Deprecated: use UserConfig from Group resource.
* `lastName` (string)
	* description: LastName A last name for the user.
* `password` (string)
	* description: Password A password for the user.
Deprecated: use PasswordSecretRef
* `passwordSecretRef` (object)
	* description: PasswordSecretRef holds a reference to a secret containing the user's password.
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
* `secAuthActive` (boolean)
	* description: SecAuthActive Indicates if secure authentication is active for the user or not.
It can not be used in create requests - can be used in update. Default: false.
	* default: false

### Required Properties

The user needs to set the following properties in order to configure the IONOS Cloud Resource:

* `administrator`
* `email`
* `firstName`
* `forceSecAuth`
* `lastName`

## Resource Definition

The corresponding resource definition can be found [here](https://github.com/ionos-cloud/crossplane-provider-ionoscloud/tree/master/package/crds/compute.ionoscloud.crossplane.io_users.yaml).

## Resource Instance Example

An example of a resource instance can be found [here](https://github.com/ionos-cloud/crossplane-provider-ionoscloud/tree/master/examples/ionoscloud/compute/user.yaml).

