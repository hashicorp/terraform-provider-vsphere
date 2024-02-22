package vsphere

import (
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-provider-vsphere/vsphere/internal/helper/structure"
)

func resourceVSphereRemoveHostVirtualSwitch() *schema.Resource {
	s := map[string]*schema.Schema{
		"vswitch_id": {
			Type:        schema.TypeString,
			Description: "The ID of the virtual switch.",
			Required:    true,
		},
		"name": {
			Type:        schema.TypeString,
			Description: "The name of the virtual switch.",
			Computed:    true,
		},
		"host_system_id": {
			Type:        schema.TypeString,
			Description: "The managed object ID of the host to set the virtual switch up on.",
			Computed:    true,
		},
	}
	structure.MergeSchema(s, schemaHostVirtualSwitchSpec())

	// Transform any necessary fields in the schema that need to be updated
	// specifically for this resource.
	s["active_nics"].Computed = true
	s["standby_nics"].Computed = true

	s["teaming_policy"].Computed = true
	s["check_beacon"].Computed = true
	s["notify_switches"].Computed = true
	s["failback"].Computed = true

	s["allow_promiscuous"].Computed = true
	s["allow_forged_transmits"].Computed = true
	s["allow_mac_changes"].Computed = true

	s["shaping_enabled"].Computed = true

	s["network_adapters"].Required = false
	s["network_adapters"].Computed = true

	return &schema.Resource{
		Create:        resourceVSphereRemoveHostVirtualSwitchCreate,
		Read:          resourceVSphereRemovedHostVirtualSwitchRead,
		Update:        resourceVSphereHostVirtualSwitchUpdate,
		Delete:        resourceVSphereRemovedHostVirtualSwitchDelete,
		CustomizeDiff: resourceVSphereHostVirtualSwitchCustomizeDiff,
		Importer: &schema.ResourceImporter{
			State: resourceVSphereHostVirtualSwitchImport,
		},
		Schema: s,
	}
}

func resourceVSphereRemoveHostVirtualSwitchCreate(d *schema.ResourceData, meta interface{}) error {
	id := d.Get("vswitch_id").(string)
	d.SetId(id)
	_, err := resourceVSphereHostVirtualSwitchImport(d, meta)
	if err != nil {
		return err
	}
	err = resourceVSphereHostVirtualSwitchRead(d, meta)
	if err != nil {
		return err
	}
	err = resourceVSphereHostVirtualSwitchDelete(d, meta)
	if err != nil {
		return err
	}
	d.SetId(id)
	return nil
}

func resourceVSphereRemovedHostVirtualSwitchRead(d *schema.ResourceData, meta interface{}) error {
	id := d.Get("vswitch_id").(string)
	d.SetId(id)
	return nil
}

func resourceVSphereRemovedHostVirtualSwitchDelete(d *schema.ResourceData, meta interface{}) error {
	d.SetId("")
	return nil
}
