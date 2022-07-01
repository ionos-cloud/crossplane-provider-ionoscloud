---
description: Manages Volume Resource on IONOS Cloud.
---

# Volume Managed Resource

## Overview

* Resource Name: `Volume`
* Resource Group: `compute.ionoscloud.crossplane.io`
* Resource Version: `v1alpha1`
* Resource Scope: `Cluster`

## Usage

In order to manage resources on IONOS Cloud using Crossplane Provider, you need to have Crossplane Provider for IONOS Cloud installed into a Kubernetes Cluster, as a prerequisite. For a step-by-step guide, check the following [link](https://github.com/ionos-cloud/crossplane-provider-ionoscloud/tree/master/examples/example.md).

It is recommended to clone the repository for easier access to the example files.

### Create

Use the following command to create a resource instance. Before applying the file, check the properties defined in the `spec.forProvider` fields:

```bash
kubectl apply -f examples/ionoscloud/compute/volume.yaml
```

_Note_: The command should be run from the root of the `crossplane-provider-ionoscloud` directory.

### Update

Use the following command to update an instance. Before applying the file, update the properties defined in the `spec.forProvider` fields:

```bash
kubectl apply -f examples/ionoscloud/compute/volume.yaml
```

_Note_: The command should be run from the root of the `crossplane-provider-ionoscloud` directory.

### Wait

Use the following commands to wait for resources to be ready and synced. Update the `<instance-name>` accordingly:

```bash
kubectl wait --for=condition=ready volumes.compute.ionoscloud.crossplane.io/<instance-name>
```

```bash
kubectl wait --for=condition=synced volumes.compute.ionoscloud.crossplane.io/<instance-name>
```

### Get

Use the following command to get a list of the existing instances:

```bash
kubectl get -f volumes.compute.ionoscloud.crossplane.io
```

_Note_: Use options `--output wide`, `--output json` to get more information about the resource instances.

### Delete

Use the following command to destroy the resources created by applying the file:

```bash
kubectl delete -f examples/ionoscloud/compute/volume.yaml
```

_Note_: The command should be run from the root of the `crossplane-provider-ionoscloud` directory.

## Properties

In order to configure the IONOS Cloud Resource, the user can set the `spec.forProvider` fields into the specification file for the resource instance. The required fields that need to be set can be found [here](#required-properties). Following, there is a list of all the properties:

* `discVirtioHotUnplug` (boolean)
	* description: Hot-unplug capable Virt-IO drive (no reboot required). Not supported with Windows VMs.
* `imageAlias` (string)
	* description: Image Alias to be used for this volume. Note: when creating a volume - set image, image alias, or licence type.
* `imagePassword` (string)
	* description: Initial password to be set for installed OS. Works with public images only. Not modifiable, forbidden in update requests. Password rules allows all characters from a-z, A-Z, 0-9.
* `name` (string)
	* description: The name of the  resource.
* `sshKeys` (array)
	* description: Public SSH keys are set on the image as authorized keys for appropriate SSH login to the instance using the corresponding private key. This field may only be set in creation requests. When reading, it always returns null. SSH keys are only supported if a public Linux image is used for the volume creation.
* `datacenterConfig` (object)
	* description: DatacenterConfig contains information about the datacenter resource on which the server will be created.
	* properties:
		* `datacenterId` (string)
			* description: DatacenterID is the ID of the Datacenter on which the resource will be created. It needs to be provided via directly or via reference.
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
* `image` (string)
	* description: Image or snapshot ID to be used as template for this volume. Make sure the image selected is compatible with the datacenter's location. Note: when creating a volume, set image, image alias, or licence type
* `nicHotUnplug` (boolean)
	* description: Hot-unplug capable NIC (no reboot required).
* `discVirtioHotPlug` (boolean)
	* description: Hot-plug capable Virt-IO drive (no reboot required).
* `backupUnitConfig` (object)
	* description: BackupUnitCfg contains information about the backup unit resource that the user has access to. The property is immutable and is only allowed to be set on creation of a new a volume. It is mandatory to provide either 'public image' or 'imageAlias' in conjunction with this property.
	* properties:
		* `backupUnitIdRef` (object)
			* description: BackupUnitIDRef references to a BackupUnit to retrieve its ID.
			* properties:
				* `name` (string)
					* description: Name of the referenced object.
			* required properties:
				* `name`
		* `backupUnitIdSelector` (object)
			* description: BackupUnitIDSelector selects reference to a BackupUnit to retrieve its BackupUnitID.
			* properties:
				* `matchLabels` (object)
					* description: MatchLabels ensures an object with matching labels is selected.
				* `matchControllerRef` (boolean)
					* description: MatchControllerRef ensures an object with the same controller reference as the selecting object is selected.
		* `backupUnitId` (string)
			* description: BackupUnitID is the ID of the BackupUnit on which the resource will be created. It needs to be provided via directly or via reference.
			* format: uuid
* `bus` (string)
	* description: The bus type of the volume.
	* default: "VIRTIO"
	* possible values: "VIRTIO";"IDE";"UNKNOWN"
* `cpuHotPlug` (boolean)
	* description: Hot-plug capable CPU (no reboot required).
* `ramHotPlug` (boolean)
	* description: Hot-plug capable RAM (no reboot required).
* `type` (string)
	* description: Hardware type of the volume. DAS (Direct Attached Storage) could be used only in a composite call with a Cube server.
	* possible values: "HDD";"SSD";"SSD Standard";"SSD Premium";"DAS";"ISO"
* `availabilityZone` (string)
	* description: The availability zone in which the volume should be provisioned. The storage volume will be provisioned on as few physical storage devices as possible, but this cannot be guaranteed upfront. This is unavailable for DAS (Direct Attached Storage), and subject to availability for SSD.
	* possible values: "AUTO";"ZONE_1";"ZONE_2";"ZONE_3"
* `nicHotPlug` (boolean)
	* description: Hot-plug capable NIC (no reboot required).
* `size` (number)
	* description: The size of the volume in GB.
* `userData` (string)
	* description: The cloud-init configuration for the volume as base64 encoded string. The property is immutable and is only allowed to be set on creation of a new a volume. It is mandatory to provide either 'public image' or 'imageAlias' that has cloud-init compatibility in conjunction with this property.
* `licenceType` (string)
	* description: OS type for this volume. Note: when creating a volume - set image, image alias, or licence type.
	* possible values: "UNKNOWN";"WINDOWS";"WINDOWS2016";"WINDOWS2022";"LINUX";"OTHER"

### Required Properties

The user needs to set the following properties in order to configure the IONOS Cloud Resource:

* `datacenterConfig`
* `size`
* `type`

## Resource Definition

The corresponding resource definition can be found [here](https://github.com/ionos-cloud/crossplane-provider-ionoscloud/tree/master/package/crds/compute.ionoscloud.crossplane.io_volumes.yaml).

## Resource Instance Example

An example of a resource instance can be found [here](https://github.com/ionos-cloud/crossplane-provider-ionoscloud/tree/master/examples/ionoscloud/compute/volume.yaml).

