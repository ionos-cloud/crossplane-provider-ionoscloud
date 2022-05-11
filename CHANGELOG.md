# Changelog

## [0.1.0-alpha.3] (upcoming release)

- **Breaking Changes**:
    - updated `spec.forProvider.ips` field from **Nic Managed Resource** to `spec.forProvider.ipsConfigs` being able to
      set IPs directly or via references and indexes of the IPBlocks 
    - updated `spec.forProvider.ip` field from **IPFailover Managed Resource** to `spec.forProvider.ipConfig` being able
      to set the required IP directly or via reference and index to an IPBlock
      set IPs directly or via references and indexes of the IPBlocks
    - removed temporarily `spec.forProvider.public` field from **K8s Cluster Managed Resource**
    - removed temporarily `spec.forProvider.gatewayIp` field from **K8s NodePool Managed Resource**
- **Enhancements**:
    - Added and updated documentation. See [docs](docs/README.md)
    - Added example for Compositions and Claims. See [example](docs/RESOURCES.md#compositions-and-claims)
- **Fixes**:
    - fixed late initialization for **Server** and **CubeServer** Managed Resources if the CPU Family is not set by the
      user, but by the API

## [0.1.0-alpha.2] (March 2022)

- **Features**:
    - New CRDs added:
        - _Compute Engine Resources_: Datacenter, Server, Volume, Lan, NIC, FirewallRule, IPFailover, IPBlock;
        - _Kubernetes Resources_: Cluster, NodePool;
    - Added validations on CRDs - regarding format, type, minimum/maximum values, specific set of values, required
      values;
    - Added references (using [crossplane-tools](https://github.com/crossplane/crossplane-tools)) on CRDs to be able to
      reference a resource dependency by name.
    - Debug Mode: see [Provider Logs](docs/README.md#debug-mode)
      using [ControllerConfig](examples/provider/debug-config.yaml)
- **Enhancements**:
    - Existing CRDs updated:
        - _DBaaS Postgres Cluster_ with Datacenter and LAN references.
    - Updated example [GUIDE](examples/example.md).
    - Removed debug mode from controller image. Status messages are displayed with `kubectl get <resource> -o json`.
- **Breaking Changes**:
    - Renamed DBaaS Postgres Cluster CR from `Cluster` to `PostgresCluster`
    - Renamed DBaaS Postgres Cluster CR API Version from `dbaas.postgres.ionoscloud.crossplane.io`
      to `dbaas.ionoscloud.crossplane.io`

## [0.1.0-alpha.1] (February 2022)

- First (private) release of Crossplane Provider IONOS Cloud! 🎉
- **Features**:
    - New CRDs:
        - _DBaaS Postgres Cluster_.
