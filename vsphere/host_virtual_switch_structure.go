package vsphere

import (
	"github.com/hashicorp/terraform/helper/schema"
	"github.com/vmware/govmomi/vim25/types"
)

// schemaHostVirtualSwitchBondBridge returns a schema sub-set of attributes
// that together represent a flattened layout of the
// HostVirtualSwitchBondBridge data object.
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
			Description: "Whether to advertise or listen for link discovery. Valid values are \"advertise\", \"both\", \"listen\", and \"none\".",
			Default:     "listen",
		},
		"link_discovery_protocol": &schema.Schema{
			Type:        schema.TypeString,
			Optional:    true,
			Description: "The discovery protocol type. Valid values are \"cdp\" and \"lldp\".",
			Default:     "cdp",
		},

		// HostVirtualSwitchBondBridge
		"network_adapters": &schema.Schema{
			Type:        schema.TypeList,
			Required:    true,
			Description: "The link discovery protocol configuration for the virtual switch.",
			Elem:        &schema.Schema{Type: schema.TypeString},
			DefaultFunc: func() (interface{}, error) { return []interface{}{}, nil },
		},
	}
}

func expandHostVirtualSwitchBeaconConfig(d *schema.ResourceData) *types.HostVirtualSwitchBeaconConfig {
	obj := &types.HostVirtualSwitchBeaconConfig{
		Interval: int32(d.Get("beacon_interval").(int)),
	}
	return obj
}

func flattenHostVirtualSwitchBeaconConfig(d *schema.ResourceData, obj *types.HostVirtualSwitchBeaconConfig) error {
	d.Set("beacon_interval", obj.Interval)
	return nil
}

func expandLinkDiscoveryProtocolConfig(d *schema.ResourceData) *types.LinkDiscoveryProtocolConfig {
	obj := &types.LinkDiscoveryProtocolConfig{
		Operation: d.Get("link_discovery_operation").(string),
		Protocol:  d.Get("link_discovery_protocol").(string),
	}
	return obj
}

func flattenLinkDiscoveryProtocolConfig(d *schema.ResourceData, obj *types.LinkDiscoveryProtocolConfig) error {
	d.Set("link_discovery_operation", obj.Operation)
	d.Set("link_discovery_protocol", obj.Protocol)
	return nil
}

func expandHostVirtualSwitchBondBridge(d *schema.ResourceData) *types.HostVirtualSwitchBondBridge {
	obj := &types.HostVirtualSwitchBondBridge{
		NicDevice: sliceInterfacesToStrings(d.Get("network_adapters").([]interface{})),
	}
	obj.Beacon = expandHostVirtualSwitchBeaconConfig(d)
	obj.LinkDiscoveryProtocolConfig = expandLinkDiscoveryProtocolConfig(d)
	return obj
}

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

func schemaHostNetworkPolicy() map[string]*schema.Schema {
	return map[string]*schema.Schema{
		// HostNicTeamingPolicy/HostNicFailureCriteria
		"check_beacon": &schema.Schema{
			Type:        schema.TypeBool,
			Optional:    true,
			Description: "Enable beacon probing. Requires that the vSwitch has been configured to use a beacon. If disabled, link status is used only.",
		},

		// HostNicTeamingPolicy/HostNicOrderPolicy
		"active_nics": &schema.Schema{
			Type:        schema.TypeList,
			Description: "List of active network adapters used for load balancing.",
			Elem:        &schema.Schema{Type: schema.TypeString},
		},
		"standby_nics": &schema.Schema{
			Type:        schema.TypeList,
			Description: "List of standby network adapters used for failover.",
			Elem:        &schema.Schema{Type: schema.TypeString},
		},

		// HostNicTeamingPolicy
		"teaming_policy": &schema.Schema{
			Type:        schema.TypeString,
			Optional:    true,
			Description: "The network adapter teaming policy. Can be one of loadbalance_ip, loadbalance_srcmac, loadbalance_srcid, or failover_explicit.",
		},
		"notify_switches": &schema.Schema{
			Type:        schema.TypeBool,
			Optional:    true,
			Description: "If true, the teaming policy will notify the broadcast network of a NIC failover, triggering cache updates.",
		},
		"failback": &schema.Schema{
			Type:        schema.TypeBool,
			Optional:    true,
			Description: "If true, the teaming policy will re-activate failed interfaces higher in precedence when they come back up.",
		},

		// HostNetworkSecurityPolicy
		"allow_promiscuous": &schema.Schema{
			Type:        schema.TypeBool,
			Optional:    true,
			Description: "Enable promiscuious mode on the network. This flag indicates whether or not all traffic is seen on a given port.",
		},
		"forged_transmits": &schema.Schema{
			Type:        schema.TypeBool,
			Optional:    true,
			Description: "Controls whether or not the virtual network adapter is allowed to send network traffic with a different MAC address than that of its own.",
		},
		"mac_changes": &schema.Schema{
			Type:        schema.TypeBool,
			Optional:    true,
			Description: "Controls whether or not the Media Access Control (MAC) address can be changed.",
		},

		// HostNetworkTrafficShapingPolicy
		"shaping_average_bandwidth": &schema.Schema{
			Type:        schema.TypeInt,
			Optional:    true,
			Description: "The average bandwidth in bits per second if shaping is enabled on the port.",
			Default:     0,
		},
		"shaping_burst_size": &schema.Schema{
			Type:        schema.TypeInt,
			Optional:    true,
			Description: "The maximum burst size allowed in bytes if shaping is enabled on the port.",
			Default:     0,
		},
		"shaping_enabled": &schema.Schema{
			Type:        schema.TypeBool,
			Optional:    true,
			Description: "True if the traffic shaper is enabled on the port.",
		},
		"shaping_peak_bandwidth": &schema.Schema{
			Type:        schema.TypeInt,
			Optional:    true,
			Description: "The peak bandwidth during bursts in bits per second if traffic shaping is enabled on the port.",
			Default:     0,
		},
	}
}

func expandHostNicFailureCriteria(d *schema.ResourceData) *types.HostNicFailureCriteria {
	obj := &types.HostNicFailureCriteria{}

	if v, ok := d.GetOkExists("check_beacon"); ok {
		obj.CheckBeacon = &([]bool{v.(bool)}[0])
	}

	// These fields are deprecated and are set only to make things work. They are
	// not exposed to Terraform.
	obj.CheckSpeed = "minimum"
	obj.Speed = 10
	obj.CheckDuplex = &([]bool{false}[0])
	obj.FullDuplex = &([]bool{false}[0])
	obj.CheckErrorPercent = &([]bool{false}[0])
	obj.Percentage = 0

	return obj
}

func flattenHostNicFailureCriteria(d *schema.ResourceData, obj *types.HostNicFailureCriteria) error {
	if obj.CheckBeacon != nil {
		d.Set("check_beacon", obj.CheckBeacon)
	}
	return nil
}

func expandHostNicOrderPolicy(d *schema.ResourceData) *types.HostNicOrderPolicy {
	obj := &types.HostNicOrderPolicy{}
	activeNics, activeOk := d.GetOkExists("active_nics")
	standbyNics, standbyOk := d.GetOkExists("standby_nics")
	if !activeOk && !standbyOk {
		return nil
	}
	obj.ActiveNic = sliceInterfacesToStrings(activeNics.([]interface{}))
	obj.StandbyNic = sliceInterfacesToStrings(standbyNics.([]interface{}))
	return obj
}

func flattenHostNicOrderPolicy(d *schema.ResourceData, obj *types.HostNicOrderPolicy) error {
	if obj == nil {
		return nil
	}
	if err := d.Set("active_nics", sliceStringsToInterfaces(obj.ActiveNic)); err != nil {
		return err
	}
	if err := d.Set("standby_nics", sliceStringsToInterfaces(obj.StandbyNic)); err != nil {
		return err
	}
	return nil
}

func expandHostNicTeamingPolicy(d *schema.ResourceData) *types.HostNicTeamingPolicy {
	obj := &types.HostNicTeamingPolicy{
		Policy: d.Get("teaming_policy").(string),
	}
	if v, ok := d.GetOkExists("failback"); ok {
		obj.RollingOrder = &([]bool{!v.(bool)}[0])
	}
	if v, ok := d.GetOkExists("notify_switches"); ok {
		obj.NotifySwitches = &([]bool{v.(bool)}[0])
	}
	obj.FailureCriteria = expandHostNicFailureCriteria(d)
	obj.NicOrder = expandHostNicOrderPolicy(d)

	// These fields are deprecated and are set only to make things work. They are
	// not exposed to Terraform.
	obj.ReversePolicy = &([]bool{true}[0])

	return obj
}

func flattenHostNicTeamingPolicy(d *schema.ResourceData, obj *types.HostNicTeamingPolicy) error {
	if obj.RollingOrder != nil {
		d.Set("failback", !*obj.RollingOrder)
	}
	if obj.NotifySwitches != nil {
		d.Set("notify_switches", *obj.NotifySwitches)
	}
	d.Set("teaming_policy", obj.Policy)
	if err := flattenHostNicFailureCriteria(d, obj.FailureCriteria); err != nil {
		return err
	}
	if err := flattenHostNicOrderPolicy(d, obj.NicOrder); err != nil {
		return err
	}
	return nil
}

func expandHostNetworkSecurityPolicy(d *schema.ResourceData) *types.HostNetworkSecurityPolicy {
	obj := &types.HostNetworkSecurityPolicy{}
	if v, ok := d.GetOkExists("allow_promiscuous"); ok {
		obj.AllowPromiscuous = &([]bool{v.(bool)}[0])
	}
	if v, ok := d.GetOkExists("forged_transmits"); ok {
		obj.ForgedTransmits = &([]bool{v.(bool)}[0])
	}
	if v, ok := d.GetOkExists("mac_changes"); ok {
		obj.MacChanges = &([]bool{v.(bool)}[0])
	}
	return obj
}

func flattenHostNetworkSecurityPolicy(d *schema.ResourceData, obj *types.HostNetworkSecurityPolicy) error {
	if obj.AllowPromiscuous != nil {
		d.Set("allow_promiscuous", *obj.AllowPromiscuous)
	}
	if obj.ForgedTransmits != nil {
		d.Set("forged_transmits", *obj.ForgedTransmits)
	}
	if obj.MacChanges != nil {
		d.Set("mac_changes", *obj.MacChanges)
	}
	return nil
}

func expandHostNetworkTrafficShapingPolicy(d *schema.ResourceData) *types.HostNetworkTrafficShapingPolicy {
	obj := &types.HostNetworkTrafficShapingPolicy{
		AverageBandwidth: int64(d.Get("shaping_average_bandwidth").(int)),
		BurstSize:        int64(d.Get("shaping_burst_size").(int)),
		PeakBandwidth:    int64(d.Get("shaping_peak_bandwidth").(int)),
	}
	if v, ok := d.GetOkExists("shaping_enabled"); ok {
		obj.Enabled = &([]bool{v.(bool)}[0])
	}
	return obj
}

func flattenHostNetworkTrafficShapingPolicy(d *schema.ResourceData, obj *types.HostNetworkTrafficShapingPolicy) error {
	if obj.Enabled != nil {
		d.Set("shaping_enabled", *obj.Enabled)
	}
	d.Set("shaping_average_bandwidth", obj.AverageBandwidth)
	d.Set("shaping_burst_size", obj.BurstSize)
	d.Set("shaping_peak_bandwidth", obj.PeakBandwidth)
	return nil
}

func expandHostNetworkPolicy(d *schema.ResourceData) *types.HostNetworkPolicy {
	obj := &types.HostNetworkPolicy{
		Security:      expandHostNetworkSecurityPolicy(d),
		NicTeaming:    expandHostNicTeamingPolicy(d),
		ShapingPolicy: expandHostNetworkTrafficShapingPolicy(d),
	}
	return obj
}

func flattenHostNetworkPolicy(d *schema.ResourceData, obj *types.HostNetworkPolicy) error {
	if err := flattenHostNetworkSecurityPolicy(d, obj.Security); err != nil {
		return err
	}
	if err := flattenHostNicTeamingPolicy(d, obj.NicTeaming); err != nil {
		return err
	}
	if err := flattenHostNetworkTrafficShapingPolicy(d, obj.ShapingPolicy); err != nil {
		return err
	}
	return nil
}

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

func expandHostVirtualSwitchSpec(d *schema.ResourceData) *types.HostVirtualSwitchSpec {
	obj := &types.HostVirtualSwitchSpec{
		Mtu:      int32(d.Get("mtu").(int)),
		NumPorts: int32(d.Get("number_of_ports").(int)),
		Bridge:   expandHostVirtualSwitchBondBridge(d),
		Policy:   expandHostNetworkPolicy(d),
	}
	return obj
}

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

func schemaHostPortGroupSpec() map[string]*schema.Schema {
	s := map[string]*schema.Schema{
		// HostPortGroupSpec
		"name": &schema.Schema{
			Type:        schema.TypeInt,
			Required:    true,
			Description: "The name of the port group.",
			ForceNew:    true,
		},
		"vlan_id": &schema.Schema{
			Type:        schema.TypeInt,
			Optional:    true,
			Description: "The VLAN ID/trunk mode for this port group. An ID of 0 denotes no tagging, an ID of 1-4094 tags with the specific ID, and an ID of 4095 enables trunk mode, allowing the guest to manage its own tagging.",
			Default:     0,
		},
		"virtual_switch_name": &schema.Schema{
			Type:        schema.TypeString,
			Required:    true,
			Description: "The name of the virtual switch to bind this port group to.",
			ForceNew:    true,
		},
	}
	mergeSchema(s, schemaHostNetworkPolicy())
	return s
}

func expandHostPortGroupSpec(d *schema.ResourceData) *types.HostPortGroupSpec {
	obj := &types.HostPortGroupSpec{
		Name:        d.Get("name").(string),
		VlanId:      int32(d.Get("vlan_id").(int)),
		VswitchName: d.Get("virtual_switch_name").(string),
		Policy:      *expandHostNetworkPolicy(d),
	}
	return obj
}

func flattenHostPortGroupSpec(d *schema.ResourceData, obj *types.HostPortGroupSpec) error {
	d.Set("vlan_id", obj.VlanId)
	if err := flattenHostNetworkPolicy(d, &obj.Policy); err != nil {
		return err
	}
	return nil
}
