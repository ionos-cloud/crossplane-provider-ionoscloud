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

* `serverCertificatesIds` (array)
	* description: Array of items in the collection.
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
* `httpRules` (array)
	* description: An array of items in the collection. The original order of rules is preserved during processing, except for Forward-type rules are processed after the rules with other action defined. The relative order of Forward-type rules is also preserved during the processing.
	* properties:
		* `contentType` (string)
			* description: Valid only for STATIC actions. Example: text/html
		* `dropQuery` (boolean)
			* description: Default is false; valid only for REDIRECT actions.
		* `name` (string)
			* description: The unique name of the Application Load Balancer HTTP rule.
		* `responseMessage` (string)
			* description: The response message of the request; mandatory for STATIC actions.
		* `type` (string)
			* description: Type of the HTTP rule.
			* possible values: "FORWARD";"STATIC";"REDIRECT"
		* `conditions` (array)
			* description: An array of items in the collection. The action is only performed if each and every condition is met; if no conditions are set, the rule will always be performed.
			* properties:
				* `condition` (string)
					* description: Matching rule for the HTTP rule condition attribute; Mandatory for HEADER, PATH, QUERY, METHOD, HOST, and COOKIE types; Must be null when type is SOURCE_IP.
					* possible values: "EXISTS";"CONTAINS";"EQUALS";"MATCHES";"STARTS_WITH";"ENDS_WITH"
				* `key` (string)
					* description: Must be null when type is PATH, METHOD, HOST, or SOURCE_IP. Key can only be set when type is COOKIES, HEADER, or QUERY.
				* `negate` (boolean)
					* description: Specifies whether the condition is negated or not; the default is False.
				* `type` (string)
					* description: Type of the HTTP rule condition.
					* possible values: "HEADER";"PATH";"QUERY";"METHOD";"HOST";"COOKIE";"SOURCE_IP"
				* `value` (string)
					* description: Mandatory for conditions CONTAINS, EQUALS, MATCHES, STARTS_WITH, ENDS_WITH; Must be null when condition is EXISTS; should be a valid CIDR if provided and if type is SOURCE_IP.
			* required properties:
				* `condition`
				* `type`
		* `location` (string)
			* description: The location for redirecting; mandatory and valid only for REDIRECT actions. Example: www.ionos.com
		* `statusCode` (integer)
			* description: Valid only for REDIRECT and STATIC actions. For REDIRECT actions, default is 301 and possible values are 301, 302, 303, 307, and 308. For STATIC actions, default is 503 and valid range is 200 to 599.
			* format: int32
			* possible values: 301;302;303;307;308;200;503;599
		* `targetGroupConfig` (object)
			* description: The ID of the target group; mandatory and only valid for FORWARD actions. The ID can be set directly or via reference.
			* properties:
				* `targetGroupId` (string)
					* description: TargetGroupID is the ID of the TargetGroup on which the resource should have access. It needs to be provided via directly or via reference.
					* format: uuid
				* `targetGroupIdRef` (object)
					* description: TargetGroupIDRef references to a TargetGroup to retrieve its ID.
					* properties:
						* `name` (string)
							* description: Name of the referenced object.
					* required properties:
						* `name`
				* `targetGroupIdSelector` (object)
					* description: TargetGroupIDSelector selects reference to a TargetGroup to retrieve its TargetGroupID.
					* properties:
						* `matchControllerRef` (boolean)
							* description: MatchControllerRef ensures an object with the same controller reference as the selecting object is selected.
						* `matchLabels` (object)
							* description: MatchLabels ensures an object with matching labels is selected.
	* required properties:
		* `name`
		* `type`
* `listenerIpConfig` (object)
	* description: Listening (inbound) IP. IP must be assigned to the listener NIC of the Application Load Balancer.
	* properties:
		* `ip` (string)
			* description: Use IP to set specific IP to the resource. If both IP and IPBlockConfig are set, only `ip` field will be considered.
			* pattern: ^(?:(?:25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?).){3}(?:25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)$
		* `ipBlockConfig` (object)
			* description: Use IpBlockConfig to reference existing IPBlock, and to mention the index for the IP. Index starts from 0 and it must be provided.
			* properties:
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
				* `index` (integer)
					* description: Index is referring to the IP index retrieved from the IPBlock. Index starts from 0.
			* required properties:
				* `index`
* `listenerPort` (integer)
	* description: Listening (inbound) port number; valid range is 1 to 65535.
	* format: int32
	* minimum: 1.000000
	* maximum: 65535.000000
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

