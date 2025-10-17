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

// Constants for custom state config map functionality
const (
    // custom statuses for VMs
    statusVMRunning = "VM-RUNNING"
    statusVMBusy    = "VM-BUSY"
    statusVMError   = "VM-ERROR"

    // keys formats for config map
    stateKeyFormat= "%s-%s-state" // <prefix>-<name>-state
    stateTimestampKeyFormat = "%s-%s-timestamp" // <prefix>-<name>-timestamp
)