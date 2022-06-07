package crds

import (
	"embed"

	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	"k8s.io/apimachinery/pkg/util/yaml"
)

// content holds our CRDs. The embedding happens via the go build tool.
//go:embed *.yaml
var crds embed.FS

// MustGetCRDs return the CRDs installed by the crossplane provider
// The CRDs are read from embedded files.
// This function can panic in theory but never should in practice.
func MustGetCRDs() []apiextensionsv1.CustomResourceDefinition {
	crds, err := GetCRDs()
	if err != nil {
		panic(err)
	}
	return crds
}

// GetCRDs return the CRDs installed by the crossplane provider
// The CRDs are read from embedded files.
func GetCRDs() ([]apiextensionsv1.CustomResourceDefinition, error) {
	files, err := crds.ReadDir(".")
	if err != nil {
		return nil, err
	}

	ret := make([]apiextensionsv1.CustomResourceDefinition, len(files))
	for i, file := range files {
		data, err := crds.ReadFile(file.Name())
		if err != nil {
			return nil, err
		}
		var crd apiextensionsv1.CustomResourceDefinition
		if err = yaml.Unmarshal(data, &crd); err != nil {
			return nil, err
		}
		ret[i] = crd
	}

	return ret, nil
}
