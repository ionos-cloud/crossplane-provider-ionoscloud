package serverset

import "testing"

func TestGetZoneFromIndex(t *testing.T) {
	type args struct {
		index int
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{
			name: "index1ExpectedZone_1",
			args: args{
				index: 0,
			},
			want: "ZONE_1",
		},
		{
			name: "index1ExpectedZone_2",
			args: args{
				index: 1,
			},
			want: "ZONE_2",
		},
		{
			name: "index2ExpectedZone_1",
			args: args{
				index: 2,
			},
			want: "ZONE_1",
		},
		{
			name: "index10ExpectedZone_1",
			args: args{
				index: 10,
			},
			want: "ZONE_1",
		},
		{
			name: "index111ExpectedZone_2",
			args: args{
				index: 111,
			},
			want: "ZONE_2",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := GetZoneFromIndex(tt.args.index); got != tt.want {
				t.Errorf("GetZoneFromIndex() = %v, want %v", got, tt.want)
			}
		})
	}
}
