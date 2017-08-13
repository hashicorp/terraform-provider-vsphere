package vsphere

import (
	"github.com/hashicorp/terraform/helper/schema"
	"github.com/vmware/govmomi/vim25/types"
)

// schemaHostVirtualSwitchBeaconConfig returns a *schema.Resource representing
// the layout for a HostVirtualSwitchBeaconConfig sub-resource.
func schemaHostVirtualSwitchBeaconConfig() *schema.Resource {
	return &schema.Resource{
		Schema: map[string]*schema.Schema{
			"interval": &schema.Schema{
				Type:        schema.TypeInt,
				Required:    true,
				Description: "Determines how often, in seconds, a beacon should be sent.",
			},
		},
	}
}

// resourceToHostVirtualSwitchBeaconConfig converts a map[string]interface{} of
// resource data to a HostVirtualSwitchBeaconConfig.
func resourceToHostVirtualSwitchBeaconConfig(m map[string]interface{}) *types.HostVirtualSwitchBeaconConfig {
	obj := &types.HostVirtualSwitchBeaconConfig{
		Interval: int32(m["interval"].(int)),
	}
	return obj
}

// hostVirtualSwitchBeaconConfigToResource converts a
// HostVirtualSwitchBeaconConfig to a *schema.Set of resource data.
func hostVirtualSwitchBeaconConfigToResource(obj *types.HostVirtualSwitchBeaconConfig) *schema.Set {
	m := make(map[string]interface{})
	m["interval"] = obj.Interval
	return schema.NewSet(schema.HashResource(schemaHostVirtualSwitchBeaconConfig()), []interface{}{m})
}

// schemaLinkDiscoveryProtocolConfig returns a *schema.Resource representing
// the layout for a LinkDiscoveryProtocolConfig sub-resource.
func schemaLinkDiscoveryProtocolConfig() *schema.Resource {
	return &schema.Resource{
		Schema: map[string]*schema.Schema{
			"operation": &schema.Schema{
				Type:        schema.TypeInt,
				Required:    true,
				Description: "Whether to advertise or listen. Valid values are \"advertise\", \"both\", \"listen\", and \"none\".",
			},
			"protocol": &schema.Schema{
				Type:        schema.TypeInt,
				Required:    true,
				Description: "The discovery protocol type. Valid values are \"cdp\" and \"lldp\".",
			},
		},
	}
}

// resourceToLinkDiscoveryProtocolConfig converts a map[string]interface{} of
// resource data to a LinkDiscoveryProtocolConfig.
func resourceToLinkDiscoveryProtocolConfig(m map[string]interface{}) *types.LinkDiscoveryProtocolConfig {
	obj := &types.LinkDiscoveryProtocolConfig{
		Operation: m["operation"].(string),
		Protocol:  m["protocol"].(string),
	}
	return obj
}

// linkDiscoveryProtocolConfigToResource converts a LinkDiscoveryProtocolConfig
// to a *schema.Set of resource data.
func linkDiscoveryProtocolConfigToResource(obj *types.LinkDiscoveryProtocolConfig) *schema.Set {
	m := make(map[string]interface{})
	m["operation"] = obj.Operation
	m["protocol"] = obj.Protocol
	return schema.NewSet(schema.HashResource(schemaLinkDiscoveryProtocolConfig()), []interface{}{m})
}

// schemaHostVirtualSwitchBondBridge returns a *schema.Resource representing
// the layout for a HostVirtualSwitchBondBridge sub-resource.
func schemaHostVirtualSwitchBondBridge() *schema.Resource {
	return &schema.Resource{
		Schema: map[string]*schema.Schema{
			"beacon": &schema.Schema{
				Type:        schema.TypeSet,
				Optional:    true,
				Description: "The beacon configuration to probe for the validity of a link. If this is set, beacon probing is configured and will be used. If this is not set, beacon probing is disabled.",
				MaxItems:    1,
				Elem:        schemaHostVirtualSwitchBeaconConfig(),
			},
			"link_discovery": &schema.Schema{
				Type:        schema.TypeSet,
				Optional:    true,
				Description: "The link discovery protocol configuration for the virtual switch.",
				MaxItems:    1,
				Elem:        schemaLinkDiscoveryProtocolConfig(),
			},
			"network_adapters": &schema.Schema{
				Type:        schema.TypeList,
				Required:    true,
				Description: "The link discovery protocol configuration for the virtual switch.",
				Elem:        &schema.Schema{Type: schema.TypeString},
			},
		},
	}
}

// resourceToHostVirtualSwitchBondBridge converts a map[string]interface{} of
// resource data to a HostVirtualSwitchBondBridge.
func resourceToHostVirtualSwitchBondBridge(m map[string]interface{}) *types.HostVirtualSwitchBondBridge {
	obj := &types.HostVirtualSwitchBondBridge{
		Beacon: resourceToHostVirtualSwitchBeaconConfig(m["beacon"].(*schema.Set).List()[0].(map[string]interface{})),
		LinkDiscoveryProtocolConfig: resourceToLinkDiscoveryProtocolConfig(m["link_discovery"].(*schema.Set).List()[0].(map[string]interface{})),
		NicDevice:                   sliceInterfacesToStrings(m["network_adapters"].([]interface{})),
	}
	return obj
}

// hostVirtualSwitchBondBridgeToResource converts a HostVirtualSwitchBondBridge
// to a *schema.Set of resource data.
func hostVirtualSwitchBondBridgeToResource(obj *types.HostVirtualSwitchBondBridge) *schema.Set {
	m := make(map[string]interface{})
	m["beacon"] = hostVirtualSwitchBeaconConfigToResource(obj.Beacon)
	m["link_discovery"] = linkDiscoveryProtocolConfigToResource(obj.LinkDiscoveryProtocolConfig)
	m["network_adapters"] = obj.NicDevice
	return schema.NewSet(schema.HashResource(schemaHostVirtualSwitchBondBridge()), []interface{}{m})
}

// schemaHostNicFailureCriteria returns a *schema.Resource representing the
// layout for a HostNicFailureCriteria sub-resource.
func schemaHostNicFailureCriteria() *schema.Resource {
	return &schema.Resource{
		Schema: map[string]*schema.Schema{
			"check_beacon": &schema.Schema{
				Type:        schema.TypeBool,
				Optional:    true,
				Description: "Enable beacon probing. Requires that the vSwitch has been configured to use a beacon. If disabled, link status is used only.",
			},
		},
	}
}

// resourceToHostNicFailureCriteria converts a map[string]interface{} of
// resource data to a HostNicFailureCriteria.
func resourceToHostNicFailureCriteria(m map[string]interface{}) *types.HostNicFailureCriteria {
	checkBeacon := m["check_beacon"].(bool)
	obj := &types.HostNicFailureCriteria{
		CheckBeacon: &checkBeacon,
	}
	return obj
}

// hostNicFailureCriteriaToResource converts a HostNicFailureCriteria to a
// *schema.Set of resource data.
func hostNicFailureCriteriaToResource(obj *types.HostNicFailureCriteria) *schema.Set {
	m := make(map[string]interface{})
	m["check_beacon"] = *obj.CheckBeacon
	return schema.NewSet(schema.HashResource(schemaHostNicFailureCriteria()), []interface{}{m})
}

// schemaHostNicOrderPolicy returns a *schema.Resource representing the layout
// for a HostNicOrderPolicy sub-resource.
func schemaHostNicOrderPolicy() *schema.Resource {
	return &schema.Resource{
		Schema: map[string]*schema.Schema{
			"active_nics": &schema.Schema{
				Type:        schema.TypeList,
				Optional:    true,
				Description: "List of active network adapters used for load balancing.",
				Elem:        &schema.Schema{Type: schema.TypeString},
			},
			"standby_nics": &schema.Schema{
				Type:        schema.TypeList,
				Optional:    true,
				Description: "List of standby network adapters used for failover.",
				Elem:        &schema.Schema{Type: schema.TypeString},
			},
		},
	}
}

// resourceToHostNicOrderPolicy converts a map[string]interface{} of resource
// data to a HostNicOrderPolicy.
func resourceToHostNicOrderPolicy(m map[string]interface{}) *types.HostNicOrderPolicy {
	obj := &types.HostNicOrderPolicy{
		ActiveNic:  sliceInterfacesToStrings(m["active_nics"].([]interface{})),
		StandbyNic: sliceInterfacesToStrings(m["standby_nics"].([]interface{})),
	}
	return obj
}

// hostNicOrderPolicyToResource converts a HostNicOrderPolicy to a *schema.Set
// of resource data.
func hostNicOrderPolicyToResource(obj *types.HostNicOrderPolicy) *schema.Set {
	m := make(map[string]interface{})
	m["active_nics"] = obj.ActiveNic
	m["standby_nics"] = obj.StandbyNic
	return schema.NewSet(schema.HashResource(schemaHostNicOrderPolicy()), []interface{}{m})
}

// schemaHostNicTeamingPolicy returns a *schema.Resource representing the layout
// for a HostNicTeamingPolicy sub-resource.
func schemaHostNicTeamingPolicy() *schema.Resource {
	return &schema.Resource{
		Schema: map[string]*schema.Schema{
			"failure_criteria": &schema.Schema{
				Type:        schema.TypeSet,
				Optional:    true,
				Description: "The failover detection policy for this network adapter team.",
				MaxItems:    1,
				Elem:        schemaHostNicFailureCriteria(),
			},
			"nic_order": &schema.Schema{
				Type:        schema.TypeSet,
				Optional:    true,
				Description: "The failover order policy for network adapters on this switch.",
				MaxItems:    1,
				Elem:        schemaHostNicOrderPolicy(),
			},
			"policy": &schema.Schema{
				Type:        schema.TypeString,
				Optional:    true,
				Description: "The network adapter teaming policy. Can be one of loadbalance_ip, loadbalance_srcmac, loadbalance_srcid, or failover_explicit.",
			},
			"failback": &schema.Schema{
				Type:        schema.TypeBool,
				Optional:    true,
				Description: "If true, the teaming policy will re-activate failed interfaces higher in precedence when they come back up.",
			},
		},
	}
}

// resourceToHostNicTeamingPolicy converts a map[string]interface{} of resource
// data to a HostNicTeamingPolicy.
func resourceToHostNicTeamingPolicy(m map[string]interface{}) *types.HostNicTeamingPolicy {
	rollingOrder := !m["failback"].(bool)
	obj := &types.HostNicTeamingPolicy{
		FailureCriteria: resourceToHostNicFailureCriteria(m["failure_criteria"].(*schema.Set).List()[0].(map[string]interface{})),
		NicOrder:        resourceToHostNicOrderPolicy(m["nic_order"].(*schema.Set).List()[0].(map[string]interface{})),
		Policy:          m["policy"].(string),
		RollingOrder:    &rollingOrder,
	}
	return obj
}

// hostNicTeamingPolicyToResource converts a HostNicTeamingPolicy to a *schema.Set
// of resource data.
func hostNicTeamingPolicyToResource(obj *types.HostNicTeamingPolicy) *schema.Set {
	m := make(map[string]interface{})
	m["failure_criteria"] = hostNicFailureCriteriaToResource(obj.FailureCriteria)
	m["nic_order"] = hostNicOrderPolicyToResource(obj.NicOrder)
	m["policy"] = obj.Policy
	m["failback"] = !*obj.RollingOrder
	return schema.NewSet(schema.HashResource(schemaHostNicTeamingPolicy()), []interface{}{m})
}

// schemaHostNetworkSecurityPolicy returns a *schema.Resource representing the layout
// for a HostNetworkSecurityPolicy sub-resource.
func schemaHostNetworkSecurityPolicy() *schema.Resource {
	return &schema.Resource{
		Schema: map[string]*schema.Schema{
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
		},
	}
}

// resourceToHostNetworkSecurityPolicy converts a map[string]interface{} of resource
// data to a HostNetworkSecurityPolicy.
func resourceToHostNetworkSecurityPolicy(m map[string]interface{}) *types.HostNetworkSecurityPolicy {
	allowPromiscuous := m["allow_promiscuous"].(bool)
	forgedTransmits := m["forged_transmits"].(bool)
	macChanges := m["mac_changes"].(bool)
	obj := &types.HostNetworkSecurityPolicy{
		AllowPromiscuous: &allowPromiscuous,
		ForgedTransmits:  &forgedTransmits,
		MacChanges:       &macChanges,
	}
	return obj
}

// hostNetworkSecurityPolicyToResource converts a HostNetworkSecurityPolicy to a *schema.Set
// of resource data.
func hostNetworkSecurityPolicyToResource(obj *types.HostNetworkSecurityPolicy) *schema.Set {
	m := make(map[string]interface{})
	m["allow_promiscuous"] = *obj.AllowPromiscuous
	m["forged_transmits"] = *obj.ForgedTransmits
	m["mac_changes"] = *obj.MacChanges
	return schema.NewSet(schema.HashResource(schemaHostNetworkSecurityPolicy()), []interface{}{m})
}

// schemaHostNetworkTrafficShapingPolicy returns a *schema.Resource representing the layout
// for a HostNetworkTrafficShapingPolicy sub-resource.
func schemaHostNetworkTrafficShapingPolicy() *schema.Resource {
	return &schema.Resource{
		Schema: map[string]*schema.Schema{
			"average_bandwidth": &schema.Schema{
				Type:        schema.TypeInt,
				Optional:    true,
				Description: "The average bandwidth in bits per second if shaping is enabled on the port.",
			},
			"burst_size": &schema.Schema{
				Type:        schema.TypeInt,
				Optional:    true,
				Description: "The maximum burst size allowed in bytes if shaping is enabled on the port.",
			},
			"enabled": &schema.Schema{
				Type:        schema.TypeBool,
				Optional:    true,
				Description: "True if the traffic shaper is enabled on the port.",
			},
			"peak_bandwidth": &schema.Schema{
				Type:        schema.TypeInt,
				Optional:    true,
				Description: "The peak bandwidth during bursts in bits per second if traffic shaping is enabled on the port.",
			},
		},
	}
}

// resourceToHostNetworkTrafficShapingPolicy converts a map[string]interface{} of resource
// data to a HostNetworkTrafficShapingPolicy.
func resourceToHostNetworkTrafficShapingPolicy(m map[string]interface{}) *types.HostNetworkTrafficShapingPolicy {
	enabled := m["enabled"].(bool)
	obj := &types.HostNetworkTrafficShapingPolicy{
		AverageBandwidth: int64(m["average_bandwidth"].(int)),
		BurstSize:        int64(m["burst_size"].(int)),
		Enabled:          &enabled,
		PeakBandwidth:    int64(m["peak_bandwidth"].(int)),
	}
	return obj
}

// hostNetworkTrafficShapingPolicyToResource converts a HostNetworkTrafficShapingPolicy to a *schema.Set
// of resource data.
func hostNetworkTrafficShapingPolicyToResource(obj *types.HostNetworkTrafficShapingPolicy) *schema.Set {
	m := make(map[string]interface{})
	m["average_bandwidth"] = obj.AverageBandwidth
	m["burst_size"] = obj.BurstSize
	m["enabled"] = *obj.Enabled
	m["peak_bandwidth"] = obj.PeakBandwidth
	return schema.NewSet(schema.HashResource(schemaHostNetworkTrafficShapingPolicy()), []interface{}{m})
}

// schemaHostNetworkPolicy returns a *schema.Resource representing the layout
// for a HostNetworkPolicy sub-resource.
func schemaHostNetworkPolicy() *schema.Resource {
	return &schema.Resource{
		Schema: map[string]*schema.Schema{
			"nic_teaming": &schema.Schema{
				Type:        schema.TypeSet,
				Optional:    true,
				Description: "The network adapter teaming policy.",
				MaxItems:    1,
				Elem:        schemaHostNicTeamingPolicy(),
			},
			"security": &schema.Schema{
				Type:        schema.TypeSet,
				Optional:    true,
				Description: "The security policy governing ports on this virtual switch.",
				MaxItems:    1,
				Elem:        schemaHostNetworkSecurityPolicy(),
			},
			"shaping_policy": &schema.Schema{
				Type:        schema.TypeSet,
				Optional:    true,
				Description: "The traffic shaping policy.",
				MaxItems:    1,
				Elem:        schemaHostNetworkTrafficShapingPolicy(),
			},
		},
	}
}

// resourceToHostNetworkPolicy converts a map[string]interface{} of resource
// data to a HostNetworkPolicy.
func resourceToHostNetworkPolicy(m map[string]interface{}) *types.HostNetworkPolicy {
	obj := &types.HostNetworkPolicy{
		NicTeaming:    resourceToHostNicTeamingPolicy(m["nic_teaming"].(*schema.Set).List()[0].(map[string]interface{})),
		Security:      resourceToHostNetworkSecurityPolicy(m["security"].(*schema.Set).List()[0].(map[string]interface{})),
		ShapingPolicy: resourceToHostNetworkTrafficShapingPolicy(m["shaping_policy"].(*schema.Set).List()[0].(map[string]interface{})),
	}
	return obj
}

// hostNetworkPolicyToResource converts a HostNetworkPolicy to a *schema.Set
// of resource data.
func hostNetworkPolicyToResource(obj *types.HostNetworkPolicy) *schema.Set {
	m := make(map[string]interface{})
	m["nic_teaming"] = hostNicTeamingPolicyToResource(obj.NicTeaming)
	m["security"] = hostNetworkSecurityPolicyToResource(obj.Security)
	m["shaping_policy"] = hostNetworkTrafficShapingPolicyToResource(obj.ShapingPolicy)
	return schema.NewSet(schema.HashResource(schemaHostNetworkPolicy()), []interface{}{m})
}

// schemaHostVirtualSwitchSpec returns a *schema.Resource representing the layout
// for a HostVirtualSwitchSpec sub-resource.
func schemaHostVirtualSwitchSpec() *schema.Resource {
	return &schema.Resource{
		Schema: map[string]*schema.Schema{
			"bridge": &schema.Schema{
				Type:        schema.TypeSet,
				Optional:    true,
				Description: "The physical network adapter specification.",
				MaxItems:    1,
				Elem:        schemaHostVirtualSwitchBondBridge(),
			},
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
			"policy": &schema.Schema{
				Type:        schema.TypeSet,
				Optional:    true,
				Description: "The virtual switch policy specification. This has a lower precedence than any port groups you assign to this switch.",
				MaxItems:    1,
				Elem:        schemaHostNetworkPolicy(),
			},
		},
	}
}

// resourceToHostVirtualSwitchSpec converts a map[string]interface{} of resource
// data to a HostVirtualSwitchSpec.
func resourceToHostVirtualSwitchSpec(m map[string]interface{}) *types.HostVirtualSwitchSpec {
	obj := &types.HostVirtualSwitchSpec{
		Bridge:   resourceToHostVirtualSwitchBondBridge(m["bridge"].(*schema.Set).List()[0].(map[string]interface{})),
		Mtu:      int32(m["mtu"].(int)),
		NumPorts: int32(m["number_of_ports"].(int)),
		Policy:   resourceToHostNetworkPolicy(m["policy"].(*schema.Set).List()[0].(map[string]interface{})),
	}
	return obj
}

// hostVirtualSwitchSpecToResource converts a HostVirtualSwitchSpec to a *schema.Set
// of resource data.
func hostVirtualSwitchSpecToResource(obj *types.HostVirtualSwitchSpec) *schema.Set {
	m := make(map[string]interface{})
	m["bridge"] = hostVirtualSwitchBondBridgeToResource(obj.Bridge.(*types.HostVirtualSwitchBondBridge))
	m["mtu"] = obj.Mtu
	m["number_of_ports"] = obj.NumPorts
	m["policy"] = hostNetworkPolicyToResource(obj.Policy)
	return schema.NewSet(schema.HashResource(schemaHostVirtualSwitchSpec()), []interface{}{m})
}
