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

	// CACertificate is an optional PEM encoded CA certificate to trust in addition to the
	// system root pool when validating the compute/cloud API's server certificate. Useful when
	// the mTLS endpoint's server certificate is not publicly trusted. Optional. Must be base64
	// encoded, like Password. Only meaningful together with ClientCertificate/ClientKey - it is
	// rejected as an error if set without them, since it would otherwise have no effect. See
	// buildComputeMTLSHTTPClient.
	CACertificate string `json:"ca_cert"`
}

// buildComputeMTLSHTTPClient builds an *http.Client configured to present a client certificate
// for mutual TLS (mTLS) when talking to the compute/cloud API. It returns (nil, nil) whenever no
// MTLS credentials are configured, so callers can leave the SDK's HTTPClient field untouched
// (nil), which preserves today's default behavior (sdkgo.NewAPIClient falls back to
// http.DefaultClient).
func buildComputeMTLSHTTPClient(creds credentials) (*http.Client, error) {
	hasCert := creds.ClientCertificate != ""
	hasKey := creds.ClientKey != ""
	hasCA := creds.CACertificate != ""

	if !hasCert && !hasKey && !hasCA {
		return nil, nil
	}
	if hasCert != hasKey {
		return nil, fmt.Errorf("mtls setup: client_cert and client_key must both be set together")
	}

	// A CA cert is only meaningful in combination with a client cert/key: this feature exists to
	// let the provider present a client certificate, and ca_cert only adjusts the trust pool used
	// while doing so. A CA-only config (no client_cert/client_key) can't have any effect, so rather
	// than silently ignoring it (which would leave an operator's ca_cert setting inert with no
	// indication why) we reject it explicitly - it is almost certainly a misconfiguration.
	if hasCA && !hasCert {
		return nil, fmt.Errorf("mtls setup: ca_cert has no effect without client_cert/client_key also being set")
	}

	var rootCAs *x509.CertPool
	if hasCA {
		caPEM, err := base64.StdEncoding.DecodeString(creds.CACertificate)
		if err != nil {
			return nil, fmt.Errorf("mtls setup: failed to decode ca_cert: %w", err)
		}
		pool, err := x509.SystemCertPool()
		if err != nil || pool == nil {
			pool = x509.NewCertPool()
		}
		if ok := pool.AppendCertsFromPEM(caPEM); !ok {
			return nil, fmt.Errorf("mtls setup: failed to parse ca_cert as PEM")
		}
		rootCAs = pool
	}

	certPEM, err := base64.StdEncoding.DecodeString(creds.ClientCertificate)
	if err != nil {
		return nil, fmt.Errorf("mtls setup: failed to decode client_cert: %w", err)
	}
	keyPEM, err := base64.StdEncoding.DecodeString(creds.ClientKey)
	if err != nil {
		return nil, fmt.Errorf("mtls setup: failed to decode client_key: %w", err)
	}

	cert, err := tls.X509KeyPair(certPEM, keyPEM)
	if err != nil {
		return nil, fmt.Errorf("mtls setup: failed to load client certificate/key pair: %w", err)
	}

	tlsConfig := &tls.Config{
		Certificates: []tls.Certificate{cert},
	}
	if rootCAs != nil {
		tlsConfig.RootCAs = rootCAs
	}

	return &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: tlsConfig,
		},
	}, nil
}

// reapplyMTLSAfterPinning repairs the compute client's Transport after sdkgo.NewAPIClient has run.
//
// sdkgo.NewAPIClient (github.com/ionos-cloud/sdk-go/v6@v6.3.4/client.go) unconditionally does
// `cfg.HTTPClient.Transport = httpTransport` - replacing the whole Transport with a bare
// *http.Transport carrying only a pinning DialTLSContext - whenever the IONOS_PINNED_CERT env var
// is set, regardless of whether cfg.HTTPClient was already customized. That silently discards the
// TLSClientConfig (client certificate, custom RootCAs) buildComputeMTLSHTTPClient configured
// above, so the provider stops presenting the client certificate the moment cert pinning is also
// enabled.
//
// It is not enough to reassign our original Transport back onto the config afterward: the SDK's
// own pinning dialer (sdkgo.AddPinnedCert/addPinnedCertVerification) does its TLS handshake with a
// hardcoded fresh *tls.Config that never carries client certificates, so simply layering the SDK's
// helper on top of our Transport would still drop the client cert. Instead, when both features are
// configured together, this builds a single DialTLSContext that dials with our own tls.Config
// (Certificates and RootCAs intact) and then performs the same manual fingerprint verification the
// SDK's pinning feature does, so mTLS and certificate pinning both take effect at once.
//
// mtlsTLSConfig must be captured by the caller *before* calling sdkgo.NewAPIClient, not read back
// off cfg.HTTPClient afterward: cfg.HTTPClient here is the very same *http.Client pointer the
// caller built and handed to sdkgo.NewAPIClient, and that call mutates its Transport field in
// place, so by the time this function runs the original Transport is already gone from that
// object too.
func reapplyMTLSAfterPinning(cfg *sdkgo.Configuration, mtlsTLSConfig *tls.Config) {
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
	cfg.HTTPClient.Transport = &http.Transport{
		TLSClientConfig: tlsConfig,
		DialTLSContext:  pinnedCertDialTLSContext(pkFingerprint, tlsConfig),
	}
}

// pinnedCertDialTLSContext returns a TLS dialer equivalent to sdkgo's own certificate-pinning
// dialer (addPinnedCertVerification in sdk-go/v6's client.go), except that it dials using the
// given base tls.Config - so any client certificate/RootCAs it carries are still presented/used -
// rather than a hardcoded empty one. Like the SDK's version, normal chain verification is disabled
// in favor of the manual fingerprint check below, since that is the trust mechanism cert pinning
// implements; the client certificate is still presented during the handshake regardless of
// InsecureSkipVerify, which only disables verification of the *server's* certificate.
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

	// Optional mTLS client certificate for the compute/cloud API endpoint. Only ComputeClient
	// uses this - DBaaS Postgres/Mongo live on different domains/products and are out of scope.
	computeHTTPClient, err := buildComputeMTLSHTTPClient(creds)
	if err != nil {
		return nil, fmt.Errorf("failed to configure mtls for compute client: %w", err)
	}
	// Captured now, before sdkgo.NewAPIClient runs: see reapplyMTLSAfterPinning for why this
	// cannot be read back off computeHTTPClient after that call.
	var computeMTLSTLSConfig *tls.Config
	if computeHTTPClient != nil {
		if tr, ok := computeHTTPClient.Transport.(*http.Transport); ok {
			computeMTLSTLSConfig = tr.TLSClientConfig
		}
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
		reapplyMTLSAfterPinning(computeEngineClient.GetConfig(), computeMTLSTLSConfig)
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
