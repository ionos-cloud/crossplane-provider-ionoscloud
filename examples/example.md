# Crossplane Provider IONOS Cloud - Example (PoC)

## Table of Contents

1. [Overview](#overview)
2. [Prerequisites](#prerequisites)
3. [Setup Crossplane Provider IONOS Cloud](#setup-crossplane-provider-ionos-cloud)
4. [Provision Resources](#provision-resources-in-ionos-cloud)
    1. [DBaaS Postgres Resources](#dbaas-postgres-resources)
    2. [Compute Engine Resources](#compute-engine-resources)
5. [Cleanup](#cleanup)
    1. [Uninstall the Provider](#uninstall-the-provider)
    2. [Uninstall K8s Cluster](#uninstall-k8s-cluster)
6. [Conclusion](#conclusion)

## Overview

Crossplane allows the user to manage infrastructure directly from Kubernetes. Crossplane extends a Kubernetes cluster to
support orchestrating any infrastructure or managed service. Providers extend Crossplane to enable infrastructure
resource provisioning of specific API.

Crossplane Provider IONOS Cloud contains a Controller and Custom Resource Definitions(CRDs). The CRDs are defined in
sync with the API and contain the desired state, the Controller has a reconcile loop, and it constantly compares the
desired state vs actual state and takes action to reach the desired state. Using the SDK Go, the controller performs
CRUD operations and resources are managed in IONOS Cloud. See [diagram](diagram.png).

In this Proof of Concept of the IONOS Cloud Provider, we will create a DBaaS Postgres Cluster resource in IONOS Cloud.

## Prerequisites

List of prerequisites:

* A Kubernetes implementation like [`kind`](https://kind.sigs.k8s.io/)
* [Kubectl](https://kubernetes.io/docs/tasks/tools/#kubectl)
* [Helm](https://helm.sh/docs/intro/install/)
* Docker
* Credentials to access IONOS Cloud
* Clone this repository locally to be able to run examples

Check prerequisites:

- K8s (in case of using kind):

```bash
kind version
```

- credentials:

```bash
export IONOS_USERNAME=xxx
export IONOS_PASSWORD=xxx
export BASE64_PW=$(echo -n "${IONOS_PASSWORD}" | base64)
```

- clone this repository locally:

```bash
git clone https://github.com/ionos-cloud/crossplane-provider-ionoscloud.git
cd crossplane-provider-ionoscloud
```

## Setup Crossplane Provider IONOS Cloud

1. Create a K8s cluster (in case of using kind):

```bash
kind create cluster --name crossplane-example
kubectl config use-context kind-crossplane-example
```

2. Create namespace for the crossplane ecosystem:

```bash
kubectl create namespace crossplane-system
```

3. Install crossplane via helm:

```bash
helm repo add crossplane-stable https://charts.crossplane.io/stable
helm repo update
helm install crossplane --namespace crossplane-system crossplane-stable/crossplane
```

4. Create CRDs:

```bash
kubectl apply -f package/crds/ -R
```

> Note: Before continuing, you can check if `kubectl get providers` will recognize the CRDs of type `providers`. The command should return `No resources found`.

5. Install provider config, for credentials:

```bash
export BASE64_PW=$(echo -n "${IONOS_PASSWORD}" | base64)
kubectl create secret generic --namespace crossplane-system example-provider-secret --from-literal=credentials="{\"user\":\"${IONOS_USERNAME}\",\"password\":\"${BASE64_PW}\"}"
kubectl apply -f examples/provider/config.yaml
```

6. Install Crossplane Provider IONOS Cloud

Create an Image Pull Secret with your credentials, to be able to pull the Crossplane provider packages from the Github
registry:

```bash
kubectl create secret --namespace crossplane-system docker-registry package-pull --docker-server ghcr.io --docker-username $GITHUB_USERNAME --docker-password $GITHUB_PERSONAL_ACCESSTOKEN
```

Install Crossplane Provider IONOS Cloud:

```bash
kubectl apply -f examples/provider/install-provider.yaml
```

You can install other providers (in this example, helm & kubernetes):

```bash
kubectl apply --namespace crossplane-system -f examples/providers/other-providers.yaml
```

7. Check if the Crossplane Provider IONOS Cloud is _installed_ and _healthy_:

```bash
kubectl get providers
```

You should be able to see pods running in the `crossplane-system` namespace, for each provider installed:

> Hint: by running `kubectl get pods -A`, you are able to see all existing pods from all namespaces.

```bash
kubectl get pods -n crossplane-system 
```

Output:

```bash
NAME                                                READY   STATUS    RESTARTS   AGE
crossplane-5b6896bb4c-nq5tl                         1/1     Running   0          66m
crossplane-rbac-manager-7874897d59-skdtt            1/1     Running   0          66m
provider-helm-a7f79daa3799-78d5959d6d-rktfs         1/1     Running   0          65m
provider-ionos-cf2fec81b474-54f5d7ddd4-w9w9h        1/1     Running   0          66m
provider-kubernetes-df601dea646a-84f7d6db54-t5dn5   1/1     Running   0          65m
```

Check CRDs:

```bash
kubectl get crds | grep ionoscloud
```

A CRD named `clusters.dbaas.postgres.ionoscloud.crossplane.io` should be displayed in the output.

Next, we will create a Custom Resource(CR) of type `clusters.dbaas.postgres.ionoscloud.crossplane.io` in order to
provision a DBaaS Postgres Cluster in the IONOS Cloud.

## Provision Resources in IONOS Cloud

### DBaaS Postgres Resources

For the DBaaS Postgres Service, there is only Cluster resource available into the Crossplane Provider IONOS Cloud.

❗ Before running the next command, make sure to **update** the values in
the `examples/ionoscloud/dbaas/postgres-cluster.yaml` file. Look for `spec.forProvider` fields. It is required to
specify the Datacenter (via ID or via reference), Lan (via ID or via reference), CIDR, and location(in sync with the
Datacenter) and also credentials for the database user.

1. **[CREATE]** Create a datacenter CR, a lan CR and a cluster CR - using the next command:

```bash
kubectl apply -f examples/ionoscloud/dbaas/postgres-cluster.yaml
```

Check if the Postgres Cluster CR created is _synced_ and _ready_:

```bash
kubectl get postgresclusters
```

Output:

```bash
NAME       READY   SYNCED   CLUSTER ID                            STATE      AGE
example    True    True     9b25ecab-83fe-11ec-8d97-828542a828c7  AVAILABLE  93m
```

For more details, use:

```bash
kubectl get postgresclusters -o wide
```

The external-name of the CR is the Cluster ID from IONOS Cloud. The cluster CR will be marked as ready when the cluster
is in available state (subject of change).

You can check if the DBaaS Postgres Cluster was created in the IONOS Cloud:

- using `ionosctl` (one of the latest [v6 versions](https://github.com/ionos-cloud/ionosctl/releases/tag/v6.1.0)), you
  can run:

```bash
ionosctl dbaas postgres cluster list
```

Output:

```bash
ClusterId                              DisplayName   Location   DatacenterId                           LanId   Cidr               Instances   State
9b25ecab-83fe-11ec-8d97-828542a828c7   testDemo      de/txl     21d8fd28-5d62-43e9-a67b-68e52dac8885   1       192.168.1.100/24   1           AVAILABLE
```

- in DCD: go to [DCD Manager](https://dcd.ionos.com/latest/?dbaas=true)
  to `Manager Resources>Database Manager>Postgres Clusters`

2. **[UPDATE]** If you want to update the cluster CR created, update values from
   the `examples/ionoscloud/dbaas/postgres-cluster.yaml` file and use the following command:

```bash
kubectl apply -f examples/ionoscloud/dbaas/postgres-cluster.yaml
```

The updates applied should be updated in the external resource in IONOS Cloud.

3. **[DELETE]** If you want to delete the cluster CR created (named `example`), use the following command:

```bash
kubectl delete postgrescluster example
```

This should trigger the destroying of the DBaaS Postgres Cluster.

⚠️ Make sure to delete the DBaaS Postgres Cluster **before** deleting the datacenter or the lan used in the Cluster's
connection!

Delete the lan and datacenter CRs:

```bash
kubectl delete lan examplelan
kubectl delete datacenter example
```

Or you can use the following command (not recommended for this particular case - it might delete the datacenter before
the cluster):

```bash
kubectl delete -f examples/ionoscloud/dbaas/postgres-cluster.yaml
```

### Compute Engine Resources

Before running the following commands, you can update the examples with the desired specifications. Keep in mind that
the Custom Resources(CRs) will manage corresponding external resources on IONOS Cloud.

Check the following tables for available commands:

<details >
<summary title="Click to toggle">See <b>CREATE/UPDATE/DELETE</b> Custom Resources Commands </summary>

| CUSTOM RESOURCE | CREATE/UPDATE | DELETE |
| --- | --- | --- |
| IPBlock | <pre lang="bash">kubectl apply -f examples/ionoscloud/compute/ipblock.yaml</pre> | <pre lang="bash">kubectl delete -f examples/ionoscloud/compute/ipblock.yaml</pre> | 
| Datacenter | <pre lang="bash">kubectl apply -f examples/ionoscloud/compute/datacenter.yaml</pre> | <pre lang="bash">kubectl delete -f examples/ionoscloud/compute/datacenter.yaml</pre> | 
| Server | <pre lang="bash">kubectl apply -f examples/ionoscloud/compute/server.yaml</pre> | <pre lang="bash">kubectl delete -f examples/ionoscloud/compute/server.yaml</pre> | 
| Volume | <pre lang="bash">kubectl apply -f examples/ionoscloud/compute/volume.yaml</pre> | <pre lang="bash">kubectl delete -f examples/ionoscloud/compute/volume.yaml</pre> | 
| Lan | <pre lang="bash">kubectl apply -f examples/ionoscloud/compute/lan.yaml</pre> | <pre lang="bash">kubectl delete -f examples/ionoscloud/compute/lan.yaml</pre> | 
| NIC | <pre lang="bash">kubectl apply -f examples/ionoscloud/compute/nic.yaml</pre> | <pre lang="bash">kubectl delete -f examples/ionoscloud/compute/nic.yaml</pre> | 
| FirewallRule | <pre lang="bash">kubectl apply -f examples/ionoscloud/compute/firewallrule.yaml</pre> | <pre lang="bash">kubectl delete -f examples/ionoscloud/compute/firewallrule.yaml</pre> | 
| IPFailover | <pre lang="bash">kubectl apply -f examples/ionoscloud/compute/ipfailover.yaml</pre> | <pre lang="bash">kubectl delete -f examples/ionoscloud/compute/ipfailover.yaml</pre> | 

</details>

<details >
<summary title="Click to toggle">See <b>GET</b> Custom Resources Commands </summary>

| CUSTOM RESOURCE | GET | GET MORE DETAILS | JSON OUTPUT |
| --- | --- | --- | --- | 
| IPBlock | <pre lang="bash">kubectl get ipblocks</pre> | <pre lang="bash">kubectl get ipblocks -o wide</pre> | <pre lang="bash">kubectl get ipblocks -o json</pre> | 
| Datacenter | <pre lang="bash">kubectl get datacenters</pre> | <pre lang="bash">kubectl get datacenters -o wide</pre> | <pre lang="bash">kubectl get datacenters -o json</pre> | 
| Server | <pre lang="bash">kubectl get servers</pre> | <pre lang="bash">kubectl get servers -o wide</pre> | <pre lang="bash">kubectl get servers -o json</pre> | 
| Volume | <pre lang="bash">kubectl get volumes</pre> | <pre lang="bash">kubectl get volumes -o wide</pre> | <pre lang="bash">kubectl get volumes -o json</pre> | 
| Lan | <pre lang="bash">kubectl get lans</pre> | <pre lang="bash">kubectl get lans -o wide</pre> | <pre lang="bash">kubectl get lans -o json</pre> | 
| NIC | <pre lang="bash">kubectl get nics</pre> | <pre lang="bash">kubectl get nics -o wide</pre> | <pre lang="bash">kubectl get nics -o json</pre> | 
| FirewallRule | <pre lang="bash">kubectl get firewallrules</pre> | <pre lang="bash">kubectl get firewallrules -o wide</pre> | <pre lang="bash">kubectl get firewallrules -o json</pre> | 
| IPFailover | <pre lang="bash">kubectl get ipfailovers</pre> | <pre lang="bash">kubectl get ipfailovers -o wide</pre> | <pre lang="bash">kubectl get ipfailovers -o json</pre> | 

</details>

_Notes_:

1. The `crossplane-provider-ionoscloud` controller waits for API requests to be DONE, for IONOS Cloud Compute Engine
   resources, and it checks for the state of resources.
2. Kubernetes Controllers main objective is to keep the system into the desired state - so if an external resource is
   deleted (using other tools: e.g. [DCD](https://dcd.ionos.com/latest/)
   , [ionosctl](https://github.com/ionos-cloud/ionosctl)), the `crossplane-provider-ionoscloud` controller will recreate
   the resource automatically.
3. JSON Output on `kubectl get` commands can be useful in checking status messages.

## Cleanup

### Uninstall the Provider

After deleting all resources, it is safe to uninstall the Crossplane Provider IONOS Cloud.

Make sure you delete the `ProviderConfig` **before** deleting the `Provider` (more
details [here](https://crossplane.io/docs/v1.6/reference/uninstall.html#uninstall-packages)):

```bash
kubectl delete -f examples/provider/config.yaml
```

Now it is safe to delete also the `Provider` (the `ProviderRevision` will be deleted automatically):

```bash
kubectl delete -f examples/provider/install-provider.yaml
```

### Uninstall K8s Cluster

Use the following command to delete the k8s cluster:

```bash
kind delete cluster --name crossplane-example
```

## Conclusion

Main advantages of the Crossplane Provider IONOS Cloud are:

- **provisioning** resources in IONOS Cloud from a Kubernetes Cluster - using CRDs (Custom Resource Definitions);
- maintaining a **healthy** setup using controller and reconciling loops;
- can be installed on a **Crossplane control plane** and add new functionality for the user along with other Cloud
  Providers.

There is always room for improvements, and we welcome feedback and contributions. Feel free to open
an [issue](https://github.com/ionos-cloud/crossplane-provider-ionoscloud/issues) or PR with your idea!
