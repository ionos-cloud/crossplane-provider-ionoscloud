package utils

import (
	"testing"
	"time"

	xpv1 "github.com/crossplane/crossplane-runtime/apis/common/v1"
	"github.com/crossplane/crossplane-runtime/pkg/resource/fake"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// testNotReadyPollInterval stands in for whatever --poll-not-ready is set to.
const testNotReadyPollInterval = 30 * time.Second

func TestValidatePollJitterPercentage(t *testing.T) {
	tests := []struct {
		name       string
		percentage uint
		expectErr  bool
	}{
		{name: "Zero disables jitter", percentage: 0},
		{name: "A moderate percentage is accepted", percentage: 10},
		{name: "Highest accepted percentage", percentage: MaxPollJitterPercentage - 1},
		{name: "A full poll interval of jitter is rejected", percentage: MaxPollJitterPercentage, expectErr: true},
		{name: "More than a full poll interval of jitter is rejected", percentage: 150, expectErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidatePollJitterPercentage(tt.percentage)
			if tt.expectErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
		})
	}
}

func TestCalculatePollIntervalUsesReadinessOfTheResource(t *testing.T) {
	tests := []struct {
		name                 string
		conditions           []xpv1.Condition
		pollInterval         time.Duration
		notReadyPollInterval time.Duration
		expectedInterval     time.Duration
	}{
		{
			name:                 "Ready resource keeps the configured poll interval",
			conditions:           []xpv1.Condition{xpv1.Available()},
			pollInterval:         time.Minute,
			notReadyPollInterval: testNotReadyPollInterval,
			expectedInterval:     time.Minute,
		},
		{
			name:                 "Unready resource is polled at the not-ready interval",
			conditions:           []xpv1.Condition{xpv1.Creating()},
			pollInterval:         time.Minute,
			notReadyPollInterval: testNotReadyPollInterval,
			expectedInterval:     testNotReadyPollInterval,
		},
		{
			name:                 "Resource without any condition is treated as unready",
			conditions:           nil,
			pollInterval:         time.Minute,
			notReadyPollInterval: testNotReadyPollInterval,
			expectedInterval:     testNotReadyPollInterval,
		},
		{
			name:                 "Not-ready interval is honoured even when longer than the configured interval",
			conditions:           []xpv1.Condition{xpv1.Creating()},
			pollInterval:         5 * time.Second,
			notReadyPollInterval: testNotReadyPollInterval,
			expectedInterval:     testNotReadyPollInterval,
		},
		{
			name:                 "Deleting resource is treated as unready",
			conditions:           []xpv1.Condition{xpv1.Deleting()},
			pollInterval:         time.Hour,
			notReadyPollInterval: testNotReadyPollInterval,
			expectedInterval:     testNotReadyPollInterval,
		},
		{
			name:                 "Zero not-ready interval leaves an unready resource on the configured interval",
			conditions:           []xpv1.Condition{xpv1.Creating()},
			pollInterval:         time.Minute,
			notReadyPollInterval: 0,
			expectedInterval:     time.Minute,
		},
		{
			name:                 "Zero not-ready interval leaves a ready resource on the configured interval",
			conditions:           []xpv1.Condition{xpv1.Available()},
			pollInterval:         time.Minute,
			notReadyPollInterval: 0,
			expectedInterval:     time.Minute,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mg := &fake.Managed{}
			mg.SetConditions(tt.conditions...)

			// Jitter is disabled, so the result is deterministic.
			result := CalculatePollInterval(mg, tt.pollInterval, tt.notReadyPollInterval, 0)
			assert.Equal(t, tt.expectedInterval, result)
		})
	}
}

func TestCalculatePollIntervalSpreadsIntervalsOverTheJitterWindow(t *testing.T) {
	const (
		pollInterval = time.Minute
		percentage   = 10
		samples      = 1000
	)
	mg := &fake.Managed{}
	mg.SetConditions(xpv1.Available())

	seen := make(map[time.Duration]struct{}, samples)
	for range samples {
		result := CalculatePollInterval(mg, pollInterval, testNotReadyPollInterval, percentage)
		assert.GreaterOrEqual(t, result, 54*time.Second)
		assert.LessOrEqual(t, result, 66*time.Second)
		seen[result] = struct{}{}
	}
	assert.Greater(t, len(seen), 1, "poll interval should not be constant across reconciles")
}

// TestCalculatePollIntervalScalesJitterToTheEffectiveInterval covers a resource that is
// up to date but not ready yet - a datacenter in state BUSY, say - under a poll interval
// long enough that a jitter sized to the configured interval would swamp the shortened
// one. A non-positive interval must never be produced: the reconciler passes it straight
// to RequeueAfter, and controller-runtime does not requeue at all for a non-positive
// RequeueAfter, so the resource would stop being polled entirely.
func TestCalculatePollIntervalScalesJitterToTheEffectiveInterval(t *testing.T) {
	mg := &fake.Managed{}
	mg.SetConditions(xpv1.Creating())

	for range 1000 {
		result := CalculatePollInterval(mg, 10*time.Minute, testNotReadyPollInterval, 10)
		assert.Positive(t, result)
		assert.GreaterOrEqual(t, result, 27*time.Second)
		assert.LessOrEqual(t, result, 33*time.Second)
	}
}

func TestCalculatePollIntervalAlwaysReturnsPositiveInterval(t *testing.T) {
	mg := &fake.Managed{}
	mg.SetConditions(xpv1.Available())

	for range 1000 {
		// Out of range, so rejected at startup, but must still not stop the
		// resource being polled if it is somehow configured programmatically.
		assert.Positive(t, CalculatePollInterval(mg, time.Minute, testNotReadyPollInterval, 5*MaxPollJitterPercentage))
	}
}

func TestCalculatePollIntervalToleratesNilManagedResource(t *testing.T) {
	assert.Equal(t, time.Minute, CalculatePollInterval(nil, time.Minute, testNotReadyPollInterval, 0))
}

func TestPollIntervalHookUsesTheConfiguredOptions(t *testing.T) {
	ready := &fake.Managed{}
	ready.SetConditions(xpv1.Available())
	unready := &fake.Managed{}
	unready.SetConditions(xpv1.Creating())

	t.Run("Applies the jitter percentage from the options", func(t *testing.T) {
		opts := &ConfigurationOptions{PollJitterPercentage: 10}
		result := opts.PollIntervalHook()(ready, time.Minute)
		assert.GreaterOrEqual(t, result, 54*time.Second)
		assert.LessOrEqual(t, result, 66*time.Second)
	})

	t.Run("Applies the not-ready interval from the options", func(t *testing.T) {
		opts := &ConfigurationOptions{NotReadyPollInterval: testNotReadyPollInterval}
		assert.Equal(t, testNotReadyPollInterval, opts.PollIntervalHook()(unready, time.Minute))
		assert.Equal(t, time.Minute, opts.PollIntervalHook()(ready, time.Minute))
	})

	t.Run("Is a no-op with zero valued options", func(t *testing.T) {
		opts := &ConfigurationOptions{}
		assert.Equal(t, time.Minute, opts.PollIntervalHook()(unready, time.Minute))
		assert.Equal(t, time.Minute, opts.PollIntervalHook()(ready, time.Minute))
	})

	t.Run("Handles nil options gracefully", func(t *testing.T) {
		var opts *ConfigurationOptions
		assert.Equal(t, time.Minute, opts.PollIntervalHook()(unready, time.Minute))
	})
}
