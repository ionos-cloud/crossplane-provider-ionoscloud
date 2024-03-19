package serverset

const (
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
