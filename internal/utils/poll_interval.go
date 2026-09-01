package utils

import (
	"fmt"
	"math/rand/v2"
	"time"

	xpv1 "github.com/crossplane/crossplane-runtime/apis/common/v1"
	"github.com/crossplane/crossplane-runtime/pkg/reconciler/managed"
	"github.com/crossplane/crossplane-runtime/pkg/resource"
	corev1 "k8s.io/api/core/v1"
)

// MaxPollJitterPercentage is the exclusive upper bound of the poll jitter percentage.
// Jitter is applied as a fraction of the poll interval it shifts, so at 100 percent or
// more a resource could be given a poll interval of zero. The reconciler passes that
// straight to RequeueAfter, and controller-runtime does not requeue at all for a
// non-positive RequeueAfter, so the resource would stop being polled entirely.
const MaxPollJitterPercentage = 100

// ValidatePollJitterPercentage rejects poll jitter percentages that are out of range.
func ValidatePollJitterPercentage(percentage uint) error {
	if percentage >= MaxPollJitterPercentage {
		return fmt.Errorf("poll jitter percentage %d must be less than %d", percentage, MaxPollJitterPercentage)
	}
	return nil
}

// PollIntervalHook returns the hook that applies this provider's poll interval
// strategy to every reconciled resource. Pass it to managed.WithPollIntervalHook.
func (o *ConfigurationOptions) PollIntervalHook() managed.PollIntervalHook {
	notReadyPollInterval := o.GetNotReadyPollInterval()
	jitterPercentage := o.GetPollJitterPercentage()
	return func(mg resource.Managed, pollInterval time.Duration) time.Duration {
		return CalculatePollInterval(mg, pollInterval, notReadyPollInterval, jitterPercentage)
	}
}

// CalculatePollInterval applies notReadyPollInterval while the managed resource is not
// ready, then shifts the result by up to jitterPercentage in either direction so that
// resources reconciled together do not keep reconciling in lockstep. A notReadyPollInterval
// of zero leaves unready resources on pollInterval, and a jitterPercentage of zero disables
// jitter.
func CalculatePollInterval(mg resource.Managed, pollInterval, notReadyPollInterval time.Duration, jitterPercentage uint) time.Duration {
	if notReadyPollInterval > 0 && mg != nil && mg.GetCondition(xpv1.TypeReady).Status != corev1.ConditionTrue {
		pollInterval = notReadyPollInterval
	}
	// Jitter is a fraction of the interval it is applied to rather than an absolute
	// duration, so it scales with the interval actually in use and stays smaller than
	// it for any accepted percentage.
	jitter := (rand.Float64() - 0.5) * 2 * float64(jitterPercentage) / MaxPollJitterPercentage //nolint:gosec // No need for secure randomness.
	interval := time.Duration(float64(pollInterval) * (1 + jitter))
	if interval <= 0 {
		// Out-of-range percentages are rejected at startup, so this is only reachable
		// programmatically. Fall back to polling without jitter rather than handing the
		// reconciler an interval that would stop the resource being polled.
		return pollInterval
	}
	return interval
}
