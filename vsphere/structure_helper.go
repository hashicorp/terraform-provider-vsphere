package vsphere

import (
	"fmt"

	"github.com/hashicorp/terraform/helper/schema"
)

// sliceInterfacesToStrings converts an interface slice to a string slice. The
// function does not attempt to do any sanity checking and will panic if one of
// the items in the slice is not a string.
func sliceInterfacesToStrings(s []interface{}) []string {
	var d []string
	for _, v := range s {
		d = append(d, v.(string))
	}
	return d
}

// sliceStringsToInterfaces converts a string slice to an interface slice.
func sliceStringsToInterfaces(s []string) []interface{} {
	var d []interface{}
	for _, v := range s {
		d = append(d, v)
	}
	return d
}

// mergeSchema merges the map[string]*schema.Schema from src into dst. Safety
// against conflicts is enforced by panicing.
func mergeSchema(dst, src map[string]*schema.Schema) {
	for k, v := range src {
		if _, ok := dst[k]; ok {
			panic(fmt.Errorf("conflicting schema key: %s", k))
		}
		dst[k] = v
	}
}

// boolPtr makes a *bool out of the value passed in through v.
//
// vSphere uses nil values in bools to omit values in the SOAP XML request, and
// helps denote inheritance in certain cases.
func boolPtr(v bool) *bool {
	return &v
}
