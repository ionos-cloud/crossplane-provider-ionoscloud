package utils

import (
	"context"
	"fmt"
	"reflect"
	"strings"
	"time"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/retry"
)

// DepthQueryParam is used in GET requests in Cloud API
const DepthQueryParam = int32(1)

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
type IsResourceDeletedFunc func(ctx context.Context, ID string) (bool, error)

// WaitForResourceToBeDeleted - keeps retrying until resource is not found(404), or until ctx is cancelled
func WaitForResourceToBeDeleted(ctx context.Context, ID string, timeoutInMinutes time.Duration, fn IsResourceDeletedFunc) error {

	err := retry.RetryContext(ctx, timeoutInMinutes*time.Minute, func() *retry.RetryError {
		isDeleted, err := fn(ctx, ID)
		if isDeleted {
			return nil
		}
		if err != nil {
			retry.NonRetryableError(err)
		}
		return retry.RetryableError(fmt.Errorf("resource with id %s found, still trying ", ID))
	})
	return err
}
