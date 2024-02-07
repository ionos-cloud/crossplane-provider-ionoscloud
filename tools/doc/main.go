// nolint: gosec
package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"reflect"
	"sort"
	"strings"

	"github.com/ionos-cloud/crossplane-provider-ionoscloud/internal/utils"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"

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
		"mongocluster":    "mongo-cluster.yaml",
		"cluster":         "k8s-cluster.yaml",
		"nodepool":        "k8s-nodepool.yaml",
		"dataplatform":    "dataplatform-cluster.yaml",
	}
	// This tool adds the generated files provided in DOCS_OUT/<service-long-name> directory, under the name <resource-name>.md.
	// The <resource-name> is taken from the Managed Resource Spec Kind, using lower case (e.g.: kind=Cluster -> resource-name=cluster).
	// The <service-name> is taken from the Managed Resource Spec Group (e.g.: group=k8s.ionoscloud.crossplane.io -> service-name=k8s).
	// The <service-long-name> is taken from the collection defined below, using the <service-name> key:
	servicesAbbrevDirectoriesMap = map[string]string{
		"alb":          "application-load-balancer",
		"compute":      "compute-engine",
		"dbaas":        "database-as-a-service",
		"k8s":          "managed-kubernetes",
		"backup":       "managed-backup",
		"dataplatform": "dataplatform",
	}
)

func main() {
	dir := getOutputDirectory()
	fmt.Printf("Generating documentation in %s directory...\n", dir)
	if err := writeContent(dir); err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("Documentation generation completed successfully.")
}

func getOutputDirectory() string {
	dir := os.Getenv("DOCS_OUT")
	if dir == "" {
		fmt.Println("DOCS_OUT environment variable not set.")
		os.Exit(1)
	}
	return strings.TrimSuffix(dir, "/") + "/"
}

func writeContent(docsFolder string) error {
	mustGetCRDs := crds.MustGetCRDs()
	for _, crd := range mustGetCRDs {
		if err := processCRD(crd, docsFolder); err != nil {
			return err
		}
	}
	return nil
}

func processCRD(crd apiextensionsv1.CustomResourceDefinition, docsFolder string) error {
	serviceName, err := getSvcShortNameFromGroup(crd)
	if err != nil || serviceName == ionoscloudServiceName {
		return err
	}

	file, err := createOrUpdateFileForCRD(crd, docsFolder, serviceName)
	if err != nil {
		return err
	}
	defer file.Close()

	return generateCRDDocumentation(crd, file)
}

func getSvcShortNameFromGroup(crd apiextensionsv1.CustomResourceDefinition) (string, error) {
	groupSplit := strings.Split(crd.Spec.Group, ".")
	if len(groupSplit) == 0 {
		return "", fmt.Errorf("error getting service name from the specification group")
	}
	return groupSplit[0], nil
}

// createOrUpdateFileForCRD creates or updates the documentation file for a CRD
func createOrUpdateFileForCRD(crd apiextensionsv1.CustomResourceDefinition, docsFolder, serviceShortName string) (*os.File, error) {
	serviceLongDirName, ok := servicesAbbrevDirectoriesMap[serviceShortName]
	if !ok {
		return nil, fmt.Errorf("error when getting service directory name. please define the new service into the collection")
	}
	resourceName := strings.ToLower(crd.Spec.Names.Kind)
	dirPath := fmt.Sprintf("%s%s", docsFolder, serviceLongDirName)
	if _, err := os.ReadDir(dirPath); err != nil {
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

// generateCRDDocumentation generates the documentation for a given CRD
func generateCRDDocumentation(crd apiextensionsv1.CustomResourceDefinition, file io.Writer) error {
	buf := new(bytes.Buffer)

	writeOverview(buf, crd)
	err := writeUsage(buf, crd)
	if err != nil {
		return err
	}
	err = writeProperties(buf, crd)
	if err != nil {
		return err
	}
	err = writeDefinition(buf, crd)
	if err != nil {
		return err
	}
	err = writeInstanceExample(buf, crd)
	if err != nil {
		return err
	}

	_, err = buf.WriteTo(file)
	return err
}

func writeOverview(buf *bytes.Buffer, crd apiextensionsv1.CustomResourceDefinition) {
	kindName := crd.Spec.Names.Kind
	buf.WriteString("---\n")
	buf.WriteString("description: Manages " + kindName + " Resource on IONOS Cloud.\n")
	buf.WriteString("---\n\n")
	buf.WriteString("# " + kindName + " Managed Resource\n\n")

	buf.WriteString("## Overview\n\n")
	buf.WriteString("* Resource Name: `" + kindName + "`\n")
	buf.WriteString("* Resource Group: `" + crd.Spec.Group + "`\n")
	if len(crd.Spec.Versions) > 0 {
		buf.WriteString("* Resource Version: `" + crd.Spec.Versions[0].Name + "`\n")
	}
	buf.WriteString("* Resource Scope: `" + string(crd.Spec.Scope) + "`\n\n")
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
	var keys []string
	for k := range propertiesCollection.Properties {
		keys = append(keys, k)
	}
	sort.Strings(keys) // Sort the keys for deterministic order
	for _, k := range keys {
		value := propertiesCollection.Properties[k]
		writePropertiesWithPrefix(buf, value, k, "")
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

func writePropertiesWithPrefix(buf *bytes.Buffer, valueProperty apiextensionsv1.JSONSchemaProps, key, prefix string) {
	// Write the basic information about the property
	buf.WriteString(prefix + "* `" + key + "` (" + valueProperty.Type + ")\n")
	if valueProperty.Description != "" {
		buf.WriteString(prefix + "\t* description: " + valueProperty.Description + "\n")
	}
	if !utils.IsEmptyValue(reflect.ValueOf(valueProperty.Format)) {
		buf.WriteString(prefix + "\t* format: " + valueProperty.Format + "\n")
	}
	if !utils.IsEmptyValue(reflect.ValueOf(valueProperty.Default)) {
		defaultVal, _ := json.Marshal(valueProperty.Default)
		buf.WriteString(prefix + "\t* default: " + string(defaultVal) + "\n")
	}
	if valueProperty.Type == "string" && len(valueProperty.Enum) > 0 {
		enumValues := make([]string, 0, len(valueProperty.Enum))
		for _, ev := range valueProperty.Enum {
			// Marshal the enum value into JSON to get the string representation
			evBytes, err := json.Marshal(ev)
			if err != nil {
				panic("failed converting enum value to string: " + err.Error())
			}
			// Convert bytes to string and add it to the enumValues slice
			enumValues = append(enumValues, string(evBytes))
		}
		// Write the joined string representations of the enum values to the buffer
		buf.WriteString(prefix + "\t* possible values: " + strings.Join(enumValues, ", ") + "\n")
	}

	// Handling nested object properties
	if valueProperty.Type == "object" && len(valueProperty.Properties) > 0 {
		buf.WriteString(prefix + "\t* properties:\n")
		var keys []string
		for k := range valueProperty.Properties {
			keys = append(keys, k)
		}
		sort.Strings(keys) // Ensure deterministic order
		for _, k := range keys {
			nestedProp := valueProperty.Properties[k]
			writePropertiesWithPrefix(buf, nestedProp, k, prefix+"\t\t")
		}
	}

	// Handling nested array properties
	if valueProperty.Type == "array" && valueProperty.Items != nil && valueProperty.Items.Schema != nil {
		buf.WriteString(prefix + "\t* properties:\n")
		if len(valueProperty.Items.Schema.Properties) > 0 {
			var keys []string
			for k := range valueProperty.Items.Schema.Properties {
				keys = append(keys, k)
			}
			sort.Strings(keys) // Ensure deterministic order
			for _, k := range keys {
				nestedProp := valueProperty.Items.Schema.Properties[k]
				writePropertiesWithPrefix(buf, nestedProp, k, prefix+"\t\t")
			}
		} else { // Handle arrays of primitive types
			buf.WriteString(prefix + "\t\t* type: " + valueProperty.Items.Schema.Type + "\n")
			if valueProperty.Type == "string" && len(valueProperty.Enum) > 0 {
				enumValues := make([]string, 0, len(valueProperty.Enum))
				for _, ev := range valueProperty.Enum {
					// Marshal the enum value into JSON to get the string representation
					evBytes, err := json.Marshal(ev)
					if err != nil {
						panic("failed converting enum value to string: " + err.Error())
					}
					// Convert bytes to string and add it to the enumValues slice
					enumValues = append(enumValues, string(evBytes))
				}
				// Write the joined string representations of the enum values to the buffer
				buf.WriteString(prefix + "\t* possible values: " + strings.Join(enumValues, ", ") + "\n")
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
	const (
		noteCommandRecommendation = "_Note_: The command should be run from the root of the `crossplane-provider-ionoscloud` directory.\n\n"
		kubectlApplyCommand       = "```bash\nkubectl apply -f %s\n```\n\n"
		kubectlWaitCommand        = "```bash\nkubectl wait --for=condition=%s %s.%s/<instance-name>\n```\n\n"
		kubectlGetCommand         = "```bash\nkubectl get -f %s.%s\n```\n\n"
		kubectlDeleteCommand      = "```bash\nkubectl delete -f %s\n```\n\n"
	)
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
		buf.WriteString(fmt.Sprintf(kubectlApplyCommand, path))
		buf.WriteString(noteCommandRecommendation)
		buf.WriteString("### Update\n\n")
		buf.WriteString("Use the following command to update an instance. Before applying the file, update the properties defined in the `spec.forProvider` fields:\n\n")
		buf.WriteString(fmt.Sprintf(kubectlApplyCommand, path))
		buf.WriteString(noteCommandRecommendation)
		buf.WriteString("### Wait\n\n")
		buf.WriteString("Use the following commands to wait for resources to be ready and synced. Update the `<instance-name>` accordingly:\n\n")
		buf.WriteString(fmt.Sprintf(kubectlWaitCommand, "ready", crd.Spec.Names.Plural, crd.Spec.Group))
		buf.WriteString(fmt.Sprintf(kubectlWaitCommand, "synced", crd.Spec.Names.Plural, crd.Spec.Group))
		buf.WriteString("### Get\n\n")
		buf.WriteString("Use the following command to get a list of the existing instances:\n\n")
		buf.WriteString(fmt.Sprintf(kubectlGetCommand, crd.Spec.Names.Plural, crd.Spec.Group))
		buf.WriteString("_Note_: Use options `--output wide`, `--output json` to get more information about the resource instances.\n\n")
		buf.WriteString("### Delete\n\n")
		buf.WriteString("Use the following command to destroy the resources created by applying the file:\n\n")
		buf.WriteString(fmt.Sprintf(kubectlDeleteCommand, path))
		buf.WriteString(noteCommandRecommendation)
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
