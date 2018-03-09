package vsphere

import (
	"errors"
	"fmt"
	"log"

	"github.com/hashicorp/terraform/helper/schema"
	"github.com/hashicorp/terraform/helper/validation"
	"github.com/terraform-providers/terraform-provider-vsphere/vsphere/internal/helper/customattribute"
	"github.com/terraform-providers/terraform-provider-vsphere/vsphere/internal/helper/datastore"
	"github.com/terraform-providers/terraform-provider-vsphere/vsphere/internal/helper/folder"
	"github.com/terraform-providers/terraform-provider-vsphere/vsphere/internal/helper/hostsystem"
	"github.com/terraform-providers/terraform-provider-vsphere/vsphere/internal/helper/resourcepool"
	"github.com/terraform-providers/terraform-provider-vsphere/vsphere/internal/helper/structure"
	"github.com/terraform-providers/terraform-provider-vsphere/vsphere/internal/helper/viapi"
	"github.com/terraform-providers/terraform-provider-vsphere/vsphere/internal/helper/virtualmachine"
	"github.com/terraform-providers/terraform-provider-vsphere/vsphere/internal/virtualdevice"
	"github.com/terraform-providers/terraform-provider-vsphere/vsphere/internal/vmworkflow"
	"github.com/vmware/govmomi"
	"github.com/vmware/govmomi/object"
	"github.com/vmware/govmomi/vim25/types"
)

// formatVirtualMachinePostCloneRollbackError defines the verbose error when
// rollback fails on a post-clone virtual machine operation.
const formatVirtualMachinePostCloneRollbackError = `
WARNING:
There was an error performing post-clone changes to virtual machine %q:
%s
Additionally, there was an error removing the cloned virtual machine:
%s

The virtual machine may still exist in Terraform state. If it does, the
resource will need to be tainted before trying again. For more information on
how to do this, see the following page:
https://www.terraform.io/docs/commands/taint.html

If the virtual machine does not exist in state, manually delete it to try again.
`

// formatVirtualMachineCustomizationWaitError defines the verbose error that is
// sent when the customization waiter returns an error. This can either be due
// to timeout waiting for respective events or a guest-specific customization
// error. The resource does not roll back in this case, to assist with
// troubleshooting.
const formatVirtualMachineCustomizationWaitError = `
Virtual machine customization failed on %q:

%s

The virtual machine has not been deleted to assist with troubleshooting. If
corrective steps are taken without modifying the "customize" block of the
resource configuration, the resource will need to be tainted before trying
again. For more information on how to do this, see the following page:
https://www.terraform.io/docs/commands/taint.html
`

func resourceVSphereVirtualMachine() *schema.Resource {
	s := map[string]*schema.Schema{
		"resource_pool_id": {
			Type:        schema.TypeString,
			Required:    true,
			Description: "The ID of a resource pool to put the virtual machine in.",
		},
		"datastore_id": {
			Type:        schema.TypeString,
			Required:    true,
			Description: "The ID of the virtual machine's datastore. The virtual machine configuration is placed here, along with any virtual disks that are created without datastores.",
		},
		"folder": {
			Type:        schema.TypeString,
			Optional:    true,
			Description: "The name of the folder to locate the virtual machine in.",
			StateFunc:   folder.NormalizePath,
		},
		"host_system_id": {
			Type:        schema.TypeString,
			Optional:    true,
			Computed:    true,
			Description: "The ID of an optional host system to pin the virtual machine to.",
		},
		"wait_for_guest_net_timeout": {
			Type:        schema.TypeInt,
			Optional:    true,
			Default:     5,
			Description: "The amount of time, in minutes, to wait for a routeable IP address on this virtual machine. A value less than 1 disables the waiter.",
		},
		"shutdown_wait_timeout": {
			Type:         schema.TypeInt,
			Optional:     true,
			Default:      3,
			Description:  "The amount of time, in minutes, to wait for shutdown when making necessary updates to the virtual machine.",
			ValidateFunc: validation.IntBetween(1, 10),
		},
		"migrate_wait_timeout": {
			Type:         schema.TypeInt,
			Optional:     true,
			Default:      30,
			Description:  "The amount of time, in minutes, to wait for a vMotion operation to complete before failing.",
			ValidateFunc: validation.IntAtLeast(10),
		},
		"force_power_off": {
			Type:        schema.TypeBool,
			Optional:    true,
			Default:     true,
			Description: "Set to true to force power-off a virtual machine if a graceful guest shutdown failed for a necessary operation.",
		},
		"scsi_controller_count": {
			Type:         schema.TypeInt,
			Optional:     true,
			Default:      1,
			Description:  "The number of SCSI controllers that Terraform manages on this virtual machine. This directly affects the amount of disks you can add to the virtual machine and the maximum disk unit number. Note that lowering this value does not remove controllers.",
			ValidateFunc: validation.IntBetween(1, 4),
		},
		"scsi_type": {
			Type:         schema.TypeString,
			Optional:     true,
			Default:      virtualdevice.SubresourceControllerTypeParaVirtual,
			Description:  "The type of SCSI bus this virtual machine will have. Can be one of lsilogic, lsilogic-sas or pvscsi.",
			ValidateFunc: validation.StringInSlice(virtualdevice.SCSIBusTypeAllowedValues, false),
		},
		// NOTE: disk is only optional so that we can flag it as computed and use
		// it in ResourceDiff. We validate this field in ResourceDiff to enforce it
		// having a minimum count of 1 for now - but may support diskless VMs
		// later.
		"disk": {
			Type:        schema.TypeList,
			Optional:    true,
			Computed:    true,
			Description: "A specification for a virtual disk device on this virtual machine.",
			MaxItems:    60,
			Elem:        &schema.Resource{Schema: virtualdevice.DiskSubresourceSchema()},
		},
		"network_interface": {
			Type:        schema.TypeList,
			Required:    true,
			Description: "A specification for a virtual NIC on this virtual machine.",
			MaxItems:    10,
			Elem:        &schema.Resource{Schema: virtualdevice.NetworkInterfaceSubresourceSchema()},
		},
		"cdrom": {
			Type:        schema.TypeList,
			Optional:    true,
			Description: "A specification for a CDROM device on this virtual machine.",
			MaxItems:    1,
			Elem:        &schema.Resource{Schema: virtualdevice.CdromSubresourceSchema()},
		},
		"clone": {
			Type:        schema.TypeList,
			Optional:    true,
			Description: "A specification for cloning a virtual machine from template.",
			MaxItems:    1,
			Elem:        &schema.Resource{Schema: vmworkflow.VirtualMachineCloneSchema()},
		},
		"reboot_required": {
			Type:        schema.TypeBool,
			Computed:    true,
			Description: "Value internal to Terraform used to determine if a configuration set change requires a reboot.",
		},
		"vmware_tools_status": {
			Type:        schema.TypeString,
			Computed:    true,
			Description: "The state of VMware tools in the guest. This will determine the proper course of action for some device operations.",
		},
		"vmx_path": {
			Type:        schema.TypeString,
			Computed:    true,
			Description: "The path of the virtual machine's configuration file in the VM's datastore.",
		},
		"imported": {
			Type:        schema.TypeBool,
			Computed:    true,
			Description: "A flag internal to Terraform that indicates that this resource was either imported or came from a earlier major version of this resource.",
		},
		"moid": &schema.Schema{
			Type:        schema.TypeString,
			Computed:    true,
			Description: "The machine object ID from VMWare",
		},
		vSphereTagAttributeKey:    tagsSchema(),
		customattribute.ConfigKey: customattribute.ConfigSchema(),
	}
	structure.MergeSchema(s, schemaVirtualMachineConfigSpec())
	structure.MergeSchema(s, schemaVirtualMachineGuestInfo())

	return &schema.Resource{
		Create:        resourceVSphereVirtualMachineCreate,
		Read:          resourceVSphereVirtualMachineRead,
		Update:        resourceVSphereVirtualMachineUpdate,
		Delete:        resourceVSphereVirtualMachineDelete,
		CustomizeDiff: resourceVSphereVirtualMachineCustomizeDiff,
		Importer: &schema.ResourceImporter{
			State: resourceVSphereVirtualMachineImport,
		},
		SchemaVersion: 3,
		MigrateState:  resourceVSphereVirtualMachineMigrateState,
		Schema:        s,
	}
}

func resourceVSphereVirtualMachineCreate(d *schema.ResourceData, meta interface{}) error {
	log.Printf("[DEBUG] %s: Beginning create", resourceVSphereVirtualMachineIDString(d))
	client := meta.(*VSphereClient).vimClient
	tagsClient, err := tagsClientIfDefined(d, meta)
	if err != nil {
		return err
	}
	// Verify a proper vCenter before proceeding if custom attributes are defined
	attrsProcessor, err := customattribute.GetDiffProcessorIfAttributesDefined(client, d)
	if err != nil {
		return err
	}

	var vm *object.VirtualMachine
	// This is where we process our various VM deploy workflows. We expect the ID
	// of the resource to be set in the workflow to ensure that any post-create
	// operations that fail during this process don't create a dangling resource.
	// The VM should also be returned powered on.
	switch {
	case len(d.Get("clone").([]interface{})) > 0:
		vm, err = resourceVSphereVirtualMachineCreateClone(d, meta)
	default:
		vm, err = resourceVSphereVirtualMachineCreateBare(d, meta)
	}

	if err != nil {
		return err
	}

	// Tag the VM
	if tagsClient != nil {
		if err := processTagDiff(tagsClient, d, vm); err != nil {
			return err
		}
	}

	// Set custom attributes
	if attrsProcessor != nil {
		if err := attrsProcessor.ProcessDiff(vm); err != nil {
			return err
		}
	}

	// Wait for a routeable address if we have been set to wait for one
	if err := virtualmachine.WaitForGuestNet(client, vm, d.Get("wait_for_guest_net_timeout").(int)); err != nil {
		return err
	}

	// All done!
	log.Printf("[DEBUG] %s: Create complete", resourceVSphereVirtualMachineIDString(d))
	return resourceVSphereVirtualMachineRead(d, meta)
}

func resourceVSphereVirtualMachineRead(d *schema.ResourceData, meta interface{}) error {
	log.Printf("[DEBUG] %s: Reading state of virtual machine", resourceVSphereVirtualMachineIDString(d))
	client := meta.(*VSphereClient).vimClient
	id := d.Id()
	vm, err := virtualmachine.FromUUID(client, id)
	if err != nil {
		if _, ok := err.(*virtualmachine.UUIDNotFoundError); ok {
			log.Printf("[DEBUG] %s: Virtual machine not found, marking resource as gone: %s", resourceVSphereVirtualMachineIDString(d), err)
			d.SetId("")
			return nil
		}
		return fmt.Errorf("error searching for with UUID %q: %s", id, err)
	}

	vprops, err := virtualmachine.Properties(vm)
	if err != nil {
		return fmt.Errorf("error fetching VM properties: %s", err)
	}

	// Set the managed object id.
	moid := vm.Reference().Value
	d.Set("moid", moid)
	log.Printf("[DEBUG] MOID for VM %q is %q", vm.InventoryPath, moid)

	// Reset reboot_required. This is an update only variable and should not be
	// set across TF runs.
	d.Set("reboot_required", false)
	// Check to see if VMware tools is running.
	if vprops.Guest != nil {
		d.Set("vmware_tools_status", vprops.Guest.ToolsRunningStatus)
	}

	// Resource pool
	if vprops.ResourcePool != nil {
		d.Set("resource_pool_id", vprops.ResourcePool.Value)
	}
	// Set the folder
	f, err := folder.RootPathParticleVM.SplitRelativeFolder(vm.InventoryPath)
	if err != nil {
		return fmt.Errorf("error parsing virtual machine path %q: %s", vm.InventoryPath, err)
	}
	d.Set("folder", folder.NormalizePath(f))
	// Set VM's current host ID if available
	if vprops.Runtime.Host != nil {
		d.Set("host_system_id", vprops.Runtime.Host.Value)
	}

	// Set the VMX path and default datastore
	dp := &object.DatastorePath{}
	if ok := dp.FromString(vprops.Config.Files.VmPathName); !ok {
		return fmt.Errorf("could not parse VMX file path: %s", vprops.Config.Files.VmPathName)
	}
	// The easiest path for us to get an exact match on the datastore in use is
	// to look for the datastore name in the list of used datastores. This is
	// something we have access to from the VM's properties. This allows us to
	// get away with not having to have the datastore unnecessarily supplied to
	// the resource when it's not used by anything else.
	var ds *object.Datastore
	for _, dsRef := range vprops.Datastore {
		dsx, err := datastore.FromID(client, dsRef.Value)
		if err != nil {
			return fmt.Errorf("error locating VMX datastore: %s", err)
		}
		dsxProps, err := datastore.Properties(dsx)
		if err != nil {
			return fmt.Errorf("error fetching VMX datastore properties: %s", err)
		}
		if dsxProps.Summary.Name == dp.Datastore {
			ds = dsx
		}
	}
	if ds == nil {
		return fmt.Errorf("VMX datastore %s not found", dp.Datastore)
	}
	d.Set("datastore_id", ds.Reference().Value)
	d.Set("vmx_path", dp.Path)

	// Read general VM config info
	if err := flattenVirtualMachineConfigInfo(d, vprops.Config); err != nil {
		return fmt.Errorf("error reading virtual machine configuration: %s", err)
	}

	// Perform pending device read operations.
	devices := object.VirtualDeviceList(vprops.Config.Hardware.Device)
	// Read the state of the SCSI bus.
	d.Set("scsi_type", virtualdevice.ReadSCSIBusState(devices, d.Get("scsi_controller_count").(int)))
	// Disks first
	if err := virtualdevice.DiskRefreshOperation(d, client, devices); err != nil {
		return err
	}
	// Network devices
	if err := virtualdevice.NetworkInterfaceRefreshOperation(d, client, devices); err != nil {
		return err
	}
	// CDROM
	if err := virtualdevice.CdromRefreshOperation(d, client, devices); err != nil {
		return err
	}

	// Read tags if we have the ability to do so
	if tagsClient, _ := meta.(*VSphereClient).TagsClient(); tagsClient != nil {
		if err := readTagsForResource(tagsClient, vm, d); err != nil {
			return err
		}
	}

	// Read set custom attributes
	if customattribute.IsSupported(client) {
		customattribute.ReadFromResource(client, vprops.Entity(), d)
	}

	// Finally, select a valid IP address for use by the VM for purposes of
	// provisioning. This also populates some computed values to present to the
	// user.
	if vprops.Guest != nil {
		if err := buildAndSelectGuestIPs(d, *vprops.Guest); err != nil {
			return fmt.Errorf("error reading virtual machine guest data: %s", err)
		}
	}

	log.Printf("[DEBUG] %s: Read complete", resourceVSphereVirtualMachineIDString(d))
	return nil
}

func resourceVSphereVirtualMachineUpdate(d *schema.ResourceData, meta interface{}) error {
	log.Printf("[DEBUG] %s: Performing update", resourceVSphereVirtualMachineIDString(d))
	client := meta.(*VSphereClient).vimClient
	tagsClient, err := tagsClientIfDefined(d, meta)
	if err != nil {
		return err
	}
	// Verify a proper vCenter before proceeding if custom attributes are defined
	attrsProcessor, err := customattribute.GetDiffProcessorIfAttributesDefined(client, d)
	if err != nil {
		return err
	}

	id := d.Id()
	vm, err := virtualmachine.FromUUID(client, id)
	if err != nil {
		return fmt.Errorf("cannot locate virtual machine with UUID %q: %s", id, err)
	}

	// Update folder if necessary
	if d.HasChange("folder") {
		folder := d.Get("folder").(string)
		if err := virtualmachine.MoveToFolder(client, vm, folder); err != nil {
			return fmt.Errorf("could not move virtual machine to folder %q: %s", folder, err)
		}
	}

	// Apply any pending tags
	if tagsClient != nil {
		if err := processTagDiff(tagsClient, d, vm); err != nil {
			return err
		}
	}

	// Update custom attributes
	if attrsProcessor != nil {
		if err := attrsProcessor.ProcessDiff(vm); err != nil {
			return err
		}
	}

	// Ready to start the VM update. All changes from here, until the update
	// operation finishes successfully, need to be done in partial mode.
	d.Partial(true)

	vprops, err := virtualmachine.Properties(vm)
	if err != nil {
		return fmt.Errorf("error fetching VM properties: %s", err)
	}

	spec, changed, err := expandVirtualMachineConfigSpecChanged(d, client, vprops.Config)
	if err != nil {
		return fmt.Errorf("error in virtual machine configuration: %s", err)
	}

	devices := object.VirtualDeviceList(vprops.Config.Hardware.Device)
	if spec.DeviceChange, err = applyVirtualDevices(d, client, devices); err != nil {
		return err
	}
	// Only carry out the reconfigure if we actually have a change to process.
	if changed || len(spec.DeviceChange) > 0 {
		//Check to see if we need to shutdown the VM for this process.
		if d.Get("reboot_required").(bool) && vprops.Runtime.PowerState != types.VirtualMachinePowerStatePoweredOff {
			// Attempt a graceful shutdown of this process. We wrap this in a VM helper.
			timeout := d.Get("shutdown_wait_timeout").(int)
			force := d.Get("force_power_off").(bool)
			if err := virtualmachine.GracefulPowerOff(client, vm, timeout, force); err != nil {
				return fmt.Errorf("error shutting down virtual machine: %s", err)
			}
		}
		// Perform updates
		if err := virtualmachine.Reconfigure(vm, spec); err != nil {
			return fmt.Errorf("error reconfiguring virtual machine: %s", err)
		}
		// Re-fetch properties
		vprops, err = virtualmachine.Properties(vm)
		if err != nil {
			return fmt.Errorf("error re-fetching VM properties after update: %s", err)
		}
		// Power back on the VM, and wait for network if necessary.
		if vprops.Runtime.PowerState != types.VirtualMachinePowerStatePoweredOn {
			if err := virtualmachine.PowerOn(vm); err != nil {
				return fmt.Errorf("error powering on virtual machine: %s", err)
			}
			if err := virtualmachine.WaitForGuestNet(client, vm, d.Get("wait_for_guest_net_timeout").(int)); err != nil {
				return err
			}
		}
	}
	// Now safe to turn off partial mode.
	d.Partial(false)
	d.Set("reboot_required", false)

	// Now that any pending changes have been done (namely, any disks that don't
	// need to be migrated have been deleted), proceed with vMotion if we have
	// one pending.
	if err := resourceVSphereVirtualMachineUpdateLocation(d, meta); err != nil {
		return fmt.Errorf("error running VM migration: %s", err)
	}

	// All done with updates.
	log.Printf("[DEBUG] %s: Update complete", resourceVSphereVirtualMachineIDString(d))
	return resourceVSphereVirtualMachineRead(d, meta)
}

func resourceVSphereVirtualMachineDelete(d *schema.ResourceData, meta interface{}) error {
	log.Printf("[DEBUG] %s: Performing delete", resourceVSphereVirtualMachineIDString(d))
	client := meta.(*VSphereClient).vimClient
	id := d.Id()
	vm, err := virtualmachine.FromUUID(client, id)
	if err != nil {
		return fmt.Errorf("cannot locate virtual machine with UUID %q: %s", id, err)
	}
	vprops, err := virtualmachine.Properties(vm)
	if err != nil {
		return fmt.Errorf("error fetching VM properties: %s", err)
	}
	// Shutdown the VM first. We do attempt a graceful shutdown for the purpose
	// of catching any edge data issues with associated virtual disks that we may
	// need to retain on delete. However, we ignore the user-set force shutdown
	// flag.
	if vprops.Runtime.PowerState != types.VirtualMachinePowerStatePoweredOff {
		timeout := d.Get("shutdown_wait_timeout").(int)
		if err := virtualmachine.GracefulPowerOff(client, vm, timeout, true); err != nil {
			return fmt.Errorf("error shutting down virtual machine: %s", err)
		}
	}
	// Now attempt to detach any virtual disks that may need to be preserved.
	devices := object.VirtualDeviceList(vprops.Config.Hardware.Device)
	spec := types.VirtualMachineConfigSpec{}
	if spec.DeviceChange, err = virtualdevice.DiskDestroyOperation(d, client, devices); err != nil {
		return err
	}
	// Only run the reconfigure operation if there's actually disks in the spec.
	if len(spec.DeviceChange) > 0 {
		if err := virtualmachine.Reconfigure(vm, spec); err != nil {
			return fmt.Errorf("error detaching virtual disks: %s", err)
		}
	}

	// The final operation here is to destroy the VM.
	if err := virtualmachine.Destroy(vm); err != nil {
		return fmt.Errorf("error destroying virtual machine: %s", err)
	}
	log.Printf("[DEBUG] %s: Delete complete", resourceVSphereVirtualMachineIDString(d))
	return nil
}

func resourceVSphereVirtualMachineCustomizeDiff(d *schema.ResourceDiff, meta interface{}) error {
	log.Printf("[DEBUG] %s: Performing diff customization and validation", resourceVSphereVirtualMachineIDString(d))
	client := meta.(*VSphereClient).vimClient

	// Block certain options from being set depending on the vSphere version.
	version := viapi.ParseVersionFromClient(client)
	if d.Get("efi_secure_boot_enabled").(bool) {
		if version.Older(viapi.VSphereVersion{Product: version.Product, Major: 6, Minor: 5}) {
			return fmt.Errorf("efi_secure_boot_enabled is only supported on vSphere 6.5 and higher")
		}
	}

	// Validate cdrom sub-resources
	if err := virtualdevice.CdromDiffOperation(d, client); err != nil {
		return err
	}

	// Validate network device sub-resources
	if err := virtualdevice.NetworkInterfaceDiffOperation(d, client); err != nil {
		return err
	}

	// Validate and normalize disk sub-resources
	if err := virtualdevice.DiskDiffOperation(d, client); err != nil {
		return err
	}
	// If this is a new resource and we are cloning, perform all clone validation
	// operations.
	if len(d.Get("clone").([]interface{})) > 0 {
		switch {
		case d.Id() == "":
			if err := vmworkflow.ValidateVirtualMachineClone(d, client); err != nil {
				return err
			}
		case d.Get("imported").(bool):
			return errors.New("this resource was imported or migrated from a previous version and does not support cloning. Please remove the \"clone\" block from its configuration")
		}
	}
	log.Printf("[DEBUG] %s: Diff customization and validation complete", resourceVSphereVirtualMachineIDString(d))
	return nil
}

func resourceVSphereVirtualMachineImport(d *schema.ResourceData, meta interface{}) ([]*schema.ResourceData, error) {
	client := meta.(*VSphereClient).vimClient

	name := d.Id()
	if name == "" {
		return nil, fmt.Errorf("path cannot be empty")
	}

	log.Printf("[DEBUG] Looking for VM by name/path %q", name)
	vm, err := virtualmachine.FromPath(client, name, nil)
	if err != nil {
		return nil, fmt.Errorf("error fetching virtual machine: %s", err)
	}
	props, err := virtualmachine.Properties(vm)
	if err != nil {
		return nil, fmt.Errorf("error fetching virtual machine properties: %s", err)
	}

	// Block the import if the VM is a template.
	if props.Config.Template {
		return nil, fmt.Errorf("VM %q is a template and cannot be imported", name)
	}

	// Quickly walk the SCSI bus and determine the number of contiguous
	// controllers starting from bus number 0. This becomes the current SCSI
	// controller count. Anything past this is managed by config.
	log.Printf("[DEBUG] Determining number of SCSI controllers for VM %q", name)
	scsiBus := make([]bool, 4)
	for _, device := range props.Config.Hardware.Device {
		sc, ok := device.(types.BaseVirtualSCSIController)
		if !ok {
			continue
		}
		scsiBus[sc.GetVirtualSCSIController().BusNumber] = true
	}
	var ctlrCnt int
	for _, v := range scsiBus {
		if !v {
			break
		}
		ctlrCnt++
	}
	if ctlrCnt < 1 {
		return nil, fmt.Errorf("VM %q has no SCSI controllers", name)
	}
	d.Set("scsi_controller_count", ctlrCnt)

	// Validate the disks in the VM to make sure that they will work with the
	// resource. This is mainly ensuring that all disks are SCSI disks, but a
	// Read operation is attempted as well to make sure it will survive that.
	if err := virtualdevice.DiskImportOperation(d, client, object.VirtualDeviceList(props.Config.Hardware.Device)); err != nil {
		return nil, err
	}
	// The VM should be ready for reading now
	log.Printf("[DEBUG] VM UUID for %q is %q", name, props.Config.Uuid)
	d.SetId(props.Config.Uuid)
	d.Set("imported", true)

	// Set some defaults. This helps possibly prevent diffs where these values
	// have not been changed.
	rs := resourceVSphereVirtualMachine().Schema
	d.Set("force_power_off", rs["force_power_off"].Default)
	d.Set("migrate_wait_timeout", rs["migrate_wait_timeout"].Default)
	d.Set("shutdown_wait_timeout", rs["shutdown_wait_timeout"].Default)
	d.Set("wait_for_guest_net_timeout", rs["wait_for_guest_net_timeout"].Default)

	log.Printf("[DEBUG] %s: Import complete, resource is ready for read", resourceVSphereVirtualMachineIDString(d))
	return []*schema.ResourceData{d}, nil
}

// resourceVSphereVirtualMachineCreateBare contains the "bare metal" VM
// deploy path. The VM is returned.
func resourceVSphereVirtualMachineCreateBare(d *schema.ResourceData, meta interface{}) (*object.VirtualMachine, error) {
	log.Printf("[DEBUG] %s: VM being created from scratch", resourceVSphereVirtualMachineIDString(d))
	client := meta.(*VSphereClient).vimClient
	poolID := d.Get("resource_pool_id").(string)
	pool, err := resourcepool.FromID(client, poolID)
	if err != nil {
		return nil, fmt.Errorf("could not find resource pool ID %q: %s", poolID, err)
	}

	// Find the folder based off the path to the resource pool. Basically what we
	// are saying here is that the VM folder that we are placing this VM in needs
	// to be in the same hierarchy as the resource pool - so in other words, the
	// same datacenter.
	fo, err := folder.VirtualMachineFolderFromObject(client, pool, d.Get("folder").(string))
	if err != nil {
		return nil, err
	}
	var hs *object.HostSystem
	if v, ok := d.GetOk("host_system_id"); ok {
		hsID := v.(string)
		var err error
		if hs, err = hostsystem.FromID(client, hsID); err != nil {
			return nil, fmt.Errorf("error locating host system at ID %q: %s", hsID, err)
		}
	}

	// Validate that the host is part of the resource pool before proceeding
	if err := resourcepool.ValidateHost(client, pool, hs); err != nil {
		return nil, err
	}

	// Ready to start making the VM here. First expand our main config spec.
	spec, err := expandVirtualMachineConfigSpec(d, client)
	if err != nil {
		return nil, fmt.Errorf("error in virtual machine configuration: %s", err)
	}

	// Set the datastore for the VM.
	ds, err := datastore.FromID(client, d.Get("datastore_id").(string))
	if err != nil {
		return nil, fmt.Errorf("error locating datastore for VM: %s", err)
	}
	spec.Files = &types.VirtualMachineFileInfo{
		VmPathName: fmt.Sprintf("[%s]", ds.Name()),
	}

	// Now we need to get the default device set - this is available in the
	// environment info in the resource pool, which we can then filter through
	// our device CRUD lifecycles to get a full deviceChange attribute for our
	// configspec.
	devices, err := resourcepool.DefaultDevices(client, pool, d.Get("guest_id").(string))
	if err != nil {
		return nil, fmt.Errorf("error loading default device list: %s", err)
	}
	log.Printf("[DEBUG] Default devices: %s", virtualdevice.DeviceListString(devices))

	if spec.DeviceChange, err = applyVirtualDevices(d, client, devices); err != nil {
		return nil, err
	}

	// We should now have a complete configSpec! Attempt to create the VM now.
	vm, err := virtualmachine.Create(client, fo, spec, pool, hs)
	if err != nil {
		return nil, fmt.Errorf("error creating virtual machine: %s", err)
	}
	// VM is created. Set the ID now before proceeding, in case the rest of the
	// process here fails.
	vprops, err := virtualmachine.Properties(vm)
	if err != nil {
		return nil, fmt.Errorf("cannot fetch properties of created virtual machine: %s", err)
	}
	log.Printf("[DEBUG] VM %q - UUID is %q", vm.InventoryPath, vprops.Config.Uuid)
	d.SetId(vprops.Config.Uuid)

	// Start the virtual machine
	if err := virtualmachine.PowerOn(vm); err != nil {
		return nil, fmt.Errorf("error powering on virtual machine: %s", err)
	}
	return vm, nil
}

// resourceVSphereVirtualMachineCreateClone contains the clone VM deploy
// path. The VM is returned.
func resourceVSphereVirtualMachineCreateClone(d *schema.ResourceData, meta interface{}) (*object.VirtualMachine, error) {
	log.Printf("[DEBUG] %s: VM being created from clone", resourceVSphereVirtualMachineIDString(d))
	client := meta.(*VSphereClient).vimClient

	// Find the folder based off the path to the resource pool. Basically what we
	// are saying here is that the VM folder that we are placing this VM in needs
	// to be in the same hierarchy as the resource pool - so in other words, the
	// same datacenter.
	poolID := d.Get("resource_pool_id").(string)
	pool, err := resourcepool.FromID(client, poolID)
	if err != nil {
		return nil, fmt.Errorf("could not find resource pool ID %q: %s", poolID, err)
	}
	fo, err := folder.VirtualMachineFolderFromObject(client, pool, d.Get("folder").(string))
	if err != nil {
		return nil, err
	}

	// Expand the clone spec. We get the source VM here too.
	cloneSpec, srcVM, err := vmworkflow.ExpandVirtualMachineCloneSpec(d, client)
	if err != nil {
		return nil, err
	}

	// Start the clone
	name := d.Get("name").(string)
	timeout := d.Get("clone.0.timeout").(int)
	vm, err := virtualmachine.Clone(client, srcVM, fo, name, cloneSpec, timeout)
	if err != nil {
		return nil, fmt.Errorf("error cloning virtual machine: %s", err)
	}

	// VM is created and updated. It's save to set the ID here now, in case the
	// rest of the process here fails.
	vprops, err := virtualmachine.Properties(vm)
	if err != nil {
		return nil, fmt.Errorf("cannot fetch properties of created virtual machine: %s", err)
	}
	log.Printf("[DEBUG] VM %q - UUID is %q", vm.InventoryPath, vprops.Config.Uuid)
	d.SetId(vprops.Config.Uuid)

	// Before starting or proceeding any further, we need to normalize the
	// configuration of the newly cloned VM. This is basically a subset of update
	// with the stipulation that there is currently no state to help move this
	// along.
	cfgSpec, err := expandVirtualMachineConfigSpec(d, client)
	if err != nil {
		return nil, resourceVSphereVirtualMachineRollbackCreate(d, meta, vm, fmt.Errorf("error in virtual machine configuration: %s", err))
	}

	// To apply device changes, we need the current devicecfgSpec from the config
	// info. We then filter this list through the same apply process we did for
	// create, which will apply the changes in an incremental fashion.
	devices := object.VirtualDeviceList(vprops.Config.Hardware.Device)
	var delta []types.BaseVirtualDeviceConfigSpec
	// First check the state of our SCSI bus. Normalize it if we need to.
	devices, delta, err = virtualdevice.NormalizeSCSIBus(devices, d.Get("scsi_type").(string), d.Get("scsi_controller_count").(int))
	if err != nil {
		return nil, err
	}
	cfgSpec.DeviceChange = virtualdevice.AppendDeviceChangeSpec(cfgSpec.DeviceChange, delta...)
	// Disks
	devices, delta, err = virtualdevice.DiskPostCloneOperation(d, client, devices)
	if err != nil {
		return nil, err
	}
	cfgSpec.DeviceChange = virtualdevice.AppendDeviceChangeSpec(cfgSpec.DeviceChange, delta...)
	// Network devices
	devices, delta, err = virtualdevice.NetworkInterfacePostCloneOperation(d, client, devices)
	if err != nil {
		return nil, err
	}
	cfgSpec.DeviceChange = virtualdevice.AppendDeviceChangeSpec(cfgSpec.DeviceChange, delta...)
	// CDROM
	devices, delta, err = virtualdevice.CdromPostCloneOperation(d, client, devices)
	if err != nil {
		return nil, err
	}
	cfgSpec.DeviceChange = virtualdevice.AppendDeviceChangeSpec(cfgSpec.DeviceChange, delta...)
	log.Printf("[DEBUG] %s: Final device list: %s", resourceVSphereVirtualMachineIDString(d), virtualdevice.DeviceListString(devices))
	log.Printf("[DEBUG] %s: Final device change cfgSpec: %s", resourceVSphereVirtualMachineIDString(d), virtualdevice.DeviceChangeString(cfgSpec.DeviceChange))

	// Perform updates
	if err := virtualmachine.Reconfigure(vm, cfgSpec); err != nil {
		return nil, resourceVSphereVirtualMachineRollbackCreate(d, meta, vm, fmt.Errorf("error reconfiguring virtual machine: %s", err))
	}

	var cw *virtualMachineCustomizationWaiter
	// Send customization spec if any has been defined.
	if len(d.Get("clone.0.customize").([]interface{})) > 0 {
		family, err := resourcepool.OSFamily(client, pool, d.Get("guest_id").(string))
		if err != nil {
			return nil, fmt.Errorf("cannot find OS family for guest ID %q: %s", d.Get("guest_id").(string), err)
		}
		custSpec := vmworkflow.ExpandCustomizationSpec(d, family)
		cw = newVirtualMachineCustomizationWaiter(client, vm, d.Get("clone.0.customize.0.timeout").(int))
		if err := virtualmachine.Customize(vm, custSpec); err != nil {
			// Roll back the VMs as per the error handling in reconfigure.
			if derr := resourceVSphereVirtualMachineDelete(d, meta); derr != nil {
				return nil, fmt.Errorf(formatVirtualMachinePostCloneRollbackError, vm.InventoryPath, err, derr)
			}
			d.SetId("")
			return nil, fmt.Errorf("error sending customization spec: %s", err)
		}
	}

	// Finally time to power on the virtual machine!
	if err := virtualmachine.PowerOn(vm); err != nil {
		return nil, fmt.Errorf("error powering on virtual machine: %s", err)
	}
	// If we customized, wait on customization.
	if cw != nil {
		log.Printf("[DEBUG] %s: Waiting for VM customization to complete", resourceVSphereVirtualMachineIDString(d))
		<-cw.Done()
		if err := cw.Err(); err != nil {
			return nil, fmt.Errorf(formatVirtualMachineCustomizationWaitError, vm.InventoryPath, err)
		}
	}
	// Clone is complete and ready to return
	return vm, nil
}

// resourceVSphereVirtualMachineRollbackCreate attempts to "roll back" a
// resource due to an error that happened post-create that will put the VM in a
// state where it cannot be worked with. This should only be done early on in
// the process, namely on clone operations between when the clone actually
// happens, and no later than after the initial post-clone update is complete.
//
// If the rollback fails, an error is displayed prompting the user to manually
// delete the virtual machine before trying again.
func resourceVSphereVirtualMachineRollbackCreate(
	d *schema.ResourceData,
	meta interface{},
	vm *object.VirtualMachine,
	origErr error,
) error {
	// Updates are largely atomic, so more than likely no disks with
	// keep_on_remove were attached, but just in case, we run this through delete
	// to make sure to safely remove any disk that may have been attached as part
	// of this process if it was flagged as such.
	if err := resourceVSphereVirtualMachineDelete(d, meta); err != nil {
		return fmt.Errorf(formatVirtualMachinePostCloneRollbackError, vm.InventoryPath, origErr, err)
	}
	d.SetId("")
	return fmt.Errorf("error reconfiguring virtual machine: %s", origErr)
}

// resourceVSphereVirtualMachineUpdateLocation manages vMotion. This includes
// the migration of a VM from one host to another, or from one datastore to
// another (storage vMotion).
//
// This function is responsible for building the top-level relocate spec. For
// disks, we call out to relocate functionality in the disk sub-resource.
func resourceVSphereVirtualMachineUpdateLocation(d *schema.ResourceData, meta interface{}) error {
	log.Printf("[DEBUG] %s: Checking for pending migration operations", resourceVSphereVirtualMachineIDString(d))
	client := meta.(*VSphereClient).vimClient

	// A little bit of duplication of VM object data is done here to keep the
	// method signature lean.
	id := d.Id()
	vm, err := virtualmachine.FromUUID(client, id)
	if err != nil {
		return fmt.Errorf("cannot locate virtual machine with UUID %q: %s", id, err)
	}

	// Determine if we are performing any storage vMotion tasks. This will generate the relocators if there are any.
	vprops, err := virtualmachine.Properties(vm)
	if err != nil {
		return fmt.Errorf("error fetching VM properties: %s", err)
	}
	devices := object.VirtualDeviceList(vprops.Config.Hardware.Device)
	relocators, err := virtualdevice.DiskMigrateRelocateOperation(d, client, devices)
	if err != nil {
		return err
	}
	// If we don't have any changes, stop here.
	if !d.HasChange("resource_pool_id") && !d.HasChange("host_system_id") && !d.HasChange("datastore_id") && len(relocators) < 1 {
		log.Printf("[DEBUG] %s: No migration operations found", resourceVSphereVirtualMachineIDString(d))
		return nil
	}
	log.Printf("[DEBUG] %s: Migration operations found, proceeding with migration", resourceVSphereVirtualMachineIDString(d))

	// Fetch and validate pool and host
	poolID := d.Get("resource_pool_id").(string)
	pool, err := resourcepool.FromID(client, poolID)
	if err != nil {
		return fmt.Errorf("could not find resource pool ID %q: %s", poolID, err)
	}
	var hs *object.HostSystem
	if v, ok := d.GetOk("host_system_id"); ok {
		hsID := v.(string)
		var err error
		if hs, err = hostsystem.FromID(client, hsID); err != nil {
			return fmt.Errorf("error locating host system at ID %q: %s", hsID, err)
		}
	}
	if err := resourcepool.ValidateHost(client, pool, hs); err != nil {
		return err
	}

	// Fetch the datastore
	ds, err := datastore.FromID(client, d.Get("datastore_id").(string))
	if err != nil {
		return fmt.Errorf("error locating datastore for VM: %s", err)
	}

	dsRef := ds.Reference()
	pRef := pool.Reference()
	// Start building the spec
	spec := types.VirtualMachineRelocateSpec{
		Datastore: &dsRef,
		Pool:      &pRef,
	}
	if hs != nil {
		hsRef := hs.Reference()
		spec.Host = &hsRef
	}

	spec.Disk = relocators

	// Ready to perform migration. Only do this if necessary.
	return virtualmachine.Relocate(vm, spec, d.Get("migrate_wait_timeout").(int))
}

// applyVirtualDevices is used by Create and Update to build a list of virtual
// device changes.
func applyVirtualDevices(d *schema.ResourceData, c *govmomi.Client, l object.VirtualDeviceList) ([]types.BaseVirtualDeviceConfigSpec, error) {
	// We filter this device list through each major device class' apply
	// operation. This will give us a final set of changes that will be our
	// deviceChange attribute.
	var spec, delta []types.BaseVirtualDeviceConfigSpec
	var err error
	// First check the state of our SCSI bus. Normalize it if we need to.
	l, delta, err = virtualdevice.NormalizeSCSIBus(l, d.Get("scsi_type").(string), d.Get("scsi_controller_count").(int))
	if err != nil {
		return nil, err
	}
	if len(delta) > 0 {
		log.Printf("[DEBUG] %s: SCSI bus has changed and requires a VM restart", resourceVSphereVirtualMachineIDString(d))
		d.Set("reboot_required", true)
	}
	spec = virtualdevice.AppendDeviceChangeSpec(spec, delta...)
	// Disks
	l, delta, err = virtualdevice.DiskApplyOperation(d, c, l)
	if err != nil {
		return nil, err
	}
	spec = virtualdevice.AppendDeviceChangeSpec(spec, delta...)
	// Network devices
	l, delta, err = virtualdevice.NetworkInterfaceApplyOperation(d, c, l)
	if err != nil {
		return nil, err
	}
	spec = virtualdevice.AppendDeviceChangeSpec(spec, delta...)
	// CDROM
	l, delta, err = virtualdevice.CdromApplyOperation(d, c, l)
	if err != nil {
		return nil, err
	}
	spec = virtualdevice.AppendDeviceChangeSpec(spec, delta...)
	log.Printf("[DEBUG] %s: Final device list: %s", resourceVSphereVirtualMachineIDString(d), virtualdevice.DeviceListString(l))
	log.Printf("[DEBUG] %s: Final device change spec: %s", resourceVSphereVirtualMachineIDString(d), virtualdevice.DeviceChangeString(spec))
	return spec, nil
}

// resourceVSphereVirtualMachineIDStringInterface is a small interface so
// that we can take ResourceData and ResourceDiff in
// resourceVSphereVirtualMachineIDString.
type resourceVSphereVirtualMachineIDStringInterface interface {
	Id() string
}

// resourceVSphereVirtualMachineIDString prints a friendly string for the
// vsphere_virtual_machine resource.
func resourceVSphereVirtualMachineIDString(d resourceVSphereVirtualMachineIDStringInterface) string {
	id := d.Id()
	if id == "" {
		id = "<new resource>"
	}
	return fmt.Sprintf("vsphere_virtual_machine (ID = %s)", id)
}
