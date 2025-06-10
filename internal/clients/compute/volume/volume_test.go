package volume

import (
	"testing"

	sdkgo "github.com/ionos-cloud/sdk-go/v6"
	"github.com/stretchr/testify/assert"

	"github.com/ionos-cloud/crossplane-provider-ionoscloud/apis/compute/v1alpha1"
)

func TestIsUpToDateWithDiff(t *testing.T) {
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
			wantDiff: "Volume is nil and custom resource is nil",
		},
		{
			name:     "CRNilVolumeNotNil",
			cr:       nil,
			volume:   &sdkgo.Volume{Properties: &sdkgo.VolumeProperties{}},
			want:     false,
			wantDiff: "Custom resource is nil, but volume properties are not nil",
		},
		{
			name:     "CRNotNilVolumeNil",
			cr:       &v1alpha1.Volume{},
			volume:   &sdkgo.Volume{},
			want:     false,
			wantDiff: "Volume properties are nil, but custom resource is not nil",
		},
		{
			name: "VolumeBusy",
			cr:   &v1alpha1.Volume{},
			volume: &sdkgo.Volume{
				Metadata:   &sdkgo.DatacenterElementMetadata{State: &busy},
				Properties: &sdkgo.VolumeProperties{},
			},
			want:     true,
			wantDiff: "Volume is busy, cannot update it now",
		},
		{
			name: "NameMismatch",
			cr:   &v1alpha1.Volume{Spec: v1alpha1.VolumeSpec{ForProvider: v1alpha1.VolumeParameters{Name: "foo"}}},
			volume: &sdkgo.Volume{
				Properties: &sdkgo.VolumeProperties{Name: &name},
			},
			want:     false,
			wantDiff: "Volume name does not match the one in the CR: vol != foo",
		},
		{
			name: "NameNilCRNotEmpty",
			cr:   &v1alpha1.Volume{Spec: v1alpha1.VolumeSpec{ForProvider: v1alpha1.VolumeParameters{Name: "foo"}}},
			volume: &sdkgo.Volume{
				Properties: &sdkgo.VolumeProperties{},
			},
			want:     false,
			wantDiff: "Volume name is nil, but CR name is not empty: foo",
		},
		{
			name: "SizeMismatch",
			cr:   &v1alpha1.Volume{Spec: v1alpha1.VolumeSpec{ForProvider: v1alpha1.VolumeParameters{Name: name, Size: 20.0}}},
			volume: &sdkgo.Volume{
				Properties: &sdkgo.VolumeProperties{Name: &name, Size: &size},
			},
			want:     false,
			wantDiff: "Volume size does not match the one in the CR: 10.00 != 20.00",
		},
		{
			name: "CpuHotPlugMismatch",
			cr:   &v1alpha1.Volume{Spec: v1alpha1.VolumeSpec{ForProvider: v1alpha1.VolumeParameters{Name: name, CPUHotPlug: false}}},
			volume: &sdkgo.Volume{
				Properties: &sdkgo.VolumeProperties{Name: &name, CpuHotPlug: &trueVal},
			},
			want:     false,
			wantDiff: "CpuHotPlug does not match the one in the CR: true != false",
		},
		{
			name: "RamHotPlugMismatch",
			cr:   &v1alpha1.Volume{Spec: v1alpha1.VolumeSpec{ForProvider: v1alpha1.VolumeParameters{Name: name, RAMHotPlug: false}}},
			volume: &sdkgo.Volume{
				Properties: &sdkgo.VolumeProperties{Name: &name, RamHotPlug: &trueVal},
			},
			want:     false,
			wantDiff: "RamHotPlug does not match the one in the CR: true != false",
		},
		{
			name: "NicHotPlugMismatch",
			cr:   &v1alpha1.Volume{Spec: v1alpha1.VolumeSpec{ForProvider: v1alpha1.VolumeParameters{Name: name, NicHotPlug: false}}},
			volume: &sdkgo.Volume{
				Properties: &sdkgo.VolumeProperties{Name: &name, NicHotPlug: &trueVal},
			},
			want:     false,
			wantDiff: "NicHotPlug does not match the one in the CR: true != false",
		},
		{
			name: "NicHotUnplugMismatch",
			cr:   &v1alpha1.Volume{Spec: v1alpha1.VolumeSpec{ForProvider: v1alpha1.VolumeParameters{Name: name, NicHotUnplug: false}}},
			volume: &sdkgo.Volume{
				Properties: &sdkgo.VolumeProperties{Name: &name, NicHotUnplug: &trueVal},
			},
			want:     false,
			wantDiff: "NicHotUnplug does not match the one in the CR: true != false",
		},
		{
			name: "DiscVirtioHotPlugMismatch",
			cr:   &v1alpha1.Volume{Spec: v1alpha1.VolumeSpec{ForProvider: v1alpha1.VolumeParameters{Name: name, DiscVirtioHotPlug: false}}},
			volume: &sdkgo.Volume{
				Properties: &sdkgo.VolumeProperties{Name: &name, DiscVirtioHotPlug: &trueVal},
			},
			want:     false,
			wantDiff: "DiscVirtioHotPlug does not match the one in the CR: true != false",
		},
		{
			name: "DiscVirtioHotUnplugMismatch",
			cr:   &v1alpha1.Volume{Spec: v1alpha1.VolumeSpec{ForProvider: v1alpha1.VolumeParameters{Name: name, DiscVirtioHotUnplug: false}}},
			volume: &sdkgo.Volume{
				Properties: &sdkgo.VolumeProperties{Name: &name, DiscVirtioHotUnplug: &trueVal},
			},
			want:     false,
			wantDiff: "DiscVirtioHotUnplug does not match the one in the CR: true != false",
		},
		{
			name: "BusMismatch",
			cr:   &v1alpha1.Volume{Spec: v1alpha1.VolumeSpec{ForProvider: v1alpha1.VolumeParameters{Name: name, Bus: otherBus}}},
			volume: &sdkgo.Volume{
				Properties: &sdkgo.VolumeProperties{Name: &name, Bus: &bus},
			},
			want:     false,
			wantDiff: "Volume bus does not match the desired bus: VIRTIO != IDE",
		},
		{
			name: "UpToDate",
			cr:   &v1alpha1.Volume{Spec: v1alpha1.VolumeSpec{ForProvider: v1alpha1.VolumeParameters{Name: name, Size: size}}},
			volume: &sdkgo.Volume{
				Properties: &sdkgo.VolumeProperties{Name: &name, Size: &size},
			},
			want:     true,
			wantDiff: "Volume is up-to-date",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			upToDate, wantDiff := IsUpToDateWithDiff(tt.cr, tt.volume)
			assert.Equal(t, tt.want, upToDate)
			assert.Equal(t, tt.wantDiff, wantDiff)
		})
	}
}
