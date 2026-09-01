package utils

import (
	"strings"
	"time"

	"github.com/crossplane/crossplane-runtime/pkg/controller"
)

// ConfigurationOptions are options used in setting the provider
// and the controllers of the provider.
type ConfigurationOptions struct {
	CreationGracePeriod      time.Duration
	Timeout                  time.Duration
	IsUniqueNamesEnabled     bool
	MaxReconcilesPerResource map[string]int
	// PollJitterPercentage is the percentage of its poll interval by which a single
	// resource's poll interval is randomly shifted, in either direction.
	PollJitterPercentage uint
	// NotReadyPollInterval is the poll interval used for a resource that is not ready
	// yet. Zero leaves unready resources on the configured poll interval.
	NotReadyPollInterval time.Duration
	// CtrlOpts are crossplane-specific controller options
	CtrlOpts controller.Options
}

// NewConfigurationOptions sets fields for ConfigurationOptions and return a new ConfigurationOptions
func NewConfigurationOptions(timeout, createGracePeriod time.Duration, uniqueNamesEnable bool, ctrlOpts controller.Options) *ConfigurationOptions {
	return &ConfigurationOptions{
		CreationGracePeriod:  createGracePeriod,
		IsUniqueNamesEnabled: uniqueNamesEnable,
		Timeout:              timeout,
		CtrlOpts:             ctrlOpts,
	}
}

// GetPollInterval returns the value for the PollInterval option
func (o *ConfigurationOptions) GetPollInterval() time.Duration {
	if o == nil {
		return 0
	}
	return o.CtrlOpts.PollInterval
}

// GetPollJitterPercentage returns the value for the PollJitterPercentage option
func (o *ConfigurationOptions) GetPollJitterPercentage() uint {
	if o == nil {
		return 0
	}
	return o.PollJitterPercentage
}

// GetNotReadyPollInterval returns the value for the NotReadyPollInterval option
func (o *ConfigurationOptions) GetNotReadyPollInterval() time.Duration {
	if o == nil {
		return 0
	}
	return o.NotReadyPollInterval
}

// GetCreationGracePeriod returns the value for the CreationGracePeriod option
func (o *ConfigurationOptions) GetCreationGracePeriod() time.Duration {
	if o == nil {
		return 0
	}
	return o.CreationGracePeriod
}

// GetTimeout returns the value for the Timeout option
func (o *ConfigurationOptions) GetTimeout() time.Duration {
	if o == nil {
		return 0
	}
	return o.Timeout
}

// GetIsUniqueNamesEnabled returns the value for the IsUniqueNamesEnabled option
func (o *ConfigurationOptions) GetIsUniqueNamesEnabled() bool {
	if o == nil {
		return false
	}
	return o.IsUniqueNamesEnabled
}

// DefaultMaxReconcileRatePerResource is the default max reconcile rate for stateful server sets
const DefaultMaxReconcileRatePerResource = 1

// GetMaxConcurrentReconcileRate returns the value set in the map for the kind provided, or the default global values set in max-reconcile-rate
func (o *ConfigurationOptions) GetMaxConcurrentReconcileRate(kind string) int {
	if o == nil {
		return DefaultMaxReconcileRatePerResource
	}
	if reconcileRate, ok := o.MaxReconcilesPerResource[strings.ToLower(kind)]; ok {
		return reconcileRate
	}
	return o.CtrlOpts.MaxConcurrentReconciles
}
