package serverset

import "testing"

func TestGetZoneFromIndex(t *testing.T) {
	type args struct {
		deplOptions ZoneDeploymentOptions
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{
			name: "index1ExpectedZone_1",
			args: args{
				deplOptions: ZoneDeploymentOptions{
					Index: 0,
				},
			},
			want: "ZONE_1",
		},
		{
			name: "index1ExpectedZone_2",
			args: args{
				deplOptions: ZoneDeploymentOptions{
					Index: 1,
				},
			},
			want: "ZONE_2",
		},
		{
			name: "index2ExpectedZone_1",
			args: args{
				deplOptions: ZoneDeploymentOptions{
					Index: 2,
				},
			},
			want: "ZONE_1",
		},
		{
			name: "index10ExpectedZone_1",
			args: args{
				deplOptions: ZoneDeploymentOptions{
					Index: 10,
				},
			},
			want: "ZONE_1",
		},
		{
			name: "index111ExpectedZone_2",
			args: args{
				deplOptions: ZoneDeploymentOptions{
					Index: 111,
				},
			},
			want: "ZONE_2",
		},
	}
	depl := NewZoneDeploymentByType("ZONES")
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := depl.GetZone(tt.args.deplOptions); got != tt.want {
				t.Errorf("GetZoneFromIndex() = %v, want %v", got, tt.want)
			}
		})
	}
}
