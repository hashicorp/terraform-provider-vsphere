// © Broadcom. All Rights Reserved.
// The term "Broadcom" refers to Broadcom Inc. and/or its subsidiaries.
// SPDX-License-Identifier: MPL-2.0

package vsphere

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"path"
	"strconv"
	"strings"
	"sync"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/vmware/govmomi"
	"github.com/vmware/govmomi/find"
	"github.com/vmware/govmomi/object"
	"github.com/vmware/govmomi/vim25/types"
	"github.com/vmware/terraform-provider-vsphere/vsphere/internal/helper/virtualdisk"
)

var vsphereVirtualDiskMakeDirectoryMutex sync.Mutex

type virtualDisk struct {
	size              int
	vmdkPath          string
	initType          string
	adapterType       string
	datacenter        string
	datastore         string
	createDirectories bool
}

// Define VirtualDisk args
func resourceVSphereVirtualDisk() *schema.Resource {
	return &schema.Resource{
		Create: resourceVSphereVirtualDiskCreate,
		Read:   resourceVSphereVirtualDiskRead,
		Update: resourceVSphereVirtualDiskUpdate,
		Delete: resourceVSphereVirtualDiskDelete,
		Importer: &schema.ResourceImporter{
			State: resourceVSphereVirtualDiskImport,
		},

		Schema: map[string]*schema.Schema{
			// Size in GB
			"size": {
				Type:     schema.TypeInt,
				Required: true,
			},

			// TODO:
			//
			// * Add extra lifecycles (move, rename, etc). May not be possible
			// without breaking other resources though.
			// * Add validation (make sure it ends in .vmdk)
			"vmdk_path": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
				ValidateFunc: func(v interface{}, _ string) (warns []string, errors []error) {
					if !strings.HasSuffix(v.(string), ".vmdk") {
						errors = append(errors, fmt.Errorf("vmdk_path must end with '.vmdk'"))
					}
					return
				},
			},

			"datastore": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"type": {
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
				Default:  "eagerZeroedThick",
				ValidateFunc: func(v interface{}, _ string) (ws []string, errors []error) {
					value := v.(string)
					if value != "thin" && value != "eagerZeroedThick" && value != "lazy" {
						errors = append(errors, fmt.Errorf(
							"only 'thin', 'eagerZeroedThick', and 'lazy' are supported values for 'type'"))
					}
					return
				},
			},

			"adapter_type": {
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
				Default:  "lsiLogic",
				// TODO: Move this to removed after we remove the support to specify this in later versions
				Deprecated: "this attribute has no effect on controller types - please use scsi_type in vsphere_virtual_machine instead",
				ValidateFunc: func(v interface{}, _ string) (ws []string, errors []error) {
					value := v.(string)
					if value != "ide" && value != "busLogic" && value != "lsiLogic" {
						errors = append(errors, fmt.Errorf(
							"only 'ide', 'busLogic', and 'lsiLogic' are supported values for 'adapter_type'"))
					}
					return
				},
			},

			"datacenter": {
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
			},

			"create_directories": {
				Type:     schema.TypeBool,
				Optional: true,
				ForceNew: true,
			},
		},
	}
}

func resourceVSphereVirtualDiskCreate(d *schema.ResourceData, meta interface{}) error {
	log.Printf("[INFO] Creating Virtual Disk")
	client := meta.(*Client).vimClient

	vDisk := virtualDisk{
		size: d.Get("size").(int),
	}

	if v, ok := d.GetOk("vmdk_path"); ok {
		vDisk.vmdkPath = v.(string)
	}

	if v, ok := d.GetOk("type"); ok {
		vDisk.initType = v.(string)
	}

	if v, ok := d.GetOk("adapter_type"); ok {
		vDisk.adapterType = v.(string)
	}

	if v, ok := d.GetOk("datacenter"); ok {
		vDisk.datacenter = v.(string)
	}

	if v, ok := d.GetOk("datastore"); ok {
		vDisk.datastore = v.(string)
	}

	if v, ok := d.GetOk("create_directories"); ok {
		vDisk.createDirectories = v.(bool)
	}

	finder := find.NewFinder(client.Client, true)

	dc, err := getDatacenter(client, d.Get("datacenter").(string))
	if err != nil {
		return fmt.Errorf("error finding datacenter: %s: %s", vDisk.datacenter, err)
	}
	finder = finder.SetDatacenter(dc)

	ds, err := getDatastore(finder, vDisk.datastore)
	if err != nil {
		return fmt.Errorf("error finding datastore: %s: %s", vDisk.datastore, err)
	}

	fm := object.NewFileManager(client.Client)

	if vDisk.createDirectories {
		directoryPathIndex := strings.LastIndex(vDisk.vmdkPath, "/")
		if directoryPathIndex > 0 {
			// Only allow one MakeDirectory operation at a time in order to avoid
			// overlapping attempts to create the same directory, which can result
			// in some of the attempts failing.
			vsphereVirtualDiskMakeDirectoryMutex.Lock()
			vmdkPath := vDisk.vmdkPath[0:directoryPathIndex]
			log.Printf("[DEBUG] Creating parent directories: %v", ds.Path(vmdkPath))
			err = fm.MakeDirectory(context.TODO(), ds.Path(vmdkPath), dc, true)
			vsphereVirtualDiskMakeDirectoryMutex.Unlock()
			if err != nil && !isAlreadyExists(err) {
				log.Printf("[DEBUG] Failed to create parent directories:  %v", err)
				return err
			}

			err = searchForDirectory(client, vDisk.datacenter, vDisk.datastore, vmdkPath)
			if err != nil {
				log.Printf("[DEBUG] Failed to find newly created parent directories:  %v", err)
				return err
			}
		}
	}

	err = createHardDisk(client, vDisk.size, ds.Path(vDisk.vmdkPath), vDisk.initType, vDisk.adapterType, vDisk.datacenter)
	if err != nil {
		return err
	}

	d.SetId(ds.Path(vDisk.vmdkPath))
	log.Printf("[DEBUG] Virtual Disk id: %v", ds.Path(vDisk.vmdkPath))

	return resourceVSphereVirtualDiskRead(d, meta)
}

func resourceVSphereVirtualDiskRead(d *schema.ResourceData, meta interface{}) error {
	log.Printf("[DEBUG] Reading virtual disk.")
	client := meta.(*Client).vimClient

	vDisk := virtualDisk{
		size: d.Get("size").(int),
	}

	if v, ok := d.GetOk("vmdk_path"); ok {
		vDisk.vmdkPath = v.(string)
	}

	if v, ok := d.GetOk("type"); ok {
		vDisk.initType = v.(string)
	}

	if v, ok := d.GetOk("adapter_type"); ok {
		vDisk.adapterType = v.(string)
	}

	if v, ok := d.GetOk("datacenter"); ok {
		vDisk.datacenter = v.(string)
	}

	if v, ok := d.GetOk("datastore"); ok {
		vDisk.datastore = v.(string)
	}

	dc, err := getDatacenter(client, d.Get("datacenter").(string))
	if err != nil {
		return err
	}

	finder := find.NewFinder(client.Client, true)
	finder = finder.SetDatacenter(dc)

	ds, err := finder.Datastore(context.TODO(), d.Get("datastore").(string))
	if err != nil {
		return err
	}

	ctx := context.TODO()
	b, err := ds.Browser(ctx)
	if err != nil {
		return err
	}

	// `Datastore.Stat` does not allow to query `VmDiskFileQuery`. Instead, we
	// search the datastore manually.
	spec := types.HostDatastoreBrowserSearchSpec{
		Query: []types.BaseFileQuery{&types.VmDiskFileQuery{Details: &types.VmDiskFileQueryFlags{
			CapacityKb: true,
			DiskType:   true,
		}}},
		Details: &types.FileQueryFlags{
			FileSize:     true,
			FileType:     true,
			Modification: true,
			FileOwner:    types.NewBool(true),
		},
		MatchPattern: []string{path.Base(vDisk.vmdkPath)},
	}

	dsPath := ds.Path(path.Dir(vDisk.vmdkPath))
	task, err := b.SearchDatastore(context.TODO(), dsPath, &spec)

	if err != nil {
		log.Printf("[DEBUG] resourceVSphereVirtualDiskRead - could not search datastore for: %v", vDisk.vmdkPath)
		return err
	}

	info, err := task.WaitForResultEx(context.TODO(), nil)
	if err != nil {
		if info != nil && info.Error != nil {
			_, ok := info.Error.Fault.(*types.FileNotFound)
			if ok {
				log.Printf("[DEBUG] resourceVSphereVirtualDiskRead - could not find: %v", vDisk.vmdkPath)
				d.SetId("")
				return nil
			}
		}

		log.Printf("[DEBUG] resourceVSphereVirtualDiskRead - could not search datastore for: %v", vDisk.vmdkPath)
		return err
	}

	res := info.Result.(types.HostDatastoreBrowserSearchResults)
	log.Printf("[DEBUG] num results: %d", len(res.File))
	if len(res.File) == 0 {
		d.SetId("")
		log.Printf("[DEBUG] resourceVSphereVirtualDiskRead - could not find: %v", vDisk.vmdkPath)
		return nil
	}

	if len(res.File) != 1 {
		return errors.New("datastore search did not return exactly one result")
	}

	fileInfo := res.File[0]
	log.Printf("[DEBUG] resourceVSphereVirtualDiskRead - fileinfo: %#v", fileInfo)
	size := fileInfo.(*types.VmDiskFileInfo).CapacityKb / 1024 / 1024

	dp := object.DatastorePath{
		Datastore: vDisk.datastore,
		Path:      vDisk.vmdkPath,
	}
	diskType, err := virtualdisk.QueryDiskType(client, dp.String(), dc)
	/**
	Thick Provisioned Lazy Zeroed disk type a.k.a "lazy" in the provider context is actually
	"preallocated" for the VC. Due to historical reasons i.e. the disk type is documented as "lazy".
	In order to fix https://github.com/vmware/terraform-provider-vsphere/issues/1824 the value must be converted,
	otherwise the disk is recreated
	*/
	if diskType == "preallocated" {
		diskType = "lazy"
	}

	if err != nil {
		return errors.New("failed to query disk type")
	}

	// adapter_type is deprecated, so just default.
	rs := resourceVSphereVirtualDisk().Schema
	_ = d.Set("adapter_type", rs["adapter_type"].Default)

	d.SetId(vDisk.vmdkPath)

	_ = d.Set("size", size)
	_ = d.Set("type", diskType)
	_ = d.Set("vmdk_path", vDisk.vmdkPath)
	_ = d.Set("datacenter", d.Get("datacenter"))
	_ = d.Set("datastore", d.Get("datastore"))

	return nil
}

func resourceVSphereVirtualDiskUpdate(d *schema.ResourceData, meta interface{}) error {
	log.Printf("[INFO] Updating Virtual Disk")
	client := meta.(*Client).vimClient

	oldSize, newSize := d.GetChange("size")
	if newSize.(int) < oldSize.(int) {
		return fmt.Errorf("shrinking a virtual disk is not supported")
	}

	vDisk := virtualDisk{
		size: d.Get("size").(int),
	}

	if v, ok := d.GetOk("vmdk_path"); ok {
		vDisk.vmdkPath = v.(string)
	}

	if v, ok := d.GetOk("datastore"); ok {
		vDisk.datastore = v.(string)
	}

	if v, ok := d.GetOk("datacenter"); ok {
		vDisk.datacenter = v.(string)
	}

	finder := find.NewFinder(client.Client, true)

	dc, err := getDatacenter(client, d.Get("datacenter").(string))
	if err != nil {
		return fmt.Errorf("error finding Datacenter: %s: %s", vDisk.datacenter, err)
	}
	finder = finder.SetDatacenter(dc)

	ds, err := getDatastore(finder, vDisk.datastore)
	if err != nil {
		return fmt.Errorf("error finding Datastore: %s: %s", vDisk.datastore, err)
	}

	if err := extendHardDisk(client, vDisk.size, ds.Path(vDisk.vmdkPath), vDisk.datacenter); err != nil {
		return err
	}

	return resourceVSphereVirtualDiskRead(d, meta)
}

func resourceVSphereVirtualDiskDelete(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*Client).vimClient

	vDisk := virtualDisk{}

	if v, ok := d.GetOk("vmdk_path"); ok {
		vDisk.vmdkPath = v.(string)
	}
	if v, ok := d.GetOk("datastore"); ok {
		vDisk.datastore = v.(string)
	}

	dc, err := getDatacenter(client, d.Get("datacenter").(string))
	if err != nil {
		return err
	}

	finder := find.NewFinder(client.Client, true)
	finder = finder.SetDatacenter(dc)

	ds, err := getDatastore(finder, vDisk.datastore)
	if err != nil {
		return err
	}

	diskPath := ds.Path(vDisk.vmdkPath)

	virtualDiskManager := object.NewVirtualDiskManager(client.Client)

	task, err := virtualDiskManager.DeleteVirtualDisk(context.TODO(), diskPath, dc)
	if err != nil {
		return err
	}

	_, err = task.WaitForResultEx(context.TODO(), nil)
	if err != nil {
		log.Printf("[INFO] Failed to delete disk:  %v", err)
		return err
	}

	log.Printf("[INFO] Deleted disk: %v", diskPath)
	d.SetId("")
	return nil
}

func isAlreadyExists(err error) bool {
	return strings.HasPrefix(err.Error(), "Cannot complete the operation because the file or folder") &&
		strings.HasSuffix(err.Error(), "already exists")
}

// createHardDisk creates a new Hard Disk.
func createHardDisk(client *govmomi.Client, size int, diskPath string, diskType string, adapterType string, dc string) error {
	var vDiskType string
	switch diskType {
	case "thin":
		vDiskType = "thin"
	case "eagerZeroedThick":
		vDiskType = "eagerZeroedThick"
	case "lazy":
		vDiskType = "preallocated"
	}

	virtualDiskManager := object.NewVirtualDiskManager(client.Client)
	spec := &types.FileBackedVirtualDiskSpec{
		VirtualDiskSpec: types.VirtualDiskSpec{
			AdapterType: adapterType,
			DiskType:    vDiskType,
		},
		CapacityKb: int64(1024 * 1024 * size),
	}
	datacenter, err := getDatacenter(client, dc)
	if err != nil {
		return err
	}
	log.Printf("[DEBUG] Disk spec: %v", spec)

	task, err := virtualDiskManager.CreateVirtualDisk(context.TODO(), diskPath, datacenter, spec)
	if err != nil {
		return err
	}

	_, err = task.WaitForResultEx(context.TODO(), nil)
	if err != nil {
		log.Printf("[INFO] Failed to create disk:  %v", err)
		return err
	}
	log.Printf("[INFO] Created disk.")

	return nil
}

func extendHardDisk(client *govmomi.Client, capacity int, diskPath string, dc string) error {
	virtualDiskManager := object.NewVirtualDiskManager(client.Client)
	datacenter, err := getDatacenter(client, dc)
	if err != nil {
		return err
	}

	capacityKb := int64(1024 * 1024 * capacity)
	task, err := virtualDiskManager.ExtendVirtualDisk(context.TODO(), diskPath, datacenter, capacityKb, nil)
	if err != nil {
		return err
	}

	_, err = task.WaitForResultEx(context.TODO(), nil)
	if err != nil {
		log.Printf("[INFO] Failed to extend disk:  %v", err)
		return err
	}
	log.Printf("[INFO] Extended disk.")

	return nil
}

// Searches for the presence of a directory path.
func searchForDirectory(client *govmomi.Client, datacenter string, datastore string, directoryPath string) error {
	log.Printf("[DEBUG] Searching for Directory")
	finder := find.NewFinder(client.Client, true)

	dc, err := getDatacenter(client, datacenter)
	if err != nil {
		return fmt.Errorf("error finding datacenter: %s: %s", datacenter, err)
	}
	finder = finder.SetDatacenter(dc)

	ds, err := finder.Datastore(context.TODO(), datastore)
	if err != nil {
		return fmt.Errorf("error finding datastore: %s: %s", datastore, err)
	}

	ctx := context.TODO()
	b, err := ds.Browser(ctx)
	if err != nil {
		return err
	}

	spec := types.HostDatastoreBrowserSearchSpec{
		Query: []types.BaseFileQuery{&types.FolderFileQuery{}},
		Details: &types.FileQueryFlags{
			FileSize:     true,
			FileType:     true,
			Modification: true,
			FileOwner:    types.NewBool(true),
		},
		MatchPattern: []string{path.Base(directoryPath)},
	}

	dsPath := ds.Path(path.Dir(directoryPath))
	task, err := b.SearchDatastore(context.TODO(), dsPath, &spec)

	if err != nil {
		log.Printf("[DEBUG] searchForDirectory - could not search datastore for: %v", directoryPath)
		return err
	}

	info, err := task.WaitForResultEx(context.TODO(), nil)
	if err != nil {
		if info != nil && info.Error != nil {
			_, ok := info.Error.Fault.(*types.FileNotFound)
			if ok {
				log.Printf("[DEBUG] searchForDirectory - could not find: %v", directoryPath)
				return nil
			}
		}

		log.Printf("[DEBUG] searchForDirectory - could not search datastore for: %v", directoryPath)
		return err
	}

	res := info.Result.(types.HostDatastoreBrowserSearchResults)
	log.Printf("[DEBUG] num results: %d", len(res.File))
	if len(res.File) == 0 {
		log.Printf("[DEBUG] searchForDirectory - could not find: %v", directoryPath)
		return nil
	}

	if len(res.File) != 1 {
		return errors.New("datastore search did not return exactly one result")
	}

	fileInfo := res.File[0]
	log.Printf("[DEBUG] searchForDirectory - fileinfo: %#v", fileInfo)

	return nil
}

func resourceVSphereVirtualDiskImport(d *schema.ResourceData, meta interface{}) ([]*schema.ResourceData, error) {
	client := meta.(*Client).vimClient
	var data map[string]string
	if err := json.Unmarshal([]byte(d.Id()), &data); err != nil {
		return nil, err
	}
	createDirectories, ok := data["create_directories"]
	if ok {
		createDirectoriesBool, _ := strconv.ParseBool(createDirectories)
		log.Printf("[INFO] Set create_directories during import: %v", createDirectoriesBool)
		_ = d.Set("create_directories", createDirectoriesBool)
	}

	p, ok := data["virtual_disk_path"]
	if !ok {
		return nil, errors.New("missing virtual_disk_path in input data")
	}
	if !strings.HasPrefix(p, "/") {
		return nil, errors.New("ID must start with a trailing slash")
	}

	// The path in the ID has the form `/<datacenter>/[<datastore>] path/to/vmdk`.
	// Note that the values we care about in addrParts start at the first element as
	// the zero-th element will be empty (that is, everything before the / prefix).
	addrParts := strings.SplitN(p, "/", 3)
	if len(addrParts) != 3 {
		return nil, errors.New("ID must be of the form /<datacenter>/[<datastore>] path/to/vmdk")
	}

	dc, err := getDatacenter(client, addrParts[1])
	if err != nil {
		return nil, err
	}

	di, err := virtualdisk.FromPath(client, addrParts[2], dc)
	if err != nil {
		return nil, err
	}

	dp, success := virtualdisk.DatastorePathFromString(di.Name)
	if !success {
		return nil, fmt.Errorf("invalid datastore path '%s'", di.Name)
	}

	//addrParts[2] is in form: [<datastore>]path/to/vmdk
	vmdkPath := strings.Split(addrParts[2], "]")[1]
	_ = d.Set("datacenter", dc.Name())
	_ = d.Set("datastore", dp.Datastore)
	_ = d.Set("vmdk_path", vmdkPath)
	d.SetId(vmdkPath)

	return []*schema.ResourceData{d}, nil
}
