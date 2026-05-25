package volume

import (
	sdkgo "github.com/ionos-cloud/sdk-go/v6"

	"github.com/ionos-cloud/crossplane-provider-ionoscloud/apis/compute/v1alpha1"
	"github.com/ionos-cloud/crossplane-provider-ionoscloud/internal/diff"
)

// IsUpToDate returns whether the Volume is up-to-date and a diff string
// describing any differences between the CR spec and the observed SDK state.
func IsUpToDate(cr *v1alpha1.Volume, volume *sdkgo.Volume) (bool, string) {
	if cr == nil && volume.Properties == nil {
		return true, ""
	}
	if cr == nil || volume.Properties == nil {
		return false, "volume properties presence mismatch"
	}
	if volume.Metadata != nil && volume.Metadata.State != nil && *volume.Metadata.State == "BUSY" {
		return true, ""
	}
	p := volume.Properties
	d := diff.New()
	if cr.Spec.ForProvider.Name != "" || p.Name != nil {
		d.Str("name", &cr.Spec.ForProvider.Name, p.Name)
	}
	if p.Size != nil {
		d.Float32("size", &cr.Spec.ForProvider.Size, p.Size)
	}
	if !cr.Spec.ForProvider.SetHotPlugsFromImage {
		if p.CpuHotPlug != nil {
			d.Bool("cpuHotPlug", &cr.Spec.ForProvider.CPUHotPlug, p.CpuHotPlug)
		}
		if p.RamHotPlug != nil {
			d.Bool("ramHotPlug", &cr.Spec.ForProvider.RAMHotPlug, p.RamHotPlug)
		}
		if p.NicHotPlug != nil {
			d.Bool("nicHotPlug", &cr.Spec.ForProvider.NicHotPlug, p.NicHotPlug)
		}
		if p.NicHotUnplug != nil {
			d.Bool("nicHotUnplug", &cr.Spec.ForProvider.NicHotUnplug, p.NicHotUnplug)
		}
		if p.DiscVirtioHotPlug != nil {
			d.Bool("discVirtioHotPlug", &cr.Spec.ForProvider.DiscVirtioHotPlug, p.DiscVirtioHotPlug)
		}
		if p.DiscVirtioHotUnplug != nil {
			d.Bool("discVirtioHotUnplug", &cr.Spec.ForProvider.DiscVirtioHotUnplug, p.DiscVirtioHotUnplug)
		}
	}
	if cr.Spec.ForProvider.Bus != "" || p.Bus != nil {
		d.Str("bus", &cr.Spec.ForProvider.Bus, p.Bus)
	}
	return d.Result()
}
