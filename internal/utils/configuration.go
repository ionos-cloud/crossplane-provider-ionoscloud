package utils

import "time"

// ConfigurationOptions are options used in setting the provider
// and the controllers of the provider.
type ConfigurationOptions struct {
	PollInterval         time.Duration
	CreationGracePeriod  time.Duration
	Timeout              time.Duration
	IsUniqueNamesEnabled bool
}

// NewConfigurationOptions sets fields for ConfigurationOptions and return a new ConfigurationOptions
func NewConfigurationOptions(poll, createGracePeriod, timeout time.Duration, uniqueNamesEnable bool) *ConfigurationOptions {
	return &ConfigurationOptions{
		PollInterval:         poll,
		CreationGracePeriod:  createGracePeriod,
		Timeout:              timeout,
		IsUniqueNamesEnabled: uniqueNamesEnable,
	}
}

// GetPollInterval returns the value for the PollInterval option
func (o *ConfigurationOptions) GetPollInterval() time.Duration {
	if o == nil {
		return 0
	}
	return o.PollInterval
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
