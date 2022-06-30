---
description: Manages Nic Resource on IONOS Cloud.
---

# Nic Managed Resource

## Overview

* Resource Name: `Nic`
* Resource Group: `compute.ionoscloud.crossplane.io`
* Resource Version: `v1alpha1`
* Resource Scope: `Cluster`

## Usage

In order to manage resources on IONOS Cloud using Crossplane Provider, you need to have Crossplane Provider for IONOS Cloud installed into a Kubernetes Cluster, as a prerequisite. For a step-by-step guide, check the following [link](https://github.com/ionos-cloud/crossplane-provider-ionoscloud/tree/master/examples/example.md).

It is recommended to clone the repository for easier access to the example files.

### Create

Use the following command to create a resource instance. Before applying the file, check the properties defined in the `spec.forProvider` fields:

```
kubectl apply -f examples/ionoscloud/compute/nic.yaml
```

_Note_: The command should be run from the root of the `crossplane-provider-ionoscloud` directory.

### Update

Use the following command to update an instance. Before applying the file, update the properties defined in the `spec.forProvider` fields:

```
kubectl apply -f examples/ionoscloud/compute/nic.yaml
```

_Note_: The command should be run from the root of the `crossplane-provider-ionoscloud` directory.

### Wait

Use the following commands to wait for resources to be ready and synced. Update the `<instance-name>` accordingly:

```
kubectl wait --for=condition=ready nics.compute.ionoscloud.crossplane.io/<instance-name>
kubectl wait --for=condition=synced nics.compute.ionoscloud.crossplane.io/<instance-name>
```

### Get

Use the following command to get a list of the existing instances:

```
kubectl get nics.compute.ionoscloud.crossplane.io
```

Use the following command to get a list of the existing instances with more details displayed:

```
kubectl get nics.compute.ionoscloud.crossplane.io -o wide
```

Use the following command to get a list of the existing instances in JSON format:

```
kubectl get nics.compute.ionoscloud.crossplane.io -o json
```

### Delete

Use the following command to destroy the resources created by applying the file:

```
kubectl delete -f examples/ionoscloud/compute/nic.yaml
```

_Note_: The command should be run from the root of the `crossplane-provider-ionoscloud` directory.

## Properties

In order to configure the IONOS Cloud Resource, the user can set the `spec.forProvider` fields into the specification file for the resource instance. The required fields that need to be set can be found [here](#required-properties). Following, there is a list of all the properties:

* `datacenterConfig` (object)
	* description: DatacenterConfig contains information about the datacenter resource on which the nic will be created
	* properties:
		* `datacenterId` (string)
			* description: DatacenterID is the ID of the Datacenter on which the resource will be created. It needs to be provided via directly or via reference.
			* format: uuid
		* `datacenterIdRef` (object)
			* description: DatacenterIDRef references to a Datacenter to retrieve its ID
			* properties:
				* `name` (string)
					* description: Name of the referenced object.
			* required properties:
				* `name`
		* `datacenterIdSelector` (object)
			* description: DatacenterIDSelector selects reference to a Datacenter to retrieve its datacenterId
			* properties:
				* `matchControllerRef` (boolean)
					* description: MatchControllerRef ensures an object with the same controller reference as the selecting object is selected.
				* `matchLabels` (object)
					* description: MatchLabels ensures an object with matching labels is selected.
* `dhcp` (boolean)
	* description: Indicates if the NIC will reserve an IP using DHCP.
* `firewallType` (string)
	* description: The type of firewall rules that will be allowed on the NIC. If not specified, the default INGRESS value is used.
	* possible values: "BIDIRECTIONAL";"EGRESS";"INGRESS"
* `ipsConfigs` (object)
	* description: Collection of IP addresses, assigned to the NIC. Explicitly assigned public IPs need to come from reserved IP blocks. Passing value null or empty array will assign an IP address automatically. The IPs can be set directly or using reference to the existing IPBlocks and indexes. If no indexes are set, all IPs from the corresponding IPBlock will be assigned. All IPs set on the Nic will be displayed on the status's ips field.
	* properties:
		* `ips` (array)
		* `ipsBlockConfigs` (array)
* `lanConfig` (object)
	* description: LanConfig contains information about the lan resource on which the nic will be on
	* properties:
		* `lanId` (string)
			* description: LanID is the ID of the Lan on which the resource will be created. It needs to be provided via directly or via reference.
		* `lanIdRef` (object)
			* description: LanIDRef references to a Lan to retrieve its ID
			* properties:
				* `name` (string)
					* description: Name of the referenced object.
			* required properties:
				* `name`
		* `lanIdSelector` (object)
			* description: LanIDSelector selects reference to a Lan to retrieve its lanId
			* properties:
				* `matchControllerRef` (boolean)
					* description: MatchControllerRef ensures an object with the same controller reference as the selecting object is selected.
				* `matchLabels` (object)
					* description: MatchLabels ensures an object with matching labels is selected.
* `name` (string)
	* description: The name of the  resource.
* `serverConfig` (object)
	* description: ServerConfig contains information about the server resource on which the nic will be created
	* properties:
		* `serverIdRef` (object)
			* description: ServerIDRef references to a Server to retrieve its ID
			* properties:
				* `name` (string)
					* description: Name of the referenced object.
			* required properties:
				* `name`
		* `serverIdSelector` (object)
			* description: ServerIDSelector selects reference to a Server to retrieve its serverId
			* properties:
				* `matchControllerRef` (boolean)
					* description: MatchControllerRef ensures an object with the same controller reference as the selecting object is selected.
				* `matchLabels` (object)
					* description: MatchLabels ensures an object with matching labels is selected.
		* `serverId` (string)
			* description: ServerID is the ID of the Server on which the resource will be created. It needs to be provided via directly or via reference.
			* format: uuid
* `firewallActive` (boolean)
	* description: Activate or deactivate the firewall. By default, an active firewall without any defined rules will block all incoming network traffic except for the firewall rules that explicitly allows certain protocols, IP addresses and ports.
* `mac` (string)
	* description: The MAC address of the NIC.

### Required Properties

The user needs to set the following properties in order to configure the IONOS Cloud Resource:

* `datacenterConfig`
* `dhcp`
* `lanConfig`
* `serverConfig`

## Resource Definition

The corresponding resource definition can be found [here](https://github.com/ionos-cloud/crossplane-provider-ionoscloud/tree/master/package/crds/compute.ionoscloud.crossplane.io_nics.yaml).

## Resource Instance Example

An example of a resource instance can be found [here](https://github.com/ionos-cloud/crossplane-provider-ionoscloud/tree/master/examples/ionoscloud/compute/nic.yaml).

