package vsphere

import (
	"log"

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
			Default:     false,
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
			Default:     "loadbalance_srcid",
		},
		"notify_switches": &schema.Schema{
			Type:        schema.TypeBool,
			Optional:    true,
			Description: "If true, the teaming policy will notify the broadcast network of a NIC failover, triggering cache updates.",
			Default:     true,
		},
		"failback": &schema.Schema{
			Type:        schema.TypeBool,
			Optional:    true,
			Description: "If true, the teaming policy will re-activate failed interfaces higher in precedence when they come back up.",
			Default:     true,
		},

		// HostNetworkSecurityPolicy
		"allow_promiscuous": &schema.Schema{
			Type:        schema.TypeBool,
			Optional:    true,
			Description: "Enable promiscuious mode on the network. This flag indicates whether or not all traffic is seen on a given port.",
			Default:     false,
		},
		"forged_transmits": &schema.Schema{
			Type:        schema.TypeBool,
			Optional:    true,
			Description: "Controls whether or not the virtual network adapter is allowed to send network traffic with a different MAC address than that of its own.",
			Default:     true,
		},
		"mac_changes": &schema.Schema{
			Type:        schema.TypeBool,
			Optional:    true,
			Description: "Controls whether or not the Media Access Control (MAC) address can be changed.",
			Default:     true,
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
			Default:     false,
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
	checkBeacon := d.Get("check_beacon").(bool)
	obj := &types.HostNicFailureCriteria{
		CheckBeacon: &checkBeacon,
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
	d.Set("check_beacon", obj.CheckBeacon)
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
	log.Printf("[DEBUG] HostNicOrderPolicy: %#v", obj)
	if err := d.Set("active_nics", sliceStringsToInterfaces(obj.ActiveNic)); err != nil {
		return err
	}
	if err := d.Set("standby_nics", sliceStringsToInterfaces(obj.StandbyNic)); err != nil {
		return err
	}
	return nil
}

func expandHostNicTeamingPolicy(d *schema.ResourceData) *types.HostNicTeamingPolicy {
	rollingOrder := !d.Get("failback").(bool)
	notifySwitches := d.Get("notify_switches").(bool)
	obj := &types.HostNicTeamingPolicy{
		Policy:         d.Get("teaming_policy").(string),
		RollingOrder:   &rollingOrder,
		NotifySwitches: &notifySwitches,
	}
	obj.FailureCriteria = expandHostNicFailureCriteria(d)
	obj.NicOrder = expandHostNicOrderPolicy(d)

	// These fields are deprecated and are set only to make things work. They are
	// not exposed to Terraform.
	obj.ReversePolicy = &([]bool{true}[0])

	return obj
}

func flattenHostNicTeamingPolicy(d *schema.ResourceData, obj *types.HostNicTeamingPolicy) error {
	d.Set("failback", !*obj.RollingOrder)
	d.Set("notify_switches", *obj.NotifySwitches)
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
	allowPromiscuous := d.Get("allow_promiscuous").(bool)
	forgedTransmits := d.Get("forged_transmits").(bool)
	macChanges := d.Get("mac_changes").(bool)
	obj := &types.HostNetworkSecurityPolicy{
		AllowPromiscuous: &allowPromiscuous,
		ForgedTransmits:  &forgedTransmits,
		MacChanges:       &macChanges,
	}
	return obj
}

func flattenHostNetworkSecurityPolicy(d *schema.ResourceData, obj *types.HostNetworkSecurityPolicy) error {
	d.Set("allow_promiscuous", *obj.AllowPromiscuous)
	d.Set("forged_transmits", *obj.ForgedTransmits)
	d.Set("mac_changes", *obj.MacChanges)
	return nil
}

func expandHostNetworkTrafficShapingPolicy(d *schema.ResourceData) *types.HostNetworkTrafficShapingPolicy {
	enabled := d.Get("shaping_enabled").(bool)
	obj := &types.HostNetworkTrafficShapingPolicy{
		AverageBandwidth: int64(d.Get("shaping_average_bandwidth").(int)),
		BurstSize:        int64(d.Get("shaping_burst_size").(int)),
		Enabled:          &enabled,
		PeakBandwidth:    int64(d.Get("shaping_peak_bandwidth").(int)),
	}
	return obj
}

func flattenHostNetworkTrafficShapingPolicy(d *schema.ResourceData, obj *types.HostNetworkTrafficShapingPolicy) error {
	d.Set("shaping_enabled", *obj.Enabled)
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
