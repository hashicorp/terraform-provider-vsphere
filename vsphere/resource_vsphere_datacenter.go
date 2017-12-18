package vsphere

import (
	"fmt"
	"log"
	"time"

	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/helper/schema"
	"github.com/terraform-providers/terraform-provider-vsphere/vsphere/internal/helper/customattribute"
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
		Update: resourceVSphereDatacenterUpdate,
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

			// Add tags schema
			vSphereTagAttributeKey: tagsSchema(),

			// Custom Attributes
			customattribute.ConfigKey: customattribute.ConfigSchema(),
		},
	}
}

func resourceVSphereDatacenterCreate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*VSphereClient).vimClient

	// Load up the tags client, which will validate a proper vCenter before
	// attempting to proceed if we have tags defined.
	tagsClient, err := tagsClientIfDefined(d, meta)
	if err != nil {
		return err
	}
	// Verify a proper vCenter before proceeding if custom attributes are defined
	attrsProcessor, err := customattribute.GetDiffProcessorIfAttributesDefined(client, d)
	if err != nil {
		return err
	}

	name := d.Get("name").(string)

	var f *object.Folder
	if v, ok := d.GetOk("folder"); ok {
		finder := find.NewFinder(client.Client, true)
		var err error
		f, err = finder.Folder(context.TODO(), v.(string))
		if err != nil {
			return fmt.Errorf("failed to find folder that will contain the datacenter: %s", err)
		}
	} else {
		f = object.NewRootFolder(client.Client)
	}

	dc, err := f.CreateDatacenter(context.TODO(), name)
	if err != nil || dc == nil {
		return fmt.Errorf("failed to create datacenter: %s", err)
	}
	// From govmomi code: "Response will be nil if this is an ESX host that does not belong to a vCenter"
	if dc == nil {
		return fmt.Errorf("ESX host does not belong to a vCenter")
	}

	// Wait for the datacenter resource to be ready
	stateConf := &resource.StateChangeConf{
		Pending:    []string{"InProgress"},
		Target:     []string{"Created"},
		Refresh:    resourceVSphereDatacenterStateRefreshFunc(d, meta),
		Timeout:    10 * time.Minute,
		MinTimeout: 3 * time.Second,
		Delay:      5 * time.Second,
	}

	_, err = stateConf.WaitForState()
	if err != nil {
		return fmt.Errorf("error waiting for datacenter (%s) to become ready: %s", name, err)
	}

	// Apply any pending tags now
	if tagsClient != nil {
		if err := processTagDiff(tagsClient, d, dc); err != nil {
			return err
		}
	}

	// Set custom attributes
	if attrsProcessor != nil {
		if err := attrsProcessor.ProcessDiff(dc); err != nil {
			return err
		}
	}

	d.SetId(name)

	return resourceVSphereDatacenterRead(d, meta)
}

func resourceVSphereDatacenterStateRefreshFunc(d *schema.ResourceData, meta interface{}) resource.StateRefreshFunc {
	return func() (interface{}, string, error) {
		log.Print("[DEBUG] Refreshing datacenter state")
		dc, err := datacenterExists(d, meta)
		if err != nil {
			switch err.(type) {
			case *find.NotFoundError:
				log.Printf("[DEBUG] Refreshing state. Datacenter not found: %s", err)
				return nil, "InProgress", nil
			default:
				return nil, "Failed", err
			}
		}
		log.Print("[DEBUG] Refreshing state. Datacenter found")
		return dc, "Created", nil
	}
}

func datacenterExists(d *schema.ResourceData, meta interface{}) (*object.Datacenter, error) {
	client := meta.(*VSphereClient).vimClient
	name := d.Get("name").(string)

	path := name
	if v, ok := d.GetOk("folder"); ok {
		path = v.(string) + "/" + name
	}

	finder := find.NewFinder(client.Client, true)
	dc, err := finder.Datacenter(context.TODO(), path)
	return dc, err
}

func resourceVSphereDatacenterRead(d *schema.ResourceData, meta interface{}) error {
	dc, err := datacenterExists(d, meta)
	if err != nil {
		log.Printf("couldn't find the specified datacenter: %s", err)
		d.SetId("")
		return nil
	}

	// Read tags if we have the ability to do so
	if tagsClient, _ := meta.(*VSphereClient).TagsClient(); tagsClient != nil {
		if err := readTagsForResource(tagsClient, dc, d); err != nil {
			return err
		}
	}

	// Read set custom attributes
	client := meta.(*VSphereClient).vimClient
	if customattribute.IsSupported(client) {
		moDc, err := datacenterCustomAttributes(dc)
		if err != nil {
			return err
		}
		customattribute.ReadFromResource(client, moDc.Entity(), d)
	}

	return nil
}

func resourceVSphereDatacenterUpdate(d *schema.ResourceData, meta interface{}) error {
	// Load up the tags client, which will validate a proper vCenter before
	// attempting to proceed if we have tags defined.
	tagsClient, err := tagsClientIfDefined(d, meta)
	if err != nil {
		return err
	}
	// Verify a proper vCenter before proceeding if custom attributes are defined
	client := meta.(*VSphereClient).vimClient
	attrsProcessor, err := customattribute.GetDiffProcessorIfAttributesDefined(client, d)
	if err != nil {
		return err
	}

	dc, err := datacenterExists(d, meta)
	if err != nil {
		return fmt.Errorf("couldn't find the specified datacenter: %s", err)
	}

	// Apply any pending tags now
	if tagsClient != nil {
		if err := processTagDiff(tagsClient, d, dc); err != nil {
			return err
		}
	}

	// Set custom attributes
	if attrsProcessor != nil {
		if err := attrsProcessor.ProcessDiff(dc); err != nil {
			return err
		}
	}

	return nil
}

func resourceVSphereDatacenterDelete(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*VSphereClient).vimClient
	name := d.Get("name").(string)

	path := name
	if v, ok := d.GetOk("folder"); ok {
		path = v.(string) + "/" + name
	}

	finder := find.NewFinder(client.Client, true)
	dc, err := finder.Datacenter(context.TODO(), path)
	if err != nil {
		log.Printf("couldn't find the specified datacenter: %s", err)
		d.SetId("")
		return nil
	}

	req := &types.Destroy_Task{
		This: dc.Common.Reference(),
	}

	_, err = methods.Destroy_Task(context.TODO(), client, req)
	if err != nil {
		return fmt.Errorf("%s", err)
	}

	// Wait for the datacenter resource to be destroyed
	stateConf := &resource.StateChangeConf{
		Pending:    []string{"Created"},
		Target:     []string{},
		Refresh:    resourceVSphereDatacenterStateRefreshFunc(d, meta),
		Timeout:    10 * time.Minute,
		MinTimeout: 3 * time.Second,
		Delay:      5 * time.Second,
	}

	_, err = stateConf.WaitForState()
	if err != nil {
		return fmt.Errorf("error waiting for datacenter (%s) to become ready: %s", name, err)
	}

	return nil
}
