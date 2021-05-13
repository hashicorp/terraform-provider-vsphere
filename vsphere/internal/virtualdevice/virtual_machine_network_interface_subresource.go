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

// networkInterfacePciDeviceOffset defines the PCI offset for virtual NICs on a vSphere PCI bus.
const networkInterfacePciDeviceOffset = 7

// Sunny this is all the stuff from https://github.com/vmware/govmomi/issues/2060
// sriovNetworkInterfacePciDeviceOffset defines the PCI offset for virtual SR-IOV NICs on a vSphere PCI bus.
// sriov NICs have unitNumber 45, 44 etc.
// Sunny changed from 48 to 45
const sriovNetworkInterfacePciDeviceOffset = 45

const maxNetworkInterfaceCount = 10

//const sriovNetworkInterfacePciDeviceOffset = 48

const (
	networkInterfaceSubresourceTypeE1000   = "e1000"
	networkInterfaceSubresourceTypeE1000e  = "e1000e"
	networkInterfaceSubresourceTypePCNet32 = "pcnet32"
	networkInterfaceSubresourceTypeSriov   = "sriov"
	networkInterfaceSubresourceTypeVmxnet2 = "vmxnet2"
	networkInterfaceSubresourceTypeVmxnet3 = "vmxnet3"
	networkInterfaceSubresourceTypeUnknown = "unknown"
)

// Sunny allow "sriov" as adapter_type in main.tf I think
var networkInterfaceSubresourceTypeAllowedValues = []string{
	networkInterfaceSubresourceTypeE1000,
	networkInterfaceSubresourceTypeE1000e,
	networkInterfaceSubresourceTypeSriov,
	networkInterfaceSubresourceTypeVmxnet3,
}

var networkInterfaceSubresourceMACAddressTypeAllowedValues = []string{
	string(types.VirtualEthernetCardMacTypeManual),
}

// NetworkInterfaceSubresourceSchema returns the schema for the disk
// sub-resource.
func NetworkInterfaceSubresourceSchema() map[string]*schema.Schema {
	s := map[string]*schema.Schema{
		// VirtualEthernetCardResourceAllocation
		"bandwidth_limit": {
			Type:         schema.TypeInt,
			Optional:     true,
			Default:      -1,
			Description:  "The upper bandwidth limit of this network interface, in Mbits/sec.",
			ValidateFunc: validation.IntAtLeast(-1),
		},
		"bandwidth_reservation": {
			Type:         schema.TypeInt,
			Optional:     true,
			Default:      0,
			Description:  "The bandwidth reservation of this network interface, in Mbits/sec.",
			ValidateFunc: validation.IntAtLeast(0),
		},
		"bandwidth_share_level": {
			Type:         schema.TypeString,
			Optional:     true,
			Default:      string(types.SharesLevelNormal),
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
			Description:  "The controller type. Can be one of e1000, e1000e, sriov, or vmxnet3.",
			ValidateFunc: validation.StringInSlice(networkInterfaceSubresourceTypeAllowedValues, false),
		},
		// Sunny saying that physical_function is valid in the schema as a String.
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
	log.Printf("ANDREW NewNetworkInterfaceSubresource Index %d and data %s", sr.Index, sr.data)
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

	log.Printf("[ANDREW] o is %s, ni s %", o, n)
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
				log.Printf("ANDREW NetworkInterfaceApplyOperation Found key %s in new map matching key %s in old map", nm["key"], om["key"])
				continue nextOld
			} else {
				log.Printf("ANDREW NetworkInterfaceApplyOperation New map key %s doesn't match old map key %s. If none found delete", nm["key"], om["key"])
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
			} else {
				log.Printf("[DEBUG] NetworkInterfaceApplyOperation: key %d looks to have changed", nm["key"].(int))
				log.Printf("[ANDREWNEW] nm is %s, om in %s", nm, om)
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
		// Sunny the r.Index is the last parameter here, namely n
		r := NewNetworkInterfaceSubresource(c, d, nm, nil, n)
		cspec, err := r.Create(l)
		if err != nil {
			return nil, nil, fmt.Errorf("%s: %s", r.Addr(), err)
		}
		// Sunny This updates the VirtualDeviceList l with the newly created resource cspec, and l is eventually returned
		// from this function
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
// the networks interfaces attached  to this resource.
//
// This functions similar to NetworkInterfaceApplyOperation, but nothing to
// change is returned, all necessary values are just set and committed to
// state.
func NetworkInterfaceRefreshOperation(d *schema.ResourceData, c *govmomi.Client, l object.VirtualDeviceList) error {
	log.Printf("[DEBUG] NetworkInterfaceRefreshOperation: Beginning refresh")
	// Sunny find all the devices that are of type VirtualEthernetCard (the Base prefix is added to the if.go interface don't know why)
	devices := l.Select(func(device types.BaseVirtualDevice) bool {
		if _, ok := device.(types.BaseVirtualEthernetCard); ok {
			return true
		}
		return false
	})
	// Sunny [DEBUG] NetworkInterfaceRefreshOperation: Network devices located: ethernet-37,ethernet-38,ethernet-0,ethernet-1: timestamp=2021-04-29T15:06:58.116+0100
	log.Printf("[DEBUG] NetworkInterfaceRefreshOperation: Network devices located: %s", DeviceListString(devices))
	// Sunny not sure what this curSet is but presumably reading the terraform state what is actually there?
	curSet := d.Get(subresourceTypeNetworkInterface).([]interface{})
	// Sunny Current resource set from state: (key -201 at pci:0:7),(key -202 at pci:0:8),(key -203 at pci:0:46),(key -204 at pci:0:45): timestamp=2021-04-29T15:06:58.116+0100
	log.Printf("[DEBUG] NetworkInterfaceRefreshOperation: Current resource set from state: %s", subresourceListString(curSet))
	// Sunny all these
	// nicUnitRange calculates a range of units given a certain VirtualDeviceList,
	// which should be network interfaces.  It's used in network interface refresh
	// logic to determine how many subresources may end up in state.
	// It returns the COUNT of all virtual devices with a unit number > 7
	urange, err := nicUnitRange(devices)
	if err != nil {
		return fmt.Errorf("error calculating network device range: %s", err)
	}
	//nonSriovRange, _ := nonSriovNicUnitRange(devices)
	//sriovRange, _ := sriovNicUnitRange(devices)
	// Sunny make an array of Anys of length urange? i.e. the count of deviceswith unit number > 7 in the resourcedata schema.
	// 4 devices over a 3 unit range: timestamp=2021-04-29T15:06:58.116+0100
	newSetAll := make([]interface{}, urange)
	newSetNonSriov := make([]interface{}, maxNetworkInterfaceCount)
	newSetSriov := make([]interface{}, maxNetworkInterfaceCount)
	log.Printf("ANDREW test\n")
	log.Printf("[DEBUG] NetworkInterfaceRefreshOperation: %d devices over a %d unit range", len(devices), urange)
	log.Printf("ANDREW devices is %s", devices)
	// First check for negative keys. These are freshly added devices that are
	// usually coming into read post-create.
	//
	// If we find what we are looking for, we remove the device from the working
	// set so that we don't try and process it in the next few passes.
	log.Printf("[DEBUG] NetworkInterfaceRefreshOperation: Looking for freshly-created resources to read in")
	log.Printf("ANDREW curset is definitely: \n%s\n", curSet)
	for n, item := range curSet {
		log.Printf("ANDREW into loop, n is %d and item is %s", n, item)
		// Sunny Check that item value is of type map[string]interface{}
		m := item.(map[string]interface{})
		if m["key"].(int) < 1 {
			log.Printf("ANDREW the map key is %d so create a new Net If Subresource with index %d", m["key"].(int), n)
			r := NewNetworkInterfaceSubresource(c, d, m, nil, n)
			// Sunny FindVirtualDevice: Looking for device with address pci:0:7: timestamp=2021-04-29T15:08:17.958+0100
			// FindVirtualDevice: Device found: ethernet-0: timestamp=2021-04-29T15:08:17.958+0100
			// network_interface.0 (key 4000 at pci:0:7): Read finished (key and device address may have changed): timestamp=2021-04-29T15:08:17.998+0100
			// Repeat, starting with network_interface.1 (key -202 at pci:0:8): Reading state: timestamp=2021-04-29T15:08:17.998+0100 for the next one.
			// Then FindVirtualDevice: Looking for device with address pci:0:46: timestamp=2021-04-29T15:08:18.069+0100
			// Error: network_interface.2: cannot find network device: invalid device result - 0 results returned (expected 1): controller key 'd', disk number: 46
			if err := r.Read(l); err != nil {
				return fmt.Errorf("%s: %s", r.Addr(), err)
			} else {
				log.Printf("ANDREW I have Read the resource %s ok", r.Addr())
			}

			// Sunny ?? Network devices located: ethernet-37,ethernet-38,ethernet-0,ethernet-1: timestamp=2021-04-29T15:08:17.958+0100
			if r.Get("key").(int) < 1 {
				// This should not have happened - if it did, our device
				// creation/update logic failed somehow that we were not able to track.
				return fmt.Errorf("device %d with address %s still unaccounted for after update/read", r.Get("key").(int), r.Get("device_address").(string))
			}
			// Sunny work out what this does

			_, _, idx, err := splitDevAddr(r.Get("device_address").(string))
			log.Printf("ANDREW found id %d for device_address %s", idx, r.Get("device_address").(string))
			if err != nil {
				return fmt.Errorf("%s: error parsing device address: %s", r, err)
			}

			// newSet is a list of interface with the first interfaces the non-SRIOV interfaces
			// and the last few interfaces all the non-SRIOV interfaces
			log.Printf("ANDREW the resource adapter type is %s and r is %s\n", r.Get("adapter_type").(string), r)
			if r.Get("adapter_type").(string) != networkInterfaceSubresourceTypeSriov {
				//newSet[idx-networkInterfacePciDeviceOffset] = r.Data()
				newSetNonSriov[idx-networkInterfacePciDeviceOffset] = r.Data()
				log.Printf("ANDREW  I have added VMXNET to new set at index %d with %s", idx-networkInterfacePciDeviceOffset, r.Addr())
			} else {
				// Sunny urange is slice of capacity <count of networks> so if 4 urange will have these indexes [0, 1, 2, 3]
				// idx is the unitNumber from the device address, e.g. pci:0:45 will have unit number 45
				// sriovNet..Offset is currently 48 but I think should be 45
				// Currently if idx is 45 and urange is 4, this will make newSet[4+45-48] = data which is newSet[1]
				// Think this should be offset 45 and newSet[urange-1+idx-sriov..Offset] = newSet[4-1+45-45] = newSet[3]
				// For idx 44 this would be 4-1+45-44] = newSet[2]. Think that is right
				// Update - it wasn't right because if 45 down were missing and we only have unit 40 left, this would go off the
				// end of the Array - we'd try to set newSet[5] on an array of length 3. I have fixed.
				// This creates a slice of newSetSriov with elements populated in order of unitNumber 45, 44, 43, 42, 41
				newSetSriov[sriovNetworkInterfacePciDeviceOffset-idx] = r.Data()
				//newSet[urange-1+idx-sriovNetworkInterfacePciDeviceOffset] = r.Data()
				//OLDnewSet[urange+idx-sriovNetworkInterfacePciDeviceOffset] = r.Data()
				//log.Printf("ANDREW  I have added SRIOV to new set at index %d with %s", urange-1+idx-sriovNetworkInterfacePciDeviceOffset, r.Addr())
				log.Printf("ANDREW  I have added SRIOV to SRIOV new set at index %d with %s", sriovNetworkInterfacePciDeviceOffset-idx, r.Addr())
			}
			log.Printf("newSets are now %s and %s", subresourceListString(newSetNonSriov), subresourceListString(newSetSriov))
			// Sunny remove the device r from the list of devices
			for i := 0; i < len(devices); i++ {
				device := devices[i]
				if device.GetVirtualDevice().Key == int32(r.Get("key").(int)) {
					log.Printf("ANDREW removing new device from devices with key %d from index %d", int32(r.Get("key").(int)), i+1)
					devices = append(devices[:i], devices[i+1:]...)
					i--
					log.Printf("ANDREW devices is now %s", DeviceListString(devices))
				}
			}
			log.Printf("ANDREW curset is \n%s\n", curSet)
		}
	}
	log.Printf("ANDREW out of loop")
	log.Printf("[DEBUG] NetworkInterfaceRefreshOperation: Network devices after freshly-created device search: %s", DeviceListString(devices))
	log.Printf("[DEBUG] NetworkInterfaceRefreshOperation: Resource sets to write after freshly-created device search: non-SRIOV %s and SRIOV %s", subresourceListString(newSetNonSriov), subresourceListString(newSetSriov))
	//log.Printf("[DEBUG] NetworkInterfaceRefreshOperation: Resource set to write after freshly-created device search: %s", subresourceListString(newSet))

	// Go over the remaining devices, refresh via key, and then remove their
	// entries as well.
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
			// Done reading, push this onto our new set and remove the device from
			// the list
			_, _, idx, err := splitDevAddr(r.Get("device_address").(string))
			if err != nil {
				return fmt.Errorf("%s: error parsing device address: %s", r, err)
			}
			if r.Get("adapter_type").(string) != networkInterfaceSubresourceTypeSriov {
				newSetNonSriov[idx-networkInterfacePciDeviceOffset] = r.Data()
			} else {
				// Sunny same here
				// @@@TODO
				// idx = 44, urange = 3, this goes at 3-1+44-45 which = 3 which isn't in the range
				//newSet[urange-1+idx-sriovNetworkInterfacePciDeviceOffset] = r.Data()
				newSetSriov[sriovNetworkInterfacePciDeviceOffset-idx] = r.Data()
				log.Printf("ANDREW NetworkInterfaceRefreshOperation Add known resource %s PCI ID %d to newSetSriov at index %d", r.Addr(), idx, sriovNetworkInterfacePciDeviceOffset-idx)
				//newSet[urange+idx-sriovNetworkInterfacePciDeviceOffset] = r.Data()

			}

			devices = append(devices[:i], devices[i+1:]...)
			i--
		}
	}
	log.Printf("[DEBUG] NetworkInterfaceRefreshOperation: Resource sets to write after known device search: non-SRIOV %s and SRIOV %s", subresourceListString(newSetNonSriov), subresourceListString(newSetSriov))
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
		_, _, idx, err := splitDevAddr(r.Get("device_address").(string))
		if err != nil {
			return fmt.Errorf("%s: error parsing device address: %s", r, err)
		}

		if r.Get("adapter_type").(string) != networkInterfaceSubresourceTypeSriov {
			//newSet[idx-networkInterfacePciDeviceOffset] = r.Data()
			newSetNonSriov[idx-networkInterfacePciDeviceOffset] = r.Data()
		} else {
			// Sunny same here @@TODO
			//newSet[urange-1+idx-sriovNetworkInterfacePciDeviceOffset] = r.Data()
			//OLDnewSet[urange+idx-sriovNetworkInterfacePciDeviceOffset] = r.Data()
			newSetSriov[sriovNetworkInterfacePciDeviceOffset-idx] = r.Data()
			log.Printf("ANDREW NetworkInterfaceRefreshOperation Add orphaned resource %s PCI ID %d to newSetSriov at index %d", r.Addr(), idx, sriovNetworkInterfacePciDeviceOffset-idx)

		}

	}

	// Prune any nils from the new device state. This could potentially happen in
	// edge cases where device unit numbers are not 100% sequential.
	for i := 0; i < len(newSetNonSriov); i++ {
		if newSetNonSriov[i] == nil {
			newSetNonSriov = append(newSetNonSriov[:i], newSetNonSriov[i+1:]...)
			i--
		}
	}
	for i := 0; i < len(newSetSriov); i++ {
		if newSetSriov[i] == nil {
			newSetSriov = append(newSetSriov[:i], newSetSriov[i+1:]...)
			i--
		}
	}
	// Create the newSet of all devices from the combination of first the non-SRIOV devices and then the SRIOV devices
	newSetAll = append(newSetNonSriov, newSetSriov...)

	log.Printf("[DEBUG] NetworkInterfaceRefreshOperation: Resource set to write after adding orphaned devices: %s", subresourceListString(newSetAll))
	log.Printf("[DEBUG] NetworkInterfaceRefreshOperation: Refresh operation complete, sending new resource set")
	return d.Set(subresourceTypeNetworkInterface, newSetAll)
}

// NetworkInterfaceDiffOperation performs operations relevant to managing the
// diff on network_interface sub-resources.
func NetworkInterfaceDiffOperation(d *schema.ResourceDiff, c *govmomi.Client) error {
	// We just need the new values for now, as all we are doing is validating some values based on API version
	//Sunny
	//n := d.Get(subresourceTypeNetworkInterface)
	o, n := d.GetChange(subresourceTypeNetworkInterface)
	ods := o.([]interface{})
	nds := n.([]interface{})
	log.Printf("[DEBUG] NetworkInterfaceDiffOperation: Beginning diff validation")

	for ni, ne := range nds {
		nm := ne.(map[string]interface{})
		if len(ods) > ni {
			oe := ods[ni]
			om := oe.(map[string]interface{})
			log.Printf("ANDREW NetworkInterfaceDiffOperation: diff %d OLD %s and NEW %s", ni, om, nm)
			r := NewNetworkInterfaceSubresource(c, d, nm, om, ni)
			if err := r.ValidateDiff(); err != nil {
				return fmt.Errorf("%s: %s", r.Addr(), err)
			}
		}
	}
	//for ni, ne := range n.([]interface{}) {
	//	nm := ne.(map[string]interface{})
	//	r := NewNetworkInterfaceSubresource(c, d, nm, nil, ni)
	//	if err := r.ValidateDiff(); err != nil {
	//		return fmt.Errorf("%s: %s", r.Addr(), err)
	//	}
	//}
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
	urange, err := nicUnitRange(devices)
	if err != nil {
		return nil, nil, fmt.Errorf("error calculating network device range: %s", err)
	}
	srcSet := make([]interface{}, urange)
	log.Printf("[DEBUG] NetworkInterfacePostCloneOperation: Layout from source: %d devices over a %d unit range", len(devices), urange)

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
		// Sunny this will give idx as the 7 in pci:0:7, or in the case of SRIOV it will give 45 from pci:0:45
		_, _, idx, err := splitDevAddr(r.Get("device_address").(string))
		if err != nil {
			return nil, nil, fmt.Errorf("%s: error parsing device address: %s", r, err)
		}

		if r.Get("adapter_type").(string) != networkInterfaceSubresourceTypeSriov {
			srcSet[idx-networkInterfacePciDeviceOffset] = r.Data()
			log.Printf("ANDREW postCloneOp VMXNET idx %d srcSet %d resource addr", idx, idx-networkInterfacePciDeviceOffset, r.Addr())
		} else {
			// Sunny Populate srcSet slice from the top end down for SRIOV, so if idx is 45 and urange is 4 this would give
			// At this point, sriovNetwork... should be 45 not 48 and this I think should be srcSet[urange-1+idx-sriovNetw...] to avoid off by one error.
			srcSet[urange-1+idx-sriovNetworkInterfacePciDeviceOffset] = r.Data()
			//srcSet[urange+idx-sriovNetworkInterfacePciDeviceOffset] = r.Data()
			// ANDREW this is where we probably need to work out what to do if idx doesn't make it be 45 and under (not sure what is occurring though)
			log.Printf("ANDREW postCloneOp SRIOV idx %d srcSet %d resource addr", idx, urange-1+idx-sriovNetworkInterfacePciDeviceOffset, r.Addr())
		}
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
			//sunny
			log.Printf("ANDREW postClone srcSet doesn't contain this %d index, cm %s", i, cm)
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
		// Sunny create a new network interface of index i.  However r.Update doesn't seem to do anything with PCI number
		// for i so it might be all right. ??
		r := NewNetworkInterfaceSubresource(c, d, nm, sm, i)
		if !reflect.DeepEqual(sm, nm) {
			// Update
			cspec, err := r.Update(l)
			if err != nil {
				return nil, nil, fmt.Errorf("%s: %s", r.Addr(), err)
			}
			// Sunny update the list l of Virtual Devices with the updated device.
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

		m["adapter_type"] = virtualEthernetCardString(device.(types.BaseVirtualEthernetCard))
		// TOOO: these only make sense in non SR-IOV world
		m["bandwidth_limit"] = ethernetCard.ResourceAllocation.Limit
		m["bandwidth_reservation"] = ethernetCard.ResourceAllocation.Reservation
		m["bandwidth_share_level"] = ethernetCard.ResourceAllocation.Share.Level
		m["bandwidth_share_count"] = ethernetCard.ResourceAllocation.Share.Shares
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
	if len(r.Get("physical_function").(string)) > 0 {
		device, err = r.addPhysicalFunction(device)
	}

	if r.Get("adapter_type").(string) == networkInterfaceSubresourceTypeSriov {
		log.Printf("ANDREW SRIOV so set to restart")
		r.SetRestart("<device delete>")
		log.Printf("ANDREW restart should be done")
	}

	// CreateEthernetCard does not attach stuff, however, assuming that you will
	// let vSphere take care of the attachment and what not, as there is usually
	// only one PCI device per virtual machine and their tools don't really care
	// about state. Terraform does though, so we need to not only set but also
	// track that stuff.
	if err := r.assignEthernetCard(l, device, ctlr); err != nil {
		return nil, err
	}
	// Ensure the device starts connected
	l.Connect(device)

	// Set base-level card bits now
	card := device.(types.BaseVirtualEthernetCard).GetVirtualEthernetCard()
	card.Key = l.NewKey()

	// Set the rest of the settings here.
	if r.Get("use_static_mac").(bool) {
		card.AddressType = string(types.VirtualEthernetCardMacTypeManual)
		card.MacAddress = r.Get("mac_address").(string)
	}

	log.Printf("[ANDREW] Create of Ney and card.ResourceAllocation is %s", card.ResourceAllocation)
	version := viapi.ParseVersionFromClient(r.client)
	if (version.Newer(viapi.VSphereVersion{Product: version.Product, Major: 6}) && r.Get("adapter_type") != networkInterfaceSubresourceTypeSriov) {
		log.Printf("[ANDREW] we are in this place setting card.ResourceAllocation")

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
	} else {
		log.Printf("[ANDREW] not setting card.ResourceAllocation")
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
	log.Printf("ANDREW Read: Card type %T backing type is %T", card, card.Backing)
	// Card type *types.VirtualEthernetCard backing type is *types.VirtualEthernetCardDistributedVirtualPortBackingInfo
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

	//newDevice.(types.BaseVirtualEthernetCard).GetVirtualEthernetCard()
	//card.sriovBacking undefined (type *types.VirtualEthernetCard has no field or method sriovBacking)

	//if card.sriovBacking != nil {
	//	log.Printf("ANDREW Read: Physical function ")
	//}
	// @@@ Sunny these are all attempts to read the SriovBacking by testing that the card
	// type is VirtualSriovEthernetCard and if so access the backing, but all attempts return
	// the type of the card as VirtualEthernetCard which is the parent type to VirtualSriovEthernetCard
	// So I can't access the sriov fields.

	log.Printf("ANDREW Read: Pre Adapter type test ")
	log.Printf("ANDREW: device %s", device)
	switch v := interface{}(device).(type) {
	case *types.VirtualSriovEthernetCard:
		log.Printf("ANDREW Read: Adapter type IS SRIOV ")

		sriovBacking := v.SriovBacking

		if sriovBacking.PhysicalFunctionBacking == nil {
			return fmt.Errorf("could not determine SRIOV physical_function from NIC")
		}
		r.Set("physical_function", sriovBacking.PhysicalFunctionBacking.Id)
		log.Printf("ANDREW Read: Physical function is %s", r.Get("physical_function"))
		log.Printf("ANDREW Read: Physical function is ")
	default:
		log.Printf("ANDREW Read: Adapter type non SRIOV ")
	}

	var test interface{} = card
	if bve, ok := test.(types.VirtualSriovEthernetCard); ok {
		log.Printf("ANDREW Read: bve type is SRIOV %s", bve)
	} else {
		log.Printf("ANDREW Read: bve type is NOT SRIOV %s", bve)
		//types.BaseVirtualDevice(t)
	}
	switch cardtype := test.(type) {
	case *types.VirtualSriovEthernetCard:
		if cardtype.SriovBacking != nil {
			sriovBacking := cardtype.SriovBacking

			if sriovBacking.PhysicalFunctionBacking == nil {
				return fmt.Errorf("could not determine SRIOV physical_function from NIC")
			}
			r.Set("physical_function", sriovBacking.PhysicalFunctionBacking.Id)
			log.Printf("ANDREW Read: Physical function is %s", r.Get("physical_function"))
			log.Printf("ANDREW Read: Physical function is ")
		} else {
			log.Printf("ANDREW SRIOV card has no backing")
		}
	default:
		log.Printf("ANDREW Read it isn't SRIOV %T", cardtype)
	}

	sriovType := reflect.TypeOf((*types.VirtualSriovEthernetCard)(nil))
	if sriovType == nil {
		log.Printf("ANDREW Read: sriovType is nil")
	}
	dname := sriovType.Elem().Name()
	t := reflect.TypeOf(card)
	if t == sriovType {
		log.Printf("ANDREW Read: It IS SRIOV type")
	} else if _, ok := t.Elem().FieldByName(dname); ok {
		log.Printf("ANDREW Read: It IS SRIOV name %s %s ", dname, t.Elem().Name())
	} else {
		log.Printf("ANDREW Read: ANd again it is not sriov %s %s name %s and %s ", &sriovType, &t, dname, t.Elem().Name())
	}

	//_, ok := t.Elem().FieldByName(dname)

	//if cardtype, ok := test.(types.VirtualSriovEthernetCard); ok {
	//	var test2 types.VirtualSriovEthernetCard = card
	//	log.Printf("ANDREW cardtype is sriov %s", cardtype)
	//	sriovBacking := test2.SriovBacking
	//	if sriovBacking.PhysicalFunctionBacking == nil {
	//		return fmt.Errorf("could not determine SRIOV physical_function from NIC")
	//	}
	//
	//	r.Set("physical_function", sriovBacking.PhysicalFunctionBacking.Id)
	//	log.Printf("ANDREW Read: Physical function is %s", r.Get("physical_function"))
	//	log.Printf("ANDREW Read: Physical function is ")
	//} else {
	//	log.Printf("ANDREW Read it isn't SRIOV %s", cardtype)
	//}

	r.Set("network_id", netID)

	r.Set("use_static_mac", card.AddressType == string(types.VirtualEthernetCardMacTypeManual))
	r.Set("mac_address", card.MacAddress)

	version := viapi.ParseVersionFromClient(r.client)
	if (version.Newer(viapi.VSphereVersion{Product: version.Product, Major: 6}) && r.Get("adapter_type") != networkInterfaceSubresourceTypeSriov) {
		if card.ResourceAllocation != nil {
			r.Set("bandwidth_limit", card.ResourceAllocation.Limit)
			r.Set("bandwidth_reservation", card.ResourceAllocation.Reservation)
			r.Set("bandwidth_share_count", card.ResourceAllocation.Share.Shares)
			r.Set("bandwidth_share_level", card.ResourceAllocation.Share.Level)
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

	// A change in adapter_type is essentially a ForceNew. We would normally veto
	// this, but network devices are not extremely mission critical if they go
	// away, so we can support in-place modification of them in configuration by
	// just pushing a delete of the old device and adding a new version of the
	// device, with the old device unit number preserved so that it (hopefully)
	// gets the same device position as its previous incarnation, allowing old
	// device aliases to work, etc.
	// The one change that is vetoed is changing adapter type to or from sriov,
	// because the device unit numbers for sriov are from 45 downwards, and
	// those for other networks are from 7 upwards, so it is too fiddly to support
	// in-place modification.
	if r.HasChange("adapter_type") || r.HasChange("physical_function") {
		// Ensure network interfaces aren't changing adapter_type to or from sriov
		if err := r.blockAdapterTypeChangeSriov(); err != nil {
			return nil, err
		}
		if r.HasChange("adapter_type") {
			log.Printf("[DEBUG] %s: Device type changing to %s, re-creating device", r, r.Get("adapter_type").(string))
		} else if r.HasChange("physical_function") {
			log.Printf("[DEBUG] %s: SRIOV Physical function changing to %s, re-creating device", r, r.Get("physical_function").(string))
		}
		card := device.GetVirtualEthernetCard()
		newDevice, err := l.CreateEthernetCard(r.Get("adapter_type").(string), card.Backing)
		if err != nil {
			return nil, err
		}
		if len(r.Get("physical_function").(string)) > 0 {
			newDevice, err = r.addPhysicalFunction(newDevice)
		}

		newCard := newDevice.(types.BaseVirtualEthernetCard).GetVirtualEthernetCard()
		// Copy controller attributes and unit number
		newCard.ControllerKey = card.ControllerKey
		if card.UnitNumber != nil {
			var un int32
			un = *card.UnitNumber
			newCard.UnitNumber = &un
		}
		// Ensure the device starts connected
		// Set the key
		newCard.Key = l.NewKey()
		// If VMware tools is not running, this operation requires a reboot
		if r.rdd.Get("vmware_tools_status").(string) != string(types.VirtualMachineToolsRunningStatusGuestToolsRunning) {
			r.SetRestart("adapter_type")
		}

		if r.HasChange("physical_function") {
			// If SRIOV physical function has changed, this operation requires a reboot
			r.SetRestart("physical_function")
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
	log.Printf("[ANDREW] in update	card.ResourceAllocation is %s", card.ResourceAllocation)
	version := viapi.ParseVersionFromClient(r.client)
	if (version.Newer(viapi.VSphereVersion{Product: version.Product, Major: 6}) && r.Get("adapter_type") != networkInterfaceSubresourceTypeSriov) {
		log.Printf("[ANDREW] we are setting resoruce allocation")

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
	} else {
		log.Printf("[ANDREW] not setting resoruce allocation")
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

func (r *NetworkInterfaceSubresource) addPhysicalFunction(device types.BaseVirtualDevice) (types.BaseVirtualDevice, error) {
	log.Printf("[DEBUG] We have physical function")
	var d2 interface{} = device
	// Based off https://vdc-download.vmware.com/vmwb-repository/dcr-public/b50dcbbf-051d-4204-a3e7-e1b618c1e384/538cf2ec-b34f-4bae-a332-3820ef9e7773/vim.vm.device.VirtualSriovEthernetCard.SriovBackingInfo.html
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
	log.Printf("ANDREW Deleting device %s from resource %s", device, r)
	if err != nil {
		return nil, err
	}
	// If VMware tools is not running, this operation requires a reboot
	if r.rdd.Get("vmware_tools_status").(string) != string(types.VirtualMachineToolsRunningStatusGuestToolsRunning) {
		log.Printf("ANDREW VMware tools not running so set to restart")
		r.SetRestart("<device delete>")
		log.Printf("ANDREW restart should be done")
	}
	if r.Get("adapter_type").(string) == networkInterfaceSubresourceTypeSriov {
		log.Printf("ANDREW SRIOV so set to restart")
		r.SetRestart("<device delete>")
		log.Printf("ANDREW restart should be done")
	}
	log.Printf("ANDREW Finding the bVd")
	bvd := baseVirtualEthernetCardToBaseVirtualDevice(device)
	log.Printf("ANDREW Found the bVd %s", bvd)
	spec, err := object.VirtualDeviceList{bvd}.ConfigSpec(types.VirtualDeviceConfigSpecOperationRemove)
	log.Printf("ANDREW Delete spec is %s", spec)
	if err != nil {
		return nil, err
	}
	log.Printf("[DEBUG] %s: Device config operations from update: %s", r, DeviceChangeString(spec))
	log.Printf("[DEBUG] %s: Delete completed", r)
	return spec, nil
}

// A change that is vetoed is changing adapter type to or from sriov,
// because the device unit numbers for sriov are from 45 downwards, and
// those for other networks are from 7 upwards, so it is too fiddly to support
// in-place modification.
func (r *NetworkInterfaceSubresource) blockAdapterTypeChangeSriov() error {
	if r.HasChange("adapter_type") {
		oldAdapterType, newAdapterType := r.GetChange("adapter_type")
		log.Printf("Diff adapter type old %s new %s for network_interface %s index %d", oldAdapterType, newAdapterType, r, r.Index)
		if (oldAdapterType != networkInterfaceSubresourceTypeSriov && newAdapterType == networkInterfaceSubresourceTypeSriov) ||
			(oldAdapterType == networkInterfaceSubresourceTypeSriov || newAdapterType != networkInterfaceSubresourceTypeSriov) {
			log.Printf("[DEBUG] blockAdapterTypeChangeSriov: Network interface %s index %d changing type from %s to %s. Block this", r, r.Index, oldAdapterType, newAdapterType)
			return fmt.Errorf("Changing the network_interface list such that there is a change in adapter_type to"+
				" or from sriov for a particular index of network_interface is not supported.\n"+
				"Index %d, old adapter_type %s, new adapter_type %s\n"+
				"Delete the network interfaces, apply, and then re-add them instead.", r.Index, oldAdapterType, newAdapterType)
		}
		return nil
	} else {
		log.Printf("ANDREW no change to adapter type for %s", r)
	}
	return nil
}

// ValidateDiff performs any complex validation of an individual
// network_interface sub-resource that can't be done in schema alone.
func (r *NetworkInterfaceSubresource) ValidateDiff() error {
	log.Printf("[DEBUG] %s: Beginning diff validation", r)

	// Ensure that network resource allocation options are only set on vSphere
	// 6.0 and higher.
	version := viapi.ParseVersionFromClient(r.client)
	if (version.Older(viapi.VSphereVersion{Product: version.Product, Major: 6}) && r.Get("adapter_type") != networkInterfaceSubresourceTypeSriov) {
		log.Printf("[ANDREWLATE] hitting this code")
		if err := r.restrictResourceAllocationSettings(); err != nil {
			return err
		}
	} else {
		log.Printf("[ANDREW ] not calling restrictResourceAllocationSettings")
	}

	// Ensure physical adapter is set on all (and only on) SR-IOV NICs
	if r.Get("adapter_type").(string) == networkInterfaceSubresourceTypeSriov {
		if len(r.Get("physical_function").(string)) == 0 {
			return fmt.Errorf("physical_function must be set on SR-IOV Network interface")
		}
	} else {
		if len(r.Get("physical_function").(string)) > 0 {
			return fmt.Errorf("cannot set physical_function on non SR-IOV Network interface")
		}

	}

	// Ensure network interfaces aren't changing adapter_type to or from sriov
	if err := r.blockAdapterTypeChangeSriov(); err != nil {
		return err
	}

	// TODO: As per https://docs.vmware.com/en/VMware-vSphere/7.0/com.vmware.vsphere.networking.doc/GUID-898A3D66-9415-4854-8413-B40F2CB6FF8D.html we need to check that

	// (1) The host is set for this VM (i.e.the Vm is not going to be created on some random host)
	// (2) Verify that the relevant physical NIC has SR-IOV enabled and active
	// (3) Verify that the virtual machine compatibility is ESXi 5.5 and later. (alreay done)
	// (4) Verify that Red Hat Enterprise Linux 6 and later or Windows has been selected as the guest operating system
	// (5) on the VM, Expand the Memory section, select Reserve all guest memory (All locked) and click OK
	//      that is memory_reservation  is set on the VM resource and is equal to memory_limit

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
	log.Printf("[ANDREW] Suspicious to be here with %s", r)
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

// assignEthernetCard is a subset of the logic that goes into AssignController
// right now but with an unit offset of 7. This is based on what we have
// observed on vSphere in terms of reserved PCI unit numbers (the first NIC
// automatically gets re-assigned to unit number 7 if it's not that already.
// Except SRIOV NICs which get re-assigned to unit number 45 and the next to 44 etc)
func (r *NetworkInterfaceSubresource) assignEthernetCard(l object.VirtualDeviceList, device types.BaseVirtualDevice, c types.BaseVirtualController) error {
	var newUnit int32

	if r.Get("adapter_type").(string) == networkInterfaceSubresourceTypeSriov {
		// SR-IOV NICs are assgined the next free unitNumber below 45.
		// The PCI device offset. This seems to be where vSphere starts assigning
		// virtual NICs on the PCI controller.
		sriovPciDeviceOffset := int32(sriovNetworkInterfacePciDeviceOffset)

		// The first part of this is basically the private newUnitNumber function
		// from VirtualDeviceList, with a maximum unit count of 10. This basically
		// means that no more than 10 virtual NICs can be assigned right now, which
		// hopefully should be plenty.
		units := make([]bool, maxNetworkInterfaceCount)

		sriovAvailableUnits := []int32{45, 44, 43, 42, 41, 40, 39, 38, 37, 36}

		ckey := c.GetVirtualController().Key

		for _, device := range l {
			d := device.GetVirtualDevice()
			if d.ControllerKey != ckey || d.UnitNumber == nil || *d.UnitNumber > sriovPciDeviceOffset || *d.UnitNumber <= sriovPciDeviceOffset-maxNetworkInterfaceCount {
				if d.UnitNumber != nil {
					log.Printf("ANDREW skipping device with unit number %d and controllerkey %d where our controller is %d", *d.UnitNumber, d.ControllerKey, ckey)
				}
				continue
			}
			//units[sriovPciDeviceOffset-*d.UnitNumber] = true
			for indx, value := range sriovAvailableUnits {
				if value == *d.UnitNumber {
					log.Printf("ANDREW removing Unit number %d from available SRIOV units as it is in use", value)
					sriovAvailableUnits = append(sriovAvailableUnits[:indx], sriovAvailableUnits[indx+1:]...)
					log.Printf("ANDREW available units are now %s", sriovAvailableUnits)
					break
				}
			}
		}

		// Now that we know which units are used, we can pick one
		if len(sriovAvailableUnits) == 0 {
			return fmt.Errorf("All ten SRIOV device units are currently in use on the PCI bus. Cannot assign SRIOV network.")
		}
		newUnit = sriovAvailableUnits[0]

		//newUnit = sriovPciDeviceOffset - int32(r.Index)
		//sunny this is where it is going wrong.  r.Index is presumably the order that the networks appear in main.tf
		// and as Andrew said, it only works if you have 3 VMXNET and then some SRIOV, because we need r here to be 45
		// and downwards, and if there are only 2 VMXNET then r.Index will be 2 and the unit number will be 46 which
		// is not allowed.  Not sure how we remedy that as need to store the number of networks added in the call to this
		// function but it might get muddled up with the postCloneOperation bits too, so not sure what to do
		log.Printf("ANDREW  SRIOV newUnit %d ignoring r Index is %d", newUnit, int32(r.Index))
		if units[sriovPciDeviceOffset-newUnit] {
			return fmt.Errorf("device unit at %d is currently in use on the PCI bus", newUnit)
		}
		//units := make([]bool, 10)
		//
		//ckey := c.GetVirtualController().Key
		//
		//for _, device := range l {
		//	d := device.GetVirtualDevice()
		//	if d.ControllerKey != ckey || d.UnitNumber == nil || *d.UnitNumber > sriovPciDeviceOffset || *d.UnitNumber <= sriovPciDeviceOffset-10 {
		//		if d.UnitNumber != nil {
		//			log.Printf("ANDREW skiping device with unit number %d and controllerkey %d where our controller is %d", *d.UnitNumber, d.ControllerKey, ckey)
		//		}
		//		continue
		//	}
		//	log.Printf("ANDREW setting Unit number %d fto be in use", sriovPciDeviceOffset-*d.UnitNumber)
		//	units[sriovPciDeviceOffset-*d.UnitNumber] = true
		//}
		//
		//// Now that we know which units are used, we can pick one
		//newUnit = sriovPciDeviceOffset - int32(r.Index)
		////sunny this is where it is going wrong.  r.Index is presumably the order that the networks appear in main.tf
		//// and as Andrew said, it only works if you have 3 VMXNET and then some SRIOV, because we need r here to be 45
		//// and downwards, and if there are only 2 VMXNET then r.Index will be 2 and the unit number will be 46 which
		//// is not allowed.  Not sure how we remedy that as need to store the number of networks added in the call to this
		//// function but it might get muddled up with the postCloneOperation bits too, so not sure what to do
		//log.Printf("ANDREW  SRIOV newUnit %d because r Index is %d", newUnit, int32(r.Index))
		//if units[sriovPciDeviceOffset-newUnit] {
		//	return fmt.Errorf("device unit at %d is currently in use on the PCI bus", newUnit)
		//}
	} else {

		// Non-SRIOV NIC are assigned the next free unitNumber from 7

		// The PCI device offset. This seems to be where vSphere starts assigning
		// virtual NICs on the PCI controller.
		pciDeviceOffset := int32(networkInterfacePciDeviceOffset)

		// The first part of this is basically the private newUnitNumber function
		// from VirtualDeviceList, with a maximum unit count of 10. This basically
		// means that no more than 10 virtual NICs can be assigned right now, which
		// hopefully should be plenty.
		units := make([]bool, maxNetworkInterfaceCount)

		ckey := c.GetVirtualController().Key

		for _, device := range l {
			d := device.GetVirtualDevice()
			if d.ControllerKey != ckey || d.UnitNumber == nil || *d.UnitNumber < pciDeviceOffset || *d.UnitNumber >= pciDeviceOffset+maxNetworkInterfaceCount {
				continue
			}
			units[*d.UnitNumber-pciDeviceOffset] = true
		}

		// Now that we know which units are used, we can pick one
		newUnit = int32(r.Index) + pciDeviceOffset
		if units[newUnit-pciDeviceOffset] {
			return fmt.Errorf("device unit at %d is currently in use on the PCI bus", newUnit)
		}
		log.Printf("ANDREW  VMXNET newUnit %d because r Index is %d", newUnit, int32(r.Index))
	}

	d := device.GetVirtualDevice()
	d.ControllerKey = c.GetVirtualController().Key
	log.Printf("ANDREW assignEthernetCard set unit number of device %s to %d", device, newUnit)
	// It seems that setting this UnitNumber has no effect on actually which UnitNumber the network interface gets,
	// that is down to the vSphere vagaries of non-SRIOV being 7+ in order of addition,  and SRIOV being 45-, so this
	// must just be for our tracking purposes.
	d.UnitNumber = &newUnit

	if d.Key == 0 {
		d.Key = -1
	}
	return nil
}

// nicUnitRange calculates a range of units given a certain VirtualDeviceList,
// which should be network interfaces.  It's used in network interface refresh
// logic to determine how many subresources may end up in state.
// Sunny - changed from > 7
// It returns the count of all virtual devices with a unit number >= 7
func nicUnitRange(l object.VirtualDeviceList) (int, error) {
	// No NICs means no range
	if len(l) < 1 {
		return 0, nil
	}
	nonSriov, err := nonSriovNicUnitRange(l)
	if err != nil {
		return 0, fmt.Errorf("error calculating network device range: %s", err)
	}
	sriov, err2 := sriovNicUnitRange(l)
	if err2 != nil {
		return 0, fmt.Errorf("error calculating network device range: %s", err2)
	}
	return (nonSriov + sriov), nil
}

// nicUnitRange calculates a range of units given a certain VirtualDeviceList,
// which should be network interfaces.  It's used in network interface refresh
// logic to determine how many subresources may end up in state.
// Sunny - changed from > 7
// It returns the count of all virtual devices with a unit number >= 7
func nonSriovNicUnitRange(l object.VirtualDeviceList) (int, error) {
	// No NICs means no range
	if len(l) < 1 {
		return 0, nil
	}
	offset := int32(networkInterfacePciDeviceOffset)
	//Sunny
	var unitNumbers []int32
	for _, v := range l {
		d := v.GetVirtualDevice()
		log.Printf("ANDREW nicUnitRange device is %s", d)
		if d.UnitNumber == nil {
			return 0, fmt.Errorf("device at key %d has no unit number", d.Key)
		}
		log.Printf("ANDREW nicUnitRange unit number found %d\n\n", *d.UnitNumber)
		// sunny - changed from >=
		if *d.UnitNumber >= offset && *d.UnitNumber < offset+maxNetworkInterfaceCount {
			unitNumbers = append(unitNumbers, *d.UnitNumber)
		}
	}
	log.Printf("ANDREW count of %d non-SRIOV units", len(unitNumbers))
	return len(unitNumbers), nil
}

// sriovNicUnitRange calculates a range of units given a certain VirtualDeviceList,
// which should be network interfaces.  It's used in network interface refresh
// logic to determine how many subresources may end up in state.
// It returns the count of all virtual devices with a unit number between 36 and 45
// This is the range of Unit numbers for SRIOV Nics
func sriovNicUnitRange(l object.VirtualDeviceList) (int, error) {
	// No NICs means no range
	if len(l) < 1 {
		return 0, nil
	}
	offset := int32(sriovNetworkInterfacePciDeviceOffset) - maxNetworkInterfaceCount

	//Sunny
	var unitNumbers []int32
	for _, v := range l {
		d := v.GetVirtualDevice()
		log.Printf("ANDREW nicUnitRange device is %s", d)
		if d.UnitNumber == nil {
			return 0, fmt.Errorf("device at key %d has no unit number", d.Key)
		}
		log.Printf("ANDREW nicUnitRange unit number found %d\n\n", *d.UnitNumber)
		// sunny - changed from >=
		if *d.UnitNumber >= offset {
			unitNumbers = append(unitNumbers, *d.UnitNumber)
		}
	}
	log.Printf("ANDREW count of %d SRIOV units", len(unitNumbers))
	return len(unitNumbers), nil
}
