![Alt text](.github/IONOS.CLOUD.BLU.svg?raw=true "Title")

# Crossplane Provider IONOS Cloud

## Overview

Crossplane allows the user to manage infrastructure directly from Kubernetes. Crossplane extends a Kubernetes cluster to support orchestrating any infrastructure or managed service.
Providers extend Crossplane to enable infrastructure resource provisioning of specific API.

This `crossplane-provider-ionoscloud` repository is the Crossplane infrastructure provider for IONOS Cloud Services. The provider can be installed into a Crossplane control plane and adds the following new functionality:

* Custom Resource Definitions (CRDs) that model IONOS Cloud infrastructure and services
* Controllers to provision these resources in IONOS Cloud based on the users desired state captured in CRDs they create
* Implementations of Crossplane's portable resource abstractions, enabling IONOS Cloud resources to fulfill a user's general need for cloud services

## Getting Started and Documentation

For getting started, check out this [example](examples/example.md) for provisioning a DBaaS Postgres Cluster in IONOS Cloud.

## Contributing

`crossplane-provider-ionoscloud` is a community driven project and we welcome contributions.

## Report a Bug

For filing bugs, suggesting improvements, or requesting new features, please open an [issue](https://github.com/ionos-cloud/crossplane-provider-ionoscloud/issues).

## Licensing

crossplane-provider-ionoscloud is under the [Apache 2.0 License](LICENSE).
