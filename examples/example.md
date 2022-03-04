# Crossplane Provider IONOS Cloud - Example (PoC)

## Overview

Crossplane allows the user to manage infrastructure directly from Kubernetes. Crossplane extends a Kubernetes cluster to support orchestrating any infrastructure or managed service.
Providers extend Crossplane to enable infrastructure resource provisioning of specific API. 

Crossplane Provider IONOS Cloud contains a Controller and Custom Resource Definitions(CRDs). The CRDs are defined in sync with the API and contain the desired state, the Controller has a reconcile loop, and it constantly compares the desired state vs actual state and takes action to reach the desired state. Using the SDK Go, the controller performs CRUD operations and resources are managed in IONOS Cloud. See [diagram](diagram.png).

In this Proof of Concept of the IONOS Cloud Provider, we will create a DBaaS Postgres Cluster resource in the IONOS Cloud.

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

Create an Image Pull Secret with your credentials, to be able to pull the Crossplane provider packages from the Github registry:

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

Output:

```bash
clusters.dbaas.postgres.ionoscloud.crossplane.io           2022-02-02T08:01:41Z
providerconfigs.ionoscloud.crossplane.io                   2022-02-02T08:01:41Z
providerconfigusages.ionoscloud.crossplane.io              2022-02-02T08:01:41Z
```

Next, we will create a Custom Resource(CR) of type `clusters.dbaas.postgres.ionoscloud.crossplane.io` in order to provision a DBaaS Postgres Cluster in the IONOS Cloud.

### Create a resource in IONOS Cloud

❗ Before running the next command, make sure to **update** the values in the `examples/ionoscloud/dbaas-postgres/cluster.yaml` file. Look for `spec.forProvider` fields. 
It is required to specify the Datacenter (via ID or via reference), Lan (via ID or via reference), CIDR, and location(in sync with the Datacenter) and credentials for the database user.

1. **[CREATE]** Create a datacenter CR, a lan CR and a cluster CR - using the next command:

```bash
kubectl apply -f examples/ionoscloud/dbaas-postgres/cluster.yaml
```

Check if the cluster CR created is _synced_ and _ready_:

```bash
kubectl get clusters
```

Output:

```bash
NAME       READY   SYNCED   CLUSTER ID                            STATE      AGE
example    True    True     9b25ecab-83fe-11ec-8d97-828542a828c7  AVAILABLE  93m
```

For more details, use:

```bash
kubectl get clusters -o wide
```

The external-name of the CR is the Cluster ID from IONOS Cloud. The cluster CR will be marked as ready when the cluster is in available state (subject of change).

You can check if the DBaaS Postgres Cluster was created in the IONOS Cloud:

- using `ionosctl` (one of the latest [v6 versions](https://github.com/ionos-cloud/ionosctl/releases/tag/v6.1.0)), you can run:

```bash
ionosctl dbaas postgres cluster list
```

Output:

```bash
ClusterId                              DisplayName   Location   DatacenterId                           LanId   Cidr               Instances   State
9b25ecab-83fe-11ec-8d97-828542a828c7   testDemo      de/txl     21d8fd28-5d62-43e9-a67b-68e52dac8885   1       192.168.1.100/24   1           AVAILABLE
```

- in DCD: go to [DCD Manager](https://dcd.ionos.com/latest/?dbaas=true) to `Manager Resources>Database Manager>Postgres Clusters`

2. **[UPDATE]** If you want to update the cluster CR created, update values from the `examples/ionoscloud/dbaas-postgres/cluster.yaml` file and use the following command:

```bash
kubectl apply -f examples/ionoscloud/dbaas-postgres/cluster.yaml
```

The updates applied should be updated in the external resource in IONOS Cloud.

3. **[DELETE]** If you want to delete the cluster CR created (named `example`), use the following command:

```bash
kubectl delete cluster example
```

This should trigger the destroying of the DBaaS Postgres Cluster.

⚠️Make sure to delete the DBaaS Postgres Cluster _before_ deleting the datacenter or the lan used in the Cluster's connection!

## Cleanup

```bash
kind delete cluster --name crossplane-example
```

DONE 🎉
