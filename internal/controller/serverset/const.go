package serverset

const (
	// ResourceServer name for server used for serverset
	ResourceServer     = "srv"
	resourceBootVolume = "bv"
	resourceNIC        = "nic"
)

const (
	statusReady   = "READY"
	statusUnknown = "UNKNOWN"
	statusError   = "ERROR"
	statusBusy    = "BUSY"
)

const (
	// custom statuses for VMs
	statusVMRunning = "VM-RUNNING"
	statusVMError   = "VM-ERROR"

	// keys formats for config map
	stateKeyFormat          = "%s-state"     // <prefix>-<name>-state
	stateTimestampKeyFormat = "%s-timestamp" // <prefix>-<name>-timestamp
)
