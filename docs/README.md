# Crossplane Provider IONOS Cloud

## Overview

Crossplane Provider for IONOS Cloud gives the ability to manage IONOS Cloud infrastructure directly from Kubernetes. Crossplane extends a Kubernetes cluster to support orchestrating any infrastructure or managed service. Providers can use Crossplane to enable infrastructure resource provisioning of a specific API.

{% hint style="info" %}
**Note:** You can access and contribute to the [<mark style="color:blue;">crossplane-provider-ionoscloud</mark>](https://github.com/ionos-cloud/crossplane-provider-ionoscloud) repository for IONOS Cloud.
{% endhint %}

 The provider that is built from the source code from the repository can be installed into a Crossplane control plane and adds the following new functionality:

  * Custom Resource Definitions (CRDs) model IONOS Cloud infrastructure and services. For example, Compute Engine, Kubernetes, Database As a Service Postgres, etc.
  * Controllers provision these resources in IONOS Cloud based on the user's desired state captured in CRDs when created.
  * Implementations of Crossplane's portable resource abstractions that enable IONOS Cloud resources to fulfill a user's general need for cloud services.

## Getting started 

To start with Crossplane usage and concepts, see the official [<mark style="color:blue;">Crossplane Documentation</mark>](https://docs.crossplane.io/v1.15/getting-started/).

To get started with Crossplane Provider for IONOS Cloud, see [<mark style="color:blue;">Crossplane Provider IONOS Cloud Usage Example</mark>](../examples/example.md), which provides details about the provisioning of a **Postgres cluster** in IONOS Cloud.

## Prerequisites

Ensure that you have the following:

* An IONOS Cloud account that you use for [<mark style="color:blue;">DCD</mark>](https://dcd.ionos.com/latest/) or other Config Management Tools.

* A Kubernetes cluster and [<mark style="color:blue;">Install Crossplane</mark>](https://docs.crossplane.io/latest/software/install/) into a namespace called `crossplane-system`. 

{% hint style="info" %}
**Note:**  You can install a Kubernetes Cluster locally by using kind or any other lightweight Kubernetes version and Crossplane. For more information, see [<mark style="color:blue;">Crossplane Provider IONOS Cloud Usage Example</mark>](../examples/example.md). 
{% endhint %}

## Authentication on IONOS Cloud

To authenticate to the IONOS Cloud APIs, you need to provide credentials for Crossplane Provider for IONOS Cloud. This can be done using the ``base64-encoded`` static credentials in a Kubernetes ``Secret``.

### Environment variables

Crossplane Provider IONOS Cloud uses a ``ProviderConfig`` to set up credentials via ``Secrets``. You can use environment variables when creating the ``ProviderConfig`` resource.

| **Environment Variable** | **Description**                                                                                                                                                                                                                     |
|----------------------|---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|
| ``IONOS_USERNAME``     | Provide the username used to login to authenticate against the IONOS Cloud API.                                                                                                                                                | 
| ``IONOS_PASSWORD``     | Provide the password used to login to authenticate against the IONOS Cloud API.                                                                                                                                                 | 
| ``IONOS_TOKEN``        | Provide the token used to login if a token is being used instead of **username** and **password** .                                                                                                                                     |
| ``IONOS_API_URL``      | Provide the API URL. It will overwrite the API endpoint default value `api.ionos.com`. |


{% hint style="info" %}
**Note:** The host URL does not contain the ``/cloudapi/v6`` path, so it should not be included in the ``IONOS_API_URL`` environment variable.
{% endhint %}

### Create Provider secret

To create the Provider secret, you can use either of the following methods:

* [<mark style="color:blue;">Using username and password</mark>](#using-username-and-password)
* [<mark style="color:blue;">Using token</mark>](#using-token)

{% hint style="info" %}
**Note:** We recommend **using token** to create the Provider secret.
{% endhint %}

#### Using username and password

Run the following command:

```bash
export BASE64_PW=$(echo -n "${IONOS_PASSWORD}" | base64)
kubectl create secret generic --namespace crossplane-system example-provider-secret --from-literal=credentials="{\"user\":\"${IONOS_USERNAME}\",\"password\":\"${BASE64_PW}\"}"
```

#### Using token

Run the following command:

```bash
kubectl create secret generic --namespace crossplane-system example-provider-secret --from-literal=credentials="{\"token\":\"${IONOS_TOKEN}\"}"
```
{% hint style="info" %}
**Note:** 
* You can overwrite the default IONOS Cloud API endpoint, by setting the credentials to: `credentials="{\"host_url\":\"${IONOS_API_URL}\"}"`.
* You can also set the `IONOS_API_URL` environment variable in the `ControllerConfig` of the provider globally for
all resources. The following snippet shows how to set it globally in the ControllerConfig:

    ```bash
  cat <<EOF | kubectl apply -f -
  apiVersion: pkg.crossplane.io/v1alpha1
  kind: ControllerConfig
  metadata:
    name: overwrite-ionos-api-url
  spec:
    env:
      - name: IONOS_API_URL
        value: "${IONOS_API_URL}"
  EOF
  ```

{% endhint %}

### Configure the Provider

You can create the ``ProviderConfig`` to configure credentials for Crossplane Provider for IONOS Cloud:

```bash
cat <<EOF | kubectl apply -f -
apiVersion: ionoscloud.crossplane.io/v1alpha1
kind: ProviderConfig
metadata:
  name: example
spec:
  credentials:
    source: Secret
    secretRef:
      namespace: crossplane-system
      name: example-provider-secret
      key: credentials
EOF
```

## Installation

To install the Provider, follow this process:

### Install Provider

Run the following command:

```bash
cat <<EOF | kubectl apply -f -
apiVersion: pkg.crossplane.io/v1
kind: Provider
metadata:
  name: provider-ionos
spec:
  package: ghcr.io/ionos-cloud/crossplane-provider-ionoscloud:latest
EOF
```

### Check installation

To check if the Crossplane Provider IONOS Cloud is installed and healthy, run the following command:

```bash
kubectl get providers
```

You will now be able to see the pods. Run the following command to see the pods running in the `crossplane-system` namespace:

```bash
kubectl get pods -n crossplane-system 
```

## Provision resources on IONOS Cloud

Once you have configured the IONOS Cloud Provider, you can provision the resources on the IONOS Cloud directly from your Kubernetes Cluster using Custom Resource Definitions(CRDs). All CRDs are **Cluster Scoped**, that is, not being created only one specific namespace, but on the entire cluster.

### Compute Engine managed resources

You can see an up-to-date list of the CRDs and corresponding IONOS Cloud resources that Crossplane Provider IONOS Cloud supports as Managed Resources.
`
#### Compute Engine custom resource definitions

| **Resources in IONOS Cloud** | **Custom Resource Definition**                       |
|--------------------------|--------------------------------------------------|
| IPBlocks                 | ``ipblocks.compute.ionoscloud.crossplane.io``      |
| Data Centers              | ``datacenters.compute.ionoscloud.crossplane.io``   |
| Servers                  | ``servers.compute.ionoscloud.crossplane.io``      |
| Volumes                  | ``volumes.compute.ionoscloud.crossplane.io``      |
| Lans                     | ``lans.compute.ionoscloud.crossplane.io``          |
| NICs                     | ``nics.compute.ionoscloud.crossplane.io``         |
| FirewallRules            | ``firewallrules.compute.ionoscloud.crossplane.io`` |
| IPFailovers              | ``ipfailovers.compute.ionoscloud.crossplane.io``   |

For more information, see [<mark style="color:blue;">Compute Engine API</mark>](https://github.com/ionos-cloud/crossplane-provider-ionoscloud/tree/master/apis/compute).

### Application Load Balancer managed resources

You can see an up-to-date list of Application Load Balancer managed resources that Crossplane Provider IONOS Cloud supports.

#### Application Load Balancer resources custom resource definitions

| **Resources in IONOS Cloud** | **Custom Resource Definition**                           |
|--------------------------|--------------------------------------------------------|
| ApplicationLoadBalancer  | ``applicationloadbalancer.alb.ionoscloud.crossplane.io`` |
| ForwardingRule           | ``forwardingrule.alb.ionoscloud.crossplane.io``          |
| TargetGroup              | ``targetgroup.alb.ionoscloud.crossplane.io``            |

For more information, see [<mark style="color:blue;">Application Load Balancer API</mark>](https://github.com/ionos-cloud/crossplane-provider-ionoscloud/tree/master/apis/alb).

### Kubernetes managed resources
You can see an up-to-date list of Managed Kubernetes resources that Crossplane Provider IONOS Cloud supports.

#### Kubernetes resources custom resource definitions

| **Resources in IONOS Cloud** | **Custom Resource Definition**                           |
|--------------------------|------------------------------------------|
| K8s Clusters             | ``clusters.k8s.ionoscloud.crossplane.io`` |
| K8s NodePools            | ``nodepools.k8s.ionoscloud.crossplane.io`` |

For more information, see [<mark style="color:blue;">Managed Kubernetes API</mark>](https://github.com/ionos-cloud/crossplane-provider-ionoscloud/tree/master/apis/k8s).

### Backup Managed Resources

You can see an up-to-date list of Backup managed resources that Crossplane Provider IONOS Cloud supports.

#### Backup resources custom resource definitions

| **Resources in IONOS Cloud** | **Custom Resource Definition**                           |
|--------------------------|-----------------------------------------------|
| BackupUnit               | ``backupunits.backup.ionoscloud.crossplane.io`` |

For more information, see [<mark style="color:blue;">Managed Backup API</mark>](api/managed-backup).

### DBaaS Postgres Managed Resources
You can see an up-to-date list of DBaaS Postgres managed resources that Crossplane Provider IONOS Cloud supports.

#### DBaaS Postgres resources custom resource definitions

| **Resources in IONOS Cloud** | **Custom Resource Definition**                           |
|--------------------------|---------------------------------------------------|
| DBaaS Postgres Clusters  | ``postgresclusters.dbaas.ionoscloud.crossplane.io`` |

For more information, see [<mark style="color:blue;">DBaaS API</mark>](api/database-as-a-service).

## References

References are used to reference other resources on which the new resources are dependent. Using referenced
resources, you can create a data center and a LAN using one command, without manually updating the LAN
CR specification file with the required ``datacenterId``.

The references are resolved only once, when the resource is created, and the resolvers are generated
using [<mark style="color:blue;">crossplane-tools</mark>](https://github.com/crossplane/crossplane-tools).

### Example

```yaml
datacenterConfig:
  datacenterIdRef:
    name: <datacenter_CR_name>
```

You can set the ``datacenterId`` directly, using:

```yaml
datacenterConfig:
  datacenterId: <datacenter_ID>
```

{% hint style="info" %}
**Note:** If both the ``datacenterId`` and the ``datacenterIdRef`` fields are set, then the ``datacenterId`` value has priority.
{% endhint %}

## Compositions and claims

Composite Resources are designed to help you build your platform and mix-and-match schemas for different providers. You can define the schema of your Composite Resource (XR) and update Crossplane about the Managed Resources, that is, CRs or Custom Resources, it should create when a user creates the XR.

To define, configure and claim composite resources, follow these steps:

1. [<mark style="color:blue;">Define Composite Resources</mark>](#define-composite-resources)
2. [<mark style="color:blue;">Configure Compositions</mark>](#configure-compositions)
3. [<mark style="color:blue;">Claim Composite Resources</mark>](#claim-composite-resources) 

#### Define Composite Resources

You need to define the ``CompositeResourceDefinition`` so that Crossplane knows which XRs you would like to create
and which fields those XRs should have. You can do this using the [<mark style="color:blue;">Definition File</mark>](../examples/composition/definition.yaml).

#### Configure Compositions

Once you have defined the Composite Resources, you need to train your Crossplane. Compositions link an XR with
one or multiple CRs; that is, **IP Blocks**, **Postgres Clusters**, **Node pools**, **clusters**, etc. You can control the CRs for
IONOS Cloud Resources via XRs: whenever an XR is created, updated, or deleted, according to the Composition configured. Crossplane will create, update, or delete CRs. You can do this using the [<mark style="color:blue;">Composition File</mark>](../examples/composition/composition.yaml).

#### Claim Composite Resources

Once you have configured Compositions, you need to create Composite Resource Claims. The difference between Claims and XRs is that **Claims** are **namespaced scoped**, while XRs are **cluster scoped**. An XR contains references to the CRs, while a claim contains references to the corresponding XR. You can do this using the [<mark style="color:blue;">Claim File</mark>](../examples/composition/claim.yaml).

### Example

To create a data center, a Kubernetes Cluster and a Kubernetes Node Pool via Compositions and Claims, follow the [<mark style="color:blue;">Composition Example</mark>](../examples/composition/composite.yaml). For more information, see [<mark style="color:blue;">Composite Resources</mark>](https://docs.crossplane.io/latest/concepts/composite-resources/).
{% endhint %}

## Name uniqueness support for IONOS Cloud resources

To enable name uniqueness support for IONOS Cloud Resources, the Crossplane Provider IONOS Cloud you can use the ``--unique-names`` flag. If the `--unique-names` option is set, the Crossplane Provider for IONOS Cloud will check if a resource with the same name already exists. If multiple resources with the specified name are found, an error is thrown. If a single resource with the specified name is found, Crossplane Provider will perform an extra step and check if the immutable parameters are as expected. If the resource has the specified name but different immutable parameters, an error is thrown. If no resource with the specified name is found, a new resource will be created.

{% hint style="info" %}
**Note:** Resources will have unique names at their level. Example: k8s clusters will have unique name per account and k8s Node
Pools will have unique name per k8s cluster.
{% endhint %}

You can create a ``ControllerConfig`` file using:

```bash
cat <<EOF | kubectl apply -f -
apiVersion: pkg.crossplane.io/v1alpha1
kind: ControllerConfig
metadata:
  name: provider-config
spec:
  args:
    - --unique-names
EOF
```

And reference it from the ``Provider`` using:

```bash
cat <<EOF | kubectl apply -f -
apiVersion: pkg.crossplane.io/v1
kind: Provider
metadata:
  name: provider-ionos
spec:
  package: ghcr.io/ionos-cloud/crossplane-provider-ionoscloud:latest
  controllerConfigRef:
    name: provider-config
EOF
```

## Debug mode

To debug the Crossplane Provider IONOS Cloud you can use the ```--debug`` flag.

### Provider logs

You can create a ``ControllerConfig`` file using:

```bash
cat <<EOF | kubectl apply -f -
apiVersion: pkg.crossplane.io/v1alpha1
kind: ControllerConfig
metadata:
  name: debug-config
spec:
  args:
    - --debug
EOF
```

And reference it from the ``Provider`` using:

```bash
cat <<EOF | kubectl apply -f -
apiVersion: pkg.crossplane.io/v1
kind: Provider
metadata:
  name: provider-ionos
spec:
  package: ghcr.io/ionos-cloud/crossplane-provider-ionoscloud:latest
  controllerConfigRef:
    name: debug-config
EOF
```

To see logs of the Crossplane Provider IONOS Cloud controller's pod, use:

```bash
kubectl -n crossplane-system logs <name-of-ionoscloud-provider-pod>
```

For more information, see [<mark style="color:blue;">Crossplane Logs</mark>](https://docs.crossplane.io/knowledge-base/guides/troubleshoot/#crossplane-logs).

## Testing

Crossplane Provider IONOS Cloud has end-to-end integration tests for the resources supported. For more information, see [<mark style="color:blue;">CI [Weekly]</mark>](https://github.com/ionos-cloud/crossplane-provider-ionoscloud/actions/workflows/ci-weekly.yml).

## Releases

Releases can be made on Crossplane Provider IONOS Cloud via tags or manual action of the CD workflow. The CD workflow
will test and release the images. It will release images for controller and provider, with the following two tags: 
  * ``latest`` 
  * ``corresponding release``

## Conclusion

The main advantages of the Crossplane Provider IONOS Cloud are as follows:

* **Provisioning** resources in IONOS Cloud from a Kubernetes Cluster using (Custom Resource Definitions) CRDs.
* Maintaining a **healthy** setup using controller and reconciling loops.
* It can be installed on a **Crossplane Control Plane** and add new functionality for the users along with other Cloud
  Providers.

{% hint style="info" %}
**Note:** To contribute or provide feedback, you can create an [<mark style="color:blue;">issue</mark>](https://github.com/ionos-cloud/crossplane-provider-ionoscloud/issues) directly.
{% endhint %}