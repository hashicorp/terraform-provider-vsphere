package vsphere

import (
	"fmt"

	"github.com/hashicorp/terraform/helper/schema"
	"github.com/terraform-providers/terraform-provider-vsphere/vsphere/internal/helper/datacenter"
	"github.com/terraform-providers/terraform-provider-vsphere/vsphere/internal/helper/hostsystem"
)

func dataSourceVSphereHost() *schema.Resource {
	return &schema.Resource{
		Read: dataSourceVSphereHostRead,

		Schema: map[string]*schema.Schema{
			"name": &schema.Schema{
				Type: schema.TypeString,
				Description: "The name of the host. This can be a name or path.	If not provided, the default host is used.",
				Optional: true,
			},
			"datacenter_id": &schema.Schema{
				Type:        schema.TypeString,
				Description: "The managed object ID of the datacenter to look for the host in.",
				Required:    true,
			},
		},
	}
}

func dataSourceVSphereHostRead(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*VSphereClient).vimClient
	name := d.Get("name").(string)
	dcID := d.Get("datacenter_id").(string)
	dc, err := datacenter.FromID(client, dcID)
	if err != nil {
		return fmt.Errorf("error fetching datacenter: %s", err)
	}
	hs, err := hostsystem.SystemOrDefault(client, name, dc)
	if err != nil {
		return fmt.Errorf("error fetching host: %s", err)
	}

	id := hs.Reference().Value
	d.SetId(id)

	return nil
}
