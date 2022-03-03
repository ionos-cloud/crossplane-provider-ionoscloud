package v1alpha1

import (
	"github.com/crossplane/crossplane-runtime/pkg/meta"
	"github.com/crossplane/crossplane-runtime/pkg/reference"
	"github.com/crossplane/crossplane-runtime/pkg/resource"
)

const defaultStringValue = ""

// ExtractDatacenterID returns the externalName of a referenced Datacenter.
func ExtractDatacenterID() reference.ExtractValueFn {
	return func(mg resource.Managed) string {
		res, ok := mg.(*Datacenter)
		if !ok {
			return defaultStringValue
		}
		if meta.GetExternalName(res) == res.Name {
			return defaultStringValue
		}
		return meta.GetExternalName(res)
	}
}

// ExtractVolumeID returns the externalName of a referenced Volume.
func ExtractVolumeID() reference.ExtractValueFn {
	return func(mg resource.Managed) string {
		res, ok := mg.(*Volume)
		if !ok {
			return defaultStringValue
		}
		if meta.GetExternalName(res) == res.Name {
			return defaultStringValue
		}
		return meta.GetExternalName(res)
	}
}

// ExtractLanID returns the externalName of a referenced Lan.
func ExtractLanID() reference.ExtractValueFn {
	return func(mg resource.Managed) string {
		res, ok := mg.(*Lan)
		if !ok {
			return defaultStringValue
		}
		if meta.GetExternalName(res) == res.Name {
			return defaultStringValue
		}
		return meta.GetExternalName(res)
	}
}
