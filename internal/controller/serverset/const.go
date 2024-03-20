package serverset

const (
	// ResourceServer name for server used for serverset
	ResourceServer     = "server"
	resourceBootVolume = "bootvolume"
	resourceNIC        = "nic"
)

const (
	statusReady   = "READY"
	statusUnknown = "UNKNOWN"
	statusError   = "ERROR"
)

// <serverset_name>-volume-selector
const volumeSelectorName = "%s-volume-selector"
