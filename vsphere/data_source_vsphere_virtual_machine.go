package vsphere

import (
	"fmt"
	"log"

	"github.com/hashicorp/terraform/helper/schema"
	"github.com/terraform-providers/terraform-provider-vsphere/vsphere/internal/helper/virtualmachine"
	"github.com/terraform-providers/terraform-provider-vsphere/vsphere/internal/virtualdevice"
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
			"disk_sizes": {
				Type:        schema.TypeList,
				Description: "The sizes of the disks on this virtual machine, sorted by bus and unit number.",
				Computed:    true,
				Elem:        &schema.Schema{Type: schema.TypeInt},
			},
		},
	}
}

func dataSourceVSphereVirtualMachineRead(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*VSphereClient).vimClient

	name := d.Get("name").(string)
	log.Printf("[DEBUG] Looking for VM or template by name/path %q", name)
	var dc *object.Datacenter
	if dcID, ok := d.GetOk("datacenter_id"); ok {
		var err error
		dc, err = datacenterFromID(client, dcID.(string))
		if err != nil {
			return fmt.Errorf("cannot locate datacenter: %s", err)
		}
		log.Printf("[DEBUG] Datacenter for VM/template search: %s", dc.InventoryPath)
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
	sizes, err := virtualdevice.ReadDiskSizes(object.VirtualDeviceList(props.Config.Hardware.Device))
	if err != nil {
		return fmt.Errorf("error reading disk sizes: %s", err)
	}
	if d.Set("disk_sizes", sizes); err != nil {
		return fmt.Errorf("error setting disk sizes: %s", err)
	}
	log.Printf("[DEBUG] VM search for %q completed successfully (UUID %q)", name, props.Config.Uuid)
	return nil
}
