package main

import (
	"bytes"
	"io"
	"os"
	"reflect"

	"github.com/ionos-cloud/crossplane-provider-ionoscloud/internal/utils"
	"github.com/ionos-cloud/crossplane-provider-ionoscloud/package/crds"
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
	for i := 0; i < len(mustGetCRDs); i++ {
		buf.WriteString("# " + mustGetCRDs[i].Spec.Names.Kind + " Managed Resource\n\n")
		buf.WriteString("## Overview\n\n")
		buf.WriteString("* Resource Name: " + mustGetCRDs[i].Spec.Names.Kind + "\n")
		buf.WriteString("* Resource Group: " + mustGetCRDs[i].Spec.Group + "\n")
		buf.WriteString("* Resource Version: " + mustGetCRDs[i].Spec.Versions[0].Name + "\n")
		buf.WriteString("* Resource Scope: " + string(mustGetCRDs[i].Spec.Scope) + "\n\n")
		buf.WriteString("## Properties:\n\n")
		for key, value := range mustGetCRDs[i].Spec.Versions[0].Schema.OpenAPIV3Schema.Properties["spec"].Properties["forProvider"].Properties {
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
		buf.WriteString("### Required Properties:\n")
		for _, requiredValue := range mustGetCRDs[i].Spec.Versions[0].Schema.OpenAPIV3Schema.Properties["spec"].Properties["forProvider"].Required {
			buf.WriteString("* `" + requiredValue + "`\n")
		}
		buf.WriteString("\n")
		buf.WriteString("## Custom Resource Definition\n\n")
		buf.WriteString("The corresponding Custom Resource Definition can be found [here](https://github.com/ionos-cloud/crossplane-provider-ionoscloud/package/crds/" +
			mustGetCRDs[i].Spec.Group + "_" + mustGetCRDs[i].Spec.Names.Plural + ".yaml).\n")
		buf.WriteString("\n")
		buf.WriteString("## Usage\n\n")
		buf.WriteString("```\n")
		buf.WriteString("kubectl get " + mustGetCRDs[i].Spec.Names.Plural + "." + mustGetCRDs[i].Spec.Group + "\n")
		buf.WriteString("```\n")
		buf.WriteString("\n")
	}
	// add entire CRD - link
	// add example - link
	// add kubectl commands - usage
	_, err := buf.WriteTo(w)
	return err
}
