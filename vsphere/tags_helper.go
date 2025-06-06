// © Broadcom. All Rights Reserved.
// The term "Broadcom" refers to Broadcom Inc. and/or its subsidiaries.
// SPDX-License-Identifier: MPL-2.0

package vsphere

import (
	"context"
	"fmt"
	"log"
	"strings"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/vmware/govmomi"
	"github.com/vmware/govmomi/object"
	"github.com/vmware/govmomi/vapi/tags"
	"github.com/vmware/terraform-provider-vsphere/vsphere/internal/helper/structure"
	"github.com/vmware/terraform-provider-vsphere/vsphere/internal/helper/viapi"
)

// A list of valid object types for tagging are below. These are referenced by
// various helpers and tests.
const (
	vSphereTagTypeDatastore      = "Datastore"
	vSphereTagTypeVirtualMachine = "VirtualMachine"
)

var vSphereTagTypes = []string{
	"Folder",
	"ClusterComputeResource",
	"Datacenter",
	vSphereTagTypeDatastore,
	"StoragePod",
	"DistributedVirtualPortgroup",
	"DistributedVirtualSwitch",
	"VmwareDistributedVirtualSwitch",
	"HostSystem",
	"com.vmware.content.Library",
	"com.vmware.content.library.Item",
	"HostNetwork",
	"Network",
	"OpaqueNetwork",
	"ResourcePool",
	"VirtualApp",
	vSphereTagTypeVirtualMachine,
}

// vSphereTagAttributeKey is the string key that should always be used as the
// argument to pass tags in to. Various resource tag helpers will depend on
// this value being consistent across resources.
//
// When adding tags to a resource schema, the easiest way to do that (for now)
// will be to use the following line:
//
//	vSphereTagAttributeKey: tagsSchema(),
//
// This will ensure that the correct key and schema is used across all resources.
const vSphereTagAttributeKey = "tags"

// tagsMinVersion is the minimum vSphere version required for tags.
var tagsMinVersion = viapi.VSphereVersion{
	Product: "VMware vCenter Server",
	Major:   6,
	Minor:   0,
	Patch:   0,
	Build:   2559268,
}

// isEligibleRestEndpoint is a meta-validation that is used on login to see if
// the connected endpoint supports the CIS REST API, which we use for tags.
func isEligibleRestEndpoint(client *govmomi.Client) bool {
	if err := viapi.ValidateVirtualCenter(client); err != nil {
		return false
	}
	clientVer := viapi.ParseVersionFromClient(client)
	if !clientVer.ProductEqual(tagsMinVersion) || clientVer.Older(tagsMinVersion) {
		return false
	}
	return true
}

// isEligiblePBMEndpoint is a meta-validation that is used on login to see if
// the connected endpoint supports the CIS REST API, which we use for tags.
func isEligiblePBMEndpoint(client *govmomi.Client) bool {
	if err := viapi.ValidateVirtualCenter(client); err != nil {
		return false
	}
	return true
}

// isEligibleVSANEndpoint is a meta-validation that is used on login to see if
// the connected endpoint supports the CIS REST API, which we use for tags.
func isEligibleVSANEndpoint(client *govmomi.Client) bool {
	if err := viapi.ValidateVirtualCenter(client); err != nil {
		return false
	}
	return true
}

// tagCategoryByName locates a tag category by name. It's used by the
// vsphere_tag_category data source, and the resource importer.
func tagCategoryByName(tm *tags.Manager, name string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), defaultAPITimeout)
	defer cancel()
	allCats, err := tm.GetCategories(ctx)
	if err != nil {
		return "", fmt.Errorf("could not get category for name %q: %s", name, err)
	}

	var cats []*tags.Category
	for i, cat := range allCats {
		if cat.Name == name {
			cats = append(cats, &allCats[i])
		}
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
		return "", fmt.Errorf("tag category name %q returned multiple results; unique tag category names are required", name)
	}

	return cats[0].ID, nil
}

// tagByName locates a tag by it supplied name and category ID. Use
// tagCategoryByName to get the tag category ID if require the category ID as
// well.
func tagByName(tm *tags.Manager, name, categoryID string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), defaultAPITimeout)
	defer cancel()
	allTags, err := tm.GetTagsForCategory(ctx, categoryID)
	var tagList []*tags.Tag
	if err != nil {
		return "", fmt.Errorf("could not get tag for name %q: %s", name, err)
	}
	for i, tag := range allTags {
		if tag.Name == name {
			tagList = append(tagList, &allTags[i])
		}
	}

	if len(tagList) < 1 {
		return "", fmt.Errorf("tag name %q not found in category ID %q", name, categoryID)
	}
	if len(tagList) > 1 {
		// This situation is very similar to the one in tagCategoryByName. The API
		// docs even say that tagList need to be unique in categories, yet
		// GetTagByNameForCategory still returns multiple results.
		return "", fmt.Errorf("tag name %q returned multiple results; unique tag names are required", name)
	}

	return tagList[0].ID, nil
}

// tagsSchema returns the schema for the tags configuration attribute for each
// resource that needs it.
//
// The key is usually "tags" and should be a list of tag IDs to associate with
// this resource.
func tagsSchema() *schema.Schema {
	return &schema.Schema{
		Type:        schema.TypeSet,
		Description: "A list of tag IDs to apply to this object.",
		Optional:    true,
		Elem:        &schema.Schema{Type: schema.TypeString},
	}
}

// readTagsForResource reads the tags for a given reference and saves the list
// in the supplied ResourceData. It returns an error if there was an issue
// reading the tags.
func readTagsForResource(tm *tags.Manager, obj object.Reference, d *schema.ResourceData) error {
	log.Printf("[DEBUG] Reading tags for object %q", obj.Reference().Value)
	ctx, cancel := context.WithTimeout(context.Background(), defaultAPITimeout)
	defer cancel()

	ids, err := tm.ListAttachedTags(ctx, obj)
	log.Printf("[DEBUG] Tags for object %q: %s", obj.Reference().Value, strings.Join(ids, ","))
	if err != nil {
		return err
	}
	if err := d.Set(vSphereTagAttributeKey, ids); err != nil {
		return fmt.Errorf("error saving tag IDs to resource data: %s", err)
	}
	return nil
}

// tagDiffProcessor is an object that wraps the "complex" adding and removal of
// tags from an object.
type tagDiffProcessor struct {
	// The client connection.
	manager *tags.Manager

	// The object that is the subject of the tag addition and removal operations.
	subject object.Reference

	// A list of old (current) tags attached to the subject.
	oldTagIDs []string

	// The list of tags that should be attached to the subject.
	newTagIDs []string
}

// diffOldNew returns any elements of old that were missing in new.
func (p *tagDiffProcessor) diffOldNew() []string {
	return p.diff(p.oldTagIDs, p.newTagIDs)
}

// diffNewOld returns any elements of new that were missing in old.
func (p *tagDiffProcessor) diffNewOld() []string {
	return p.diff(p.newTagIDs, p.oldTagIDs)
}

// diff is what diffOldNew and diffNewOld hand off to.
func (p *tagDiffProcessor) diff(a, b []string) []string {
	var found bool
	c := make([]string, 0)
	for _, v1 := range a {
		for _, v2 := range b {
			if v1 == v2 {
				found = true
			}
		}
		if !found {
			c = append(c, v1)
		}
		found = false
	}
	return c
}

// processAttachOperations processes all pending attach operations by diffing old
// and new and adding any IDs that were not found in old.
func (p *tagDiffProcessor) processAttachOperations() error {
	tagIDs := p.diffNewOld()
	if len(tagIDs) < 1 {
		return nil
	}
	for _, tagID := range tagIDs {
		ctx, cancel := context.WithTimeout(context.Background(), defaultAPITimeout)
		log.Printf("[DEBUG] Attaching tag %q for object %q", tagID, p.subject.Reference().Value)
		err := p.manager.AttachTag(ctx, tagID, p.subject)
		cancel()
		if err != nil {
			return err
		}
	}
	return nil
}

// processDetachOperations processes all pending detach operations by diffing
// new and old, and removing any IDs that were not found in new.
func (p *tagDiffProcessor) processDetachOperations() error {
	tagIDs := p.diffOldNew()
	if len(tagIDs) < 1 {
		return nil
	}
	for _, tagID := range tagIDs {
		ctx, cancel := context.WithTimeout(context.Background(), defaultAPITimeout)
		log.Printf("[DEBUG] Detaching tag %q for object %q", tagID, p.subject.Reference().Value)
		err := p.manager.DetachTag(ctx, tagID, p.subject)
		cancel()
		if err != nil {
			return err
		}
	}
	return nil
}

// tagsManagerIfDefined goes through the client validation process and returns
// the tags manager only if there are tags defined in the supplied ResourceData.
//
// This should be used to fetch the tagging manager on resources that
// support tags, usually closer to the beginning of a CRUD function to check to
// make sure it's worth proceeding with most of the operation. The returned
// client should be checked for nil before passing it to processTagDiff.
func tagsManagerIfDefined(d *schema.ResourceData, meta interface{}) (*tags.Manager, error) {
	old, newValue := d.GetChange(vSphereTagAttributeKey)
	if len(old.(*schema.Set).List()) > 0 || len(newValue.(*schema.Set).List()) > 0 {
		log.Printf("[DEBUG] tagsClientIfDefined: Loading tagging client")
		tm, err := meta.(*Client).TagsManager()
		if err != nil {
			return nil, err
		}
		return tm, nil
	}
	log.Printf("[DEBUG] tagsClientIfDefined: No tags configured, skipping loading of tagging client")
	return nil, nil
}

// processTagDiff wraps the whole tag diffing operation into a nice clean
// function that resources can use.
func processTagDiff(tm *tags.Manager, d *schema.ResourceData, obj object.Reference) error {
	log.Printf("[DEBUG] Processing tags for object %q", obj.Reference().Value)
	old, newValue := d.GetChange(vSphereTagAttributeKey)
	tdp := &tagDiffProcessor{
		manager:   tm,
		subject:   obj,
		oldTagIDs: structure.SliceInterfacesToStrings(old.(*schema.Set).List()),
		newTagIDs: structure.SliceInterfacesToStrings(newValue.(*schema.Set).List()),
	}
	if err := tdp.processDetachOperations(); err != nil {
		return fmt.Errorf("error detaching tags to object ID %q: %s", obj.Reference().Value, err)
	}
	if err := tdp.processAttachOperations(); err != nil {
		return fmt.Errorf("error attaching tags to object ID %q: %s", obj.Reference().Value, err)
	}
	return nil
}
