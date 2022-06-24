package k8snodepool

import (
	"testing"

	ionoscloud "github.com/ionos-cloud/sdk-go/v6"

	"github.com/ionos-cloud/crossplane-provider-ionoscloud/apis/k8s/v1alpha1"
)

func TestIsNodePoolUpToDate(t *testing.T) {
	type args struct {
		cr       *v1alpha1.NodePool
		nodePool ionoscloud.KubernetesNodePool
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			name: "both empty",
			args: args{
				cr:       nil,
				nodePool: ionoscloud.KubernetesNodePool{},
			},
			want: true,
		},
		{
			name: "cr empty",
			args: args{
				cr: nil,
				nodePool: ionoscloud.KubernetesNodePool{Properties: &ionoscloud.KubernetesNodePoolProperties{
					Name: ionoscloud.PtrString("foo"),
				}},
			},
			want: false,
		},
		{
			name: "api response empty",
			args: args{
				cr: &v1alpha1.NodePool{
					Spec: v1alpha1.NodePoolSpec{
						ForProvider: v1alpha1.NodePoolParameters{
							Name: "not empty",
						},
					},
				},
				nodePool: ionoscloud.KubernetesNodePool{Properties: nil},
			},
			want: false,
		},
		{
			name: "all equal",
			args: args{
				cr: &v1alpha1.NodePool{
					Spec: v1alpha1.NodePoolSpec{
						ForProvider: v1alpha1.NodePoolParameters{
							Name:             "not empty",
							K8sVersion:       "v1.22.33",
							NodeCount:        4,
							CPUFamily:        "super fast",
							CoresCount:       5,
							RAMSize:          6,
							AvailabilityZone: "AUTO",
							StorageType:      "SSD",
							StorageSize:      7,
							MaintenanceWindow: v1alpha1.MaintenanceWindow{
								Time:         "15:24:30Z",
								DayOfTheWeek: "Mon",
							},
						},
					},
				},
				nodePool: ionoscloud.KubernetesNodePool{Properties: &ionoscloud.KubernetesNodePoolProperties{
					Name:             ionoscloud.PtrString("not empty"),
					DatacenterId:     ionoscloud.PtrString("my-dc"),
					NodeCount:        ionoscloud.PtrInt32(4),
					CpuFamily:        ionoscloud.PtrString("super fast"),
					CoresCount:       ionoscloud.PtrInt32(5),
					RamSize:          ionoscloud.PtrInt32(6),
					AvailabilityZone: ionoscloud.PtrString("AUTO"),
					StorageType:      ionoscloud.PtrString("SSD"),
					StorageSize:      ionoscloud.PtrInt32(7),
					K8sVersion:       ionoscloud.PtrString("v1.22.33"),
					MaintenanceWindow: &ionoscloud.KubernetesMaintenanceWindow{
						DayOfTheWeek: ionoscloud.PtrString("Mon"),
						Time:         ionoscloud.PtrString("15:24:30Z"),
					},
					AutoScaling:              nil,
					Lans:                     nil,
					Labels:                   nil,
					Annotations:              nil,
					PublicIps:                nil,
					AvailableUpgradeVersions: nil,
				}},
			},
			want: true,
		},
		{
			name: "all different",
			args: args{
				cr: &v1alpha1.NodePool{
					Spec: v1alpha1.NodePoolSpec{
						ForProvider: v1alpha1.NodePoolParameters{
							Name:             "different",
							K8sVersion:       "v2.33.55",
							NodeCount:        2,
							CPUFamily:        "super slow",
							CoresCount:       1,
							RAMSize:          10,
							AvailabilityZone: "1",
							StorageType:      "HDD",
							StorageSize:      14,
							MaintenanceWindow: v1alpha1.MaintenanceWindow{
								Time:         "07:33:24",
								DayOfTheWeek: "Fri",
							},
						},
					},
				},
				nodePool: ionoscloud.KubernetesNodePool{Properties: &ionoscloud.KubernetesNodePoolProperties{
					Name:             ionoscloud.PtrString("not empty"),
					DatacenterId:     ionoscloud.PtrString("my-dc"),
					NodeCount:        ionoscloud.PtrInt32(4),
					CpuFamily:        ionoscloud.PtrString("super fast"),
					CoresCount:       ionoscloud.PtrInt32(5),
					RamSize:          ionoscloud.PtrInt32(6),
					AvailabilityZone: ionoscloud.PtrString("AUTO"),
					StorageType:      ionoscloud.PtrString("SSD"),
					StorageSize:      ionoscloud.PtrInt32(7),
					K8sVersion:       ionoscloud.PtrString("v1.22.33"),
					MaintenanceWindow: &ionoscloud.KubernetesMaintenanceWindow{
						DayOfTheWeek: ionoscloud.PtrString("Mon"),
						Time:         ionoscloud.PtrString("15:24:30Z"),
					},
					AutoScaling:              nil,
					Lans:                     nil,
					Labels:                   nil,
					Annotations:              nil,
					PublicIps:                nil,
					AvailableUpgradeVersions: nil,
				}},
			},
			want: false,
		},
		{
			name: "only maintenance window differ",
			args: args{
				cr: &v1alpha1.NodePool{
					Spec: v1alpha1.NodePoolSpec{
						ForProvider: v1alpha1.NodePoolParameters{
							Name:             "not empty",
							K8sVersion:       "v1.22.33",
							NodeCount:        4,
							CPUFamily:        "super fast",
							CoresCount:       5,
							RAMSize:          6,
							AvailabilityZone: "AUTO",
							StorageType:      "SSD",
							StorageSize:      7,
							MaintenanceWindow: v1alpha1.MaintenanceWindow{
								Time:         "15:24:30Z",
								DayOfTheWeek: "Mon",
							},
						},
					},
				},
				nodePool: ionoscloud.KubernetesNodePool{Properties: &ionoscloud.KubernetesNodePoolProperties{
					Name:             ionoscloud.PtrString("not empty"),
					DatacenterId:     ionoscloud.PtrString("my-dc"),
					NodeCount:        ionoscloud.PtrInt32(4),
					CpuFamily:        ionoscloud.PtrString("super fast"),
					CoresCount:       ionoscloud.PtrInt32(5),
					RamSize:          ionoscloud.PtrInt32(6),
					AvailabilityZone: ionoscloud.PtrString("AUTO"),
					StorageType:      ionoscloud.PtrString("SSD"),
					StorageSize:      ionoscloud.PtrInt32(7),
					K8sVersion:       ionoscloud.PtrString("v1.22.33"),
					MaintenanceWindow: &ionoscloud.KubernetesMaintenanceWindow{
						DayOfTheWeek: ionoscloud.PtrString("Fri"),
						Time:         ionoscloud.PtrString("03:24:30Z"),
					},
					AutoScaling:              nil,
					Lans:                     nil,
					Labels:                   nil,
					Annotations:              nil,
					PublicIps:                nil,
					AvailableUpgradeVersions: nil,
				}},
			},
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsK8sNodePoolUpToDate(tt.args.cr, tt.args.nodePool, nil); got != tt.want {
				t.Errorf("isNodePoolUpToDate() = %v, want %v", got, tt.want)
			}
		})
	}
}
