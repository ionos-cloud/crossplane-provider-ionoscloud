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

* `name` (string)
	* description: Name of the resource.
* `accessAndManageCertificates` (boolean)
	* description: AccessAndManageCertificates privilege for a group to access and manage certificates.
* `accessAndManageMonitoring` (boolean)
	* description: AccessAndManageMonitoring privilege for a group to access and manage monitoring related functionality
* `createBackupUnit` (boolean)
	* description: CreateBackupUnit privilege to create backup unit resource
* `createPcc` (boolean)
	* description: CreatePcc privilege to create private cross connect
* `manageRegistry` (boolean)
	* description: ManageRegistry privilege to access container registry related functionality
* `createDataCenter` (boolean)
	* description: CreateDataCenter privilege to create datacenter resource
* `createFlowLog` (boolean)
	* description: CreateFlowLog privilege to create flow log resource
* `createK8sCluster` (boolean)
	* description: CreateK8sCluster privilege to create kubernetes cluster
* `manageDataplatform` (boolean)
	* description: ManageDataPlatform privilege to access and manage the Data Platform
* `reserveIp` (boolean)
	* description: ReserveIp privilege to reserve ip block
* `s3Privilege` (boolean)
	* description: S3Privilege privilege to access S3 functionality
* `accessActivityLog` (boolean)
	* description: AccessActivityLog privilege for a group to access activity logs.
* `accessAndManageDns` (boolean)
	* description: AccessAndManageDNS privilege for a group to access and manage dns records.
* `createInternetAccess` (boolean)
	* description: CreateInternetAccess privilege to create internet access
* `createSnapshot` (boolean)
	* description: CreateSnapshot privilege to create snapshot
* `manageDBaaS` (boolean)
	* description: ManageDBaaS privilege to manage DBaaS related functionality

### Required Properties

The user needs to set the following properties in order to configure the IONOS Cloud Resource:

* `name`

## Resource Definition

The corresponding resource definition can be found [here](https://github.com/ionos-cloud/crossplane-provider-ionoscloud/tree/master/package/crds/compute.ionoscloud.crossplane.io_managementgroups.yaml).

## Resource Instance Example

An example of a resource instance can be found [here](https://github.com/ionos-cloud/crossplane-provider-ionoscloud/tree/master/examples/ionoscloud/compute/managementgroup.yaml).

