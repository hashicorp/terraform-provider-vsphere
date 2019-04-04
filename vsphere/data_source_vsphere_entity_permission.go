package vsphere

import (
	"log"

	"github.com/hashicorp/terraform/helper/schema"
	"github.com/terraform-providers/terraform-provider-vsphere/vsphere/internal/helper/permission"
)

func dataSourceVSphereEntityPermission() *schema.Resource {
	return &schema.Resource{
		Read: dataSourceVSphereEntityPermissionRead,

		Schema: map[string]*schema.Schema{
			"principal": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},
			"entity_id": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},
			"entity_type": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},
			"role_id": &schema.Schema{
				Type:     schema.TypeInt,
				Computed: true,
			},
		},
	}
}

func dataSourceVSphereEntityPermissionRead(d *schema.ResourceData, meta interface{}) error {
	log.Printf("[DEBUG] Reading entity permission %q", d.Id())
	client := meta.(*VSphereClient).vimClient
	id := permission.ConcatID(d.Get("entity_id").(string), d.Get("entity_type").(string), d.Get("principal").(string))
	p, err := permission.ByID(client, id)
	if err != nil {
		d.SetId("")
		return err
	}
	if err = d.Set("role_id", p.RoleId); err != nil {
		return err
	}
	d.SetId(id)
	log.Printf("[DEBUG] Successfully read entity permission %q", d.Id())
	return nil
}
