package vsphere

import (
	"github.com/vmware/govmomi"
)

// tagsMinVersion is the minimum vSphere version required for tags.
var tagsMinVersion = vSphereVersion{
	product: "VMware vCenter Server",
	major:   6,
	minor:   0,
	patch:   0,
	build:   2559268,
}

// isEligibleTagEndpoint is a meta-validation that is used on login to see if
// the connected endpoint supports the CIS REST API, which we use for tags.
func isEligibleTagEndpoint(client *govmomi.Client) bool {
	if err := validateVirtualCenter(client); err != nil {
		return false
	}
	if parseVersionFromClient(client).Older(tagsMinVersion) {
		return false
	}
	return true
}
