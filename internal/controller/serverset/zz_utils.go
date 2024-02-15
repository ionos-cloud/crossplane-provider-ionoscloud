package serverset

import (
	"fmt"

	"github.com/ionos-cloud/crossplane-provider-ionoscloud/apis/compute/v1alpha1"
)

func getServerName(cr *v1alpha1.ServerSet, idx int) string {

	return fmt.Sprintf("%s-%d", cr.Spec.ForProvider.Template.Metadata.Name, idx)
}

func getNICName(cr *v1alpha1.ServerSet, idx int) string {
	return fmt.Sprintf("%s-%d-nic", cr.Spec.ForProvider.Template.Metadata.Name, idx)
}
