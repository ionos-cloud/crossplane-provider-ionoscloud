package statefulserverset

const (
	// <stateful_serverset_name>-<resource_type>-index
	replicaIndexLabel = "ionoscloud.com/%s-%s-index"
	// <stateful_serverset_name>-<resource_type>-volumeindex
	volumeIndexLabel       = "ionoscloud.com/%s-%s-volumeindex"
	statefulServerSetLabel = "ionoscloud.com/statefulStatefulServerSet"
)
const resourceDataVolume = "datavolume"
