package vsphere

import (
	"errors"
	"fmt"
	"log"
	"net"
	"strconv"
	"strings"

	"github.com/hashicorp/terraform/helper/schema"
	"github.com/vmware/govmomi"
	"github.com/vmware/govmomi/find"
	"github.com/vmware/govmomi/object"
	"github.com/vmware/govmomi/property"
	"github.com/vmware/govmomi/vim25/mo"
	"github.com/vmware/govmomi/vim25/types"
	"golang.org/x/net/context"
)

var DefaultDNSSuffixes = []string{
	"vsphere.local",
}

var DefaultDNSServers = []string{
	"8.8.8.8",
	"8.8.4.4",
}

var DiskControllerTypes = []string{
	"scsi",
	"scsi-lsi-parallel",
	"scsi-buslogic",
	"scsi-paravirtual",
	"scsi-lsi-sas",
	"ide",
}

type networkInterface struct {
	deviceName       string
	label            string
	key              string
	ipv4Address      string
	ipv4PrefixLength int
	ipv4Gateway      string
	ipv6Address      string
	ipv6PrefixLength int
	ipv6Gateway      string
	adapterType      string
	macAddress       string
}

type hardDisk struct {
	name       string
	size       int64
	iops       int64
	initType   string
	vmdkPath   string
	controller string
	bootable   bool
}

//Additional options Vsphere can use clones of windows machines
type windowsOptConfig struct {
	productKey         string
	adminPassword      string
	domainUser         string
	domain             string
	domainUserPassword string
}

type cdrom struct {
	datastore string
	path      string
}

type memoryAllocation struct {
	reservation int64
}

type virtualMachine struct {
	name                  string
	hostname              string
	guestId               string
	folder                string
	rebootAllowed         bool
	datacenter            string
	cluster               string
	resourcePool          string
	datastore             string
	vcpu                  int32
	cpuHotAddEnabled      bool
	memoryMb              int64
	memoryAllocation      memoryAllocation
	annotation            string
	memoryHotAddEnabled   bool
	template              string
	networkInterfaces     []networkInterface
	hardDisks             []hardDisk
	cdroms                []cdrom
	domain                string
	timeZone              string
	dnsSuffixes           []string
	dnsServers            []string
	hasBootableVmdk       bool
	linkedClone           bool
	skipCustomization     bool
	enableDiskUUID        bool
	moid                  string
	windowsOptionalConfig windowsOptConfig
	customConfigurations  map[string](types.AnyType)
}

func (v virtualMachine) Path() string {
	return vmPath(v.folder, v.name)
}

func vmPath(folder string, name string) string {
	var path string
	if len(folder) > 0 {
		path += folder + "/"
	}
	return path + name
}

func resourceVSphereVirtualMachine() *schema.Resource {
	return &schema.Resource{
		Create: resourceVSphereVirtualMachineCreate,
		Read:   resourceVSphereVirtualMachineRead,
		Update: resourceVSphereVirtualMachineUpdate,
		Delete: resourceVSphereVirtualMachineDelete,

		SchemaVersion: 1,
		MigrateState:  resourceVSphereVirtualMachineMigrateState,

		Schema: map[string]*schema.Schema{
			"name": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: false,
			},
			"hostname": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: false,
			},
			"guest_id": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: false,
			},
			"folder": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
			},
			"reboot_allowed": &schema.Schema{
				Type:     schema.TypeBool,
				Required: false,
				Optional: true,
				Default:  false,
				ForceNew: false,
			},
			"vcpu": &schema.Schema{
				Type:     schema.TypeInt,
				Required: true,
			},
			"cpu_hot_add_enabled": &schema.Schema{
				Type:     schema.TypeBool,
				Required: false,
				Optional: true,
				Default:  true,
				ForceNew: false,
			},
			"memory": &schema.Schema{
				Type:     schema.TypeInt,
				Required: true,
			},
			"memory_reservation": &schema.Schema{
				Type:     schema.TypeInt,
				Optional: true,
				Default:  0,
				ForceNew: false,
			},
			"memory_hot_add_enabled": &schema.Schema{
				Type:     schema.TypeBool,
				Required: false,
				Optional: true,
				Default:  true,
				ForceNew: false,
			},
			"annotation": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
			},
			"datacenter": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
			},
			"cluster": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: false,
			},
			"resource_pool": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: false,
			},
			"linked_clone": &schema.Schema{
				Type:     schema.TypeBool,
				Optional: true,
				Default:  false,
				ForceNew: true,
			},
			"gateway": &schema.Schema{
				Type:       schema.TypeString,
				Optional:   true,
				ForceNew:   false,
				Deprecated: "Please use network_interface.ipv4_gateway",
			},
			"domain": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: false,
				Default:  "vsphere.local",
			},
			"time_zone": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
				Default:  "Etc/UTC",
			},
			"dns_suffixes": &schema.Schema{
				Type:     schema.TypeList,
				Optional: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
				ForceNew: false,
			},
			"dns_servers": &schema.Schema{
				Type:     schema.TypeList,
				Optional: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
				ForceNew: false,
			},
			"skip_customization": &schema.Schema{
				Type:     schema.TypeBool,
				Optional: true,
				ForceNew: true,
				Default:  false,
			},
			"wait_for_guest_net": &schema.Schema{
				Type:     schema.TypeBool,
				Optional: true,
				Default:  true,
			},
			"enable_disk_uuid": &schema.Schema{
				Type:     schema.TypeBool,
				Optional: true,
				ForceNew: true,
				Default:  false,
			},
			"uuid": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},
			"moid": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},
			"custom_configuration_parameters": &schema.Schema{
				Type:     schema.TypeMap,
				Optional: true,
				ForceNew: true,
			},
			"windows_opt_config": &schema.Schema{
				Type:     schema.TypeList,
				Optional: true,
				ForceNew: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"product_key": &schema.Schema{
							Type:     schema.TypeString,
							Optional: true,
							ForceNew: true,
						},
						"admin_password": &schema.Schema{
							Type:     schema.TypeString,
							Optional: true,
							ForceNew: true,
						},
						"domain_user": &schema.Schema{
							Type:     schema.TypeString,
							Optional: true,
							ForceNew: true,
						},
						"domain": &schema.Schema{
							Type:     schema.TypeString,
							Optional: true,
							ForceNew: true,
						},
						"domain_user_password": &schema.Schema{
							Type:     schema.TypeString,
							Optional: true,
							ForceNew: true,
						},
					},
				},
			},
			"network_interface": &schema.Schema{
				Type:     schema.TypeList,
				Required: true,
				ForceNew: false,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"label": &schema.Schema{
							Type:     schema.TypeString,
							Required: true,
							ForceNew: false,
						},
						"key": &schema.Schema{
							Type:     schema.TypeInt,
							Computed: true,
						},
						"ip_address": &schema.Schema{
							Type:       schema.TypeString,
							Optional:   true,
							Computed:   true,
							Deprecated: "Please use ipv4_address",
						},
						"subnet_mask": &schema.Schema{
							Type:       schema.TypeString,
							Optional:   true,
							Computed:   true,
							Deprecated: "Please use ipv4_prefix_length",
						},
						"ipv4_address": &schema.Schema{
							Type:             schema.TypeString,
							Optional:         true,
							Computed:         true,
							DiffSuppressFunc: suppressIpDifferences,
						},
						"ipv4_prefix_length": &schema.Schema{
							Type:     schema.TypeInt,
							Optional: true,
							Computed: true,
						},
						"ipv4_gateway": &schema.Schema{
							Type:             schema.TypeString,
							Optional:         true,
							Computed:         true,
							DiffSuppressFunc: suppressIpDifferences,
						},
						"ipv6_address": &schema.Schema{
							Type:             schema.TypeString,
							Optional:         true,
							Computed:         true,
							DiffSuppressFunc: suppressIpDifferences,
						},
						"ipv6_prefix_length": &schema.Schema{
							Type:     schema.TypeInt,
							Optional: true,
							Computed: true,
						},
						"ipv6_gateway": &schema.Schema{
							Type:             schema.TypeString,
							Optional:         true,
							Computed:         true,
							DiffSuppressFunc: suppressIpDifferences,
						},
						"adapter_type": &schema.Schema{
							Type:     schema.TypeString,
							Required: true,
							ForceNew: false,
						},
						"mac_address": &schema.Schema{
							Type:     schema.TypeString,
							Optional: true,
							Computed: true,
						},
					},
				},
			},
			"disk": &schema.Schema{
				Type:     schema.TypeSet,
				Required: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"uuid": &schema.Schema{
							Type:     schema.TypeString,
							Computed: true,
						},
						"key": &schema.Schema{
							Type:     schema.TypeInt,
							Computed: true,
						},
						"template": &schema.Schema{
							Type:     schema.TypeString,
							Optional: true,
						},
						"type": &schema.Schema{
							Type:     schema.TypeString,
							Optional: true,
							Default:  "eager_zeroed",
							ValidateFunc: func(v interface{}, k string) (ws []string, errors []error) {
								value := v.(string)
								if value != "thin" && value != "eager_zeroed" && value != "lazy" {
									errors = append(errors, fmt.Errorf(
										"only 'thin', 'eager_zeroed', and 'lazy' are supported values for 'type'"))
								}
								return
							},
						},
						"datastore": &schema.Schema{
							Type:     schema.TypeString,
							Optional: true,
						},
						"size": &schema.Schema{
							Type:     schema.TypeInt,
							Optional: true,
						},
						"name": &schema.Schema{
							Type:     schema.TypeString,
							Optional: true,
						},
						"iops": &schema.Schema{
							Type:     schema.TypeInt,
							Optional: true,
						},
						"vmdk": &schema.Schema{
							// TODO: Add ValidateFunc to confirm path exists
							Type:     schema.TypeString,
							Optional: true,
						},
						"bootable": &schema.Schema{
							Type:     schema.TypeBool,
							Optional: true,
						},
						"keep_on_remove": &schema.Schema{
							Type:     schema.TypeBool,
							Optional: true,
						},
						"controller_type": &schema.Schema{
							Type:     schema.TypeString,
							Optional: true,
							Default:  "scsi",
							ValidateFunc: func(v interface{}, k string) (ws []string, errors []error) {
								value := v.(string)
								found := false
								for _, t := range DiskControllerTypes {
									if t == value {
										found = true
									}
								}
								if !found {
									errors = append(errors, fmt.Errorf(
										"Supported values for 'controller_type' are %v", strings.Join(DiskControllerTypes, ", ")))
								}
								return
							},
						},
					},
				},
			},
			"detach_unknown_disks_on_delete": &schema.Schema{
				Type:     schema.TypeBool,
				Optional: true,
				Default:  false,
			},
			"cdrom": &schema.Schema{
				Type:     schema.TypeList,
				Optional: true,
				ForceNew: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"datastore": &schema.Schema{
							Type:     schema.TypeString,
							Required: true,
							ForceNew: true,
						},
						"path": &schema.Schema{
							Type:     schema.TypeString,
							Required: true,
							ForceNew: true,
						},
					},
				},
			},
		},
	}
}

func updateNetworkDevice(f *find.Finder, key int32, label, macAddress string, adapterType string) (*types.VirtualDeviceConfigSpec, error) {
	network, err := f.Network(context.TODO(), "*"+label)
	if err != nil {
		return nil, err
	}

	backing, err := network.EthernetCardBackingInfo(context.TODO())
	if err != nil {
		return nil, err
	}

	if adapterType == "vmxnet3" {
		return &types.VirtualDeviceConfigSpec{
			Operation: types.VirtualDeviceConfigSpecOperationEdit,
			Device: &types.VirtualVmxnet3{
				types.VirtualVmxnet{
					types.VirtualEthernetCard{
						VirtualDevice: types.VirtualDevice{
							Key:     key,
							Backing: backing,
						},
						AddressType: "manual",
						MacAddress:  macAddress,
					},
				},
			},
		}, nil
	} else if adapterType == "e1000" {
		return &types.VirtualDeviceConfigSpec{
			Operation: types.VirtualDeviceConfigSpecOperationEdit,
			Device: &types.VirtualE1000{
				types.VirtualEthernetCard{
					VirtualDevice: types.VirtualDevice{
						Key:     key,
						Backing: backing,
					},
					AddressType: "manual",
					MacAddress:  macAddress,
				},
			},
		}, nil
	} else {
		return nil, fmt.Errorf("Invalid network adapter type.")
	}
}

func createNetworkDevice(f *find.Finder, label, adapterType string) (*types.VirtualDeviceConfigSpec, error) {
	network, err := f.Network(context.TODO(), "*"+label)
	if err != nil {
		return nil, err
	}

	backing, err := network.EthernetCardBackingInfo(context.TODO())
	if err != nil {
		return nil, err
	}

	if adapterType == "vmxnet3" {
		return &types.VirtualDeviceConfigSpec{
			Operation: types.VirtualDeviceConfigSpecOperationAdd,
			Device: &types.VirtualVmxnet3{
				types.VirtualVmxnet{
					types.VirtualEthernetCard{
						VirtualDevice: types.VirtualDevice{
							Backing: backing,
						},
					},
				},
			},
		}, nil
	} else if adapterType == "e1000" {
		return &types.VirtualDeviceConfigSpec{
			Operation: types.VirtualDeviceConfigSpecOperationAdd,
			Device: &types.VirtualE1000{
				types.VirtualEthernetCard{
					VirtualDevice: types.VirtualDevice{
						Backing: backing,
					},
				},
			},
		}, nil
	} else {
		return nil, fmt.Errorf("Invalid network adapter type.")
	}
}

func resourceVSphereVirtualMachineUpdate(d *schema.ResourceData, meta interface{}) error {
	log.Printf("[INFO] Entering resourceVSphereVirtualMachineUpdate")
	// make config spec
	configSpec := types.VirtualMachineConfigSpec{}
	// flag if changes have to be applied
	hasChanges := false
	// flag if changes have to be done when powered off
	rebootRequired := false
	// flag if VM is not already powered off
	vmIsOn := false
	// flag if VM is allowed to reboot in configuration file
	rebootAllowed := bool(d.Get("reboot_allowed").(bool))
	// Get vm
	client := meta.(*VSphereClient).vimClient

	dc, err := getDatacenter(client, d.Get("datacenter").(string))
	if err != nil {
		return err
	}
	finder := find.NewFinder(client.Client, true)
	finder = finder.SetDatacenter(dc)

	vm, err := finder.VirtualMachine(context.TODO(), vmPath(d.Get("folder").(string), d.Get("name").(string)))
	state, err := vm.PowerState(context.TODO())
	if err != nil {
		return err
	}
	if state == types.VirtualMachinePowerStatePoweredOn {
		vmIsOn = true
	}
	if err != nil {
		return err
	}

	// Check resources
	if d.HasChange("name") {
		return errors.New(`[ERROR] VM name cannot be changed from vsphere terraform provider. 
			Please do it manually in vsphere. This operation is not recommended !`)
	}

	if d.HasChange("guest_id") {
		configSpec.GuestId = string(d.Get("guest_id").(string))
		hasChanges = true
	}

	if d.HasChange("vcpu") {
		if vmIsOn {
			if bool(d.Get("cpu_hot_add_enabled").(bool)) {
				configSpec.NumCPUs = int32(d.Get("vcpu").(int))
				hasChanges = true
				o, n := d.GetChange("vcpu")
				if int64(o.(int)) > int64(n.(int)) { // Diminution of CPU number : reboot required
					rebootRequired = true
				}
			} else {
				return errors.New("[ERROR] CPU hot add is not enabled. Please check your virtual machine configuration.")
			}
		} else {
			configSpec.NumCPUs = int32(d.Get("vcpu").(int))
			hasChanges = true
		}
	}

	if d.HasChange("cpu_hot_add_enabled") {
		cpuHotAddEnabled := d.Get("cpu_hot_add_enabled").(bool)
		configSpec.CpuHotAddEnabled = &cpuHotAddEnabled
		hasChanges = true
		rebootRequired = true
	}

	if d.HasChange("memory") {
		if vmIsOn {
			if bool(d.Get("memoryHotAddEnabled").(bool)) {
				hasChanges = true
				o, n := d.GetChange("memory")
				if int64(o.(int)) > int64(n.(int)) { // Diminution of RAM : need to reboot VM
					log.Printf("REBOOT VRAM")
					rebootRequired = true
				}
				configSpec.MemoryMB = int64(d.Get("memory").(int))

				// Only for Linux, kernel panics may occur when hot add of RAM if RAM was less than 4GB
				if configSpec.MemoryMB < 4096 {
					rebootRequired = true
				}
			} else {
				return errors.New("[ERROR] Memory hot add is not enabled. Please check your virtual machine configuration.")
			}
		} else {
			configSpec.MemoryMB = int64(d.Get("memory").(int))
			hasChanges = true
		}
	}

	if d.HasChange("memory_hot_add_enabled") {
		memoryHotAddEnabled := d.Get("memory_hot_add_enabled").(bool)
		configSpec.MemoryHotAddEnabled = &memoryHotAddEnabled
		hasChanges = true
		rebootRequired = true
	}

	if d.HasChange("annotation") {
		configSpec.Annotation = d.Get("annotation").(string)
		hasChanges = true
	}
	if d.HasChange("network_interface") {
		hasChanges = true
		_, newNetworkInterfaces := d.GetChange("network_interface")
		newNetworkInterfacesList := newNetworkInterfaces.([]interface{})

		networkDevices := []types.BaseVirtualDeviceConfigSpec{}
		for _, v := range newNetworkInterfacesList {
			network := v.(map[string]interface{})
			label := network["label"].(string)
			macAddress := network["mac_address"].(string)
			log.Println("mac_address : %s", network["mac_address"].(string))
			adapterType := network["adapter_type"].(string)
			log.Println("adapter_type : %s", network["adapter_type"].(string))
			key := int32(network["key"].(int))
			if macAddress == "" {
				nd, err := createNetworkDevice(finder, label, adapterType)
				if err != nil {
					return err
				}
				networkDevices = append(networkDevices, nd)
			} else {
				nd, err := updateNetworkDevice(finder, key, label, macAddress, adapterType)
				if err != nil {
					return err
				}
				networkDevices = append(networkDevices, nd)
			}

		}
		configSpec.DeviceChange = networkDevices
	}

	if d.HasChange("disk") {
		hasChanges = true
		oldDisks, newDisks := d.GetChange("disk")
		oldDiskSet := oldDisks.(*schema.Set)
		newDiskSet := newDisks.(*schema.Set)

		addedDisks := newDiskSet.Difference(oldDiskSet)
		removedDisks := oldDiskSet.Difference(newDiskSet)

		// Removed disks
		for _, diskRaw := range removedDisks.List() {
			if disk, ok := diskRaw.(map[string]interface{}); ok {
				devices, err := vm.Device(context.TODO())
				if err != nil {
					return fmt.Errorf("[ERROR] Update Remove Disk - Could not get virtual device list: %v", err)
				}
				virtualDisk := devices.FindByKey(int32(disk["key"].(int)))

				keep := false
				if v, ok := disk["keep_on_remove"].(bool); ok {
					keep = v
				}

				err = vm.RemoveDevice(context.TODO(), keep, virtualDisk)
				if err != nil {
					return fmt.Errorf("[ERROR] Update Remove Disk - Error removing disk: %v", err)
				}
			}
		}
		// Added disks
		for _, diskRaw := range addedDisks.List() {
			if disk, ok := diskRaw.(map[string]interface{}); ok {

				var datastore *object.Datastore
				if disk["datastore"] == "" {
					datastore, err = finder.DefaultDatastore(context.TODO())
					if err != nil {
						return fmt.Errorf("[ERROR] Update Remove Disk - Error finding datastore: %v", err)
					}
				} else {
					datastore, err = finder.Datastore(context.TODO(), disk["datastore"].(string))
					if err != nil {
						log.Printf("[ERROR] Couldn't find datastore %v.  %s", disk["datastore"].(string), err)
						return err
					}
				}

				var size int64
				if disk["size"] == 0 {
					size = 0
				} else {
					size = int64(disk["size"].(int))
				}
				iops := int64(disk["iops"].(int))
				controller_type := disk["controller_type"].(string)

				var mo mo.VirtualMachine
				vm.Properties(context.TODO(), vm.Reference(), []string{"summary", "config"}, &mo)

				var diskPath string
				switch {
				case disk["vmdk"] != "":
					diskPath = disk["vmdk"].(string)
				case disk["name"] != "":
					snapshotFullDir := mo.Config.Files.SnapshotDirectory
					split := strings.Split(snapshotFullDir, " ")
					if len(split) != 2 {
						return fmt.Errorf("[ERROR] createVirtualMachine - failed to split snapshot directory: %v", snapshotFullDir)
					}
					vmWorkingPath := split[1]
					diskPath = vmWorkingPath + disk["name"].(string)
				default:
					return fmt.Errorf("[ERROR] resourceVSphereVirtualMachineUpdate - Neither vmdk path nor vmdk name was given")
				}

				var initType string
				if disk["type"] != "" {
					initType = disk["type"].(string)
				} else {
					initType = "thin"
				}

				log.Printf("[INFO] Attaching disk: %v", diskPath)
				err = addHardDisk(vm, size, iops, initType, datastore, diskPath, controller_type)
				if err != nil {
					log.Printf("[ERROR] Add Hard Disk Failed: %v", err)
					return err
				}
			}
			if err != nil {
				return err
			}
		}
	}

	// do nothing if there are no changes
	if !hasChanges {
		return nil
	}

	// Checking if vm is not off and if user allowed a reboot
	if rebootRequired && vmIsOn && rebootAllowed {
		log.Printf("[INFO] Shutting down virtual machine: %s", d.Id())

		task, err := vm.PowerOff(context.TODO())
		if err != nil {
			return err
		}

		err = task.Wait(context.TODO())
		if err != nil {
			return err
		}
	} else if rebootRequired && !rebootAllowed && vmIsOn {
		return errors.New(`[ERROR] A Virtual Machine need a reboot but rebootAllowed is false.
			Please set parameter reboot_allowed as true in the VM configuration file.`)
	}

	task, err := vm.Reconfigure(context.TODO(), configSpec)
	if err != nil {
		log.Printf("[ERROR] %s", err)
		return err
	}

	log.Printf("[DEBUG] Waiting for virtual machine to reconfigure: %s", d.Id())
	err = task.Wait(context.TODO())
	if err != nil {
		log.Printf("[ERROR] %s", err)
		return err
	}

	log.Printf("[DEBUG] Waiting for virtual machine to start: %s", d.Id())
	// Restart VM only if it need a reboot :
	// - the VM was powered on
	// - the VM needed a reboot
	// - the VM was allowed to reboot
	log.Printf("Reboot required : %s is on : %s rebootallowed : %s", rebootRequired, vmIsOn, rebootAllowed)
	if rebootRequired && vmIsOn && rebootAllowed {
		log.Printf("Pouet test True")
		task, err = vm.PowerOn(context.TODO())
		if err != nil {
			return err
		}

		err = task.Wait(context.TODO())
		if err != nil {
			log.Printf("[ERROR] %s", err)
			return err
		}

		// Wait for VM guest networking before returning, so that Read can get
		// accurate networking info for the state.
		if d.Get("wait_for_guest_net").(bool) {
			log.Printf("[DEBUG] Waiting for routeable guest network access")
			if err := waitForGuestVMNet(client, vm); err != nil {
				return err
			}
			log.Printf("[DEBUG] Guest has routeable network access.")
		}
	}

	return resourceVSphereVirtualMachineRead(d, meta)
}

func resourceVSphereVirtualMachineCreate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*VSphereClient).vimClient

	vm := virtualMachine{
		name:     d.Get("name").(string),
		vcpu:     int32(d.Get("vcpu").(int)),
		memoryMb: int64(d.Get("memory").(int)),
		memoryAllocation: memoryAllocation{
			reservation: int64(d.Get("memory_reservation").(int)),
		},
	}

	if v, ok := d.GetOk("folder"); ok {
		vm.folder = v.(string)
	}
	if v, ok := d.GetOk("guest_id"); ok {
		vm.guestId = v.(string)
	}
	if v, ok := d.GetOk("cpu_hot_add_enabled"); ok {
		vm.cpuHotAddEnabled = v.(bool)
	}
	if v, ok := d.GetOk("memory_hot_add_enabled"); ok {
		vm.memoryHotAddEnabled = v.(bool)
	}
	if v, ok := d.GetOk("datacenter"); ok {
		vm.datacenter = v.(string)
	}

	if v, ok := d.GetOk("cluster"); ok {
		vm.cluster = v.(string)
	}
	if v, ok := d.GetOk("resource_pool"); ok {
		vm.resourcePool = v.(string)
	}
	if v, ok := d.GetOk("domain"); ok {
		vm.domain = v.(string)
	}
	if v, ok := d.GetOk("time_zone"); ok {
		vm.timeZone = v.(string)
	}
	if v, ok := d.GetOk("annotation"); ok {
		vm.annotation = v.(string)
	} else {
		vm.annotation = ""
	}
	if v, ok := d.GetOk("linked_clone"); ok {
		vm.linkedClone = v.(bool)
	}
	if v, ok := d.GetOk("skip_customization"); ok {
		vm.skipCustomization = v.(bool)
	}
	if v, ok := d.GetOk("enable_disk_uuid"); ok {
		vm.enableDiskUUID = v.(bool)
	}
	if raw, ok := d.GetOk("dns_suffixes"); ok {
		for _, v := range raw.([]interface{}) {
			vm.dnsSuffixes = append(vm.dnsSuffixes, v.(string))
		}
	} else {
		vm.dnsSuffixes = DefaultDNSSuffixes
	}
	if raw, ok := d.GetOk("dns_servers"); ok {
		for _, v := range raw.([]interface{}) {
			vm.dnsServers = append(vm.dnsServers, v.(string))
		}
	} else {
		vm.dnsServers = DefaultDNSServers
	}
	if vL, ok := d.GetOk("custom_configuration_parameters"); ok {
		if custom_configs, ok := vL.(map[string]interface{}); ok {
			custom := make(map[string]types.AnyType)
			for k, v := range custom_configs {
				custom[k] = v
			}
			vm.customConfigurations = custom
			log.Printf("[DEBUG] custom_configuration_parameters init: %v", vm.customConfigurations)
		}
	}

	if vL, ok := d.GetOk("network_interface"); ok {
		networks := make([]networkInterface, len(vL.([]interface{})))
		for i, v := range vL.([]interface{}) {
			network := v.(map[string]interface{})
			networks[i].label = network["label"].(string)
			if v, ok := network["ip_address"].(string); ok && v != "" {
				networks[i].ipv4Address = v
			}
			if v, ok := d.GetOk("gateway"); ok {
				networks[i].ipv4Gateway = v.(string)
			}
			if v, ok := network["subnet_mask"].(string); ok && v != "" {
				ip := net.ParseIP(v).To4()
				if ip != nil {
					mask := net.IPv4Mask(ip[0], ip[1], ip[2], ip[3])
					pl, _ := mask.Size()
					networks[i].ipv4PrefixLength = pl
				} else {
					return fmt.Errorf("subnet_mask parameter is invalid.")
				}
			}
			if v, ok := network["ipv4_address"].(string); ok && v != "" {
				networks[i].ipv4Address = v
			}
			if v, ok := network["ipv4_prefix_length"].(int); ok && v != 0 {
				networks[i].ipv4PrefixLength = v
			}
			if v, ok := network["ipv4_gateway"].(string); ok && v != "" {
				networks[i].ipv4Gateway = v
			}
			if v, ok := network["ipv6_address"].(string); ok && v != "" {
				networks[i].ipv6Address = v
			}
			if v, ok := network["ipv6_prefix_length"].(int); ok && v != 0 {
				networks[i].ipv6PrefixLength = v
			}
			if v, ok := network["ipv6_gateway"].(string); ok && v != "" {
				networks[i].ipv6Gateway = v
			}
			if v, ok := network["mac_address"].(string); ok && v != "" {
				networks[i].macAddress = v
			}
			if v, ok := network["adapter_type"].(string); ok && v != "" {
				networks[i].adapterType = v
			}
		}
		vm.networkInterfaces = networks
		log.Printf("[DEBUG] network_interface init: %v", networks)
	}

	if vL, ok := d.GetOk("windows_opt_config"); ok {
		var winOpt windowsOptConfig
		custom_configs := (vL.([]interface{}))[0].(map[string]interface{})
		if v, ok := custom_configs["admin_password"].(string); ok && v != "" {
			winOpt.adminPassword = v
		}
		if v, ok := custom_configs["domain"].(string); ok && v != "" {
			winOpt.domain = v
		}
		if v, ok := custom_configs["domain_user"].(string); ok && v != "" {
			winOpt.domainUser = v
		}
		if v, ok := custom_configs["product_key"].(string); ok && v != "" {
			winOpt.productKey = v
		}
		if v, ok := custom_configs["domain_user_password"].(string); ok && v != "" {
			winOpt.domainUserPassword = v
		}
		vm.windowsOptionalConfig = winOpt
		log.Printf("[DEBUG] windows config init: %v", winOpt)
	}

	if vL, ok := d.GetOk("disk"); ok {
		if diskSet, ok := vL.(*schema.Set); ok {

			disks := []hardDisk{}
			for _, value := range diskSet.List() {
				disk := value.(map[string]interface{})
				newDisk := hardDisk{}

				if v, ok := disk["template"].(string); ok && v != "" {
					if v, ok := disk["name"].(string); ok && v != "" {
						return fmt.Errorf(`[ERROR] Cannot specify 'name' and 'size'
							attributes of disk if using a template`)
					}
					vm.template = v
					if vm.hasBootableVmdk {
						return fmt.Errorf("[ERROR] Only one bootable disk or template may be given")
					}
					vm.hasBootableVmdk = true
				}

				if v, ok := disk["type"].(string); ok && v != "" {
					newDisk.initType = v
				}

				if v, ok := disk["datastore"].(string); ok && v != "" {
					vm.datastore = v
				}

				if v, ok := disk["size"].(int); ok && v != 0 {
					if v, ok := disk["template"].(string); ok && v != "" {
						return fmt.Errorf("Cannot specify size of a template")
					}

					if v, ok := disk["name"].(string); ok && v != "" {
						newDisk.name = v
					} else {
						return fmt.Errorf("[ERROR] Disk name must be provided when creating a new disk")
					}

					newDisk.size = int64(v)
				}

				if v, ok := disk["iops"].(int); ok && v != 0 {
					newDisk.iops = int64(v)
				}

				if v, ok := disk["controller_type"].(string); ok && v != "" {
					newDisk.controller = v
				}

				if vVmdk, ok := disk["vmdk"].(string); ok && vVmdk != "" {
					if v, ok := disk["template"].(string); ok && v != "" {
						return fmt.Errorf("Cannot specify a vmdk for a template")
					}
					if v, ok := disk["size"].(string); ok && v != "" {
						return fmt.Errorf("Cannot specify size of a vmdk")
					}
					if v, ok := disk["name"].(string); ok && v != "" {
						return fmt.Errorf("Cannot specify name of a vmdk")
					}
					if vBootable, ok := disk["bootable"].(bool); ok {
						if vBootable && vm.hasBootableVmdk {
							return fmt.Errorf("[ERROR] Only one bootable disk or template may be given")
						}
						newDisk.bootable = vBootable
						vm.hasBootableVmdk = vm.hasBootableVmdk || vBootable
					}
					newDisk.vmdkPath = vVmdk
				}
				// Preserves order so bootable disk is first
				if newDisk.bootable == true || disk["template"] != "" {
					disks = append([]hardDisk{newDisk}, disks...)
				} else {
					disks = append(disks, newDisk)
				}
			}
			vm.hardDisks = disks
			log.Printf("[DEBUG] disk init: %v", disks)
		}
	}

	if vL, ok := d.GetOk("cdrom"); ok {
		cdroms := make([]cdrom, len(vL.([]interface{})))
		for i, v := range vL.([]interface{}) {
			c := v.(map[string]interface{})
			if v, ok := c["datastore"].(string); ok && v != "" {
				cdroms[i].datastore = v
			} else {
				return fmt.Errorf("Datastore argument must be specified when attaching a cdrom image.")
			}
			if v, ok := c["path"].(string); ok && v != "" {
				cdroms[i].path = v
			} else {
				return fmt.Errorf("Path argument must be specified when attaching a cdrom image.")
			}
		}
		vm.cdroms = cdroms
		log.Printf("[DEBUG] cdrom init: %v", cdroms)
	}

	err := vm.setupVirtualMachine(client)
	if err != nil {
		log.Printf("[DEBUG] setupVirtulMachine failed: %v", err)
		return err
	}

	d.SetId(vm.Path())
	log.Printf("[INFO] Created virtual machine: %s", d.Id())

	newVM, err := virtualMachineFromManagedObjectID(client, vm.moid)
	if err != nil {
		return err
	}
	newProps, err := virtualMachineProperties(newVM)
	if err != nil {
		return err
	}

	if newProps.Runtime.PowerState == types.VirtualMachinePowerStatePoweredOn && d.Get("wait_for_guest_net").(bool) {
		// We also need to wait for the guest networking to ensure an accurate set
		// of information can be read into state and reported to the provisioners.
		log.Printf("[DEBUG] Waiting for routeable guest network access")
		if err := waitForGuestVMNet(client, newVM); err != nil {
			return err
		}
		log.Printf("[DEBUG] Guest has routeable network access.")
	}
	return resourceVSphereVirtualMachineRead(d, meta)
}

func resourceVSphereVirtualMachineRead(d *schema.ResourceData, meta interface{}) error {
	// Get VM
	client := meta.(*VSphereClient).vimClient
	dc, err := getDatacenter(client, d.Get("datacenter").(string))
	if err != nil {
		return err
	}
	finder := find.NewFinder(client.Client, true)
	finder = finder.SetDatacenter(dc)

	vm, err := finder.VirtualMachine(context.TODO(), d.Id())
	if err != nil {
		d.SetId("")
		return nil
	}

	err = d.Set("moid", vm.Reference().Value)
	if err != nil {
		return fmt.Errorf("Invalid moid to set: %#v", vm.Reference().Value)
	} else {
		log.Printf("[DEBUG] Set the moid: %#v", vm.Reference().Value)
	}

	var mvm mo.VirtualMachine
	collector := property.DefaultCollector(client.Client)
	if err := collector.RetrieveOne(context.TODO(), vm.Reference(), []string{"guest", "summary", "datastore", "config", "runtime"}, &mvm); err != nil {
		return err
	}

	log.Printf("[DEBUG] Datacenter - %#v", dc)
	log.Printf("[DEBUG] mvm.Summary.Config - %#v", mvm.Summary.Config)
	log.Printf("[DEBUG] mvm.Config - %#v", mvm.Config)
	log.Printf("[DEBUG] mvm.Guest.Net - %#v", mvm.Guest.Net)

	err = d.Set("moid", mvm.Reference().Value)
	if err != nil {
		return fmt.Errorf("Invalid moid to set: %#v", mvm.Reference().Value)
	} else {
		log.Printf("[DEBUG] Set the moid: %#v", mvm.Reference().Value)
	}

	disks := make([]map[string]interface{}, 0)
	networkInterfaces := make([]map[string]interface{}, 0)
	templateDisk := make(map[string]interface{}, 1)
	for _, device := range mvm.Config.Hardware.Device {
		switch vd := device.(type) {
		case *types.VirtualVmxnet3:
			virtualDevice := vd.GetVirtualDevice()
			backingInfo := virtualDevice.Backing
			v := backingInfo.(*types.VirtualEthernetCardNetworkBackingInfo)
			label := v.VirtualDeviceDeviceBackingInfo.DeviceName
			macAddress := device.(*types.VirtualVmxnet3).VirtualEthernetCard.MacAddress

			networkInterface := make(map[string]interface{})
			networkInterface["label"] = label
			networkInterface["key"] = virtualDevice.Key
			networkInterface["adapter_type"] = "vmxnet3"
			networkInterface["mac_address"] = macAddress
			log.Printf("Pouet pouet ! MAC Address : %v", macAddress)

			networkInterfaces = append(networkInterfaces, networkInterface)

		case *types.VirtualE1000:
			virtualDevice := vd.GetVirtualDevice()
			backingInfo := virtualDevice.Backing
			v := backingInfo.(*types.VirtualEthernetCardNetworkBackingInfo)
			label := v.VirtualDeviceDeviceBackingInfo.DeviceName
			macAddress := device.(*types.VirtualE1000).VirtualEthernetCard.MacAddress

			networkInterface := make(map[string]interface{})
			networkInterface["label"] = label
			networkInterface["key"] = virtualDevice.Key
			networkInterface["adapter_type"] = "e1000"
			networkInterface["mac_address"] = macAddress
			networkInterfaces = append(networkInterfaces, networkInterface)

		case *types.VirtualDisk:
			virtualDevice := vd.GetVirtualDevice()

			backingInfo := virtualDevice.Backing
			var diskFullPath string
			var diskUuid string
			if v, ok := backingInfo.(*types.VirtualDiskFlatVer2BackingInfo); ok {
				diskFullPath = v.FileName
				diskUuid = v.Uuid
			} else if v, ok := backingInfo.(*types.VirtualDiskSparseVer2BackingInfo); ok {
				diskFullPath = v.FileName
				diskUuid = v.Uuid
			}
			log.Printf("[DEBUG] resourceVSphereVirtualMachineRead - Analyzing disk: %v", diskFullPath)

			// Separate datastore and path
			diskFullPathSplit := strings.Split(diskFullPath, " ")
			if len(diskFullPathSplit) != 2 {
				return fmt.Errorf("[ERROR] Failed trying to parse disk path: %v", diskFullPath)
			}
			diskPath := diskFullPathSplit[1]
			// Isolate filename
			diskNameSplit := strings.Split(diskPath, "/")
			diskName := diskNameSplit[len(diskNameSplit)-1]
			// Remove possible extension
			diskName = strings.Split(diskName, ".")[0]

			if prevDisks, ok := d.GetOk("disk"); ok {
				if prevDiskSet, ok := prevDisks.(*schema.Set); ok {
					for _, v := range prevDiskSet.List() {
						prevDisk := v.(map[string]interface{})

						// We're guaranteed only one template disk.  Passing value directly through since templates should be immutable
						if prevDisk["template"] != "" {
							if len(templateDisk) == 0 {
								templateDisk = prevDisk
								disks = append(disks, templateDisk)
								break
							}
						}

						// It is enforced that prevDisk["name"] should only be set in the case
						// of creating a new disk for the user.
						// size case:  name was set by user, compare parsed filename from mo.filename (without path or .vmdk extension) with name
						// vmdk case:  compare prevDisk["vmdk"] and mo.Filename
						if diskName == prevDisk["name"] || diskPath == prevDisk["vmdk"] {

							prevDisk["key"] = virtualDevice.Key
							prevDisk["uuid"] = diskUuid

							disks = append(disks, prevDisk)
							break
						}
					}
				}
			}
		}
	}

	err = d.Set("network_interface", networkInterfaces)
	if err != nil {
		return fmt.Errorf("Invalid network interfaces to set: %#v", networkInterfaces)
	}

	err = d.Set("disk", disks)
	if err != nil {
		return fmt.Errorf("Invalid disks to set: %#v", disks)
	}

	if len(networkInterfaces) > 0 {
		if _, ok := networkInterfaces[0]["ipv4_address"]; ok {
			log.Printf("[DEBUG] ip address: %v", networkInterfaces[0]["ipv4_address"].(string))
			d.SetConnInfo(map[string]string{
				"type": "ssh",
				"host": networkInterfaces[0]["ipv4_address"].(string),
			})
		}
	}

	var rootDatastore string
	for _, v := range mvm.Datastore {
		var md mo.Datastore
		if err := collector.RetrieveOne(context.TODO(), v, []string{"name", "parent"}, &md); err != nil {
			return err
		}
		if md.Parent.Type == "StoragePod" {
			var msp mo.StoragePod
			if err := collector.RetrieveOne(context.TODO(), *md.Parent, []string{"name"}, &msp); err != nil {
				return err
			}
			rootDatastore = msp.Name
			log.Printf("[DEBUG] %#v", msp.Name)
		} else {
			rootDatastore = md.Name
			log.Printf("[DEBUG] %#v", md.Name)
		}
		break
	}

	d.Set("datacenter", dc)
	d.Set("memory", mvm.Summary.Config.MemorySizeMB)
	d.Set("cpu_hot_add_enabled", mvm.Config.CpuHotAddEnabled)
	d.Set("memory_hot_add_enabled", mvm.Config.MemoryHotAddEnabled)
	d.Set("memory_reservation", mvm.Summary.Config.MemoryReservation)
	d.Set("cpu", mvm.Summary.Config.NumCpu)
	d.Set("datastore", rootDatastore)
	d.Set("uuid", mvm.Summary.Config.Uuid)
	d.Set("annotation", mvm.Summary.Config.Annotation)
	d.Set("power_state", mvm.Runtime.PowerState)

	return nil
}

func resourceVSphereVirtualMachineDelete(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*VSphereClient).vimClient
	dc, err := getDatacenter(client, d.Get("datacenter").(string))
	if err != nil {
		return err
	}
	finder := find.NewFinder(client.Client, true)
	finder = finder.SetDatacenter(dc)

	vm, err := finder.VirtualMachine(context.TODO(), vmPath(d.Get("folder").(string), d.Get("name").(string)))
	if err != nil {
		return err
	}
	devices, err := vm.Device(context.TODO())
	if err != nil {
		log.Printf("[DEBUG] resourceVSphereVirtualMachineDelete - Failed to get device list: %v", err)
		return err
	}

	log.Printf("[INFO] Deleting virtual machine: %s", d.Id())
	state, err := vm.PowerState(context.TODO())
	if err != nil {
		return err
	}

	if state == types.VirtualMachinePowerStatePoweredOn {
		task, err := vm.PowerOff(context.TODO())
		if err != nil {
			return err
		}

		err = task.Wait(context.TODO())
		if err != nil {
			return err
		}
	}

	// Safely eject any disks the user marked as keep_on_remove
	var diskSetList []interface{}
	if vL, ok := d.GetOk("disk"); ok {
		if diskSet, ok := vL.(*schema.Set); ok {
			diskSetList = diskSet.List()
			for _, value := range diskSetList {
				disk := value.(map[string]interface{})

				if v, ok := disk["keep_on_remove"].(bool); ok && v == true {
					log.Printf("[DEBUG] not destroying %v", disk["name"])
					virtualDisk := devices.FindByKey(int32(disk["key"].(int)))
					err = vm.RemoveDevice(context.TODO(), true, virtualDisk)
					if err != nil {
						log.Printf("[ERROR] Update Remove Disk - Error removing disk: %v", err)
						return err
					}
				}
			}
		}
	}

	// Safely eject any disks that are not managed by this resource
	if v, ok := d.GetOk("detach_unknown_disks_on_delete"); ok && v.(bool) {
		var disksToRemove object.VirtualDeviceList
		for _, device := range devices {
			if devices.TypeName(device) != "VirtualDisk" {
				continue
			}
			vd := device.GetVirtualDevice()
			var skip bool
			for _, value := range diskSetList {
				disk := value.(map[string]interface{})
				if int32(disk["key"].(int)) == vd.Key {
					skip = true
					break
				}
			}
			if skip {
				continue
			}
			disksToRemove = append(disksToRemove, device)
		}
		if len(disksToRemove) != 0 {
			err = vm.RemoveDevice(context.TODO(), true, disksToRemove...)
			if err != nil {
				log.Printf("[ERROR] Update Remove Disk - Error removing disk: %v", err)
				return err
			}
		}
	}

	task, err := vm.Destroy(context.TODO())
	if err != nil {
		return err
	}

	err = task.Wait(context.TODO())
	if err != nil {
		return err
	}

	d.SetId("")
	return nil
}

// addHardDisk adds a new Hard Disk to the VirtualMachine.
func addHardDisk(vm *object.VirtualMachine, size, iops int64, diskType string, datastore *object.Datastore, diskPath string, controller_type string) error {
	devices, err := vm.Device(context.TODO())
	if err != nil {
		return err
	}
	log.Printf("[DEBUG] vm devices: %#v\n", devices)

	var controller types.BaseVirtualController
	switch controller_type {
	case "scsi":
		controller, err = devices.FindDiskController(controller_type)
	case "scsi-lsi-parallel":
		controller = devices.PickController(&types.VirtualLsiLogicController{})
	case "scsi-buslogic":
		controller = devices.PickController(&types.VirtualBusLogicController{})
	case "scsi-paravirtual":
		controller = devices.PickController(&types.ParaVirtualSCSIController{})
	case "scsi-lsi-sas":
		controller = devices.PickController(&types.VirtualLsiLogicSASController{})
	case "ide":
		controller, err = devices.FindDiskController(controller_type)
	default:
		return fmt.Errorf("[ERROR] Unsupported disk controller provided: %v", controller_type)
	}

	if err != nil || controller == nil {
		// Check if max number of scsi controller are already used
		diskControllers := getSCSIControllers(devices)
		if len(diskControllers) >= 4 {
			return fmt.Errorf("[ERROR] Maximum number of SCSI controllers created")
		}

		log.Printf("[DEBUG] Couldn't find a %v controller.  Creating one..", controller_type)

		var c types.BaseVirtualDevice
		switch controller_type {
		case "scsi":
			// Create scsi controller
			c, err = devices.CreateSCSIController("scsi")
			if err != nil {
				return fmt.Errorf("[ERROR] Failed creating SCSI controller: %v", err)
			}
		case "scsi-lsi-parallel":
			// Create scsi controller
			c, err = devices.CreateSCSIController("lsilogic")
			if err != nil {
				return fmt.Errorf("[ERROR] Failed creating SCSI controller: %v", err)
			}
		case "scsi-buslogic":
			// Create scsi controller
			c, err = devices.CreateSCSIController("buslogic")
			if err != nil {
				return fmt.Errorf("[ERROR] Failed creating SCSI controller: %v", err)
			}
		case "scsi-paravirtual":
			// Create scsi controller
			c, err = devices.CreateSCSIController("pvscsi")
			if err != nil {
				return fmt.Errorf("[ERROR] Failed creating SCSI controller: %v", err)
			}
		case "scsi-lsi-sas":
			// Create scsi controller
			c, err = devices.CreateSCSIController("lsilogic-sas")
			if err != nil {
				return fmt.Errorf("[ERROR] Failed creating SCSI controller: %v", err)
			}
		case "ide":
			// Create ide controller
			c, err = devices.CreateIDEController()
			if err != nil {
				return fmt.Errorf("[ERROR] Failed creating IDE controller: %v", err)
			}
		default:
			return fmt.Errorf("[ERROR] Unsupported disk controller provided: %v", controller_type)
		}

		vm.AddDevice(context.TODO(), c)
		// Update our devices list
		devices, err := vm.Device(context.TODO())
		if err != nil {
			return err
		}
		controller = devices.PickController(c.(types.BaseVirtualController))
		if controller == nil {
			log.Printf("[ERROR] Could not find the new %v controller", controller_type)
			return fmt.Errorf("Could not find the new %v controller", controller_type)
		}
	}

	log.Printf("[DEBUG] disk controller: %#v\n", controller)

	// TODO Check if diskPath & datastore exist
	// If diskPath is not specified, pass empty string to CreateDisk()
	if diskPath == "" {
		return fmt.Errorf("[ERROR] addHardDisk - No path proided")
	} else {
		diskPath = datastore.Path(diskPath)
	}
	log.Printf("[DEBUG] addHardDisk - diskPath: %v", diskPath)
	disk := devices.CreateDisk(controller, datastore.Reference(), diskPath)

	if strings.Contains(controller_type, "scsi") {
		unitNumber, err := getNextUnitNumber(devices, controller)
		if err != nil {
			return err
		}
		*disk.UnitNumber = unitNumber
	}

	existing := devices.SelectByBackingInfo(disk.Backing)
	log.Printf("[DEBUG] disk: %#v\n", disk)

	if len(existing) == 0 {
		disk.CapacityInKB = int64(size * 1024 * 1024)
		if iops != 0 {
			disk.StorageIOAllocation = &types.StorageIOAllocationInfo{
				Limit: iops,
			}
		}
		backing := disk.Backing.(*types.VirtualDiskFlatVer2BackingInfo)

		if diskType == "eager_zeroed" {
			// eager zeroed thick virtual disk
			backing.ThinProvisioned = types.NewBool(false)
			backing.EagerlyScrub = types.NewBool(true)
		} else if diskType == "lazy" {
			// lazy zeroed thick virtual disk
			backing.ThinProvisioned = types.NewBool(false)
			backing.EagerlyScrub = types.NewBool(false)
		} else if diskType == "thin" {
			// thin provisioned virtual disk
			backing.ThinProvisioned = types.NewBool(true)
		}

		log.Printf("[DEBUG] addHardDisk: %#v\n", disk)
		log.Printf("[DEBUG] addHardDisk capacity: %#v\n", disk.CapacityInKB)

		return vm.AddDevice(context.TODO(), disk)
	} else {
		log.Printf("[DEBUG] addHardDisk: Disk already present.\n")

		return nil
	}
}

func getSCSIControllers(vmDevices object.VirtualDeviceList) []*types.VirtualController {
	// get virtual scsi controllers of all supported types
	var scsiControllers []*types.VirtualController
	for _, device := range vmDevices {
		devType := vmDevices.Type(device)
		switch devType {
		case "scsi", "lsilogic", "buslogic", "pvscsi", "lsilogic-sas":
			if c, ok := device.(types.BaseVirtualController); ok {
				scsiControllers = append(scsiControllers, c.GetVirtualController())
			}
		}
	}
	return scsiControllers
}

func getNextUnitNumber(devices object.VirtualDeviceList, c types.BaseVirtualController) (int32, error) {
	key := c.GetVirtualController().Key

	var unitNumbers [16]bool
	unitNumbers[7] = true

	for _, device := range devices {
		d := device.GetVirtualDevice()

		if d.ControllerKey == key {
			if d.UnitNumber != nil {
				unitNumbers[*d.UnitNumber] = true
			}
		}
	}
	for i, taken := range unitNumbers {
		if !taken {
			return int32(i), nil
		}
	}
	return -1, fmt.Errorf("[ERROR] getNextUnitNumber - controller is full")
}

// addCdrom adds a new virtual cdrom drive to the VirtualMachine and attaches an image (ISO) to it from a datastore path.
func addCdrom(client *govmomi.Client, vm *object.VirtualMachine, datacenter *object.Datacenter, datastore, path string) error {
	devices, err := vm.Device(context.TODO())
	if err != nil {
		return err
	}
	log.Printf("[DEBUG] vm devices: %#v", devices)

	var controller *types.VirtualIDEController
	controller, err = devices.FindIDEController("")
	if err != nil {
		log.Printf("[DEBUG] Couldn't find a ide controller.  Creating one..")

		var c types.BaseVirtualDevice
		c, err := devices.CreateIDEController()
		if err != nil {
			return fmt.Errorf("[ERROR] Failed creating IDE controller: %v", err)
		}

		if v, ok := c.(*types.VirtualIDEController); ok {
			controller = v
		} else {
			return fmt.Errorf("[ERROR] Controller type could not be asserted")
		}
		vm.AddDevice(context.TODO(), c)
		// Update our devices list
		devices, err := vm.Device(context.TODO())
		if err != nil {
			return err
		}
		controller, err = devices.FindIDEController("")
		if err != nil {
			log.Printf("[ERROR] Could not find the new disk IDE controller: %v", err)
			return err
		}
	}
	log.Printf("[DEBUG] ide controller: %#v", controller)

	c, err := devices.CreateCdrom(controller)
	if err != nil {
		return err
	}

	finder := find.NewFinder(client.Client, true)
	finder = finder.SetDatacenter(datacenter)
	ds, err := getDatastore(finder, datastore)
	if err != nil {
		return err
	}

	c = devices.InsertIso(c, ds.Path(path))
	log.Printf("[DEBUG] addCdrom: %#v", c)

	return vm.AddDevice(context.TODO(), c)
}

// buildNetworkDevice builds VirtualDeviceConfigSpec for Network Device.
func buildNetworkDevice(f *find.Finder, label, adapterType string, macAddress string) (*types.VirtualDeviceConfigSpec, error) {
	network, err := f.Network(context.TODO(), "*"+label)
	if err != nil {
		return nil, err
	}

	backing, err := network.EthernetCardBackingInfo(context.TODO())
	if err != nil {
		return nil, err
	}

	var address_type string
	if macAddress == "" {
		address_type = string(types.VirtualEthernetCardMacTypeGenerated)
	} else {
		address_type = string(types.VirtualEthernetCardMacTypeManual)
	}

	if adapterType == "vmxnet3" {
		return &types.VirtualDeviceConfigSpec{
			Operation: types.VirtualDeviceConfigSpecOperationAdd,
			Device: &types.VirtualVmxnet3{
				VirtualVmxnet: types.VirtualVmxnet{
					VirtualEthernetCard: types.VirtualEthernetCard{
						VirtualDevice: types.VirtualDevice{
							Key:     -1, // Create
							Backing: backing,
						},
						AddressType: address_type,
						MacAddress:  macAddress,
					},
				},
			},
		}, nil
	} else if adapterType == "e1000" {
		return &types.VirtualDeviceConfigSpec{
			Operation: types.VirtualDeviceConfigSpecOperationAdd,
			Device: &types.VirtualE1000{
				VirtualEthernetCard: types.VirtualEthernetCard{
					VirtualDevice: types.VirtualDevice{
						Key:     -1,
						Backing: backing,
					},
					AddressType: address_type,
					MacAddress:  macAddress,
				},
			},
		}, nil
	} else {
		return nil, fmt.Errorf("Invalid network adapter type.")
	}
}

// buildVMRelocateSpec builds VirtualMachineRelocateSpec to set a place for a new VirtualMachine.
func buildVMRelocateSpec(rp *object.ResourcePool, ds *object.Datastore, vm *object.VirtualMachine, linkedClone bool, initType string) (types.VirtualMachineRelocateSpec, error) {
	var key int32
	var moveType string
	if linkedClone {
		moveType = "createNewChildDiskBacking"
	} else {
		moveType = "moveAllDiskBackingsAndDisallowSharing"
	}
	log.Printf("[DEBUG] relocate type: [%s]", moveType)

	devices, err := vm.Device(context.TODO())
	if err != nil {
		return types.VirtualMachineRelocateSpec{}, err
	}
	for _, d := range devices {
		if devices.Type(d) == "disk" {
			key = int32(d.GetVirtualDevice().Key)
		}
	}

	isThin := initType == "thin"
	eagerScrub := initType == "eager_zeroed"
	rpr := rp.Reference()
	dsr := ds.Reference()
	return types.VirtualMachineRelocateSpec{
		Datastore:    &dsr,
		Pool:         &rpr,
		DiskMoveType: moveType,
		Disk: []types.VirtualMachineRelocateSpecDiskLocator{
			{
				Datastore: dsr,
				DiskBackingInfo: &types.VirtualDiskFlatVer2BackingInfo{
					DiskMode:        "persistent",
					ThinProvisioned: types.NewBool(isThin),
					EagerlyScrub:    types.NewBool(eagerScrub),
				},
				DiskId: key,
			},
		},
	}, nil
}

// getDatastoreObject gets datastore object.
func getDatastoreObject(client *govmomi.Client, f *object.DatacenterFolders, name string) (types.ManagedObjectReference, error) {
	s := object.NewSearchIndex(client.Client)
	ref, err := s.FindChild(context.TODO(), f.DatastoreFolder, name)
	if err != nil {
		return types.ManagedObjectReference{}, err
	}
	if ref == nil {
		return types.ManagedObjectReference{}, fmt.Errorf("Datastore '%s' not found.", name)
	}
	log.Printf("[DEBUG] getDatastoreObject: reference: %#v", ref)
	return ref.Reference(), nil
}

// buildStoragePlacementSpecCreate builds StoragePlacementSpec for create action.
func buildStoragePlacementSpecCreate(f *object.DatacenterFolders, rp *object.ResourcePool, storagePod object.StoragePod, configSpec types.VirtualMachineConfigSpec) types.StoragePlacementSpec {
	vmfr := f.VmFolder.Reference()
	rpr := rp.Reference()
	spr := storagePod.Reference()

	sps := types.StoragePlacementSpec{
		Type:       "create",
		ConfigSpec: &configSpec,
		PodSelectionSpec: types.StorageDrsPodSelectionSpec{
			StoragePod: &spr,
		},
		Folder:       &vmfr,
		ResourcePool: &rpr,
	}
	log.Printf("[DEBUG] findDatastore: StoragePlacementSpec: %#v\n", sps)
	return sps
}

// buildStoragePlacementSpecClone builds StoragePlacementSpec for clone action.
func buildStoragePlacementSpecClone(c *govmomi.Client, f *object.DatacenterFolders, vm *object.VirtualMachine, rp *object.ResourcePool, storagePod object.StoragePod) types.StoragePlacementSpec {
	vmr := vm.Reference()
	vmfr := f.VmFolder.Reference()
	rpr := rp.Reference()
	spr := storagePod.Reference()

	var o mo.VirtualMachine
	err := vm.Properties(context.TODO(), vmr, []string{"datastore"}, &o)
	if err != nil {
		return types.StoragePlacementSpec{}
	}
	ds := object.NewDatastore(c.Client, o.Datastore[0])
	log.Printf("[DEBUG] findDatastore: datastore: %#v\n", ds)

	devices, err := vm.Device(context.TODO())
	if err != nil {
		return types.StoragePlacementSpec{}
	}

	var key int32
	for _, d := range devices.SelectByType((*types.VirtualDisk)(nil)) {
		key = int32(d.GetVirtualDevice().Key)
		log.Printf("[DEBUG] findDatastore: virtual devices: %#v\n", d.GetVirtualDevice())
	}

	sps := types.StoragePlacementSpec{
		Type: "clone",
		Vm:   &vmr,
		PodSelectionSpec: types.StorageDrsPodSelectionSpec{
			StoragePod: &spr,
		},
		CloneSpec: &types.VirtualMachineCloneSpec{
			Location: types.VirtualMachineRelocateSpec{
				Disk: []types.VirtualMachineRelocateSpecDiskLocator{
					{
						Datastore:       ds.Reference(),
						DiskBackingInfo: &types.VirtualDiskFlatVer2BackingInfo{},
						DiskId:          key,
					},
				},
				Pool: &rpr,
			},
			PowerOn:  false,
			Template: false,
		},
		CloneName: "dummy",
		Folder:    &vmfr,
	}
	return sps
}

// findDatastore finds Datastore object.
func findDatastore(c *govmomi.Client, sps types.StoragePlacementSpec) (*object.Datastore, error) {
	var datastore *object.Datastore
	log.Printf("[DEBUG] findDatastore: StoragePlacementSpec: %#v\n", sps)

	srm := object.NewStorageResourceManager(c.Client)
	rds, err := srm.RecommendDatastores(context.TODO(), sps)
	if err != nil {
		return nil, err
	}
	log.Printf("[DEBUG] findDatastore: recommendDatastores: %#v\n", rds)

	spa := rds.Recommendations[0].Action[0].(*types.StoragePlacementAction)
	datastore = object.NewDatastore(c.Client, spa.Destination)
	log.Printf("[DEBUG] findDatastore: datastore: %#v", datastore)

	return datastore, nil
}

// createCdroms is a helper function to attach virtual cdrom devices (and their attached disk images) to a virtual IDE controller.
func createCdroms(client *govmomi.Client, vm *object.VirtualMachine, datacenter *object.Datacenter, cdroms []cdrom) error {
	log.Printf("[DEBUG] add cdroms: %v", cdroms)
	for _, cd := range cdroms {
		log.Printf("[DEBUG] add cdrom (datastore): %v", cd.datastore)
		log.Printf("[DEBUG] add cdrom (cd path): %v", cd.path)
		err := addCdrom(client, vm, datacenter, cd.datastore, cd.path)
		if err != nil {
			return err
		}
	}

	return nil
}

func (vm *virtualMachine) setupVirtualMachine(c *govmomi.Client) error {
	var cw *virtualMachineCustomizationWaiter
	dc, err := getDatacenter(c, vm.datacenter)

	if err != nil {
		log.Printf("[DEBUG] getDatacenter failed: %v", err)
		return err
	}
	finder := find.NewFinder(c.Client, true)
	finder = finder.SetDatacenter(dc)

	var template *object.VirtualMachine
	var template_mo mo.VirtualMachine
	var vm_mo mo.VirtualMachine
	if vm.template != "" {
		template, err = finder.VirtualMachine(context.TODO(), vm.template)
		if err != nil {
			log.Printf("[DEBUG] template finder failed: %v", err)
			return err
		}
		log.Printf("[DEBUG] template: %#v", template)

		err = template.Properties(context.TODO(), template.Reference(), []string{"parent", "config.template", "config.guestId", "resourcePool", "snapshot", "guest.toolsVersionStatus2", "config.guestFullName"}, &template_mo)
		if err != nil {
			log.Printf("[DEBUG] template.Properties failed : %v", err)
			return err
		}
	}

	var resourcePool *object.ResourcePool
	if vm.resourcePool == "" {
		if vm.cluster == "" {
			resourcePool, err = finder.DefaultResourcePool(context.TODO())
			if err != nil {
				return err
			}
		} else {
			resourcePool, err = finder.ResourcePool(context.TODO(), "*"+vm.cluster+"/Resources")
			if err != nil {
				return err
			}
		}
	} else {
		resourcePool, err = finder.ResourcePool(context.TODO(), vm.resourcePool)
		if err != nil {
			return err
		}
	}
	log.Printf("[DEBUG] resource pool: %#v", resourcePool)

	dcFolders, err := dc.Folders(context.TODO())
	if err != nil {
		return err
	}
	log.Printf("[DEBUG] folder: %#v", vm.folder)

	folder := dcFolders.VmFolder
	if len(vm.folder) > 0 {
		si := object.NewSearchIndex(c.Client)
		folderRef, err := si.FindByInventoryPath(
			context.TODO(), fmt.Sprintf("%v/vm/%v", vm.datacenter, vm.folder))
		if err != nil {
			return fmt.Errorf("Error reading folder %s: %s", vm.folder, err)
		} else if folderRef == nil {
			return fmt.Errorf("Cannot find folder %s", vm.folder)
		} else {
			folder = folderRef.(*object.Folder)
		}
	}

	// make config spec
	configSpec := types.VirtualMachineConfigSpec{
		Name:              vm.name,
		NumCPUs:           vm.vcpu,
		GuestId:           vm.guestId,
		NumCoresPerSocket: 1,
		MemoryMB:          vm.memoryMb,
		MemoryAllocation: &types.ResourceAllocationInfo{
			Reservation: vm.memoryAllocation.reservation,
		},
		Flags: &types.VirtualMachineFlagInfo{
			DiskUuidEnabled: &vm.enableDiskUUID,
		},
		Annotation:          vm.annotation,
		CpuHotAddEnabled:    &vm.cpuHotAddEnabled,
		MemoryHotAddEnabled: &vm.memoryHotAddEnabled,
	}
	log.Printf("[DEBUG] virtual machine config spec: %v", configSpec)

	// make ExtraConfig
	log.Printf("[DEBUG] virtual machine Extra Config spec start")
	if len(vm.customConfigurations) > 0 {
		var ov []types.BaseOptionValue
		for k, v := range vm.customConfigurations {
			key := k
			value := v
			o := types.OptionValue{
				Key:   key,
				Value: &value,
			}
			log.Printf("[DEBUG] virtual machine Extra Config spec: %s,%s", k, v)
			ov = append(ov, &o)
		}
		configSpec.ExtraConfig = ov
		log.Printf("[DEBUG] virtual machine Extra Config spec: %v", configSpec.ExtraConfig)
	}

	var datastore *object.Datastore
	if vm.datastore == "" {
		datastore, err = finder.DefaultDatastore(context.TODO())
		if err != nil {
			return err
		}
	} else {
		datastore, err = finder.Datastore(context.TODO(), vm.datastore)
		if err != nil {
			// TODO: datastore cluster support in govmomi finder function
			d, err := getDatastoreObject(c, dcFolders, vm.datastore)
			if err != nil {
				return err
			}

			if d.Type == "StoragePod" {
				sp := object.StoragePod{
					Folder: object.NewFolder(c.Client, d),
				}

				var sps types.StoragePlacementSpec
				if vm.template != "" {
					sps = buildStoragePlacementSpecClone(c, dcFolders, template, resourcePool, sp)
				} else {
					sps = buildStoragePlacementSpecCreate(dcFolders, resourcePool, sp, configSpec)
				}

				datastore, err = findDatastore(c, sps)
				if err != nil {
					return err
				}
			} else {
				datastore = object.NewDatastore(c.Client, d)
			}
		}
	}

	log.Printf("[DEBUG] datastore: %#v", datastore)

	// network
	networkDevices := []types.BaseVirtualDeviceConfigSpec{}
	networkConfigs := []types.CustomizationAdapterMapping{}
	for _, network := range vm.networkInterfaces {
		nd, err := buildNetworkDevice(finder, network.label, network.adapterType, network.macAddress)
		if err != nil {
			return err
		}
		log.Printf("[DEBUG] network device: %+v", nd.Device)
		networkDevices = append(networkDevices, nd)

		if vm.template != "" {
			var ipSetting types.CustomizationIPSettings
			if network.ipv4Address == "" {
				ipSetting.Ip = &types.CustomizationDhcpIpGenerator{}
			} else {
				if network.ipv4PrefixLength == 0 {
					return fmt.Errorf("Error: ipv4_prefix_length argument is empty.")
				}
				m := net.CIDRMask(network.ipv4PrefixLength, 32)
				sm := net.IPv4(m[0], m[1], m[2], m[3])
				subnetMask := sm.String()
				log.Printf("[DEBUG] ipv4 gateway: %v\n", network.ipv4Gateway)
				log.Printf("[DEBUG] ipv4 address: %v\n", network.ipv4Address)
				log.Printf("[DEBUG] ipv4 prefix length: %v\n", network.ipv4PrefixLength)
				log.Printf("[DEBUG] ipv4 subnet mask: %v\n", subnetMask)
				ipSetting.Gateway = []string{
					network.ipv4Gateway,
				}
				ipSetting.Ip = &types.CustomizationFixedIp{
					IpAddress: network.ipv4Address,
				}
				ipSetting.SubnetMask = subnetMask
			}

			ipv6Spec := &types.CustomizationIPSettingsIpV6AddressSpec{}
			if network.ipv6Address == "" {
				ipv6Spec.Ip = []types.BaseCustomizationIpV6Generator{
					&types.CustomizationDhcpIpV6Generator{},
				}
			} else {
				log.Printf("[DEBUG] ipv6 gateway: %v\n", network.ipv6Gateway)
				log.Printf("[DEBUG] ipv6 address: %v\n", network.ipv6Address)
				log.Printf("[DEBUG] ipv6 prefix length: %v\n", network.ipv6PrefixLength)

				ipv6Spec.Ip = []types.BaseCustomizationIpV6Generator{
					&types.CustomizationFixedIpV6{
						IpAddress:  network.ipv6Address,
						SubnetMask: int32(network.ipv6PrefixLength),
					},
				}
				ipv6Spec.Gateway = []string{network.ipv6Gateway}
			}
			ipSetting.IpV6Spec = ipv6Spec

			// network config
			config := types.CustomizationAdapterMapping{
				Adapter: ipSetting,
			}
			networkConfigs = append(networkConfigs, config)
		}
		log.Printf("[DEBUG] Wesh MAC address : %v", network.macAddress)
	}

	var task *object.Task
	if vm.template == "" {
		var mds mo.Datastore
		if err = datastore.Properties(context.TODO(), datastore.Reference(), []string{"name"}, &mds); err != nil {
			return err
		}
		log.Printf("[DEBUG] datastore: %#v", mds.Name)
		scsi, err := object.SCSIControllerTypes().CreateSCSIController("scsi")
		if err != nil {
			log.Printf("[ERROR] %s", err)
			return err
		}

		configSpec.DeviceChange = append(configSpec.DeviceChange, &types.VirtualDeviceConfigSpec{
			Operation: types.VirtualDeviceConfigSpecOperationAdd,
			Device:    scsi,
		})

		configSpec.Files = &types.VirtualMachineFileInfo{VmPathName: fmt.Sprintf("[%s]", mds.Name)}
		log.Printf("[DEBUG] Creating task for creating VM")
		task, err = folder.CreateVM(context.TODO(), configSpec, resourcePool, nil)
		if err != nil {
			log.Printf("[ERROR] %s", err)
			return err
		}

		log.Printf("[DEBUG] Waiting for task")
		err = task.Wait(context.TODO())
		if err != nil {
			log.Printf("[ERROR] %s", err)
			return err
		}
		log.Printf("[DEBUG] Finishing...")

	} else {

		relocateSpec, err := buildVMRelocateSpec(resourcePool, datastore, template, vm.linkedClone, vm.hardDisks[0].initType)
		if err != nil {
			return err
		}

		log.Printf("[DEBUG] relocate spec: %v", relocateSpec)

		// make vm clone spec
		cloneSpec := types.VirtualMachineCloneSpec{
			Location: relocateSpec,
			Template: false,
			Config:   &configSpec,
			PowerOn:  false,
		}
		if vm.linkedClone {
			if template_mo.Snapshot == nil {
				return fmt.Errorf("`linkedClone=true`, but image VM has no snapshots")
			}
			cloneSpec.Snapshot = template_mo.Snapshot.CurrentSnapshot
		}
		log.Printf("[DEBUG] clone spec: %v", cloneSpec)

		task, err = template.Clone(context.TODO(), folder, vm.name, cloneSpec)
		if err != nil {
			return err
		}
	}

	err = task.Wait(context.TODO())
	if err != nil {
		log.Printf("[ERROR] %s", err)
		return err
	}

	newVM, err := finder.VirtualMachine(context.TODO(), vm.Path())
	if err != nil {
		return err
	}
	log.Printf("[DEBUG] new vm: %v", newVM)

	devices, err := newVM.Device(context.TODO())
	if err != nil {
		log.Printf("[DEBUG] Template devices can't be found")
		return err
	}

	for _, dvc := range devices {
		// Issue 3559/3560: Delete all ethernet devices to add the correct ones later
		if devices.Type(dvc) == "ethernet" {
			err := newVM.RemoveDevice(context.TODO(), false, dvc)
			if err != nil {
				return err
			}
		}
	}
	// Add Network devices
	for _, dvc := range networkDevices {
		err := newVM.AddDevice(
			context.TODO(), dvc.GetVirtualDeviceConfigSpec().Device)
		if err != nil {
			return err
		}
	}

	// Create the cdroms if needed.
	if err := createCdroms(c, newVM, dc, vm.cdroms); err != nil {
		return err
	}

	newVM.Properties(context.TODO(), newVM.Reference(), []string{"summary", "config"}, &vm_mo)
	firstDisk := 0
	if vm.template != "" {
		firstDisk++
	}
	for i := firstDisk; i < len(vm.hardDisks); i++ {
		log.Printf("[DEBUG] disk index: %v", i)

		var diskPath string
		switch {
		case vm.hardDisks[i].vmdkPath != "":
			diskPath = vm.hardDisks[i].vmdkPath
		case vm.hardDisks[i].name != "":
			snapshotFullDir := vm_mo.Config.Files.SnapshotDirectory
			split := strings.Split(snapshotFullDir, " ")
			if len(split) != 2 {
				return fmt.Errorf("[ERROR] setupVirtualMachine - failed to split snapshot directory: %v", snapshotFullDir)
			}
			vmWorkingPath := split[1]
			diskPath = vmWorkingPath + vm.hardDisks[i].name
		default:
			return fmt.Errorf("[ERROR] setupVirtualMachine - Neither vmdk path nor vmdk name was given: %#v", vm.hardDisks[i])
		}
		err = addHardDisk(newVM, vm.hardDisks[i].size, vm.hardDisks[i].iops, vm.hardDisks[i].initType, datastore, diskPath, vm.hardDisks[i].controller)
		if err != nil {
			err2 := addHardDisk(newVM, vm.hardDisks[i].size, vm.hardDisks[i].iops, vm.hardDisks[i].initType, datastore, diskPath, vm.hardDisks[i].controller)
			if err2 != nil {
				return err2
			}
			return err
		}
	}

	if vm.skipCustomization || vm.template == "" {
		log.Printf("[DEBUG] VM customization skipped")
	} else {
		var identity_options types.BaseCustomizationIdentitySettings
		if strings.HasPrefix(template_mo.Config.GuestId, "win") {
			var timeZone int
			if vm.timeZone == "Etc/UTC" {
				vm.timeZone = "085"
			}
			timeZone, err := strconv.Atoi(vm.timeZone)
			if err != nil {
				return fmt.Errorf("Error converting TimeZone: %s", err)
			}

			guiUnattended := types.CustomizationGuiUnattended{
				AutoLogon:      false,
				AutoLogonCount: 1,
				TimeZone:       int32(timeZone),
			}

			customIdentification := types.CustomizationIdentification{}

			userData := types.CustomizationUserData{
				ComputerName: &types.CustomizationFixedName{
					Name: strings.Split(vm.name, ".")[0],
				},
				ProductId: vm.windowsOptionalConfig.productKey,
				FullName:  "terraform",
				OrgName:   "terraform",
			}

			if vm.windowsOptionalConfig.domainUserPassword != "" && vm.windowsOptionalConfig.domainUser != "" && vm.windowsOptionalConfig.domain != "" {
				customIdentification.DomainAdminPassword = &types.CustomizationPassword{
					PlainText: true,
					Value:     vm.windowsOptionalConfig.domainUserPassword,
				}
				customIdentification.DomainAdmin = vm.windowsOptionalConfig.domainUser
				customIdentification.JoinDomain = vm.windowsOptionalConfig.domain
			}

			if vm.windowsOptionalConfig.adminPassword != "" {
				guiUnattended.Password = &types.CustomizationPassword{
					PlainText: true,
					Value:     vm.windowsOptionalConfig.adminPassword,
				}
			}

			identity_options = &types.CustomizationSysprep{
				GuiUnattended:  guiUnattended,
				Identification: customIdentification,
				UserData:       userData,
			}
		} else {
			identity_options = &types.CustomizationLinuxPrep{
				HostName: &types.CustomizationFixedName{
					Name: strings.Split(vm.name, ".")[0],
				},
				Domain:     vm.domain,
				TimeZone:   vm.timeZone,
				HwClockUTC: types.NewBool(true),
			}
		}

		// create CustomizationSpec
		customSpec := types.CustomizationSpec{
			Identity: identity_options,
			GlobalIPSettings: types.CustomizationGlobalIPSettings{
				DnsSuffixList: vm.dnsSuffixes,
				DnsServerList: vm.dnsServers,
			},
			NicSettingMap: networkConfigs,
		}
		log.Printf("[DEBUG] custom spec: %v", customSpec)

		log.Printf("[DEBUG] VM customization starting")
		cw = newVirtualMachineCustomizationWaiter(c, newVM)
		taskb, err := newVM.Customize(context.TODO(), customSpec)
		if err != nil {
			return err
		}
		_, err = taskb.WaitForResult(context.TODO(), nil)
		if err != nil {
			return err
		}
	}

	if vm.hasBootableVmdk || vm.template != "" {
		t, err := newVM.PowerOn(context.TODO())
		if err != nil {
			return err
		}
		_, err = t.WaitForResult(context.TODO(), nil)
		if err != nil {
			return err
		}
		err = newVM.WaitForPowerState(context.TODO(), types.VirtualMachinePowerStatePoweredOn)
		if err != nil {
			return err
		}
		if cw != nil {
			// Customization is not yet done here 100%. We need to wait for the
			// customization completion events to confirm, so start listening for those
			// now.
			<-cw.Done()
			if cw.Err() != nil {
				return cw.Err()
			}
			log.Printf("[DEBUG] VM customization finished")
		}
	}

	vm.moid = newVM.Reference().Value
	return nil
}

func getNetworkName(c *govmomi.Client, vm *object.VirtualMachine, nic types.BaseVirtualEthernetCard) (string, error) {
	backingInfo := nic.GetVirtualEthernetCard().Backing
	var deviceName string
	switch backingInfo.(type) {
	case *types.VirtualEthernetCardNetworkBackingInfo:
		deviceName = backingInfo.(*types.VirtualEthernetCardNetworkBackingInfo).DeviceName
		break
	case *types.VirtualEthernetCardDistributedVirtualPortBackingInfo:
		portInfo := backingInfo.(*types.VirtualEthernetCardDistributedVirtualPortBackingInfo).Port
		log.Printf("network Port %#v", portInfo)
		o := object.NewDistributedVirtualPortgroup(c.Client, types.ManagedObjectReference{
			Type:  "DistributedVirtualPortgroup",
			Value: portInfo.PortgroupKey,
		})
		var dvp mo.DistributedVirtualPortgroup
		err := o.Properties(context.TODO(), o.Reference(), []string{"name", "config.distributedVirtualSwitch"}, &dvp)
		if err != nil {
			log.Printf("[ERROR]: Error retrieving portgroup %v", err)
			return "", err
		}
		deviceName = dvp.Name
	}
	log.Printf("network Port DeviceName %#v", deviceName)
	return deviceName, nil
}

// Suppress Diff on equal ip
func suppressIpDifferences(k, old, new string, d *schema.ResourceData) bool {
	o := net.ParseIP(old)
	n := net.ParseIP(new)
	if o != nil && n != nil {
		return o.Equal(n)
	}
	return false
}
