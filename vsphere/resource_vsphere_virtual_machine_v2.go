package vsphere

import (
	"fmt"

	"github.com/hashicorp/terraform/helper/schema"
	"github.com/terraform-providers/terraform-provider-vsphere/vsphere/internal/helper/folder"
	"github.com/terraform-providers/terraform-provider-vsphere/vsphere/internal/helper/hostsystem"
	"github.com/terraform-providers/terraform-provider-vsphere/vsphere/internal/helper/resourcepool"
	"github.com/terraform-providers/terraform-provider-vsphere/vsphere/internal/helper/structure"
	"github.com/terraform-providers/terraform-provider-vsphere/vsphere/internal/helper/virtualmachine"
	"github.com/terraform-providers/terraform-provider-vsphere/vsphere/internal/virtualdevice"
	"github.com/vmware/govmomi"
	"github.com/vmware/govmomi/object"
	"github.com/vmware/govmomi/vim25/types"
)

func resourceVSphereVirtualMachineV2() *schema.Resource {
	s := map[string]*schema.Schema{
		"resource_pool_id": {
			Type:        schema.TypeString,
			Optional:    true,
			Description: "The ID of a resource pool to put the virtual machine in.",
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
		"disk": {
			Type:        schema.TypeSet,
			Required:    true,
			Description: "A specification for a virtual disk device on this virtual machine.",
			Set:         virtualdevice.SubresourceHashFunc,
			Elem:        &schema.Resource{Schema: virtualdevice.NewDiskSubresource(nil, 0, nil).Schema()},
		},
		"network_interface": {
			Type:        schema.TypeSet,
			Required:    true,
			Description: "A specification for a virtual NIC on this virtual machine.",
			Set:         virtualdevice.SubresourceHashFunc,
			Elem:        &schema.Resource{Schema: virtualdevice.NewNetworkInterfaceSubresource(nil, 0, nil).Schema()},
		},
		"cdrom": {
			Type:        schema.TypeSet,
			Required:    true,
			Description: "A specification for a CDROM device on this virtual machine.",
			Set:         virtualdevice.SubresourceHashFunc,
			Elem:        &schema.Resource{Schema: virtualdevice.NewCdromSubresource(nil, 0, nil).Schema()},
		},
		vSphereTagAttributeKey: tagsSchema(),
	}
	structure.MergeSchema(s, schemaVirtualMachineConfigSpec())
	structure.MergeSchema(s, schemaVirtualMachineGuestInfo())

	return &schema.Resource{
		Create: resourceVSphereVirtualMachineV2Create,
		Read:   resourceVSphereVirtualMachineV2Read,
		Update: resourceVSphereVirtualMachineV2Update,
		Delete: resourceVSphereVirtualMachineV2Delete,
		Schema: s,
	}
}

func resourceVSphereVirtualMachineV2Create(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*VSphereClient).vimClient
	tagsClient, err := tagsClientIfDefined(d, meta)
	if err != nil {
		return err
	}

	poolID := d.Get("resource_pool_id").(string)
	pool, err := resourcepool.FromID(client, poolID)
	if err != nil {
		return fmt.Errorf("could not find resource pool ID %q: %s", poolID, err)
	}

	// Find the folder based off the path to the resource pool. Basically what we
	// are saying here is that the VM folder that we are placing this VM in needs
	// to be in the same hierarchy as the resource pool - so in other words, the
	// same datacenter.
	fo, err := folder.VirtualMachineFolderFromObject(client, pool, d.Get("path").(string))
	if err != nil {
		return err
	}
	var hs *object.HostSystem
	if v, ok := d.GetOk("host_system_id"); ok {
		hsID := v.(string)
		var err error
		if hs, err = hostsystem.FromID(client, hsID); err != nil {
			return fmt.Errorf("error locating host system at ID %q: %s", hsID, err)
		}
	}

	// Validate that the host is part of the resource pool before proceeding
	if err := resourcepool.ValidateHost(client, pool, hs); err != nil {
		return err
	}

	// Ready to start making the here. First expand our main config spec.
	spec := expandVirtualMachineConfigSpec(d)

	// Now we need to get the defualt device set - this is available in the
	// environment info in the resource pool, which we can then filter through
	// our device CRUD lifecycles to get a full deviceChange attribute for our
	// configspec.
	devices, err := resourcepool.DefaultDevices(client, pool, d.Get("guest_id").(string))
	if err != nil {
		return fmt.Errorf("error loading default device list: %s", err)
	}

	if spec.DeviceChange, err = applyVirtualDevices(d, client, devices); err != nil {
		return err
	}

	// We should now have a complete configSpec! Attempt to create the VM now.
	vm, err := virtualmachine.Create(client, fo, *spec, pool, hs)
	if err != nil {
		return fmt.Errorf("error creating virtual machine: %s", err)
	}
	// VM is created. Set the ID now before proceeding, in case the rest of the
	// process here fails.
	vprops, err := virtualmachine.Properties(vm)
	if err != nil {
		return fmt.Errorf("cannot fetch properties of created virtual machine: %s", err)
	}
	d.SetId(vprops.Config.InstanceUuid)
	// Tag the VM before we go any further if we need to.
	if tagsClient != nil {
		if err := processTagDiff(tagsClient, d, vm); err != nil {
			return err
		}
	}

	// Start the virtual machine, and wait for a routeable address, if we have
	// been set to wait for one.
	if err := virtualmachine.PowerOn(vm); err != nil {
		return fmt.Errorf("error powering on virtual machine: %s", err)
	}
	if err := virtualmachine.WaitForGuestNet(client, vm, d.Get("wait_for_guest_net_timeout").(int)); err != nil {
		return err
	}

	// All done!
	return resourceVSphereVirtualMachineV2Read(d, meta)
}

func resourceVSphereVirtualMachineV2Read(d *schema.ResourceData, meta interface{}) error {
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
	// Resource pool
	if vprops.ResourcePool != nil {
		d.Set("resource_pool_id", vprops.ResourcePool.Value)
	}
	// Set the folder
	f, err := folder.RootPathParticleDatastore.SplitRelativeFolder(vm.InventoryPath)
	if err != nil {
		return fmt.Errorf("error parsing virtual machine path %q: %s", vm.InventoryPath, err)
	}
	d.Set("folder", folder.NormalizePath(f))
	// Set VM's current host ID if available
	if vprops.Runtime.Host != nil {
		d.Set("host_system_id", vprops.Runtime.Host.Value)
	}

	// Read general VM config info
	if err := flattenVirtualMachineConfigInfo(d, vprops.Config); err != nil {
		return fmt.Errorf("error reading virtual machine configuration: %s", err)
	}

	// Perform pending device read operations.
	devices := object.VirtualDeviceList(vprops.Config.Hardware.Device)
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

	// Finally, select a valid IP address for use by the VM for purposes of
	// provisioning. This also populates some computed values to present to the
	// user.
	if vprops.Guest != nil {
		if err := buildAndSelectGuestIPs(d, *vprops.Guest); err != nil {
			return fmt.Errorf("error reading virtual machine guest data: %s", err)
		}
	}

	return nil
}

func resourceVSphereVirtualMachineV2Update(d *schema.ResourceData, meta interface{}) error {
	return resourceVSphereVirtualMachineV2Read(d, meta)
}

func resourceVSphereVirtualMachineV2Delete(d *schema.ResourceData, meta interface{}) error {
	return nil
}

// applyVirtualDevices is used by Create and Update to build a list of virtual
// device changes.
func applyVirtualDevices(d *schema.ResourceData, c *govmomi.Client, l object.VirtualDeviceList) ([]types.BaseVirtualDeviceConfigSpec, error) {
	// We filter this device list through each major device class' apply
	// operation. This will give us a final set of changes that will be our
	// deviceChange attribute.
	var spec, delta []types.BaseVirtualDeviceConfigSpec
	var err error
	// Disks first
	l, delta, err = virtualdevice.DiskApplyOperation(d, c, l)
	if err != nil {
		return nil, err
	}
	spec = append(spec, delta...)
	// Network devices
	l, delta, err = virtualdevice.NetworkInterfaceApplyOperation(d, c, l)
	if err != nil {
		return nil, err
	}
	spec = append(spec, delta...)
	// CDROM
	l, delta, err = virtualdevice.CdromApplyOperation(d, c, l)
	if err != nil {
		return nil, err
	}
	spec = append(spec, delta...)
	return spec, nil
}
