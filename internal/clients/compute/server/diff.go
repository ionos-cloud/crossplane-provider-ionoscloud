package server

import (
	sdkgo "github.com/ionos-cloud/sdk-go/v6"

	"github.com/ionos-cloud/crossplane-provider-ionoscloud/apis/compute/v1alpha1"
	"github.com/ionos-cloud/crossplane-provider-ionoscloud/internal/diff"
)

// IsUpToDate returns whether the Server is up-to-date and a diff string
// describing any differences between the CR spec and the observed SDK state.
func IsUpToDate(cr *v1alpha1.Server, server sdkgo.Server) (bool, string) {
	if cr == nil && server.Properties == nil {
		return true, ""
	}
	if cr == nil || server.Properties == nil {
		return false, "server properties presence mismatch"
	}
	if server.Metadata != nil && server.Metadata.State != nil && *server.Metadata.State == sdkgo.Busy {
		return true, ""
	}
	p := server.Properties
	d := diff.New()
	if cr.Spec.ForProvider.Name != "" || p.Name != nil {
		d.Str("name", &cr.Spec.ForProvider.Name, p.Name)
	}
	d.Bool("nicMultiQueue", cr.Spec.ForProvider.NicMultiQueue, p.NicMultiQueue)
	if p.Cores != nil {
		d.Int32("cores", &cr.Spec.ForProvider.Cores, p.Cores)
	}
	if p.Ram != nil {
		d.Int32("ram", &cr.Spec.ForProvider.RAM, p.Ram)
	}
	if cr.Spec.ForProvider.CPUFamily != "" && p.CpuFamily != nil {
		d.Str("cpuFamily", &cr.Spec.ForProvider.CPUFamily, p.CpuFamily)
	}
	if cr.Spec.ForProvider.AvailabilityZone != "" || p.AvailabilityZone != nil {
		d.Str("availabilityZone", &cr.Spec.ForProvider.AvailabilityZone, p.AvailabilityZone)
	}
	if cr.Spec.ForProvider.VolumeCfg.VolumeID != cr.Status.AtProvider.VolumeID {
		volStatus := cr.Status.AtProvider.VolumeID
		d.Str("volumeId", &cr.Spec.ForProvider.VolumeCfg.VolumeID, &volStatus)
	}
	if cr.Spec.ForProvider.PlacementGroupID != "" || p.PlacementGroupId != nil {
		d.Str("placementGroupId", &cr.Spec.ForProvider.PlacementGroupID, p.PlacementGroupId)
	}
	if cr.Spec.ForProvider.VmState != "" && p.VmState != nil {
		d.Str("vmState", &cr.Spec.ForProvider.VmState, p.VmState)
	}
	return d.Result()
}

// IsCubeServerUpToDate returns whether the CubeServer is up-to-date and a
// diff string describing differences between the CR spec and observed SDK state.
func IsCubeServerUpToDate(cr *v1alpha1.CubeServer, server sdkgo.Server) (bool, string) {
	if cr == nil && server.Properties == nil {
		return true, ""
	}
	if cr == nil || server.Properties == nil {
		return false, "server properties presence mismatch"
	}
	if server.Metadata != nil && server.Metadata.State != nil && *server.Metadata.State == sdkgo.Busy {
		return true, ""
	}
	p := server.Properties
	d := diff.New()
	if cr.Spec.ForProvider.Name != "" || p.Name != nil {
		d.Str("name", &cr.Spec.ForProvider.Name, p.Name)
	}
	if cr.Spec.ForProvider.AvailabilityZone != "" || p.AvailabilityZone != nil {
		d.Str("availabilityZone", &cr.Spec.ForProvider.AvailabilityZone, p.AvailabilityZone)
	}
	if p.BootVolume != nil && p.BootVolume.Id != nil && *p.BootVolume.Id != cr.Status.AtProvider.VolumeID {
		volStatus := cr.Status.AtProvider.VolumeID
		d.Str("bootVolume.id", &volStatus, p.BootVolume.Id)
	}
	if cr.Status.AtProvider.VolumeID != "" && !p.HasBootVolume() {
		d.Add("bootVolume", cr.Status.AtProvider.VolumeID, diff.NilSentinel)
	}
	if cr.Spec.ForProvider.VmState != "" && p.VmState != nil {
		d.Str("vmState", &cr.Spec.ForProvider.VmState, p.VmState)
	}
	if server.HasEntities() && server.Entities.HasVolumes() && server.Entities.Volumes.HasItems() {
		items := *server.Entities.Volumes.Items
		if len(items) > 0 && items[0].Properties != nil {
			vp := items[0].Properties
			das := cr.Spec.ForProvider.DasVolumeProperties
			vd := d.Sub("dasVolumeProperties")
			vd.Str("name", &das.Name, vp.Name)
			vd.Str("bus", &das.Bus, vp.Bus)
			vd.Bool("cpuHotPlug", &das.CPUHotPlug, vp.CpuHotPlug)
			vd.Bool("ramHotPlug", &das.RAMHotPlug, vp.RamHotPlug)
			vd.Bool("nicHotPlug", &das.NicHotPlug, vp.NicHotPlug)
			vd.Bool("nicHotUnplug", &das.NicHotUnplug, vp.NicHotUnplug)
			vd.Bool("discVirtioHotPlug", &das.DiscVirtioHotPlug, vp.DiscVirtioHotPlug)
			vd.Bool("discVirtioHotUnplug", &das.DiscVirtioHotUnplug, vp.DiscVirtioHotUnplug)
		}
	}
	return d.Result()
}
