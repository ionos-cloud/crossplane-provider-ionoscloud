package flowlog

import (
	"github.com/ionos-cloud/crossplane-provider-ionoscloud/internal/clients/compute"
	"github.com/ionos-cloud/crossplane-provider-ionoscloud/internal/compare"
	sdkgo "github.com/ionos-cloud/sdk-go/v6"
)

// customResource allows comparison and input generation for flow logs of different custom resources
type customResource interface {
	SetState(string)
	GetName() string
	GetAction() string
	GetDirection() string
	GetBucket() string
}

// SetStatus sets the status of the flow log custom resource observation based on sdkgo.FlowLog
func SetStatus(in customResource, flowLog sdkgo.FlowLog) {
	if flowLog.Metadata != nil && flowLog.Metadata.State != nil {
		in.SetState(*flowLog.Metadata.State)
	}
}

// GenerateCreateInput returns sdkgo.FlowLog for Create requests based on CR spec
func GenerateCreateInput(cr customResource) sdkgo.FlowLog {
	flowLogProperties := GenerateUpdateInput(cr)
	return sdkgo.FlowLog{Properties: &flowLogProperties}
}

// GenerateUpdateInput returns sdkgo.FlowLogProperties for Update requests based on CR spec
func GenerateUpdateInput(cr customResource) sdkgo.FlowLogProperties {
	name := cr.GetName()
	action := cr.GetAction()
	direction := cr.GetDirection()
	bucket := cr.GetBucket()
	return sdkgo.FlowLogProperties{
		Name:      &name,
		Action:    &action,
		Direction: &direction,
		Bucket:    &bucket,
	}
}

// IsUpToDate returns true if the FlowLog is up-to-date or false otherwise
func IsUpToDate(cr customResource, observed sdkgo.FlowLog) bool { // nolint:gocyclo
	switch {
	case cr == nil && observed.Properties == nil:
		return true
	case cr == nil && observed.Properties != nil:
		return false
	case cr != nil && observed.Properties == nil:
		return false
	case observed.Metadata != nil && observed.Metadata.State != nil && (*observed.Metadata.State == compute.BUSY || *observed.Metadata.State == compute.UPDATING):
		return true
	case !EqualFlowLogProperties(cr, *observed.Properties):
		return false
	}
	return true
}

// EqualFlowLogProperties compares a target flow log customResource to the observed sdkgo.FlowLogProperties
func EqualFlowLogProperties(target customResource, observed sdkgo.FlowLogProperties) bool {
	return compare.EqualString(target.GetName(), observed.GetName()) &&
		compare.EqualString(target.GetAction(), observed.GetAction()) &&
		compare.EqualString(target.GetDirection(), observed.GetDirection()) &&
		compare.EqualString(target.GetBucket(), observed.GetBucket())
}
