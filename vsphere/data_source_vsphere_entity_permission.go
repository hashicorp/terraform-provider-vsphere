package vsphere

import (
	"fmt"
	"github.com/hashicorp/terraform/helper/schema"
	permissionsHelper "github.com/terraform-providers/terraform-provider-vsphere/vsphere/internal/helper/permissions"
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
			"propogate": &schema.Schema{
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
	principal := d.Get("principal").(string)
	folderPath := d.Get("folder_path").(string)
	permission, err := permissionsHelper.Exists(client, principal, folderPath)
	if err != nil {
		d.SetId("")
		return err
	}

	d.Set("propogate", permission.Propagate)
	d.UnsafeSetFieldRaw("role_id", fmt.Sprint(permission.RoleId))
	d.Set("group", permission.Group)
	d.SetId(permission.Principal)
	return nil
}
