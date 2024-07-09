package compare

import (
	"testing"

	"github.com/ionos-cloud/sdk-go-bundle/shared"
	ionoscloud "github.com/ionos-cloud/sdk-go/v6"
	"github.com/stretchr/testify/assert"

	"github.com/ionos-cloud/crossplane-provider-ionoscloud/apis/k8s/v1alpha1"

	ionosdbaas "github.com/ionos-cloud/sdk-go-bundle/products/dbaas/psql/v2"

	dbaasv1alpha1 "github.com/ionos-cloud/crossplane-provider-ionoscloud/apis/dbaas/postgres/v1alpha1"
)

func TestEqualString(t *testing.T) {
	type args struct {
		targetValue   string
		observedValue *string
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			name: "nil observed, empty target",
			args: args{
				targetValue:   "",
				observedValue: nil,
			},
			want: true,
		},
		{
			name: "nil observed, non-empty target",
			args: args{
				targetValue:   "foo",
				observedValue: nil,
			},
			want: false,
		},
		{
			name: "non-empty observed, non-empty target",
			args: args{
				targetValue:   "foo",
				observedValue: shared.ToPtr("foo"),
			},
			want: true,
		},
		{
			name: "non-empty observed, empty target",
			args: args{
				targetValue:   "",
				observedValue: shared.ToPtr("foo"),
			},
			want: false,
		},
		{
			name: "empty observed, empty target",
			args: args{
				targetValue:   "",
				observedValue: shared.ToPtr(""),
			},
			want: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := EqualString(tt.args.targetValue, tt.args.observedValue)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestEqualKubernetesMaintenanceWindow(t *testing.T) {
	type args struct {
		targetValue   v1alpha1.MaintenanceWindow
		observedValue *ionoscloud.KubernetesMaintenanceWindow
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			name: "nil observed, empty target window",
			args: args{
				targetValue:   v1alpha1.MaintenanceWindow{},
				observedValue: nil,
			},
			want: true,
		},
		{
			name: "targetValue observed, empty target window",
			args: args{
				targetValue: v1alpha1.MaintenanceWindow{},
				observedValue: &ionoscloud.KubernetesMaintenanceWindow{
					DayOfTheWeek: shared.ToPtr("foo"),
					Time:         shared.ToPtr("13:00:44"),
				},
			},
			want: false,
		},
		{
			name: "targetValue observed,non-empty target window, different values",
			args: args{
				targetValue: v1alpha1.MaintenanceWindow{
					DayOfTheWeek: "fri",
					Time:         "13:00:44",
				},
				observedValue: &ionoscloud.KubernetesMaintenanceWindow{
					DayOfTheWeek: shared.ToPtr("foo"),
					Time:         shared.ToPtr("13:32:44Z"),
				},
			},
			want: false,
		},
		{
			name: "targetValue observed,non-empty target window, same values",
			args: args{
				targetValue: v1alpha1.MaintenanceWindow{
					DayOfTheWeek: "foo",
					Time:         "13:00:44",
				},
				observedValue: &ionoscloud.KubernetesMaintenanceWindow{
					DayOfTheWeek: shared.ToPtr("foo"),
					Time:         shared.ToPtr("13:00:44Z"),
				},
			},
			want: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, EqualKubernetesMaintenanceWindow(tt.args.targetValue, tt.args.observedValue))
		})
	}
}

func TestEqualDatabaseMaintenanceWindow(t *testing.T) {
	type args struct {
		targetValue   dbaasv1alpha1.MaintenanceWindow
		observedValue *ionosdbaas.MaintenanceWindow
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			name: "nil observed, empty target window",
			args: args{
				targetValue:   dbaasv1alpha1.MaintenanceWindow{},
				observedValue: nil,
			},
			want: true,
		},
		{
			name: "targetValue observed, empty target window",
			args: args{
				targetValue: dbaasv1alpha1.MaintenanceWindow{},
				observedValue: &ionosdbaas.MaintenanceWindow{
					DayOfTheWeek: "foo",
					Time:         "13:00:44",
				},
			},
			want: false,
		},
		{
			name: "targetValue observed,non-empty target window, different values",
			args: args{
				targetValue: dbaasv1alpha1.MaintenanceWindow{
					DayOfTheWeek: "fri",
					Time:         "13:00:44",
				},
				observedValue: &ionosdbaas.MaintenanceWindow{
					DayOfTheWeek: "foo",
					Time:         "13:32:44Z",
				},
			},
			want: false,
		},
		{
			name: "targetValue observed,non-empty target window, same values",
			args: args{
				targetValue: dbaasv1alpha1.MaintenanceWindow{
					DayOfTheWeek: "foo",
					Time:         "13:00:44",
				},
				observedValue: &ionosdbaas.MaintenanceWindow{
					DayOfTheWeek: "foo",
					Time:         "13:00:44Z",
				},
			},
			want: true,
		},
		{
			name: "targetValue observed,non-empty target window, no day of the week set",
			args: args{
				targetValue: dbaasv1alpha1.MaintenanceWindow{
					DayOfTheWeek: "foo",
					Time:         "13:00:44",
				},
				observedValue: &ionosdbaas.MaintenanceWindow{
					Time: "13:00:44Z",
				},
			},
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, EqualDatabaseMaintenanceWindow(tt.args.targetValue, tt.args.observedValue))
		})
	}
}

func TestEqualTimeString(t *testing.T) {
	type args struct {
		targetValue   string
		observedValue *string
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			name: "nil observed, empty target",
			args: args{
				targetValue:   "",
				observedValue: nil,
			},
			want: true,
		},
		{
			name: "nil observed, non-empty target",
			args: args{
				targetValue:   "13:00:44Z",
				observedValue: nil,
			},
			want: false,
		},
		{
			name: "both values equal",
			args: args{
				targetValue:   "13:00:44Z",
				observedValue: shared.ToPtr("13:00:44Z"),
			},
			want: true,
		},
		{
			name: "both values equal, missing Z",
			args: args{
				targetValue:   "13:00:44",
				observedValue: shared.ToPtr("13:00:44Z"),
			},
			want: true,
		},
		{
			name: "unparseable target value",
			args: args{
				targetValue:   "13:00:44:44:333",
				observedValue: shared.ToPtr("13:00:44Z"),
			},
			want: false,
		},
		{
			name: "unparseable observedValue value",
			args: args{
				targetValue:   "13:00:44Z",
				observedValue: shared.ToPtr("13:00:44:44:333"),
			},
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := EqualTimeString(tt.args.targetValue, tt.args.observedValue); got != tt.want {
				t.Errorf("EqualTimeString() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestEqualConnections(t *testing.T) {
	type args struct {
		targetValue   []dbaasv1alpha1.Connection
		observedValue []ionosdbaas.Connection
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			name: "empty observed, empty target",
			args: args{
				targetValue:   []dbaasv1alpha1.Connection{},
				observedValue: []ionosdbaas.Connection{},
			},
			want: true,
		},
		{
			name: "empty observed, non-empty target",
			args: args{
				targetValue: []dbaasv1alpha1.Connection{
					{
						DatacenterCfg: dbaasv1alpha1.DatacenterConfig{},
						LanCfg: dbaasv1alpha1.LanConfig{
							LanID: "test",
						},
						Cidr: "test",
					},
				},
				observedValue: []ionosdbaas.Connection{},
			},
			want: false,
		},
		// equal true test
		{
			name: "equal connections",
			args: args{
				targetValue: []dbaasv1alpha1.Connection{
					{
						DatacenterCfg: dbaasv1alpha1.DatacenterConfig{
							DatacenterID: "test",
						},
						LanCfg: dbaasv1alpha1.LanConfig{
							LanID: "test",
						},
						Cidr: "test",
					},
				},
				observedValue: []ionosdbaas.Connection{
					{
						DatacenterId: "test",
						LanId:        "test",
						Cidr:         "test",
					},
				},
			},
			want: true,
		},
		{
			name: "multiple equal connections",
			args: args{
				targetValue: []dbaasv1alpha1.Connection{
					{
						DatacenterCfg: dbaasv1alpha1.DatacenterConfig{
							DatacenterID: "test",
						},
						LanCfg: dbaasv1alpha1.LanConfig{
							LanID: "test",
						},
						Cidr: "test",
					},
					{
						DatacenterCfg: dbaasv1alpha1.DatacenterConfig{
							DatacenterID: "test2",
						},
						LanCfg: dbaasv1alpha1.LanConfig{
							LanID: "test2",
						},
						Cidr: "test2",
					},
				},
				observedValue: []ionosdbaas.Connection{
					{
						DatacenterId: "test",
						LanId:        "test",
						Cidr:         "test",
					},
					{
						DatacenterId: "test2",
						LanId:        "test2",
						Cidr:         "test2",
					},
				},
			},
			want: true,
		},
		{
			name: "multiple not equal connections",
			args: args{
				targetValue: []dbaasv1alpha1.Connection{
					{
						DatacenterCfg: dbaasv1alpha1.DatacenterConfig{
							DatacenterID: "test",
						},
						LanCfg: dbaasv1alpha1.LanConfig{
							LanID: "test",
						},
						Cidr: "test",
					},
					{
						DatacenterCfg: dbaasv1alpha1.DatacenterConfig{
							DatacenterID: "test3",
						},
						LanCfg: dbaasv1alpha1.LanConfig{
							LanID: "test2",
						},
						Cidr: "test2",
					},
				},
				observedValue: []ionosdbaas.Connection{
					{
						DatacenterId: "test",
						LanId:        "test",
						Cidr:         "test",
					},
					{
						DatacenterId: "test2",
						LanId:        "test2",
						Cidr:         "test2",
					},
				},
			},
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equalf(t, tt.want, EqualConnections(tt.args.targetValue, tt.args.observedValue), "EqualConnections(%v, %v)", tt.args.targetValue, tt.args.observedValue)
		})
	}
}
