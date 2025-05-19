package utils

import (
	"testing"

	"github.com/crossplane/crossplane-runtime/pkg/controller"
	"github.com/stretchr/testify/assert"
)

func TestGetMaxConcurrentReconcileRateHandlesVariousScenarios(t *testing.T) {
	tests := []struct {
		name                  string
		options               *ConfigurationOptions
		kind                  string
		expectedReconcileRate int
	}{
		{
			name: "Returns specific rate for known kind",
			options: &ConfigurationOptions{
				MaxReconcilesPerResource: map[string]int{"statefulserverset": 3},
				CtrlOpts:                 controller.Options{MaxConcurrentReconciles: 5},
			},
			kind:                  "statefulserverset",
			expectedReconcileRate: 3,
		},
		{
			name: "Returns default rate for unknown kind",
			options: &ConfigurationOptions{
				MaxReconcilesPerResource: map[string]int{},
				CtrlOpts:                 controller.Options{MaxConcurrentReconciles: 5},
			},
			kind:                  "unknown",
			expectedReconcileRate: 5,
		},
		{
			name:                  "Handles nil options gracefully",
			options:               nil,
			kind:                  "statefulserverset",
			expectedReconcileRate: DefaultMaxReconcileRatePerResource,
		},
		{
			name: "Handles case-insensitive kind",
			options: &ConfigurationOptions{
				MaxReconcilesPerResource: map[string]int{"statefulserverset": 3},
				CtrlOpts:                 controller.Options{MaxConcurrentReconciles: 5},
			},
			kind:                  "StatefulServerSet",
			expectedReconcileRate: 3,
		},
		{
			name: "Returns default rate when map is nil",
			options: &ConfigurationOptions{
				MaxReconcilesPerResource: nil,
				CtrlOpts:                 controller.Options{MaxConcurrentReconciles: 5},
			},
			kind:                  "statefulserverset",
			expectedReconcileRate: 5,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.options.GetMaxConcurrentReconcileRate(tt.kind)
			assert.Equal(t, tt.expectedReconcileRate, result)
		})
	}
}
