package main

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"reflect"
	"strings"

	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"

	"github.com/ionos-cloud/crossplane-provider-ionoscloud/internal/utils"
	"github.com/ionos-cloud/crossplane-provider-ionoscloud/package/crds"
)

const (
	repositoryMasterGithubURL = "https://github.com/ionos-cloud/crossplane-provider-ionoscloud/tree/master/"
	guideMasterGithubURL      = "https://github.com/ionos-cloud/crossplane-provider-ionoscloud/tree/master/examples/example.md"
	examplesDirectoryPath     = "examples/ionoscloud"
	definitionsDirectoryPath  = "package/crds"
	ionoscloudServiceName     = "ionoscloud"
)

// NOTES - new integrating a new service into Crossplane Provider IONOS Cloud:
// * Check the exceptionsFileNamesExamples collection below in case the example is provided under a different name than <resource-name>.yaml.
// * Check the servicesAbbrevDirectoriesMap collection below and define the new service's entire name to be used in directory naming (the directory can be created or not).
// * You can easily generate documentation automatically using `make docs.update` target.

var (
	// This tool expects that the examples files provided are in the <service-name> directory, under the name <resource-name>.yaml.
	// The <service-name> is taken from the Managed Resource Spec Group (e.g.: group=k8s.ionoscloud.crossplane.io -> service-name=k8s).
	// The <resource-name> is taken from the Managed Resource Spec Kind, using lower case (e.g.: kind=Cluster -> resource-name=cluster).
	// If the example file for a Managed Resource does not follow the template above, please define it in the next key-value collection.
	// Define here exceptions for the example filenames:
	// <resource_name>.yaml
	exceptionsFileNamesExamples = map[string]string{
		"postgrescluster": "postgres-cluster.yaml",
		"cluster":         "k8s-cluster.yaml",
		"nodepool":        "k8s-nodepool.yaml",
	}
	// This tool adds the generated files provided in DOCS_OUT/<service-long-name> directory, under the name <resource-name>.md.
	// The <resource-name> is taken from the Managed Resource Spec Kind, using lower case (e.g.: kind=Cluster -> resource-name=cluster).
	// The <service-name> is taken from the Managed Resource Spec Group (e.g.: group=k8s.ionoscloud.crossplane.io -> service-name=k8s).
	// The <service-long-name> is taken from the collection defined below, using the <service-name> key:
	servicesAbbrevDirectoriesMap = map[string]string{
		"alb":     "application-load-balancer",
		"compute": "compute-engine",
		"dbaas":   "database-as-a-service",
		"k8s":     "managed-kubernetes",
	}
)

func main() {
	// DOCS_OUT - represents the absolute path to the directory where
	// the tool will generate the documentation files.
	dir := os.Getenv("DOCS_OUT")
	if dir == "" {
		fmt.Printf("DOCS_OUT environment variable not set.\n")
		os.Exit(1)
	}
	if _, err := os.Stat(dir); err != nil {
		fmt.Printf("error getting directory: %v\n", err)
		os.Exit(1)
	}
	if !strings.HasSuffix(dir, "/") {
		dir += "/"
	}
	fmt.Printf("Generating documentation in %s directory...\n", dir)
	err := writeContent(dir)
	if err != nil {
		panic(err)
	}
	fmt.Println("DONE!🎉")
}

func writeContent(docsFolder string) error { // nolint: gocyclo
	const errorPrinting = "resource: %s - error: %w"

	buf := new(bytes.Buffer)
	mustGetCRDs := crds.MustGetCRDs()
	for i := 0; i < len(mustGetCRDs); i++ {
		serviceName, err := getSvcShortNameFromGroup(mustGetCRDs[i])
		if err != nil {
			return err
		}
		if serviceName == ionoscloudServiceName {
			continue
		}
		w, err := createOrUpdateFileForCRD(mustGetCRDs[i], docsFolder, serviceName)
		if err != nil {
			return err
		}
		kindName := mustGetCRDs[i].Spec.Names.Kind
		buf.WriteString("---\n")
		buf.WriteString("description: Manages " + kindName + " Resource on IONOS Cloud.\n")
		buf.WriteString("---\n\n")
		buf.WriteString("# " + kindName + " Managed Resource\n\n")
		if err = getOverview(buf, mustGetCRDs[i]); err != nil {
			return fmt.Errorf(errorPrinting, kindName, err)
		}
		if err = writeUsage(buf, mustGetCRDs[i]); err != nil {
			return fmt.Errorf(errorPrinting, kindName, err)
		}
		if err = writeProperties(buf, mustGetCRDs[i]); err != nil {
			return fmt.Errorf(errorPrinting, kindName, err)
		}
		if err = writeDefinition(buf, mustGetCRDs[i]); err != nil {
			return fmt.Errorf(errorPrinting, kindName, err)
		}
		if err = writeInstanceExample(buf, mustGetCRDs[i]); err != nil {
			return fmt.Errorf(errorPrinting, kindName, err)
		}
		if _, err = buf.WriteTo(w); err != nil {
			return fmt.Errorf(errorPrinting, kindName, err)
		}
	}
	return nil
}

func createOrUpdateFileForCRD(crd apiextensionsv1.CustomResourceDefinition, docsFolder, serviceShortName string) (io.Writer, error) {
	serviceLongDirName, ok := servicesAbbrevDirectoriesMap[serviceShortName]
	if !ok {
		return nil, fmt.Errorf("error when getting service directory name. please define the new service into the collection")
	}
	resourceName := strings.ToLower(crd.Spec.Names.Kind)
	dirPath := fmt.Sprintf("%s%s", docsFolder, serviceLongDirName)
	if _, err := os.ReadDir(dirPath); err != nil {
		// If the directory does not exist yet, create it with the 0775 permissions.
		if strings.Contains(err.Error(), "no such file or directory") {
			if err = os.MkdirAll(dirPath, 0750); err != nil {
				return nil, fmt.Errorf("error creating directory %s: %w", dirPath, err)
			}
		} else {
			return nil, fmt.Errorf("error reading directory %s: %w", dirPath, err)
		}
	}
	filePath := fmt.Sprintf("%s/%s.md", dirPath, resourceName)
	f, err := os.Create(filePath)
	if err != nil {
		return nil, err
	}
	return f, nil
}

func getOverview(buf *bytes.Buffer, crd apiextensionsv1.CustomResourceDefinition) error { // nolint: interfacer
	if buf == nil {
		return fmt.Errorf("error getting overview, buffer must be different than nil")
	}
	buf.WriteString("## Overview\n\n")
	buf.WriteString("* Resource Name: `" + crd.Spec.Names.Kind + "`\n")
	buf.WriteString("* Resource Group: `" + crd.Spec.Group + "`\n")
	if len(crd.Spec.Versions) == 0 {
		return fmt.Errorf("error: CRD must have at least one version in spec.Versions")
	}
	buf.WriteString("* Resource Version: `" + crd.Spec.Versions[0].Name + "`\n")
	buf.WriteString("* Resource Scope: `" + string(crd.Spec.Scope) + "`\n\n")
	return nil
}

func writeProperties(buf *bytes.Buffer, crd apiextensionsv1.CustomResourceDefinition) error { // nolint: interfacer
	const (
		spec        = "spec"
		forProvider = "forProvider"
	)
	if buf == nil {
		return fmt.Errorf("error getting properties, buffer must be different than nil")
	}
	buf.WriteString("## Properties\n\n")
	buf.WriteString("In order to configure the IONOS Cloud Resource, the user can set the `spec.forProvider` fields into the specification file for the resource instance. The required fields that need to be set can be found [here](#required-properties). Following, there is a list of all the properties:\n\n")
	if len(crd.Spec.Versions) == 0 {
		return fmt.Errorf("error: CRD must have at least one version in spec.Versions")
	}
	propertiesCollection := crd.Spec.Versions[0].Schema.OpenAPIV3Schema.Properties[spec].Properties[forProvider]
	for key, value := range propertiesCollection.Properties {
		writePropertiesWithPrefix(buf, value, key, "")
	}
	buf.WriteString("\n")
	buf.WriteString("### Required Properties\n\n")
	buf.WriteString("The user needs to set the following properties in order to configure the IONOS Cloud Resource:\n\n")
	for _, requiredValue := range propertiesCollection.Required {
		buf.WriteString("* `" + requiredValue + "`\n")
	}
	buf.WriteString("\n")
	return nil
}

func writePropertiesWithPrefix(buf *bytes.Buffer, valueProperty apiextensionsv1.JSONSchemaProps, keyProperty, prefix string) { // nolint: interfacer,gocyclo
	buf.WriteString(prefix + "* `" + keyProperty + "` (" + valueProperty.Type + ")\n")
	if !utils.IsEmptyValue(reflect.ValueOf(valueProperty.Description)) {
		buf.WriteString(prefix + "\t* description: " + valueProperty.Description + "\n")
	}
	if !utils.IsEmptyValue(reflect.ValueOf(valueProperty.Default)) {
		buf.WriteString(prefix + "\t* default: " + string(valueProperty.Default.Raw) + "\n")
	}
	if !utils.IsEmptyValue(reflect.ValueOf(valueProperty.Format)) {
		buf.WriteString(prefix + "\t* format: " + valueProperty.Format + "\n")
	}
	if !utils.IsEmptyValue(reflect.ValueOf(valueProperty.Pattern)) {
		buf.WriteString(prefix + "\t* pattern: " + valueProperty.Pattern + "\n")
	}
	if !utils.IsEmptyValue(reflect.ValueOf(valueProperty.Enum)) {
		var possibleValues string
		for _, v := range valueProperty.Enum {
			possibleValues = possibleValues + string(v.Raw) + ";"
		}
		buf.WriteString(prefix + "\t* possible values: " + strings.TrimRight(possibleValues, ";") + "\n")
	}
	if !utils.IsEmptyValue(reflect.ValueOf(valueProperty.Minimum)) {
		buf.WriteString(prefix + "\t* minimum: " + fmt.Sprintf("%f", *valueProperty.Minimum) + "\n")
	}
	if !utils.IsEmptyValue(reflect.ValueOf(valueProperty.Maximum)) {
		buf.WriteString(prefix + "\t* maximum: " + fmt.Sprintf("%f", *valueProperty.Maximum) + "\n")
	}
	if !utils.IsEmptyValue(reflect.ValueOf(valueProperty.MultipleOf)) {
		buf.WriteString(prefix + "\t* multiple of: " + fmt.Sprintf("%f", *valueProperty.MultipleOf) + "\n")
	}
	if valueProperty.Type == "object" && len(valueProperty.Properties) > 0 {
		buf.WriteString(prefix + "\t* properties:\n")
		for keyPropertySec, valuePropertySec := range valueProperty.Properties {
			newPrefix := prefix + "\t\t"
			writePropertiesWithPrefix(buf, valuePropertySec, keyPropertySec, newPrefix)
		}
		if !utils.IsEmptyValue(reflect.ValueOf(valueProperty.Required)) {
			buf.WriteString(prefix + "\t* required properties:\n")
			for _, valuePropertyReq := range valueProperty.Required {
				buf.WriteString(prefix + "\t\t* `" + valuePropertyReq + "`\n")
			}
		}
	}
}

func writeDefinition(buf *bytes.Buffer, crd apiextensionsv1.CustomResourceDefinition) error { // nolint: interfacer
	if buf == nil {
		return fmt.Errorf("error getting definition file path, buffer must be different than nil")
	}
	path, err := getDefinitionFilePath(crd)
	if err != nil {
		return err
	}
	if path != "" {
		buf.WriteString("## Resource Definition\n\n")
		buf.WriteString("The corresponding resource definition can be found [here](" + repositoryMasterGithubURL + path + ").\n\n")
	}
	return nil
}

func writeInstanceExample(buf *bytes.Buffer, crd apiextensionsv1.CustomResourceDefinition) error { // nolint: interfacer
	if buf == nil {
		return fmt.Errorf("error getting instance example file path, buffer must be different than nil")
	}
	path, err := getExampleFilePath(crd)
	if err != nil {
		return err
	}
	if path != "" {
		buf.WriteString("## Resource Instance Example\n\n")
		buf.WriteString("An example of a resource instance can be found [here](" + repositoryMasterGithubURL + path + ").\n\n")
	}
	return nil
}

func writeUsage(buf *bytes.Buffer, crd apiextensionsv1.CustomResourceDefinition) error { // nolint: interfacer
	if buf == nil {
		return fmt.Errorf("error getting usage, buffer must be different than nil")
	}
	path, err := getExampleFilePath(crd)
	if err != nil {
		return err
	}
	if path != "" {
		buf.WriteString("## Usage\n\n")
		buf.WriteString("In order to manage resources on IONOS Cloud using Crossplane Provider, you need to have Crossplane Provider for IONOS Cloud installed into a Kubernetes Cluster, as a prerequisite. For a step-by-step guide, check the following [link](" + guideMasterGithubURL + ").\n\n")
		buf.WriteString("It is recommended to clone the repository for easier access to the example files.\n\n")
		buf.WriteString("### Create\n\n")
		buf.WriteString("Use the following command to create a resource instance. Before applying the file, check the properties defined in the `spec.forProvider` fields:\n\n")
		buf.WriteString("```\n")
		buf.WriteString("kubectl apply -f " + path + "\n")
		buf.WriteString("```\n\n")
		buf.WriteString("_Note_: The command should be run from the root of the `crossplane-provider-ionoscloud` directory.\n\n")
		buf.WriteString("### Update\n\n")
		buf.WriteString("Use the following command to update an instance. Before applying the file, update the properties defined in the `spec.forProvider` fields:\n\n")
		buf.WriteString("```\n")
		buf.WriteString("kubectl apply -f " + path + "\n")
		buf.WriteString("```\n\n")
		buf.WriteString("_Note_: The command should be run from the root of the `crossplane-provider-ionoscloud` directory.\n\n")
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
		buf.WriteString("_Note_: The command should be run from the root of the `crossplane-provider-ionoscloud` directory.\n\n")
	}
	return nil
}

func getDefinitionFilePath(crd apiextensionsv1.CustomResourceDefinition) (string, error) {
	const pathYamlDefinitionFilePrinting = "%s/%s_%s.yaml"

	filePath := fmt.Sprintf(pathYamlDefinitionFilePrinting, definitionsDirectoryPath, crd.Spec.Group, crd.Spec.Names.Plural)
	if err := yamlFileExists(filePath); err != nil {
		return "", err
	}
	return filePath, nil
}

func getExampleFilePath(crd apiextensionsv1.CustomResourceDefinition) (string, error) {
	const (
		pathPathPrinting     = "%s/%s"
		pathYamlNamePrinting = "%s.yaml"
	)

	var fileName string

	svc, err := getSvcShortNameFromGroup(crd)
	if err != nil {
		return "", err
	}
	dirExample := fmt.Sprintf(pathPathPrinting, examplesDirectoryPath, svc)
	if key, isPresent := exceptionsFileNamesExamples[strings.ToLower(crd.Spec.Names.Kind)]; isPresent {
		fileName = key
	} else {
		fileName = fmt.Sprintf(pathYamlNamePrinting, strings.ToLower(crd.Spec.Names.Kind))
	}
	filePath := fmt.Sprintf(pathPathPrinting, dirExample, fileName)
	if err = yamlFileExists(filePath); err != nil {
		return "", err
	}
	return filePath, nil
}

func yamlFileExists(filePath string) error {
	const (
		yamlSuffix = ".yaml"
		ymlSuffix  = ".yml"
	)

	if !strings.HasSuffix(filePath, yamlSuffix) && !strings.HasSuffix(filePath, ymlSuffix) {
		return fmt.Errorf("error: not a valid path to a YAML file")
	}
	if _, err := os.ReadFile(filePath); err != nil {
		return err
	}
	return nil
}

func getSvcShortNameFromGroup(crd apiextensionsv1.CustomResourceDefinition) (string, error) {
	groupSplit := strings.Split(crd.Spec.Group, ".")
	if len(groupSplit) == 0 {
		return "", fmt.Errorf("error getting service name from the specification group")
	}
	return groupSplit[0], nil
}
