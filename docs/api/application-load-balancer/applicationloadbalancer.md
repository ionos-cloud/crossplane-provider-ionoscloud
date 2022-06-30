---
description: Manages ApplicationLoadBalancer Resource on IONOS Cloud.
---

# ApplicationLoadBalancer Managed Resource

## Overview

* Resource Name: `ApplicationLoadBalancer`
* Resource Group: `alb.ionoscloud.crossplane.io`
* Resource Version: `v1alpha1`
* Resource Scope: `Cluster`

## Usage

In order to manage resources on IONOS Cloud using Crossplane Provider, you need to have Crossplane Provider for IONOS Cloud installed into a Kubernetes Cluster, as a prerequisite. For a step-by-step guide, check the following [link](https://github.com/ionos-cloud/crossplane-provider-ionoscloud/tree/master/examples/example.md).

It is recommended to clone the repository for easier access to the example files.

### Create

Use the following command to create a resource instance. Before applying the file, check the properties defined in the `spec.forProvider` fields:

```
kubectl apply -f examples/ionoscloud/alb/applicationloadbalancer.yaml
```

_Note_: The command should be run from the root of the `crossplane-provider-ionoscloud` directory.

### Update

Use the following command to update an instance. Before applying the file, update the properties defined in the `spec.forProvider` fields:

```
kubectl apply -f examples/ionoscloud/alb/applicationloadbalancer.yaml
```

_Note_: The command should be run from the root of the `crossplane-provider-ionoscloud` directory.

### Wait

Use the following commands to wait for resources to be ready and synced. Update the `<instance-name>` accordingly:

```
kubectl wait --for=condition=ready applicationloadbalancers.alb.ionoscloud.crossplane.io/<instance-name>
kubectl wait --for=condition=synced applicationloadbalancers.alb.ionoscloud.crossplane.io/<instance-name>
```

### Get

Use the following command to get a list of the existing instances:

```
kubectl get applicationloadbalancers.alb.ionoscloud.crossplane.io
```

Use the following command to get a list of the existing instances with more details displayed:

```
kubectl get applicationloadbalancers.alb.ionoscloud.crossplane.io -o wide
```

Use the following command to get a list of the existing instances in JSON format:

```
kubectl get applicationloadbalancers.alb.ionoscloud.crossplane.io -o json
```

### Delete

Use the following command to destroy the resources created by applying the file:

```
kubectl delete -f examples/ionoscloud/alb/applicationloadbalancer.yaml
```

_Note_: The command should be run from the root of the `crossplane-provider-ionoscloud` directory.

## Properties

In order to configure the IONOS Cloud Resource, the user can set the `spec.forProvider` fields into the specification file for the resource instance. The required fields that need to be set can be found [here](#required-properties). Following, there is a list of all the properties:

* `lbPrivateIps` (array)
	* description: Collection of private IP addresses with the subnet mask of the Application Load Balancer. IPs must contain valid a subnet mask. If no IP is provided, the system will generate an IP with /24 subnet.
* `listenerLanConfig` (object)
	* description: ID of the listening (inbound) LAN. Lan ID can be set directly or via reference.
	* properties:
		* `lanId` (string)
			* description: LanID is the ID of the Lan on which the resource will be created. It needs to be provided via directly or via reference.
		* `lanIdRef` (object)
			* description: LanIDRef references to a Lan to retrieve its ID.
			* properties:
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
* `name` (string)
	* description: The name of the Application Load Balancer.
* `targetLanConfig` (object)
	* description: ID of the balanced private target LAN (outbound). Lan ID can be set directly or via reference.
	* properties:
		* `lanId` (string)
			* description: LanID is the ID of the Lan on which the resource will be created. It needs to be provided via directly or via reference.
		* `lanIdRef` (object)
			* description: LanIDRef references to a Lan to retrieve its ID.
			* properties:
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
* `datacenterConfig` (object)
	* description: A Datacenter, to which the user has access, to provision the ApplicationLoadBalancer in.
	* properties:
		* `datacenterId` (string)
			* description: DatacenterID is the ID of the Datacenter on which the resource should have access. It needs to be provided via directly or via reference.
			* format: uuid
		* `datacenterIdRef` (object)
			* description: DatacenterIDRef references to a Datacenter to retrieve its ID.
			* properties:
				* `name` (string)
					* description: Name of the referenced object.
			* required properties:
				* `name`
		* `datacenterIdSelector` (object)
			* description: DatacenterIDSelector selects reference to a Datacenter to retrieve its DatacenterID.
			* properties:
				* `matchLabels` (object)
					* description: MatchLabels ensures an object with matching labels is selected.
				* `matchControllerRef` (boolean)
					* description: MatchControllerRef ensures an object with the same controller reference as the selecting object is selected.
* `ipsConfig` (object)
	* description: Collection of the Application Load Balancer IP addresses. (Inbound and outbound) IPs of the listenerLan are customer-reserved public IPs for the public Load Balancers, and private IPs for the private Load Balancers. The IPs can be set directly or using reference to the existing IPBlocks and indexes. If no indexes are set, all IPs from the corresponding IPBlock will be assigned. All IPs set on the Nic will be displayed on the status's ips field.
	* properties:
		* `ips` (array)
			* description: Use IPs to set specific IPs to the resource. If both IPs and IPsBlockConfigs are set, only `ips` field will be considered.
		* `ipsBlockConfigs` (array)
			* description: Use IpsBlockConfigs to reference existing IPBlocks, and to mention the indexes for the IPs. Indexes start from 0, and multiple indexes can be set. If no index is set, all IPs from the corresponding IPBlock will be assigned to the resource.

### Required Properties

The user needs to set the following properties in order to configure the IONOS Cloud Resource:

* `datacenterConfig`
* `listenerLanConfig`
* `name`
* `targetLanConfig`

## Resource Definition

The corresponding resource definition can be found [here](https://github.com/ionos-cloud/crossplane-provider-ionoscloud/tree/master/package/crds/alb.ionoscloud.crossplane.io_applicationloadbalancers.yaml).

## Resource Instance Example

An example of a resource instance can be found [here](https://github.com/ionos-cloud/crossplane-provider-ionoscloud/tree/master/examples/ionoscloud/alb/applicationloadbalancer.yaml).

