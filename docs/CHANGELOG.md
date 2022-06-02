# Changelog

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
