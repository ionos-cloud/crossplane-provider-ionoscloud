package utils

import (
	"context"
	"fmt"
	"reflect"
	"strings"
	"time"

	xpv1 "github.com/crossplane/crossplane-runtime/apis/common/v1"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/retry"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// Error422 is the error code for unprocessable entity
const Error422 = "422 Unprocessable Entity"

// DepthQueryParam is used in GET requests in Cloud API
const DepthQueryParam = int32(1)

// UpdateSucceededConditionType is the condition type that should be set when an update is successful
const UpdateSucceededConditionType = xpv1.ConditionType("UpdateSucceeded")

// DereferenceOrZero returns the value of a pointer or a zero value if the pointer is nil.
func DereferenceOrZero[T any](v *T) T {
	if v == nil {
		return *new(T)
	}

	return *v
}

// IsEmptyValue checks if a value is empty or not.
// nolint
func IsEmptyValue(v reflect.Value) bool {
	switch v.Kind() {
	case reflect.Array, reflect.Map, reflect.Slice, reflect.String:
		return v.Len() == 0
	case reflect.Bool:
		return !v.Bool()
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return v.Int() == 0
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		return v.Uint() == 0
	case reflect.Float32, reflect.Float64:
		return v.Float() == 0
	case reflect.Interface, reflect.Ptr:
		return v.IsNil()
	case reflect.Struct:
		return v.IsZero()
	}
	return false
}

// IsEqStringSlices will return true if the slices are equal
// (having the same length, and the same value at the same index)
func IsEqStringSlices(first, second []string) bool {
	if len(first) != len(second) {
		return false
	}
	for i, v := range first {
		if v != second[i] {
			return false
		}
	}
	return true
}

// IsEqStringMaps will return true if the maps are equal
func IsEqStringMaps(first, second map[string]string) bool {
	if len(first) != len(second) {
		return false
	}
	for firstKey, firstValue := range first {
		if secondValue, ok := second[firstKey]; !ok || secondValue != firstValue {
			return false
		}
	}
	return true
}

// IsStringInSlice will return true if the slice contains the specific string
func IsStringInSlice(input []string, specific string) bool {
	for _, element := range input {
		if element == specific {
			return true
		}
	}
	return false
}

// ContainsStringSlices will return true if the slices
// have the same length and the same elements, even if
// they are located at different indexes.
func ContainsStringSlices(first, second []string) bool {
	if len(first) != len(second) {
		return false
	}
	if len(first) == 0 {
		return true
	}
	for _, v := range first {
		if !ContainsStringInSlice(second, v) {
			return false
		}
	}
	return true
}

// ContainsStringInSlice will return true if the slice contains string
func ContainsStringInSlice(input []string, specific string) bool {
	for _, element := range input {
		if strings.Contains(element, specific) {
			return true
		}
	}
	return false
}

// IsResourceDeletedFunc polls api to see if resource exists based on id
type IsResourceDeletedFunc func(ctx context.Context, ids ...string) (bool, error)

// WaitForResourceToBeDeleted - keeps retrying until resource is not found(404), or until ctx is cancelled
func WaitForResourceToBeDeleted(ctx context.Context, timeoutInMinutes time.Duration, fn IsResourceDeletedFunc, ids ...string) error {

	err := retry.RetryContext(ctx, timeoutInMinutes, func() *retry.RetryError {
		isDeleted, err := fn(ctx, ids...)
		if isDeleted {
			return nil
		}
		if err != nil {
			retry.NonRetryableError(err)
		}
		return retry.RetryableError(fmt.Errorf("resource with ids %v found, still trying ", ids))
	})
	return err
}

// MapStringToAny converts map[string]string to map[string]any
func MapStringToAny(sMap map[string]string) map[string]any {
	aMap := make(map[string]any)
	for k, v := range sMap {
		aMap[k] = v
	}
	return aMap
}

// NewOwnerReference creates a new OwnerReference to be added to a child resource's metadata
func NewOwnerReference(parentTypeMeta v1.TypeMeta, parentObjectMeta v1.ObjectMeta, isController, blockOwnerDeletion bool) v1.OwnerReference {
	return v1.OwnerReference{
		APIVersion:         parentTypeMeta.APIVersion,
		Kind:               parentTypeMeta.Kind,
		Name:               parentObjectMeta.Name,
		UID:                parentObjectMeta.UID,
		Controller:         &isController,
		BlockOwnerDeletion: &blockOwnerDeletion,
	}
}

// UpdateSucceededCondition returns the condition that should be set when an update is successful
func UpdateSucceededCondition() xpv1.Condition {
	return xpv1.Condition{
		Type:               UpdateSucceededConditionType,
		Status:             corev1.ConditionTrue,
		Reason:             "UpdateFinishedSuccessfully",
		Message:            "Update was performed successfully.",
		LastTransitionTime: v1.Now(),
	}
}
