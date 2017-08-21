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

func resourceVSphereHostVirtualSwitch() *schema.Resource {
	s := map[string]*schema.Schema{
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
		"datacenter": &schema.Schema{
			Type:        schema.TypeString,
			Description: "The path to the datacenter the host is located in. This is ignored if connecting directly to ESXi. If not specified on vCenter, the default datacenter is used.",
			Optional:    true,
			ForceNew:    true,
		},
	}
	mergeSchema(s, schemaHostVirtualSwitchSpec())

	// Transform any necessary fields in the schema that need to be updated
	// specifically for this resource.
	s["active_nics"].Required = true
	s["standby_nics"].Required = true

	return &schema.Resource{
		Create: resourceVSphereHostVirtualSwitchCreate,
		Read:   resourceVSphereHostVirtualSwitchRead,
		Update: resourceVSphereHostVirtualSwitchUpdate,
		Delete: resourceVSphereHostVirtualSwitchDelete,
		Schema: s,
	}
}

func resourceVSphereHostVirtualSwitchCreate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*govmomi.Client)
	host := d.Get("host").(string)
	datacenter := d.Get("datacenter").(string)
	ns, err := hostNetworkSystemFromName(client, host, datacenter)
	if err != nil {
		return fmt.Errorf("error loading network system: %s", err)
	}

	timeout := time.Duration(float64(d.Timeout(schema.TimeoutCreate)) * 0.8)
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	spec := expandHostVirtualSwitchSpec(d)
	if err := ns.AddVirtualSwitch(ctx, d.Get("name").(string), spec); err != nil {
		return fmt.Errorf("error adding host vSwitch: %s", err)
	}

	d.SetId(d.Get("name").(string))
	return resourceVSphereHostVirtualSwitchRead(d, meta)
}

func resourceVSphereHostVirtualSwitchRead(d *schema.ResourceData, meta interface{}) error {
	timeout := time.Duration(float64(d.Timeout(schema.TimeoutCreate)) * 0.8)
	id := d.Id()
	host := d.Get("host").(string)
	datacenter := d.Get("datacenter").(string)
	sw, err := hostVSwitchFromName(meta.(*govmomi.Client), id, host, datacenter, timeout)
	if err != nil {
		return fmt.Errorf("error fetching virtual switch data: %s", err)
	}

	if err := flattenHostVirtualSwitchSpec(d, &sw.Spec); err != nil {
		return fmt.Errorf("error setting resource data: %s", err)
	}

	return nil
}

func resourceVSphereHostVirtualSwitchUpdate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*govmomi.Client)
	host := d.Get("host").(string)
	datacenter := d.Get("datacenter").(string)
	ns, err := hostNetworkSystemFromName(client, host, datacenter)
	if err != nil {
		return fmt.Errorf("error loading network system: %s", err)
	}

	timeout := time.Duration(float64(d.Timeout(schema.TimeoutCreate)) * 0.8)
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	spec := expandHostVirtualSwitchSpec(d)
	if err := ns.UpdateVirtualSwitch(ctx, d.Get("name").(string), *spec); err != nil {
		return fmt.Errorf("error updating host vSwitch: %s", err)
	}

	return resourceVSphereHostVirtualSwitchRead(d, meta)
}

func resourceVSphereHostVirtualSwitchDelete(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*govmomi.Client)
	ns, err := hostNetworkSystemFromName(client, d.Get("host").(string), d.Get("datacenter").(string))
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
func hostNetworkSystemFromName(client *govmomi.Client, host, datacenter string) (*object.HostNetworkSystem, error) {
	finder := find.NewFinder(client.Client, false)

	var hs *object.HostSystem
	var err error
	switch t := client.ServiceContent.About.ApiType; t {
	case "HostAgent":
		dc, err := getDatacenter(client, "")
		if err != nil {
			return nil, fmt.Errorf("could not get datacenter: %s", err)
		}
		finder.SetDatacenter(dc)
		hs, err = finder.DefaultHostSystem(context.TODO())
	case "VirtualCenter":
		dc, err := getDatacenter(client, datacenter)
		if err != nil {
			return nil, fmt.Errorf("could not get datacenter: %s", err)
		}
		finder.SetDatacenter(dc)
		hs, err = finder.HostSystem(context.TODO(), host)
	default:
		return nil, fmt.Errorf("unsupported ApiType: %s", t)
	}
	if err != nil {
		return nil, fmt.Errorf("error loading host system: %s", err)
	}
	return hs.ConfigManager().NetworkSystem(context.TODO())
}

// hostVSwitchFromName locates a host virtual switch from its assigned name and
// host, using the client's default property collector.
func hostVSwitchFromName(client *govmomi.Client, name, host, datacenter string, timeout time.Duration) (*types.HostVirtualSwitch, error) {
	ns, err := hostNetworkSystemFromName(client, host, datacenter)
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
