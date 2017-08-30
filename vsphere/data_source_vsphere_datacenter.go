package vsphere

import (
	"fmt"

	"github.com/hashicorp/terraform/helper/schema"
	"github.com/vmware/govmomi"
)

func dataSourceVSphereDatacenter() *schema.Resource {
	return &schema.Resource{
		Read: dataSourceVSphereDatacenterRead,

		Schema: map[string]*schema.Schema{
			"name": &schema.Schema{
				Type: schema.TypeString,
				Description: "The name of the datacenter. This can be a name or path.	If not provided, the default datacenter is used.",
				Optional: true,
			},
		},
	}
}

func dataSourceVSphereDatacenterRead(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*govmomi.Client)
	datacenter := d.Get("name").(string)
	dc, err := getDatacenter(client, datacenter)
	if err != nil {
		return fmt.Errorf("error fetching datacenter: %s", err)
	}
	id := dc.Reference().Value
	d.SetId(id)

	return nil
}
