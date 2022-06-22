package utils

import (
	"fmt"

	"github.com/crossplane/crossplane-runtime/pkg/logging"
	"github.com/go-logr/zapr"
	"go.uber.org/zap"
)

// NewTestLogger returns a logger used in unit tests.
func NewTestLogger() logging.Logger {
	zapLog, err := zap.NewDevelopment()
	if err != nil {
		panic(fmt.Sprintf("who watches the watchmen (%v)?", err))
	}
	return logging.NewLogrLogger(zapr.NewLogger(zapLog))
}
