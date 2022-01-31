![Alt text](.github/IONOS.CLOUD.BLU.svg?raw=true "Title")

# Crossplane Provider IONOS Cloud

## Overview

This `crossplane-provider-ionoscloud` repository is the Crossplane infrastructure provider for IONOS Cloud Services. The provider can be installed into a Crossplane control plane and adds the following new functionality:

* Custom Resource Definitions (CRDs) that model IONOS Cloud infrastructure and services
* Controllers to provision these resources in IONOS Cloud based on the users desired state captured in CRDs they create
* Implementations of Crossplane's portable resource abstractions, enabling IONOS Cloud resources to fulfill a user's general need for cloud services

## Getting Started and Documentation

Setup:

```text
kind version
```

Create cluster:

```text
kind create cluster --name crossplane-test
kubectl config use-context kind-crossplane-test
```

```bash
go run cmd/provider/main.go -d
```

Credentials:

```text
export IONOS_USERNAME=xxx
export IONOS_PASSWORD=xxx
export BASE64_PW=$(echo -n "${IONOS_PASSWORD}" | base64)
```

```text
# Create namespace
kubectl create namespace crossplane-system

# Install crossplane via helm
helm repo add crossplane-stable https://charts.crossplane.io/stable
helm repo update
helm install crossplane --namespace crossplane-system crossplane-stable/crossplane

# Create CRDs
kubectl apply -f package/crds/ -R

# Deploy other providers: provider-helm & provider-kubernetes
kubectl apply --namespace crossplane-system -f examples/providers/other-providers.yaml

# Create secret with credentials for IONOS Cloud
export BASE64_PW=$(echo -n "${IONOS_PASSWORD}" | base64)
kubectl create secret generic --namespace crossplane-system example-provider-secret --from-literal=credentials="{\"user\":\"${IONOS_USERNAME}\",\"password\":\"${BASE64_PW}\"}"

# Create config for credentials to IONOS CLOUD
kubectl apply -f examples/provider/config.yaml

# Create CR of type cluster
kubectl apply -f examples/ionoscloud/dbaas-postgres/cluster.yaml

# Get CRs
kubectl get clusters -A
```

Build image locally:

```text
make docker-build
```

Install provider locally:

```text
# Load image on current cluster
kind load docker-image docker.io/docker2801/provider-test:latest --name crossplane-test

# Install IONOS Crossplane Provider:
kubectl apply -f examples/provider/install-provider.yaml
```

Clean-up:

```text
kind delete cluster --name crossplane-test
```

## Contributing

crossplane-provider-ionoscloud is a community driven project and we welcome contributions.

## Report a Bug

For filing bugs, suggesting improvements, or requesting new features, please open an [issue](https://github.com/ionos-cloud/crossplane-provider-ionoscloud/issues).

## Licensing

crossplane-provider-ionoscloud is under the Apache 2.0 license.
