// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package vsphere

import (
	"fmt"
	"sync"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	"github.com/hashicorp/terraform-provider-vsphere/vsphere/internal/helper/viapi"
	"github.com/vmware/govmomi/vim25/types"
)

var vsphereDistributedVirtualSwitchModificationMutex sync.Mutex

func resourceVSphereDistributedVirtualSwitchPvlanMapping() *schema.Resource {
	s := map[string]*schema.Schema{
		"distributed_virtual_switch_id": {
			Type:        schema.TypeString,
			Description: "The ID of the distributed virtual switch to attach this mapping to.",
			Required:    true,
			ForceNew:    true,
		},
		"primary_vlan_id": {
			Type:         schema.TypeInt,
			Required:     true,
			Description:  "The primary VLAN ID. The VLAN IDs of 0 and 4095 are reserved and cannot be used in this property.",
			ValidateFunc: validation.IntBetween(1, 4094),
			ForceNew:     true,
		},
		"secondary_vlan_id": {
			Type:         schema.TypeInt,
			Required:     true,
			Description:  "The secondary VLAN ID. The VLAN IDs of 0 and 4095 are reserved and cannot be used in this property.",
			ValidateFunc: validation.IntBetween(1, 4094),
			ForceNew:     true,
		},
		"pvlan_type": {
			Type:         schema.TypeString,
			Required:     true,
			Description:  "The private VLAN type. Valid values are promiscuous, community and isolated.",
			ValidateFunc: validation.StringInSlice(privateVLANTypeAllowedValues, false),
			ForceNew:     true,
		},
	}

	return &schema.Resource{
		Create: resourceVSphereDistributedVirtualSwitchPvlanMappingCreate,
		Read:   resourceVSphereDistributedVirtualSwitchPvlanMappingRead,
		Delete: resourceVSphereDistributedVirtualSwitchPvlanMappingDelete,
		Schema: s,
	}
}

func resourceVSphereDistributedVirtualSwitchPvlanMappingOperation(operation string, d *schema.ResourceData, meta interface{}) error {
	client := meta.(*Client).vimClient
	if err := viapi.ValidateVirtualCenter(client); err != nil {
		return err
	}

	// vSphere only allows one modification operation at a time, so lock locally
	//   to avoid errors if multiple pvlan mappings are created at once.
	vsphereDistributedVirtualSwitchModificationMutex.Lock()

	// Fetch the target dvswitch
	dvs, err := dvsFromUUID(client, d.Get("distributed_virtual_switch_id").(string))
	if err != nil {
		return fmt.Errorf("cannot locate distributed_virtual_switch: %s", err)
	}

	// Perform add operation
	entry := types.VMwareDVSPvlanMapEntry{
		PrimaryVlanId:   int32(d.Get("primary_vlan_id").(int)),
		SecondaryVlanId: int32(d.Get("secondary_vlan_id").(int)),
		PvlanType:       d.Get("pvlan_type").(string),
	}

	pvlanConfig := []types.VMwareDVSPvlanConfigSpec{
		{
			Operation:  operation,
			PvlanEntry: entry,
		},
	}

	err = updateDVSPvlanMappings(dvs, pvlanConfig)
	if err != nil {
		return fmt.Errorf("cannot reconfigure distributed virtual switch: %s", err)
	}

	vsphereDistributedVirtualSwitchModificationMutex.Unlock()
	return nil
}

func resourceVSphereDistributedVirtualSwitchPvlanMappingCreate(d *schema.ResourceData, meta interface{}) error {
	err := resourceVSphereDistributedVirtualSwitchPvlanMappingOperation("add", d, meta)
	if err != nil {
		return err
	}

	// Try to read the mapping back, which will also generate the ID.
	return resourceVSphereDistributedVirtualSwitchPvlanMappingRead(d, meta)
}

func resourceVSphereDistributedVirtualSwitchPvlanMappingDelete(d *schema.ResourceData, meta interface{}) error {
	return resourceVSphereDistributedVirtualSwitchPvlanMappingOperation("remove", d, meta)
}

func resourceVSphereDistributedVirtualSwitchPvlanMappingRead(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*Client).vimClient
	if err := viapi.ValidateVirtualCenter(client); err != nil {
		return err
	}

	// Ensure the DVSwitch still exists
	dvs, err := dvsFromUUID(client, d.Get("distributed_virtual_switch_id").(string))
	if err != nil {
		return fmt.Errorf("cannot locate distributed_virtual_switch: %s", err)
	}

	// Get its properties
	props, err := dvsProperties(dvs)
	if err != nil {
		return fmt.Errorf("cannot read properties of distributed_virtual_switch: %s", err)
	}
	d.Set("distributed_virtual_switch_id", props.Uuid)

	for _, mapping := range props.Config.(*types.VMwareDVSConfigInfo).PvlanConfig {
		if mapping.PrimaryVlanId == int32(d.Get("primary_vlan_id").(int)) && mapping.SecondaryVlanId == int32(d.Get("secondary_vlan_id").(int)) && mapping.PvlanType == d.Get("pvlan_type").(string) {
			d.SetId(fmt.Sprintf("dvswitch-%s-mapping-%d-%d-%s", props.Config.(*types.VMwareDVSConfigInfo).Uuid, mapping.PrimaryVlanId, mapping.SecondaryVlanId, mapping.PvlanType))
			return nil
		}
	}

	d.SetId("")
	return nil
}
