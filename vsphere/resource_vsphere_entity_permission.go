package vsphere

import (
	"fmt"

	"github.com/hashicorp/terraform/helper/schema"
	"github.com/terraform-providers/terraform-provider-vsphere/vsphere/internal/helper/permissions"
)

func resourceVSphereEntityPermission() *schema.Resource {
	return &schema.Resource{
		Read:   resourceVSphereEntityPermissionRead,
		Create: resourceVSphereEntityPermissionCreate,
		Delete: resourceVSphereEntityPermissionDelete,
		Importer: &schema.ResourceImporter{
			State: resourceVSphereEntityPermissionImport,
		},

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
	principal, folderPath, err := permission.SplitID(d.Id())
	if err != nil {
		return err
	}
	p, err := permission.ByID(client, d.Id())
	if err != nil {
		d.SetId("")
		return err
	}

	d.Set("propagate", p.Propagate)
	d.Set("role_id", fmt.Sprint(p.RoleId))
	d.Set("group", p.Group)
	d.Set("principal", principal)
	d.Set("folder_path", folderPath)
	return nil
}

func resourceVSphereEntityPermissionCreate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*VSphereClient).vimClient
	principal := d.Get("principal").(string)
	folderPath := d.Get("folder_path").(string)
	group := d.Get("group").(bool)
	roleID := d.Get("role_id").(int)
	propagate := d.Get("propagate").(bool)
	err := permission.Create(client, principal, folderPath, roleID, group, propagate)
	if err != nil {
		d.SetId("")
		return err
	}
	d.SetId(permission.ConcatID(folderPath, principal))
	return resourceVSphereEntityPermissionRead(d, meta)
}

func resourceVSphereEntityPermissionDelete(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*VSphereClient).vimClient
	p, err := permission.ByID(client, d.Id())
	if err != nil {
		d.SetId("")
		return err
	}

	err = permission.Remove(client, p)
	if err != nil {
		return err
	}
	d.SetId("")
	return nil
}

func resourceVSphereEntityPermissionImport(d *schema.ResourceData, meta interface{}) ([]*schema.ResourceData, error) {
	return []*schema.ResourceData{d}, nil
}
