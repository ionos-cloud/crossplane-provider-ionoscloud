package clients

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"

	xpv1 "github.com/crossplane/crossplane-runtime/apis/common/v1"
	"github.com/crossplane/crossplane-runtime/pkg/resource"
	sdkdbaas "github.com/ionos-cloud/sdk-go-bundle/products/dbaas/psql/v2"
	"github.com/ionos-cloud/sdk-go-bundle/shared"
	dataplatform "github.com/ionos-cloud/sdk-go-dataplatform"
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
	DataplatformClient  *dataplatform.APIClient
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
	computeEngineClient := sdkgo.NewAPIClient(computeEngineConfig)

	// Dataplatform Engine Client
	dpConfig := dataplatform.NewConfiguration(creds.User, string(decodedPW), creds.Token, apiHostURL)
	dpConfig.UserAgent = fmt.Sprintf("%v/%v_%v", UserAgent, version.Version, dpConfig.UserAgent)
	dpEngineClient := dataplatform.NewAPIClient(dpConfig)

	return &IonosServices{
		DBaaSMongoClient:    dbaasMongoClient,
		DBaaSPostgresClient: dbaasPostgresClient,
		ComputeClient:       computeEngineClient,
		DataplatformClient:  dpEngineClient,
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
