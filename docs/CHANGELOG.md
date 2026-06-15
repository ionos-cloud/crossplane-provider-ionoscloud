## [1.2.3] (February 2026)
- Fix lan update when `Pcc`when updating from nil
- Update trivy to v0.35.0
- Fix e2e tests by setting crossplane version for chart to 1.20.1

## [1.2.2] (February 2026)
### Chore
- Set up Dependabot for automated dependency updates.
- Bump Go to 1.25.7 to address CVEs.

## [1.2.1] (February 2026)
### Fixes:
- ICNAS-651: Fix stateMap feature to behave according to expectations.
### Chore
- Bump Go to 1.25.6 to fix CVEs.
### Tests:
- Add stateful server set (sss) e2e test in Go.

## [1.2.0] (December 2025)
### Features:
- Update provider to use a single-image setup, replacing the previous two-image approach.
- Add Trivy and govulncheck security scanning to CI, ci-weekly, and CD pipelines.
### Fixes:
- Fix syntax oversight in `cd.yaml`.
- Enforce platforms in Makefile.
### Documentation:
- Add notice for old two-image setups in README.

## [1.1.19] (December 2025)
### Features:
- Add `spec.forProvider.vmState` and `status.atProvider.vmState` fields for `Server` and `CubeServer` resources.
- Add late initialization for volume hotplug attributes.
### Fixes:
- Make `PublicIpCfgs` nullable in nodepool and update CRD accordingly.
- Bump Go to 1.25.3 to fix vulnerabilities; update controller-tools to latest.

## [1.1.18] (October 2025)
### Features:
- Add custom VM state support for serverset (sset) and stateful serverset (ssset).
### Fixes:
- Add serverset diff to stateful server set reconciliation.
- Fix ssset reconciling too often by re-adding event filter.
- Fix K8s version shown incorrectly after NodePool update.
- Align multiline descriptions in generated docs.
- Fix previous workflow refactor issues.
### Chore
- Rewrite `issue-creation.yml` workflow.

## [1.1.17] (October 2025)
### Features:
- Allow setting hotplugs from image with failover mechanism for server updates.
- Add NIC multi-queue support.
### Fixes:
- Fix small bug introduced by hotplug-from-image change.
- Fix nodepool `instanceUpdateInput` generation to handle optional `k8sVersion`.
- Update `pcislot` even if value is not 0.
- Bump postgresVersion from 13 to 15 in examples and tests.
### Chore
- Remove dataplatform resources.
- Run Trivy security scans in CI.
### Tests:
- Add sss e2e test; use token for testing.
- Use secret for image password in tests.
- Run tests on de/txl; increase update e2e timeout for volumes and LAN.

## [1.1.16] (July 2025)
### Fixes:
- Update Alpine image to fix CVE-2024-12797.
- Fix configmap and error handling on same bootvolume.
### Chore
- Bump Go to 1.24.4.

## [1.1.15] (July 2025)
### Features:
- Add `serverType` field to Kubernetes nodepool.
### Fixes:
- Remove redundant `cpuFamily` API request when field is empty in k8s nodepool.

## [1.1.14] (June 2025)
### Features:
- Attach stateful server set volumes in parallel.
- Remove creation-pending annotations from ssset on Crossplane reboot.
### Fixes:
- Fix firewall assignment to NIC in stateful server set.
- Improve diff ordering consistency.
- Add diffs and improve logging messages.
### Chore
- Replace deprecated `ControllerConfig` with `DeploymentRuntimeConfig`.
- Change default max-reconcile-rate to 10.
- Update kind version for e2e; fix LAN tests.

## [1.1.13] (May 2025)
### Chore
- Upgrade Go to 1.24 and crossplane-runtime to 1.20; update build image.

## [1.1.12] (May 2025)
### Features:
- Add per-resource `reconcile-rate` configuration.
- Add parallel creation for NICs, LANs, and data volumes.
- Expose `ipv4cidr` as a readonly field on LAN; update SDK version; replace mockgen.
- Set `K8sVersion` returned from API in `status.atProvider` instead of overriding `spec.forProvider`.
### Fixes:
- Context timeout recovery: deletion now handled through Kubernetes garbage collector with owner references.
- Fix deletion of CRD sub-resources only on 422 errors; do not delete stateful server set, update it instead.
- Fix flaky user assignment.
- Fix linter errors and import issues.

## [1.1.11] (May 2025)
### Features:
- Add metrics to the provider.
- DB-4567: Allow configuring database user password via Kubernetes secret.
### Fixes:
- Add base path for PostgreSQL in case of endpoint override.

## [1.1.10] (April 2025)
### Features:
- Clean up orphan resources automatically.
### Fixes:
- Fix stateful server set data volume update when multiple volumes are configured with different settings.
### Chore
- Update crossplane-runtime to 1.19; apply crypto vulnerability fix.

## [1.1.9] (March 2025)
### Fixes:
- Do not trigger update on nodecount when autoscaling is active.
- Fix kubebuilder default tags that had wrong format.

## [1.1.8] (February 2025)
### Features:
- Add private Kubernetes cluster support.
- SDK-2087: Add firewall rules support for serverset and stateful serverset.
### Fixes:
- Fix unpredictable ssset creation and deletion behavior caused by root context expiring.
- Fix S3 key stuck in creation loop after external resource deletion.
- Update hostname faster on server resource.
- Improve ssset logging; add sset and ssset name to log output.
### Documentation:
- Allow resource exclusion from documentation generation.
- Fix k8s documentation.

## [1.1.7]
### Fixes:
- Write kubeconfig to k8s cluster, only if not in deploying state.

## [1.1.6]
### Chore
- Update crossplane-runtime to 1.18.0

## [1.1.4]
- **Fixes**:
 - Volume should populate the status ID from the bootvolume of the server, because it cannot store it on create. This can cause larger wait times for volume attach.

## [1.1.3]
- **Features**:
    - Enable `publishConnectionDetails` option for s3 key, compute user and k8s clusters
- **Misc**:
    - Removed enum validation for `cpuFamily` fields.

## [1.1.2]
- **Features**:
    - Connection pooler for psql, switch to sdk-go-bundle for postgres
- **Fixes**:
    - Add missing ipv6 property for nic

## [1.1.1] (July 2024)
- **Fixes**:
  - Revert merge for nodepool fix

## [1.1.0] (July 2024)
- **Features**:
    - Add `database` CRD for postgres
    - Upgrade crossplane-runtime to 1.16.0
    - Shortnames for some of the resources. E.g. `alb` for `applicationloadbalancer`
    - Add serverset and statefulserverset CRDS
    - Add `iPV6cidr` to NIC
    - Add shortnames for volume - vol, datacenter - dc.
    - Add `ipv6Cidr` to LAN.
    - Run unit tests on PR

## [1.0.14] (June 2024)
- **Fixes**:
    - Failing unit tests for compute user
    - K8s Nodepool creation issue due to empty `DatacenterID` value on node pool lan.

## [1.0.13] (May 2024)
- **Features**:
    - Add `datacenterID` field for node pool lan in k8s `NodePool` CRD managed resources:
    - Save `s3SecretKey` and `s3keyID` to the `Secrets` field in the `S3Key` CRD

- **Fixes**:
    - S3Key should be properly deleted
    - Remove useless `secretKey` field in s3Key
- **Misc**:
    - Use builtin `controller.Options` in controller setup functions

## [1.0.12] (May 2024)
- **Fixes**:
    - Fixes for `MongoUser` and `PostgresUser`:
        - Panic caused by improper dereference of `password` pointer
        - Passwords that are provided via `Secrets` no longer appear in clear text in the resource `spec`

## [1.0.11] (April 2024)
- **Features**:
  - Add `NLB` managed resources:
    - `Network Load Balancer`
    - `Forwarding Rule`
    - `Flowlog`

- **Fixes**:
  - Changed fields for `CubeServer` CR:
    - `cpuFamily` field has been removed as it prevented external resource creation.
    - `template.name` is now immutable

- **Misc**:
    - Added local registry usage example
    - Changed e2e tests location to es/vit
    - Renamed `Private Cross Connect` to `Cross Connect` in `pcc` documentation

## [1.0.10] (March 2024)
- **Features**:
  - Allow conversion between schema types and go types
  - Add `group` CRD to support CRUD of compute Groups
  - Update `sdk-go` to v6.1.11

- **Misc**:
  - Minor `user` CRD refactor

## [1.0.9] (February 2024)
- **Features**:
- Add `MongoCluster` crd to support CRUD of MongoDB clusters
- Add `MongoUser` crd to support CRUD of MongoDB users
- Add `DataplatformCluster` crd to support CRUD of Dataplatform clusters
- Add `DataplatformNodepool` crd to support CRUD of Dataplatform clusters
- Add `PICSlot` status field to `volume` and `nic` crds
- Use `make provider.addtype` to add new types to the provider
- Update crossplane-runtime to 1.14.4.

- **Documentation**:
- Add server composition and claim example
- Add docs on how to set pinning for crossplane provider. See [here](docs/README.md#authentication-on-ionos-cloud)
- Preserve order of fields in the generated documentation.

## [1.0.8] (December 2023)
- **Features**:
- Add `postgresuser` crd
- Update `GO` version to v1.21
- Update `golanci-lint` to v1.54.0

## [1.0.7] (October 2023)
- **Features**:
 - Option to provide postgres credentials via secret, env variable or path to file
 - k8s #116 enrich connection details with token,servername and server Url
 - add s3key crd
 - add pcc(privatecrossconnect) crd
 - added link between lan and pcc. This is a small breaking change, as before there was only the option of providing the UUID directly
 -
## [1.0.6] (August 2023)
- **Features**:
    - Update Crossplane-Runtime to latest version (v0.20.0). CRDs - now require `managementPolicies` to be defined
    - Update golang to 1.19
    - Update `build` submodule to latest version
    - Update workflows to use latest versions

## [1.0.5] (May 2023)
- **Fixes**:
    - Updated validation logic to prevent constant updates on NICs, in case they are configured with DHCP=true
    - Removed `oldIPsNic` package private variable, as it could cause race conditions with other NIC resources and doesn't seem to be required for the validation logic
    - Removed checks which would always be true as the `AtProvider.IPs` will always be updated in the `Observe()` function

## [1.0.4] (February 2023)

- **Documentation**:
  - Add docs on how to enable pinning and debugging using env variables

## [1.0.3] (February 2023)

- **Features**:
  - Add fields `vnet` to `nic` and `placementGroupId` to `server`. These are internally used fields, they can only be set if the account has special permissions granted
- **Tests**:
    - Added unit tests for server node nic
- **Misc**:
    - Refactor to increase readability and remove some duplicated code

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
