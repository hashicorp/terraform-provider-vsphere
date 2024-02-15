// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package virtualdevice

import (
	"context"
	"fmt"
	"log"
	"reflect"
	"sort"
	"strings"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	"github.com/hashicorp/terraform-provider-vsphere/vsphere/internal/helper/dvportgroup"
	"github.com/hashicorp/terraform-provider-vsphere/vsphere/internal/helper/hostsystem"
	"github.com/hashicorp/terraform-provider-vsphere/vsphere/internal/helper/network"
	"github.com/hashicorp/terraform-provider-vsphere/vsphere/internal/helper/nsx"
	"github.com/hashicorp/terraform-provider-vsphere/vsphere/internal/helper/provider"
	"github.com/hashicorp/terraform-provider-vsphere/vsphere/internal/helper/structure"
	"github.com/hashicorp/terraform-provider-vsphere/vsphere/internal/helper/viapi"
	"github.com/mitchellh/copystructure"
	"github.com/vmware/govmomi"
	"github.com/vmware/govmomi/object"
	"github.com/vmware/govmomi/vim25/types"
)

const maxNetworkInterfaceCount = 10

const (
	networkInterfaceSubresourceTypeE1000   = "e1000"
	networkInterfaceSubresourceTypeE1000e  = "e1000e"
	networkInterfaceSubresourceTypePCNet32 = "pcnet32"
	networkInterfaceSubresourceTypeSriov   = "sriov"
	networkInterfaceSubresourceTypeVRdma   = "vmxnet3vrdma"
	networkInterfaceSubresourceTypeVmxnet2 = "vmxnet2"
	networkInterfaceSubresourceTypeVmxnet3 = "vmxnet3"
	networkInterfaceSubresourceTypeUnknown = "unknown"
)

const defaultBandwidthLimit = -1
const defaultBandwidthReservation = 0

var defaultBandwidthShareLevel = string(types.SharesLevelNormal)

var networkInterfaceSubresourceTypeAllowedValues = []string{
	networkInterfaceSubresourceTypeE1000,
	networkInterfaceSubresourceTypeE1000e,
	networkInterfaceSubresourceTypeSriov,
	networkInterfaceSubresourceTypeVmxnet3,
	networkInterfaceSubresourceTypeVRdma,
}

// NetworkInterfaceSubresourceSchema returns the schema for the disk
// sub-resource.
func NetworkInterfaceSubresourceSchema() map[string]*schema.Schema {
	s := map[string]*schema.Schema{
		// VirtualEthernetCardResourceAllocation
		"bandwidth_limit": {
			Type:         schema.TypeInt,
			Optional:     true,
			Default:      defaultBandwidthLimit,
			Description:  "The upper bandwidth limit of this network interface, in Mbits/sec.",
			ValidateFunc: validation.IntAtLeast(defaultBandwidthLimit),
		},
		"bandwidth_reservation": {
			Type:         schema.TypeInt,
			Optional:     true,
			Default:      defaultBandwidthReservation,
			Description:  "The bandwidth reservation of this network interface, in Mbits/sec.",
			ValidateFunc: validation.IntAtLeast(defaultBandwidthReservation),
		},
		"bandwidth_share_level": {
			Type:         schema.TypeString,
			Optional:     true,
			Default:      defaultBandwidthShareLevel,
			Description:  "The bandwidth share allocation level for this interface. Can be one of low, normal, high, or custom.",
			ValidateFunc: validation.StringInSlice(sharesLevelAllowedValues, false),
		},
		"bandwidth_share_count": {
			Type:         schema.TypeInt,
			Optional:     true,
			Computed:     true,
			Description:  "The share count for this network interface when the share level is custom.",
			ValidateFunc: validation.IntAtLeast(0),
		},

		// VirtualEthernetCard and friends
		"network_id": {
			Type:         schema.TypeString,
			Required:     true,
			Description:  "The ID of the network to connect this network interface to.",
			ValidateFunc: validation.NoZeroValues,
		},
		"adapter_type": {
			Type:         schema.TypeString,
			Optional:     true,
			Default:      networkInterfaceSubresourceTypeVmxnet3,
			Description:  "The controller type. Can be one of e1000, e1000e, sriov, vmxnet3, or vrdma.",
			ValidateFunc: validation.StringInSlice(networkInterfaceSubresourceTypeAllowedValues, false),
		},
		"physical_function": {
			Type:        schema.TypeString,
			Optional:    true,
			Description: "The ID of the Physical SR-IOV NIC to attach to, e.g. '0000:d8:00.0'",
		},
		"use_static_mac": {
			Type:        schema.TypeBool,
			Optional:    true,
			Description: "If true, the mac_address field is treated as a static MAC address and set accordingly.",
		},
		"mac_address": {
			Type:        schema.TypeString,
			Optional:    true,
			Computed:    true,
			Description: "The MAC address of this network interface. Can only be manually set if use_static_mac is true.",
		},
		"ovf_mapping": {
			Type:        schema.TypeString,
			Optional:    true,
			ForceNew:    true,
			Description: "Mapping of network interface to OVF network.",
		},
	}
	structure.MergeSchema(s, subresourceSchema())
	return s
}

// NetworkInterfaceSubresource represents a vsphere_virtual_machine
// network_interface sub-resource, with a complex device lifecycle.
type NetworkInterfaceSubresource struct {
	*Subresource
}

// NewNetworkInterfaceSubresource returns a network_interface subresource
// populated with all of the necessary fields.
func NewNetworkInterfaceSubresource(client *govmomi.Client, rdd resourceDataDiff, d, old map[string]interface{}, idx int) *NetworkInterfaceSubresource {
	sr := &NetworkInterfaceSubresource{
		Subresource: &Subresource{
			schema:  NetworkInterfaceSubresourceSchema(),
			client:  client,
			srtype:  subresourceTypeNetworkInterface,
			data:    d,
			olddata: old,
			rdd:     rdd,
		},
	}
	sr.Index = idx
	return sr
}

// NetworkInterfaceApplyOperation processes an apply operation for all
// network_interfaces in the resource.
//
// The function takes the root resource's ResourceData, the provider
// connection, and the device list as known to vSphere at the start of this
// operation. All network_interface operations are carried out, with both the
// complete, updated, VirtualDeviceList, and the complete list of changes
// returned as a slice of BaseVirtualDeviceConfigSpec.
func NetworkInterfaceApplyOperation(d *schema.ResourceData, c *govmomi.Client, l object.VirtualDeviceList) (object.VirtualDeviceList, []types.BaseVirtualDeviceConfigSpec, error) {
	log.Printf("[DEBUG] NetworkInterfaceApplyOperation: Beginning apply operation")

	o, n := d.GetChange(subresourceTypeNetworkInterface)
	ods := o.([]interface{})
	nds := n.([]interface{})

	var spec []types.BaseVirtualDeviceConfigSpec

	// Our old and new sets now have an accurate description of devices that may
	// have been added, removed, or changed. Look for removed devices first.
	log.Printf("[DEBUG] NetworkInterfaceApplyOperation: Looking for resources to delete")
nextOld:
	for n, oe := range ods {
		om := oe.(map[string]interface{})
		for _, ne := range nds {
			nm := ne.(map[string]interface{})
			if om["key"] == nm["key"] {
				continue nextOld
			}
		}
		r := NewNetworkInterfaceSubresource(c, d, om, nil, n)
		dspec, err := r.Delete(l)
		if err != nil {
			return nil, nil, fmt.Errorf("%s: %s", r.Addr(), err)
		}
		l = applyDeviceChange(l, dspec)
		spec = append(spec, dspec...)
	}

	// Now check for creates and updates. The results of this operation are
	// committed to state after the operation completes.
	log.Printf("[DEBUG] NetworkInterfaceApplyOperation: Looking for resources to create or update")
	var updates []interface{}
	for n, ne := range nds {
		nm := ne.(map[string]interface{})
		if n < len(ods) {
			// This is an update
			oe := ods[n]
			om := oe.(map[string]interface{})
			if nm["key"] != om["key"] {
				return nil, nil, fmt.Errorf("key mismatch on %s.%d (old: %d, new: %d). This is a bug with the provider, please report it", subresourceTypeNetworkInterface, n, nm["key"].(int), om["key"].(int))
			}
			if reflect.DeepEqual(nm, om) {
				// no change is a no-op
				log.Printf("[DEBUG] NetworkInterfaceApplyOperation: No-op resource: key %d", nm["key"].(int))
				updates = append(updates, nm)
				continue
			}
			r := NewNetworkInterfaceSubresource(c, d, nm, om, n)
			uspec, err := r.Update(l)
			if err != nil {
				return nil, nil, fmt.Errorf("%s: %s", r.Addr(), err)
			}
			l = applyDeviceChange(l, uspec)
			spec = append(spec, uspec...)
			updates = append(updates, r.Data())
			continue
		}
		// New device
		r := NewNetworkInterfaceSubresource(c, d, nm, nil, n)
		cspec, err := r.Create(l)
		if err != nil {
			return nil, nil, fmt.Errorf("%s: %s", r.Addr(), err)
		}
		// Update the VirtualDeviceList l with the newly created resource cspec
		l = applyDeviceChange(l, cspec)
		spec = append(spec, cspec...)
		updates = append(updates, r.Data())
	}

	log.Printf("[DEBUG] NetworkInterfaceApplyOperation: Post-apply final resource list: %s", subresourceListString(updates))
	// We are now done! Return the updated device list and config spec. Save updates as well.
	if err := d.Set(subresourceTypeNetworkInterface, updates); err != nil {
		return nil, nil, err
	}
	log.Printf("[DEBUG] NetworkInterfaceApplyOperation: Device list at end of operation: %s", DeviceListString(l))
	log.Printf("[DEBUG] NetworkInterfaceApplyOperation: Device config operations from apply: %s", DeviceChangeString(spec))
	log.Printf("[DEBUG] NetworkInterfaceApplyOperation: Apply complete, returning updated spec")
	return l, spec, nil
}

// NetworkInterfaceRefreshOperation processes a refresh operation for all of
// the network interfaces attached to this resource.
//
// This functions similar to NetworkInterfaceApplyOperation, but nothing to
// change is returned, all necessary values are just set and committed to
// state.
func NetworkInterfaceRefreshOperation(d *schema.ResourceData, c *govmomi.Client, l object.VirtualDeviceList) error {
	log.Printf("[DEBUG] NetworkInterfaceRefreshOperation: Beginning refresh")
	devices := l.Select(func(device types.BaseVirtualDevice) bool {
		if _, ok := device.(types.BaseVirtualEthernetCard); ok {
			return true
		}
		return false
	})
	log.Printf("[DEBUG] NetworkInterfaceRefreshOperation: Network devices located: %s", DeviceListString(devices))
	curSet := d.Get(subresourceTypeNetworkInterface).([]interface{})
	log.Printf("[DEBUG] NetworkInterfaceRefreshOperation: Current resource set from state: %s", subresourceListString(curSet))

	newSet := make([]interface{}, 0, maxNetworkInterfaceCount)

	// First check for negative keys. These are freshly added devices that are
	// usually coming into read post-create.
	log.Printf("[DEBUG] NetworkInterfaceRefreshOperation: Looking for freshly-created resources to read in")
	for n, item := range curSet {
		m := item.(map[string]interface{})
		if m["key"].(int) < 1 {
			r := NewNetworkInterfaceSubresource(c, d, m, nil, n)
			if r.Get("key").(int) < 1 {
				r.Set("key", devices[n].GetVirtualDevice().Key)
			}
		}
	}

	log.Printf("[DEBUG] NetworkInterfaceRefreshOperation: Network devices after freshly-created device search: %s", DeviceListString(devices))
	log.Printf("[DEBUG] NetworkInterfaceRefreshOperation: Resource sets to write after known device search: %s", subresourceListString(newSet))

	// Go over all devices, refresh via key, and then remove their entries.
	log.Printf("[DEBUG] NetworkInterfaceRefreshOperation: Looking for devices known in state")
	for i := 0; i < len(devices); i++ {
		device := devices[i]
		for n, item := range curSet {
			m := item.(map[string]interface{})
			if m["key"].(int) < 0 {
				// Skip any of these keys as we won't be matching any of those anyway here
				continue
			}
			if device.GetVirtualDevice().Key != int32(m["key"].(int)) {
				// Skip any device that doesn't match key as well
				continue
			}
			// We should have our device -> resource match, so read now.
			r := NewNetworkInterfaceSubresource(c, d, m, nil, n)
			if err := r.Read(l); err != nil {
				return fmt.Errorf("%s: %s", r.Addr(), err)
			}
			// Done reading, push this onto our new sets and remove the device from
			// the list
			newSet = append(newSet, r.Data())

			devices = append(devices[:i], devices[i+1:]...)
			i--
		}
	}
	log.Printf("[DEBUG] NetworkInterfaceRefreshOperation: Resource sets to write after known device search: %s", subresourceListString(newSet))
	log.Printf("[DEBUG] NetworkInterfaceRefreshOperation: Probable orphaned network interfaces: %s", DeviceListString(devices))

	// Finally, any device that is still here is orphaned. They should be added
	// as new devices.
	log.Printf("[DEBUG] NetworkInterfaceRefreshOperation: Adding orphaned devices")
	for n, device := range devices {
		m := make(map[string]interface{})
		vd := device.GetVirtualDevice()
		ctlr := l.FindByKey(vd.ControllerKey)
		if ctlr == nil {
			return fmt.Errorf("could not find controller with key %d", vd.Key)
		}
		m["key"] = int(vd.Key)
		var err error
		m["device_address"], err = computeDevAddr(vd, ctlr.(types.BaseVirtualController))
		if err != nil {
			return fmt.Errorf("error computing device address: %s", err)
		}
		r := NewNetworkInterfaceSubresource(c, d, m, nil, n)
		if err := r.Read(l); err != nil {
			return fmt.Errorf("%s: %s", r.Addr(), err)
		}
		// Done reading, push this onto our new sets and remove the device from
		// the list
		newSet = append(newSet, r.Data())
	}

	log.Printf("[DEBUG] NetworkInterfaceRefreshOperation: %d devices and a new set of length %d", len(devices), len(newSet))

	log.Printf("[DEBUG] NetworkInterfaceRefreshOperation: Resource set to write after adding orphaned devices: %s", subresourceListString(newSet))
	log.Printf("[DEBUG] NetworkInterfaceRefreshOperation: Refresh operation complete, sending new resource set")
	return d.Set(subresourceTypeNetworkInterface, newSet)
}

// NetworkInterfaceDiffOperation performs operations relevant to managing the
// diff on network_interface sub-resources.
func NetworkInterfaceDiffOperation(d *schema.ResourceDiff, c *govmomi.Client) error {
	o, n := d.GetChange(subresourceTypeNetworkInterface)
	ods := o.([]interface{})
	nds := n.([]interface{})
	log.Printf("[DEBUG] NetworkInterfaceDiffOperation: Beginning diff validation")

	for ni, ne := range nds {
		nm := ne.(map[string]interface{})
		if len(ods) > ni {
			oe := ods[ni]
			om := oe.(map[string]interface{})
			r := NewNetworkInterfaceSubresource(c, d, nm, om, ni)
			if err := r.ValidateDiff(); err != nil {
				return fmt.Errorf("%s: %s", r.Addr(), err)
			}
		} else {
			r := NewNetworkInterfaceSubresource(c, d, nm, nil, ni)
			if err := r.ValidateDiff(); err != nil {
				return fmt.Errorf("%s: %s", r.Addr(), err)
			}
		}
	}

	// Various steps related to using SR-IOV NICs:
	// First we want to check that the declaration of nics in the config isn't interleaved with
	// nonSriov amongst sriov (they should be groups of all non-sriov, then sriov).
	// We allow a maximum of 20 interfaces in the terraform file, 10 of each type.
	maxNonSriovIndex := -1
	minSriovIndex := (maxNetworkInterfaceCount * 2) + 1
	countNonSriov := 0
	countSriov := 0
	var sriovPhysicalAdapters []string
	nInt := d.Get("network_interface")
loopInterfaces:
	for ni, ne := range nInt.([]interface{}) {
		nm := ne.(map[string]interface{})
		r := NewNetworkInterfaceSubresource(c, d, nm, nil, ni)
		// If the resource adapter_type is sriov then add to our array of physical
		// adapters the name of the physical_function found on the resource, if
		// not there already
		if r.Get("adapter_type").(string) == networkInterfaceSubresourceTypeSriov {
			countSriov++
			if ni < minSriovIndex {
				minSriovIndex = ni
			}
			for _, adapter := range sriovPhysicalAdapters {
				if adapter == r.Get("physical_function").(string) {
					// It is a duplicate so go to the next interface
					continue loopInterfaces
				}
			}

			sriovPhysicalAdapters = append(sriovPhysicalAdapters, r.Get("physical_function").(string))
		} else {
			countNonSriov++
			if ni > maxNonSriovIndex {
				maxNonSriovIndex = ni
			}
		}
	}

	// Explicitly check for too many interfaces, as the schema MaxItems doesn't differentiate between non-SRIOV and SRIOV
	if countSriov > maxNetworkInterfaceCount || countNonSriov > maxNetworkInterfaceCount {
		return fmt.Errorf("network_interface list exceeded max items of %d non-sriov adapter_types and %d sriov adapter_types."+
			" Config has %d and %d declared.", maxNetworkInterfaceCount, maxNetworkInterfaceCount, countNonSriov, countSriov)
	}

	if countSriov > 0 {
		// Check that all the sriov NICs are declared after the non-sriov ones
		if maxNonSriovIndex > minSriovIndex {
			log.Printf("[DEBUG] network_interfaces out of order. First SRIOV index %d, Last non-SRIOV index %d", minSriovIndex, maxNonSriovIndex)
			return fmt.Errorf("network_interfaces out of order.\n" +
				"network_interfaces with adapter_type 'sriov' must be declared after all network_interfaces with " +
				"other adapter_types. Please reorder the network_interface sections.")
		}

		// Relevant SRIOV checks from https://docs.vmware.com/en/VMware-vSphere/7.0/com.vmware.vsphere.networking.doc/GUID-898A3D66-9415-4854-8413-B40F2CB6FF8D.html
		// First check that the host system is known
		host, err := hostsystem.FromID(c, d.Get("host_system_id").(string))
		if err != nil {
			return fmt.Errorf("trying to use an SR-IOV network interface but target host is not known")
		}
		hprops, err := hostsystem.Properties(host)
		if err != nil {
			return err
		}
		pnics := hprops.Config.Network.Pnic

		// Next, loop through the sriovPhysicalAdapters and check they exist on the host
	loopAdapters:
		for _, sriovPhysicalAdapter := range sriovPhysicalAdapters {
			foundPhysicalNic := false
			for _, pnic := range pnics {
				if pnic.Pci == sriovPhysicalAdapter {
					log.Printf("[DEBUG] Found physical NIC with name %s", sriovPhysicalAdapter)
					foundPhysicalNic = true
					continue loopAdapters
				}
			}
			if !foundPhysicalNic {
				return fmt.Errorf("unable to find SR-IOV physical adapter %s on host %s", sriovPhysicalAdapter, host.Name())
			}
		}
		// Check the physical adapters have SRIOV enabled
		// Sort the sriovPhysicalAdapter addresses so we can look for each in the pciPassthru list without starting
		// from the beginning each time.
		pciPassthru := hprops.Config.PciPassthruInfo
		sort.Strings(sriovPhysicalAdapters)
		adapterIdx := 0
		pciIdx := 0

		// As the pciPassthru list can be quite long, avoid looping through from the top for each adapter.
		// This relies upon the pciPassthruInfo being sorted in id order, which it does appear to be.
		for adapterIdx < len(sriovPhysicalAdapters) {
			foundAdapter := false
			foundSriovEnabled := false

			if pciIdx >= len(pciPassthru) {
				return fmt.Errorf("unable to find SR-IOV physical adapter PCI passthrough Id %s on host %s", sriovPhysicalAdapters[adapterIdx], host.Name())
			}
			switch nicType := pciPassthru[pciIdx].(type) {
			case *types.HostSriovInfo:
				// Check for the SriovEnabled property of the SRIOV PCIPassthrough
				if nicType.Id == sriovPhysicalAdapters[adapterIdx] {
					foundAdapter = true
					if nicType.SriovEnabled == true {
						foundSriovEnabled = true
						log.Printf("[DEBUG] found SR-IOV enabled NIC with name %s", sriovPhysicalAdapters[adapterIdx])
						adapterIdx++
					}
				}
				pciIdx++
			case *types.HostPciPassthruInfo:
				// If the PciPassthruInfo type isn't HostSriovInfo and it matches our configured sriov adapter ID,
				// then SRIOV cannot be enabled on the nic, which is an error. This will be thrown below.
				if nicType.Id == sriovPhysicalAdapters[adapterIdx] {
					foundAdapter = true
				}
				pciIdx++
			default:
				// This would be most unexpected but just carry on, we will error out later
				log.Printf("[DEBUG] diff customization and validation: Found a different type PCI passthrough info %T", nicType)
				pciIdx++
			}

			if foundAdapter && !foundSriovEnabled {
				return fmt.Errorf("physical adapter %s on host %s has SR-IOV function disabled and cannot be used as the physical_function of an SR-IOV network_interface", sriovPhysicalAdapters[adapterIdx], host.Name())
			}
		}

		// Next check Memory reservations have been locked to max
		if d.Get("memory_reservation").(int) != d.Get("memory").(int) {
			return fmt.Errorf("trying to use SR-IOV NIC but memory reservation is not equal to memory, set memory_reservation equal to memory of virtual machine")
		}
	}

	log.Printf("[DEBUG] NetworkInterfaceDiffOperation: Diff validation complete")
	return nil
}

// NetworkInterfacePostCloneOperation normalizes the network interfaces on a
// freshly-cloned virtual machine and outputs any necessary device change
// operations. It also sets the state in advance of the post-create read.
//
// This differs from a regular apply operation in that a configuration is
// already present, but we don't have any existing state, which the standard
// virtual device operations rely pretty heavily on.
func NetworkInterfacePostCloneOperation(d *schema.ResourceData, c *govmomi.Client, l object.VirtualDeviceList) (object.VirtualDeviceList, []types.BaseVirtualDeviceConfigSpec, error) {
	log.Printf("[DEBUG] NetworkInterfacePostCloneOperation: Looking for post-clone device changes")
	devices := l.Select(func(device types.BaseVirtualDevice) bool {
		if _, ok := device.(types.BaseVirtualEthernetCard); ok {
			return true
		}
		return false
	})
	log.Printf("[DEBUG] NetworkInterfacePostCloneOperation: Network devices located: %s", DeviceListString(devices))
	curSet := d.Get(subresourceTypeNetworkInterface).([]interface{})
	log.Printf("[DEBUG] NetworkInterfacePostCloneOperation: Current resource set from configuration: %s", subresourceListString(curSet))
	// Create arrays for the refreshed set of network interfaces which we will now populate. We have a maximum number
	// of 10 network interfaces, so for simplicity create arrays of this length. The final array is the length of
	// the count of network interfaces though.
	srcSet := make([]interface{}, maxNetworkInterfaceCount)
	log.Printf("[DEBUG] NetworkInterfacePostCloneOperation: Layout from source: %d devices", len(devices))

	// Populate the source set as if the devices were orphaned. This give us a
	// base to diff off of.
	log.Printf("[DEBUG] NetworkInterfacePostCloneOperation: Reading existing devices")
	for n, device := range devices {
		m := make(map[string]interface{})
		vd := device.GetVirtualDevice()
		ctlr := l.FindByKey(vd.ControllerKey)
		if ctlr == nil {
			return nil, nil, fmt.Errorf("could not find controller with key %d", vd.Key)
		}
		m["key"] = int(vd.Key)
		var err error
		m["device_address"], err = computeDevAddr(vd, ctlr.(types.BaseVirtualController))
		if err != nil {
			return nil, nil, fmt.Errorf("error computing device address: %s", err)
		}
		r := NewNetworkInterfaceSubresource(c, d, m, nil, n)
		if err := r.Read(l); err != nil {
			return nil, nil, fmt.Errorf("%s: %s", r.Addr(), err)
		}

		srcSet = append(srcSet, r.Data())
	}

	// Now go over our current set, kind of treating it like an apply:
	//
	// * No source data at the index is a create
	// * Source data at the index is an update if it has changed
	// * Data at the source with the same data after patching config data is a
	// no-op, but we still push the device's state
	var spec []types.BaseVirtualDeviceConfigSpec
	var updates []interface{}
	for i, ci := range curSet {
		cm := ci.(map[string]interface{})
		if i > len(srcSet)-1 || srcSet[i] == nil {
			// New device
			r := NewNetworkInterfaceSubresource(c, d, cm, nil, i)
			cspec, err := r.Create(l)
			if err != nil {
				return nil, nil, fmt.Errorf("%s: %s", r.Addr(), err)
			}
			l = applyDeviceChange(l, cspec)
			spec = append(spec, cspec...)
			updates = append(updates, r.Data())
			continue
		}
		sm := srcSet[i].(map[string]interface{})
		nc, err := copystructure.Copy(sm)
		if err != nil {
			return nil, nil, fmt.Errorf("error copying source network interface state data at index %d: %s", i, err)
		}
		nm := nc.(map[string]interface{})
		for k, v := range cm {
			// Skip key and device_address here
			switch k {
			case "key", "device_address":
				continue
			}
			nm[k] = v
		}

		r := NewNetworkInterfaceSubresource(c, d, nm, sm, i)
		if !reflect.DeepEqual(sm, nm) {
			// Update
			cspec, err := r.Update(l)
			if err != nil {
				return nil, nil, fmt.Errorf("%s: %s", r.Addr(), err)
			}
			l = applyDeviceChange(l, cspec)
			spec = append(spec, cspec...)
		}
		updates = append(updates, r.Data())
	}

	// Any other device past the end of the network devices listed in config needs to be removed.
	if len(curSet) < len(srcSet) {
		for i, si := range srcSet[len(curSet):] {
			sm, ok := si.(map[string]interface{})
			if !ok {
				log.Printf("[DEBUG] Extra entry in NIC source list, but not of expected type")
				continue
			}
			r := NewNetworkInterfaceSubresource(c, d, sm, nil, i+len(curSet))
			dspec, err := r.Delete(l)
			if err != nil {
				return nil, nil, fmt.Errorf("%s: %s", r.Addr(), err)
			}
			l = applyDeviceChange(l, dspec)
			spec = append(spec, dspec...)
		}
	}

	log.Printf("[DEBUG] NetworkInterfacePostCloneOperation: Post-clone final resource list: %s", subresourceListString(updates))
	// We are now done! Return the updated device list and config spec. Save updates as well.
	if err := d.Set(subresourceTypeNetworkInterface, updates); err != nil {
		return nil, nil, err
	}
	log.Printf("[DEBUG] NetworkInterfacePostCloneOperation: Device list at end of operation: %s", DeviceListString(l))
	log.Printf("[DEBUG] NetworkInterfacePostCloneOperation: Device config operations from post-clone: %s", DeviceChangeString(spec))
	log.Printf("[DEBUG] NetworkInterfacePostCloneOperation: Operation complete, returning updated spec")
	return l, spec, nil
}

// ReadNetworkInterfaceTypes returns a list of network interface types. This is used
// in the VM data source to discover the types of the NIC drivers on the
// virtual machine. The list is sorted by the order that they would be added in
// if a clone were to be done.
func ReadNetworkInterfaceTypes(l object.VirtualDeviceList) ([]string, error) {
	log.Printf("[DEBUG] ReadNetworkInterfaceTypes: Fetching interface types")
	devices := l.Select(func(device types.BaseVirtualDevice) bool {
		if _, ok := device.(types.BaseVirtualEthernetCard); ok {
			return true
		}
		return false
	})
	log.Printf("[DEBUG] ReadNetworkInterfaceTypes: Network devices located: %s", DeviceListString(devices))
	// Sort the device list, in case it's not sorted already.
	devSort := virtualDeviceListSorter{
		Sort:       devices,
		DeviceList: l,
	}
	sort.Sort(devSort)
	devices = devSort.Sort
	log.Printf("[DEBUG] ReadNetworkInterfaceTypes: Network devices order after sort: %s", DeviceListString(devices))
	var out []string
	for _, device := range devices {
		out = append(out, virtualEthernetCardString(device.(types.BaseVirtualEthernetCard)))
	}
	log.Printf("[DEBUG] ReadNetworkInterfaceTypes: Network types returned: %+v", out)
	return out, nil
}

// ReadNetworkInterfaces returns a list of network interfaces. This is used
// in the VM data source to discover the properties of the network interfaces on the
// virtual machine. The list is sorted by the order that they would be added in
// if a clone were to be done.
func ReadNetworkInterfaces(l object.VirtualDeviceList) ([]map[string]interface{}, error) {
	log.Printf("[DEBUG] ReadNetworkInterfaces: Fetching network interfaces")
	devices := l.Select(func(device types.BaseVirtualDevice) bool {
		if _, ok := device.(types.BaseVirtualEthernetCard); ok {
			return true
		}
		return false
	})
	log.Printf("[DEBUG] ReadNetworkInterface: Network devices located: %s", DeviceListString(devices))
	// Sort the device list, in case it's not sorted already.
	devSort := virtualDeviceListSorter{
		Sort:       devices,
		DeviceList: l,
	}
	sort.Sort(devSort)
	devices = devSort.Sort
	log.Printf("[DEBUG] ReadNetworkInterfaceTypes: Network devices order after sort: %s", DeviceListString(devices))
	var out []map[string]interface{}
	for _, device := range devices {
		m := make(map[string]interface{})

		ethernetCard := device.(types.BaseVirtualEthernetCard).GetVirtualEthernetCard()

		// Determine the network from the backing object
		var networkID string

		switch backing := ethernetCard.Backing.(type) {
		case *types.VirtualEthernetCardNetworkBackingInfo:
			if backing.Network != nil {
				networkID = backing.Network.Value
			}
		case *types.VirtualEthernetCardOpaqueNetworkBackingInfo:
			networkID = backing.OpaqueNetworkId
		case *types.VirtualEthernetCardDistributedVirtualPortBackingInfo:
			networkID = backing.Port.PortgroupKey
		default:
		}

		// Set properties
		switch v := interface{}(device).(type) {
		case *types.VirtualSriovEthernetCard:
			sriovBacking := v.SriovBacking
			if sriovBacking.PhysicalFunctionBacking != nil {
				m["physical_function"] = sriovBacking.PhysicalFunctionBacking.Id
			}
		default:
			// Set the bandwidth properties
			m["bandwidth_limit"] = ethernetCard.ResourceAllocation.Limit
			m["bandwidth_reservation"] = ethernetCard.ResourceAllocation.Reservation
			m["bandwidth_share_level"] = ethernetCard.ResourceAllocation.Share.Level
			m["bandwidth_share_count"] = ethernetCard.ResourceAllocation.Share.Shares
		}

		m["adapter_type"] = virtualEthernetCardString(device.(types.BaseVirtualEthernetCard))
		m["mac_address"] = ethernetCard.MacAddress
		m["network_id"] = networkID

		out = append(out, m)
	}
	log.Printf("[DEBUG] ReadNetworkInterfaces: Network interfaces returned: %+v", out)
	return out, nil
}

// baseVirtualEthernetCardToBaseVirtualDevice converts a
// BaseVirtualEthernetCard value into a BaseVirtualDevice.
func baseVirtualEthernetCardToBaseVirtualDevice(v types.BaseVirtualEthernetCard) types.BaseVirtualDevice {
	switch t := v.(type) {
	case *types.VirtualE1000:
		return types.BaseVirtualDevice(t)
	case *types.VirtualE1000e:
		return types.BaseVirtualDevice(t)
	case *types.VirtualPCNet32:
		return types.BaseVirtualDevice(t)
	case *types.VirtualSriovEthernetCard:
		return types.BaseVirtualDevice(t)
	case *types.VirtualVmxnet2:
		return types.BaseVirtualDevice(t)
	case *types.VirtualVmxnet3:
		return types.BaseVirtualDevice(t)
	case *types.VirtualVmxnet3Vrdma:
		return types.BaseVirtualDevice(t)
	}
	panic(fmt.Errorf("unknown ethernet card type %T", v))
}

// baseVirtualDeviceToBaseVirtualEthernetCard converts a BaseVirtualDevice
// value into a BaseVirtualEthernetCard.
func baseVirtualDeviceToBaseVirtualEthernetCard(v types.BaseVirtualDevice) (types.BaseVirtualEthernetCard, error) {
	if bve, ok := v.(types.BaseVirtualEthernetCard); ok {
		return bve, nil
	}
	return nil, fmt.Errorf("device is not a network device (%T)", v)
}

// virtualEthernetCardString prints a string representation of the ethernet device passed in.
func virtualEthernetCardString(d types.BaseVirtualEthernetCard) string {
	switch d.(type) {
	case *types.VirtualE1000:
		return networkInterfaceSubresourceTypeE1000
	case *types.VirtualE1000e:
		return networkInterfaceSubresourceTypeE1000e
	case *types.VirtualPCNet32:
		return networkInterfaceSubresourceTypePCNet32
	case *types.VirtualSriovEthernetCard:
		return networkInterfaceSubresourceTypeSriov
	case *types.VirtualVmxnet2:
		return networkInterfaceSubresourceTypeVmxnet2
	case *types.VirtualVmxnet3:
		return networkInterfaceSubresourceTypeVmxnet3
	case *types.VirtualVmxnet3Vrdma:
		return networkInterfaceSubresourceTypeVRdma
	}
	return networkInterfaceSubresourceTypeUnknown
}

// Create creates a vsphere_virtual_machine network_interface sub-resource.
func (r *NetworkInterfaceSubresource) Create(l object.VirtualDeviceList) ([]types.BaseVirtualDeviceConfigSpec, error) {
	log.Printf("[DEBUG] %s: Running create", r)
	var spec []types.BaseVirtualDeviceConfigSpec
	ctlr, err := r.ControllerForCreateUpdate(l, SubresourceControllerTypePCI, 0)
	if err != nil {
		return nil, err
	}

	// govmomi has helpers that allow the easy fetching of a network's backing
	// info, once we actually know what that backing is. Set all of that stuff up
	// now.
	net, err := network.FromID(r.client, r.Get("network_id").(string))
	if err != nil {
		return nil, err
	}
	bctx, bcancel := context.WithTimeout(context.Background(), provider.DefaultAPITimeout)
	defer bcancel()
	backing, err := net.EthernetCardBackingInfo(bctx)
	if err != nil {
		return nil, err
	}

	device, err := l.CreateEthernetCard(r.Get("adapter_type").(string), backing)
	if err != nil {
		return nil, err
	}

	// Add SRIOV physical function if this network interface resource has it defined
	if len(r.Get("physical_function").(string)) > 0 {
		device, err = r.addPhysicalFunction(device)
	}

	// SRIOV device creation requires a restart
	if r.Get("adapter_type").(string) == networkInterfaceSubresourceTypeSriov {
		log.Printf("[DEBUG] create: SR-IOV, set to restart")
		r.SetRestart("<device sriov create>")
	}

	// Ensure the device starts connected
	err = l.Connect(device)
	if err != nil && !strings.Contains(err.Error(), "is not connectable") {
		return nil, err
	}

	// Set base-level card bits now
	card := device.(types.BaseVirtualEthernetCard).GetVirtualEthernetCard()
	card.Key = l.NewKey()

	// Set the rest of the settings here.
	if r.Get("use_static_mac").(bool) {
		card.AddressType = string(types.VirtualEthernetCardMacTypeManual)
		card.MacAddress = r.Get("mac_address").(string)
	}

	version := viapi.ParseVersionFromClient(r.client)
	if (version.Newer(viapi.VSphereVersion{Product: version.Product, Major: 6}) && r.Get("adapter_type") != networkInterfaceSubresourceTypeSriov) {
		bandwidth_limit := structure.Int64Ptr(-1)
		bandwidth_reservation := structure.Int64Ptr(0)
		bandwidth_share_level := types.SharesLevelNormal
		if r.Get("bandwidth_limit") != nil {
			bandwidth_limit = structure.Int64Ptr(int64(r.Get("bandwidth_limit").(int)))
		}
		if r.Get("bandwidth_reservation") != nil {
			bandwidth_reservation = structure.Int64Ptr(int64(r.Get("bandwidth_reservation").(int)))
		}
		if r.Get("bandwidth_share_level") != nil {
			bandwidth_share_level = types.SharesLevel(r.Get("bandwidth_share_level").(string))
		}

		alloc := &types.VirtualEthernetCardResourceAllocation{
			Limit:       bandwidth_limit,
			Reservation: bandwidth_reservation,
			Share: types.SharesInfo{
				Shares: int32(r.Get("bandwidth_share_count").(int)),
				Level:  bandwidth_share_level,
			},
		}
		card.ResourceAllocation = alloc
	}

	// Done here. Save ID, push the device to the new device list and return.
	if err := r.SaveDevIDs(device, ctlr); err != nil {
		return nil, err
	}
	dspec, err := object.VirtualDeviceList{device}.ConfigSpec(types.VirtualDeviceConfigSpecOperationAdd)
	if err != nil {
		return nil, err
	}
	spec = append(spec, dspec...)
	log.Printf("[DEBUG] %s: Device config operations from create: %s", r, DeviceChangeString(spec))
	log.Printf("[DEBUG] %s: Create finished", r)
	return spec, nil
}

// Read reads a vsphere_virtual_machine network_interface sub-resource.
func (r *NetworkInterfaceSubresource) Read(l object.VirtualDeviceList) error {
	log.Printf("[DEBUG] %s: Reading state", r)
	vd, err := r.FindVirtualDevice(l)
	if err != nil {
		return fmt.Errorf("cannot find network device: %s", err)
	}

	device, err := baseVirtualDeviceToBaseVirtualEthernetCard(vd)
	if err != nil {
		return err
	}

	// Determine the interface type, and set the field appropriately. As a fallback,
	// we actually set adapter_type here to "unknown" if we don't support the NIC
	// type, as we can determine all of the other settings without having to
	// worry about the adapter type, and on update, the adapter type will be
	// rectified by removing the existing NIC and replacing it with a new one.
	r.Set("adapter_type", virtualEthernetCardString(device))

	// The rest of the information we need to get by reading the attributes off
	// the base card object.
	card := device.GetVirtualEthernetCard()

	// Determine the network
	var netID string
	switch backing := card.Backing.(type) {
	case *types.VirtualEthernetCardNetworkBackingInfo:
		if backing.Network == nil {
			return fmt.Errorf("could not determine network information from NIC backing")
		}
		netID = backing.Network.Value
	case *types.VirtualEthernetCardOpaqueNetworkBackingInfo:
		onet, err := nsx.OpaqueNetworkFromNetworkID(r.client, backing.OpaqueNetworkId)
		if err != nil {
			return err
		}
		netID = onet.Reference().Value
	case *types.VirtualEthernetCardDistributedVirtualPortBackingInfo:
		pg, err := dvportgroup.FromKey(r.client, backing.Port.SwitchUuid, backing.Port.PortgroupKey)
		if err != nil {
			if strings.Contains(err.Error(), "The object or item referred to could not be found") {
				netID = ""
			} else {
				return err
			}
		} else {
			netID = pg.Reference().Value
		}
	default:
		return fmt.Errorf("unknown network interface backing %T", card.Backing)
	}

	switch v := interface{}(device).(type) {
	case *types.VirtualSriovEthernetCard:
		sriovBacking := v.SriovBacking
		if sriovBacking.PhysicalFunctionBacking == nil {
			return fmt.Errorf("cannot determine SR-IOV physical_function from NIC")
		}
		r.Set("physical_function", sriovBacking.PhysicalFunctionBacking.Id)
		log.Printf("[DEBUG] Read: Adapter type is SR-IOV. Read the physical function and set to %s", r.Get("physical_function"))
	default:
		log.Printf("[DEBUG] Read: Adapter type not SR-IOV")
	}

	r.Set("network_id", netID)
	r.Set("use_static_mac", card.AddressType == string(types.VirtualEthernetCardMacTypeManual))
	r.Set("mac_address", card.MacAddress)

	version := viapi.ParseVersionFromClient(r.client)
	if version.Newer(viapi.VSphereVersion{Product: version.Product, Major: 6}) {
		if r.Get("adapter_type") != networkInterfaceSubresourceTypeSriov {
			if card.ResourceAllocation != nil {
				r.Set("bandwidth_limit", card.ResourceAllocation.Limit)
				r.Set("bandwidth_reservation", card.ResourceAllocation.Reservation)
				r.Set("bandwidth_share_count", card.ResourceAllocation.Share.Shares)
				r.Set("bandwidth_share_level", card.ResourceAllocation.Share.Level)
			}
		} else {
			// SRIOV adapters don't support bandwidth properties. Set them to the defaults on the read resource
			// to ensure that import and such work (as the schema has defaults for them). The bandwidth_share_count
			// is computed and has no default, so doesn't need setting.
			r.Set("bandwidth_limit", defaultBandwidthLimit)
			r.Set("bandwidth_reservation", defaultBandwidthReservation)
			r.Set("bandwidth_share_level", defaultBandwidthShareLevel)
		}
	}

	// Save the device key and address data
	ctlr, err := findControllerForDevice(l, vd)
	if err != nil {
		return err
	}
	if err := r.SaveDevIDs(vd, ctlr); err != nil {
		return err
	}
	log.Printf("[DEBUG] %s: Read finished (key and device address may have changed)", r)
	return nil
}

// Update updates a vsphere_virtual_machine network_interface sub-resource.
func (r *NetworkInterfaceSubresource) Update(l object.VirtualDeviceList) ([]types.BaseVirtualDeviceConfigSpec, error) {
	log.Printf("[DEBUG] %s: Beginning update", r)
	vd, err := r.FindVirtualDevice(l)
	if err != nil {
		return nil, fmt.Errorf("cannot find network device: %s", err)
	}
	device, err := baseVirtualDeviceToBaseVirtualEthernetCard(vd)
	if err != nil {
		return nil, err
	}

	// We maintain the final update spec in place, versus just the simple device
	// list, to support deletion of virtual devices so that they can replaced by
	// ones with different device types.
	var spec []types.BaseVirtualDeviceConfigSpec

	// A change in adapter_type or physical_function is essentially a ForceNew.
	// We would normally veto
	// this, but network devices are not extremely mission critical if they go
	// away, so we can support in-place modification of them in configuration by
	// just pushing a delete of the old device and adding a new version of the
	// device, with the old device unit number preserved so that it (hopefully)
	// gets the same device position as its previous incarnation, allowing old
	// device aliases to work, etc.
	if r.HasChange("adapter_type") || physicalFunctionChanged(r) {
		if r.HasChange("adapter_type") {
			log.Printf("[DEBUG] %s: Device type changing to %s, re-creating device", r, r.Get("adapter_type").(string))
		} else if r.HasChange("physical_function") {
			log.Printf("[DEBUG] %s: SR-IOV physical function changing to %s, re-creating device", r, r.Get("physical_function").(string))
		}
		card := device.GetVirtualEthernetCard()
		newDevice, err := l.CreateEthernetCard(r.Get("adapter_type").(string), card.Backing)
		if err != nil {
			return nil, err
		}
		if len(r.Get("physical_function").(string)) > 0 {
			newDevice, err = r.addPhysicalFunction(newDevice)
		}

		r.Set("key", l.NewKey())
		// If VMware Tools is not running, this operation requires a reboot
		if r.rdd.Get("vmware_tools_status").(string) != string(types.VirtualMachineToolsRunningStatusGuestToolsRunning) {
			r.SetRestart("<adapter_type>")
		}

		if r.HasChange("physical_function") {
			// If SRIOV physical function has changed, this operation requires a reboot
			r.SetRestart("<physical_function>")
		}
		// Push the delete of the old device
		bvd := baseVirtualEthernetCardToBaseVirtualDevice(device)
		dspec, err := object.VirtualDeviceList{bvd}.ConfigSpec(types.VirtualDeviceConfigSpecOperationRemove)
		if err != nil {
			return nil, err
		}
		spec = append(spec, dspec...)
		// new device now becomes the old device and we proceed with the rest
		device, err = baseVirtualDeviceToBaseVirtualEthernetCard(newDevice)
		if err != nil {
			return nil, err
		}
	}

	card := device.GetVirtualEthernetCard()

	// Has the backing changed?
	if r.HasChange("network_id") {
		net, err := network.FromID(r.client, r.Get("network_id").(string))
		if err != nil {
			return nil, err
		}
		bctx, bcancel := context.WithTimeout(context.Background(), provider.DefaultAPITimeout)
		defer bcancel()
		backing, err := net.EthernetCardBackingInfo(bctx)
		if err != nil {
			return nil, err
		}
		card.Backing = backing
	}

	if r.HasChange("use_static_mac") {
		if r.Get("use_static_mac").(bool) {
			card.AddressType = string(types.VirtualEthernetCardMacTypeManual)
			card.MacAddress = r.Get("mac_address").(string)
		} else {
			// If we've gone from a static MAC address to a auto-generated one, we need
			// to check what address type we need to set things to.
			if r.client.ServiceContent.About.ApiType != "VirtualCenter" {
				// ESXi - type is "generated"
				card.AddressType = string(types.VirtualEthernetCardMacTypeGenerated)
			} else {
				// vCenter - type is "assigned"
				card.AddressType = string(types.VirtualEthernetCardMacTypeAssigned)
			}
			card.MacAddress = ""
		}
	}

	version := viapi.ParseVersionFromClient(r.client)
	if (version.Newer(viapi.VSphereVersion{Product: version.Product, Major: 6}) && r.Get("adapter_type") != networkInterfaceSubresourceTypeSriov) {
		bandwidth_limit := structure.Int64Ptr(-1)
		bandwidth_reservation := structure.Int64Ptr(0)
		bandwidth_share_level := types.SharesLevelNormal
		if r.Get("bandwidth_limit") != nil {
			bandwidth_limit = structure.Int64Ptr(int64(r.Get("bandwidth_limit").(int)))
		}
		if r.Get("bandwidth_reservation") != nil {
			bandwidth_reservation = structure.Int64Ptr(int64(r.Get("bandwidth_reservation").(int)))
		}
		if r.Get("bandwidth_share_level") != nil {
			bandwidth_share_level = types.SharesLevel(r.Get("bandwidth_share_level").(string))
		}

		alloc := &types.VirtualEthernetCardResourceAllocation{
			Limit:       bandwidth_limit,
			Reservation: bandwidth_reservation,
			Share: types.SharesInfo{
				Shares: int32(r.Get("bandwidth_share_count").(int)),
				Level:  bandwidth_share_level,
			},
		}
		card.ResourceAllocation = alloc
	}

	var op types.VirtualDeviceConfigSpecOperation
	if card.Key < 0 {
		// Negative key means that we are re-creating this device
		op = types.VirtualDeviceConfigSpecOperationAdd
	} else {
		op = types.VirtualDeviceConfigSpecOperationEdit
	}

	bvd := baseVirtualEthernetCardToBaseVirtualDevice(device)
	uspec, err := object.VirtualDeviceList{bvd}.ConfigSpec(op)
	if err != nil {
		return nil, err
	}
	spec = append(spec, uspec...)
	log.Printf("[DEBUG] %s: Device config operations from update: %s", r, DeviceChangeString(spec))
	log.Printf("[DEBUG] %s: Update complete", r)
	return spec, nil
}

// Add SRIOV physical function setting the device to a VirtualSriovEthernetCard
// and by adding VirtualSriovEthernetCardSriovBackingInfo
func (r *NetworkInterfaceSubresource) addPhysicalFunction(device types.BaseVirtualDevice) (types.BaseVirtualDevice, error) {
	// Based off https://vdc-download.vmware.com/vmwb-repository/dcr-public/b50dcbbf-051d-4204-a3e7-e1b618c1e384/538cf2ec-b34f-4bae-a332-3820ef9e7773/vim.vm.device.VirtualSriovEthernetCard.SriovBackingInfo.html
	log.Printf("[DEBUG] physical function detected")
	var d2 interface{} = device

	// These seem to be the correct DeviceId, SystemId and VendorId settings if you
	// investigate a manually created vSphere SRIOV network interface
	physical_function_conf := &types.VirtualPCIPassthroughDeviceBackingInfo{
		Id:       r.Get("physical_function").(string),
		DeviceId: "0",
		SystemId: "BYPASS",
		VendorId: 0,
	}
	sriov_conf := &types.VirtualSriovEthernetCardSriovBackingInfo{
		PhysicalFunctionBacking: physical_function_conf,
	}

	device = &types.VirtualSriovEthernetCard{
		VirtualEthernetCard: *d2.(types.BaseVirtualEthernetCard).GetVirtualEthernetCard(),
		SriovBacking:        sriov_conf,
	}

	return device, nil
}

// Delete deletes a vsphere_virtual_machine network_interface sub-resource.
func (r *NetworkInterfaceSubresource) Delete(l object.VirtualDeviceList) ([]types.BaseVirtualDeviceConfigSpec, error) {
	log.Printf("[DEBUG] %s: Beginning delete", r)
	vd, err := r.FindVirtualDevice(l)
	if err != nil {
		return nil, fmt.Errorf("cannot find network device: %s", err)
	}
	device, err := baseVirtualDeviceToBaseVirtualEthernetCard(vd)

	if err != nil {
		return nil, err
	}
	// If VMware Tools is not running, this operation requires a reboot
	if r.rdd.Get("vmware_tools_status").(string) != string(types.VirtualMachineToolsRunningStatusGuestToolsRunning) {
		r.SetRestart("<device delete>")
	}
	// Sriov network interfaces require a reboot to delete
	if r.Get("adapter_type").(string) == networkInterfaceSubresourceTypeSriov {
		r.SetRestart("<sriov device delete>")
	}

	bvd := baseVirtualEthernetCardToBaseVirtualDevice(device)
	spec, err := object.VirtualDeviceList{bvd}.ConfigSpec(types.VirtualDeviceConfigSpecOperationRemove)
	if err != nil {
		return nil, err
	}
	log.Printf("[DEBUG] %s: Device config operations from update: %s", r, DeviceChangeString(spec))
	log.Printf("[DEBUG] %s: Delete completed", r)
	return spec, nil
}

// Bandwidth settings are irrelevant for SR-IOV interfaces so we should warn if the user is trying
// to set them.
func (r *NetworkInterfaceSubresource) blockBandwidthSettingsSriov() error {
	if r.Get("adapter_type") == networkInterfaceSubresourceTypeSriov {
		if r.Get("bandwidth_limit") != defaultBandwidthLimit ||
			r.Get("bandwidth_reservation") != defaultBandwidthReservation ||
			r.Get("bandwidth_share_level") != defaultBandwidthShareLevel {
			return fmt.Errorf("invalid bandwidth properties on sriov network interface. " +
				"bandwidth settings do not apply to sriov interfaces")
		}

		return nil
	}

	return nil
}

// ValidateDiff performs any complex validation of an individual
// network_interface sub-resource that can't be done in schema alone.
func (r *NetworkInterfaceSubresource) ValidateDiff() error {
	log.Printf("[DEBUG] %s: Beginning diff validation", r)

	// Ensure that network resource allocation options are only set on vSphere
	// 6.0 and higher. They are not relevant for SR-IOV networks in either case.
	version := viapi.ParseVersionFromClient(r.client)
	if (version.Older(viapi.VSphereVersion{Product: version.Product, Major: 6}) &&
		r.Get("adapter_type") != networkInterfaceSubresourceTypeSriov) {
		if err := r.restrictResourceAllocationSettings(); err != nil {
			return err
		}
	}

	// Ensure physical adapter is set on all (and only on) SR-IOV NICs
	if r.Get("adapter_type").(string) == networkInterfaceSubresourceTypeSriov {
		if len(r.Get("physical_function").(string)) == 0 {
			return fmt.Errorf("physical_function must be set on SR-IOV Network interface")
		}
	} else {
		if len(r.Get("physical_function").(string)) > 0 {
			return fmt.Errorf("cannot set physical_function on non SR-IOV network interface")
		}

	}

	// Don't allow bandwidth settings on SRIOV that aren't the defaults
	if err := r.blockBandwidthSettingsSriov(); err != nil {
		return err
	}

	log.Printf("[DEBUG] %s: Diff validation complete", r)
	return nil
}

func (r *NetworkInterfaceSubresource) restrictResourceAllocationSettings() error {
	rs := NetworkInterfaceSubresourceSchema()
	keys := []string{
		"bandwidth_limit",
		"bandwidth_reservation",
		"bandwidth_share_level",
		"bandwidth_share_count",
	}
	for _, key := range keys {
		expected := rs[key].Default
		if expected == nil {
			expected = rs[key].ZeroValue()
		}
		if r.Get(key) != expected {
			return fmt.Errorf("%s requires vSphere 6.0 or higher", key)
		}
	}
	return nil
}

func physicalFunctionChanged(r *NetworkInterfaceSubresource) bool {
	old, n := r.GetChange("physical_function")
	var oldVal, newVal string
	if old == nil {
		oldVal = ""
	} else {
		oldVal = old.(string)
	}

	if n == nil {
		newVal = ""
	} else {
		newVal = n.(string)
	}

	return newVal != oldVal
}
