![Alt text](.github/IONOS.CLOUD.BLU.svg?sanitize=true&raw=true "Title")

# Crossplane Provider IONOS Cloud

## Overview

This `crossplane-provider-ionoscloud` repository is the Crossplane infrastructure provider for IONOS Cloud. The provider that is built from the source code from this repository can be installed into a Crossplane control plane and adds the following new functionality:

* Custom Resource Definitions (CRDs) that model IONOS Cloud infrastructure and services (e.g. Database As a Service Postgres, etc.)
* Controllers to provision these resources in IONOS Cloud based on the users desired state captured in CRDs they create
* Implementations of Crossplane's portable resource abstractions, enabling IONOS Cloud resources to fulfill a user's general need for cloud services

## Getting Started and Documentation

For getting started, check out this [example](examples/example.md) for provisioning a DBaaS Postgres Cluster in IONOS Cloud.

## Build

For building images, use:

```bash
make build
```

A version can be set via `$VERSION` variable. By running `make build VERSION=v0.x.x`, the specified version will be added into the `package/crossplane.yaml`.

For tagging images, use:

```bash
make docker.tag VERSION=v0.x.x
```

## Testing

For running end-to-end integration tests, use:

```bash
make e2e
```

If the images have a specific version, other than `latest`, this can be set via `make e2e VERSION=v0.x.x`.

## Releases

Releases can be made on Crossplane Provider IONOS Cloud via tags or manual action of the CD workflow. The CD workflow will test and release the images. It will release images for controller and provider, with 2 tags each: `latest` and the corresponding release tag.

## Contributing

`crossplane-provider-ionoscloud` is a community driven project and we welcome contributions.

## Report a Bug

For filing bugs, suggesting improvements, or requesting new features, please open an [issue](https://github.com/ionos-cloud/crossplane-provider-ionoscloud/issues).

## License

crossplane-provider-ionoscloud is under the [Apache 2.0 License](LICENSE).
