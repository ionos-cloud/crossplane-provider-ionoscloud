package v1alpha1

import (
	"github.com/crossplane/crossplane-runtime/pkg/meta"
	"github.com/crossplane/crossplane-runtime/pkg/reference"
	"github.com/crossplane/crossplane-runtime/pkg/resource"
)

const defaultStringValue = ""

// ExtractApplicationLoadBalancerID returns the externalName of a referenced ApplicationLoadBalancer.
func ExtractApplicationLoadBalancerID() reference.ExtractValueFn {
	return func(mg resource.Managed) string {
		res, ok := mg.(*ApplicationLoadBalancer)
		if !ok {
			return defaultStringValue
		}
		if meta.GetExternalName(res) == res.Name {
			return defaultStringValue
		}
		return meta.GetExternalName(res)
	}
}

// ExtractTargetGroupID returns the externalName of a referenced TargetGroup.
func ExtractTargetGroupID() reference.ExtractValueFn {
	return func(mg resource.Managed) string {
		res, ok := mg.(*TargetGroup)
		if !ok {
			return defaultStringValue
		}
		if meta.GetExternalName(res) == res.Name {
			return defaultStringValue
		}
		return meta.GetExternalName(res)
	}
}
