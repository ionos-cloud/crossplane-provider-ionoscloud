package compare

import (
	"strings"
	"time"

	ionoscloud "github.com/ionos-cloud/sdk-go/v6"

	"github.com/ionos-cloud/crossplane-provider-ionoscloud/apis/k8s/v1alpha1"
)

// EqualString returns if the strings are equal. An observed nil value is equal to ""
func EqualString(targetValue string, observedValue *string) bool {
	if observedValue == nil {
		return targetValue == ""
	}
	return targetValue == *observedValue
}

// EqualMaintananceWindow returns true if the maintenance windows are equal
func EqualMaintananceWindow(
	targetValue v1alpha1.MaintenanceWindow,
	observedValue *ionoscloud.KubernetesMaintenanceWindow,
) bool {
	if observedValue == nil {
		return targetValue.Time == "" && targetValue.DayOfTheWeek == ""
	}
	return EqualTimeString(targetValue.Time, observedValue.Time) &&
		EqualString(targetValue.DayOfTheWeek, observedValue.DayOfTheWeek)
}

// EqualTimeString compares the two given strings if they are represent the same point in time.
// This function assumes the timeformat is HH:mm:ssZ. If the Z is missing, it will be added.
func EqualTimeString(targetValue string, observedValue *string) bool {
	if observedValue == nil {
		return targetValue == ""
	}
	const layout = "15:04:05Z"
	target, err := time.Parse(layout, addOptionalZ(targetValue))
	if err != nil {
		return false
	}
	observed, err := time.Parse(layout, addOptionalZ(*observedValue))
	if err != nil {
		return false
	}
	return target.Equal(observed)
}

func addOptionalZ(in string) string {
	if strings.HasSuffix(in, "Z") {
		return in
	}
	return in + "Z"
}
