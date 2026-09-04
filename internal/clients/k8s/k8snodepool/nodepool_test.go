package k8snodepool

import (
	"slices"
	"testing"

	ionoscloud "github.com/ionos-cloud/sdk-go/v6"

	"github.com/ionos-cloud/crossplane-provider-ionoscloud/apis/k8s/v1alpha1"
)

func TestGenerateUpdateK8sNodePoolInput_AutoScalingOmitsNodeCount(t *testing.T) {
	cr := &v1alpha1.NodePool{
		Spec: v1alpha1.NodePoolSpec{
			ForProvider: v1alpha1.NodePoolParameters{
				Name:       "test-pool",
				ServerType: "VCPU",
				NodeCount:  3,
				AutoScaling: v1alpha1.KubernetesAutoScaling{
					MinNodeCount: 2,
					MaxNodeCount: 10,
				},
			},
		},
	}

	result := GenerateUpdateK8sNodePoolInput(cr, nil)

	if result.Properties.NodeCount != nil {
		t.Errorf("NodeCount should be nil when AutoScaling is set, got %d", *result.Properties.NodeCount)
	}
	if result.Properties.AutoScaling == nil {
		t.Fatal("AutoScaling should be set")
	}
	if *result.Properties.AutoScaling.MinNodeCount != 2 {
		t.Errorf("MinNodeCount = %d, want 2", *result.Properties.AutoScaling.MinNodeCount)
	}
	if *result.Properties.AutoScaling.MaxNodeCount != 10 {
		t.Errorf("MaxNodeCount = %d, want 10", *result.Properties.AutoScaling.MaxNodeCount)
	}
}

func TestGenerateUpdateK8sNodePoolInput_NoAutoScalingIncludesNodeCount(t *testing.T) {
	cr := &v1alpha1.NodePool{
		Spec: v1alpha1.NodePoolSpec{
			ForProvider: v1alpha1.NodePoolParameters{
				Name:       "test-pool",
				ServerType: "VCPU",
				NodeCount:  5,
			},
		},
	}

	result := GenerateUpdateK8sNodePoolInput(cr, nil)

	if result.Properties.NodeCount == nil {
		t.Fatal("NodeCount should be set when AutoScaling is not configured")
	}
	if *result.Properties.NodeCount != 5 {
		t.Errorf("NodeCount = %d, want 5", *result.Properties.NodeCount)
	}
	if result.Properties.AutoScaling != nil {
		t.Errorf("AutoScaling should be nil when not configured")
	}
}

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
					Name: new("foo"),
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
							ServerType:       "VCPU",
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
					Name:             new("not empty"),
					DatacenterId:     new("my-dc"),
					NodeCount:        ionoscloud.PtrInt32(4),
					CpuFamily:        new("super fast"),
					CoresCount:       ionoscloud.PtrInt32(5),
					RamSize:          ionoscloud.PtrInt32(6),
					AvailabilityZone: new("AUTO"),
					StorageType:      new("SSD"),
					StorageSize:      ionoscloud.PtrInt32(7),
					K8sVersion:       new("v1.22.33"),
					ServerType:       ionoscloud.ToPtr(ionoscloud.KubernetesNodePoolServerType("VCPU")),
					MaintenanceWindow: &ionoscloud.KubernetesMaintenanceWindow{
						DayOfTheWeek: new("Mon"),
						Time:         new("15:24:30Z"),
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
							ServerType:       "VCPU",
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
					Name:             new("not empty"),
					DatacenterId:     new("my-dc"),
					NodeCount:        ionoscloud.PtrInt32(4),
					CpuFamily:        new("super fast"),
					CoresCount:       ionoscloud.PtrInt32(5),
					RamSize:          ionoscloud.PtrInt32(6),
					AvailabilityZone: new("AUTO"),
					StorageType:      new("SSD"),
					StorageSize:      ionoscloud.PtrInt32(7),
					K8sVersion:       new("v1.22.33"),
					ServerType:       ionoscloud.ToPtr(ionoscloud.KubernetesNodePoolServerType("DedicatedCore")),
					MaintenanceWindow: &ionoscloud.KubernetesMaintenanceWindow{
						DayOfTheWeek: new("Mon"),
						Time:         new("15:24:30Z"),
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
					Name:             new("not empty"),
					DatacenterId:     new("my-dc"),
					NodeCount:        ionoscloud.PtrInt32(4),
					CpuFamily:        new("super fast"),
					CoresCount:       ionoscloud.PtrInt32(5),
					RamSize:          ionoscloud.PtrInt32(6),
					AvailabilityZone: new("AUTO"),
					StorageType:      new("SSD"),
					StorageSize:      ionoscloud.PtrInt32(7),
					K8sVersion:       new("v1.22.33"),
					MaintenanceWindow: &ionoscloud.KubernetesMaintenanceWindow{
						DayOfTheWeek: new("Fri"),
						Time:         new("03:24:30Z"),
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

func TestLateStatusInitializer(t *testing.T) {
	cr := &v1alpha1.NodePool{
		Spec: v1alpha1.NodePoolSpec{
			ForProvider: v1alpha1.NodePoolParameters{
				Name:             "not empty",
				K8sVersion:       "v1.22.33",
				ServerType:       "VCPU",
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
		Status: v1alpha1.NodePoolStatus{
			AtProvider: v1alpha1.NodePoolObservation{},
		},
	}

	nodePool := ionoscloud.KubernetesNodePool{Properties: &ionoscloud.KubernetesNodePoolProperties{
		Name:             new("not empty"),
		DatacenterId:     new("my-dc"),
		NodeCount:        ionoscloud.PtrInt32(4),
		CpuFamily:        new("super fast"),
		CoresCount:       ionoscloud.PtrInt32(5),
		RamSize:          ionoscloud.PtrInt32(6),
		AvailabilityZone: new("AUTO"),
		StorageType:      new("SSD"),
		StorageSize:      ionoscloud.PtrInt32(7),
		K8sVersion:       new("v1.22.33"),
		ServerType:       ionoscloud.ToPtr(ionoscloud.KubernetesNodePoolServerType("VCPU")),
		MaintenanceWindow: &ionoscloud.KubernetesMaintenanceWindow{
			DayOfTheWeek: new("Fri"),
			Time:         new("03:24:30Z"),
		},
		AutoScaling:              nil,
		Lans:                     nil,
		Labels:                   nil,
		Annotations:              nil,
		PublicIps:                &[]string{"172.10.1.1"},
		AvailableUpgradeVersions: &[]string{"v1.22.33", "v1.22.34"},
	}}

	LateStatusInitializer(&cr.Status.AtProvider, &nodePool)

	if *cr.Status.AtProvider.NodeCount != *nodePool.Properties.NodeCount {
		t.Errorf("NodeCount not equal")
	}

	if cr.Status.AtProvider.CPUFamily != *nodePool.Properties.CpuFamily {
		t.Errorf("CPU Family not equal")
	}

	if !slices.Equal(cr.Status.AtProvider.AvailableUpgradeVersions, *nodePool.Properties.AvailableUpgradeVersions) {
		t.Errorf("AvailableUpgradeVersions not equal")
	}

	if !slices.Equal(cr.Status.AtProvider.PublicIPs, *nodePool.Properties.PublicIps) {
		t.Errorf("PublicIPs not equal")
	}
}
