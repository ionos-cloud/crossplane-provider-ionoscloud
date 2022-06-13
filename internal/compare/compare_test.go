package compare

import (
	"testing"

	ionoscloud "github.com/ionos-cloud/sdk-go/v6"
	"github.com/stretchr/testify/assert"

	"github.com/ionos-cloud/crossplane-provider-ionoscloud/apis/k8s/v1alpha1"
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
				observedValue: PointerString("foo"),
			},
			want: true,
		},
		{
			name: "non-empty observed, empty target",
			args: args{
				targetValue:   "",
				observedValue: PointerString("foo"),
			},
			want: false,
		},
		{
			name: "empty observed, empty target",
			args: args{
				targetValue:   "",
				observedValue: PointerString(""),
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

func TestEqualMaintananceWindow(t *testing.T) {
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
					DayOfTheWeek: PointerString("foo"),
					Time:         PointerString("13:00:44"),
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
					DayOfTheWeek: PointerString("foo"),
					Time:         PointerString("13:32:44Z"),
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
					DayOfTheWeek: PointerString("foo"),
					Time:         PointerString("13:00:44Z"),
				},
			},
			want: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, EqualMaintananceWindow(tt.args.targetValue, tt.args.observedValue))
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
				observedValue: PointerString("13:00:44Z"),
			},
			want: true,
		},
		{
			name: "both values equal, missing Z",
			args: args{
				targetValue:   "13:00:44",
				observedValue: PointerString("13:00:44Z"),
			},
			want: true,
		},
		{
			name: "unparseable target value",
			args: args{
				targetValue:   "13:00:44:44:333",
				observedValue: PointerString("13:00:44Z"),
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

func PointerString(in string) *string {
	return &in
}
