package clients

import (
	"bytes"
	"context"
	"crypto/sha256"
	"crypto/tls"
	"crypto/x509"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"os"
	"strings"

	xpv1 "github.com/crossplane/crossplane-runtime/apis/common/v1"
	"github.com/crossplane/crossplane-runtime/pkg/resource"
	sdkdbaas "github.com/ionos-cloud/sdk-go-bundle/products/dbaas/psql/v2"
	"github.com/ionos-cloud/sdk-go-bundle/shared"
	mongo "github.com/ionos-cloud/sdk-go-dbaas-mongo"
	sdkgo "github.com/ionos-cloud/sdk-go/v6"
	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/types"
	kubeclient "sigs.k8s.io/controller-runtime/pkg/client"

	apisv1alpha1 "github.com/ionos-cloud/crossplane-provider-ionoscloud/apis/v1alpha1"
	"github.com/ionos-cloud/crossplane-provider-ionoscloud/internal/clients/compute"
	"github.com/ionos-cloud/crossplane-provider-ionoscloud/internal/clients/k8s"
	"github.com/ionos-cloud/crossplane-provider-ionoscloud/internal/version"
)

const (
	// UserAgent is the user agent addition that identifies the Crossplane IONOS Cloud Clients
	UserAgent = "crossplane-provider-ionoscloud"
)

const (
	errTrackPCUsage = "cannot track ProviderConfig usage"
	errGetPC        = "cannot get ProviderConfig"
	errGetCreds     = "cannot get credentials"
	errNewClient    = "cannot create new Service"
)

// allow to set a default IONOS APIs for all clients via env variable.
var ionosAPIEndpoint string

// loadEnv is an indirection from the init function. The init function itself is not callable, but the loadEnv function.
// This allows us to reset the env before and after each test.
func loadEnv() {
	ionosAPIEndpoint = os.Getenv(sdkgo.IonosApiUrlEnvVar)
}

func init() {
	loadEnv()
}

// IonosServices contains ionos clients
type IonosServices struct {
	DBaaSPostgresClient *sdkdbaas.APIClient
	DBaaSMongoClient    *mongo.APIClient
	ComputeClient       *sdkgo.APIClient
}

// credentials specify how to authenticate with the IONOS Cloud API
type credentials struct {
	// Username to use
	User string `json:"user"`

	// Password to use
	// The password must be base64 encoded to prevent parsing anc escaping issues with special characters.
	Password string `json:"password"`

	// Token can be used instead of username and password
	Token string `json:"token"`

	// HostURL is the baseURL of the IONOS Cloud API.
	// It can be used for overwriting the default endpoint. Optional.
	HostURL string `json:"host_url"`

	// ClientCertificate is a PEM encoded client (leaf) certificate presented for mutual TLS
	// (mTLS) authentication against the compute/cloud API endpoint, e.g. an internal endpoint
	// that enforces client certificate verification. Optional; requires ClientKey to also be
	// set. Like Password, it must be base64 encoded to survive JSON string escaping.
	ClientCertificate string `json:"client_cert"`

	// ClientKey is the PEM encoded private key matching ClientCertificate. Optional; requires
	// ClientCertificate to also be set. Must be base64 encoded, like Password.
	ClientKey string `json:"client_key"`

	// CACertificate is an optional PEM encoded CA certificate to trust in addition to the system
	// root pool, base64 encoded like Password. Only valid together with
	// ClientCertificate/ClientKey - rejected as an error if set alone, since it would otherwise
	// have no effect.
	CACertificate string `json:"ca_cert"`

	// StripCloudAPIPrefix, when true, strips a leading "/cloudapi" segment from outgoing
	// compute-API requests, for endpoints that serve the API at /v6/... instead of
	// /cloudapi/v6/.... Opt-in only: ignored unless ClientCertificate/ClientKey are set, since a
	// generic endpoint using the standard /cloudapi/v6 layout must not have its paths rewritten.
	StripCloudAPIPrefix bool `json:"strip_cloudapi_prefix"`
}

// buildComputeMTLSHTTPClient builds an *http.Client presenting a client certificate for mutual
// TLS, plus the *tls.Config it used (reapplyMTLSAfterPinning needs it directly, since the
// Transport may be wrapped and no longer type-assertable). Returns (nil, nil, nil) when no mTLS
// credentials are configured, leaving the SDK's default HTTPClient behavior untouched.
func buildComputeMTLSHTTPClient(creds credentials) (*http.Client, *tls.Config, error) {
	hasCert := creds.ClientCertificate != ""
	hasKey := creds.ClientKey != ""
	hasCA := creds.CACertificate != ""

	if !hasCert && !hasKey && !hasCA {
		return nil, nil, nil
	}
	if hasCert != hasKey {
		return nil, nil, fmt.Errorf("mtls setup: client_cert and client_key must both be set together")
	}

	// ca_cert only adjusts the trust pool used when presenting a client cert, so it has no effect
	// without client_cert/client_key - reject it explicitly rather than silently ignoring it.
	if hasCA && !hasCert {
		return nil, nil, fmt.Errorf("mtls setup: ca_cert has no effect without client_cert/client_key also being set")
	}

	var rootCAs *x509.CertPool
	if hasCA {
		caPEM, err := base64.StdEncoding.DecodeString(creds.CACertificate)
		if err != nil {
			return nil, nil, fmt.Errorf("mtls setup: failed to decode ca_cert: %w", err)
		}
		pool, err := x509.SystemCertPool()
		if err != nil || pool == nil {
			pool = x509.NewCertPool()
		}
		if !pool.AppendCertsFromPEM(caPEM) {
			return nil, nil, fmt.Errorf("mtls setup: failed to parse ca_cert as PEM")
		}
		rootCAs = pool
	}

	certPEM, err := base64.StdEncoding.DecodeString(creds.ClientCertificate)
	if err != nil {
		return nil, nil, fmt.Errorf("mtls setup: failed to decode client_cert: %w", err)
	}
	keyPEM, err := base64.StdEncoding.DecodeString(creds.ClientKey)
	if err != nil {
		return nil, nil, fmt.Errorf("mtls setup: failed to decode client_key: %w", err)
	}

	cert, err := tls.X509KeyPair(certPEM, keyPEM)
	if err != nil {
		return nil, nil, fmt.Errorf("mtls setup: failed to load client certificate/key pair: %w", err)
	}

	tlsConfig := &tls.Config{
		Certificates: []tls.Certificate{cert},
	}
	if rootCAs != nil {
		tlsConfig.RootCAs = rootCAs
	}

	// Clone http.DefaultTransport instead of a zero-valued one, to keep its defaults
	// (ProxyFromEnvironment, timeouts, HTTP/2) - a bare &http.Transport{} would silently drop
	// all of those, e.g. breaking proxy support.
	transport := http.DefaultTransport.(*http.Transport).Clone()
	transport.TLSClientConfig = tlsConfig

	var rt http.RoundTripper = transport
	if creds.StripCloudAPIPrefix {
		rt = &stripCloudAPIPrefixRoundTripper{next: transport}
	}

	return &http.Client{
		Transport: rt,
	}, tlsConfig, nil
}

// cloudAPIPathPrefix is the path segment sdk-go/v6 always appends to a configured host_url (see
// getServerUrl in sdk-go/v6@v6.3.4/configuration.go), matching the public
// api.ionos.com/cloudapi/v6 surface.
const cloudAPIPathPrefix = "/cloudapi"

// stripCloudAPIPrefixRoundTripper strips a leading "/cloudapi" segment from outgoing requests.
// Some internal endpoints serve the API at /v6/... instead of the public /cloudapi/v6/... layout
// the SDK always targets. Wired in only when credentials.StripCloudAPIPrefix is set - never
// automatically - since a standard-layout endpoint must not have its paths rewritten.
type stripCloudAPIPrefixRoundTripper struct {
	next http.RoundTripper
}

func (t *stripCloudAPIPrefixRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	p := req.URL.Path
	if p == cloudAPIPathPrefix || strings.HasPrefix(p, cloudAPIPathPrefix+"/") {
		req = req.Clone(req.Context())
		req.URL.Path = strings.TrimPrefix(p, cloudAPIPathPrefix)
		if req.URL.RawPath != "" {
			req.URL.RawPath = strings.TrimPrefix(req.URL.RawPath, cloudAPIPathPrefix)
		}
	}
	return t.next.RoundTrip(req)
}

// reapplyMTLSAfterPinning restores the mTLS Transport after sdkgo.NewAPIClient overwrites it when
// IONOS_PINNED_CERT is set, rebuilding a Transport that both presents the client certificate and
// enforces the pinned fingerprint. mtlsTLSConfig must be captured by the caller before calling
// sdkgo.NewAPIClient - cfg.HTTPClient's original Transport is gone by the time this runs.
func reapplyMTLSAfterPinning(cfg *sdkgo.Configuration, mtlsTLSConfig *tls.Config, stripCloudAPIPrefix bool) {
	pkFingerprint := os.Getenv(sdkgo.IonosPinnedCertEnvVar)
	if pkFingerprint == "" {
		// sdkgo.NewAPIClient only touches cfg.HTTPClient.Transport when the pinning env var is
		// set, so our Transport is still intact - nothing to repair.
		return
	}

	tlsConfig := mtlsTLSConfig.Clone()
	if cfg.HTTPClient == nil {
		cfg.HTTPClient = &http.Client{}
	}
	// Same reasoning as buildComputeMTLSHTTPClient: clone http.DefaultTransport instead of
	// starting from a zero-valued *http.Transport, to keep its non-TLS defaults intact.
	transport := http.DefaultTransport.(*http.Transport).Clone()
	transport.TLSClientConfig = tlsConfig
	transport.DialTLSContext = pinnedCertDialTLSContext(pkFingerprint, tlsConfig)

	var rt http.RoundTripper = transport
	if stripCloudAPIPrefix {
		rt = &stripCloudAPIPrefixRoundTripper{next: transport}
	}
	cfg.HTTPClient.Transport = rt
}

// pinnedCertDialTLSContext mirrors sdk-go's own pinning dialer, but dials using the given
// tls.Config so any client certificate/RootCAs it carries are still presented, instead of a
// hardcoded empty config. InsecureSkipVerify only disables the *server* cert check, in favor of
// the manual fingerprint check below - the client cert is still presented regardless.
func pinnedCertDialTLSContext(fingerprint string, base *tls.Config) func(ctx context.Context, network, addr string) (net.Conn, error) {
	// Fingerprints can be supplied with ':' or ' ' separators, matching sdk-go/v6's own handling.
	trimmed := []byte(fingerprint)
	trimmed = bytes.ReplaceAll(trimmed, []byte(":"), nil)
	trimmed = bytes.ReplaceAll(trimmed, []byte(" "), nil)

	tlsConfig := base.Clone()
	tlsConfig.InsecureSkipVerify = true

	return func(ctx context.Context, network, addr string) (net.Conn, error) {
		conn, err := (&tls.Dialer{Config: tlsConfig}).DialContext(ctx, network, addr)
		if err != nil {
			return nil, err
		}
		tlsConn, ok := conn.(*tls.Conn)
		if !ok {
			_ = conn.Close()
			return nil, fmt.Errorf("pinned cert dial: unexpected connection type %T", conn)
		}
		if err := verifyPinnedCertFingerprint(trimmed, tlsConn.ConnectionState().PeerCertificates); err != nil {
			_ = conn.Close()
			return nil, err
		}
		return conn, nil
	}
}

// verifyPinnedCertFingerprint mirrors sdk-go/v6's unexported verifyPinnedCert: it accepts the
// connection if any non-CA peer certificate's SHA-256 fingerprint matches.
func verifyPinnedCertFingerprint(fingerprint []byte, peerCerts []*x509.Certificate) error {
	for _, cert := range peerCerts {
		sum := sha256.Sum256(cert.Raw)
		hexSum := make([]byte, hex.EncodedLen(len(sum)))
		hex.Encode(hexSum, sum[:])
		if !cert.IsCA && bytes.EqualFold(hexSum, fingerprint) {
			return nil
		}
	}
	return fmt.Errorf("remote server presented a certificate which does not match the provided fingerprint")
}

// NewIonosClients creates a IonosService from the given data. The data must be a json struct with the fields `User`,
// `Password`, `Token`. Both fields must be a string value. The password string must be base64 encoded.
func NewIonosClients(data []byte) (*IonosServices, error) {
	creds := credentials{}
	if err := json.Unmarshal(data, &creds); err != nil {
		return nil, fmt.Errorf("failed to decode credentials: %w", err)
	}
	decodedPW := []byte("")
	var err error
	if creds.Password != "" {
		decodedPW, err = base64.StdEncoding.DecodeString(creds.Password)
		if err != nil {
			return nil, fmt.Errorf("failed to decode password: %w", err)
		}
	}

	apiHostURL := creds.HostURL
	if apiHostURL == "" && ionosAPIEndpoint != "" {
		apiHostURL = ionosAPIEndpoint
	}

	// Optional mTLS client certificate; only ComputeClient uses this. computeMTLSTLSConfig is
	// captured directly from the builder, since computeHTTPClient.Transport may be wrapped and
	// reapplyMTLSAfterPinning can't recover it after sdkgo.NewAPIClient runs below.
	computeHTTPClient, computeMTLSTLSConfig, err := buildComputeMTLSHTTPClient(creds)
	if err != nil {
		return nil, fmt.Errorf("failed to configure mtls for compute client: %w", err)
	}

	// DBaaS Mongo Client
	dbaasMongoConfig := mongo.NewConfiguration(creds.User, string(decodedPW), creds.Token, apiHostURL)
	dbaasMongoConfig.UserAgent = fmt.Sprintf("%v/%v_%v", UserAgent, version.Version, dbaasMongoConfig.UserAgent)
	dbaasMongoClient := mongo.NewAPIClient(dbaasMongoConfig)
	// DBaaS Postgres Client
	dbaasPostgresConfig := shared.NewConfiguration(creds.User, string(decodedPW), creds.Token, apiHostURL)
	dbaasPostgresClient := sdkdbaas.NewAPIClient(dbaasPostgresConfig)
	dbaasPostgresClient.GetConfig().UserAgent = fmt.Sprintf("%v/sdk_go_bundle_%v_%v", UserAgent, version.Version, sdkdbaas.Version)
	// Compute Engine Client
	computeEngineConfig := sdkgo.NewConfiguration(creds.User, string(decodedPW), creds.Token, apiHostURL)
	computeEngineConfig.UserAgent = fmt.Sprintf("%v/%v_%v", UserAgent, version.Version, computeEngineConfig.UserAgent)
	if computeHTTPClient != nil {
		computeEngineConfig.HTTPClient = computeHTTPClient
	}
	computeEngineClient := sdkgo.NewAPIClient(computeEngineConfig)
	if computeMTLSTLSConfig != nil {
		// sdkgo.NewAPIClient silently discards the mTLS Transport we just configured whenever
		// IONOS_PINNED_CERT is also set - see reapplyMTLSAfterPinning for why and how this is
		// repaired.
		reapplyMTLSAfterPinning(computeEngineClient.GetConfig(), computeMTLSTLSConfig, creds.StripCloudAPIPrefix)
	}

	return &IonosServices{
		DBaaSMongoClient:    dbaasMongoClient,
		DBaaSPostgresClient: dbaasPostgresClient,
		ComputeClient:       computeEngineClient,
	}, nil
}

// ConnectForCRD resolves the referenced ProviderConfig and extracts the connection secret from that ProviderConfig.
// After that an ionos client is setup with those credentials.
func ConnectForCRD(ctx context.Context, mg resource.Managed, client kubeclient.Client, t resource.Tracker) (*IonosServices, error) {
	if err := t.Track(ctx, mg); err != nil {
		return nil, errors.Wrap(err, errTrackPCUsage)
	}

	pc := &apisv1alpha1.ProviderConfig{}
	if err := client.Get(ctx, types.NamespacedName{Name: mg.GetProviderConfigReference().Name}, pc); err != nil {
		return nil, errors.Wrap(err, errGetPC)
	}

	cd := pc.Spec.Credentials
	data, err := resource.CommonCredentialExtractor(ctx, cd.Source, client, cd.CommonCredentialSelectors)
	if err != nil {
		return nil, errors.Wrap(err, errGetCreds)
	}

	svc, err := NewIonosClients(data)
	if err != nil {
		return nil, errors.Wrap(err, errNewClient)
	}
	return svc, nil
}

// CoreResource is an ionos cloud API object with metadata
type CoreResource interface {
	GetMetadataOk() (*sdkgo.DatacenterElementMetadata, bool)
}

// GetCoreResourceState fetches the state of the metadata of the CoreResource
// If either the metadata is nil, or the state is nil, the empty string is returned
func GetCoreResourceState(object CoreResource) string {
	if metadata, metadataOk := object.GetMetadataOk(); metadataOk {
		if state, stateOk := metadata.GetStateOk(); stateOk {
			if state != nil {
				return *state
			}
			return ""
		}
	}
	return ""
}

// DBaaSResource is a dbaas cloud API object with metadata
type DBaaSResource interface {
	GetMetadataOk() (*sdkdbaas.ClusterMetadata, bool)
}

// GetDBaaSPsqlResourceState fetches the state of the metadata of the CoreResource
// If either the metadata is nil, or the state is nil, the empty string is returned
func GetDBaaSPsqlResourceState(object DBaaSResource) sdkdbaas.State {
	if metadata, metadataOk := object.GetMetadataOk(); metadataOk {
		if state, stateOk := metadata.GetStateOk(); stateOk {
			if state != nil {
				return *state
			}
			return ""
		}
	}
	return ""
}

// ResourceWithState is a resource which allow to update the conditions
type ResourceWithState interface {
	SetConditions(c ...xpv1.Condition)
}

// UpdateCondition will update the condition of the given ResourceWithState to the given state. This
// function implements the common mapping of ionos cloud states to crossplane conditions
func UpdateCondition(cr ResourceWithState, state string) {
	switch state {
	case compute.AVAILABLE, compute.ACTIVE:
		cr.SetConditions(xpv1.Available())
	case compute.DESTROYING, k8s.TERMINATED:
		cr.SetConditions(xpv1.Deleting())
	case compute.BUSY, k8s.DEPLOYING, compute.UPDATING:
		cr.SetConditions(xpv1.Creating())
	default:
		cr.SetConditions(xpv1.Unavailable())
	}
}
