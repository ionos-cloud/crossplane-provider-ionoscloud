# CCPatch

CCPatch is a tool for patching cloud-config files. It is designed to be used in conjunction with [cloud-init](https://cloudinit.readthedocs.io/en/latest/), but can be used with any tool that reads cloud-config files in YAML format. 

Is supports a few edge cases for our `ionos-cloudinit`, like `environment` keys and `substitutions`.

## Substitutions

You must save the `globalState` object either in the state of your controller or in a global configmap. 
The `globalState` stores Key-Value pairs for each `identifier` and is used to replace the `substitutions` in the cloud-config file.

**Substitutions with unknown handlers will be ignored.**

### Usage

```go
raw := []byte(`#cloud-config ...`)
// or load state
gs := substitution.NewGlobalState()

substitutions = []substitution.Substitution{
		{
			Type:   "ipv6Address",
			Key:    "$ipv6Address",
			Unique: true,
			AdditionalProperties: map[string]string{
				"cidr": "fc00:1::1/64",
			},
		},
		{
			Type:   "ipv4Address",
			Key:    "$ipv4",
			Unique: true,
			AdditionalProperties: map[string]string{
				"cidr": "192.0.2.0/24",
			},
		},
	}

patcher, err := ccpatch.NewCloudInitPatcherWithSubstitutions(raw, identifier, substitutions, gs)

// ... do as usual with the patcher

// save State in crd status
```

Take a look at [func TestSubstitutionManager(t *testing.T) {](./substitutions_test.go#L39-L40)