# Using a local registry with Crossplane
### Overview

The installation of crossplane and the crossplane ionos provider can be performed from a local container registry. This is useful during development, as it allows you to install the locally built provider as a Crossplane Package, which more closely replicates typical deployment compared to running out-of-cluster.

This example also uses [Chart Museum](https://github.com/helm/chartmuseum) for a local helm chart repository.


### Setting up the Kind cluster with a local registry

#### Generate self-signed CA certificate
Note: certificate CN and SAN should match the network name of your registry container.
```bash
    echo "generating x509 certificates"
    openssl genrsa -out .certs/kind-registry.key 2048
    openssl req -x509 -sha256 -new -nodes -key .certs/kind-registry.key -days 3650 -subj "/CN=kind-registry" -addext  "subjectAltName = DNS:kind-registry"  -out .certs/kind-registry.cert
```

#### Create the registry container
```bash
  reg_name='kind-registry'
  reg_port='5001'
  echo "creating ${reg_name} container"
  docker run \
    -v "${PWD}"/.certs:/certs \
    -e REGISTRY_HTTP_TLS_CERTIFICATE=/certs/kind-registry.cert \
    -e REGISTRY_HTTP_TLS_KEY=/certs/kind-registry.key \
    -d --restart=always -p "127.0.0.1:${reg_port}:5000" --network bridge --name "${reg_name}" \
    registry:2
```

#### Create the Kind cluster with patched containerd config
```bash
cat <<EOF | kind create cluster --name crossplane-example --config=-
kind: Cluster
apiVersion: kind.x-k8s.io/v1alpha4
nodes:
  - role: control-plane
    extraMounts:
      - containerPath: /etc/containerd/certs.d/kind-registry
        hostPath: $PWD/.certs
containerdConfigPatches:
  - |-
    [plugins."io.containerd.grpc.v1.cri".registry.configs."kind-registry:5000".tls]
      ca_file = "/etc/containerd/certs.d/kind-registry/kind-registry.cert"
EOF
```

#### Connect the registry to the cluster network
```bash
docker network connect "kind" "${reg_name}"
```


#### Configure cluster for crossplane installation
The `ca-bundle-config` ConfigMap must contain the certificate that has been added to the registry
```bash
kubectl config use-context kind-crossplane-example
kubectl create namespace crossplane-system
kubectl config set-context --current --namespace=crossplane-system
kubectl -n crossplane-system create cm ca-bundle-config --from-file=ca-bundle=.certs/kind-registry.cert
```

### Using helm to install crossplane

The crossplane helm charts requires a few changes to `values.yaml` depending on use case. Download a version from [the crossplane helm chart repo](https://charts.crossplane.io/). This example uses `v1.52.2-stable`

##### Pulling crossplane from a local registry
This is optional, you can keep the default value which will install crossplane from the Upbound registry.

In `values.yaml` set the `repository` field to indicate the local registry.
```yaml
image:
  # -- Repository for the Crossplane pod image.
  repository: kind-registry:5000/crossplane/crossplane
  # -- The Crossplane image tag. Defaults to the value of `appVersion` in `Chart.yaml`.
  tag: ""
  # -- The image pull policy used for Crossplane and RBAC Manager pods.
  pullPolicy: IfNotPresent
```

##### Add CA certificate 
Set the name and key of the ConfigMap that contains the registry certificate
```yaml
registryCaBundleConfig:
  # -- The ConfigMap name containing a custom CA bundle to enable fetching packages from registries with unknown or untrusted certificates.
  name: "ca-bundle-config"
  # -- The ConfigMap key containing a custom CA bundle to enable fetching packages from registries with unknown or untrusted certificates.
  key: "ca-bundle"
```

#### [Optional: push the chart package to ChartMuseum](https://chartmuseum.com/docs/#uploading-a-chart-package)

##### Install crossplane

```bash
helm repo add chartmuseum http://localhost:8080
helm repo update
helm install crossplane --namespace crossplane-system chartmuseum/crossplane
```

Verify that crossplane was successfully installed by inspecting the crossplane pods. If the `crossplane` image couldn't be pulled from the registry, a `imagepullBackoff` error will appear.

#### Install the ionos cloud crossplane provider

```bash
export BASE64_PW=$(echo -n "${IONOS_PASSWORD}" | base64)
kubectl create secret generic --namespace crossplane-system example-provider-secret --from-literal=credentials="{\"user\":\"${IONOS_USERNAME}\",\"password\":\"${BASE64_PW}\"}"
kubectl apply -f examples/provider/config.yaml
sleep 120
cat <<EOF | kubectl apply -f -
apiVersion: pkg.crossplane.io/v1
kind: Provider
metadata:
  name: provider-ionos
spec:
  package: kind-registry:5000/ionos-cloud/crossplane-provider-ionoscloud:latest
EOF
```

Verify that crossplane was able to perform the provider package installation by inspecting the provider pod. If the `crossplane-provider-ionoscloud` image couldn't be pulled from the registry, a `imagepullBackoff` error will appear.