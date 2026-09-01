package clients

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/base64"
	"encoding/hex"
	"encoding/pem"
	"fmt"
	"math/big"
	"net"
	"net/http"
	"net/http/httptest"
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

// generateTestCertPEM generates a fresh, self-signed ECDSA certificate/key pair at test-run
// time, PEM encoded, for use as either a client certificate or a CA certificate in the MTLS
// tests below. It returns the raw (non-base64) PEM bytes plus the parsed certificate.
func generateTestCertPEM(t *testing.T, commonName string, isCA bool) (certPEM, keyPEM []byte, cert *x509.Certificate) {
	t.Helper()

	priv, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)

	serial, err := rand.Int(rand.Reader, big.NewInt(1<<62))
	require.NoError(t, err)

	template := &x509.Certificate{
		SerialNumber:          serial,
		Subject:               pkix.Name{CommonName: commonName},
		NotBefore:             time.Now().Add(-time.Hour),
		NotAfter:              time.Now().Add(time.Hour),
		KeyUsage:              x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment | x509.KeyUsageCertSign,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth},
		BasicConstraintsValid: true,
		IsCA:                  isCA,
	}

	der, err := x509.CreateCertificate(rand.Reader, template, template, &priv.PublicKey, priv)
	require.NoError(t, err)

	cert, err = x509.ParseCertificate(der)
	require.NoError(t, err)

	certPEM = pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der})

	keyDER, err := x509.MarshalECPrivateKey(priv)
	require.NoError(t, err)
	keyPEM = pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: keyDER})

	return certPEM, keyPEM, cert
}

// b64 base64-encodes raw bytes, matching the convention used for the `password` field.
func b64(in []byte) string {
	return base64.StdEncoding.EncodeToString(in)
}

// generateTestServerCertPEM generates a self-signed ECDSA server certificate/key pair for a real
// (non-pinned) TLS handshake in tests - unlike generateTestCertPEM, it needs a ServerAuth EKU and
// a 127.0.0.1 IP SAN to validate under normal (non-InsecureSkipVerify) verification.
func generateTestServerCertPEM(t *testing.T) (certPEM, keyPEM []byte) {
	t.Helper()

	priv, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)

	serial, err := rand.Int(rand.Reader, big.NewInt(1<<62))
	require.NoError(t, err)

	template := &x509.Certificate{
		SerialNumber:          serial,
		Subject:               pkix.Name{CommonName: "test-server"},
		NotBefore:             time.Now().Add(-time.Hour),
		NotAfter:              time.Now().Add(time.Hour),
		KeyUsage:              x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
		IPAddresses:           []net.IP{net.ParseIP("127.0.0.1")},
	}

	der, err := x509.CreateCertificate(rand.Reader, template, template, &priv.PublicKey, priv)
	require.NoError(t, err)

	certPEM = pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der})

	keyDER, err := x509.MarshalECPrivateKey(priv)
	require.NoError(t, err)
	keyPEM = pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: keyDER})

	return certPEM, keyPEM
}

// unwrapMTLSTransport recovers the *http.Transport carrying the TLS client cert, handling both
// the bare case and the case where buildComputeMTLSHTTPClient wrapped it in
// stripCloudAPIPrefixRoundTripper (only when StripCloudAPIPrefix is set).
func unwrapMTLSTransport(t *testing.T, rt http.RoundTripper) *http.Transport {
	t.Helper()
	if wrapper, ok := rt.(*stripCloudAPIPrefixRoundTripper); ok {
		rt = wrapper.next
	}
	tr, ok := rt.(*http.Transport)
	require.True(t, ok, "expected an *http.Transport (optionally wrapped in stripCloudAPIPrefixRoundTripper) carrying the TLS client cert")
	return tr
}

func TestNewIonosClient(t *testing.T) {

	type args struct {
		data []byte
	}

	clientCertPEM, clientKeyPEM, clientCert := generateTestCertPEM(t, "provider-client", false)
	caCertPEM, _, caCert := generateTestCertPEM(t, "test-ca", true)
	_, unrelatedKeyPEM, _ := generateTestCertPEM(t, "unrelated-key", false)

	tests := []struct {
		name              string
		args              args
		env               map[string]string
		wantComputeConfig *ionos.Configuration
		wantDbaasConfig   *shared.Configuration
		wantErr           bool
		// checkComputeHTTPClient, when set, replaces the plain assert.Equal comparison of
		// ComputeClient's HTTPClient with targeted TLS assertions (a real tls.Config with key/pool
		// material can't be reliably deep-equal compared).
		checkComputeHTTPClient func(t *testing.T, hc *http.Client)
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
		{
			name: "mtls client cert and key",
			args: args{
				data: []byte(fmt.Sprintf(
					`{"user": "username","password": "cGFzc3dvcmQ=", "client_cert": "%s", "client_key": "%s"}`,
					b64(clientCertPEM), b64(clientKeyPEM),
				)),
			},
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
			checkComputeHTTPClient: func(t *testing.T, hc *http.Client) {
				require.NotNil(t, hc)
				_, wrapped := hc.Transport.(*stripCloudAPIPrefixRoundTripper)
				assert.False(t, wrapped, "strip_cloudapi_prefix was not set, Transport must not be wrapped")
				tr := unwrapMTLSTransport(t, hc.Transport)
				require.NotNil(t, tr.TLSClientConfig)
				require.Len(t, tr.TLSClientConfig.Certificates, 1)
				assert.Equal(t, clientCert.Raw, tr.TLSClientConfig.Certificates[0].Certificate[0])
				assert.Nil(t, tr.TLSClientConfig.RootCAs, "no ca_cert was supplied, RootCAs must stay nil")
			},
		},
		{
			name: "mtls client cert, key and ca",
			args: args{
				data: []byte(fmt.Sprintf(
					`{"user": "username","password": "cGFzc3dvcmQ=", "client_cert": "%s", "client_key": "%s", "ca_cert": "%s"}`,
					b64(clientCertPEM), b64(clientKeyPEM), b64(caCertPEM),
				)),
			},
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
			checkComputeHTTPClient: func(t *testing.T, hc *http.Client) {
				require.NotNil(t, hc)
				tr := unwrapMTLSTransport(t, hc.Transport)
				require.NotNil(t, tr.TLSClientConfig)
				require.Len(t, tr.TLSClientConfig.Certificates, 1)
				assert.Equal(t, clientCert.Raw, tr.TLSClientConfig.Certificates[0].Certificate[0])
				require.NotNil(t, tr.TLSClientConfig.RootCAs, "ca_cert was supplied, RootCAs must be set")

				wantPool, err := x509.SystemCertPool()
				if err != nil || wantPool == nil {
					wantPool = x509.NewCertPool()
				}
				wantPool.AddCert(caCert)
				assert.True(t, tr.TLSClientConfig.RootCAs.Equal(wantPool), "RootCAs must be the system pool plus the supplied CA")
			},
		},
		{
			name: "mtls client cert without key",
			args: args{
				data: []byte(fmt.Sprintf(
					`{"user": "username","password": "cGFzc3dvcmQ=", "client_cert": "%s"}`,
					b64(clientCertPEM),
				)),
			},
			wantComputeConfig: nil,
			wantDbaasConfig:   nil,
			wantErr:           true,
		},
		{
			name: "mtls client key without cert",
			args: args{
				data: []byte(fmt.Sprintf(
					`{"user": "username","password": "cGFzc3dvcmQ=", "client_key": "%s"}`,
					b64(clientKeyPEM),
				)),
			},
			wantComputeConfig: nil,
			wantDbaasConfig:   nil,
			wantErr:           true,
		},
		{
			name: "mtls ca cert without client cert/key",
			args: args{
				data: []byte(fmt.Sprintf(
					`{"user": "username","password": "cGFzc3dvcmQ=", "ca_cert": "%s"}`,
					b64(caCertPEM),
				)),
			},
			wantComputeConfig: nil,
			wantDbaasConfig:   nil,
			wantErr:           true,
		},
		{
			name: "mtls mismatched client cert and key",
			args: args{
				data: []byte(fmt.Sprintf(
					`{"user": "username","password": "cGFzc3dvcmQ=", "client_cert": "%s", "client_key": "%s"}`,
					b64(clientCertPEM), b64(unrelatedKeyPEM),
				)),
			},
			wantComputeConfig: nil,
			wantDbaasConfig:   nil,
			wantErr:           true,
		},
		{
			name: "mtls malformed base64 client cert",
			args: args{
				data: []byte(fmt.Sprintf(
					`{"user": "username","password": "cGFzc3dvcmQ=", "client_cert": "not-valid-base64!", "client_key": "%s"}`,
					b64(clientKeyPEM),
				)),
			},
			wantComputeConfig: nil,
			wantDbaasConfig:   nil,
			wantErr:           true,
		},
		{
			name: "mtls invalid PEM client cert",
			args: args{
				data: []byte(fmt.Sprintf(
					`{"user": "username","password": "cGFzc3dvcmQ=", "client_cert": "%s", "client_key": "%s"}`,
					b64([]byte("not a real certificate")), b64(clientKeyPEM),
				)),
			},
			wantComputeConfig: nil,
			wantDbaasConfig:   nil,
			wantErr:           true,
		},
		{
			name: "mtls malformed base64 ca cert",
			args: args{
				data: []byte(fmt.Sprintf(
					`{"user": "username","password": "cGFzc3dvcmQ=", "client_cert": "%s", "client_key": "%s", "ca_cert": "not-valid-base64!"}`,
					b64(clientCertPEM), b64(clientKeyPEM),
				)),
			},
			wantComputeConfig: nil,
			wantDbaasConfig:   nil,
			wantErr:           true,
		},
		{
			name: "mtls invalid PEM ca cert",
			args: args{
				data: []byte(fmt.Sprintf(
					`{"user": "username","password": "cGFzc3dvcmQ=", "client_cert": "%s", "client_key": "%s", "ca_cert": "%s"}`,
					b64(clientCertPEM), b64(clientKeyPEM), b64([]byte("not a real ca certificate")),
				)),
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

				if tt.checkComputeHTTPClient != nil {
					tt.checkComputeHTTPClient(t, ccfg.HTTPClient)
					// TLS bits were just checked above; swap in the baseline HTTPClient so the
					// rest of the Configuration struct can still be compared with assert.Equal.
					ccfg.HTTPClient = tt.wantComputeConfig.HTTPClient
				}

				assert.Equal(t, tt.wantComputeConfig, ccfg)
				assert.Equal(t, tt.wantDbaasConfig, dcfg)
			} else {
				assert.Nil(t, got)
			}
		})
	}
}

// TestNewIonosClient_MTLSWithCertPinning verifies a client cert and IONOS_PINNED_CERT configured
// together both take effect: the client cert is still presented and the pinned fingerprint is
// still enforced (see reapplyMTLSAfterPinning).
func TestNewIonosClient_MTLSWithCertPinning(t *testing.T) {
	clientCertPEM, clientKeyPEM, _ := generateTestCertPEM(t, "provider-client", false)
	serverCertPEM, serverKeyPEM, _ := generateTestCertPEM(t, "test-server", false)
	otherCertPEM, _, _ := generateTestCertPEM(t, "other-server", false)

	serverKeyPair, err := tls.X509KeyPair(serverCertPEM, serverKeyPEM)
	require.NoError(t, err)

	fingerprintOf := func(certPEM []byte) string {
		block, _ := pem.Decode(certPEM)
		require.NotNil(t, block)
		sum := sha256.Sum256(block.Bytes)
		return hex.EncodeToString(sum[:])
	}

	var sawClientCert bool
	srv := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		sawClientCert = r.TLS != nil && len(r.TLS.PeerCertificates) > 0
		w.WriteHeader(http.StatusOK)
	}))
	srv.TLS = &tls.Config{
		Certificates: []tls.Certificate{serverKeyPair},
		ClientAuth:   tls.RequireAnyClientCert,
	}
	srv.StartTLS()
	defer srv.Close()

	creds := []byte(fmt.Sprintf(
		`{"user": "username","password": "cGFzc3dvcmQ=", "client_cert": "%s", "client_key": "%s"}`,
		b64(clientCertPEM), b64(clientKeyPEM),
	))

	t.Run("matching pinned fingerprint: client cert still presented", func(t *testing.T) {
		sawClientCert = false
		require.NoError(t, os.Setenv(ionos.IonosPinnedCertEnvVar, fingerprintOf(serverCertPEM)))
		loadEnv()
		defer func() {
			require.NoError(t, os.Unsetenv(ionos.IonosPinnedCertEnvVar))
			loadEnv()
		}()

		svc, err := NewIonosClients(creds)
		require.NoError(t, err)
		hc := svc.ComputeClient.GetConfig().HTTPClient
		require.NotNil(t, hc)

		req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, srv.URL, nil)
		require.NoError(t, err)
		resp, err := hc.Do(req)
		require.NoError(t, err, "handshake must succeed: fingerprint matches and client cert is presented")
		defer resp.Body.Close()
		assert.Equal(t, http.StatusOK, resp.StatusCode)
		assert.True(t, sawClientCert, "server must have received the client certificate")
	})

	t.Run("mismatched pinned fingerprint: connection rejected", func(t *testing.T) {
		require.NoError(t, os.Setenv(ionos.IonosPinnedCertEnvVar, fingerprintOf(otherCertPEM)))
		loadEnv()
		defer func() {
			require.NoError(t, os.Unsetenv(ionos.IonosPinnedCertEnvVar))
			loadEnv()
		}()

		svc, err := NewIonosClients(creds)
		require.NoError(t, err)
		hc := svc.ComputeClient.GetConfig().HTTPClient
		require.NotNil(t, hc)

		req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, srv.URL, nil)
		require.NoError(t, err)
		resp, err := hc.Do(req)
		if resp != nil {
			defer resp.Body.Close()
		}
		assert.Error(t, err, "handshake must fail when the pinned fingerprint does not match the server certificate")
	})
}

func TestStripCloudAPIPrefixRoundTripper(t *testing.T) {
	cases := []struct {
		name         string
		requestPath  string
		expectedPath string
	}{
		{"strips /cloudapi/v6 prefix", "/cloudapi/v6/datacenters", "/v6/datacenters"},
		{"strips bare /cloudapi", "/cloudapi", ""},
		{"leaves unrelated path alone", "/v6/datacenters", "/v6/datacenters"},
		{"does not strip a merely-prefixed segment", "/cloudapifoo/v6", "/cloudapifoo/v6"},
		{"root path untouched", "/", "/"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var gotPath string
			next := roundTripFunc(func(req *http.Request) (*http.Response, error) {
				gotPath = req.URL.Path
				return &http.Response{StatusCode: http.StatusOK, Body: http.NoBody}, nil
			})
			rt := &stripCloudAPIPrefixRoundTripper{next: next}

			req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, "https://example.invalid"+tc.requestPath, nil)
			require.NoError(t, err)
			resp, err := rt.RoundTrip(req)
			require.NoError(t, err)
			defer resp.Body.Close()
			assert.Equal(t, tc.expectedPath, gotPath)
			assert.Equal(t, tc.requestPath, req.URL.Path, "original request must not be mutated in place")
		})
	}
}

// roundTripFunc lets a plain function satisfy http.RoundTripper, for tests that only care about
// observing/stubbing the final request rather than exercising a real network transport.
type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) { return f(req) }

func TestNewIonosClient_MTLSStripsCloudAPIPrefix(t *testing.T) {
	clientCertPEM, clientKeyPEM, _ := generateTestCertPEM(t, "provider-client", false)
	// This test doesn't use cert pinning, so (unlike the other mTLS tests here) the server cert
	// is actually verified - it needs a 127.0.0.1 IP SAN matching httptest's address.
	serverCertPEM, serverKeyPEM := generateTestServerCertPEM(t)

	serverKeyPair, err := tls.X509KeyPair(serverCertPEM, serverKeyPEM)
	require.NoError(t, err)

	var gotPath string
	srv := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		w.WriteHeader(http.StatusOK)
	}))
	srv.TLS = &tls.Config{
		Certificates: []tls.Certificate{serverKeyPair},
		ClientAuth:   tls.RequireAnyClientCert,
	}
	srv.StartTLS()
	defer srv.Close()

	creds := []byte(fmt.Sprintf(
		`{"user": "username","password": "cGFzc3dvcmQ=", "client_cert": "%s", "client_key": "%s", "ca_cert": "%s", "strip_cloudapi_prefix": true}`,
		b64(clientCertPEM), b64(clientKeyPEM), b64(serverCertPEM),
	))

	svc, err := NewIonosClients(creds)
	require.NoError(t, err)
	hc := svc.ComputeClient.GetConfig().HTTPClient
	require.NotNil(t, hc)

	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, srv.URL+"/cloudapi/v6/datacenters", nil)
	require.NoError(t, err)
	resp, err := hc.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Equal(t, "/v6/datacenters", gotPath, "the /cloudapi prefix must be stripped before the request reaches the internal mTLS endpoint")
}

// TestNewIonosClient_MTLSDoesNotStripCloudAPIPrefixByDefault verifies that stripping is strictly
// opt-in via strip_cloudapi_prefix: a generic mTLS-enforcing endpoint that retains the standard
// /cloudapi/v6 layout (e.g. the default api.ionos.com surface itself) must not have its paths
// rewritten just because a client certificate is configured.
func TestNewIonosClient_MTLSDoesNotStripCloudAPIPrefixByDefault(t *testing.T) {
	clientCertPEM, clientKeyPEM, _ := generateTestCertPEM(t, "provider-client", false)
	serverCertPEM, serverKeyPEM := generateTestServerCertPEM(t)

	serverKeyPair, err := tls.X509KeyPair(serverCertPEM, serverKeyPEM)
	require.NoError(t, err)

	var gotPath string
	srv := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		w.WriteHeader(http.StatusOK)
	}))
	srv.TLS = &tls.Config{
		Certificates: []tls.Certificate{serverKeyPair},
		ClientAuth:   tls.RequireAnyClientCert,
	}
	srv.StartTLS()
	defer srv.Close()

	creds := []byte(fmt.Sprintf(
		`{"user": "username","password": "cGFzc3dvcmQ=", "client_cert": "%s", "client_key": "%s", "ca_cert": "%s"}`,
		b64(clientCertPEM), b64(clientKeyPEM), b64(serverCertPEM),
	))

	svc, err := NewIonosClients(creds)
	require.NoError(t, err)
	hc := svc.ComputeClient.GetConfig().HTTPClient
	require.NotNil(t, hc)
	_, wrapped := hc.Transport.(*stripCloudAPIPrefixRoundTripper)
	assert.False(t, wrapped, "strip_cloudapi_prefix was not set, Transport must not be wrapped")

	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, srv.URL+"/cloudapi/v6/datacenters", nil)
	require.NoError(t, err)
	resp, err := hc.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Equal(t, "/cloudapi/v6/datacenters", gotPath, "without strip_cloudapi_prefix, the path must reach the endpoint unmodified")
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
