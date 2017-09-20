package vsphere

import (
	"fmt"

	"github.com/hashicorp/terraform/helper/schema"
)

func dataSourceVSphereDistributedVirtualSwitch() *schema.Resource {
	return &schema.Resource{
		Read: dataSourceVSphereDistributedVirtualSwitchRead,

		Schema: map[string]*schema.Schema{
			"name": &schema.Schema{
				Type:        schema.TypeString,
				Description: "The name of the distributed virtual switch. This can be a name or path.",
				Required:    true,
			},
			"datacenter_id": &schema.Schema{
				Type:        schema.TypeString,
				Description: "The managed object ID of the datacenter to look for the host in.",
				Required:    true,
			},
		},
	}
}

func dataSourceVSphereDistributedVirtualSwitchRead(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*govmomi.Client)
	name := d.Get("name").(string)
	dcID := d.Get("datacenter_id").(string)
	dvs, err := dvsFromName(client, dcID, name)
	if err != nil {
		return fmt.Errorf("error fetching distributed virtual switch: %s", err)
	}

	id := dvs.Uuid
	d.SetId(id)

	return nil
}
