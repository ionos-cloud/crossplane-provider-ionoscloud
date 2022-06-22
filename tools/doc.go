package main

import (
	"bytes"
	"io"
	"os"

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
	buf.WriteString("# " + mustGetCRDs[0].Spec.Names.Kind + " Managed Resource\n\n")
	buf.WriteString("# Overview\n\n")
	buf.WriteString("* Resource Name: " + mustGetCRDs[0].Spec.Names.Kind + "\n")
	buf.WriteString("* Resource Group: " + mustGetCRDs[0].Spec.Group + "\n")
	buf.WriteString("* Resource Version: " + mustGetCRDs[0].Spec.Versions[0].Name + "\n")
	buf.WriteString("* Resource Scope: " + string(mustGetCRDs[0].Spec.Scope) + "\n\n")
	buf.WriteString("## Properties:\n\n")
	for key, requiredValue := range mustGetCRDs[0].Spec.Versions[0].Schema.OpenAPIV3Schema.Properties["spec"].Properties["forProvider"].Properties {
		buf.WriteString("* " + key + "\n")
		buf.WriteString("	* description: " + requiredValue.Description + "\n")
		buf.WriteString("	* type: " + requiredValue.Type + "\n")
		buf.WriteString("	* default: " + requiredValue.Default.String() + "\n")
		buf.WriteString("	* format: " + requiredValue.Format + "\n")
		buf.WriteString("	* pattern: " + requiredValue.Pattern + "\n")
		// Check all validations added on apis_types
	}
	buf.WriteString("\n")
	buf.WriteString("## Required Properties:\n")
	for _, requiredValue := range mustGetCRDs[0].Spec.Versions[0].Schema.OpenAPIV3Schema.Properties["spec"].Properties["forProvider"].Required {
		buf.WriteString("* " + requiredValue + "\n")
	}
	// add entire CRD - link
	// add example - link
	// add kubectl commands - usage
	_, err := buf.WriteTo(w)
	return err
}
