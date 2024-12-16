package serverset

import "fmt"

// ZoneDeploymentOptions is used to pass the zone and index to the ZoneStrategy
type ZoneDeploymentOptions struct {
	// Zone needed for whateverIsSet
	Zone string
	// Index is needed for eachServerInAZone
	Index int
}

// ZoneStrategy is an interface that returns the zone for a server based on the ZoneDeploymentOptions
type ZoneStrategy interface {
	GetZone(ZoneDeploymentOptions) string
}

// eachServerInAZone is a ZoneStrategy that returns ZONE_2 for odd and ZONE_1 for even index
type eachServerInAZone struct {
}

// GetZone returns ZONE_2 for odd and ZONE_1 for even index
func (e eachServerInAZone) GetZone(so ZoneDeploymentOptions) string {
	return fmt.Sprintf("ZONE_%d", so.Index%2+1)
}

// whateverIsSet is a ZoneStrategy that returns the zone set in the ZoneDeploymentOptions
type whateverIsSet struct {
}

// GetZone returns the zone set in the ZoneDeploymentOptions
func (w whateverIsSet) GetZone(so ZoneDeploymentOptions) string {
	return so.Zone
}

// NewZoneDeploymentByType returns a ZoneStrategy based on the zone string
func NewZoneDeploymentByType(zone string) ZoneStrategy {
	if zone == "" || zone == "ZONES" {
		return eachServerInAZone{}
	}
	return whateverIsSet{}
}
