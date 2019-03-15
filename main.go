package main

import (
	"fmt"

	"github.com/hashicorp/terraform/helper/schema"
	"github.com/terraform-providers/terraform-provider-vsphere/vsphere"
)

var count map[string]bool

func main() {
	//plugin.Serve(&plugin.ServeOpts{
	//ProviderFunc: vsphere.Provider})

	stuff := vsphere.GetResourceMap()
	count = make(map[string]bool, 0)

	for name, resource := range stuff {
		// fmt.Println(name)
		if resource.Update == nil {
			continue
		}
		if checkForProblems(name, "", resource) {
			fmt.Printf("There is a problem with %s\n", name)
		}
	}
	fmt.Println(count, len(count))
}

func checkForProblems(base string, path string, resource *schema.Resource) bool {

	for property, item := range resource.Schema {
		// if item.Type == schema.TypeSet {
		// fmt.Printf("%s[%s] is a list of %T\n", resourcePath, property, item.Elem)

		switch item.Elem.(type) {
		case *schema.Resource:
			if item.Optional && item.Computed && item.Removed == "" {
				count[base] = true
				fmt.Printf("%s, %s, %s, %d\n", base, path+"."+property, item.Type, item.MaxItems)
			}
			// fmt.Printf("%s is Elem/Resource, type is %s\n", property, item.Type)
			checkForProblems(base, path+"."+property, item.Elem.(*schema.Resource))
			break
		}

	}

	return false
}
