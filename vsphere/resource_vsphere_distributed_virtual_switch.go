package vsphere

import (
	"fmt"
	"log"
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
		"datacenter_id": &schema.Schema{
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
	dId := d.Get("datacenter_id").(string)

	dc, err := datacenterFromID(client, dId)
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

	hosts_mor, err := getHostSystemManagedObjectReference(d, client)
	if err != nil {
		return fmt.Errorf("%s", err)
	}

	spec := expandDVSConfigSpec(client, d, nil, hosts_mor)
	dvsCreateSpec := types.DVSCreateSpec{ConfigSpec: spec}

	task, err := f.CreateDVS(context.TODO(), dvsCreateSpec)
	if err != nil {
		return fmt.Errorf("%s", err)
	}

	err = task.Wait(context.TODO())
	if err != nil {
		return fmt.Errorf("%s", err)
	}

	// Ideally from the CreateDVS opperation we should be able to access the UUID
	// but I'm not sure how with the current operations exposed by the SDK
	dvs, err := dvsFromName(client, dId, name)
	if err != nil {
		return fmt.Errorf("%s", err)
	}

	d.SetId(dvs.Uuid)

	return resourceVSphereDistributedVirtualSwitchRead(d, meta)
}

func resourceVSphereDistributedVirtualSwitchRead(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*govmomi.Client)
	dvs, err := dvsFromUuid(client, d.Id())
	if err != nil {
		d.SetId("")
		return fmt.Errorf("error reading data: %s", err)
	}

	if err := flattenDVSConfigSpec(client, d, dvs); err != nil {
		return fmt.Errorf("error setting resource data: %s", err)
	}

	return nil
}

func resourceVSphereDistributedVirtualSwitchUpdate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*govmomi.Client)
	dvs, err := dvsFromUuid(client, d.Id())
	if err != nil {
		return fmt.Errorf("%s", err)
	}

	hosts_refs, err := getHostSystemManagedObjectReference(d, client)
	if err != nil {
		return fmt.Errorf("%s", err)
	}

	spec := expandDVSConfigSpec(client, d, dvs, hosts_refs)

	n := object.NewDistributedVirtualSwitch(client.Client, dvs.Reference())
	req := &types.ReconfigureDvs_Task{
		This: n.Reference(),
		Spec: spec,
	}

	_, err = methods.ReconfigureDvs_Task(context.TODO(), client, req)
	if err != nil {
		return fmt.Errorf("%s", err)
	}

	// Wait for the distributed virtual switch resource to be destroyed
	stateConf := &resource.StateChangeConf{
		Pending:    []string{"Updating"},
		Target:     []string{"Updated"},
		Refresh:    resourceVSphereDVSStateUpdateRefreshFunc(d, meta),
		Timeout:    10 * time.Minute,
		MinTimeout: 3 * time.Second,
		Delay:      5 * time.Second,
	}

	_, err = stateConf.WaitForState()
	if err != nil {
		name := d.Get("name").(string)
		return fmt.Errorf("error waiting for distributed virtual switch (%s) to be updated: %s", name, err)
	}

	return resourceVSphereDistributedVirtualSwitchRead(d, meta)
}

func resourceVSphereDVSStateUpdateRefreshFunc(d *schema.ResourceData, meta interface{}) resource.StateRefreshFunc {
	return func() (interface{}, string, error) {
		log.Print("[TRACE] Refreshing distributed virtual switch state, looking for config changes")
		client := meta.(*govmomi.Client)
		dvs, err := dvsFromUuid(client, d.Id())
		if err != nil {
			return nil, "Failed", err
		}
		config := dvs.Config.GetDVSConfigInfo()
		cv := d.Get("config_version").(string)
		log.Printf("[TRACE] Current version %s. Old version %s", config.ConfigVersion, cv)
		if config.ConfigVersion != cv {
			log.Print("[TRACE] Distributed virtual switch config updated")
			return dvs, "Updated", nil
		} else {
			return dvs, "Updating", nil
		}
	}
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
