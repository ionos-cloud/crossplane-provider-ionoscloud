---
description: Manages TargetGroup Resource on IONOS Cloud.
---

# TargetGroup Managed Resource

## Overview

* Resource Name: `TargetGroup`
* Resource Group: `alb.ionoscloud.crossplane.io`
* Resource Version: `v1alpha1`
* Resource Scope: `Cluster`

## Usage

In order to manage resources on IONOS Cloud using Crossplane Provider, you need to have Crossplane Provider for IONOS Cloud installed into a Kubernetes Cluster, as a prerequisite. For a step-by-step guide, check the following [link](https://github.com/ionos-cloud/crossplane-provider-ionoscloud/tree/master/examples/example.md).

It is recommended to clone the repository for easier access to the example files.

### Create

Use the following command to create a resource instance. Before applying the file, check the properties defined in the `spec.forProvider` fields:

```
kubectl apply -f examples/ionoscloud/alb/targetgroup.yaml
```

_Note_: The command should be run from the root of the `crossplane-provider-ionoscloud` directory.

### Update

Use the following command to update an instance. Before applying the file, update the properties defined in the `spec.forProvider` fields:

```
kubectl apply -f examples/ionoscloud/alb/targetgroup.yaml
```

_Note_: The command should be run from the root of the `crossplane-provider-ionoscloud` directory.

### Wait

Use the following commands to wait for resources to be ready and synced. Update the `<instance-name>` accordingly:

```
kubectl wait --for=condition=ready targetgroups.alb.ionoscloud.crossplane.io/<instance-name>
kubectl wait --for=condition=synced targetgroups.alb.ionoscloud.crossplane.io/<instance-name>
```

### Get

Use the following command to get a list of the existing instances:

```
kubectl get targetgroups.alb.ionoscloud.crossplane.io
```

Use the following command to get a list of the existing instances with more details displayed:

```
kubectl get targetgroups.alb.ionoscloud.crossplane.io -o wide
```

Use the following command to get a list of the existing instances in JSON format:

```
kubectl get targetgroups.alb.ionoscloud.crossplane.io -o json
```

### Delete

Use the following command to destroy the resources created by applying the file:

```
kubectl delete -f examples/ionoscloud/alb/targetgroup.yaml
```

_Note_: The command should be run from the root of the `crossplane-provider-ionoscloud` directory.

## Properties

In order to configure the IONOS Cloud Resource, the user can set the `spec.forProvider` fields into the specification file for the resource instance. The required fields that need to be set can be found [here](#required-properties). Following, there is a list of all the properties:

* `algorithm` (string)
	* description: Balancing algorithm
	* possible values: "ROUND_ROBIN";"LEAST_CONNECTION";"RANDOM";"SOURCE_IP"
* `healthCheck` (object)
	* description: Health check properties for target group
	* properties:
		* `checkInterval` (integer)
			* description: The interval in milliseconds between consecutive health checks; default is 2000.
			* format: int32
		* `checkTimeout` (integer)
			* description: The maximum time in milliseconds to wait for a target to respond to a check. For target VMs with 'Check Interval' set, the lesser of the two  values is used once the TCP connection is established.
			* format: int32
		* `retries` (integer)
			* description: The maximum number of attempts to reconnect to a target after a connection failure. Valid range is 0 to 65535, and default is three reconnection attempts.
			* format: int32
* `httpHealthCheck` (object)
	* description: HTTP health check properties for target group
	* properties:
		* `regex` (boolean)
		* `response` (string)
			* description: The response returned by the request, depending on the match type.
		* `matchType` (string)
			* possible values: "";"STATUS_CODE";"RESPONSE_BODY"
		* `method` (string)
			* description: The method for the HTTP health check.
			* possible values: "HEAD";"PUT";"POST";"GET";"TRACE";"PATCH";"OPTIONS"
		* `negate` (boolean)
		* `path` (string)
			* description: The path (destination URL) for the HTTP health check request; the default is /.
	* required properties:
		* `matchType`
		* `response`
* `name` (string)
	* description: The name of the target group.
* `protocol` (string)
	* description: Balancing protocol
	* possible values: "HTTP"
* `targets` (array)
	* description: Array of items in the collection.

### Required Properties

The user needs to set the following properties in order to configure the IONOS Cloud Resource:

* `algorithm`
* `name`
* `protocol`

## Resource Definition

The corresponding resource definition can be found [here](https://github.com/ionos-cloud/crossplane-provider-ionoscloud/tree/master/package/crds/alb.ionoscloud.crossplane.io_targetgroups.yaml).

## Resource Instance Example

An example of a resource instance can be found [here](https://github.com/ionos-cloud/crossplane-provider-ionoscloud/tree/master/examples/ionoscloud/alb/targetgroup.yaml).

