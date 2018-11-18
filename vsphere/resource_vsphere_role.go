package vsphere

import (
	"errors"
	"fmt"

	"github.com/hashicorp/terraform/helper/schema"
	"github.com/hashicorp/terraform/helper/validation"
	roleHelper "github.com/terraform-providers/terraform-provider-vsphere/vsphere/internal/helper/role"
)

func resourceVSphereRole() *schema.Resource {
	return &schema.Resource{
		Create: resourceVSphereRoleCreate,
		Read:   resourceVSphereRoleRead,
		Update: resourceVSphereRoleUpdate,
		Delete: resourceVSphereRoleDelete,

		Schema: map[string]*schema.Schema{
			"name": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"permissions": &schema.Schema{
				Type:     schema.TypeList,
				Required: true,
				Elem: &schema.Schema{
					Type:         schema.TypeString,
					ValidateFunc: validation.StringInSlice(roleHelper.PermissionsList, false),
				},
			},
		},
	}
}

func resourceVSphereRoleCreate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*VSphereClient).vimClient
	name := d.Get("name").(string)
	permsTemp := d.Get("permissions").([]interface{})
	perms := make([]string, len(permsTemp))
	for i := range permsTemp {
		perms[i] = permsTemp[i].(string)
	}

	roleID, err := roleHelper.Create(client, name, perms)
	if err != nil {
		d.SetId("")
		return errors.New("Error during creating role: " + err.Error())
	}

	d.SetId(fmt.Sprint(roleID))

	return resourceVSphereRoleRead(d, meta)
}

func resourceVSphereRoleRead(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*VSphereClient).vimClient
	role, err := roleHelper.ExistsByName(client, d.Get("name").(string))
	if err != nil {
		d.SetId("")
		return errors.New("couldn't find the specified role: " + err.Error())
	}

	d.SetId(fmt.Sprint(role.RoleId))
	return nil
}

func resourceVSphereRoleUpdate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*VSphereClient).vimClient
	name := d.Get("name").(string)
	permsTemp := d.Get("permissions").([]interface{})
	perms := make([]string, len(permsTemp))
	for i := range permsTemp {
		perms[i] = permsTemp[i].(string)
	}

	role, err := roleHelper.ExistsByName(client, d.Get("name").(string))
	if err != nil {
		d.SetId("")
		return errors.New("couldn't find the specified role: " + err.Error())
	}

	err = roleHelper.Update(client, role.RoleId, name, perms)
	if err != nil {
		return errors.New("Error while updating role: " + err.Error())
	}

	return nil
}

func resourceVSphereRoleDelete(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*VSphereClient).vimClient

	role, err := roleHelper.ExistsByName(client, d.Get("name").(string))
	if err != nil {
		d.SetId("")
		return errors.New("couldn't find the specified role: " + err.Error())
	}

	err = roleHelper.Remove(client, role.RoleId)
	if err != nil {
		return errors.New("couldn't find the specified role: " + err.Error())
	}
	d.SetId("")
	return nil
}
