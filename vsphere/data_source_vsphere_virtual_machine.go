package vsphere

import (
	"fmt"

	"github.com/hashicorp/terraform/helper/schema"
	"github.com/terraform-providers/terraform-provider-vsphere/vsphere/internal/helper/virtualmachine"
	"github.com/vmware/govmomi/object"
)

func dataSourceVSphereVirtualMachine() *schema.Resource {
	return &schema.Resource{
		Read: dataSourceVSphereVirtualMachineRead,

		Schema: map[string]*schema.Schema{
			"name": {
				Type:        schema.TypeString,
				Description: "The name or path of the virtual machine.",
				Required:    true,
			},
			"datacenter_id": {
				Type:        schema.TypeString,
				Description: "The managed object ID of the datacenter the virtual machine is in. This is not required when using ESXi directly, or if there is only one datacenter in your infrastructure.",
				Optional:    true,
			},
			"guest_id": {
				Type:        schema.TypeString,
				Description: "The guest ID of the virtual machine.",
				Computed:    true,
			},
			"alternate_guest_name": {
				Type:        schema.TypeString,
				Description: "The alternate guest name of the virtual machine when guest_id is a non-specific operating system, like otherGuest.",
				Computed:    true,
			},
		},
	}
}

func dataSourceVSphereVirtualMachineRead(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*VSphereClient).vimClient

	name := d.Get("name").(string)
	var dc *object.Datacenter
	if dcID, ok := d.GetOk("datacenter_id"); ok {
		var err error
		dc, err = datacenterFromID(client, dcID.(string))
		if err != nil {
			return fmt.Errorf("cannot locate datacenter: %s", err)
		}
	}
	vm, err := virtualmachine.FromPath(client, name, dc)
	if err != nil {
		return fmt.Errorf("error fetching virtual machine: %s", err)
	}
	props, err := virtualmachine.Properties(vm)
	if err != nil {
		return fmt.Errorf("error fetching virtual machine properties: %s", err)
	}

	d.SetId(props.Config.Uuid)
	d.Set("guest_id", props.Config.GuestId)
	d.Set("alternate_guest_name", props.Config.AlternateGuestName)
	return nil
}
