package volume

import (
	"testing"

	sdkgo "github.com/ionos-cloud/sdk-go/v6"
	"github.com/stretchr/testify/assert"

	"github.com/ionos-cloud/crossplane-provider-ionoscloud/apis/compute/v1alpha1"
)

func TestIsUpToDate(t *testing.T) {
	name := "vol"
	var size float32 = 10.0
	busy := "BUSY"
	bus := "VIRTIO"
	otherBus := "IDE"
	trueVal := true

	tests := []struct {
		name     string
		cr       *v1alpha1.Volume
		volume   *sdkgo.Volume
		want     bool
		wantDiff string
	}{
		{
			name:     "BothNil",
			cr:       nil,
			volume:   &sdkgo.Volume{},
			want:     true,
			wantDiff: "",
		},
		{
			name:     "CRNilVolumeNotNil",
			cr:       nil,
			volume:   &sdkgo.Volume{Properties: &sdkgo.VolumeProperties{}},
			want:     false,
			wantDiff: "volume properties presence mismatch",
		},
		{
			name:     "CRNotNilVolumeNil",
			cr:       &v1alpha1.Volume{},
			volume:   &sdkgo.Volume{},
			want:     false,
			wantDiff: "volume properties presence mismatch",
		},
		{
			name: "VolumeBusy",
			cr:   &v1alpha1.Volume{},
			volume: &sdkgo.Volume{
				Metadata:   &sdkgo.DatacenterElementMetadata{State: &busy},
				Properties: &sdkgo.VolumeProperties{},
			},
			want:     true,
			wantDiff: "",
		},
		{
			name: "NameMismatch",
			cr:   &v1alpha1.Volume{Spec: v1alpha1.VolumeSpec{ForProvider: v1alpha1.VolumeParameters{Name: "foo"}}},
			volume: &sdkgo.Volume{
				Properties: &sdkgo.VolumeProperties{Name: &name},
			},
			want:     false,
			wantDiff: "name exp=foo act=vol",
		},
		{
			name: "NameNilCRNotEmpty",
			cr:   &v1alpha1.Volume{Spec: v1alpha1.VolumeSpec{ForProvider: v1alpha1.VolumeParameters{Name: "foo"}}},
			volume: &sdkgo.Volume{
				Properties: &sdkgo.VolumeProperties{},
			},
			want:     false,
			wantDiff: "name exp=foo act=<nil>",
		},
		{
			name: "SizeMismatch",
			cr:   &v1alpha1.Volume{Spec: v1alpha1.VolumeSpec{ForProvider: v1alpha1.VolumeParameters{Name: name, Size: 20.0}}},
			volume: &sdkgo.Volume{
				Properties: &sdkgo.VolumeProperties{Name: &name, Size: &size},
			},
			want:     false,
			wantDiff: "size exp=20 act=10",
		},
		{
			name: "CpuHotPlugMismatch",
			cr:   &v1alpha1.Volume{Spec: v1alpha1.VolumeSpec{ForProvider: v1alpha1.VolumeParameters{Name: name, CPUHotPlug: false}}},
			volume: &sdkgo.Volume{
				Properties: &sdkgo.VolumeProperties{Name: &name, CpuHotPlug: &trueVal},
			},
			want:     false,
			wantDiff: "cpuHotPlug exp=false act=true",
		},
		{
			name: "RamHotPlugMismatch",
			cr:   &v1alpha1.Volume{Spec: v1alpha1.VolumeSpec{ForProvider: v1alpha1.VolumeParameters{Name: name, RAMHotPlug: false}}},
			volume: &sdkgo.Volume{
				Properties: &sdkgo.VolumeProperties{Name: &name, RamHotPlug: &trueVal},
			},
			want:     false,
			wantDiff: "ramHotPlug exp=false act=true",
		},
		{
			name: "NicHotPlugMismatch",
			cr:   &v1alpha1.Volume{Spec: v1alpha1.VolumeSpec{ForProvider: v1alpha1.VolumeParameters{Name: name, NicHotPlug: false}}},
			volume: &sdkgo.Volume{
				Properties: &sdkgo.VolumeProperties{Name: &name, NicHotPlug: &trueVal},
			},
			want:     false,
			wantDiff: "nicHotPlug exp=false act=true",
		},
		{
			name: "NicHotUnplugMismatch",
			cr:   &v1alpha1.Volume{Spec: v1alpha1.VolumeSpec{ForProvider: v1alpha1.VolumeParameters{Name: name, NicHotUnplug: false}}},
			volume: &sdkgo.Volume{
				Properties: &sdkgo.VolumeProperties{Name: &name, NicHotUnplug: &trueVal},
			},
			want:     false,
			wantDiff: "nicHotUnplug exp=false act=true",
		},
		{
			name: "DiscVirtioHotPlugMismatch",
			cr:   &v1alpha1.Volume{Spec: v1alpha1.VolumeSpec{ForProvider: v1alpha1.VolumeParameters{Name: name, DiscVirtioHotPlug: false}}},
			volume: &sdkgo.Volume{
				Properties: &sdkgo.VolumeProperties{Name: &name, DiscVirtioHotPlug: &trueVal},
			},
			want:     false,
			wantDiff: "discVirtioHotPlug exp=false act=true",
		},
		{
			name: "DiscVirtioHotUnplugMismatch",
			cr:   &v1alpha1.Volume{Spec: v1alpha1.VolumeSpec{ForProvider: v1alpha1.VolumeParameters{Name: name, DiscVirtioHotUnplug: false}}},
			volume: &sdkgo.Volume{
				Properties: &sdkgo.VolumeProperties{Name: &name, DiscVirtioHotUnplug: &trueVal},
			},
			want:     false,
			wantDiff: "discVirtioHotUnplug exp=false act=true",
		},
		{
			name: "BusMismatch",
			cr:   &v1alpha1.Volume{Spec: v1alpha1.VolumeSpec{ForProvider: v1alpha1.VolumeParameters{Name: name, Bus: otherBus}}},
			volume: &sdkgo.Volume{
				Properties: &sdkgo.VolumeProperties{Name: &name, Bus: &bus},
			},
			want:     false,
			wantDiff: "bus exp=IDE act=VIRTIO",
		},
		{
			name: "UpToDate",
			cr:   &v1alpha1.Volume{Spec: v1alpha1.VolumeSpec{ForProvider: v1alpha1.VolumeParameters{Name: name, Size: size}}},
			volume: &sdkgo.Volume{
				Properties: &sdkgo.VolumeProperties{Name: &name, Size: &size},
			},
			want:     true,
			wantDiff: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			upToDate, wantDiff := IsUpToDate(tt.cr, tt.volume)
			assert.Equal(t, tt.want, upToDate)
			assert.Equal(t, tt.wantDiff, wantDiff)
		})
	}
}
