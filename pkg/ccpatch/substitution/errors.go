package substitution

import "errors"

var (
	// ErrMissingCIDR is returned when the CIDR is missing
	ErrMissingCIDR = errors.New("missing CIDR")
)
