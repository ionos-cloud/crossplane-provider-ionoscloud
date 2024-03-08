package ccpatch

import (
	"encoding/base64"
	"errors"
	"strings"
	"unicode"

	"gopkg.in/yaml.v3"
)

var (
	// ErrDecodeFailed is returned when base64 decoding fails
	ErrDecodeFailed = errors.New("failed to decode base64")
	// ErrNoCloudConfig is returned when no cloud-config header is found
	ErrNoCloudConfig = errors.New("no cloud-config header found")
	// ErrMalformedData is returned when the cloud-init data is malformed
	ErrMalformedData = errors.New("malformed cloud-init data")
)

// CloudInitPatcher is a helper to patch cloud-init userdata
type CloudInitPatcher struct {
	raw     string
	decoded string
	data    map[string]interface{}
}

// NewCloudInitPatcher returns a new CloudInitPatcher instance
// from a base64 encoded string
func NewCloudInitPatcher(raw string) (*CloudInitPatcher, error) {
	byt, err := base64.StdEncoding.DecodeString(raw)
	if err != nil {
		return nil, ErrDecodeFailed
	}

	if len(byt) == 0 {
		return &CloudInitPatcher{raw: raw, decoded: "", data: make(map[string]interface{})}, nil
	}

	if !IsCloudConfig(string(byt)) {
		return nil, ErrNoCloudConfig
	}

	data := make(map[string]interface{})
	if err := yaml.Unmarshal(byt, &data); err != nil {
		return nil, ErrMalformedData
	}

	return &CloudInitPatcher{raw: raw, decoded: string(byt), data: data}, nil
}

// Patch adds or modifies a key-value pair in the cloud-init data
func (c *CloudInitPatcher) Patch(key string, value any) *CloudInitPatcher {
	c.data[key] = value
	return c
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
	byt = append([]byte("#cloud-config\n"), byt...)

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

	return (header == "#cloud-config")
}
