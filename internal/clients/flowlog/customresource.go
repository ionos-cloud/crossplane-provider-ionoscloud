package flowlog

import (
	sdkgo "github.com/ionos-cloud/sdk-go/v6"
)

// customResource allows comparison and input generation for flow logs of different custom resources
type customResource interface {
	SetState(string)
	GetFlowLogName() string
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
	name := cr.GetFlowLogName()
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

