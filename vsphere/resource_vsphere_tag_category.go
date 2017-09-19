package vsphere

import (
	"context"
	"errors"
	"fmt"

	"github.com/hashicorp/terraform/helper/schema"
	"github.com/hashicorp/terraform/helper/validation"
	"github.com/vmware/vic/pkg/vsphere/tags"
)

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
	client, err := meta.(*VSphereClient).TagsClient()
	if err != nil {
		return nil, err
	}

	id, err := tagCategoryByName(client, d.Id())
	if err != nil {
		return nil, err
	}

	d.SetId(id)
	return []*schema.ResourceData{d}, nil
}
