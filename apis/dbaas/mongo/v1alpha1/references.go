package v1alpha1

import (
	"github.com/crossplane/crossplane-runtime/pkg/meta"
	"github.com/crossplane/crossplane-runtime/pkg/reference"
	"github.com/crossplane/crossplane-runtime/pkg/resource"
)

// ExtractMongoClusterID returns the externalName of a referenced Cluster.
func ExtractMongoClusterID() reference.ExtractValueFn {
	return func(mg resource.Managed) string {
		res, ok := mg.(*MongoCluster)
		if !ok {
			return ""
		}
		if meta.GetExternalName(res) == res.Name {
			return ""
		}
		return meta.GetExternalName(res)
	}
}
