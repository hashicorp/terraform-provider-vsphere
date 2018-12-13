package vsphere

import (
	"errors"
	"fmt"

	"github.com/hashicorp/terraform/helper/schema"
	roleHelper "github.com/terraform-providers/terraform-provider-vsphere/vsphere/internal/helper/role"
	"github.com/vmware/govmomi/vim25/types"
)

func dataSourceVSphereRole() *schema.Resource {
	return &schema.Resource{
		Read: dataSourceVSphereRoleRead,

		Schema: map[string]*schema.Schema{
			"name": &schema.Schema{
				Type:          schema.TypeString,
				Optional:      true,
				ConflictsWith: []string{"role_id"},
			},
			"role_id": &schema.Schema{
				Type:          schema.TypeInt,
				Optional:      true,
				ConflictsWith: []string{"name"},
			},
			"permissions": &schema.Schema{
				Type:     schema.TypeList,
				Computed: true,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
			},
		},
	}
}

func dataSourceVSphereRoleRead(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*VSphereClient).vimClient
	name := d.Get("name").(string)
	role := new(types.AuthorizationRole)
	roleID := d.Get("role_id").(int)

	if name == "" && roleID == 0 {
		return errors.New("Atleast one of either role_id or name must set")
	} else if name == "" {
		role, _ = roleHelper.ByID(client, roleID)
	} else {
		role, _ = roleHelper.ByName(client, name)
	}

	if role == nil {
		d.SetId("")
		return errors.New("couldn't find the specified role: " + name)
	}

	d.Set("permissions", role.Privilege)
	d.Set("name", role.Name)
	d.SetId(fmt.Sprint(role.RoleId))
	return nil
}
