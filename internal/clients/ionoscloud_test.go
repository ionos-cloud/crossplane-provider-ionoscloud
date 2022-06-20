package clients

import (
	"net/http"
	"os"
	"testing"

	ionosdbaas "github.com/ionos-cloud/sdk-go-dbaas-postgres"
	ionos "github.com/ionos-cloud/sdk-go/v6"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	expectedUserAgentCompute = "crossplane-provider-ionoscloud_ionos-cloud-sdk-go/v6.1.0"
	expectedUserAgentDbaas   = "crossplane-provider-ionoscloud_ionos-cloud-sdk-go-dbaas-postgres/vv1.0.3"

	hostnameFromSecret = "https://host"
	hostnameFromEnv    = "http://host-from-env"
)

func setComputeDefaults(cfg *ionos.Configuration) {
	cfg.HTTPClient = http.DefaultClient
	cfg.UserAgent = expectedUserAgentCompute
}

func setDbaaSDefaults(cfg *ionosdbaas.Configuration) {
	cfg.HTTPClient = http.DefaultClient
	cfg.UserAgent = expectedUserAgentDbaas
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
		wantDbaasConfig   *ionosdbaas.Configuration
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
			wantDbaasConfig: func() *ionosdbaas.Configuration {
				cfg := ionosdbaas.NewConfiguration("username", "password", "", "")
				setDbaaSDefaults(cfg)
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
			wantDbaasConfig: func() *ionosdbaas.Configuration {
				cfg := ionosdbaas.NewConfiguration("username", "password", "token", hostnameFromSecret)
				setDbaaSDefaults(cfg)
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
			wantDbaasConfig: func() *ionosdbaas.Configuration {
				cfg := ionosdbaas.NewConfiguration("username", "password", "token", hostnameFromEnv)
				setDbaaSDefaults(cfg)
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
			wantDbaasConfig: func() *ionosdbaas.Configuration {
				cfg := ionosdbaas.NewConfiguration("username", "password", "token", hostnameFromSecret)
				setDbaaSDefaults(cfg)
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
				assert.Equal(t, tt.wantComputeConfig, got.ComputeClient.GetConfig())
				assert.Equal(t, tt.wantDbaasConfig, got.DBaaSPostgresClient.GetConfig())
			} else {
				assert.Nil(t, got)
			}
		})
	}
}
