package k8s

import "github.com/ionos-cloud/crossplane-provider-ionoscloud/internal/clients/compute"

// States of K8s Resources
const (
	AVAILABLE  = compute.AVAILABLE
	BUSY       = compute.BUSY
	DEPLOYING  = "DEPLOYING"
	ACTIVE     = compute.ACTIVE
	UPDATING   = compute.UPDATING
	DESTROYING = compute.DESTROYING
	TERMINATED = "TERMINATED"
)
