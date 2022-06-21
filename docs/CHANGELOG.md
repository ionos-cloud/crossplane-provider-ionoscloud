# Changelog

## upcoming release

- **Features**:
  - Allow to set a global IONOS_API_URL overwrite in the provider pod via environment variables

- **Tests** 
  - Add unit tests for k8s cluster 
## [1.0.0-beta.2] (June 2022)

- **Features**:
  - Added `cpuFamily` field to the `status`
    - Note: this update applies to Kubernetes NodePool, Compute Server and Compute Cube Server resources
  - Added access to the CRDs in the repository
- **Fixes**:
  - Added correct categories to the `providerConfig` types
  - Added fix for comparison on `mantenanceWindow` field, for timestamp ending in `Z` suffix
    - Note: this update applies to Kubernetes Cluster, Kubernetes NodePool and DBaaS Postgres Cluster resources
  - Removed late initialization by the provider for the `spec.cpuFamily` field, since the field is immutable - it will be displayed into the `status`
    - Note: this update applies to Kubernetes NodePool, Compute Server and Compute Cube Server resources
- **Dependency-update**:
  - Updated SDK Go to [v6.0.4](https://github.com/ionos-cloud/sdk-go/releases/tag/v6.0.4)

## [1.0.0-beta.1] (May 2022)

**First release of the Crossplane Provider IONOS Cloud!** 🎉

- **Features**:
    - Added Managed Resources:
        - _Compute Engine Resources_: 
          - Datacenter;
          - Server;
          - CubeServer;
          - Volume;
          - Lan;
          - NIC;
          - FirewallRule;
          - IPFailover;
          - IPBlock;
        - _Kubernetes Resources_:
          - Cluster;
          - NodePool;
        - _DBaaS Postgres Resources_:
          - Postgres Cluster;
    - Added references to resources in order to solve dependencies (using [crossplane-tools](https://github.com/crossplane/crossplane-tools));
    - Added support to set IPs fields automatically using references to IPBlock and indexes for NICs, IPFailover, FirewallRule, NodePools; 
- **Documentation**:
  - Added [step-by-step guide](../examples/example.md) for installing a DBaaS Postgres Cluster using Crossplane Provider IONOS Cloud;
  - Added overview of Managed Resources and Cloud Services Resources supported. See [here](RESOURCES.md);
  - Added examples of configuration files for creating resources. See [examples](../examples);
  - Added example for Compositions and Claims. See [example](RESOURCES.md#compositions-and-claims).
