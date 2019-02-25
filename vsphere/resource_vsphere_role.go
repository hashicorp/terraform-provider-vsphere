package vsphere

import (
	"errors"
	"fmt"
	"log"

	"github.com/hashicorp/terraform/helper/schema"
	"github.com/hashicorp/terraform/helper/validation"
	"github.com/terraform-providers/terraform-provider-vsphere/vsphere/internal/helper/role"
	"github.com/terraform-providers/terraform-provider-vsphere/vsphere/internal/helper/structure"
)

const resourceVSphereRoleName = "vsphere_role"

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
	log.Printf("[DEBUG] %s: Beginning create", resourceVSphereRoleIDString(d))
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

	log.Printf("[DEBUG] %s: Create completed successfully", resourceVSphereRoleIDString(d))
	return resourceVSphereRoleRead(d, meta)
}

func resourceVSphereRoleRead(d *schema.ResourceData, meta interface{}) error {
	log.Printf("[DEBUG] %s: Beginning read", resourceVSphereRoleIDString(d))
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

	log.Printf("[DEBUG] %s: Read completed successfully", resourceVSphereRoleIDString(d))
	return nil
}

func resourceVSphereRoleUpdate(d *schema.ResourceData, meta interface{}) error {
	log.Printf("[DEBUG] %s: Beginning update", resourceVSphereRoleIDString(d))
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

	log.Printf("[DEBUG] %s: Update completed successfully", resourceVSphereRoleIDString(d))
	return resourceVSphereRoleRead(d, meta)
}

func resourceVSphereRoleDelete(d *schema.ResourceData, meta interface{}) error {
	log.Printf("[DEBUG] %s: Beginning delete", resourceVSphereRoleIDString(d))
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
	log.Printf("[DEBUG] %s: Deleted successfully", resourceVSphereRoleIDString(d))
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

// resourceVSphereRoleIDString prints a friendly string for the
// vsphere_role resource.
func resourceVSphereRoleIDString(d structure.ResourceIDStringer) string {
	return structure.ResourceIDString(d, resourceVSphereRoleName)
}
