# CCPatch

CCPatch is a tool for patching cloud-config files. It is designed to be used in conjunction with [cloud-init](https://cloudinit.readthedocs.io/en/latest/), but can be used with any tool that reads cloud-config files in YAML format. 

Is supports a few edge cases for our `ionos-cloudinit`, like `environment` keys and `substitutions`.

## Substitutions

You must save the `globalState` object either in the state of your controller or in a global configmap. 
The `globalState` stores Key-Value pairs for each `identifier` and is used to replace the `substitutions` in the cloud-config file.

**Substitutions with unknown handlers will be ignored.**

```go
ccpatch.NewCloudInitPatcherWithSubstitutions(raw, identifier, substitutions, gs)
```