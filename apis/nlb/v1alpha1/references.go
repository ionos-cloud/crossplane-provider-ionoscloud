package v1alpha1

import (
	"github.com/crossplane/crossplane-runtime/pkg/meta"
	"github.com/crossplane/crossplane-runtime/pkg/reference"
	"github.com/crossplane/crossplane-runtime/pkg/resource"
)

// ExtractNetworkLoadBalancerID returns the externalName of a referenced NetworkLoadBalancer.
func ExtractNetworkLoadBalancerID() reference.ExtractValueFn {
	return func(mg resource.Managed) string {
		res, ok := mg.(*NetworkLoadBalancer)
		if !ok {
			return ""
		}
		if meta.GetExternalName(res) == res.Name {
			return ""
		}
		return meta.GetExternalName(res)
	}
}
