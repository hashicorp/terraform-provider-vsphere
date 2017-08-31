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
	name := d.Get("name").(string)

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

	// Configure the host and nic cards used as uplink for the DVS
	var host []types.DistributedVirtualSwitchHostMemberConfigSpec

	if v, ok := d.GetOk("host"); ok {
		for _, vi := range v.([]interface{}) {
			hi := vi.(map[string]interface{})
			bi := hi["backing"].([]interface{})
			// Get the HostSystem reference
			hs, err := finder.HostSystem(context.TODO(), hi["host"].(string))
			if err != nil {
				return fmt.Errorf("%s", err)
			}

			// Get the physical NIC backing
			backing := new(types.DistributedVirtualSwitchHostMemberPnicBacking)
			backing.PnicSpec = append(backing.PnicSpec, types.DistributedVirtualSwitchHostMemberPnicSpec{
				PnicDevice: strings.TrimSpace(bi[0].(string)),
			})
			h := types.DistributedVirtualSwitchHostMemberConfigSpec{
				Host:      hs.Common.Reference(),
				Backing:   backing,
				Operation: "add", // Options: "add", "edit", "remove"
			}
			host = append(host, h)
		}
	}

	configSpec := types.DVSConfigSpec{
		Name: name,
		Host: host,
	}
	dvsCreateSpec := types.DVSCreateSpec{ConfigSpec: &configSpec}

	f := df.NetworkFolder

	task, err := f.CreateDVS(context.TODO(), dvsCreateSpec)
	if err != nil {
		return fmt.Errorf("%s", err)
	}

	err = task.Wait(context.TODO())
	if err != nil {
		return fmt.Errorf("%s", err)
	}

	d.SetId(name)

	return nil
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
	return nil
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
