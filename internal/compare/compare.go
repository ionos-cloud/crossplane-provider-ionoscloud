package compare

import (
	"strings"
	"time"

	ionosdbaas "github.com/ionos-cloud/sdk-go-bundle/products/dbaas/psql/v2"
	mongo "github.com/ionos-cloud/sdk-go-dbaas-mongo"
	ionoscloud "github.com/ionos-cloud/sdk-go/v6"

	mongoalphav1 "github.com/ionos-cloud/crossplane-provider-ionoscloud/apis/dbaas/mongo/v1alpha1"
	dbaasv1alpha1 "github.com/ionos-cloud/crossplane-provider-ionoscloud/apis/dbaas/postgres/v1alpha1"
	k8sv1alpha1 "github.com/ionos-cloud/crossplane-provider-ionoscloud/apis/k8s/v1alpha1"
)

// EqualString returns if the strings are equal. An observed nil value is equal to ""
func EqualString(targetValue string, observedValue *string) bool {
	if observedValue == nil {
		return targetValue == ""
	}
	return targetValue == *observedValue
}

// EqualDayOfTheWeek returns if the string representation of the DayOfTheWeek are equal. An observed nil value is equal to ""
func EqualDayOfTheWeek(targetValue string, observedValue *ionosdbaas.DayOfTheWeek) bool {
	if observedValue == nil {
		return targetValue == ""
	}
	return targetValue == string(*observedValue)
}

// EqualMongoDayOfTheWeek returns if the string representation of the DayOfTheWeek are equal. An observed nil value is equal to ""
func EqualMongoDayOfTheWeek(targetValue string, observedValue *mongo.DayOfTheWeek) bool {
	if observedValue == nil {
		return targetValue == ""
	}
	return targetValue == string(*observedValue)
}

// EqualKubernetesMaintenanceWindow returns true if the maintenance windows are equal
func EqualKubernetesMaintenanceWindow(targetValue k8sv1alpha1.MaintenanceWindow, observedValue *ionoscloud.KubernetesMaintenanceWindow) bool {
	if observedValue == nil {
		return targetValue.Time == "" && targetValue.DayOfTheWeek == ""
	}
	return EqualTimeString(targetValue.Time, observedValue.Time) &&
		EqualString(targetValue.DayOfTheWeek, observedValue.DayOfTheWeek)
}

// MaintenanceWindowPtrResource to be able to compare maintenance windows from different sdks
type MaintenanceWindowPtrResource interface {
	GetTime() *string
	GetDayOfTheWeek() *string
}

// MaintenanceWindowResource - interface to be able to compare maintenance windows from different sdks
type MaintenanceWindowResource interface {
	GetTime() string
	GetDayOfTheWeek() string
}

// EqualMaintenanceWindow returns true if the maintenance windows are equal
func EqualMaintenanceWindow(targetValue MaintenanceWindowResource, observedValue MaintenanceWindowPtrResource) bool {
	if observedValue == nil {
		return targetValue.GetTime() == "" && targetValue.GetDayOfTheWeek() == ""
	}

	return EqualTimeString(targetValue.GetTime(), observedValue.GetTime()) &&
		EqualString(targetValue.GetDayOfTheWeek(), observedValue.GetDayOfTheWeek())
}

// EqualDatabaseMaintenanceWindow returns true if the maintenance windows are equal
func EqualDatabaseMaintenanceWindow(targetValue dbaasv1alpha1.MaintenanceWindow, observedValue *ionosdbaas.MaintenanceWindow) bool {
	if observedValue == nil {
		return targetValue.Time == "" && targetValue.DayOfTheWeek == ""
	}
	return EqualTimeString(targetValue.Time, &observedValue.Time) &&
		EqualDayOfTheWeek(targetValue.DayOfTheWeek, &observedValue.DayOfTheWeek)
}

func EqualConnectionPooler(targetValue dbaasv1alpha1.ConnectionPooler, observedValue *ionosdbaas.ConnectionPooler) bool {
	if observedValue == nil {
		return targetValue.PoolMode == "" && targetValue.Enabled == false
	}
	return targetValue.Enabled == *observedValue.Enabled &&
		targetValue.PoolMode == *observedValue.PoolMode
}

func EqualConnections(targetValue []dbaasv1alpha1.Connection, observedValue []ionosdbaas.Connection) bool {
	if len(targetValue) != len(observedValue) {
		return false
	}
	equal := false
	for _, target := range targetValue {
		for i, _ := range observedValue {
			if EqualConnection(target, &observedValue[i]) {
				equal = true
			}
		}
		if !equal {
			return false
		}
	}
	return equal
}

func EqualConnection(targetValue dbaasv1alpha1.Connection, observedValue *ionosdbaas.Connection) bool {
	if observedValue == nil {
		return targetValue.Cidr == "" && targetValue.DatacenterCfg.DatacenterID == "" && targetValue.LanCfg.LanID == ""
	}
	return targetValue.Cidr == observedValue.Cidr &&
		targetValue.DatacenterCfg.DatacenterID == observedValue.DatacenterId &&
		targetValue.LanCfg.LanID == observedValue.LanId
}

// EqualMongoDatabaseMaintenanceWindow returns true if the maintenance windows are equal
func EqualMongoDatabaseMaintenanceWindow(targetValue mongoalphav1.MaintenanceWindow, observedValue *mongo.MaintenanceWindow) bool {
	if observedValue == nil {
		return targetValue.Time == "" && targetValue.DayOfTheWeek == ""
	}
	return EqualTimeString(targetValue.Time, observedValue.Time) &&
		EqualMongoDayOfTheWeek(targetValue.DayOfTheWeek, observedValue.DayOfTheWeek)
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
