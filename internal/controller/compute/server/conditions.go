package server

import (
	"fmt"

	"github.com/crossplane/crossplane-runtime/apis/common/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// UpdatedConditionType is the condition type that should be set when an update is successful
const UpdatedConditionType = v1.ConditionType("Updated")

// UpdateSucceededCondition returns the condition that should be set when an update is successful
func UpdateSucceededCondition() v1.Condition {
	return v1.Condition{
		Type:               UpdatedConditionType,
		Status:             corev1.ConditionTrue,
		Reason:             "UpdateSuccessful",
		Message:            "Update was performed successfully.",
		LastTransitionTime: metav1.Now(),
	}
}

// UpdateFailedCondition returns the condition that should be set when an update fails
func UpdateFailedCondition(err error) v1.Condition {
	return v1.Condition{
		Type:               UpdatedConditionType,
		Status:             corev1.ConditionFalse,
		Reason:             "UpdateFailed",
		Message:            fmt.Sprintf("Update failed: %v", err),
		LastTransitionTime: metav1.Now(),
	}
}
