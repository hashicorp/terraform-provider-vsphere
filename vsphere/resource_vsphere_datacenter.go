package vsphere

import (
	"fmt"
	"log"

	"github.com/hashicorp/terraform/helper/schema"
	"github.com/vmware/govmomi"
	"github.com/vmware/govmomi/find"
	"github.com/vmware/govmomi/object"
	"github.com/vmware/govmomi/vim25/methods"
	"github.com/vmware/govmomi/vim25/types"
	"golang.org/x/net/context"
)

func resourceVSphereDatacenter() *schema.Resource {
	return &schema.Resource{
		Create: resourceVSphereDatacenterCreate,
		Read:   resourceVSphereDatacenterRead,
		Delete: resourceVSphereDatacenterDelete,

		Schema: map[string]*schema.Schema{
			"name": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"folder": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
			},
		},
	}
}

func resourceVSphereDatacenterCreate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*govmomi.Client)
	name := d.Get("name").(string)

	var f *object.Folder
	if v, ok := d.GetOk("folder"); ok {
		finder := find.NewFinder(client.Client, true)
		var err error
		f, err = finder.Folder(context.TODO(), v.(string))
		if err != nil {
			return fmt.Errorf("[ERROR] Failed to find folder that will contain the datacenter: %s", err)
		}
	} else {
		f = object.NewRootFolder(client.Client)
	}

	dc, err := f.CreateDatacenter(context.TODO(), name)
	if err != nil || dc == nil {
		return fmt.Errorf("[ERROR] Failed to create datacenter: %s", err)
	}
	// From govmomi code: "Response will be nil if this is an ESX host that does not belong to a vCenter"
	if dc == nil {
		return fmt.Errorf("[ERROR] ESX host does not belong to a vCenter")
	}

	d.SetId(name)

	return nil

}

func resourceVSphereDatacenterRead(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*govmomi.Client)
	name := d.Get("name").(string)

	path := name
	if v, ok := d.GetOk("folder"); ok {
		path = v.(string) + "/" + name
	}

	finder := find.NewFinder(client.Client, true)
	_, err := finder.Datacenter(context.TODO(), path)
	if err != nil {
		log.Printf("[ERROR] Couldn't find the specified datacenter: %s", err)
		d.SetId("")
	}

	return nil
}

func resourceVSphereDatacenterDelete(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*govmomi.Client)
	name := d.Get("name").(string)

	path := name
	if v, ok := d.GetOk("folder"); ok {
		path = v.(string) + "/" + name
	}

	finder := find.NewFinder(client.Client, true)
	dc, err := finder.Datacenter(context.TODO(), path)
	if err != nil {
		log.Printf("[ERROR] Couldn't find the specified datacenter: %s", err)
		d.SetId("")
		return nil
	}

	req := &types.Destroy_Task{
		This: dc.Common.Reference(),
	}

	_, err = methods.Destroy_Task(context.TODO(), client, req)
	if err != nil {
		return fmt.Errorf("[ERROR] %s", err)
	}

	return nil
}
