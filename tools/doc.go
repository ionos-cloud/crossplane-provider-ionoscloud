package main

import (
	"fmt"
	"github.com/ionos-cloud/crossplane-provider-ionoscloud/package/crds"
)

func main() {
	//dir := os.Getenv("DOCS_OUT")
	//if dir == "" {
	//	fmt.Printf("DOCS_OUT environment variable not set.\n")
	//	os.Exit(1)
	//}
	//if _, err := os.Stat(dir); err != nil {
	//	fmt.Printf("Error getting directory: %v\n", err)
	//	os.Exit(1)
	//}

	mustGetCRDs := crds.MustGetCRDs()
	fmt.Println(mustGetCRDs[0].Spec.Versions[0].Schema.OpenAPIV3Schema.Properties["spec"].Properties["forProvider"].Properties["name"].Description)
	fmt.Println(mustGetCRDs[0].Spec.Versions[0].Schema.OpenAPIV3Schema.Properties["spec"].Properties["forProvider"].Required)
}
