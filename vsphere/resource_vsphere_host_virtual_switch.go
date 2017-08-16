package vsphere

import (
	"fmt"

	"context"
	"github.com/hashicorp/terraform/helper/schema"
	"github.com/vmware/govmomi"
	"github.com/vmware/govmomi/find"
	"github.com/vmware/govmomi/object"
)

func resourceVsphereHostVirtualSwitch() *schema.Resource {
	return &schema.Resource{
		Create: resourceVsphereHostVirtualSwitchCreate,
		Read:   resourceVsphereHostVirtualSwitchRead,
		Update: resourceVsphereHostVirtualSwitchUpdate,
		Delete: resourceVsphereHostVirtualSwitchDelete,

		Schema: map[string]*schema.Schema{
			"name": &schema.Schema{
				Type:        schema.TypeString,
				Description: "Name name of the virtual switch.",
				Required:    true,
				ForceNew:    true,
			},
			"host": &schema.Schema{
				Type:        schema.TypeString,
				Description: "The name of the host to put this virtual switch on. This is ignored if connecting directly to ESXi, but required if not.",
				Optional:    true,
				ForceNew:    true,
			},
			"spec": &schema.Schema{
				Type:        schema.TypeSet,
				Description: "The specification for the virtual switch.",
				Required:    true,
				MaxItems:    1,
				Elem:        schemaHostVirtualSwitchSpec(),
			},
		},
	}
}

func resourceVsphereHostVirtualSwitchCreate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*govmomi.Client)
	ns, err := hostNetworkSystemFromName(client, d.Get("host").(string))
	if err != nil {
		return fmt.Errorf("error loading network system: %s", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), d.Timeout(schema.TimeoutCreate))
	defer cancel()
	spec := resourceToHostVirtualSwitchSpec(d.Get("spec").(*schema.Set).List()[0].(map[string]interface{}))
	if err := ns.AddVirtualSwitch(ctx, d.Get("name").(string), spec); err != nil {
		return fmt.Errorf("error adding host vSwitch: %s", err)
	}

	d.SetId(d.Get("name").(string))
	return resourceVsphereHostVirtualSwitchRead(d, meta)
}

func resourceVsphereHostVirtualSwitchRead(d *schema.ResourceData, meta interface{}) error {
	return nil
}

func resourceVsphereHostVirtualSwitchUpdate(d *schema.ResourceData, meta interface{}) error {
	return resourceVsphereHostVirtualSwitchRead(d, meta)
}

func resourceVsphereHostVirtualSwitchDelete(d *schema.ResourceData, meta interface{}) error {
	d.SetId("")
	return nil
}

// hostNetworkSystemFromName locates a HostNetworkSystem from a specified host
// name. The default host system is used if the client is connected to an ESXi
// host, versus vCenter.
func hostNetworkSystemFromName(client *govmomi.Client, name string) (*object.HostNetworkSystem, error) {
	finder := find.NewFinder(client.Client, false)

	var host *object.HostSystem
	var err error
	switch t := client.ServiceContent.About.ApiType; t {
	case "HostAgent":
		host, err = finder.DefaultHostSystem(context.TODO())
	case "VirtualCenter":
		host, err = finder.HostSystem(context.TODO(), name)
	default:
		return nil, fmt.Errorf("unsupported ApiType: %s", t)
	}
	if err != nil {
		return nil, fmt.Errorf("error loading host system: %s", err)
	}
	return host.ConfigManager().NetworkSystem(context.TODO())
}
