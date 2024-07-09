# Usage with Argocd

### Installation

##### Create the cluster and install Argocd
```bash
kind create cluster --name crossplane-argocd
kubectl config use-context crossplane-argocd
kubectl create namespace argocd
kubectl create namespace crossplane-system

helm repo add crossplane-stable https://charts.crossplane.io/stable
helm repo update
helm install crossplane --namespace crossplane-system charts/crossplane
helm install argo-cd --namespace argocd charts/argo-cd/
```

##### Wait for the argocd application controller pod to be ready, and retrieve the admin account password afterwards
Note: the initial admin account username is `admin`
#####
```bash
kubectl wait --for=condition=ready pods/argo-cd-argocd-application-controller-0 --timeout=10m --namespace argocd
kubectl get secret argocd-initial-admin-secret --namespace argocd -o jsonpath="{.data.password}" | base64 -d > .local/argopw
```

##### Create the credentials secret for the crossplane provider and the provider config.
```bash
export BASE64_PW=$(echo -n "${IONOS_PASSWORD}" | base64)
kubectl create secret generic --namespace crossplane-system example-provider-secret --from-literal=credentials="{\"user\":\"${IONOS_USERNAME}\",\"password\":\"${BASE64_PW}\"}"
kubectl apply -f examples/provider/config.yaml --namespace crossplane-system
```

##### Install the root-app chart.
```bash
helm template charts/root-app/ | kubectl apply --namespace=argocd -f -
```

### Example overview

The example will create 3 argocd `Applications` in the cluster.
- The first application is Argocd itself, allowing the installation to then be managed via git based on the chart defined in `charts/argo-cd`


- The second application is crossplane, here the ionoscloud provider is specified as a parameter for the crossplane chart `packages: [ghcr.io/ionos-cloud/crossplane-provider-ionoscloud:v1.0.10]`, therefore it will be added to the cluster together with the crossplane installation. This has the advantage of simplifying the provider installation, but the provider won't have its own independent argocd Application. This means it can't be managed separately from the crossplane Application.


- The third application is an example of creating Ionoscloud resources. This Application will contain the created Managed Resources together with their health state and sync status. Changes to `examples/argo-cd/server` that are committed to git will prompt Argocd to perform the necessary changes to sync the Application to the new state. 

The definitions of Argocd applications are placed in the `charts/root-app/templates` directory.

##### Private registry installation.
The `charts/crossplane-ag` chart is an example for deploying the example from private container registries and helm repo.

In this example, the container registry is a local container running [registry v2](https://hub.docker.com/_/registry) and the helm repo is a local container running [chartmuseum](https://github.com/helm/chartmuseum).

More info on how to setup these can be found in [local-registry.md](local-registry.md)