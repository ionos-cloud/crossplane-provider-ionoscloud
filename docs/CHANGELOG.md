# Changelog

## [1.0.2] (January 2023)

- **Fixes**:
    - Update `sourceIpConfig` and `targetIpConfig` `ip` fields on `FirewallRule` will also allow cidr to be set, not only ips
- **Dependency-update**:
    - Updated dependencies for all libraries

## [1.0.1] (September 2022)

- **Fixes**:
  - Update `APISubnetAllowList` field on `Cluster` K8s Managed Resource only if it is not empty


## [1.0.0] (July 2022)

- **Features**:
    - Added `--unique-names` option support for name uniqueness for IONOS Cloud resources;
    - Added check for `spec.forProvider.name` field on `NodePool` K8s Managed Resource - for reconciliation loops;
    - Added check for resources to be updated when name from IONOS Cloud is empty and `spec.forProvider.name` is not
      empty;
- **Fixes**:
    - Removed read-only field `mac` from `Nic` Compute Managed Resource:
        - New field: `status.atProvider.mac`;
    - Updated User Agent for Crossplane Provider for IONOS Cloud to contain provider version;
- **Documentation**:
    - Updated documentation with the `--unique-names` option support.

## [1.0.0-beta.5] (July 2022)

- **Features**:
    - Added Managed Resources:
        - _Managed Backup_:
            - BackupUnit.
    - Added reference support for BackupUnit in Volume Managed Resource:
        - **Breaking-change**: `spec.forProvider.backupunitId` field is renamed
          into `spec.forProvider.backupUnitConfig` (reference to a `backupunit` instance).
- **Fixes**:
    - Added missing features for `CubeServer` Managed Resource:
        - New fields: `spec.forProvider.backupUnitConfig`, `spec.forProvider.userData`
- **Documentation**:
    - Separated documentation per service
    - Added support for generation of the documentation

## [1.0.0-beta.4] (June 2022)

- **Features**:
    - Added Managed Resources:
        - _Application Load Balancer_:
            - ApplicationLoadBalancer;
            - ForwardingRule;
            - TargetGroup.
- **Tests**:
    - Added unit tests for k8s node pools
- **Dependency-update**:
    - Updated SDK Go to [v6.1.0](https://github.com/ionos-cloud/sdk-go/releases/tag/v6.1.0)

## [1.0.0-beta.3] (June 2022)

- **Features**:
    - Allow to set a global `IONOS_API_URL` overwrite in the provider pod via environment variables
    - Added timeout option for all the calls happening in the reconciliation functions: `--timeout`
    - Added `SonarCloud` integration and improved duplicate code
- **Dependency-updates**:
    - Updated `sigs.k8s.io/controller-runtime` to v0.12.1
    - Updated `k8s.io/client-go and k8s.io/api-machinery` to v0.24.0
- **Tests**:
    - Added unit tests for k8s cluster

## [1.0.0-beta.2] (June 2022)

- **Features**:
    - Added `cpuFamily` field to the `status`
        - Note: this update applies to Kubernetes NodePool, Compute Server and Compute Cube Server resources
    - Added access to the CRDs in the repository
- **Fixes**:
    - Added correct categories to the `providerConfig` types
    - Added fix for comparison on `mantenanceWindow` field, for timestamp ending in `Z` suffix
        - Note: this update applies to Kubernetes Cluster, Kubernetes NodePool and DBaaS Postgres Cluster resources
    - Removed late initialization by the provider for the `spec.cpuFamily` field, since the field is immutable - it will
      be displayed into the `status`
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
    - Added references to resources in order to solve dependencies (
      using [crossplane-tools](https://github.com/crossplane/crossplane-tools));
    - Added support to set IPs fields automatically using references to IPBlock and indexes for NICs, IPFailover,
      FirewallRule, NodePools;
- **Documentation**:
    - Added [step-by-step guide](../examples/example.md) for installing a DBaaS Postgres Cluster using Crossplane
      Provider IONOS Cloud;
    - Added overview of Managed Resources and Cloud Services Resources supported. See [here](RESOURCES.md);
    - Added examples of configuration files for creating resources. See [examples](../examples);
    - Added example for Compositions and Claims. See [example](RESOURCES.md#compositions-and-claims).
