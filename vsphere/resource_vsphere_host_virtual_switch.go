package vsphere

import (
	"fmt"
	"time"

	"context"

	"github.com/hashicorp/terraform/helper/schema"
	"github.com/vmware/govmomi"
)

func resourceVSphereHostVirtualSwitch() *schema.Resource {
	s := map[string]*schema.Schema{
		"name": &schema.Schema{
			Type:        schema.TypeString,
			Description: "The name of the virtual switch.",
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

	s["teaming_policy"].Default = "loadbalance_srcid"
	s["check_beacon"].Default = false
	s["notify_switches"].Default = true
	s["failback"].Default = true

	s["allow_promiscuous"].Default = false
	s["forged_transmits"].Default = true
	s["mac_changes"].Default = true

	s["shaping_enabled"].Default = false

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
	if err := ns.UpdateVirtualSwitch(ctx, d.Id(), *spec); err != nil {
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
