---
description: Manages FirewallRule Resource on IONOS Cloud.
---

# FirewallRule Managed Resource

## Overview

* Resource Name: `FirewallRule`
* Resource Group: `compute.ionoscloud.crossplane.io`
* Resource Version: `v1alpha1`
* Resource Scope: `Cluster`

## Usage

In order to manage resources on IONOS Cloud using Crossplane Provider, you need to have Crossplane Provider for IONOS Cloud installed into a Kubernetes Cluster, as a prerequisite. For a step-by-step guide, check the following [link](https://github.com/ionos-cloud/crossplane-provider-ionoscloud/tree/master/examples/example.md).

It is recommended to clone the repository for easier access to the example files.

### Create

Use the following command to create a resource instance. Before applying the file, check the properties defined in the `spec.forProvider` fields:

```
kubectl apply -f examples/ionoscloud/compute/firewallrule.yaml
```

_Note_: The command should be run from the root of the `crossplane-provider-ionoscloud` directory.

### Update

Use the following command to update an instance. Before applying the file, update the properties defined in the `spec.forProvider` fields:

```
kubectl apply -f examples/ionoscloud/compute/firewallrule.yaml
```

_Note_: The command should be run from the root of the `crossplane-provider-ionoscloud` directory.

### Wait

Use the following commands to wait for resources to be ready and synced. Update the `<instance-name>` accordingly:

```
kubectl wait --for=condition=ready firewallrules.compute.ionoscloud.crossplane.io/<instance-name>
kubectl wait --for=condition=synced firewallrules.compute.ionoscloud.crossplane.io/<instance-name>
```

### Get

Use the following command to get a list of the existing instances:

```
kubectl get firewallrules.compute.ionoscloud.crossplane.io
```

Use the following command to get a list of the existing instances with more details displayed:

```
kubectl get firewallrules.compute.ionoscloud.crossplane.io -o wide
```

Use the following command to get a list of the existing instances in JSON format:

```
kubectl get firewallrules.compute.ionoscloud.crossplane.io -o json
```

### Delete

Use the following command to destroy the resources created by applying the file:

```
kubectl delete -f examples/ionoscloud/compute/firewallrule.yaml
```

_Note_: The command should be run from the root of the `crossplane-provider-ionoscloud` directory.

## Properties

In order to configure the IONOS Cloud Resource, the user can set the `spec.forProvider` fields into the specification file for the resource instance. The required fields that need to be set can be found [here](#required-properties). Following, there is a list of all the properties:

* `datacenterConfig` (object)
	* description: DatacenterConfig contains information about the datacenter resource on which the resource will be created
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
* `nicConfig` (object)
	* description: NicConfig contains information about the nic resource on which the resource will be created
	* properties:
		* `nicId` (string)
			* description: NicID is the ID of the Nic on which the resource will be created. It needs to be provided via directly or via reference.
			* format: uuid
		* `nicIdRef` (object)
			* description: NicIDRef references to a Nic to retrieve its ID
			* properties:
				* `name` (string)
					* description: Name of the referenced object.
			* required properties:
				* `name`
		* `nicIdSelector` (object)
			* description: NicIDSelector selects reference to a Nic to retrieve its nicId
			* properties:
				* `matchControllerRef` (boolean)
					* description: MatchControllerRef ensures an object with the same controller reference as the selecting object is selected.
				* `matchLabels` (object)
					* description: MatchLabels ensures an object with matching labels is selected.
* `portRangeStart` (integer)
	* description: Defines the start range of the allowed port (from 1 to 65534) if protocol TCP or UDP is chosen. Leave portRangeStart and portRangeEnd value null to allow all ports.
	* format: int32
	* minimum: 1.000000
	* maximum: 65534.000000
* `protocol` (string)
	* description: The protocol for the rule. Property cannot be modified after it is created (disallowed in update requests).
	* possible values: "TCP";"UDP";"ICMP";"ANY"
* `sourceMac` (string)
	* description: Only traffic originating from the respective MAC address is allowed. Valid format: aa:bb:cc:dd:ee:ff. Value null allows traffic from any MAC address.
	* pattern: ^([0-9A-Fa-f]{2}[:-]){5}([0-9A-Fa-f]{2})$
* `icmpCode` (integer)
	* description: Defines the allowed code (from 0 to 254) if protocol ICMP is chosen. Value null allows all codes.
	* format: int32
	* minimum: 0.000000
	* maximum: 254.000000
* `icmpType` (integer)
	* description: Defines the allowed type (from 0 to 254) if the protocol ICMP is chosen. Value null allows all types.
	* format: int32
	* minimum: 0.000000
	* maximum: 254.000000
* `name` (string)
	* description: The name of the  resource.
* `portRangeEnd` (integer)
	* description: Defines the end range of the allowed port (from 1 to 65534) if the protocol TCP or UDP is chosen. Leave portRangeStart and portRangeEnd null to allow all ports.
	* format: int32
	* minimum: 1.000000
	* maximum: 65534.000000
* `serverConfig` (object)
	* description: ServerConfig contains information about the server resource on which the resource will be created
	* properties:
		* `serverId` (string)
			* description: ServerID is the ID of the Server on which the resource will be created. It needs to be provided via directly or via reference.
			* format: uuid
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
* `sourceIpConfig` (object)
	* description: Only traffic originating from the respective IPv4 address is allowed. Value null allows traffic from any IP address. SourceIP can be set directly or via reference to an IP Block and index.
	* properties:
		* `ip` (string)
			* pattern: ^(?:(?:25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?).){3}(?:25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)$
		* `ipBlockConfig` (object)
			* description: IPBlockConfig - used by resources that need to link IPBlock via id or via reference to get one single IP.
			* properties:
				* `index` (integer)
					* description: Index is referring to the IP index retrieved from the IPBlock. Index is starting from 0.
				* `ipBlockId` (string)
					* description: NicID is the ID of the IPBlock on which the resource will be created. It needs to be provided via directly or via reference.
					* format: uuid
				* `ipBlockIdRef` (object)
					* description: IPBlockIDRef references to a IPBlock to retrieve its ID
					* properties:
						* `name` (string)
							* description: Name of the referenced object.
					* required properties:
						* `name`
				* `ipBlockIdSelector` (object)
					* description: IPBlockIDSelector selects reference to a IPBlock to retrieve its nicId
					* properties:
						* `matchControllerRef` (boolean)
							* description: MatchControllerRef ensures an object with the same controller reference as the selecting object is selected.
						* `matchLabels` (object)
							* description: MatchLabels ensures an object with matching labels is selected.
			* required properties:
				* `index`
* `targetIpConfig` (object)
	* description: If the target NIC has multiple IP addresses, only the traffic directed to the respective IP address of the NIC is allowed. Value null allows traffic to any target IP address. TargetIP can be set directly or via reference to an IP Block and index.
	* properties:
		* `ipBlockConfig` (object)
			* description: IPBlockConfig - used by resources that need to link IPBlock via id or via reference to get one single IP.
			* properties:
				* `index` (integer)
					* description: Index is referring to the IP index retrieved from the IPBlock. Index is starting from 0.
				* `ipBlockId` (string)
					* description: NicID is the ID of the IPBlock on which the resource will be created. It needs to be provided via directly or via reference.
					* format: uuid
				* `ipBlockIdRef` (object)
					* description: IPBlockIDRef references to a IPBlock to retrieve its ID
					* properties:
						* `name` (string)
							* description: Name of the referenced object.
					* required properties:
						* `name`
				* `ipBlockIdSelector` (object)
					* description: IPBlockIDSelector selects reference to a IPBlock to retrieve its nicId
					* properties:
						* `matchControllerRef` (boolean)
							* description: MatchControllerRef ensures an object with the same controller reference as the selecting object is selected.
						* `matchLabels` (object)
							* description: MatchLabels ensures an object with matching labels is selected.
			* required properties:
				* `index`
		* `ip` (string)
			* pattern: ^(?:(?:25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?).){3}(?:25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)$
* `type` (string)
	* description: The type of the firewall rule. If not specified, the default INGRESS value is used.
	* possible values: "INGRESS";"EGRESS"

### Required Properties

The user needs to set the following properties in order to configure the IONOS Cloud Resource:

* `datacenterConfig`
* `nicConfig`
* `protocol`
* `serverConfig`

## Resource Definition

The corresponding resource definition can be found [here](https://github.com/ionos-cloud/crossplane-provider-ionoscloud/tree/master/package/crds/compute.ionoscloud.crossplane.io_firewallrules.yaml).

## Resource Instance Example

An example of a resource instance can be found [here](https://github.com/ionos-cloud/crossplane-provider-ionoscloud/tree/master/examples/ionoscloud/compute/firewallrule.yaml).

