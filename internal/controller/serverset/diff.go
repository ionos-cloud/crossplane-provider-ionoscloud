package serverset

import (
	ionoscloud "github.com/ionos-cloud/sdk-go/v6"

	"github.com/ionos-cloud/crossplane-provider-ionoscloud/apis/compute/v1alpha1"
	"github.com/ionos-cloud/crossplane-provider-ionoscloud/internal/diff"
)

func buildDiff(cr *v1alpha1.ServerSet, servers []v1alpha1.Server, volumes []v1alpha1.Volume, nics []v1alpha1.Nic) string {
	d := diff.New()
	replicas := cr.Spec.ForProvider.Replicas
	t := cr.Spec.ForProvider.Template.Spec
	bt := cr.Spec.ForProvider.BootVolumeTemplate.Spec

	gotServers := len(servers)
	if gotServers != replicas {
		d.Int("servers", &replicas, &gotServers)
	}
	gotVolumes := len(volumes)
	if gotVolumes != replicas {
		d.Int("bootVolumes", &replicas, &gotVolumes)
	}
	expectedNICs := len(t.NICs) * replicas
	gotNICs := len(nics)
	if gotNICs != expectedNICs {
		d.Int("nics", &expectedNICs, &gotNICs)
	}

	available := ionoscloud.Available
	for i, s := range servers {
		sd := d.Sub("servers").Index(i)
		sd.Int32("cores", &t.Cores, new(s.Spec.ForProvider.Cores))
		sd.Int32("ram", &t.RAM, new(s.Spec.ForProvider.RAM))
		sd.Str("cpuFamily", &t.CPUFamily, new(s.Spec.ForProvider.CPUFamily))
		if t.NicMultiQueue != nil && s.Spec.ForProvider.NicMultiQueue != nil {
			sd.Bool("nicMultiQueue", t.NicMultiQueue, s.Spec.ForProvider.NicMultiQueue)
		}
		state := s.Status.AtProvider.State
		if state != ionoscloud.Available {
			sd.Str("state", &available, &state)
		}
	}

	for i, v := range volumes {
		vd := d.Sub("bootVolumes").Index(i)
		vd.Float32("size", &bt.Size, new(v.Spec.ForProvider.Size))
		vd.Str("image", &bt.Image, new(v.Spec.ForProvider.Image))
		vd.Str("type", &bt.Type, new(v.Spec.ForProvider.Type))
		vd.Bool("setHotPlugsFromImage", &bt.SetHotPlugsFromImage, new(v.Spec.ForProvider.SetHotPlugsFromImage))
		state := v.Status.AtProvider.State
		if state != ionoscloud.Available {
			vd.Str("state", &available, &state)
		}
	}

	return d.String()
}
