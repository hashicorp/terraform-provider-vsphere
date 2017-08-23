package vsphere

import (
	"context"
	"fmt"
	"log"
	"strings"

	"github.com/hashicorp/terraform/helper/schema"
	"github.com/vmware/govmomi"
	"github.com/vmware/govmomi/find"
	"github.com/vmware/govmomi/object"
	"github.com/vmware/govmomi/vim25/types"
)

func resourceVSphereSnapshot() *schema.Resource {
	return &schema.Resource{
		Create: resourceVSphereSnapshotCreate,
		Read:   resourceVSphereSnapshotRead,
		Delete: resourceVSphereSnapshotDelete,

		Schema: map[string]*schema.Schema{
			"vm_id": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"datacenter": {
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
			},
			"snapshot_name": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"description": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"memory": {
				Type:     schema.TypeBool,
				Required: true,
				ForceNew: true,
			},
			"quiesce": {
				Type:     schema.TypeBool,
				Required: true,
				ForceNew: true,
			},
			"remove_children": {
				Type:     schema.TypeBool,
				Optional: true,
				ForceNew: true,
			},
			"consolidate": {
				Type:     schema.TypeBool,
				Optional: true,
				ForceNew: true,
			},
		},
	}
}

func resourceVSphereSnapshotCreate(d *schema.ResourceData, meta interface{}) error {
	vm, err := findVM(d, meta)
	if err != nil {
		return fmt.Errorf("Error while getting the VirtualMachine :%s", err)
	}
	task, err := vm.CreateSnapshot(context.TODO(), d.Get("snapshot_name").(string), d.Get("description").(string), d.Get("memory").(bool), d.Get("quiesce").(bool))
	taskInfo, err := task.WaitForResult(context.TODO(), nil)
	if err != nil {
		log.Printf("[ERROR] Error While Creating the Task for Create Snapshot: %v", err)
		return fmt.Errorf(" Error While Creating the Task for Create Snapshot: %s", err)
	}
	log.Printf("[INFO] Task created for Create Snapshot: %v", task)
	err = task.Wait(context.TODO())

	if err != nil {
		log.Printf("[ERROR] Error While waiting for the Task for Create Snapshot: %v", err)
		return fmt.Errorf(" Error While waiting for the Task for Create Snapshot: %s", err)
	}
	log.Printf("[INFO] Create Snapshot completed %v", d.Get("snapshot_name").(string))
	log.Println("[INFO] Managed Object Reference: " + taskInfo.Result.(types.ManagedObjectReference).Value)
	d.SetId(taskInfo.Result.(types.ManagedObjectReference).Value)
	return nil
}

func resourceVSphereSnapshotDelete(d *schema.ResourceData, meta interface{}) error {
	vm, err := findVM(d, meta)
	if err != nil {
		return fmt.Errorf("Error while getting the VirtualMachine :%s", err)
	}
	resourceVSphereSnapshotRead(d, meta)
	if d.Id() == "" {
		log.Printf("[ERROR] Error While finding the Snapshot: %v", err)
		return nil
	}
	log.Printf("[INFO] Deleting snapshot with name: %v", d.Get("snapshot_name").(string))
	var consolidate_ptr *bool
	var remove_children bool

	if v, ok := d.GetOk("consolidate"); ok {
		consolidate := v.(bool)
		consolidate_ptr = &consolidate
	} else {

		consolidate := true
		consolidate_ptr = &consolidate
	}
	if v, ok := d.GetOk("remove_children"); ok {
		remove_children = v.(bool)
	} else {

		remove_children = false
	}

	task, err := vm.RemoveSnapshot(context.TODO(), d.Id(), remove_children, consolidate_ptr)

	if err != nil {
		log.Printf("[ERROR] Error While Creating the Task for Delete Snapshot: %v", err)
		return fmt.Errorf("Error While Creating the Task for Delete Snapshot: %s", err)
	}
	log.Printf("[INFO] Task created for Delete Snapshot: %v", task)

	err = task.Wait(context.TODO())
	if err != nil {
		log.Printf("[ERROR] Error While waiting for the Task of Delete Snapshot: %v", err)
		return fmt.Errorf("Error While waiting for the Task of Delete Snapshot: %s", err)
	}
	log.Printf("[INFO] Delete Snapshot completed %v", d.Get("snapshot_name").(string))

	return nil
}

func resourceVSphereSnapshotRead(d *schema.ResourceData, meta interface{}) error {
	vm, err := findVM(d, meta)
	if err != nil {
		return fmt.Errorf("Error while getting the VirtualMachine :%s", err)
	}
	snapshot, err := vm.FindSnapshot(context.TODO(), d.Id())

	if err != nil {
		if strings.Contains(err.Error(), "No snapshots for this VM") || strings.Contains(err.Error(), "snapshot \""+d.Get("snapshot_name").(string)+"\" not found") {
			log.Printf("[ERROR] Error While finding the Snapshot: %v", err)
			d.SetId("")
			return nil
		}
		log.Printf("[ERROR] Error While finding the Snapshot: %v", err)
		return fmt.Errorf("Error while finding the Snapshot :%s", err)
	}
	log.Printf("[INFO] Snapshot found: %v", snapshot)
	return nil
}

func findVM(d *schema.ResourceData, meta interface{}) (*object.VirtualMachine, error) {
	client := meta.(*govmomi.Client)
	var dc *object.Datacenter
	var err error
	if v, ok := d.GetOk("datacenter"); ok {
		dc, err = getDatacenter(client, v.(string))
	} else {
		dc, err = getDatacenter(client, "")
	}
	if err != nil {
		log.Printf("[ERROR] Error While getting the DC: %v", err)
		return nil, fmt.Errorf("Error While getting the DC: %s", err)
	}
	log.Printf("[INFO] DataCenter is: %v", dc)
	log.Println("[INFO] Getting Finder:")
	finder := find.NewFinder(client.Client, true)
	log.Printf("[INFO] Finder is: %v", finder)
	log.Println("[INFO] Setting DataCenter:")
	finder = finder.SetDatacenter(dc)
	log.Printf("[INFO] DataCenter is Set: %v", finder)
	log.Println("[INFO] Getting VM Object: ")
	vm, err := finder.VirtualMachine(context.TODO(), d.Get("vm_id").(string))
	if err != nil {
		log.Printf("[ERROR] Error While getting the Virtual machine object: %v", err)
		return nil, err
	}
	log.Printf("[INFO] Virtual Machine FOUND: %v", vm)
	return vm, nil
}
