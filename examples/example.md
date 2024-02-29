# Crossplane Provider IONOS Cloud Usage Example

## Overview

Crossplane allows the user to manage infrastructure directly from Kubernetes. Crossplane extends a Kubernetes cluster to
support orchestrating any infrastructure or managed service. Providers extend Crossplane to enable infrastructure
resource provisioning of specific APIs.

Crossplane Provider IONOS Cloud contains a Controller and Custom Resource Definitions(CRDs). The CRDs are defined in
sync with the API and contain the desired state. The Controller has a reconcile loop, and it constantly compares the
desired state vs the actual state and takes action to reach the desired state. Using the SDK Go, the Controller performs
CRUD operations and resources are managed in the IONOS Cloud.

In this Proof of Concept of the IONOS Cloud Provider, we will create a DBaaS Postgres cluster resource in the IONOS Cloud.

## Prerequisites

Ensure that you have the following:

* A Kubernetes implementation, such as [`kind`](https://kind.sigs.k8s.io/)
* [``Kubectl``](https://kubernetes.io/docs/tasks/tools/#kubectl)
* [``Helm``](https://helm.sh/docs/intro/install/)
* Docker
* Credentials to access IONOS Cloud
* Clone this repository locally to be able to run examples

### Check prerequisites

You can now check your prerequisites. To check K8s, in case of using ``kind``, run the following command:

```bash
kind version
```

To check the **credentials**, run the following command:

```bash
export IONOS_USERNAME=xxx
export IONOS_PASSWORD=xxx
export BASE64_PW=$(echo -n "${IONOS_PASSWORD}" | base64)
```

OR

```bash
export IONOS_TOKEN=xxx
```

To clone the repository locally, run the following command:

```bash
git clone https://github.com/ionos-cloud/crossplane-provider-ionoscloud.git
cd crossplane-provider-ionoscloud
```

## Set up Crossplane Provider IONOS Cloud

To set up Crossplane Provider IONOS Cloud, follow these steps:

* [<mark style="color:blue;">Create a K8s cluster (in case of using kind)</mark>](#create-a-k8s-cluster-in-case-of-using-kind)
* [<mark style="color:blue;">Create namespace for the crossplane ecosystem</mark>](#create-namespace-for-the-crossplane-ecosystem)
* [<mark style="color:blue;">Install crossplane via ``helm``</mark>](#install-crossplane-via-helm)
* [<mark style="color:blue;">Register CRDs into k8s cluster</mark>](#register-crds-into-k8s-cluster)
* [<mark style="color:blue;"> Install ProviderConfig for credentials</mark>](#install-providerconfig-for-credentials)
* [<mark style="color:blue;">Install Crossplane Provider IONOS Cloud</mark>](#install-crossplane-provider-ionos-cloud)

### Create a K8s cluster (in case of using kind)

To create a cluster in case of using ``kind``, run the following command:

```bash
kind create cluster --name crossplane-example
kubectl config use-context kind-crossplane-example
```

### Create namespace for the crossplane ecosystem

To create namespace for the crossplane ecosystem, run the following command:

```bash
kubectl create namespace crossplane-system
```

### Install Crossplane via helm

To create a cluster in case of using ``helm``, run the following command:

```bash
helm repo add crossplane-stable https://charts.crossplane.io/stable
helm repo update
helm install crossplane --namespace crossplane-system crossplane-stable/crossplane
```

### Register CRDs into k8s cluster

To register CRDs into k8s cluster, run the following command:

```bash
kubectl apply -f package/crds/ -R
```

{% hint style="info" %}
**Note:** Before continuing, you can check if ``kubectl get providers`` will recognize the CRDs of type ``providers``. The command should return: ``No resources found``.
{% endhint %}

### Install ProviderConfig for credentials

To install ``ProviderConfig`` for credentials, run the following commands:

  ```bash
  export BASE64_PW=$(echo -n "${IONOS_PASSWORD}" | base64)
  kubectl create secret generic --namespace crossplane-system example-provider-secret --from-literal=credentials="{\"user\":\"${IONOS_USERNAME}\",\"password\":\"${BASE64_PW}\"}"
  kubectl apply -f examples/provider/config.yaml
  ```

OR

  ```bash
  kubectl create secret generic --namespace crossplane-system example-provider-secret --from-literal=credentials="{\"token\":\"${IONOS_TOKEN}\"}"
  kubectl apply -f examples/provider/config.yaml
  ```

{% hint style="info" %}
**Note:**   You can overwrite the default IONOS Cloud API endpoint, by setting ``host_url`` to: ``--from-literal=credentials="{\"host_url\":\"${IONOS_API_URL}\"}"``.
{% endhint %}

### Install Crossplane Provider IONOS Cloud

To install Crossplane Provider IONOS Cloud, run the following command:

```bash
kubectl apply -f examples/provider/install-provider.yaml
```

You can install other providers; such as, ``helm`` and ``Kubernetes`` using:

```bash
kubectl apply --namespace crossplane-system -f examples/providers/other-providers.yaml
```

### Check the health of Crossplane Provider IONOS Cloud 

To check if the Crossplane Provider IONOS Cloud is installed and healthy, run the following command:

```bash
kubectl get providers
```

You should be able to see pods running in the ``crossplane-system`` namespace, for each provider installed. To see the existing pods from all namespaces, run: ``kubectl get pods -A``

Run the following command to see the pods:

```bash
kubectl get pods -n crossplane-system 
```

#### Output

```bash
NAME                                                READY   STATUS    RESTARTS   AGE
crossplane-5b6896bb4c-nq5tl                         1/1     Running   0          66m
crossplane-rbac-manager-7874897d59-skdtt            1/1     Running   0          66m
provider-helm-a7f79daa3799-78d5959d6d-rktfs         1/1     Running   0          65m
provider-ionos-cf2fec81b474-54f5d7ddd4-w9w9h        1/1     Running   0          66m
provider-kubernetes-df601dea646a-84f7d6db54-t5dn5   1/1     Running   0          65m
```

#### Check CRDs

To check the CRDs, run the following command:

```bash
kubectl get crds | grep ionoscloud
```

A CRD named ``postgresclusters.dbaas.ionoscloud.crossplane.io`` should be displayed in the output.

After that, you can create a Custom Resource (CR) of type ``postgresclusters.dbaas.ionoscloud.crossplane.io`` to
provision a DBaaS Postgres cluster in the IONOS Cloud.

### Provision DBaaS Postgres cluster

For the DBaaS Postgres service, there is only cluster resource available into the Crossplane Provider IONOS Cloud.

{% hint style="warning" %}
**Warning:** Before running the next command, make sure to **update** the values in
the ``examples/ionoscloud/dbaas/postgres-cluster.yaml`` file. Look for ``spec.forProvider`` fields. It is required to specify the datacenter (via ID or via reference),LAN (via ID or via reference), CIDR, and location(in sync with the datacenter) and also credentials for the database user.
{% endhint %}

### Create datacenter CR, LAN CR, and Postgres cluster CR

To ceate datacenter CR, LAN CR, and Postgres cluster CR, run the following command:

```bash
kubectl apply -f examples/ionoscloud/dbaas/postgres-cluster.yaml
```

### Get datacenter, LAN, and Postgres cluster CRs

To check if the Postgres cluster CR created is synced and ready, run the following command:

```bash
kubectl get postgresclusters
```

#### Output

```bash
NAME       READY   SYNCED   CLUSTER ID                            STATE      AGE
example    True    True     9b25ecab-83fe-11ec-8d97-828542a828c7  AVAILABLE  93m
```

To view more details, run the following command:

```bash
kubectl get postgresclusters -o wide
```

The ``external-name`` of the CR is the ``Cluster ID`` from IONOS Cloud. The cluster CR will be marked as ``ready`` when the cluster
is in available state.

You can check if the DBaaS Postgres cluster was created in the IONOS Cloud using [<mark style="color:blue;">``ionosctl`` latest versions</mark>](https://github.com/ionos-cloud/ionosctl/releases/tag/v6.1.0). Run the following command:

```bash
ionosctl dbaas postgres cluster list
```
#### Output

```bash
ClusterId                              DisplayName   Location   DatacenterId                           LanId   Cidr               Instances   State
9b25ecab-83fe-11ec-8d97-828542a828c7   testDemo      de/txl     21d8fd28-5d62-43e9-a67b-68e52dac8885   1       192.168.1.100/24   1           AVAILABLE
```

1. In the **DCD**, go to the **Menu** > **Databases** > **Postgres clusters**.

2. Check if the datacenter and LAN CRs are created using:

  ```bash
  kubectl get datacenters
  kubectl get lans
  ```

### Update datacenter, LAN and Postgres ccuster CRs

If you want to update the CRs created, update values from the ``examples/ionoscloud/dbaas/postgres-cluster.yaml``
file using the following command:

```bash
kubectl apply -f examples/ionoscloud/dbaas/postgres-cluster.yaml
```

The updates applied should be updated in the external resource in IONOS Cloud.

### Delete datacenter, LAN and Postgres cluster CRs

If you want to delete the ``example`` Postgres cluster CR created, use the following command:

```bash
kubectl delete postgrescluster example
```

This would trigger the destroying of the DBaaS Postgres cluster.

{% hint style="warning" %}
**Warning:** Make sure to delete the DBaaS Postgres cluster before deleting the datacenter or the LAN used in the Cluster's
connection.
{% endhint %}

Delete the LAN and datacenter CRs using:

```bash
kubectl delete lan examplelan
kubectl delete datacenter example
```

OR

You can use the following command 

```bash
kubectl delete -f examples/ionoscloud/dbaas/postgres-cluster.yaml
```
{% hint style="warning" %}
**Warning:** This command is not recommended and can be used for this particular case only. It might delete the datacenter before
the cluster.
{% endhint %}

### Summary

Refer to the following tables for DBaaS Postgres resources commands:


| **Custom Resource**        | **Create/Delete/Update**                                                          |
|------------------------|-----|
| DBaaS Postgres cluster | <pre lang="bash">kubectl apply -f examples/ionoscloud/dbaas/postgres-cluster.yaml</pre> | <pre lang="bash">kubectl delete -f examples/ionoscloud/dbaas/postgres-cluster.yaml</pre> |


| **Custom Resource**          | **GET**                                                 | **GET More Details**                                            | **JSON Output**                                                 |
|------------------------|-----------------------------------------------------|-------------------------------------------------------------|-------------------------------------------------------------|
| DBaaS Postgres cluster | <pre lang="bash">kubectl get postgresclusters</pre> | <pre lang="bash">kubectl get postgresclusters -o wide</pre> | <pre lang="bash">kubectl get postgresclusters -o json</pre> | 


For more information on all Managed Resources of Crossplane Provider IONOS Cloud, see [<mark style="color:blue;">Provision Resources on IONOS Cloud</mark>](../docs/README.md#provision-resources-on-ionos-cloud).

## Uninstallation

To uninstall, you need to follow these steps:

* [<mark style="color:blue;">Uninstall the Provider</mark>](#uninstall-the-provider)
* [<mark style="color:blue;">Uninstall K8s Cluster</mark>](#uninstall-k8s-cluster)

### Uninstall the Provider

After deleting all resources, it is safe to uninstall the Crossplane Provider IONOS Cloud. Run the following command:

```bash
kubectl delete -f examples/provider/config.yaml
```

{% hint style="info" %}
**Note:** Make sure you delete the ``ProviderConfig`` before deleting the ``Provider``. For more information, see [<mark style="color:blue;">Uninstall Crossplane</mark>](https://docs.crossplane.io/latest/software/uninstall/).
{% endhint %}

Now it is safe to delete also the ``Provider``. The ``ProviderRevision`` will be deleted automatically using:

```bash
kubectl delete -f examples/provider/install-provider.yaml
```

### Uninstall K8s cluster

Use the following command to delete the k8s cluster:

```bash
kind delete cluster --name crossplane-example
```

{% hint style="success" %}
**Result:** This way you can create and then delete the DBaaS Postgres cluster resource in the IONOS Cloud. 
{% endhint %}