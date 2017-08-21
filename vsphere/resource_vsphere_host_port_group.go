package vsphere

import (
	"fmt"
	"time"

	"context"

	"github.com/hashicorp/terraform/helper/schema"
	"github.com/hashicorp/terraform/terraform"
	"github.com/vmware/govmomi"
	"github.com/vmware/govmomi/vim25/mo"
	"github.com/vmware/govmomi/vim25/types"
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

// hostPortGroupFromName locates a host port group from its assigned name and
// host, using the client's default property collector.
func hostPortGroupFromName(client *govmomi.Client, name, host, datacenter string, timeout time.Duration) (*types.HostPortGroup, error) {
	ns, err := hostNetworkSystemFromName(client, host, datacenter)
	if err != nil {
		return nil, fmt.Errorf("error loading network system: %s", err)
	}

	var mns mo.HostNetworkSystem
	pc := client.PropertyCollector()
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	if err := pc.RetrieveOne(ctx, ns.Reference(), []string{"networkInfo.portgroup"}, &mns); err != nil {
		return nil, fmt.Errorf("error fetching host network properties: %s", err)
	}

	for _, pg := range mns.NetworkInfo.Portgroup {
		if pg.Spec.Name == name {
			return &pg, nil
		}
	}

	return nil, fmt.Errorf("port group %s not found on host %s", name, host)
}

// calculateComputedPolicy is a utility function to compute a map of state
// attributes for the port group's effective policy. It uses a bit of a
// roundabout way to set the attributes, but allows us to utilize our
// functional deep reading helpers to perform this task, versus having to
// re-write code.
//
// This function relies a bit on some of the lower-level utility functionality
// of helper/schema, so it may need to change in the future.
func calculateComputedPolicy(policy types.HostNetworkPolicy) (map[string]string, error) {
	cpr := &schema.Resource{Schema: schemaHostNetworkPolicy()}
	cpd := cpr.Data(&terraform.InstanceState{})
	cpd.SetId("effectivepolicy")
	if err := flattenHostNetworkPolicy(cpd, &policy); err != nil {
		return nil, fmt.Errorf("error setting effective policy data: %s", err)
	}
	cpm := cpd.State().Attributes
	delete(cpm, "id")
	return cpm, nil
}

// calculatePorts is a utility function that returns a set of port data.
func calculatePorts(ports []types.HostPortGroupPort) *schema.Set {
	s := make([]interface{}, 0)
	for _, port := range ports {
		m := make(map[string]interface{})
		m["key"] = port.Key
		m["mac_addresses"] = sliceStringsToInterfaces(port.Mac)
		m["type"] = port.Type
		s = append(s, m)
	}
	return schema.NewSet(schema.HashResource(portGroupPortSchema()), s)
}

func portGroupPortSchema() *schema.Resource {
	return &schema.Resource{
		Schema: map[string]*schema.Schema{
			"key": &schema.Schema{
				Type:        schema.TypeString,
				Description: "The linkable identifier for this port entry.",
				Computed:    true,
			},
			"mac_addresses": &schema.Schema{
				Type:        schema.TypeList,
				Description: "The MAC addresses of the network service of the virtual machine connected on this port.",
				Computed:    true,
				Elem:        &schema.Schema{Type: schema.TypeString},
			},
			"type": &schema.Schema{
				Type:        schema.TypeString,
				Description: "Type type of the entity connected on this port. Possible values are host (VMKkernel), systemManagement (service console), virtualMachine, or unknown.",
				Computed:    true,
			},
		},
	}
}
