package ccpatch

import (
	"encoding/base64"
	"errors"
	"fmt"
	"strings"
	"unicode"

	"gopkg.in/yaml.v3"

	"github.com/ionos-cloud/crossplane-provider-ionoscloud/pkg/ccpatch/substitution"
)

var (
	// ErrDecodeFailed is returned when base64 decoding fails
	ErrDecodeFailed = errors.New("failed to decode base64")
	// ErrNoCloudConfig is returned when no cloud-config header is found
	ErrNoCloudConfig = errors.New("no cloud-config header found")
	// ErrMalformedData is returned when the cloud-init data is malformed
	ErrMalformedData = errors.New("malformed cloud-init data")
)

const cloudConfigHeader = "#cloud-config"

// CloudInitPatcher is a helper to patch cloud-init userdata
type CloudInitPatcher struct {
	raw           string
	decoded       string
	data          map[string]interface{}
	globalState   *substitution.GlobalState
	identifier    substitution.Identifier
	substitutions []substitution.Substitution
}

// NewCloudInitPatcherWithSubstitutions returns a new CloudInitPatcher instance
// with a list of substitutions
func NewCloudInitPatcherWithSubstitutions(raw string, identifier substitution.Identifier, substitutions []substitution.Substitution, globalState *substitution.GlobalState) (*CloudInitPatcher, error) {
	if globalState == nil {
		globalState = &substitution.GlobalState{}
	}

	patcher, err := newCloudInitPatcher(raw)
	if err != nil {
		return nil, err
	}

	patcher.identifier = identifier
	patcher.substitutions = substitutions
	patcher.globalState = globalState

	if err := buildState(
		patcher.identifier,
		patcher.substitutions,
		patcher.globalState,
	); err != nil {
		return nil, err
	}

	patcher.decoded, err = substitution.ReplaceByState(
		patcher.identifier,
		patcher.globalState,
		patcher.decoded,
	)
	if err != nil {
		return nil, err
	}

	// write patched decoded back to data
	if err := yaml.Unmarshal([]byte(patcher.decoded), &patcher.data); err != nil {
		return nil, fmt.Errorf("%w (%w)", ErrMalformedData, err)
	}

	return patcher, nil
}

// NewCloudInitPatcher returns a new CloudInitPatcher instance
// from a base64 encoded string
func NewCloudInitPatcher(raw string) (*CloudInitPatcher, error) {
	return newCloudInitPatcher(raw)
}

func newCloudInitPatcher(raw string) (*CloudInitPatcher, error) {
	byt, err := base64.StdEncoding.DecodeString(raw)
	if err != nil {
		return nil, fmt.Errorf("%w (%w)", ErrDecodeFailed, err)
	}

	if len(byt) == 0 {
		return &CloudInitPatcher{raw: raw, decoded: "", data: make(map[string]interface{})}, nil
	}

	if !IsCloudConfig(string(byt)) {
		return nil, ErrNoCloudConfig
	}

	data := make(map[string]interface{})
	if err := yaml.Unmarshal(byt, &data); err != nil {
		return nil, fmt.Errorf("%w (%w)", ErrMalformedData, err)
	}

	return &CloudInitPatcher{raw: raw, decoded: string(byt), data: data, substitutions: nil}, nil
}

// Patch adds or modifies a key-value pair in the cloud-init data
func (c *CloudInitPatcher) Patch(key string, value any) *CloudInitPatcher {
	c.data[key] = value
	return c
}

// SetEnv sets an environment variable in the cloud-init data
// within the "environment" key
func (c *CloudInitPatcher) SetEnv(key string, value string) *CloudInitPatcher {
	if c.data["environment"] == nil {
		c.data["environment"] = make(map[string]interface{})
	}

	c.data["environment"].(map[string]interface{})[key] = value

	return c
}

// GetEnv returns the value of an environment variable in the cloud-init data
func (c *CloudInitPatcher) GetEnv(key string) string {
	if c.data["environment"] == nil {
		return ""
	}

	return c.data["environment"].(map[string]interface{})[key].(string)
}

// Get returns the value of a key in the cloud-init data
func (c *CloudInitPatcher) Get(key string) any {
	return c.data[key]
}

// String returns the cloud-init data as a string
func (c *CloudInitPatcher) String() string {
	byt, err := yaml.Marshal(c.data)
	if err != nil {
		return ""
	}

	// add #cloud-config header
	byt = append([]byte(cloudConfigHeader+"\n"), byt...)

	return string(byt)
}

// Encode returns the base64 encoded cloud-init data
func (c *CloudInitPatcher) Encode() string {
	return base64.StdEncoding.EncodeToString([]byte(c.String()))
}

// IsCloudConfig checks if the given userdata is a cloud-config
func IsCloudConfig(userdata string) bool {
	userdata = strings.TrimLeftFunc(userdata, unicode.IsSpace)

	header := strings.SplitN(userdata, "\n", 2)[0]

	// Trim trailing whitespaces
	header = strings.TrimRightFunc(header, unicode.IsSpace)

	return header == cloudConfigHeader
}
