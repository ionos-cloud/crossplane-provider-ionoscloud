package main

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"reflect"
	"strings"

	"github.com/ionos-cloud/crossplane-provider-ionoscloud/internal/utils"
	"github.com/ionos-cloud/crossplane-provider-ionoscloud/package/crds"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
)

const (
	crossplaneProviderGithubUrl = "https://github.com/ionos-cloud/crossplane-provider-ionoscloud/tree/master/"
	examplesDirectoryPath       = "examples/ionoscloud/"
	definitionsDirectoryPath    = "package/crds/"
)

func main() {
	f, err := os.Create("test.md")
	if err != nil {
		panic(err)
	}
	defer f.Close()
	err = writeContent(f)
	if err != nil {
		panic(err)
	}
}

func writeContent(w io.Writer) error {
	buf := new(bytes.Buffer)
	mustGetCRDs := crds.MustGetCRDs()
	for i := 0; i < 1; i++ {
		buf.WriteString("# " + mustGetCRDs[i].Spec.Names.Kind + " Managed Resource\n\n")
		getOverview(buf, mustGetCRDs[i])
		getProperties(buf, mustGetCRDs[i])
		getDefinition(buf, mustGetCRDs[i])
		getInstance(buf, mustGetCRDs[i])
		getUsage(buf, mustGetCRDs[i])
	}
	_, err := buf.WriteTo(w)
	return err
}

func getOverview(buf *bytes.Buffer, crd apiextensionsv1.CustomResourceDefinition) {
	if buf == nil {
		return
	}
	buf.WriteString("## Overview\n\n")
	buf.WriteString("* Resource Name: " + crd.Spec.Names.Kind + "\n")
	buf.WriteString("* Resource Group: " + crd.Spec.Group + "\n")
	buf.WriteString("* Resource Version: " + crd.Spec.Versions[0].Name + "\n")
	buf.WriteString("* Resource Scope: " + string(crd.Spec.Scope) + "\n\n")
}

func getProperties(buf *bytes.Buffer, crd apiextensionsv1.CustomResourceDefinition) {
	if buf == nil {
		return
	}
	buf.WriteString("## Properties\n\n")
	buf.WriteString("The user can set the following properties in order to configure the IONOS Cloud Resource:\n\n")
	for key, value := range crd.Spec.Versions[0].Schema.OpenAPIV3Schema.Properties["spec"].Properties["forProvider"].Properties {
		buf.WriteString("* `" + key + "`\n")
		if !utils.IsEmptyValue(reflect.ValueOf(value.Description)) {
			buf.WriteString("	* description: " + value.Description + "\n")
		}
		if !utils.IsEmptyValue(reflect.ValueOf(value.Type)) {
			buf.WriteString("	* type: " + value.Type + "\n")
			if value.Type == "object" {
				buf.WriteString("	* properties:\n")
				for keyProperty, valueProperty := range value.Properties {
					buf.WriteString("		* `" + keyProperty + "`\n")
					if !utils.IsEmptyValue(reflect.ValueOf(valueProperty.Description)) {
						buf.WriteString("			* description: " + valueProperty.Description + "\n")
					}
				}
				if !utils.IsEmptyValue(reflect.ValueOf(value.Required)) {
					buf.WriteString("	* required properties:\n")
					for _, valueProperty := range value.Required {
						buf.WriteString("		* `" + valueProperty + "`\n")
					}
				}
			}
		}
		if !utils.IsEmptyValue(reflect.ValueOf(value.Default)) {
			buf.WriteString("	* default: " + value.Default.String() + "\n")
		}
		if !utils.IsEmptyValue(reflect.ValueOf(value.Format)) {
			buf.WriteString("	* format: " + value.Format + "\n")
		}
		if !utils.IsEmptyValue(reflect.ValueOf(value.Pattern)) {
			buf.WriteString("	* pattern: " + value.Pattern + "\n")
		}
		// Check all validations added on apis_types
	}
	buf.WriteString("\n")
	buf.WriteString("### Required Properties\n")
	buf.WriteString("The user needs to set the following properties in order to configure the IONOS Cloud Resource:\n\n")
	for _, requiredValue := range crd.Spec.Versions[0].Schema.OpenAPIV3Schema.Properties["spec"].Properties["forProvider"].Required {
		buf.WriteString("* `" + requiredValue + "`\n")
	}
	buf.WriteString("\n")
}

func getDefinition(buf *bytes.Buffer, crd apiextensionsv1.CustomResourceDefinition) {
	if buf == nil {
		return
	}
	path, _ := getDefinitionFilePath(crd)
	if path != "" {
		buf.WriteString("## Resource Definition\n\n")
		buf.WriteString("The corresponding resource definition can be found [here](" + crossplaneProviderGithubUrl + path + ").\n")
		buf.WriteString("\n")
	}
}

func getInstance(buf *bytes.Buffer, crd apiextensionsv1.CustomResourceDefinition) {
	if buf == nil {
		return
	}
	path, _ := getExampleFilePath(crd)
	if path != "" {
		buf.WriteString("## Resource\n\n")
		buf.WriteString("An example for a resource instance can be found [here](" + crossplaneProviderGithubUrl + path + ").\n")
		buf.WriteString("\n")
	}
}

func getUsage(buf *bytes.Buffer, crd apiextensionsv1.CustomResourceDefinition) {
	if buf == nil {
		return
	}
	path, _ := getExampleFilePath(crd)
	if path != "" {
		buf.WriteString("## Usage\n\n")
		buf.WriteString("### Create/Update\n\n")
		buf.WriteString("The following command should be run from the root of the `crossplane-provider-ionoscloud` directory. Before applying the file, make sure to check the properties defined in the `spec.forProvider` fields:\n\n")
		buf.WriteString("```\n")
		buf.WriteString("kubectl apply -f " + path + "\n")
		buf.WriteString("```\n")
		buf.WriteString("\n")
		buf.WriteString("### Get\n\n")
		buf.WriteString("```\n")
		buf.WriteString("kubectl get " + crd.Spec.Names.Plural + "." + crd.Spec.Group + "\n")
		buf.WriteString("```\n")
		buf.WriteString("\n")
		buf.WriteString("### Delete\n\n")
		buf.WriteString("```\n")
		buf.WriteString("kubectl delete -f " + path + "\n")
		buf.WriteString("```\n\n")
		buf.WriteString("**Note**: the commands presented should be run from the root of the `crossplane-provider-ionoscloud` directory. " +
			"Please clone the repository for easier access.")
		buf.WriteString("\n")
	}
}

func getDefinitionFilePath(crd apiextensionsv1.CustomResourceDefinition) (string, error) {
	filePath := fmt.Sprintf("%s%s_%s.yaml", definitionsDirectoryPath, crd.Spec.Group, crd.Spec.Names.Plural)
	_, err := os.ReadFile(filePath)
	if err != nil {
		return "", err
	}
	return filePath, nil
}

func getExampleFilePath(crd apiextensionsv1.CustomResourceDefinition) (string, error) {
	svc, err := getServiceFromGroup(crd)
	if err != nil {
		return "", err
	}
	filePath := fmt.Sprintf("%s%s/%s.yaml", examplesDirectoryPath, svc, crd.Spec.Names.Singular)
	_, err = os.ReadFile(filePath)
	if err != nil {
		return "", err
	}
	return filePath, nil
}

func getServiceFromGroup(crd apiextensionsv1.CustomResourceDefinition) (string, error) {
	groupSplit := strings.Split(crd.Spec.Group, ".")
	if len(groupSplit) == 0 {
		return "", fmt.Errorf("error getting service name from the specification group")
	}
	return groupSplit[0], nil
}
