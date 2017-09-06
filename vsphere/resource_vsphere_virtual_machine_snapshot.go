package vsphere

import (
	"context"
	"fmt"
	"log"
	"strings"

	"github.com/hashicorp/terraform/helper/schema"
	"github.com/vmware/govmomi"
	"github.com/vmware/govmomi/vim25/types"
)

func resourceVSphereSnapshot() *schema.Resource {
	return &schema.Resource{
		Create: resourceVSphereSnapshotCreate,
		Read:   resourceVSphereSnapshotRead,
		Delete: resourceVSphereSnapshotDelete,

		Schema: map[string]*schema.Schema{
			"vm_uuid": {
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
	client := meta.(*govmomi.Client)
	vm, err := virtualMachineFromUUID(client, d.Get("vm_uuid").(string))
	if err != nil {
		return fmt.Errorf("Error while getting the VirtualMachine :%s", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), defaultAPITimeout) // This is 5 mins
	defer cancel()
	task, err := vm.CreateSnapshot(ctx, d.Get("snapshot_name").(string), d.Get("description").(string), d.Get("memory").(bool), d.Get("quiesce").(bool))
	tctx, tcancel := context.WithTimeout(context.Background(), defaultAPITimeout)
	defer tcancel()
	taskInfo, err := task.WaitForResult(tctx, nil)
	if err != nil {
		log.Printf("[DEBUG] Error While Creating the Task for Create Snapshot: %v", err)
		return fmt.Errorf(" Error While Creating the Task for Create Snapshot: %s", err)
	}
	log.Printf("[DEBUG] Task created for Create Snapshot: %v", task)
	if err != nil {
		log.Printf("[DEBUG] Error While waiting for the Task for Create Snapshot: %v", err)
		return fmt.Errorf(" Error While waiting for the Task for Create Snapshot: %s", err)
	}
	log.Printf("[DEBUG] Create Snapshot completed %v", d.Get("snapshot_name").(string))
	log.Println("[DEBUG] Managed Object Reference: " + taskInfo.Result.(types.ManagedObjectReference).Value)
	d.SetId(taskInfo.Result.(types.ManagedObjectReference).Value)
	return nil
}

func resourceVSphereSnapshotDelete(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*govmomi.Client)
	vm, err := virtualMachineFromUUID(client, d.Get("vm_uuid").(string))
	if err != nil {
		return fmt.Errorf("Error while getting the VirtualMachine :%s", err)
	}
	resourceVSphereSnapshotRead(d, meta)
	if d.Id() == "" {
		log.Printf("[DEBUG] Error While finding the Snapshot: %v", err)
		return nil
	}
	log.Printf("[DEBUG] Deleting snapshot with name: %v", d.Get("snapshot_name").(string))
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
	ctx, cancel := context.WithTimeout(context.Background(), defaultAPITimeout) // This is 5 mins
	defer cancel()
	task, err := vm.RemoveSnapshot(ctx, d.Id(), remove_children, consolidate_ptr)
	if err != nil {
		log.Printf("[DEBUG] Error While Creating the Task for Delete Snapshot: %v", err)
		return fmt.Errorf("Error While Creating the Task for Delete Snapshot: %s", err)
	}
	log.Printf("[DEBUG] Task created for Delete Snapshot: %v", task)

	err = task.Wait(ctx)
	if err != nil {
		log.Printf("[DEBUG] Error While waiting for the Task of Delete Snapshot: %v", err)
		return fmt.Errorf("Error While waiting for the Task of Delete Snapshot: %s", err)
	}
	log.Printf("[DEBUG] Delete Snapshot completed %v", d.Get("snapshot_name").(string))

	return nil
}

func resourceVSphereSnapshotRead(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*govmomi.Client)
	vm, err := virtualMachineFromUUID(client, d.Get("vm_uuid").(string))
	if err != nil {
		return fmt.Errorf("Error while getting the VirtualMachine :%s", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), defaultAPITimeout) // This is 5 mins
	defer cancel()
	snapshot, err := vm.FindSnapshot(ctx, d.Id())
	if err != nil {
		if strings.Contains(err.Error(), "No snapshots for this VM") || strings.Contains(err.Error(), "snapshot \""+d.Get("snapshot_name").(string)+"\" not found") {
			log.Printf("[DEBUG] Error While finding the Snapshot: %v", err)
			d.SetId("")
			return nil
		}
		log.Printf("[DEBUG] Error While finding the Snapshot: %v", err)
		return fmt.Errorf("Error while finding the Snapshot :%s", err)
	}
	log.Printf("[DEBUG] Snapshot found: %v", snapshot)
	return nil
}
