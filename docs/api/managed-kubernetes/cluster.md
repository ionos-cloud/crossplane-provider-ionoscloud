---
description: Manages Cluster Resource on IONOS Cloud.
---

# Cluster Managed Resource

## Overview

* Description: A Cluster is an example API type.
* Resource Name: `Cluster`
* Resource Group: `k8s.ionoscloud.crossplane.io`
* Resource Version: `v1alpha1`
* Resource Scope: `Cluster`

## Usage

In order to manage resources on IONOS Cloud using Crossplane Provider, you need to have Crossplane Provider for IONOS Cloud installed into a Kubernetes Cluster, as a prerequisite. For a step-by-step guide, check the following [link](https://github.com/ionos-cloud/crossplane-provider-ionoscloud/tree/master/examples/example.md).

It is recommended to clone the repository for easier access to the example files.

### Create

Use the following command to create a resource instance. Before applying the file, check the properties defined in the `spec.forProvider` fields:

```bash
kubectl apply -f examples/ionoscloud/k8s/k8s-cluster.yaml
```

_Note_: The command should be run from the root of the `crossplane-provider-ionoscloud` directory.

### Update

Use the following command to update an instance. Before applying the file, update the properties defined in the `spec.forProvider` fields:

```bash
kubectl apply -f examples/ionoscloud/k8s/k8s-cluster.yaml
```

_Note_: The command should be run from the root of the `crossplane-provider-ionoscloud` directory.

### Wait

Use the following commands to wait for resources to be ready and synced. Update the `<instance-name>` accordingly:

```bash
kubectl wait --for=condition=ready clusters.k8s.ionoscloud.crossplane.io/<instance-name>
```

```bash
kubectl wait --for=condition=synced clusters.k8s.ionoscloud.crossplane.io/<instance-name>
```

### Get

Use the following command to get a list of the existing instances:

```bash
kubectl get -f clusters.k8s.ionoscloud.crossplane.io
```

_Note_: Use options `--output wide`, `--output json` to get more information about the resource instances.

### Delete

Use the following command to destroy the resources created by applying the file:

```bash
kubectl delete -f examples/ionoscloud/k8s/k8s-cluster.yaml
```

_Note_: The command should be run from the root of the `crossplane-provider-ionoscloud` directory.

## Properties

In order to configure the IONOS Cloud Resource, the user can set the `spec.forProvider` fields into the specification file for the resource instance. The required fields that need to be set can be found [here](#required-properties). Following, there is a list of all the properties:

* `apiSubnetAllowList` (array)
	* description: Access to the K8s API server is restricted to these CIDRs. Traffic, internal to the cluster, is not affected by this restriction.
	  If no allow-list is specified, access is not restricted.
	  If an IP without subnet mask is provided, the default value is used: 32 for IPv4 and 128 for IPv6.
	  Example: "1.2.3.4/32", "2002::1234:abcd:ffff:c0a8:101/64", "1.2.3.4", "2002::1234:abcd:ffff:c0a8:101"
* `k8sVersion` (string)
	* description: The Kubernetes version the cluster is running. This imposes restrictions on what Kubernetes versions can be run in a cluster's nodepools.
	  Additionally, not all Kubernetes versions are viable upgrade targets for all prior versions.
	  Example: 1.15.4
* `location` (string)
	* description: This attribute is mandatory if the cluster is private.
	  The location must be enabled for your contract, or you must have a data center at that location.
	  This attribute is immutable.
* `maintenanceWindow` (object)
	* description: The maintenance window is used for updating the cluster's control plane and for upgrading the cluster's K8s version.
	  If no value is given, one is chosen dynamically, so there is no fixed default.
	* properties:
		* `dayOfTheWeek` (string)
			* description: DayOfTheWeek The name of the week day.
		* `time` (string)
* `name` (string)
	* description: A Kubernetes cluster name. Valid Kubernetes cluster name must be 63 characters or less and must be empty
	  or begin and end with an alphanumeric character ([a-z0-9A-Z]) with dashes (-), underscores (_), dots (.), and alphanumerics between.
* `natGatewayIpConfig` (object)
	* description: The nat gateway IP of the cluster if the cluster is private. This
	  property is immutable. Must be a reserved IP in the same location as
	  the cluster's location. This attribute is mandatory if the cluster
	  is private.
	* properties:
		* `ip` (string)
			* description: Use IP to set specific IP to the resource. If both IP and IPBlockConfig are set,
			  only `ip` field will be considered.
		* `ipBlockConfig` (object)
			* description: Use IpBlockConfig to reference existing IPBlock, and to mention the index for the IP.
			  Index starts from 0 and it must be provided.
			* properties:
				* `index` (integer)
					* description: Index is referring to the IP index retrieved from the IPBlock.
					  Index is starting from 0.
				* `ipBlockId` (string)
					* description: IPBlockID is the ID of the IPBlock on which the resource will be created.
					  It needs to be provided via directly or via reference.
					* format: uuid
				* `ipBlockIdRef` (object)
					* description: IPBlockIDRef references to a IPBlock to retrieve its ID.
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
				* `ipBlockIdSelector` (object)
					* description: IPBlockIDSelector selects reference to a IPBlock to retrieve its IPBlockID.
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
			* required properties:
				* `index`
* `nodeSubnet` (string)
	* description: The node subnet of the cluster, if the cluster is private.
	  This attribute is optional and immutable.
	  Must be a valid CIDR notation for an IPv4 network prefix of 16 bits length.
* `public` (boolean)
	* description: The indicator if the cluster is public or private.
	  Be aware that setting it to false is currently in beta phase.
	* default: true
* `s3Buckets` (array)
	* description: List of IONOS Object Storage buckets configured for K8s usage.
	  For now, it contains only an IONOS Object Storage bucket used to store K8s API audit logs.
	* properties:
		* `name` (string)
	* required properties:
		* `name`

### Required Properties

The user needs to set the following properties in order to configure the IONOS Cloud Resource:

* `name`

## Resource Definition

The corresponding resource definition can be found [here](https://github.com/ionos-cloud/crossplane-provider-ionoscloud/tree/master/package/crds/k8s.ionoscloud.crossplane.io_clusters.yaml).

## Resource Instance Example

An example of a resource instance can be found [here](https://github.com/ionos-cloud/crossplane-provider-ionoscloud/tree/master/examples/ionoscloud/k8s/k8s-cluster.yaml).

