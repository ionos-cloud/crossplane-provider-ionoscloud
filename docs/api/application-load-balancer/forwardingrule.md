---
description: Manages ForwardingRule Resource on IONOS Cloud.
---

# ForwardingRule Managed Resource

## Overview

* Resource Name: `ForwardingRule`
* Resource Group: `alb.ionoscloud.crossplane.io`
* Resource Version: `v1alpha1`
* Resource Scope: `Cluster`

## Usage

In order to manage resources on IONOS Cloud using Crossplane Provider, you need to have Crossplane Provider for IONOS Cloud installed into a Kubernetes Cluster, as a prerequisite. For a step-by-step guide, check the following [link](https://github.com/ionos-cloud/crossplane-provider-ionoscloud/tree/master/examples/example.md).

It is recommended to clone the repository for easier access to the example files.

### Create

Use the following command to create a resource instance. Before applying the file, check the properties defined in the `spec.forProvider` fields:

```bash
kubectl apply -f examples/ionoscloud/alb/forwardingrule.yaml
```

_Note_: The command should be run from the root of the `crossplane-provider-ionoscloud` directory.

### Update

Use the following command to update an instance. Before applying the file, update the properties defined in the `spec.forProvider` fields:

```bash
kubectl apply -f examples/ionoscloud/alb/forwardingrule.yaml
```

_Note_: The command should be run from the root of the `crossplane-provider-ionoscloud` directory.

### Wait

Use the following commands to wait for resources to be ready and synced. Update the `<instance-name>` accordingly:

```bash
kubectl wait --for=condition=ready forwardingrules.alb.ionoscloud.crossplane.io/<instance-name>
```

```bash
kubectl wait --for=condition=synced forwardingrules.alb.ionoscloud.crossplane.io/<instance-name>
```

### Get

Use the following command to get a list of the existing instances:

```bash
kubectl get -f forwardingrules.alb.ionoscloud.crossplane.io
```

_Note_: Use options `--output wide`, `--output json` to get more information about the resource instances.

### Delete

Use the following command to destroy the resources created by applying the file:

```bash
kubectl delete -f examples/ionoscloud/alb/forwardingrule.yaml
```

_Note_: The command should be run from the root of the `crossplane-provider-ionoscloud` directory.

## Properties

In order to configure the IONOS Cloud Resource, the user can set the `spec.forProvider` fields into the specification file for the resource instance. The required fields that need to be set can be found [here](#required-properties). Following, there is a list of all the properties:

* `clientTimeout` (integer)
	* description: The maximum time in milliseconds to wait for the client to acknowledge or send data; default is 50,000 (50 seconds).
	* format: int32
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
				* `matchControllerRef` (boolean)
					* description: MatchControllerRef ensures an object with the same controller reference as the selecting object is selected.
				* `matchLabels` (object)
					* description: MatchLabels ensures an object with matching labels is selected.
* `listenerIpConfig` (object)
	* description: Listening (inbound) IP. IP must be assigned to the listener NIC of the Application Load Balancer.
	* properties:
		* `ip` (string)
			* description: Use IP to set specific IP to the resource. If both IP and IPBlockConfig are set, only `ip` field will be considered.
			* pattern: ^(?:(?:25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?).){3}(?:25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)$
		* `ipBlockConfig` (object)
			* description: Use IpBlockConfig to reference existing IPBlock, and to mention the index for the IP. Index starts from 0 and it must be provided.
			* properties:
				* `index` (integer)
					* description: Index is referring to the IP index retrieved from the IPBlock. Index starts from 0.
				* `ipBlockId` (string)
					* description: IPBlockID is the ID of the IPBlock on which the resource will be created. It needs to be provided via directly or via reference.
					* format: uuid
				* `ipBlockIdRef` (object)
					* description: IPBlockIDRef references to a IPBlock to retrieve its ID.
					* properties:
						* `name` (string)
							* description: Name of the referenced object.
					* required properties:
						* `name`
				* `ipBlockIdSelector` (object)
					* description: IPBlockIDSelector selects reference to a IPBlock to retrieve its IPBlockID.
					* properties:
						* `matchControllerRef` (boolean)
							* description: MatchControllerRef ensures an object with the same controller reference as the selecting object is selected.
						* `matchLabels` (object)
							* description: MatchLabels ensures an object with matching labels is selected.
			* required properties:
				* `index`
* `name` (string)
	* description: The name of the Application Load Balancer Forwarding Rule.
* `protocol` (string)
	* description: Balancing protocol
	* possible values: "HTTP"
* `applicationLoadBalancerConfig` (object)
	* description: An ApplicationLoadBalancer, to which the user has access, to provision the Forwarding Rule in.
	* properties:
		* `applicationLoadBalancerId` (string)
			* description: ApplicationLoadBalancerID is the ID of the ApplicationLoadBalancer on which the resource should have access. It needs to be provided via directly or via reference.
			* format: uuid
		* `applicationLoadBalancerIdRef` (object)
			* description: ApplicationLoadBalancerIDRef references to a Datacenter to retrieve its ID.
			* properties:
				* `name` (string)
					* description: Name of the referenced object.
			* required properties:
				* `name`
		* `applicationLoadBalancerIdSelector` (object)
			* description: ApplicationLoadBalancerIDSelector selects reference to a Datacenter to retrieve its DatacenterID.
			* properties:
				* `matchControllerRef` (boolean)
					* description: MatchControllerRef ensures an object with the same controller reference as the selecting object is selected.
				* `matchLabels` (object)
					* description: MatchLabels ensures an object with matching labels is selected.
* `httpRules` (array)
	* description: An array of items in the collection. The original order of rules is preserved during processing, except for Forward-type rules are processed after the rules with other action defined. The relative order of Forward-type rules is also preserved during the processing.
* `listenerPort` (integer)
	* description: Listening (inbound) port number; valid range is 1 to 65535.
	* format: int32
	* minimum: 1.000000
	* maximum: 65535.000000
* `serverCertificatesIds` (array)
	* description: Array of items in the collection.

### Required Properties

The user needs to set the following properties in order to configure the IONOS Cloud Resource:

* `applicationLoadBalancerConfig`
* `datacenterConfig`
* `listenerIpConfig`
* `listenerPort`
* `name`
* `protocol`

## Resource Definition

The corresponding resource definition can be found [here](https://github.com/ionos-cloud/crossplane-provider-ionoscloud/tree/master/package/crds/alb.ionoscloud.crossplane.io_forwardingrules.yaml).

## Resource Instance Example

An example of a resource instance can be found [here](https://github.com/ionos-cloud/crossplane-provider-ionoscloud/tree/master/examples/ionoscloud/alb/forwardingrule.yaml).

