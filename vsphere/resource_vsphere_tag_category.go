package vsphere

import (
	"context"
	"errors"
	"fmt"

	"github.com/hashicorp/terraform/helper/schema"
	"github.com/hashicorp/terraform/helper/validation"
	"github.com/vmware/vic/pkg/vsphere/tags"
)

// resourceVSphereTagCategoryImportErrMultiple is an error message format for a
// tag category import that returned multiple results. This is a bug and needs
// to be reported so we can adjust the API.
const resourceVSphereTagCategoryImportErrMultiple = `
Category name %q returned multiple results!

This is a bug - please report it at:
https://github.com/terraform-providers/terraform-provider-vsphere/issues

This version of the provider requires unique category names. To work around
this issue, please rename your category to a name unique within your vCenter
system.
`

// A list of valid types for cardinality and associable types are below. The
// latter is more significant, even though they are not used in the resource
// itself, to ensure all associable types are properly documented so we can
// reference it later, in addition to providing the list for future validation
// if we add the ability to validate lists and sets in core.
const (
	vSphereTagCategoryCardinalitySingle   = "SINGLE"
	vSphereTagCategoryCardinalityMultiple = "MULTIPLE"

	vSphereTagCategoryAssociableTypeFolder                         = "Folder"
	vSphereTagCategoryAssociableTypeClusterComputeResource         = "ClusterComputeResource"
	vSphereTagCategoryAssociableTypeDatacenter                     = "Datacenter"
	vSphereTagCategoryAssociableTypeDatastore                      = "Datastore"
	vSphereTagCategoryAssociableTypeStoragePod                     = "StoragePod"
	vSphereTagCategoryAssociableTypeDistributedVirtualPortgroup    = "DistributedVirtualPortgroup"
	vSphereTagCategoryAssociableTypeDistributedVirtualSwitch       = "DistributedVirtualSwitch"
	vSphereTagCategoryAssociableTypeVmwareDistributedVirtualSwitch = "VmwareDistributedVirtualSwitch"
	vSphereTagCategoryAssociableTypeHostSystem                     = "HostSystem"
	vSphereTagCategoryAssociableTypeContentLibrary                 = "com.vmware.content.Library"
	vSphereTagCategoryAssociableTypeContentLibraryItem             = "com.vmware.content.library.Item"
	vSphereTagCategoryAssociableTypeHostNetwork                    = "HostNetwork"
	vSphereTagCategoryAssociableTypeNetwork                        = "Network"
	vSphereTagCategoryAssociableTypeOpaqueNetwork                  = "OpaqueNetwork"
	vSphereTagCategoryAssociableTypeResourcePool                   = "ResourcePool"
	vSphereTagCategoryAssociableTypeVirtualApp                     = "VirtualApp"
	vSphereTagCategoryAssociableTypeVirtualMachine                 = "VirtualMachine"

	vSphereTagCategoryAssociableTypeAll = "All"
)

// The following groups are type groups that are associated with the same type
// selection in the vSphere Client tag category UI.
var (
	// vSphereTagCategoryAssociableTypesForDistributedVirtualSwitch represents
	// types for virtual switches.
	vSphereTagCategoryAssociableTypesForDistributedVirtualSwitch = []string{
		vSphereTagCategoryAssociableTypeDistributedVirtualSwitch,
		vSphereTagCategoryAssociableTypeVmwareDistributedVirtualSwitch,
	}

	// vSphereTagCategoryAssociableTypesForNetwork represents the types for
	// networks.
	vSphereTagCategoryAssociableTypesForNetwork = []string{
		vSphereTagCategoryAssociableTypeHostNetwork,
		vSphereTagCategoryAssociableTypeNetwork,
		vSphereTagCategoryAssociableTypeOpaqueNetwork,
	}
)

func resourceVSphereTagCategory() *schema.Resource {
	return &schema.Resource{
		Create: resourceVSphereTagCategoryCreate,
		Read:   resourceVSphereTagCategoryRead,
		Update: resourceVSphereTagCategoryUpdate,
		Delete: resourceVSphereTagCategoryDelete,
		Importer: &schema.ResourceImporter{
			State: resourceVSphereTagCategoryImport,
		},

		Schema: map[string]*schema.Schema{
			"name": {
				Type:        schema.TypeString,
				Description: "The display name of the category.",
				Required:    true,
			},
			"description": {
				Type:        schema.TypeString,
				Description: "The description of the category.",
				Optional:    true,
			},
			"cardinality": {
				Type:        schema.TypeString,
				Description: "The associated cardinality of the category. Can be one of SINGLE (object can only be assigned one tag in this category) or MULTIPLE (object can be assigned multiple tags in this category).",
				ForceNew:    true,
				Required:    true,
				ValidateFunc: validation.StringInSlice(
					[]string{
						vSphereTagCategoryCardinalitySingle,
						vSphereTagCategoryCardinalityMultiple,
					},
					false,
				),
			},
			"associable_types": {
				Type:        schema.TypeSet,
				Description: "Object types to which this category's tags can be attached.",
				Elem:        &schema.Schema{Type: schema.TypeString},
				Required:    true,
			},
		},
	}
}

func resourceVSphereTagCategoryCreate(d *schema.ResourceData, meta interface{}) error {
	client, err := meta.(*VSphereClient).TagsClient()
	if err != nil {
		return err
	}

	spec := &tags.CategoryCreateSpec{
		CreateSpec: tags.CategoryCreate{
			AssociableTypes: sliceInterfacesToStrings(d.Get("associable_types").(*schema.Set).List()),
			Cardinality:     d.Get("cardinality").(string),
			Description:     d.Get("description").(string),
			Name:            d.Get("name").(string),
		},
	}
	ctx, cancel := context.WithTimeout(context.Background(), defaultAPITimeout)
	defer cancel()
	id, err := client.CreateCategory(ctx, spec)
	if err != nil {
		return fmt.Errorf("could not create category: %s", err)
	}
	if id == nil {
		return errors.New("no ID was returned")
	}
	d.SetId(*id)
	return resourceVSphereTagCategoryRead(d, meta)
}

func resourceVSphereTagCategoryRead(d *schema.ResourceData, meta interface{}) error {
	client, err := meta.(*VSphereClient).TagsClient()
	if err != nil {
		return err
	}

	id := d.Id()

	ctx, cancel := context.WithTimeout(context.Background(), defaultAPITimeout)
	defer cancel()
	category, err := client.GetCategory(ctx, id)
	if err != nil {
		return fmt.Errorf("could not locate category with id %q: %s", id, err)
	}
	d.Set("name", category.Name)
	d.Set("description", category.Description)
	d.Set("cardinality", category.Cardinality)

	if err := d.Set("associable_types", category.AssociableTypes); err != nil {
		return fmt.Errorf("could not set associable type data for category: %s", err)
	}

	return nil
}

func resourceVSphereTagCategoryUpdate(d *schema.ResourceData, meta interface{}) error {
	client, err := meta.(*VSphereClient).TagsClient()
	if err != nil {
		return err
	}

	// Block the update if the user has removed types
	oldts, newts := d.GetChange("associable_types")
	for _, v1 := range oldts.(*schema.Set).List() {
		var found bool
		for _, v2 := range newts.(*schema.Set).List() {
			if v1 == v2 {
				found = true
			}
		}
		if !found {
			return fmt.Errorf("cannot remove type %q (removal of associable types is not supported)", v1)
		}
	}

	id := d.Id()
	spec := &tags.CategoryUpdateSpec{
		UpdateSpec: tags.CategoryUpdate{
			AssociableTypes: sliceInterfacesToStrings(d.Get("associable_types").(*schema.Set).List()),
			Cardinality:     d.Get("cardinality").(string),
			Description:     d.Get("description").(string),
			Name:            d.Get("name").(string),
		},
	}
	ctx, cancel := context.WithTimeout(context.Background(), defaultAPITimeout)
	defer cancel()
	err = client.UpdateCategory(ctx, id, spec)
	if err != nil {
		return fmt.Errorf("could not update category with id %q: %s", id, err)
	}
	return resourceVSphereTagCategoryRead(d, meta)
}

func resourceVSphereTagCategoryDelete(d *schema.ResourceData, meta interface{}) error {
	client, err := meta.(*VSphereClient).TagsClient()
	if err != nil {
		return err
	}

	id := d.Id()

	ctx, cancel := context.WithTimeout(context.Background(), defaultAPITimeout)
	defer cancel()
	err = client.DeleteCategory(ctx, id)
	if err != nil {
		return fmt.Errorf("could not delete category with id %q: %s", id, err)
	}
	return nil
}

func resourceVSphereTagCategoryImport(d *schema.ResourceData, meta interface{}) ([]*schema.ResourceData, error) {
	// Although GetCategoriesByName does not seem to think that tag categories
	// are unique, empirical observation via the console and API show that they
	// are. Hence we can just import by the tag name. If for some reason the
	// returned results includes more than one ID, we give an error.
	client, err := meta.(*VSphereClient).TagsClient()
	if err != nil {
		return nil, err
	}

	name := d.Id()

	ctx, cancel := context.WithTimeout(context.Background(), defaultAPITimeout)
	defer cancel()
	cats, err := client.GetCategoriesByName(ctx, name)
	if err != nil {
		return nil, fmt.Errorf("could not get category for name %q: %s", name, err)
	}

	if len(cats) < 1 {
		return nil, fmt.Errorf("category name %q not found", name)
	}
	if len(cats) > 1 {
		// This should not happen but guarding just in case it does. This is a bug
		// and needs to be reported.
		return nil, fmt.Errorf(resourceVSphereTagCategoryImportErrMultiple, name)
	}

	d.SetId(cats[0].ID)
	return []*schema.ResourceData{d}, nil
}
