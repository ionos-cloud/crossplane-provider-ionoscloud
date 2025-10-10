package server

import (
	"fmt"

	v1 "github.com/crossplane/crossplane-runtime/apis/common/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// UpdatedConditionType is the condition type that should be set when an update is successful
const UpdatedConditionType = v1.ConditionType("Updated")

// UpdateSucceededCondition returns the condition that should be set when an update is successful.
// A timestamp is included in the message to allow updating the condition even if it already exists with the same status.
func UpdateSucceededCondition(timestamp metav1.Time) v1.Condition {
	return v1.Condition{
		Type:               UpdatedConditionType,
		Status:             corev1.ConditionTrue,
		Reason:             "UpdateSuccessful",
		Message:            fmt.Sprintf("Update succeeded. Timestamp: %s", timestamp.String()),
		LastTransitionTime: timestamp,
	}
}

// UpdateFailedCondition returns the condition that should be set when an update fails.
// A timestamp is included in the message to allow updating the condition even if it already exists with the same status.
func UpdateFailedCondition(err error, timestamp metav1.Time) v1.Condition {
	return v1.Condition{
		Type:               UpdatedConditionType,
		Status:             corev1.ConditionFalse,
		Reason:             "UpdateFailed",
		Message:            fmt.Sprintf("Update failed: %v. Timestamp: %s", err, timestamp.String()),
		LastTransitionTime: timestamp,
	}
}
