package clients

import (
	"fmt"
	"net/http"
	"os"
	"testing"
	"time"

	"github.com/ionos-cloud/sdk-go-bundle/shared"

	"github.com/ionos-cloud/crossplane-provider-ionoscloud/internal/version"

	xpv1 "github.com/crossplane/crossplane-runtime/apis/common/v1"
	"github.com/ionos-cloud/sdk-go-bundle/products/dbaas/psql/v2"
	ionos "github.com/ionos-cloud/sdk-go/v6"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ionos-cloud/crossplane-provider-ionoscloud/internal/clients/compute"
	"github.com/ionos-cloud/crossplane-provider-ionoscloud/internal/clients/k8s"
)

const (
	hostnameFromSecret = "https://host"
	hostnameFromEnv    = "http://host-from-env"
)

func setComputeDefaults(cfg *ionos.Configuration) {
	cfg.HTTPClient = http.DefaultClient
	cfg.UserAgent = fmt.Sprintf("%v/%v_ionos-cloud-sdk-go/v%v", UserAgent, version.Version, ionos.Version)
}

func setDbaaSDefaults(cfg *shared.Configuration) {
	cfg.HTTPClient = http.DefaultClient
	cfg.UserAgent = fmt.Sprintf("%v/sdk_go_bundle_%v_%v", UserAgent, version.Version, psql.Version)
	cfg.DefaultHeader = nil
	cfg.DefaultQueryParams = nil

}

func TestNewIonosClient(t *testing.T) {

	type args struct {
		data []byte
	}
	tests := []struct {
		name              string
		args              args
		env               map[string]string
		wantComputeConfig *ionos.Configuration
		wantDbaasConfig   *shared.Configuration
		wantErr           bool
	}{
		{
			name:              "nil data",
			args:              args{data: nil},
			wantComputeConfig: nil,
			wantErr:           true,
		},
		{
			name: "basic auth",
			args: args{data: []byte(`{"user": "username","password": "cGFzc3dvcmQ="}`)},
			wantComputeConfig: func() *ionos.Configuration {
				cfg := ionos.NewConfiguration("username", "password", "", "")
				setComputeDefaults(cfg)
				return cfg
			}(),
			wantDbaasConfig: func() *shared.Configuration {
				cfg := shared.NewConfiguration("username", "password", "", "")
				setDbaaSDefaults(cfg)
				cfg.Servers = shared.ServerConfigurations{
					{
						URL:         "https://api.ionos.com/databases/postgresql",
						Description: "Production",
					},
				}
				return cfg
			}(),
			wantErr: false,
		},
		{
			name: "2fa token auth and host url",
			args: args{data: []byte(`{"user": "username","password": "cGFzc3dvcmQ=", "token": "token", "host_url":"https://host"}`)},
			wantComputeConfig: func() *ionos.Configuration {
				cfg := ionos.NewConfiguration("username", "password", "token", hostnameFromSecret)
				setComputeDefaults(cfg)
				return cfg
			}(),
			wantDbaasConfig: func() *shared.Configuration {
				cfg := shared.NewConfiguration("username", "password", "token", hostnameFromSecret)
				setDbaaSDefaults(cfg)
				cfg.Servers[0].URL = "https://host/databases/postgresql"
				return cfg
			}(),
			wantErr: false,
		},
		{
			name: "2fa token auth and global host url",
			env:  map[string]string{"IONOS_API_URL": "http://host-from-env"},
			args: args{data: []byte(`{"user": "username","password": "cGFzc3dvcmQ=", "token": "token"}`)},
			wantComputeConfig: func() *ionos.Configuration {
				cfg := ionos.NewConfiguration("username", "password", "token", hostnameFromEnv)
				setComputeDefaults(cfg)
				return cfg
			}(),
			wantDbaasConfig: func() *shared.Configuration {
				cfg := shared.NewConfiguration("username", "password", "token", hostnameFromEnv)
				setDbaaSDefaults(cfg)
				cfg.Servers[0].URL = "http://host-from-env/databases/postgresql"
				return cfg
			}(),
			wantErr: false,
		},
		{
			name: "2fa token auth dont overwrite secret specific with global host url",
			env:  map[string]string{"IONOS_API_URL": hostnameFromEnv},
			args: args{data: []byte(`{"user": "username","password": "cGFzc3dvcmQ=", "token": "token", "host_url":"https://host"}`)},
			wantComputeConfig: func() *ionos.Configuration {
				cfg := ionos.NewConfiguration("username", "password", "token", hostnameFromSecret)
				setComputeDefaults(cfg)
				return cfg
			}(),
			wantDbaasConfig: func() *shared.Configuration {
				cfg := shared.NewConfiguration("username", "password", "token", hostnameFromSecret)
				setDbaaSDefaults(cfg)
				cfg.Servers[0].URL = "https://host/databases/postgresql"

				return cfg
			}(),
			wantErr: false,
		},
		{
			name:              "malformed json",
			args:              args{data: []byte(`{"user": "foo",`)},
			wantComputeConfig: nil,
			wantDbaasConfig:   nil,
			wantErr:           true,
		},
		{
			name: "malformed base64 password",
			args: args{
				data: []byte(`{"user": "username","password": "cGFzc3dvcm", "token": "token", "host_url": "foo"}`),
			},
			wantComputeConfig: nil,
			wantDbaasConfig:   nil,
			wantErr:           true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			for name, value := range tt.env {
				require.NoError(t, os.Setenv(name, value))
			}
			loadEnv()
			defer func() {
				for name := range tt.env {
					require.NoError(t, os.Unsetenv(name))
				}
				loadEnv()
			}()

			got, err := NewIonosClients(tt.args.data)
			if tt.wantErr {
				assert.NotNil(t, err)
			} else {
				assert.Nil(t, err)
			}
			if tt.wantComputeConfig != nil {
				require.NotNil(t, got)
				ccfg := got.ComputeClient.GetConfig()
				dcfg := got.DBaaSPostgresClient.GetConfig()
				// Drop Logger from the comparison as log.Logger structs cannot be compared with DeepEqual
				ccfg.Logger = nil
				tt.wantComputeConfig.Logger = nil

				assert.Equal(t, tt.wantComputeConfig, ccfg)
				assert.Equal(t, tt.wantDbaasConfig, dcfg)
			} else {
				assert.Nil(t, got)
			}
		})
	}
}

func TestGetCoreResourceState(t *testing.T) {

	tests := []struct {
		name string
		args *testCoreResource
		want string
	}{
		{
			name: "nil test resource",
			args: nil,
			want: "",
		},
		{
			name: "found nil metadata",
			args: &testCoreResource{metadata: nil, found: true},
			want: "",
		},
		{
			name: "found metadata with nil state",
			args: &testCoreResource{metadata: &ionos.DatacenterElementMetadata{State: nil}, found: true},
			want: "",
		},
		{
			name: "found metadata with state",
			args: &testCoreResource{metadata: &ionos.DatacenterElementMetadata{State: ionos.PtrString("foo")}, found: true},
			want: "foo",
		},
		{
			name: "found metadata no metadata, but it's present",
			args: &testCoreResource{metadata: &ionos.DatacenterElementMetadata{State: ionos.PtrString("foo")}, found: false},
			want: "",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, GetCoreResourceState(tt.args))
		})
	}
}

type testCoreResource struct {
	metadata *ionos.DatacenterElementMetadata
	found    bool
}

func (t *testCoreResource) GetMetadataOk() (*ionos.DatacenterElementMetadata, bool) {
	if t == nil {
		return nil, false
	}
	return t.metadata, t.found
}

func TestGetDBaaSResourceState(t *testing.T) {

	ptrState := func(in string) *psql.State {
		state := psql.State(in)
		return &state
	}

	tests := []struct {
		name string
		args *testDbaaSResource
		want psql.State
	}{
		{
			name: "nil test resource",
			args: nil,
			want: "",
		},
		{
			name: "found nil metadata",
			args: &testDbaaSResource{metadata: nil, found: true},
			want: "",
		},
		{
			name: "found metadata with nil state",
			args: &testDbaaSResource{metadata: &psql.ClusterMetadata{State: nil}, found: true},
			want: "",
		},
		{
			name: "found metadata with state",
			args: &testDbaaSResource{metadata: &psql.ClusterMetadata{State: ptrState("foo")}, found: true},
			want: "foo",
		},
		{
			name: "found metadata no metadata, but it's present",
			args: &testDbaaSResource{metadata: &psql.ClusterMetadata{State: ptrState("foo")}, found: false},
			want: "",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, GetDBaaSPsqlResourceState(tt.args))
		})
	}
}

type testDbaaSResource struct {
	metadata *psql.ClusterMetadata
	found    bool
}

func (t *testDbaaSResource) GetMetadataOk() (*psql.ClusterMetadata, bool) {
	if t == nil {
		return nil, false
	}
	return t.metadata, t.found
}

type testConditionedResource struct {
	t                 *testing.T
	expectedCondition xpv1.Condition
}

func (t testConditionedResource) SetConditions(c ...xpv1.Condition) {
	assert.Len(t.t, c, 1)
	fixedTime := time.Now()
	t.expectedCondition.LastTransitionTime.Time = fixedTime
	c[0].LastTransitionTime.Time = fixedTime
	assert.Equal(t.t, t.expectedCondition, c[0])
}

func TestUpdateCondition(t *testing.T) {

	tests := []struct {
		name     string
		states   []string
		resource testConditionedResource
	}{
		{
			name:     "creating",
			states:   []string{compute.BUSY, k8s.BUSY, string(psql.STATE_BUSY), k8s.DEPLOYING},
			resource: testConditionedResource{expectedCondition: xpv1.Creating()},
		},
		{
			name:     "destroying",
			states:   []string{string(psql.STATE_DESTROYING), k8s.DESTROYING, compute.DESTROYING, k8s.TERMINATED},
			resource: testConditionedResource{expectedCondition: xpv1.Deleting()},
		},
		{
			name:     "available",
			states:   []string{string(psql.STATE_AVAILABLE), compute.AVAILABLE, compute.ACTIVE, k8s.ACTIVE, k8s.AVAILABLE},
			resource: testConditionedResource{expectedCondition: xpv1.Available()},
		},
		{
			name:     "unavailable",
			states:   []string{string(psql.STATE_FAILED), string(psql.STATE_UNKNOWN), "", "FOOBAR"},
			resource: testConditionedResource{expectedCondition: xpv1.Unavailable()},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.resource.t = t
			for _, state := range tt.states {
				UpdateCondition(tt.resource, state)
			}

		})
	}
}
