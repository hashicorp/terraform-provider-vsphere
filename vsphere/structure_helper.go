package vsphere

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
