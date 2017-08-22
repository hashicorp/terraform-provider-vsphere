package vsphere

import (
	"github.com/hashicorp/terraform/helper/schema"
	"github.com/vmware/govmomi/vim25/types"
)

// schemaHostVirtualSwitchBondBridge returns schema items for resources that
// need to work with a HostVirtualSwitchBondBridge, such as virtual switches.
func schemaHostVirtualSwitchBondBridge() map[string]*schema.Schema {
	return map[string]*schema.Schema{
		// HostVirtualSwitchBeaconConfig
		"beacon_interval": &schema.Schema{
			Type:        schema.TypeInt,
			Optional:    true,
			Description: "Determines how often, in seconds, a beacon should be sent to probe for the validity of a link.",
			Default:     1,
		},

		// LinkDiscoveryProtocolConfig
		"link_discovery_operation": &schema.Schema{
			Type:        schema.TypeString,
			Optional:    true,
			Description: "Whether to advertise or listen for link discovery. Valid values are advertise, both, listen, and none.",
			Default:     "listen",
		},
		"link_discovery_protocol": &schema.Schema{
			Type:        schema.TypeString,
			Optional:    true,
			Description: "The discovery protocol type. Valid values are cdp and lldp.",
			Default:     "cdp",
		},

		// HostVirtualSwitchBondBridge
		"network_adapters": &schema.Schema{
			Type:        schema.TypeList,
			Required:    true,
			Description: "The link discovery protocol configuration for the virtual switch.",
			Elem:        &schema.Schema{Type: schema.TypeString},
		},
	}
}

// expandHostVirtualSwitchBeaconConfig reads certain ResourceData keys and
// returns a HostVirtualSwitchBeaconConfig.
func expandHostVirtualSwitchBeaconConfig(d *schema.ResourceData) *types.HostVirtualSwitchBeaconConfig {
	obj := &types.HostVirtualSwitchBeaconConfig{
		Interval: int32(d.Get("beacon_interval").(int)),
	}
	return obj
}

// flattenHostVirtualSwitchBeaconConfig reads various fields from a
// HostVirtualSwitchBeaconConfig into the passed in ResourceData.
func flattenHostVirtualSwitchBeaconConfig(d *schema.ResourceData, obj *types.HostVirtualSwitchBeaconConfig) error {
	d.Set("beacon_interval", obj.Interval)
	return nil
}

// expandLinkDiscoveryProtocolConfig reads certain ResourceData keys and
// returns a LinkDiscoveryProtocolConfig.
func expandLinkDiscoveryProtocolConfig(d *schema.ResourceData) *types.LinkDiscoveryProtocolConfig {
	obj := &types.LinkDiscoveryProtocolConfig{
		Operation: d.Get("link_discovery_operation").(string),
		Protocol:  d.Get("link_discovery_protocol").(string),
	}
	return obj
}

// flattenLinkDiscoveryProtocolConfig reads various fields from a
// LinkDiscoveryProtocolConfig into the passed in ResourceData.
func flattenLinkDiscoveryProtocolConfig(d *schema.ResourceData, obj *types.LinkDiscoveryProtocolConfig) error {
	d.Set("link_discovery_operation", obj.Operation)
	d.Set("link_discovery_protocol", obj.Protocol)
	return nil
}

// expandHostVirtualSwitchBondBridge reads certain ResourceData keys and
// returns a HostVirtualSwitchBondBridge.
func expandHostVirtualSwitchBondBridge(d *schema.ResourceData) *types.HostVirtualSwitchBondBridge {
	obj := &types.HostVirtualSwitchBondBridge{
		NicDevice: sliceInterfacesToStrings(d.Get("network_adapters").([]interface{})),
	}
	obj.Beacon = expandHostVirtualSwitchBeaconConfig(d)
	obj.LinkDiscoveryProtocolConfig = expandLinkDiscoveryProtocolConfig(d)
	return obj
}

// flattenHostVirtualSwitchBondBridge reads various fields from a
// HostVirtualSwitchBondBridge into the passed in ResourceData.
func flattenHostVirtualSwitchBondBridge(d *schema.ResourceData, obj *types.HostVirtualSwitchBondBridge) error {
	if err := d.Set("network_adapters", sliceStringsToInterfaces(obj.NicDevice)); err != nil {
		return err
	}
	if err := flattenHostVirtualSwitchBeaconConfig(d, obj.Beacon); err != nil {
		return err
	}
	if err := flattenLinkDiscoveryProtocolConfig(d, obj.LinkDiscoveryProtocolConfig); err != nil {
		return err
	}
	return nil
}

// schemaHostVirtualSwitchSpec returns schema items for resources that need to
// work with a HostVirtualSwitchSpec, such as virtual switches.
func schemaHostVirtualSwitchSpec() map[string]*schema.Schema {
	s := map[string]*schema.Schema{
		// HostVirtualSwitchSpec
		"mtu": &schema.Schema{
			Type:        schema.TypeInt,
			Optional:    true,
			Description: "The maximum transmission unit (MTU) of the virtual switch in bytes.",
			Default:     1500,
		},
		"number_of_ports": &schema.Schema{
			Type:        schema.TypeInt,
			Optional:    true,
			Description: "The number of ports that this virtual switch is configured to use.",
			Default:     128,
		},
	}
	mergeSchema(s, schemaHostVirtualSwitchBondBridge())
	mergeSchema(s, schemaHostNetworkPolicy())
	return s
}

// expandHostVirtualSwitchSpec reads certain ResourceData keys and returns a
// HostVirtualSwitchSpec.
func expandHostVirtualSwitchSpec(d *schema.ResourceData) *types.HostVirtualSwitchSpec {
	obj := &types.HostVirtualSwitchSpec{
		Mtu:      int32(d.Get("mtu").(int)),
		NumPorts: int32(d.Get("number_of_ports").(int)),
		Bridge:   expandHostVirtualSwitchBondBridge(d),
		Policy:   expandHostNetworkPolicy(d),
	}
	return obj
}

// flattenHostVirtualSwitchSpec reads various fields from a
// HostVirtualSwitchSpec into the passed in ResourceData.
func flattenHostVirtualSwitchSpec(d *schema.ResourceData, obj *types.HostVirtualSwitchSpec) error {
	d.Set("mtu", obj.Mtu)
	d.Set("number_of_ports", obj.NumPorts)
	if err := flattenHostVirtualSwitchBondBridge(d, obj.Bridge.(*types.HostVirtualSwitchBondBridge)); err != nil {
		return err
	}
	if err := flattenHostNetworkPolicy(d, obj.Policy); err != nil {
		return err
	}
	return nil
}
