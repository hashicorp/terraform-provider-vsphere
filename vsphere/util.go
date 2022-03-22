package vsphere

func stringInSlice(s string, list []string) bool {
	for _, i := range list {
		if i == s {
			return true
		}
	}
	return false
}
