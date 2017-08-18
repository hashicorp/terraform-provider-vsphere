package vsphere

import (
	"fmt"
	"time"

	"context"

	"github.com/hashicorp/terraform/helper/schema"
	"github.com/vmware/govmomi"
	"github.com/vmware/govmomi/find"
	"github.com/vmware/govmomi/object"
	"github.com/vmware/govmomi/vim25/mo"
	"github.com/vmware/govmomi/vim25/types"
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

	timeout := time.Duration(float64(d.Timeout(schema.TimeoutCreate)) * 0.8)
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	spec := resourceToHostVirtualSwitchSpec(d.Get("spec").(*schema.Set).List()[0].(map[string]interface{}))
	if err := ns.AddVirtualSwitch(ctx, d.Get("name").(string), spec); err != nil {
		return fmt.Errorf("error adding host vSwitch: %s", err)
	}

	d.SetId(d.Get("name").(string))
	return resourceVsphereHostVirtualSwitchRead(d, meta)
}

func resourceVsphereHostVirtualSwitchRead(d *schema.ResourceData, meta interface{}) error {
	timeout := time.Duration(float64(d.Timeout(schema.TimeoutCreate)) * 0.8)
	sw, err := hostVSwitchFromName(meta.(*govmomi.Client), d.Id(), d.Get("host").(string), timeout)
	if err != nil {
		return fmt.Errorf("error fetching virtual switch data: %s", err)
	}

	spec := hostVirtualSwitchSpecToResource(&sw.Spec)
	if err := d.Set("spec", spec); err != nil {
		return fmt.Errorf("error setting resource data: %s", err)
	}

	return nil
}

func resourceVsphereHostVirtualSwitchUpdate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*govmomi.Client)
	ns, err := hostNetworkSystemFromName(client, d.Get("host").(string))
	if err != nil {
		return fmt.Errorf("error loading network system: %s", err)
	}

	timeout := time.Duration(float64(d.Timeout(schema.TimeoutCreate)) * 0.8)
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	spec := resourceToHostVirtualSwitchSpec(d.Get("spec").(*schema.Set).List()[0].(map[string]interface{}))
	if err := ns.UpdateVirtualSwitch(ctx, d.Get("name").(string), *spec); err != nil {
		return fmt.Errorf("error updating host vSwitch: %s", err)
	}

	return resourceVsphereHostVirtualSwitchRead(d, meta)
}

func resourceVsphereHostVirtualSwitchDelete(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*govmomi.Client)
	ns, err := hostNetworkSystemFromName(client, d.Get("host").(string))
	if err != nil {
		return fmt.Errorf("error loading network system: %s", err)
	}

	timeout := time.Duration(float64(d.Timeout(schema.TimeoutCreate)) * 0.8)
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	if err := ns.RemoveVirtualSwitch(ctx, d.Id()); err != nil {
		return fmt.Errorf("error deleting host vSwitch: %s", err)
	}

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

// hostVSwitchFromName locates a host virtual switch from its assigned name and
// host, using the client's default property collector.
func hostVSwitchFromName(client *govmomi.Client, name, host string, timeout time.Duration) (*types.HostVirtualSwitch, error) {
	ns, err := hostNetworkSystemFromName(client, host)
	if err != nil {
		return nil, fmt.Errorf("error loading network system: %s", err)
	}

	var mns mo.HostNetworkSystem
	pc := client.PropertyCollector()
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	if err := pc.RetrieveOne(ctx, ns.Reference(), []string{"networkInfo.vswitch"}, &mns); err != nil {
		return nil, fmt.Errorf("error fetching host network properties: %s", err)
	}

	for _, sw := range mns.NetworkInfo.Vswitch {
		if sw.Name == name {
			return &sw, nil
		}
	}

	return nil, fmt.Errorf("vSwitch %s not found on host %s", name, host)
}
