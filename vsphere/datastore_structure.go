package vsphere

import (
	"github.com/hashicorp/terraform/helper/schema"
	"github.com/terraform-providers/terraform-provider-vsphere/vsphere/internal/helper/structure"
	"github.com/vmware/govmomi/vim25/types"
)

// schemaDatastoreSummary returns schema items for resources that
// need to work with a DatastoreSummary.
func schemaDatastoreSummary() map[string]*schema.Schema {
	return map[string]*schema.Schema{
		// Note that the following fields are not represented in the schema here:
		// * Name (more than likely the ID attribute and will be represented in
		// resource schema)
		// * Type (redundant attribute as the datastore type will be represented by
		// the resource)
		"accessible": &schema.Schema{
			Type:        schema.TypeBool,
			Description: "The connectivity status of the datastore. If this is false, some other computed attributes may be out of date.",
			Computed:    true,
		},
		"capacity": &schema.Schema{
			Type:        schema.TypeInt,
			Description: "Maximum capacity of the datastore, in MB.",
			Computed:    true,
		},
		"free_space": &schema.Schema{
			Type:        schema.TypeInt,
			Description: "Available space of this datastore, in MB.",
			Computed:    true,
		},
		"maintenance_mode": &schema.Schema{
			Type:        schema.TypeString,
			Description: "The current maintenance mode state of the datastore.",
			Computed:    true,
		},
		"multiple_host_access": &schema.Schema{
			Type:        schema.TypeBool,
			Description: "If true, more than one host in the datacenter has been configured with access to the datastore.",
			Computed:    true,
		},
		"uncommitted_space": &schema.Schema{
			Type:        schema.TypeInt,
			Description: "Total additional storage space, in MB, potentially used by all virtual machines on this datastore.",
			Computed:    true,
		},
		"url": &schema.Schema{
			Type:        schema.TypeString,
			Description: "The unique locator for the datastore.",
			Computed:    true,
		},
	}
}

// flattenDatastoreSummary reads various fields from a DatastoreSummary into
// the passed in ResourceData.
func flattenDatastoreSummary(d *schema.ResourceData, obj *types.DatastoreSummary) error {
	d.Set("accessible", obj.Accessible)
	d.Set("capacity", structure.ByteToMB(obj.Capacity))
	d.Set("free_space", structure.ByteToMB(obj.FreeSpace))
	d.Set("maintenance_mode", obj.MaintenanceMode)
	d.Set("multiple_host_access", obj.MultipleHostAccess)
	d.Set("uncommitted_space", structure.ByteToMB(obj.Uncommitted))
	d.Set("url", obj.Url)

	// Set the name attribute off of the name here - since we do not track this
	// here we check for errors
	if err := d.Set("name", obj.Name); err != nil {
		return err
	}
	return nil
}
