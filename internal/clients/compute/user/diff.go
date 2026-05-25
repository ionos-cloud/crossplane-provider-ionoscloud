package user

import (
	ionosdk "github.com/ionos-cloud/sdk-go/v6"
	"k8s.io/apimachinery/pkg/util/sets"

	"github.com/ionos-cloud/crossplane-provider-ionoscloud/apis/compute/v1alpha1"
	"github.com/ionos-cloud/crossplane-provider-ionoscloud/internal/diff"
)

// IsUserUpToDate returns whether the User is up-to-date and a diff string.
func IsUserUpToDate(params v1alpha1.UserParameters, observed ionosdk.User, observedGroups []string) (bool, string) {
	if !observed.HasProperties() {
		return false, "user properties not set"
	}

	// After creation the password is stored as a connection detail secret
	// and removed from the cr. If the cr has a password it means
	// the client wants to update it.
	if params.Password != "" {
		return false, "password update requested"
	}

	props := observed.GetProperties()
	d := diff.New()
	if adm := props.GetAdministrator(); adm != nil {
		d.Bool("administrator", &params.Administrator, adm)
	}
	if email := props.GetEmail(); email != nil {
		d.Str("email", &params.Email, email)
	}
	if fname := props.GetFirstname(); fname != nil {
		d.Str("firstName", &params.FirstName, fname)
	}
	if fsec := props.GetForceSecAuth(); fsec != nil {
		d.Bool("forceSecAuth", &params.ForceSecAuth, fsec)
	}
	if lname := props.GetLastname(); lname != nil {
		d.Str("lastName", &params.LastName, lname)
	}
	if active := props.GetActive(); active != nil {
		d.Bool("active", &params.Active, active)
	}
	if params.GroupIDs != nil {
		configuredGroups := sets.New[string](*params.GroupIDs...)
		if !configuredGroups.Equal(sets.New[string](observedGroups...)) {
			d.Add("groups", "<changed>", "<changed>")
		}
	}
	return d.Result()
}
