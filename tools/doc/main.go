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
	guideGithubUrl              = "https://github.com/ionos-cloud/crossplane-provider-ionoscloud/tree/master/examples/example.md"
	examplesDirectoryPath       = "examples/ionoscloud/"
	definitionsDirectoryPath    = "package/crds/"
)

func main() {
	f, err := os.Create("docs/compute/cubeserver.md")
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
		getUsage(buf, mustGetCRDs[i])
		getProperties(buf, mustGetCRDs[i])
		err := getDefinition(buf, mustGetCRDs[i])
		if err != nil {
			return err
		}
		err = getInstance(buf, mustGetCRDs[i])
		if err != nil {
			return err
		}
	}
	_, err := buf.WriteTo(w)
	return err
}

func getOverview(buf *bytes.Buffer, crd apiextensionsv1.CustomResourceDefinition) {
	if buf == nil {
		return
	}
	buf.WriteString("## Overview\n\n")
	buf.WriteString("* Resource Name: `" + crd.Spec.Names.Kind + "`\n")
	buf.WriteString("* Resource Group: `" + crd.Spec.Group + "`\n")
	buf.WriteString("* Resource Version: `" + crd.Spec.Versions[0].Name + "`\n")
	buf.WriteString("* Resource Scope: `" + string(crd.Spec.Scope) + "`\n\n")
}

func getProperties(buf *bytes.Buffer, crd apiextensionsv1.CustomResourceDefinition) {
	if buf == nil {
		return
	}
	buf.WriteString("## Properties\n\n")
	buf.WriteString("In order to configure the IONOS Cloud Resource, the user can set the `spec.forProvider` fields into the specification file for the resource instance. The required fields that need to be set can be found [here](#required-properties). Following, there is a list of all the properties:\n\n")
	for key, value := range crd.Spec.Versions[0].Schema.OpenAPIV3Schema.Properties["spec"].Properties["forProvider"].Properties {
		buf.WriteString("* `" + key + "` (" + value.Type + ")\n")
		if !utils.IsEmptyValue(reflect.ValueOf(value.Description)) {
			buf.WriteString("	* description: " + value.Description + "\n")
		}
		if value.Type == "object" {
			buf.WriteString("	* properties:\n")
			for keyProperty, valueProperty := range value.Properties {
				buf.WriteString("		* `" + keyProperty + "` (" + valueProperty.Type + ")\n")
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
		if !utils.IsEmptyValue(reflect.ValueOf(value.Default)) {
			buf.WriteString("	* default: " + string(value.Default.Raw) + "\n")
		}
		if !utils.IsEmptyValue(reflect.ValueOf(value.Format)) {
			buf.WriteString("	* format: " + value.Format + "\n")
		}
		if !utils.IsEmptyValue(reflect.ValueOf(value.Pattern)) {
			buf.WriteString("	* pattern: " + value.Pattern + "\n")
		}
		if !utils.IsEmptyValue(reflect.ValueOf(value.Enum)) {
			var possibleValues string
			for _, v := range value.Enum {
				possibleValues = possibleValues + string(v.Raw) + ";"
			}
			buf.WriteString("	* possible values: " + strings.TrimRight(possibleValues, ";") + "\n")
		}
		// Check all validations added on apis_types
	}
	buf.WriteString("\n")
	buf.WriteString("### Required Properties\n\n")
	buf.WriteString("The user needs to set the following properties in order to configure the IONOS Cloud Resource:\n\n")
	for _, requiredValue := range crd.Spec.Versions[0].Schema.OpenAPIV3Schema.Properties["spec"].Properties["forProvider"].Required {
		buf.WriteString("* `" + requiredValue + "`\n")
	}
	buf.WriteString("\n")
}

func getDefinition(buf *bytes.Buffer, crd apiextensionsv1.CustomResourceDefinition) error {
	if buf == nil {
		return fmt.Errorf("error getting definition file path, buffer must be different than nil")
	}
	path, err := getDefinitionFilePath(crd)
	if err != nil {
		return err
	}
	if path != "" {
		buf.WriteString("## Resource Definition\n\n")
		buf.WriteString("The corresponding resource definition can be found [here](" + crossplaneProviderGithubUrl + path + ").\n")
		buf.WriteString("\n")
	}
	return nil
}

func getInstance(buf *bytes.Buffer, crd apiextensionsv1.CustomResourceDefinition) error {
	if buf == nil {
		return fmt.Errorf("error getting instance example file path, buffer must be different than nil")
	}
	path, err := getExampleFilePath(crd)
	if err != nil {
		return err
	}
	if path != "" {
		buf.WriteString("## Resource Instance Example\n\n")
		buf.WriteString("An example of a resource instance can be found [here](" + crossplaneProviderGithubUrl + path + ").\n")
		buf.WriteString("\n")
	}
	return nil
}

func getUsage(buf *bytes.Buffer, crd apiextensionsv1.CustomResourceDefinition) {
	if buf == nil {
		return
	}
	path, _ := getExampleFilePath(crd)
	if path != "" {
		buf.WriteString("## Usage\n\n")
		buf.WriteString("In order to manage resources on IONOS Cloud using Crossplane Provider, you need to have Crossplane Provider for IONOS Cloud installed into a Kubernetes Cluster, as a prerequisite. For a step-by-step guide, check the following [link](" + guideGithubUrl + ").\n\n")
		buf.WriteString("It is recommended to clone the repository for easier access to the example files.\n\n")
		buf.WriteString("### Create\n\n")
		buf.WriteString("Use the following command to create a resource instance. Before applying the file, check the properties defined in the `spec.forProvider` fields:\n\n")
		buf.WriteString("```\n")
		buf.WriteString("kubectl apply -f " + path + "\n")
		buf.WriteString("```\n\n")
		buf.WriteString("_Note_: The command should be run from the root of the `crossplane-provider-ionoscloud` directory.\n")
		buf.WriteString("### Update\n\n")
		buf.WriteString("Use the following command to update an instance. Before applying the file, update the properties defined in the `spec.forProvider` fields:\n\n")
		buf.WriteString("```\n")
		buf.WriteString("kubectl apply -f " + path + "\n")
		buf.WriteString("```\n\n")
		buf.WriteString("_Note_: The command should be run from the root of the `crossplane-provider-ionoscloud` directory.\n")
		buf.WriteString("### Wait\n\n")
		buf.WriteString("Use the following commands to wait for resources to be ready and synced. Update the `<instance-name>` accordingly:\n\n")
		buf.WriteString("```\n")
		buf.WriteString("kubectl wait --for=condition=ready " + crd.Spec.Names.Plural + "." + crd.Spec.Group + "/<instance-name>\n")
		buf.WriteString("kubectl wait --for=condition=synced " + crd.Spec.Names.Plural + "." + crd.Spec.Group + "/<instance-name>\n")
		buf.WriteString("```\n\n")
		buf.WriteString("### Get\n\n")
		buf.WriteString("Use the following command to get a list of the existing instances:\n\n")
		buf.WriteString("```\n")
		buf.WriteString("kubectl get " + crd.Spec.Names.Plural + "." + crd.Spec.Group + "\n")
		buf.WriteString("```\n\n")
		buf.WriteString("Use the following command to get a list of the existing instances with more details displayed:\n\n")
		buf.WriteString("```\n")
		buf.WriteString("kubectl get " + crd.Spec.Names.Plural + "." + crd.Spec.Group + " -o wide\n")
		buf.WriteString("```\n\n")
		buf.WriteString("Use the following command to get a list of the existing instances in JSON format:\n\n")
		buf.WriteString("```\n")
		buf.WriteString("kubectl get " + crd.Spec.Names.Plural + "." + crd.Spec.Group + " -o json\n")
		buf.WriteString("```\n\n")
		buf.WriteString("### Delete\n\n")
		buf.WriteString("Use the following command to destroy the resources created by applying the file:\n\n")
		buf.WriteString("```\n")
		buf.WriteString("kubectl delete -f " + path + "\n")
		buf.WriteString("```\n\n")
		buf.WriteString("_Note_: The command should be run from the root of the `crossplane-provider-ionoscloud` directory.\n")
		buf.WriteString("\n")
	}
}

func getDefinitionFilePath(crd apiextensionsv1.CustomResourceDefinition) (string, error) {
	filePath := fmt.Sprintf("%s%s_%s.yaml", definitionsDirectoryPath, crd.Spec.Group, crd.Spec.Names.Plural)
	if _, err := yamlFileExists(filePath); err != nil {
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
	if _, err = yamlFileExists(filePath); err != nil {
		return "", err
	}
	return filePath, nil
}

func yamlFileExists(filePath string) (bool, error) {
	if !strings.HasSuffix(filePath, ".yaml") && !strings.HasSuffix(filePath, ".yml") {
		return false, fmt.Errorf("error: not a valid path to a YAML file")
	}
	if _, err := os.ReadFile(filePath); err != nil {
		return false, err
	}
	return true, nil
}

func getServiceFromGroup(crd apiextensionsv1.CustomResourceDefinition) (string, error) {
	groupSplit := strings.Split(crd.Spec.Group, ".")
	if len(groupSplit) == 0 {
		return "", fmt.Errorf("error getting service name from the specification group")
	}
	return groupSplit[0], nil
}
