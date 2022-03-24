package utils

import (
	"reflect"
	"strings"
)

// DepthQueryParam is used in GET requests in Cloud API
const DepthQueryParam = int32(5)

// IsEmptyValue checks if a value is empty or not.
//nolint
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

// ContainsStringSlices will return true if the second slice string
// contains string of the first slice string.
func ContainsStringSlices(first, second []string) bool {
	if len(first) != len(second) {
		return false
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
