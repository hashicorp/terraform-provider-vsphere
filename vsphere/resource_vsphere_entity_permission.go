package vsphere

import (
	"fmt"
	"github.com/hashicorp/terraform/helper/schema"
	permissionsHelper "github.com/terraform-providers/terraform-provider-vsphere/vsphere/internal/helper/permissions"
)

func resourceVSphereEntityPermission() *schema.Resource {
	return &schema.Resource{
		Read:   resourceVSphereEntityPermissionRead,
		Create: resourceVSphereEntityPermissionCreate,
		Delete: resourceVSphereEntityPermissionDelete,

		Schema: map[string]*schema.Schema{
			"principal": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"role_id": &schema.Schema{
				Type:     schema.TypeInt,
				Required: true,
				ForceNew: true,
			},
			"folder_path": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
				Default:  "/",
			},
			"propagate": &schema.Schema{
				Type:     schema.TypeBool,
				Optional: true,
				ForceNew: true,
			},
			"group": &schema.Schema{
				Type:     schema.TypeBool,
				Optional: true,
				ForceNew: true,
			},
		},
	}
}

func resourceVSphereEntityPermissionRead(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*VSphereClient).vimClient
	principal := d.Get("principal").(string)
	folderPath := d.Get("folder_path").(string)
	permission, err := permissionsHelper.Exists(client, principal, folderPath)
	if err != nil {
		d.SetId("")
		return err
	}

	d.Set("propagate", permission.Propagate)
	d.UnsafeSetFieldRaw("role_id", fmt.Sprint(permission.RoleId))
	d.Set("group", permission.Group)
	d.SetId(permission.Principal)
	return nil
}

func resourceVSphereEntityPermissionCreate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*VSphereClient).vimClient
	principal := d.Get("principal").(string)
	folderPath := d.Get("folder_path").(string)
	group := d.Get("group").(bool)
	roleID := d.Get("role_id").(int)
	propagate := d.Get("propagate").(bool)
	err := permissionsHelper.Create(client, principal, folderPath, roleID, group, propagate)
	if err != nil {
		d.SetId("")
		return err
	}
	return resourceVSphereEntityPermissionRead(d, meta)
}

func resourceVSphereEntityPermissionDelete(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*VSphereClient).vimClient
	principal := d.Get("principal").(string)
	folderPath := d.Get("folder_path").(string)
	permission, err := permissionsHelper.Exists(client, principal, folderPath)
	if err != nil {
		d.SetId("")
		return err
	}

	err = permissionsHelper.Remove(client, permission)
	if err != nil {
		return err
	}
	d.SetId("")
	return nil
}
