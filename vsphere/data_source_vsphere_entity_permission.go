package vsphere

import (
	"fmt"

	"github.com/hashicorp/terraform/helper/schema"
	"github.com/terraform-providers/terraform-provider-vsphere/vsphere/internal/helper/permissions"
)

func dataSourceVSphereEntityPermission() *schema.Resource {
	return &schema.Resource{
		Read: dataSourceVSphereEntityPermissionRead,

		Schema: map[string]*schema.Schema{
			"principal": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},
			"folder_path": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				Default:  "/",
			},
			"role_id": &schema.Schema{
				Type:     schema.TypeInt,
				Computed: true,
			},
			"propagate": &schema.Schema{
				Type:     schema.TypeBool,
				Computed: true,
			},
			"group": &schema.Schema{
				Type:     schema.TypeBool,
				Computed: true,
			},
		},
	}
}

func dataSourceVSphereEntityPermissionRead(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*VSphereClient).vimClient
	p, err := permission.ByID(client, d.Id())
	if err != nil {
		d.SetId("")
		return err
	}
	_, f, err := permission.SplitID(d.Id())
	if err != nil {
		return err
	}
	d.Set("propagate", p.Propagate)
	d.Set("role_id", fmt.Sprint(p.RoleId))
	d.Set("folder_path", f)
	d.Set("group", p.Group)
	d.SetId(p.Principal)
	return nil
}
