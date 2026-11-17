package clients

import (
	"net/http"
	"testing"
	"time"

	"github.com/crossplane/crossplane-runtime/pkg/logging"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// The throttle wraps the injector, so a fake 429 or 422 starts a block like a real one, and the
// held back writes never reach the injector.

func TestFaultInjector_ThrottleHoldsWritesAfterFake429(t *testing.T) {
	freshComputeThrottle(t)
	ft := newFaultTest(t, "dc="+faultTestDC+",mode=write-429,"+faultTestUntil+",retryAfter=30")
	hc := withComputeThrottle(ft.hc, throttleTestOptions)

	got := throttleTestDo(t, hc, http.MethodDelete, faultTestDCURL+"/servers/s1")
	assert.Equal(t, http.StatusTooManyRequests, got.status)
	assert.False(t, got.throttled(), "the first 429 comes from the injector")
	require.Len(t, ft.log.answers(), 1)

	for _, method := range []string{http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete} {
		got = throttleTestDo(t, hc, method, faultTestDCURL+"/lans/1")
		assert.Equal(t, http.StatusTooManyRequests, got.status, method)
		assert.True(t, got.throttled(), method)
		assert.Equal(t, "30", got.header.Get(retryAfterHeader), method)
	}
	assert.Len(t, ft.log.answers(), 1, "held back writes do not reach the injector")
	assert.Equal(t, 0, ft.next.calls)
}

func TestFaultInjector_ThrottleHoldsDeleteAfterFake422(t *testing.T) {
	freshComputeThrottle(t)
	ft := newFaultTest(t, "dc="+faultTestDC+",mode=delete-422,"+faultTestUntil)
	hc := withComputeThrottle(ft.hc, throttleTestOptions)
	lanURL := faultTestDCURL + "/lans/1"

	got := throttleTestDo(t, hc, http.MethodDelete, lanURL)
	assert.Equal(t, http.StatusUnprocessableEntity, got.status)
	assert.False(t, got.throttled())

	got = throttleTestDo(t, hc, http.MethodDelete, lanURL)
	assert.Equal(t, http.StatusTooManyRequests, got.status)
	assert.True(t, got.throttled())
	assert.Contains(t, got.body, "after Cloud API 422 for DELETE /cloudapi/v6/datacenters/"+faultTestDC+"/lans/1")
	assert.Equal(t, []string{"422"}, ft.log.answers(), "the held back DELETE does not reach the injector")
	assert.Equal(t, 0, ft.next.calls)
}

func TestNewIonosClient_CloudAPIFaults(t *testing.T) {
	creds := []byte(`{"user": "username","password": "cGFzc3dvcmQ="}`)
	until := time.Now().Add(time.Hour).UTC().Format(time.RFC3339)
	faults, err := ParseCloudAPIFaults([]string{"dc=" + faultTestDC + ",mode=delete-stuck,until=" + until}, time.Now(), logging.NewNopLogger())
	require.NoError(t, err)

	t.Run("injector below the throttle", func(t *testing.T) {
		freshComputeThrottle(t)
		freshIonosClientCache(t)
		freshCloudAPIFaults(t)
		ConfigureComputeThrottle(throttleTestOptions)
		ConfigureCloudAPIFaults(faults, logging.NewNopLogger())
		defaultTransport := http.DefaultClient.Transport

		svc, err := NewIonosClients(creds)
		require.NoError(t, err)
		rt, ok := svc.ComputeClient.GetConfig().HTTPClient.Transport.(*computeThrottleRoundTripper)
		require.True(t, ok, "expected *computeThrottleRoundTripper, got %T", svc.ComputeClient.GetConfig().HTTPClient.Transport)
		fi, ok := rt.next.(*faultInjector)
		require.True(t, ok, "expected *faultInjector, got %T", rt.next)
		assert.False(t, rt.nextShared)
		assert.True(t, fi.nextShared)
		assert.Equal(t, defaultTransport, http.DefaultClient.Transport, "http.DefaultClient must not be changed")
		assert.Same(t, http.DefaultClient, svc.DBaaSPostgresClient.GetConfig().HTTPClient, "only the compute client is wrapped")
	})

	t.Run("injector without the throttle", func(t *testing.T) {
		freshComputeThrottle(t)
		freshIonosClientCache(t)
		freshCloudAPIFaults(t)
		ConfigureCloudAPIFaults(faults, logging.NewNopLogger())

		svc, err := NewIonosClients(creds)
		require.NoError(t, err)
		_, ok := svc.ComputeClient.GetConfig().HTTPClient.Transport.(*faultInjector)
		assert.True(t, ok, "expected *faultInjector, got %T", svc.ComputeClient.GetConfig().HTTPClient.Transport)
	})

	t.Run("not installed without the flag", func(t *testing.T) {
		freshComputeThrottle(t)
		freshIonosClientCache(t)
		freshCloudAPIFaults(t)

		svc, err := NewIonosClients(creds)
		require.NoError(t, err)
		assert.Same(t, http.DefaultClient, svc.ComputeClient.GetConfig().HTTPClient)
	})
}
