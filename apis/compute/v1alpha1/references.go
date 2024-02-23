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

// ExtractServerID returns the externalName of a referenced Server.
func ExtractServerID() reference.ExtractValueFn {
	return func(mg resource.Managed) string {
		res, ok := mg.(*Server)
		if !ok {
			return defaultStringValue
		}
		if meta.GetExternalName(res) == res.Name {
			return defaultStringValue
		}
		return meta.GetExternalName(res)
	}
}

// ExtractNicID returns the externalName of a referenced Nic.
func ExtractNicID() reference.ExtractValueFn {
	return func(mg resource.Managed) string {
		res, ok := mg.(*Nic)
		if !ok {
			return defaultStringValue
		}
		if meta.GetExternalName(res) == res.Name {
			return defaultStringValue
		}
		return meta.GetExternalName(res)
	}
}

// ExtractIPBlockID returns the externalName of a referenced IPBlock.
func ExtractIPBlockID() reference.ExtractValueFn {
	return func(mg resource.Managed) string {
		res, ok := mg.(*IPBlock)
		if !ok {
			return defaultStringValue
		}
		if meta.GetExternalName(res) == res.Name {
			return defaultStringValue
		}
		return meta.GetExternalName(res)
	}
}

// ExtractPccID returns the externalName of a referenced Pcc.
func ExtractPccID() reference.ExtractValueFn {
	return func(mg resource.Managed) string {
		res, ok := mg.(*Pcc)
		if !ok {
			return defaultStringValue
		}
		if meta.GetExternalName(res) == res.Name {
			return defaultStringValue
		}
		return meta.GetExternalName(res)
	}
}

// ExtractUserID returns the externalName of a referenced User.
func ExtractUserID() reference.ExtractValueFn {
	return func(mg resource.Managed) string {
		res, ok := mg.(*User)
		if !ok {
			return defaultStringValue
		}
		if meta.GetExternalName(res) == res.Name {
			return defaultStringValue
		}
		return meta.GetExternalName(res)
	}
}
