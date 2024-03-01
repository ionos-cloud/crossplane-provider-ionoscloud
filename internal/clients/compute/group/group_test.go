package group

import (
	"testing"

	psql "github.com/ionos-cloud/sdk-go-dbaas-postgres"
	ionoscloud "github.com/ionos-cloud/sdk-go/v6"

	"github.com/ionos-cloud/crossplane-provider-ionoscloud/apis/compute/v1alpha1"
)

func TestIsGroupUpToDate(t *testing.T) {

	type args struct {
		cr    *v1alpha1.Group
		Group ionoscloud.Group
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			name: "both empty",
			args: args{
				cr:    nil,
				Group: ionoscloud.Group{},
			},
			want: true,
		},
		{
			name: "cr empty",
			args: args{
				cr:    nil,
				Group: ionoscloud.Group{Properties: &ionoscloud.GroupProperties{Name: psql.ToPtr("meow")}},
			},
			want: false,
		},
		{
			name: "observed empty",
			args: args{
				cr: &v1alpha1.Group{
					Spec: v1alpha1.GroupSpec{
						ForProvider: v1alpha1.GroupParameters{
							Name: "foo",
						},
					},
				},
				Group: ionoscloud.Group{},
			},
			want: false,
		},
		{
			name: "equal properties",
			args: args{
				cr: &v1alpha1.Group{
					Spec: v1alpha1.GroupSpec{
						ForProvider: v1alpha1.GroupParameters{
							Name:                 "foo",
							CreateDataCenter:     true,
							CreateInternetAccess: true,
							ReserveIP:            true,
							CreateK8sCluster:     true,
						},
					},
				},
				Group: ionoscloud.Group{Properties: &ionoscloud.GroupProperties{
					Name:                 psql.ToPtr("foo"),
					CreateDataCenter:     psql.ToPtr(true),
					CreateInternetAccess: psql.ToPtr(true),
					ReserveIp:            psql.ToPtr(true),
					CreateK8sCluster:     psql.ToPtr(true),
				},
				},
			},
			want: true,
		},
		{
			name: "different properties",
			args: args{
				cr: &v1alpha1.Group{
					Spec: v1alpha1.GroupSpec{
						ForProvider: v1alpha1.GroupParameters{
							Name:                 "meow",
							CreateDataCenter:     true,
							CreateInternetAccess: false,
							ReserveIP:            true,
							CreateK8sCluster:     true,
						},
					},
				},
				Group: ionoscloud.Group{Properties: &ionoscloud.GroupProperties{
					Name:                 psql.ToPtr("foo"),
					CreateDataCenter:     psql.ToPtr(true),
					CreateInternetAccess: psql.ToPtr(true),
					ReserveIp:            psql.ToPtr(true),
					CreateK8sCluster:     psql.ToPtr(false),
				},
				},
			},
			want: false,
		},
		{
			name: "equal properties with members and shares",
			args: args{
				cr: &v1alpha1.Group{
					Spec: v1alpha1.GroupSpec{
						ForProvider: v1alpha1.GroupParameters{
							Name:                 "foo",
							CreateDataCenter:     true,
							CreateInternetAccess: true,
							ReserveIP:            true,
							CreateK8sCluster:     true,
							UserCfg: []v1alpha1.UserConfig{
								{UserID: "17dc05fa-8e39-4d68-9529-5a428494882a"},
								{UserID: "338fa0ac-ab6e-4add-90d9-4850b743371f"},
								{UserID: "777c08be-1c95-4b6f-a15d-02994f3de1a1"},
							},
							ResourceShareCfg: []v1alpha1.ResourceShareConfig{
								{
									ResourceShare: v1alpha1.ResourceShare{ResourceID: "eb55a2d5-bb57-464f-b44f-f843dd059895", EditPrivilege: true, SharePrivilege: true},
								}, {
									ResourceShare: v1alpha1.ResourceShare{ResourceID: "5ad718b2-18c4-4923-974b-70a1df39f64c", EditPrivilege: false, SharePrivilege: true},
								}, {
									ResourceShare: v1alpha1.ResourceShare{ResourceID: "f3fdf8fc-e736-4b7d-9c70-f69cf50882a9", EditPrivilege: true, SharePrivilege: false},
								},
							},
						},
					},
					Status: v1alpha1.GroupStatus{
						AtProvider: v1alpha1.GroupObservation{
							UserIDs: []string{
								"777c08be-1c95-4b6f-a15d-02994f3de1a1",
								"338fa0ac-ab6e-4add-90d9-4850b743371f",
								"17dc05fa-8e39-4d68-9529-5a428494882a",
							},
							ResourceShares: []v1alpha1.ResourceShare{
								{ResourceID: "f3fdf8fc-e736-4b7d-9c70-f69cf50882a9", EditPrivilege: true, SharePrivilege: false},
								{ResourceID: "5ad718b2-18c4-4923-974b-70a1df39f64c", EditPrivilege: false, SharePrivilege: true},
								{ResourceID: "eb55a2d5-bb57-464f-b44f-f843dd059895", EditPrivilege: true, SharePrivilege: true},
							},
						},
					},
				},
				Group: ionoscloud.Group{Properties: &ionoscloud.GroupProperties{
					Name:                 psql.ToPtr("foo"),
					CreateDataCenter:     psql.ToPtr(true),
					CreateInternetAccess: psql.ToPtr(true),
					ReserveIp:            psql.ToPtr(true),
					CreateK8sCluster:     psql.ToPtr(true),
				},
				},
			},
			want: true,
		},

		{
			name: "different members",
			args: args{
				cr: &v1alpha1.Group{
					Spec: v1alpha1.GroupSpec{
						ForProvider: v1alpha1.GroupParameters{
							Name: "meow",
							UserCfg: []v1alpha1.UserConfig{
								{UserID: "17dc05fa-8e39-4d68-9529-5a428494882a"},
								{UserID: "338fa0ac-ab6e-4add-90d9-4850b743371f"},
								{UserID: "777c08be-1c95-4b6f-a15d-02994f3de1a1"},
							},
						},
					},
					Status: v1alpha1.GroupStatus{
						AtProvider: v1alpha1.GroupObservation{
							UserIDs: []string{
								"17dc05fa-8e39-4d68-9529-5a428494882a",
								"338fa0ac-ab6e-4add-90d9-4850b743371f",
							},
						},
					},
				},
				Group: ionoscloud.Group{Properties: &ionoscloud.GroupProperties{
					Name: psql.ToPtr("meow")},
				},
			},
			want: false,
		},
		{
			name: "different shares",
			args: args{
				cr: &v1alpha1.Group{
					Spec: v1alpha1.GroupSpec{
						ForProvider: v1alpha1.GroupParameters{
							Name: "meow",
							ResourceShareCfg: []v1alpha1.ResourceShareConfig{
								{
									ResourceShare: v1alpha1.ResourceShare{ResourceID: "f3fdf8fc-e736-4b7d-9c70-f69cf50882a9", EditPrivilege: true, SharePrivilege: false},
								}, {}, {
									ResourceShare: v1alpha1.ResourceShare{ResourceID: "5ad718b2-18c4-4923-974b-70a1df39f64c", EditPrivilege: false, SharePrivilege: true},
								},
							},
						},
					},
					Status: v1alpha1.GroupStatus{
						AtProvider: v1alpha1.GroupObservation{
							ResourceShares: []v1alpha1.ResourceShare{
								{ResourceID: "f3fdf8fc-e736-4b7d-9c70-f69cf50882a9", EditPrivilege: false, SharePrivilege: false},
								{ResourceID: "5ad718b2-18c4-4923-974b-70a1df39f64c", EditPrivilege: false, SharePrivilege: true},
							},
						},
					},
				},
				Group: ionoscloud.Group{Properties: &ionoscloud.GroupProperties{
					Name: psql.ToPtr("meow")},
				},
			},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsGroupUpToDate(tt.args.cr, tt.args.Group); got != tt.want {
				t.Errorf("IsGroupUpToDate() = %v, want %v", got, tt.want)
			}
		})
	}

}
