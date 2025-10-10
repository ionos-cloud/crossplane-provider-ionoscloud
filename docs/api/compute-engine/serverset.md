---
description: Manages ServerSet Resource on IONOS Cloud.
---

# ServerSet Managed Resource

## Overview

* Description: ServerSet represents a stateful set of servers in the Ionos Cloud.
The number of replicas controls how many resources it creates in the Ionos Cloud.
For 2 replicas defined, it will create for each: 1 server, 1 bootvolume, the nics configured(for each server).
Each sub-resource created(server, bootvolume, nic) will have it's own CR that can be observed using kubectl.
The SSet reads the active(master) identity from a configMap that needs to be named `config-lease`. If the configMap is not found, the active replica will be the first server created.
* Resource Name: `ServerSet`
* Resource Group: `compute.ionoscloud.crossplane.io`
* Resource Version: `v1alpha1`
* Resource Scope: `Cluster`

## Usage

In order to manage resources on IONOS Cloud using Crossplane Provider, you need to have Crossplane Provider for IONOS Cloud installed into a Kubernetes Cluster, as a prerequisite. For a step-by-step guide, check the following [link](https://github.com/ionos-cloud/crossplane-provider-ionoscloud/tree/master/examples/example.md).

It is recommended to clone the repository for easier access to the example files.

### Create

Use the following command to create a resource instance. Before applying the file, check the properties defined in the `spec.forProvider` fields:

```bash
kubectl apply -f examples/ionoscloud/compute/serverset.yaml
```

_Note_: The command should be run from the root of the `crossplane-provider-ionoscloud` directory.

### Update

Use the following command to update an instance. Before applying the file, update the properties defined in the `spec.forProvider` fields:

```bash
kubectl apply -f examples/ionoscloud/compute/serverset.yaml
```

_Note_: The command should be run from the root of the `crossplane-provider-ionoscloud` directory.

### Wait

Use the following commands to wait for resources to be ready and synced. Update the `<instance-name>` accordingly:

```bash
kubectl wait --for=condition=ready serversets.compute.ionoscloud.crossplane.io/<instance-name>
```

```bash
kubectl wait --for=condition=synced serversets.compute.ionoscloud.crossplane.io/<instance-name>
```

### Get

Use the following command to get a list of the existing instances:

```bash
kubectl get -f serversets.compute.ionoscloud.crossplane.io
```

_Note_: Use options `--output wide`, `--output json` to get more information about the resource instances.

### Delete

Use the following command to destroy the resources created by applying the file:

```bash
kubectl delete -f examples/ionoscloud/compute/serverset.yaml
```

_Note_: The command should be run from the root of the `crossplane-provider-ionoscloud` directory.

## Properties

In order to configure the IONOS Cloud Resource, the user can set the `spec.forProvider` fields into the specification file for the resource instance. The required fields that need to be set can be found [here](#required-properties). Following, there is a list of all the properties:

* `bootVolumeTemplate` (object)
	* description: BootVolumeTemplate are the configurable fields of a BootVolumeTemplate.
	* properties:
		* `metadata` (object)
			* description: ServerSetBootVolumeMetadata are the configurable fields of a ServerSetBootVolumeMetadata.
			* properties:
				* `labels` (object)
				* `name` (string)
					* description: Name of the BootVolume. Replica index, volume index, and version are appended to the name.
Resulting name will be in format: {name}-{replicaIndex}-{version}.
Version increases if the bootvolume is re-created due to an immutable field changing. E.g. if the image or the disk type are changed, the bootvolume is re-created and the version is increased.
					* pattern: [a-z0-9]([-a-z0-9]*[a-z0-9])?
			* required properties:
				* `name`
		* `spec` (object)
			* description: ServerSetBootVolumeSpec are the configurable fields of a ServerSetBootVolumeSpec.
			* properties:
				* `image` (string)
					* description: Image or snapshot ID to be used as template for this volume.
Make sure the image selected is compatible with the datacenter's location.
Note: when creating a volume and setting image, set imagePassword or SSKeys as well.
				* `imagePassword` (string)
					* description: Initial password to be set for installed OS. Works with public images only. Not modifiable, forbidden in update requests.
Password rules allows all characters from a-z, A-Z, 0-9.
					* pattern: ^[A-Za-z0-9]+$
				* `selector` (object)
					* description: A label selector is a label query over a set of resources. The result of matchLabels and
matchExpressions are ANDed. An empty label selector matches all objects. A null
label selector matches no objects.
					* properties:
						* `matchExpressions` (array)
							* description: matchExpressions is a list of label selector requirements. The requirements are ANDed.
							* properties:
								* `key` (string)
									* description: key is the label key that the selector applies to.
								* `operator` (string)
									* description: operator represents a key's relationship to a set of values.
Valid operators are In, NotIn, Exists and DoesNotExist.
								* `values` (array)
									* description: values is an array of string values. If the operator is In or NotIn,
the values array must be non-empty. If the operator is Exists or DoesNotExist,
the values array must be empty. This array is replaced during a strategic
merge patch.
							* required properties:
								* `key`
								* `operator`
						* `matchLabels` (object)
							* description: matchLabels is a map of {key,value} pairs. A single {key,value} in the matchLabels
map is equivalent to an element of matchExpressions, whose key field is "key", the
operator is "In", and the values array contains only "value". The requirements are ANDed.
				* `size` (number)
					* description: The size of the volume in GB.
				* `sshKeys` (array)
					* description: Public SSH keys are set on the image as authorized keys for appropriate SSH login to the instance using the corresponding private key.
This field may only be set in creation requests. When reading, it always returns null.
SSH keys are only supported if a public Linux image is used for the volume creation.
				* `substitutions` (array)
					* description: Substitutions are used to replace placeholders in the cloud-init configuration.
The property is immutable and is only allowed to be set on creation of a new a volume.
					* properties:
						* `key` (string)
							* description: The key that will be replaced by the value computed by the handler
						* `options` (object)
							* description: The options for the handler. For example, for ipv4Address and ipv6Address handlers, we need to specify cidr as an option
						* `type` (string)
							* description: The type of the handler that will be used for this substitution. The handler will
be responsible for computing the value we put in place of te key
							* possible values: "ipv4Address";"ipv6Address"
						* `unique` (boolean)
							* description: The value is unique across multiple ServerSets
					* required properties:
						* `key`
						* `options`
						* `type`
						* `unique`
				* `type` (string)
					* description: Changing type re-creates either the bootvolume, or the bootvolume, server and nic depending on the UpdateStrategy chosen`
					* possible values: "HDD";"SSD";"SSD Standard";"SSD Premium";"DAS";"ISO"
				* `updateStrategy` (object)
					* description: UpdateStrategy is the update strategy for the boot volume.
					* properties:
						* `type` (string)
							* description: UpdateStrategyType is the type of the update strategy for the boot volume.
							* default: "createBeforeDestroyBootVolume"
							* possible values: "createAllBeforeDestroy";"createBeforeDestroyBootVolume"
					* required properties:
						* `type`
				* `userData` (string)
					* description: The cloud-init configuration for the volume as base64 encoded string.
The property is immutable and is only allowed to be set on creation of a new a volume.
It is mandatory to provide either 'public image' or 'imageAlias' that has cloud-init compatibility in conjunction with this property.
Hostname is injected automatically in the userdata, in the format: {bootvolumeNameFromMetadata}-{replicaIndex}-{version}
PCI slots of the nics attached to the server are injected automatically in the userdata, with the key : {nic_pcislot}_{nicNameFromMetadata with - replaced by _} and the value : {pciSlot}
			* required properties:
				* `image`
				* `size`
				* `type`
				* `updateStrategy`
	* required properties:
		* `spec`
* `datacenterConfig` (object)
	* description: DatacenterConfig contains information about the datacenter resource
on which the server will be created.
	* properties:
		* `datacenterId` (string)
			* description: DatacenterID is the ID of the Datacenter on which the resource will be created.
It needs to be provided via directly or via reference.
			* format: uuid
		* `datacenterIdRef` (object)
			* description: DatacenterIDRef references to a Datacenter to retrieve its ID.
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
		* `datacenterIdSelector` (object)
			* description: DatacenterIDSelector selects reference to a Datacenter to retrieve its DatacenterID.
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
* `identityConfigMap` (object)
	* description: IdentityConfigMap is the configMap from which the identity of the ACTIVE server in the ServerSet is read. The configMap
should be created separately. The serverset only reads the status from it. If it does not find it, it sets
	// the first server as the ACTIVE.
	* properties:
		* `keyName` (string)
			* description: KeyName the key name in the configMap from which the identity of the ACTIVE server in the ServerSet is read.
		* `name` (string)
			* description: Name of the configMap from which the identity of the ACTIVE server in the ServerSet is read.
		* `namespace` (string)
			* description: Namespace of the configMap from which the identity of the ACTIVE server in the ServerSet is read.
* `replicas` (integer)
	* description: The number of servers that will be created. Cannot be decreased once set, only increased.
	* minimum: 1.000000
* `template` (object)
	* description: ServerSetTemplate are the configurable fields of a ServerSetTemplate.
	* properties:
		* `metadata` (object)
			* description: ServerSetMetadata are the configurable fields of a ServerSetMetadata.
			* properties:
				* `labels` (object)
				* `name` (string)
					* description: Name of the Server. Replica index and version are appended to the name. Resulting name will be in format: {name}-{replicaIndex}-{version}
Version increases if the Server is re-created due to an immutable field changing. E.g. if the bootvolume type or image are changed and the strategy is createAllBeforeDestroy, the Server is re-created and the version is increased.
					* pattern: [a-z0-9]([-a-z0-9]*[a-z0-9])?
			* required properties:
				* `name`
		* `spec` (object)
			* description: ServerSetTemplateSpec are the configurable fields of a ServerSetTemplateSpec.
			* properties:
				* `cores` (integer)
					* description: The total number of cores for the server.
					* format: int32
				* `nicMultiqueue` (boolean)
					* description: Activate or deactivate the Multi Queue feature on all NICs of this server.
				* `cpuFamily` (string)
					* description: CPU architecture on which server gets provisioned; not all CPU architectures are available in all datacenter regions;
available CPU architectures can be retrieved from the datacenter resource.
				* `nics` (array)
					* description: NICs are the network interfaces of the server.
					* properties:
						* `dhcp` (boolean)
						* `dhcpv6` (boolean)
						* `firewallActive` (boolean)
							* default: false
						* `firewallRules` (array)
							* properties:
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
									* description: Defines the end range of the allowed port (from 1 to 65534) if the protocol TCP or UDP is chosen.
Leave portRangeStart and portRangeEnd null to allow all ports.
									* format: int32
									* minimum: 1.000000
									* maximum: 65534.000000
								* `portRangeStart` (integer)
									* description: Defines the start range of the allowed port (from 1 to 65534) if protocol TCP or UDP is chosen.
Leave portRangeStart and portRangeEnd value null to allow all ports.
									* format: int32
									* minimum: 1.000000
									* maximum: 65534.000000
								* `protocol` (string)
									* description: The protocol for the rule. Property cannot be modified after it is created (disallowed in update requests).
									* possible values: "TCP";"UDP";"ICMP";"ANY"
								* `sourceIpConfig` (object)
									* description: Only traffic originating from the respective IPv4 address is allowed.
Value null allows traffic from any IP address.
SourceIP can be set directly or via reference to an IP Block and index.
									* properties:
										* `ip` (string)
											* description: Use IP or CIDR to set specific IP or CIDR to the resource. If both IP and IPBlockConfig are set,
only `ip` field will be considered.
											* pattern: ^([0-9]{1,3}\.){3}[0-9]{1,3}(/([0-9]|[1-2][0-9]|3[0-2]))?$
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
								* `sourceMac` (string)
									* description: Only traffic originating from the respective MAC address is allowed.
Valid format: aa:bb:cc:dd:ee:ff. Value null allows traffic from any MAC address.
									* pattern: ^([0-9A-Fa-f]{2}[:-]){5}([0-9A-Fa-f]{2})$
								* `targetIpConfig` (object)
									* description: If the target NIC has multiple IP addresses, only the traffic directed to the respective IP address of the NIC is allowed.
Value null allows traffic to any target IP address.
TargetIP can be set directly or via reference to an IP Block and index.
									* properties:
										* `ip` (string)
											* description: Use IP or CIDR to set specific IP or CIDR to the resource. If both IP and IPBlockConfig are set,
only `ip` field will be considered.
											* pattern: ^([0-9]{1,3}\.){3}[0-9]{1,3}(/([0-9]|[1-2][0-9]|3[0-2]))?$
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
								* `type` (string)
									* description: The type of the firewall rule. If not specified, the default INGRESS value is used.
									* default: "INGRESS"
									* possible values: "INGRESS";"EGRESS"
							* required properties:
								* `protocol`
						* `firewallType` (string)
							* description: The type of firewall rules that will be allowed on the NIC. If not specified, the default INGRESS value is used.
							* default: "INGRESS"
							* possible values: "BIDIRECTIONAL";"EGRESS";"INGRESS"
						* `lanReference` (string)
							* description: The Referenced LAN must be created before the ServerSet is applied
						* `name` (string)
							* description: Name of the NIC. Replica index, NIC index, and version are appended to the name. Resulting name will be in format: {name}-{replicaIndex}-{nicIndex}-{version}.
Version increases if the NIC is re-created due to an immutable field changing. E.g. if the bootvolume type or image are changed and the strategy is createAllBeforeDestroy, the NIC is re-created and the version is increased.
							* pattern: [a-z0-9]([-a-z0-9]*[a-z0-9])?
						* `vnetId` (string)
					* required properties:
						* `dhcp`
						* `lanReference`
						* `name`
				* `ram` (integer)
					* description: The memory size for the server in MB, such as 2048. Size must be specified in multiples of 256 MB with a minimum of 256 MB.
however, if you set ramHotPlug to TRUE then you must use a minimum of 1024 MB. If you set the RAM size more than 240GB,
then ramHotPlug will be set to FALSE and can not be set to TRUE unless RAM size not set to less than 240GB.
					* format: int32
					* multiple of: 1024.000000
			* required properties:
				* `cores`
				* `nics`
				* `ram`
	* required properties:
		* `metadata`
		* `spec`

### Required Properties

The user needs to set the following properties in order to configure the IONOS Cloud Resource:

* `bootVolumeTemplate`
* `datacenterConfig`
* `replicas`
* `template`

## Resource Definition

The corresponding resource definition can be found [here](https://github.com/ionos-cloud/crossplane-provider-ionoscloud/tree/master/package/crds/compute.ionoscloud.crossplane.io_serversets.yaml).

## Resource Instance Example

An example of a resource instance can be found [here](https://github.com/ionos-cloud/crossplane-provider-ionoscloud/tree/master/examples/ionoscloud/compute/serverset.yaml).

