package vsphere

import (
	"context"
	"fmt"

	"github.com/vmware/govmomi"
	"github.com/vmware/vic/pkg/vsphere/tags"
)

// vSphereTagCategorySearchErrMultiple is an error message format for a tag
// category search that returned multiple results. This is a bug and needs to
// be reported so we can adjust the API.
const vSphereTagCategorySearchErrMultiple = `
Category name %q returned multiple results!

This is a bug - please report it at:
https://github.com/terraform-providers/terraform-provider-vsphere/issues

This version of the provider requires unique category names. To work around
this issue, please use a category name unique within your vCenter system.
`

// vSphereTagSearchErrMultiple is an error message format for a tag search that
// returned multiple results. This is a bug and needs to be reported so we can
// adjust the API.
const vSphereTagSearchErrMultiple = `
Tag name %q returned multiple results!

This is a bug - please report it at:
https://github.com/terraform-providers/terraform-provider-vsphere/issues

This version of the provider requires unique tag names. To work around
this issue, please use a tag name unique within your vCenter system.
`

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
	clientVer := parseVersionFromClient(client)
	if !clientVer.ProductEqual(tagsMinVersion) || clientVer.Older(tagsMinVersion) {
		return false
	}
	return true
}

// tagCategoryByName locates a tag category by name. It's used by the
// vsphere_tag_category data source, and the resource importer.
func tagCategoryByName(client *tags.RestClient, name string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), defaultAPITimeout)
	defer cancel()
	cats, err := client.GetCategoriesByName(ctx, name)
	if err != nil {
		return "", fmt.Errorf("could not get category for name %q: %s", name, err)
	}

	if len(cats) < 1 {
		return "", fmt.Errorf("category name %q not found", name)
	}
	if len(cats) > 1 {
		// Although GetCategoriesByName does not seem to think that tag categories
		// are unique, empirical observation via the console and API show that they
		// are. If for some reason the returned results includes more than one ID,
		// we give an error, indicating that this is a bug and the user should
		// submit an issue.
		return "", fmt.Errorf(vSphereTagCategorySearchErrMultiple, name)
	}

	return cats[0].ID, nil
}

// tagByName locates a tag by it supplied name and category ID. Use
// tagCategoryByName to get the tag category ID if require the category ID as
// well.
func tagByName(client *tags.RestClient, name, categoryID string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), defaultAPITimeout)
	defer cancel()
	tags, err := client.GetTagByNameForCategory(ctx, name, categoryID)
	if err != nil {
		return "", fmt.Errorf("could not get tag for name %q: %s", name, err)
	}

	if len(tags) < 1 {
		return "", fmt.Errorf("tag name %q not found in category ID %q", name, categoryID)
	}
	if len(tags) > 1 {
		// This situation is very similar to the one in tagCategoryByName. The API
		// docs even say that tags need to be unique in categories, yet
		// GetTagByNameForCategory still returns multiple results.
		return "", fmt.Errorf(vSphereTagSearchErrMultiple, name)
	}

	return tags[0].ID, nil
}
