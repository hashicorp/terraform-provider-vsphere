package vsphere

import (
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/helper/schema"
	"github.com/vmware/govmomi"
	"github.com/vmware/govmomi/find"
	"github.com/vmware/govmomi/object"
	"github.com/vmware/govmomi/vim25/methods"
	"github.com/vmware/govmomi/vim25/types"
	"golang.org/x/net/context"
)

func resourceVSphereDistributedVirtualSwitch() *schema.Resource {
	s := map[string]*schema.Schema{
		"datacenter": &schema.Schema{
			Type:     schema.TypeString,
			Required: true,
			ForceNew: true,
		},
	}
	mergeSchema(s, schemaDVSConfiSpec())

	return &schema.Resource{
		Create: resourceVSphereDistributedVirtualSwitchCreate,
		Read:   resourceVSphereDistributedVirtualSwitchRead,
		Update: resourceVSphereDistributedVirtualSwitchUpdate,
		Delete: resourceVSphereDistributedVirtualSwitchDelete,
		Schema: s,
	}
}

func resourceVSphereDistributedVirtualSwitchCreate(d *schema.ResourceData, meta interface{}) error {

	client := meta.(*govmomi.Client)

	dc, err := getDatacenter(client, d.Get("datacenter").(string))
	if err != nil {
		return fmt.Errorf("%s", err)
	}

	finder := find.NewFinder(client.Client, true)
	finder = finder.SetDatacenter(dc)

	df, err := dc.Folders(context.TODO())
	if err != nil {
		return fmt.Errorf("%s", err)
	}
	f := df.NetworkFolder

	spec := expandDVSConfigSpec(d)
	dvsCreateSpec := types.DVSCreateSpec{ConfigSpec: &spec}

	task, err := f.CreateDVS(context.TODO(), dvsCreateSpec)
	if err != nil {
		return fmt.Errorf("%s", err)
	}

	err = task.Wait(context.TODO())
	if err != nil {
		return fmt.Errorf("%s", err)
	}

	d.SetId(name)

	return resourceVSphereDistributedVirtualSwitchRead(d, meta)
}

func resourceVSphereDistributedVirtualSwitchRead(d *schema.ResourceData, meta interface{}) error {
	_, err := dvsExists(d, meta)
	if err != nil {
		d.SetId("")
	}

	return nil
}

func dvsExists(d *schema.ResourceData, meta interface{}) (object.NetworkReference, error) {
	client := meta.(*govmomi.Client)
	name := d.Get("name").(string)

	dc, err := getDatacenter(client, d.Get("datacenter").(string))
	if err != nil {
		return nil, err
	}

	finder := find.NewFinder(client.Client, true)
	finder = finder.SetDatacenter(dc)

	dvs, err := finder.Network(context.TODO(), name)
	return dvs, err
}

func resourceVSphereDVSStateRefreshFunc(d *schema.ResourceData, meta interface{}) resource.StateRefreshFunc {
	return func() (interface{}, string, error) {
		log.Print("[TRACE] Refreshing distributed virtual switch state")
		dvs, err := dvsExists(d, meta)
		if err != nil {
			switch err.(type) {
			case *find.NotFoundError:
				log.Printf("[TRACE] Refreshing state. Distributed virtual switch not found: %s", err)
				return nil, "InProgress", nil
			default:
				return nil, "Failed", err
			}
		}
		log.Print("[TRACE] Refreshing state. Distributed virtual switch found")
		return dvs, "Created", nil
	}
}

func resourceVSphereDistributedVirtualSwitchUpdate(d *schema.ResourceData, meta interface{}) error {
	dvs, err := dvsExists(d, meta)
	if err != nil {
		return fmt.Errorf("%s", err)
	}

	// I might need to use something different since for example for the uplinks it's
	// not enough to remove them from the config spec but keep them there and
	// set the operation to "remove"
	spec := expandDVSConfigSpec(d)

	n := object.NewDistributedVirtualSwitch(client.Client, dvs.Reference())
	req := &types.ReconfigureDvs_Task{
		This: n.Reference(),
		Spec: spec,
	}

	return resourceVSphereDistributedVirtualSwitchRead(d, meta)
}

func resourceVSphereDistributedVirtualSwitchDelete(d *schema.ResourceData, meta interface{}) error {
	dvs, err := dvsExists(d, meta)
	if err != nil {
		return fmt.Errorf("%s", err)
	}

	client := meta.(*govmomi.Client)
	n := object.NewDistributedVirtualSwitch(client.Client, dvs.Reference())
	req := &types.Destroy_Task{
		This: n.Reference(),
	}

	_, err = methods.Destroy_Task(context.TODO(), client, req)
	if err != nil {
		return fmt.Errorf("%s", err)
	}

	// Wait for the distributed virtual switch resource to be destroyed
	stateConf := &resource.StateChangeConf{
		Pending:    []string{"Created"},
		Target:     []string{},
		Refresh:    resourceVSphereDVSStateRefreshFunc(d, meta),
		Timeout:    10 * time.Minute,
		MinTimeout: 3 * time.Second,
		Delay:      5 * time.Second,
	}

	_, err = stateConf.WaitForState()
	if err != nil {
		name := d.Get("name").(string)
		return fmt.Errorf("error waiting for distributed virtual switch (%s) to be deleted: %s", name, err)
	}

	return nil
}
