package ccpatch_test

import (
	"encoding/base64"
	"testing"

	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"

	"github.com/ionos-cloud/crossplane-provider-ionoscloud/pkg/ccpatch"
)

var (
	rawUserData = `
#cloud-config
fqdn: myhostname.example.com
bootcmd:
    - echo "hello, world" > /root/hello.txt
`
)

func TestPatchUserdata(t *testing.T) {
	encoded := base64.StdEncoding.EncodeToString([]byte(rawUserData))

	patcher, err := ccpatch.NewCloudInitPatcher(encoded)
	require.NoError(t, err)

	patcher.Patch("hostname", "local.svc.test")
	require.Equal(t, "local.svc.test", patcher.Get("hostname"))

	// Test String()
	require.Contains(t, patcher.String(), "hostname: local.svc.test")

	// Test Encode()
	encoded = patcher.Encode()
	require.NotEmpty(t, encoded)

	// decode and check if the patch was successful
	decoded, err := base64.StdEncoding.DecodeString(encoded)
	require.NoError(t, err)

	data := make(map[string]interface{})
	err = yaml.Unmarshal(decoded, &data)
	require.NoError(t, err)

	require.Equal(t, "local.svc.test", data["hostname"])
}

func TestPatchUserdataEmpty(t *testing.T) {
	patcher, err := ccpatch.NewCloudInitPatcher("")
	require.NoError(t, err)

	patcher.Patch("hostname", "local.svc.test")
	require.Equal(t, "local.svc.test", patcher.Get("hostname"))

	// Test String()
	require.Contains(t, patcher.String(), "hostname: local.svc.test")

	// Test Encode()
	encoded := patcher.Encode()
	require.NotEmpty(t, encoded)

	// decode and check if the patch was successful
	decoded, err := base64.StdEncoding.DecodeString(encoded)
	require.NoError(t, err)

	data := make(map[string]interface{})
	err = yaml.Unmarshal(decoded, &data)
	require.NoError(t, err)

	require.Equal(t, "local.svc.test", data["hostname"])
}

func TestPatchInvalidUserdata(t *testing.T) {
	_, err := ccpatch.NewCloudInitPatcher("invalid")
	require.Error(t, err)
}

func TestPatchInvalidUserdataNoCloudConfig(t *testing.T) {
	_, err := ccpatch.NewCloudInitPatcher("dGVzdA==")
	require.Error(t, err)
}

func TestPatchInvalidUserdataMalformedData(t *testing.T) {
	var invalid = `
#cloud-config
{}	- 
-gg 
`
	_, err := ccpatch.NewCloudInitPatcher(base64.StdEncoding.EncodeToString([]byte(invalid)))
	require.Error(t, err)
}

func TestPatchSetEnv(t *testing.T) {
	encoded := base64.StdEncoding.EncodeToString([]byte(rawUserData))

	patcher, err := ccpatch.NewCloudInitPatcher(encoded)
	require.NoError(t, err)

	patcher.SetEnv("key", "value")
	require.Equal(t, "value", patcher.GetEnv("key"))

	// Test String()
	require.Contains(t, patcher.String(), "key: value")

	// Test Encode()
	encoded = patcher.Encode()
	require.NotEmpty(t, encoded)

	// decode and check if the patch was successful
	decoded, err := base64.StdEncoding.DecodeString(encoded)
	require.NoError(t, err)

	data := make(map[string]interface{})
	err = yaml.Unmarshal(decoded, &data)
	require.NoError(t, err)

	require.Equal(t, "value", data["environment"].(map[string]interface{})["key"])
}
