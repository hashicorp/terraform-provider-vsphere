package vsphere

import (
	"fmt"
	"time"

	"context"

	"github.com/hashicorp/terraform/helper/schema"
	"github.com/vmware/govmomi"
)

func resourceVSphereHostPortGroup() *schema.Resource {
	s := map[string]*schema.Schema{
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
		"computed_policy": &schema.Schema{
			Type:        schema.TypeMap,
			Description: "The effective network policy after inheritance. Note that this will look similar to, but is not the same, as the policy attributes defined in this resource.",
			Computed:    true,
		},
		"key": &schema.Schema{
			Type:        schema.TypeString,
			Description: "The linkable identifier for this port group.",
			Computed:    true,
		},
		"ports": &schema.Schema{
			Type:        schema.TypeSet,
			Description: "The ports that currently exist and are used on this port group.",
			Computed:    true,
			MaxItems:    1,
			Elem:        portGroupPortSchema(),
		},
	}
	mergeSchema(s, schemaHostPortGroupSpec())

	// Transform any necessary fields in the schema that need to be updated
	// specifically for this resource.
	s["active_nics"].Optional = true
	s["standby_nics"].Optional = true

	return &schema.Resource{
		Create: resourceVSphereHostPortGroupCreate,
		Read:   resourceVSphereHostPortGroupRead,
		Update: resourceVSphereHostPortGroupUpdate,
		Delete: resourceVSphereHostPortGroupDelete,
		Schema: s,
	}
}

func resourceVSphereHostPortGroupCreate(d *schema.ResourceData, meta interface{}) error {
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
	spec := expandHostPortGroupSpec(d)
	if err := ns.AddPortGroup(ctx, *spec); err != nil {
		return fmt.Errorf("error adding port group: %s", err)
	}

	d.SetId(d.Get("name").(string))
	return resourceVSphereHostPortGroupRead(d, meta)
}

func resourceVSphereHostPortGroupRead(d *schema.ResourceData, meta interface{}) error {
	timeout := time.Duration(float64(d.Timeout(schema.TimeoutCreate)) * 0.8)
	id := d.Id()
	host := d.Get("host").(string)
	datacenter := d.Get("datacenter").(string)
	pg, err := hostPortGroupFromName(meta.(*govmomi.Client), id, host, datacenter, timeout)
	if err != nil {
		return fmt.Errorf("error fetching port group data: %s", err)
	}

	if err := flattenHostPortGroupSpec(d, &pg.Spec); err != nil {
		return fmt.Errorf("error setting resource data: %s", err)
	}

	d.Set("key", pg.Key)
	cpm, err := calculateComputedPolicy(pg.ComputedPolicy)
	if err != nil {
		return err
	}
	if err := d.Set("computed_policy", cpm); err != nil {
		return fmt.Errorf("error saving effective policy to state: %s", err)
	}
	if err := d.Set("ports", calculatePorts(pg.Port)); err != nil {
		return fmt.Errorf("error setting port list: %s", err)
	}

	return nil
}

func resourceVSphereHostPortGroupUpdate(d *schema.ResourceData, meta interface{}) error {
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
	spec := expandHostPortGroupSpec(d)
	if err := ns.UpdatePortGroup(ctx, d.Id(), *spec); err != nil {
		return fmt.Errorf("error updating port group: %s", err)
	}

	return resourceVSphereHostPortGroupRead(d, meta)
}

func resourceVSphereHostPortGroupDelete(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*govmomi.Client)
	ns, err := hostNetworkSystemFromName(client, d.Get("host").(string), d.Get("datacenter").(string))
	if err != nil {
		return fmt.Errorf("error loading network system: %s", err)
	}

	timeout := time.Duration(float64(d.Timeout(schema.TimeoutCreate)) * 0.8)
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	if err := ns.RemovePortGroup(ctx, d.Id()); err != nil {
		return fmt.Errorf("error deleting port group: %s", err)
	}

	d.SetId("")
	return nil
}
