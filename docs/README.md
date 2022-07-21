# Crossplane Provider IONOS Cloud

## Overview

Crossplane Provider for IONOS Cloud gives the ability to manage IONOS Cloud infrastructure directly from Kubernetes.
Crossplane extends a Kubernetes cluster to support orchestrating any infrastructure or managed service. Providers extend
Crossplane to enable infrastructure resource provisioning of specific API.

The [crossplane-provider-ionoscloud](https://github.com/ionos-cloud/crossplane-provider-ionoscloud) repository is the
Crossplane infrastructure provider for IONOS Cloud. The provider that is built from the source code from the repository
can be installed into a Crossplane control plane and adds the following new functionality:

* Custom Resource Definitions (CRDs) that model IONOS Cloud infrastructure and services (e.g. Compute Engine,
  Kubernetes, Database As a Service Postgres, etc.)
* Controllers to provision these resources in IONOS Cloud based on the users desired state captured in CRDs they create
* Implementations of Crossplane's portable resource abstractions, enabling IONOS Cloud resources to fulfill a user's
  general need for cloud services

## Getting Started and Documentation

For getting started with Crossplane usage and concepts, see the
official [Documentation](https://crossplane.io/docs/v1.7/).

For getting started with Crossplane Provider for IONOS Cloud, check out this
step-by-step [guide](../examples/example.md) which provides details about the provisioning of a Postgres Cluster in
IONOS Cloud.

## Prerequisites

To use the Crossplane Provider IONOS Cloud you will need an IONOS Cloud account, the same account you may use
with [DCD](https://dcd.ionos.com/latest/) or other config management tools.

Make sure you have a Kubernetes cluster
and [installed a Self-Hosted Crossplane](https://crossplane.github.io/docs/v1.7/) into a namespace
called `crossplane-system`.

In the examples [guide](../examples/example.md), you can find information of how to install a Kubernetes Cluster
locally (using kind or other lightweight Kubernetes) and Crossplane.

## Authentication on IONOS Cloud

Crossplane Provider for IONOS Cloud requires credentials to be provided in order to authenticate to the IONOS Cloud
APIs. This can be done using a base64-encoded static credentials in a Kubernetes `Secret`.

### Environment Variables

Crossplane Provider IONOS Cloud uses a `ProviderConfig` in order to setup credentials via `Secrets`. You can use
environments variables when creating the `ProviderConfig` resource.

| Environment Variable | Description                                                                                                                                                                                                                     |
|----------------------|---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|
| `IONOS_USERNAME`     | Specify the username used to login, to authenticate against the IONOS Cloud API                                                                                                                                                 | 
| `IONOS_PASSWORD`     | Specify the password used to login, to authenticate against the IONOS Cloud API                                                                                                                                                 | 
| `IONOS_TOKEN`        | Specify the token used to login, if a token is being used instead of username and password                                                                                                                                      |
| `IONOS_API_URL`      | Specify the API URL. It will overwrite the API endpoint default value `api.ionos.com`. Note: the host URL does not contain the `/cloudapi/v6` path, so it should _ not_ be included in the `IONOS_API_URL` environment variable |

### Create Provider Secret

- Using username and password:

```bash
export BASE64_PW=$(echo -n "${IONOS_PASSWORD}" | base64)
kubectl create secret generic --namespace crossplane-system example-provider-secret --from-literal=credentials="{\"user\":\"${IONOS_USERNAME}\",\"password\":\"${BASE64_PW}\"}"
```

- Using token:

```bash
kubectl create secret generic --namespace crossplane-system example-provider-secret --from-literal=credentials="{\"token\":\"${IONOS_TOKEN}\"}"
```

_Note_: You can overwrite the default IONOS Cloud API endpoint, by setting the following option in credentials
struct: `credentials="{\"host_url\":\"${IONOS_API_URL}\"}"`.

_Note_: You can also set the `IONOS_API_URL` environment variable in the `ControllerConfig` of the provider globally for
all resources. The following snipped shows how to set it globally in the ControllerConfig:

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

### Configure the Provider

We will create the following `ProviderConfig` to configure credentials for Crossplane Provider for IONOS Cloud:

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

### Install Provider

Install Provider using:

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

Check if the Crossplane Provider IONOS Cloud is _installed_ and _healthy_:

```bash
kubectl get providers
```

You should be able to see pods running in the `crossplane-system` namespace:

```bash
kubectl get pods -n crossplane-system 
```

## Provision Resources on IONOS Cloud

Now that you have the IONOS Cloud Provider configured, you can provision resources on IONOS Cloud directly from your
Kubernetes Cluster, using Custom Resource Definitions(CRDs). All CRDs are _Cluster Scoped_ (not being created in only
one specific namespace, but on the entire cluster).

Check below to see an up-to-date list of the CRDs and corresponding IONOS Cloud Resources that Crossplane Provider IONOS
Cloud supports as Managed Resources:

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

See more details [here](api/compute-engine).

### Application Load Balancer Managed Resources

#### Application Load Balancer Resources Custom Resource Definitions

| RESOURCES IN IONOS CLOUD | CUSTOM RESOURCE DEFINITION                             |
|--------------------------|--------------------------------------------------------|
| ApplicationLoadBalancer  | `applicationloadbalancer.alb.ionoscloud.crossplane.io` |
| ForwardingRule           | `forwardingrule.alb.ionoscloud.crossplane.io`          |
| TargetGroup              | `targetgroup.alb.ionoscloud.crossplane.io`             |

See more details [here](api/application-load-balancer).

### Kubernetes Managed Resources

#### Kubernetes Resources Custom Resource Definitions

| RESOURCES IN IONOS CLOUD | CUSTOM RESOURCE DEFINITION               |
|--------------------------|------------------------------------------|
| K8s Clusters             | `clusters.k8s.ionoscloud.crossplane.io`  |
| K8s NodePools            | `nodepools.k8s.ionoscloud.crossplane.io` |

See more details [here](api/managed-kubernetes).

### Backup Managed Resources

#### Backup Resources Custom Resource Definitions

| RESOURCES IN IONOS CLOUD | CUSTOM RESOURCE DEFINITION                    |
|--------------------------|-----------------------------------------------|
| BackupUnit               | `backupunits.backup.ionoscloud.crossplane.io` |

See more details [here](api/managed-backup).

### DBaaS Postgres Managed Resources

#### DBaaS Postgres Resources Custom Resource Definitions

| RESOURCES IN IONOS CLOUD | CUSTOM RESOURCE DEFINITION                        |
|--------------------------|---------------------------------------------------|
| DBaaS Postgres Clusters  | `postgresclusters.dbaas.ionoscloud.crossplane.io` |

See more details [here](api/database-as-a-service).

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

## Name Uniqueness Support for IONOS Cloud Resources

The Crossplane Provider IONOS Cloud has support for `--unique-names` flag, in order to enable name uniqueness support
for IONOS Cloud Resources.

You can create a `ControllerConfig`:

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

And reference it from the `Provider`:

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

## Debug Mode

### Provider Logs

The Crossplane Provider IONOS Cloud has support for `--debug` flag.

You can create a `ControllerConfig`:

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

And reference it from the `Provider`:

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

In order to see logs of the Crossplane Provider IONOS Cloud controller's pod, use:

```bash
kubectl -n crossplane-system logs <name-of-ionoscloud-provider-pod>
```

More details [here](https://crossplane.github.io/docs/v1.7/reference/troubleshoot.html#provider-logs).

## Testing

Crossplane Provider IONOS Cloud has end-to-end integration tests for the resources supported. See more
details [here](https://github.com/ionos-cloud/crossplane-provider-ionoscloud/actions/workflows/ci-daily.yml).

## Releases

Releases can be made on Crossplane Provider IONOS Cloud via tags or manual action of the CD workflow. The CD workflow
will test and release the images. It will release images for controller and provider, with 2 tags each: `latest` and the
corresponding release tag.

## Conclusion

Main advantages of the Crossplane Provider IONOS Cloud are:

- **provisioning** resources in IONOS Cloud from a Kubernetes Cluster - using CRDs (Custom Resource Definitions);
- maintaining a **healthy** setup using controller and reconciling loops;
- can be installed on a **Crossplane Control Plane** and add new functionality for the user along with other Cloud
  Providers.

There is always room for improvements, and we welcome feedback and contributions. Feel free to open
an [issue](https://github.com/ionos-cloud/crossplane-provider-ionoscloud/issues) or PR with your idea.
