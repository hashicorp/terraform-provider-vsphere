package vsphere

import (
	"errors"
	"fmt"

	"github.com/hashicorp/terraform/helper/schema"
	"github.com/hashicorp/terraform/helper/validation"
	"github.com/terraform-providers/terraform-provider-vsphere/vsphere/internal/helper/role"
)

func resourceVSphereRole() *schema.Resource {
	return &schema.Resource{
		Create: resourceVSphereRoleCreate,
		Read:   resourceVSphereRoleRead,
		Update: resourceVSphereRoleUpdate,
		Delete: resourceVSphereRoleDelete,
		Importer: &schema.ResourceImporter{
			State: resourceVSphereRoleImport,
		},

		Schema: map[string]*schema.Schema{
			"name": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"permissions": &schema.Schema{
				Type:     schema.TypeSet,
				Required: true,
				Elem: &schema.Schema{
					Type:         schema.TypeString,
					ValidateFunc: validation.StringInSlice(role.PermissionsList, false),
				},
			},
		},
	}
}

func resourceVSphereRoleCreate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*VSphereClient).vimClient
	name := d.Get("name").(string)
	permsTemp := d.Get("permissions").(*schema.Set)
	permsList := make([]string, len(permsTemp.List()))
	for i, v := range permsTemp.List() {
		permsList[i] = v.(string)
	}

	roleID, err := role.Create(client, name, permsList)
	if err != nil {
		return errors.New("Error during creating role: " + err.Error())
	}

	d.SetId(fmt.Sprint(roleID))

	return resourceVSphereRoleRead(d, meta)
}

func resourceVSphereRoleRead(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*VSphereClient).vimClient
	roleVal, err := role.ByName(client, d.Get("name").(string))
	if err != nil || roleVal == nil {
		d.SetId("")
		return fmt.Errorf("couldn't find the specified role: %s", err.Error())
	}

	d.Set("name", roleVal.Name)
	permissions := []string{}
	for _, v := range roleVal.Privilege {
		switch v {
		case "System.Anonymous":
			continue
		case "System.Read":
			continue
		case "System.View":
			continue
		default:
			permissions = append(permissions, v)
		}
	}
	err = d.Set("permissions", permissions)
	if err != nil {
		return err
	}

	return nil
}

func resourceVSphereRoleUpdate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*VSphereClient).vimClient
	name := d.Get("name").(string)
	permsTemp := d.Get("permissions").(*schema.Set)
	permsList := make([]string, len(permsTemp.List()))
	for i, v := range permsTemp.List() {
		permsList[i] = v.(string)
	}

	roleVal, err := role.ByName(client, d.Get("name").(string))
	if err != nil {
		d.SetId("")
		return errors.New("couldn't find the specified role: " + err.Error())
	}

	err = role.Update(client, roleVal.RoleId, name, permsList)
	if err != nil {
		return errors.New("Error while updating role: " + err.Error())
	}

	return resourceVSphereRoleRead(d, meta)
}

func resourceVSphereRoleDelete(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*VSphereClient).vimClient

	roleVal, err := role.ByName(client, d.Get("name").(string))
	if err != nil {
		d.SetId("")
		return errors.New("couldn't find the specified role: " + err.Error())
	}

	err = role.Remove(client, roleVal.RoleId)
	if err != nil {
		return errors.New("couldn't find the specified role: " + err.Error())
	}
	d.SetId("")
	return nil
}

func resourceVSphereRoleImport(d *schema.ResourceData, meta interface{}) ([]*schema.ResourceData, error) {
	client := meta.(*VSphereClient).vimClient
	name := d.Id()
	if name == "" {
		return nil, fmt.Errorf("role name cannot be empty")
	}

	roleVal, err := role.ByName(client, name)
	if err != nil {
		return nil, fmt.Errorf("couldn't find the specified role")
	}

	d.Set("name", roleVal.Name)
	permissions := []string{}
	for _, v := range roleVal.Privilege {
		switch v {
		case "System.Anonymous":
			continue
		case "System.Read":
			continue
		case "System.View":
			continue
		default:
			permissions = append(permissions, v)
		}
	}

	d.Set("permissions", permissions)
	d.SetId(fmt.Sprint(roleVal.RoleId))

	return []*schema.ResourceData{d}, nil
}
