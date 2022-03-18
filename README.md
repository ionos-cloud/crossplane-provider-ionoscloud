![Alt text](.github/IONOS.CLOUD.BLU.svg?sanitize=true&raw=true "Title")

# Crossplane Provider IONOS Cloud

## Overview

This `crossplane-provider-ionoscloud` repository is the Crossplane infrastructure provider for IONOS Cloud. The provider
that is built from the source code from this repository can be installed into a Crossplane control plane and adds the
following new functionality:

* Custom Resource Definitions (CRDs) that model IONOS Cloud infrastructure and services (e.g. Database As a Service
  Postgres, Compute Engine, etc.)
* Controllers to provision these resources in IONOS Cloud based on the users desired state captured in CRDs they create
* Implementations of Crossplane's portable resource abstractions, enabling IONOS Cloud resources to fulfill a user's
  general need for cloud services

## Getting Started and Documentation

For getting started, check out this step-by-step [GUIDE](examples/example.md).

## Build

For building images, use:

```bash
make build
```

A version can be set via `$VERSION` variable. By running `make build VERSION=v0.x.x`, the specified version will be
added into the `package/crossplane.yaml`.

For tagging images, use:

```bash
make docker.tag VERSION=v0.x.x
```

## Provision Resources on IONOS Cloud

Check the following tables to see an updated list of the CRDs and corresponding IONOS Cloud Resources that Crossplane
Provider IONOS Cloud supports.

For more details on how to provision resources on IONOS Cloud using Crossplane Provider,
check: [Provision Resources in IONOS Cloud](examples/example.md#provision-resources-in-ionos-cloud).

<details >
<summary title="Click to toggle">See <b>DBaaS Postgres</b> Resources </summary>

| RESOURCES IN IONOS CLOUD | CUSTOM RESOURCE DEFINITION |
| --- | --- |
| DBaaS Postgres Clusters | `clusters.dbaas.postgres.ionoscloud.crossplane.io` |

</details>

For more information and commands on how to manage DBaaS Postgres resources on IONOS Cloud using Crossplane Provider,
see: [DBaaS Postgres](examples/example.md#dbaas-postgres-resources).

<details >
<summary title="Click to toggle">See <b>Compute Engine</b> Resources </summary>

| RESOURCES IN IONOS CLOUD | CUSTOM RESOURCE DEFINITION |
| --- | --- |
| IPBlocks | `ipblocks.compute.ionoscloud.crossplane.io` |
| Datacenters | `datacenters.compute.ionoscloud.crossplane.io` |
| Servers | `servers.compute.ionoscloud.crossplane.io` |
| Volumes | `volumes.compute.ionoscloud.crossplane.io` |
| Lans | `lans.compute.ionoscloud.crossplane.io` |
| NICs | `nics.compute.ionoscloud.crossplane.io` |
| FirewallRules | `firewallrules.compute.ionoscloud.crossplane.io` |
| IPFailovers | `ipfailovers.compute.ionoscloud.crossplane.io` |

</details>

For more information and commands on how to manage Compute Engine resources on IONOS Cloud using Crossplane Provider,
see: [Compute Engine Resources](examples/example.md#compute-engine-resources).

### References

References are used in order to reference other resources on which the new created resources are dependent. Using
referenced resources, the user can create for example, a datacenter and a lan using one command, without to manually
update the lan CR specification file with the required datacenter ID.

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

## Testing

Crossplane Provider IONOS Cloud has end-to-end integration tests for the resources supported.

For running end-to-end integration tests, use:

```bash
make e2e
```

If the images have a specific version, other than `latest`, this can be set via `make e2e VERSION=v0.x.x`.

## Debug Mode

### Provider Logs

The Crossplane Provider IONOS Cloud has support for `--debug` flag. You can create
a [ControllerConfig](examples/provider/debug-config.yaml) and reference it from
the [Provider](examples/provider/install-provider.yaml).

In order to see logs of the Crossplane Provider IONOS Cloud controller's pod, use:

```bash
kubectl -n crossplane-system logs <name-of-ionoscloud-provider-pod>
```

More details [here](https://negz.github.io/crossplane.github.io/docs/v1.4/reference/troubleshoot.html#provider-logs).

## Releases

Releases can be made on Crossplane Provider IONOS Cloud via tags or manual action of the CD workflow. The CD workflow
will test and release the images. It will release images for controller and provider, with 2 tags each: `latest` and the
corresponding release tag.

## Contributing

`crossplane-provider-ionoscloud` is a community driven project and we welcome contributions.

## Report a Bug

For filing bugs, suggesting improvements, or requesting new features, please open
an [issue](https://github.com/ionos-cloud/crossplane-provider-ionoscloud/issues).

## License

crossplane-provider-ionoscloud is under the [Apache 2.0 License](LICENSE).
