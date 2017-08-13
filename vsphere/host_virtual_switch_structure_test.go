package vsphere

import (
	"reflect"
	"testing"

	"github.com/davecgh/go-spew/spew"
	"github.com/hashicorp/terraform/helper/schema"
)

var schemaHostVirtualSwitchBeaconConfigExpected = &schema.Resource{
	Schema: map[string]*schema.Schema{
		"interval": &schema.Schema{
			Type:        schema.TypeInt,
			Required:    true,
			Description: "Determines how often, in seconds, a beacon should be sent.",
		},
	},
}

var schemaLinkDiscoveryProtocolConfigExpected = &schema.Resource{
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

var schemaHostVirtualSwitchBondBridgeExpected = &schema.Resource{
	Schema: map[string]*schema.Schema{
		"beacon": &schema.Schema{
			Type:        schema.TypeSet,
			Optional:    true,
			Description: "The beacon configuration to probe for the validity of a link. If this is set, beacon probing is configured and will be used. If this is not set, beacon probing is disabled.",
			MaxItems:    1,
			Elem:        schemaHostVirtualSwitchBeaconConfigExpected,
		},
		"link_discovery": &schema.Schema{
			Type:        schema.TypeSet,
			Optional:    true,
			Description: "The link discovery protocol configuration for the virtual switch.",
			MaxItems:    1,
			Elem:        schemaLinkDiscoveryProtocolConfigExpected,
		},
		"network_adapters": &schema.Schema{
			Type:        schema.TypeList,
			Required:    true,
			Description: "The link discovery protocol configuration for the virtual switch.",
			Elem:        &schema.Schema{Type: schema.TypeString},
		},
	},
}

var schemaHostNicFailureCriteriaExpected = &schema.Resource{
	Schema: map[string]*schema.Schema{
		"check_beacon": &schema.Schema{
			Type:        schema.TypeBool,
			Optional:    true,
			Description: "Enable beacon probing. Requires that the vSwitch has been configured to use a beacon. If disabled, link status is used only.",
		},
	},
}

var schemaHostNicOrderPolicyExpected = &schema.Resource{
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

var schemaHostNicTeamingPolicyExpected = &schema.Resource{
	Schema: map[string]*schema.Schema{
		"failure_criteria": &schema.Schema{
			Type:        schema.TypeSet,
			Optional:    true,
			Description: "The failover detection policy for this network adapter team.",
			MaxItems:    1,
			Elem:        schemaHostNicFailureCriteriaExpected,
		},
		"nic_order": &schema.Schema{
			Type:        schema.TypeSet,
			Optional:    true,
			Description: "The failover order policy for network adapters on this switch.",
			MaxItems:    1,
			Elem:        schemaHostNicOrderPolicyExpected,
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

var schemaHostNetworkSecurityPolicyExpected = &schema.Resource{
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

var schemaHostNetworkTrafficShapingPolicyExpected = &schema.Resource{
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

var schemaHostNetworkPolicyExpected = &schema.Resource{
	Schema: map[string]*schema.Schema{
		"nic_teaming": &schema.Schema{
			Type:        schema.TypeSet,
			Optional:    true,
			Description: "The network adapter teaming policy.",
			MaxItems:    1,
			Elem:        schemaHostNicTeamingPolicyExpected,
		},
		"security": &schema.Schema{
			Type:        schema.TypeSet,
			Optional:    true,
			Description: "The security policy governing ports on this virtual switch.",
			MaxItems:    1,
			Elem:        schemaHostNetworkSecurityPolicyExpected,
		},
		"shaping_policy": &schema.Schema{
			Type:        schema.TypeSet,
			Optional:    true,
			Description: "The traffic shaping policy.",
			MaxItems:    1,
			Elem:        schemaHostNetworkTrafficShapingPolicyExpected,
		},
	},
}

var schemaHostVirtualSwitchSpecExpected = &schema.Resource{
	Schema: map[string]*schema.Schema{
		"bridge": &schema.Schema{
			Type:        schema.TypeSet,
			Optional:    true,
			Description: "The physical network adapter specification.",
			MaxItems:    1,
			Elem:        schemaHostVirtualSwitchBondBridgeExpected,
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
			Elem:        schemaHostNetworkPolicyExpected,
		},
	},
}

var testHostVirtualSwitchSchemaCases = []struct {
	Name       string
	SchemaFunc func() *schema.Resource
	Expected   *schema.Resource
}{
	{
		Name:       "HostVirtualSwitchBeaconConfig",
		SchemaFunc: schemaHostVirtualSwitchBeaconConfig,
		Expected:   schemaHostVirtualSwitchBeaconConfigExpected,
	},
	{
		Name:       "LinkDiscoveryProtocolConfig",
		SchemaFunc: schemaLinkDiscoveryProtocolConfig,
		Expected:   schemaLinkDiscoveryProtocolConfigExpected,
	},
	{
		Name:       "HostVirtualSwitchBondBridge",
		SchemaFunc: schemaHostVirtualSwitchBondBridge,
		Expected:   schemaHostVirtualSwitchBondBridgeExpected,
	},
	{
		Name:       "HostNicFailureCriteria",
		SchemaFunc: schemaHostNicFailureCriteria,
		Expected:   schemaHostNicFailureCriteriaExpected,
	},
	{
		Name:       "HostNicOrderPolicy",
		SchemaFunc: schemaHostNicOrderPolicy,
		Expected:   schemaHostNicOrderPolicyExpected,
	},
	{
		Name:       "HostNicTeamingPolicy",
		SchemaFunc: schemaHostNicTeamingPolicy,
		Expected:   schemaHostNicTeamingPolicyExpected,
	},
	{
		Name:       "HostNetworkSecurityPolicy",
		SchemaFunc: schemaHostNetworkSecurityPolicy,
		Expected:   schemaHostNetworkSecurityPolicyExpected,
	},
	{
		Name:       "HostNetworkTrafficShapingPolicy",
		SchemaFunc: schemaHostNetworkTrafficShapingPolicy,
		Expected:   schemaHostNetworkTrafficShapingPolicyExpected,
	},
	{
		Name:       "HostNetworkPolicy",
		SchemaFunc: schemaHostNetworkPolicy,
		Expected:   schemaHostNetworkPolicyExpected,
	},
	{
		Name:       "HostVirtualSwitchSpec",
		SchemaFunc: schemaHostVirtualSwitchSpec,
		Expected:   schemaHostVirtualSwitchSpecExpected,
	},
}

func TestHostVirtualSwitchSchema(t *testing.T) {
	for _, tc := range testHostVirtualSwitchSchemaCases {
		t.Run(tc.Name, func(t *testing.T) {
			actual := tc.SchemaFunc()
			if !reflect.DeepEqual(tc.Expected, actual) {
				t.Fatalf("\n\nExpected:\n\n %s\ngot:\n\n%s\n", spew.Sdump(tc.Expected), spew.Sdump(actual))
			}
		})
	}
}
