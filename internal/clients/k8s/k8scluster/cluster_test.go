package k8scluster

import (
	"testing"

	"github.com/ionos-cloud/sdk-go-bundle/shared"
	ionoscloud "github.com/ionos-cloud/sdk-go/v6"

	"github.com/ionos-cloud/crossplane-provider-ionoscloud/apis/k8s/v1alpha1"
)

func TestIsUpToDate(t *testing.T) {
	type args struct {
		cr      *v1alpha1.Cluster
		cluster ionoscloud.KubernetesCluster
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			name: "both are empty",
			args: args{
				cr:      nil,
				cluster: ionoscloud.KubernetesCluster{},
			},
			want: true,
		},
		{
			name: "cr is empty",
			args: args{
				cr: nil,
				cluster: ionoscloud.KubernetesCluster{Properties: &ionoscloud.KubernetesClusterProperties{
					K8sVersion: shared.ToPtr("v1.2.3"),
				}},
			},
			want: false,
		},
		{
			name: "api response is empty",
			args: args{
				cr: &v1alpha1.Cluster{
					Spec: v1alpha1.ClusterSpec{},
				},
				cluster: ionoscloud.KubernetesCluster{},
			},
			want: false,
		},
		{
			name: "both are up to date",
			args: args{
				cr: &v1alpha1.Cluster{
					Spec: v1alpha1.ClusterSpec{
						ForProvider: v1alpha1.ClusterParameters{
							K8sVersion: "v1.2.3",
							MaintenanceWindow: v1alpha1.MaintenanceWindow{
								Time:         "15:24:30Z",
								DayOfTheWeek: "Mon",
							},
						},
					},
				},
				cluster: ionoscloud.KubernetesCluster{Properties: &ionoscloud.KubernetesClusterProperties{
					K8sVersion: shared.ToPtr("v1.2.3"),
					MaintenanceWindow: &ionoscloud.KubernetesMaintenanceWindow{
						DayOfTheWeek: shared.ToPtr("Mon"),
						Time:         shared.ToPtr("15:24:30Z"),
					},
				}}},
			want: true,
		},

		{
			name: "only maintenance window differ",
			args: args{
				cr: &v1alpha1.Cluster{
					Spec: v1alpha1.ClusterSpec{
						ForProvider: v1alpha1.ClusterParameters{
							K8sVersion: "v1.2.3",
							MaintenanceWindow: v1alpha1.MaintenanceWindow{
								Time:         "02:24:30Z",
								DayOfTheWeek: "Fri",
							},
						},
					},
				},
				cluster: ionoscloud.KubernetesCluster{Properties: &ionoscloud.KubernetesClusterProperties{
					K8sVersion: shared.ToPtr("v1.2.3"),
					MaintenanceWindow: &ionoscloud.KubernetesMaintenanceWindow{
						DayOfTheWeek: shared.ToPtr("Mon"),
						Time:         shared.ToPtr("15:24:30Z"),
					},
				}}},
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsK8sClusterUpToDate(tt.args.cr, tt.args.cluster); got != tt.want {
				t.Errorf("isUpToDate() = %v, want %v", got, tt.want)
			}
		})
	}
}
