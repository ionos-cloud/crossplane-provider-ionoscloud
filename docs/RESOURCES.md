# Crossplane Provider IONOS Cloud Managed Resources

## Introduction

Crossplane Provider IONOS Cloud contains a Controller and Custom Resource Definitions(CRDs). The CRDs are defined in
sync with the API and contain the desired state, the Controller has a reconcile loop, and it constantly compares the
desired state vs actual state and takes action to reach the desired state. Using the Go SDKs for the corresponding cloud
servies, the controller performs CRUD operations and resources are managed in IONOS Cloud.

This file contains an up-to-date list of the Managed Resources supported by Crossplane Provider IONOS Cloud.

## Provisioning Resources in IONOS Cloud

Before using the following commands for resources, make sure to follow the next recommendations:

- clone the [Github Repository](https://github.com/ionos-cloud/crossplane-provider-ionoscloud) locally to easily access
  the [examples](https://github.com/ionos-cloud/crossplane-provider-ionoscloud/tree/master/examples) folder(**Note**:
  all commands presented in this document should be run from the root of the `crossplane-provider-ionoscloud` directory)
  ;
- update the [examples](https://github.com/ionos-cloud/crossplane-provider-ionoscloud/tree/master/examples) provided
  accordingly with the desired specifications for your infrastructure.

### Compute Engine Managed Resources

#### Compute Engine Resources Custom Resource Definitions

| RESOURCES IN IONOS CLOUD | CUSTOM RESOURCE DEFINITION                       |
|--------------------------|--------------------------------------------------|
| IPBlocks                 | `ipblocks.compute.ionoscloud.crossplane.io`      |
| Datacenters              | `datacenters.compute.ionoscloud.crossplane.io`   |
| Servers                  | `servers.compute.ionoscloud.crossplane.io`       |
| Volumes                  | `volumes.compute.ionoscloud.crossplane.io`       |
| Lans                     | `lans.compute.ionoscloud.crossplane.io`          |
| NICs                     | `nics.compute.ionoscloud.crossplane.io`          |
| FirewallRules            | `firewallrules.compute.ionoscloud.crossplane.io` |
| IPFailovers              | `ipfailovers.compute.ionoscloud.crossplane.io`   |

#### Compute Engine Resources CREATE/UPDATE Custom Resources

| CUSTOM RESOURCE | CREATE/UPDATE                                                                         |
|-----------------|---------------------------------------------------------------------------------------|
| IPBlock         | <pre lang="bash">kubectl apply -f examples/ionoscloud/compute/ipblock.yaml</pre>      | 
| Datacenter      | <pre lang="bash">kubectl apply -f examples/ionoscloud/compute/datacenter.yaml</pre>   | 
| Server          | <pre lang="bash">kubectl apply -f examples/ionoscloud/compute/server.yaml</pre>       | 
| Volume          | <pre lang="bash">kubectl apply -f examples/ionoscloud/compute/volume.yaml</pre>       | 
| Lan             | <pre lang="bash">kubectl apply -f examples/ionoscloud/compute/lan.yaml</pre>          | 
| NIC             | <pre lang="bash">kubectl apply -f examples/ionoscloud/compute/nic.yaml</pre>          | 
| FirewallRule    | <pre lang="bash">kubectl apply -f examples/ionoscloud/compute/firewallrule.yaml</pre> | 
| IPFailover      | <pre lang="bash">kubectl apply -f examples/ionoscloud/compute/ipfailover.yaml</pre>   | 

#### Compute Engine Resources GET Custom Resources

| CUSTOM RESOURCE | GET                                              | GET MORE DETAILS                                         |
|-----------------|--------------------------------------------------|----------------------------------------------------------|
| IPBlock         | <pre lang="bash">kubectl get ipblocks</pre>      | <pre lang="bash">kubectl get ipblocks -o wide</pre>      |
| Datacenter      | <pre lang="bash">kubectl get datacenters</pre>   | <pre lang="bash">kubectl get datacenters -o wide</pre>   |
| Server          | <pre lang="bash">kubectl get servers</pre>       | <pre lang="bash">kubectl get servers -o wide</pre>       |
| Volume          | <pre lang="bash">kubectl get volumes</pre>       | <pre lang="bash">kubectl get volumes -o wide</pre>       |
| Lan             | <pre lang="bash">kubectl get lans</pre>          | <pre lang="bash">kubectl get lans -o wide</pre>          |
| NIC             | <pre lang="bash">kubectl get nics</pre>          | <pre lang="bash">kubectl get nics -o wide</pre>          |
| FirewallRule    | <pre lang="bash">kubectl get firewallrules</pre> | <pre lang="bash">kubectl get firewallrules -o wide</pre> |
| IPFailover      | <pre lang="bash">kubectl get ipfailovers</pre>   | <pre lang="bash">kubectl get ipfailovers -o wide</pre>   |

#### Compute Engine Resources DELETE Custom Resources

| CUSTOM RESOURCE | DELETE                                                                                 |
|-----------------|----------------------------------------------------------------------------------------|
| IPBlock         | <pre lang="bash">kubectl delete -f examples/ionoscloud/compute/ipblock.yaml</pre>      | 
| Datacenter      | <pre lang="bash">kubectl delete -f examples/ionoscloud/compute/datacenter.yaml</pre>   | 
| Server          | <pre lang="bash">kubectl delete -f examples/ionoscloud/compute/server.yaml</pre>       | 
| Volume          | <pre lang="bash">kubectl delete -f examples/ionoscloud/compute/volume.yaml</pre>       | 
| Lan             | <pre lang="bash">kubectl delete -f examples/ionoscloud/compute/lan.yaml</pre>          | 
| NIC             | <pre lang="bash">kubectl delete -f examples/ionoscloud/compute/nic.yaml</pre>          | 
| FirewallRule    | <pre lang="bash">kubectl delete -f examples/ionoscloud/compute/firewallrule.yaml</pre> | 
| IPFailover      | <pre lang="bash">kubectl delete -f examples/ionoscloud/compute/ipfailover.yaml</pre>   | 

Notes:

- The `crossplane-provider-ionoscloud` controller waits for API requests to be DONE, for IONOS Cloud Compute Engine
  resources, and it checks for the state of resources.
- Kubernetes Controllers main objective is to keep the system into the desired state - so if an external resource is
  updated or deleted (using other tools: e.g. [DCD](https://dcd.ionos.com/latest/)
  , [ionosctl](https://github.com/ionos-cloud/ionosctl)), the `crossplane-provider-ionoscloud` controller will recreate
  the resource automatically. Crossplane acts like it is the only source of truth for the resources provisioned via
  Managed Resources.

### Kubernetes Managed Resources

#### Kubernetes Resources Custom Resource Definitions

| RESOURCES IN IONOS CLOUD | CUSTOM RESOURCE DEFINITION               |
|--------------------------|------------------------------------------|
| K8s Clusters             | `clusters.k8s.ionoscloud.crossplane.io`  |
| K8s NodePools            | `nodepools.k8s.ionoscloud.crossplane.io` |

#### Kubernetes Resources CREATE/UPDATE Custom Resources

| CUSTOM RESOURCE | CREATE/UPDATE                                                                     |
|-----------------|-----------------------------------------------------------------------------------|
| K8s Cluster     | <pre lang="bash">kubectl apply -f examples/ionoscloud/k8s/k8s-cluster.yaml</pre>  | 
| K8s NodePool    | <pre lang="bash">kubectl apply -f examples/ionoscloud/k8s/k8s-nodepool.yaml</pre> |

#### Kubernetes Resources GET Custom Resources

| CUSTOM RESOURCE | GET                                                                       | GET MORE DETAILS                                                                  |
|-----------------|---------------------------------------------------------------------------|-----------------------------------------------------------------------------------|
| K8s Cluster     | <pre lang="bash">kubectl get clusters.k8s.ionoscloud.crossplane.io</pre>  | <pre lang="bash">kubectl get clusters.k8s.ionoscloud.crossplane.io -o wide</pre>  | 
| K8s NodePool    | <pre lang="bash">kubectl get nodepools.k8s.ionoscloud.crossplane.io</pre> | <pre lang="bash">kubectl get nodepools.k8s.ionoscloud.crossplane.io -o wide</pre> |

#### Kubernetes Resources DELETE Custom Resources

| CUSTOM RESOURCE | DELETE                                                                             |
|-----------------|------------------------------------------------------------------------------------|
| K8s Cluster     | <pre lang="bash">kubectl delete -f examples/ionoscloud/k8s/k8s-cluster.yaml</pre>  | 
| K8s NodePool    | <pre lang="bash">kubectl delete -f examples/ionoscloud/k8s/k8s-nodepool.yaml</pre> |

### DBaaS Postgres Managed Resources

#### DBaaS Postgres Resources Custom Resource Definitions

| RESOURCES IN IONOS CLOUD | CUSTOM RESOURCE DEFINITION                        |
|--------------------------|---------------------------------------------------|
| DBaaS Postgres Clusters  | `postgresclusters.dbaas.ionoscloud.crossplane.io` |

#### DBaaS Postgres Resources CREATE/UPDATE Custom Resources Commands

| RESOURCE               | CREATE/UPDATE                                                                           |
|------------------------|-----------------------------------------------------------------------------------------|
| DBaaS Postgres Cluster | <pre lang="bash">kubectl apply -f examples/ionoscloud/dbaas/postgres-cluster.yaml</pre> |

#### DBaaS Postgres Resources GET Custom Resources Commands

| RESOURCE               | GET                                                 | GET MORE DETAILS                                            |
|------------------------|-----------------------------------------------------|-------------------------------------------------------------|
| DBaaS Postgres Cluster | <pre lang="bash">kubectl get postgresclusters</pre> | <pre lang="bash">kubectl get postgresclusters -o wide</pre> |

#### DBaaS Postgres Resources DELETE Custom Resources Commands

| RESOURCE               | DELETE                                                                                   |
|------------------------|------------------------------------------------------------------------------------------|
| DBaaS Postgres Cluster | <pre lang="bash">kubectl delete -f examples/ionoscloud/dbaas/postgres-cluster.yaml</pre> |

## References

References are used in order to reference other resources on which the new resources are dependent. Using referenced
resources, the user can create for example, a datacenter and a lan using one command, without to manually update the lan
CR specification file with the required datacenter ID.

The references are resolved **only once**, when the resource is created, and the resolvers are generated
using [crossplane-tools](https://github.com/crossplane/crossplane-tools).

Example:

```yaml
datacenterConfig:
  datacenterIdRef:
    name: <datacenter_CR_name>
```

The user can set the datacenter ID directly, using:

```yaml
datacenterConfig:
  datacenterId: <datacenter_ID>
```

_Note_: If both the `datacenterId` and the `datacenterIdRef` fields are set, the `datacenterId` value has priority.

## Compositions and Claims

Composite Resources are designed to help you build your own platform and mix-and-match schemas for different providers.
You define the schema of your Composite Resource(XR) and teach Crossplane which Managed Resources(CRs or Custom
Resources) it should create when someone creates the XR.

### Steps

#### Define Composite Resources

First step is to define the `CompositeResourceDefinition` so that Crossplane knows which XRs you would like to create
and what fields that XRs should have. In the example provided, this is done in
the [definition file](../examples/composition/definition.yaml).

#### Configure Compositions

Next step is to teach Crossplane what to do when a Composite Resource is created. Compositions are linking an XR with
one or multiple CRs (ipblocks, postgresclusters, nodepools, clusters, etc). Basically, the user controls the CRs for
IONOS Cloud Resources via XRs: when an XR is created, updated or deleted, according to the Composition configured,
Crossplane will create, update, or delete CRs. In the example provided, this is done in
the [composition file](../examples/composition/composition.yaml).

#### Claim Composite Resources

After defining Composite Resources and configuring Compositions, the next step is to create Composite Resource Claims (
aka claims). A difference between and XRs and claims is that claims are namespaced scoped, while XRs are cluster scoped.
Also, an XR contains references to the CRs, while claim contains reference to the corresponding XR. In the example
provided, a claim is defined in the [claim file](../examples/composition/claim.yaml), while an XR
in [composite file](../examples/composition/composite.yaml).

### Example

An example for creating a Datacenter, a Kubernetes Cluster and a Kubernetes NodePool via Compositions and Claims can be
found [here](../examples/composition).

### More Details

More details about Composite Resources can be found here:

- [Composite Resources Concept](https://crossplane.io/docs/v1.7/concepts/composition.html)
- [Composite Resources Reference](https://crossplane.io/docs/v1.7/reference/composition.html)
