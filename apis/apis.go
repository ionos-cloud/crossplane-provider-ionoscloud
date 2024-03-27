/*
Copyright 2020 The Crossplane Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

// Package apis contains Kubernetes API for the Template provider.
package apis

import (
	"k8s.io/apimachinery/pkg/runtime"

	albv1alpha1 "github.com/ionos-cloud/crossplane-provider-ionoscloud/apis/alb/v1alpha1"
	backupv1alpha1 "github.com/ionos-cloud/crossplane-provider-ionoscloud/apis/backup/v1alpha1"
	computev1alpha1 "github.com/ionos-cloud/crossplane-provider-ionoscloud/apis/compute/v1alpha1"
	dataplatformv1alpha1 "github.com/ionos-cloud/crossplane-provider-ionoscloud/apis/dataplatform/v1alpha1"
	mongov1alpha1 "github.com/ionos-cloud/crossplane-provider-ionoscloud/apis/dbaas/mongo/v1alpha1"
	postgresv1alpha1 "github.com/ionos-cloud/crossplane-provider-ionoscloud/apis/dbaas/postgres/v1alpha1"
	k8sv1alpha1 "github.com/ionos-cloud/crossplane-provider-ionoscloud/apis/k8s/v1alpha1"
	nlbv1alpha1 "github.com/ionos-cloud/crossplane-provider-ionoscloud/apis/nlb/v1alpha1"
	templatev1alpha1 "github.com/ionos-cloud/crossplane-provider-ionoscloud/apis/v1alpha1"
)

func init() {
	// Register the types with the Scheme so the components can map objects to GroupVersionKinds and back
	AddToSchemes = append(AddToSchemes,
		templatev1alpha1.SchemeBuilder.AddToScheme,
		postgresv1alpha1.SchemeBuilder.AddToScheme,
		mongov1alpha1.SchemeBuilder.AddToScheme,
		computev1alpha1.SchemeBuilder.AddToScheme,
		k8sv1alpha1.SchemeBuilder.AddToScheme,
		albv1alpha1.SchemeBuilder.AddToScheme,
		backupv1alpha1.SchemeBuilder.AddToScheme,
		dataplatformv1alpha1.SchemeBuilder.AddToScheme,
		nlbv1alpha1.SchemeBuilder.AddToScheme,
	)
}

// AddToSchemes may be used to add all resources defined in the project to a Scheme
var AddToSchemes runtime.SchemeBuilder

// AddToScheme adds all Resources to the Scheme
func AddToScheme(s *runtime.Scheme) error {
	return AddToSchemes.AddToScheme(s)
}
