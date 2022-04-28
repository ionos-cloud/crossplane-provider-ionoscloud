![Alt text](.github/IONOS.CLOUD.BLU.svg?sanitize=true&raw=true "Title")

# Crossplane Provider IONOS Cloud

## Overview

This `crossplane-provider-ionoscloud` repository is the Crossplane infrastructure provider for IONOS Cloud. The provider
that is built from the source code from this repository can be installed into a Crossplane control plane and adds the
following new functionality:

* Custom Resource Definitions (CRDs) that model IONOS Cloud infrastructure and services (e.g. Database As a Service
  Postgres, Compute Engine, Kubernetes, etc.)
* Controllers to provision these resources in IONOS Cloud based on the users desired state captured in CRDs they create
* Implementations of Crossplane's portable resource abstractions, enabling IONOS Cloud resources to fulfill a user's
  general need for cloud services

## Getting Started and Documentation

For getting started with Crossplane Provider IONOS Cloud, check out this step-by-step [example](examples/example.md).

## Setup Crossplane Provider IONOS Cloud

In order to setup Crossplane Provider IONOS Cloud, see details
in [here](examples/example.md#setup-crossplane-provider-ionos-cloud).

## Authentication on IONOS Cloud

Crossplane Provider IONOS Cloud uses [ProviderConfig](examples/provider/config.yaml) in order to setup credentials via
secrets. You can use environments variables when creating the `ProviderConfig` resource.

| Environment Variable | Description                                                                                |
|----------------------|--------------------------------------------------------------------------------------------|
| `IONOS_USERNAME`     | Specify the username used to login, to authenticate against the IONOS Cloud API            | 
| `IONOS_PASSWORD`     | Specify the password used to login, to authenticate against the IONOS Cloud API            | 
| `IONOS_TOKEN`        | Specify the token used to login, if a token is being used instead of username and password |
| `IONOS_API_URL`      | Specify the API URL. It will overwrite the API endpoint default value `api.ionos.com`      |                                                                                                                                                                    |

More details about ProviderConfig and authentication [here](docs/README.md#authentication-on-ionos-cloud).

## Provision Resources on IONOS Cloud

Crossplane Provider IONOS Cloud Managed Resoures list is available [here](docs/RESOURCES.md).

## Build images

For building Docker images, use:

```bash
make build
```

A version can be set via `$VERSION` variable. By running `make build VERSION=v0.x.x`, the specified version will be
added into the `package/crossplane.yaml`.

For tagging Docker images, use:

```bash
make docker.tag VERSION=v0.x.x
```

## Usage

To run a K8s Cluster and install Crossplane:

```bash
make dev
```

To run e2e tests:

```bash
make e2e
```

To run linters on the code before opening a PR:

```bash
make reviewable
```

To clean up the K8s Cluster:

```bash
make dev-clean
```

To list all available options:

```bash
make help
```

## Testing

Crossplane Provider IONOS Cloud has end-to-end integration tests for the resources supported.

For running end-to-end integration tests locally, use:

```bash
make e2e
```

If the images have a specific version, other than `latest`, this can be set via `make e2e VERSION=v0.x.x`.

Daily workflows with all end-to-end integration tests are running using Github Actions. You can check their
status [here](https://github.com/ionos-cloud/crossplane-provider-ionoscloud/actions/workflows/ci-daily.yml).

## Releases

Releases can be made on Crossplane Provider IONOS Cloud via tags or manual action of the CD workflow. The CD workflow
will test and release the images. It will release images for controller and provider, with 2 tags each: `latest` and the
corresponding release tag.

## Contributing

`crossplane-provider-ionoscloud` is a community driven project and we welcome contributions. See the Crossplane
[Contributing](https://github.com/crossplane/crossplane/blob/master/CONTRIBUTING.md) guidelines to get started.

### Adding New Resource

New resources can be added by defining the required types in `apis` and the controllers `internal/controller/`.

To generate the CRDs YAML files run

```bash
make generate
```

## Report a Bug

For filing bugs, suggesting improvements, or requesting new features, please open
an [issue](https://github.com/ionos-cloud/crossplane-provider-ionoscloud/issues).

## Code of Conduct

`crossplane-provider-ionoscloud` adheres to the
same [Code of Conduct](https://github.com/crossplane/crossplane/blob/master/CODE_OF_CONDUCT.md) as the core Crossplane
project.

## License

crossplane-provider-ionoscloud is under the [Apache 2.0 License](LICENSE).
