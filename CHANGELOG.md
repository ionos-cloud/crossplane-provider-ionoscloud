# Changelog

## [0.1.0-alpha.2] (upcoming release)

- **Features**:
    - New CRDs added:
        - _Compute Engine Resources_: Datacenter, Server, Volume, Lan, NIC, FirewallRule, IPFailover, IPBlock;
    - Added validations on CRDs - regarding format, type, minimum/maximum values, specific set of values, required
      values;
    - Added references (using [crossplane-tools](https://github.com/crossplane/crossplane-tools)) on CRDs to be able to
      reference a resource dependency by name.
- **Enhancements**:
    - Existing CRDs updated:
        - _DBaaS Postgres Cluster_ with Datacenter and LAN references.
    - Updated example [GUIDE](examples/example.md).

## [0.1.0-alpha.1] (February 2022)

- First release of Crossplane Provider IONOS Cloud! 🎉
- **Features**:
    - New CRDs:
        - _DBaaS Postgres Cluster_.
