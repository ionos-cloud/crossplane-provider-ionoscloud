package utils

import (
	"context"
	"fmt"
	"reflect"
	"slices"
	"strings"
	"time"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/retry"
)

// DepthQueryParam is used in GET requests in Cloud API
const DepthQueryParam = int32(1)

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

// StructToMap converts arbitrary structures to a flat map representation, does not recurse into slices/maps
func StructToMap(v any) map[string]any {
	m := make(map[string]any)
	var fn func(v any, keyPrefix string)
	_asMap := func(v any, keyPrefix string) {
		baseTypes := []reflect.Kind{
			reflect.Bool,
			reflect.String,
			reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64, reflect.Int,
			reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uint,
			reflect.Float32, reflect.Float64,
		}
		compositeTypes := []reflect.Kind{
			reflect.Array, reflect.Slice, reflect.Map,
		}

		rt := reflect.TypeOf(v)
		rv := reflect.ValueOf(v)

		for i := 0; i < rt.NumField(); i++ {
			f := rt.Field(i)
			v := rv.Field(i)
			key := f.Name
			if keyPrefix != "" {
				key = fmt.Sprintf("%s.%s", keyPrefix, f.Name)
			}
			key = strings.ToLower(key)
			if v.Kind() == reflect.Pointer {
				if v.IsNil() {
					continue
				}
				v = v.Elem()
			}
			kind := v.Kind()
			if kind == reflect.Struct {
				fn(v.Interface(), f.Name)
			}
			if slices.Contains(baseTypes, kind) {
				m[key] = v.Interface()
			}
			if slices.Contains(compositeTypes, kind) {
				continue
			}
		}
	}
	fn = _asMap
	_asMap(v, "")
	return m
}

// IsEqSdkPropertiesToCR compares an observed sdk struct to the crType parameters struct
func IsEqSdkPropertiesToCR(crTypeParameters any, sdkStructProperties any) bool {
	crMap := StructToMap(crTypeParameters)
	sdkMap := StructToMap(sdkStructProperties)
	for sdkField, sdkValue := range sdkMap {
		if crValue, ok := crMap[sdkField]; ok {
			if crValue != sdkValue {
				return false
			}
		}
	}
	return true
}

// Set is a generic map with comparable keys and empty struct elements with all the usual set operations
type Set[T comparable] map[T]struct{}

// Add an element to the set
func (S Set[T]) Add(elements ...T) {
	for _, e := range elements {
		S[e] = struct{}{}
	}
}

// Contains verifies if element is contained by the set
func (S Set[T]) Contains(e T) bool {
	_, ok := S[e]
	return ok
}

// Remove eliminates an element from the set, if it exists
func (S Set[T]) Remove(elements ...T) {
	for _, e := range elements {
		delete(S, e)
	}
}

// EqualTo verifies set equality
func (S Set[T]) EqualTo(s Set[T]) bool {
	if len(S) != len(s) {
		return false
	}
	for e := range s {
		if _, ok := S[e]; !ok {
			return false
		}
	}
	return true
}

// Difference returns a new Set resulting from the set difference operation
func (S Set[T]) Difference(s Set[T]) Set[T] {
	diff := Set[T]{}
	for e := range S {
		if !s.Contains(e) {
			diff[e] = struct{}{}
		}
	}
	return diff
}

// NewSetFromSlice returns a new Set from the elements of a slice
func NewSetFromSlice[T comparable](s []T) Set[T] {
	S := make(Set[T])
	for _, v := range s {
		S[v] = struct{}{}
	}
	return S
}
