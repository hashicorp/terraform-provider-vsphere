package vsphere

import (
	"reflect"
	"testing"

	"github.com/davecgh/go-spew/spew"
	"github.com/hashicorp/terraform/helper/schema"
	"github.com/vmware/govmomi/vim25/types"
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
			Type:        schema.TypeString,
			Required:    true,
			Description: "Whether to advertise or listen. Valid values are \"advertise\", \"both\", \"listen\", and \"none\".",
		},
		"protocol": &schema.Schema{
			Type:        schema.TypeString,
			Required:    true,
			Description: "The discovery protocol type. Valid values are \"cdp\" and \"lldp\".",
		},
	},
}

var schemaHostVirtualSwitchBondBridgeExpected = &schema.Resource{
	Schema: map[string]*schema.Schema{
		"beacon": &schema.Schema{
			Type:        schema.TypeList,
			Optional:    true,
			Computed:    true,
			Description: "The beacon configuration to probe for the validity of a link. If this is set, beacon probing is configured and will be used. If this is not set, beacon probing is disabled.",
			MaxItems:    1,
			Elem:        schemaHostVirtualSwitchBeaconConfigExpected,
		},
		"link_discovery": &schema.Schema{
			Type:        schema.TypeList,
			Optional:    true,
			Computed:    true,
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
			Computed:    true,
			Description: "Enable beacon probing. Requires that the vSwitch has been configured to use a beacon. If disabled, link status is used only.",
		},
	},
}

var schemaHostNicOrderPolicyExpected = &schema.Resource{
	Schema: map[string]*schema.Schema{
		"active_nics": &schema.Schema{
			Type:        schema.TypeList,
			Optional:    true,
			Computed:    true,
			Description: "List of active network adapters used for load balancing.",
			Elem:        &schema.Schema{Type: schema.TypeString},
		},
		"standby_nics": &schema.Schema{
			Type:        schema.TypeList,
			Optional:    true,
			Computed:    true,
			Description: "List of standby network adapters used for failover.",
			Elem:        &schema.Schema{Type: schema.TypeString},
		},
	},
}

var schemaHostNicTeamingPolicyExpected = &schema.Resource{
	Schema: map[string]*schema.Schema{
		"failure_criteria": &schema.Schema{
			Type:        schema.TypeList,
			Optional:    true,
			Computed:    true,
			Description: "The failover detection policy for this network adapter team.",
			MaxItems:    1,
			Elem:        schemaHostNicFailureCriteriaExpected,
		},
		"nic_order": &schema.Schema{
			Type:        schema.TypeList,
			Optional:    true,
			Computed:    true,
			Description: "The failover order policy for network adapters on this switch.",
			MaxItems:    1,
			Elem:        schemaHostNicOrderPolicyExpected,
		},
		"policy": &schema.Schema{
			Type:        schema.TypeString,
			Optional:    true,
			Computed:    true,
			Description: "The network adapter teaming policy. Can be one of loadbalance_ip, loadbalance_srcmac, loadbalance_srcid, or failover_explicit.",
		},
		"failback": &schema.Schema{
			Type:        schema.TypeBool,
			Optional:    true,
			Computed:    true,
			Description: "If true, the teaming policy will re-activate failed interfaces higher in precedence when they come back up.",
		},
	},
}

var schemaHostNetworkSecurityPolicyExpected = &schema.Resource{
	Schema: map[string]*schema.Schema{
		"allow_promiscuous": &schema.Schema{
			Type:        schema.TypeBool,
			Optional:    true,
			Computed:    true,
			Description: "Enable promiscuious mode on the network. This flag indicates whether or not all traffic is seen on a given port.",
		},
		"forged_transmits": &schema.Schema{
			Type:        schema.TypeBool,
			Optional:    true,
			Computed:    true,
			Description: "Controls whether or not the virtual network adapter is allowed to send network traffic with a different MAC address than that of its own.",
		},
		"mac_changes": &schema.Schema{
			Type:        schema.TypeBool,
			Optional:    true,
			Computed:    true,
			Description: "Controls whether or not the Media Access Control (MAC) address can be changed.",
		},
	},
}

var schemaHostNetworkTrafficShapingPolicyExpected = &schema.Resource{
	Schema: map[string]*schema.Schema{
		"average_bandwidth": &schema.Schema{
			Type:        schema.TypeInt,
			Optional:    true,
			Computed:    true,
			Description: "The average bandwidth in bits per second if shaping is enabled on the port.",
		},
		"burst_size": &schema.Schema{
			Type:        schema.TypeInt,
			Optional:    true,
			Computed:    true,
			Description: "The maximum burst size allowed in bytes if shaping is enabled on the port.",
		},
		"enabled": &schema.Schema{
			Type:        schema.TypeBool,
			Optional:    true,
			Computed:    true,
			Description: "True if the traffic shaper is enabled on the port.",
		},
		"peak_bandwidth": &schema.Schema{
			Type:        schema.TypeInt,
			Optional:    true,
			Computed:    true,
			Description: "The peak bandwidth during bursts in bits per second if traffic shaping is enabled on the port.",
		},
	},
}

var schemaHostNetworkPolicyExpected = &schema.Resource{
	Schema: map[string]*schema.Schema{
		"nic_teaming": &schema.Schema{
			Type:        schema.TypeList,
			Optional:    true,
			Computed:    true,
			Description: "The network adapter teaming policy.",
			MaxItems:    1,
			Elem:        schemaHostNicTeamingPolicyExpected,
		},
		"security": &schema.Schema{
			Type:        schema.TypeList,
			Optional:    true,
			Computed:    true,
			Description: "The security policy governing ports on this virtual switch.",
			MaxItems:    1,
			Elem:        schemaHostNetworkSecurityPolicyExpected,
		},
		"shaping_policy": &schema.Schema{
			Type:        schema.TypeList,
			Optional:    true,
			Computed:    true,
			Description: "The traffic shaping policy.",
			MaxItems:    1,
			Elem:        schemaHostNetworkTrafficShapingPolicyExpected,
		},
	},
}

var schemaHostVirtualSwitchSpecExpected = &schema.Resource{
	Schema: map[string]*schema.Schema{
		"bridge": &schema.Schema{
			Type:        schema.TypeList,
			Optional:    true,
			Computed:    true,
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
			Type:        schema.TypeList,
			Optional:    true,
			Computed:    true,
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

var resourceToHostVirtualSwitchBeaconConfigInput = map[string]interface{}{
	"interval": 10,
}

var resourceToHostVirtualSwitchBeaconConfigExpected = &types.HostVirtualSwitchBeaconConfig{
	Interval: 10,
}

var hostVirtualSwitchBeaconConfigToResourceExpected = []interface{}{resourceToHostVirtualSwitchBeaconConfigInput}

var resourceToLinkDiscoveryProtocolConfigInput = map[string]interface{}{
	"operation": "listen",
	"protocol":  "cdp",
}

var resourceToLinkDiscoveryProtocolConfigExpected = &types.LinkDiscoveryProtocolConfig{
	Operation: "listen",
	Protocol:  "cdp",
}

var linkDiscoveryProtocolConfigToResourceExpected = []interface{}{resourceToLinkDiscoveryProtocolConfigInput}

var resourceToHostVirtualSwitchBondBridgeInput = map[string]interface{}{
	"beacon":           hostVirtualSwitchBeaconConfigToResourceExpected,
	"link_discovery":   linkDiscoveryProtocolConfigToResourceExpected,
	"network_adapters": []interface{}{"vmnic0", "vmnic1"},
}

var resourceToHostVirtualSwitchBondBridgeExpected = &types.HostVirtualSwitchBondBridge{
	Beacon: resourceToHostVirtualSwitchBeaconConfigExpected,
	LinkDiscoveryProtocolConfig: resourceToLinkDiscoveryProtocolConfigExpected,
	NicDevice:                   []string{"vmnic0", "vmnic1"},
}

var hostVirtualSwitchBondBridgeToResourceExpected = []interface{}{resourceToHostVirtualSwitchBondBridgeInput}

var resourceToHostNicFailureCriteriaInput = map[string]interface{}{
	"check_beacon": true,
}

var resourceToHostNicFailureCriteriaExpected = &types.HostNicFailureCriteria{
	CheckBeacon: &[]bool{true}[0],
}

var hostNicFailureCriteriaToResourceExpected = []interface{}{resourceToHostNicFailureCriteriaInput}

var resourceToHostNicOrderPolicyInput = map[string]interface{}{
	"active_nics":  []interface{}{"vmnic0", "vmnic1"},
	"standby_nics": []interface{}{"vmnic2", "vmnic3"},
}

var resourceToHostNicOrderPolicyExpected = &types.HostNicOrderPolicy{
	ActiveNic:  []string{"vmnic0", "vmnic1"},
	StandbyNic: []string{"vmnic2", "vmnic3"},
}

var hostNicOrderPolicyToResourceExpected = []interface{}{resourceToHostNicOrderPolicyInput}

var resourceToHostNicTeamingPolicyInput = map[string]interface{}{
	"failure_criteria": hostNicFailureCriteriaToResourceExpected,
	"nic_order":        hostNicOrderPolicyToResourceExpected,
	"policy":           "failover_explicit",
	"failback":         true,
}

var resourceToHostNicTeamingPolicyExpected = &types.HostNicTeamingPolicy{
	FailureCriteria: resourceToHostNicFailureCriteriaExpected,
	NicOrder:        resourceToHostNicOrderPolicyExpected,
	Policy:          "failover_explicit",
	RollingOrder:    &[]bool{false}[0],
}

var hostNicTeamingPolicyToResourceExpected = []interface{}{resourceToHostNicTeamingPolicyInput}

var resourceToHostNetworkSecurityPolicyInput = map[string]interface{}{
	"allow_promiscuous": true,
	"forged_transmits":  true,
	"mac_changes":       true,
}

var resourceToHostNetworkSecurityPolicyExpected = &types.HostNetworkSecurityPolicy{
	AllowPromiscuous: &[]bool{true}[0],
	ForgedTransmits:  &[]bool{true}[0],
	MacChanges:       &[]bool{true}[0],
}

var hostNetworkSecurityPolicyToResourceExpected = []interface{}{resourceToHostNetworkSecurityPolicyInput}

var resourceToHostNetworkTrafficShapingPolicyInput = map[string]interface{}{
	"average_bandwidth": 100000000,
	"burst_size":        1000000000,
	"enabled":           true,
	"peak_bandwidth":    500000000,
}

var resourceToHostNetworkTrafficShapingPolicyExpected = &types.HostNetworkTrafficShapingPolicy{
	AverageBandwidth: 100000000,
	BurstSize:        1000000000,
	Enabled:          &[]bool{true}[0],
	PeakBandwidth:    500000000,
}

var hostNetworkTrafficShapingPolicyToResourceExpected = []interface{}{resourceToHostNetworkTrafficShapingPolicyInput}

var resourceToHostNetworkPolicyInput = map[string]interface{}{
	"nic_teaming":    hostNicTeamingPolicyToResourceExpected,
	"security":       hostNetworkSecurityPolicyToResourceExpected,
	"shaping_policy": hostNetworkTrafficShapingPolicyToResourceExpected,
}

var resourceToHostNetworkPolicyExpected = &types.HostNetworkPolicy{
	NicTeaming:    resourceToHostNicTeamingPolicyExpected,
	Security:      resourceToHostNetworkSecurityPolicyExpected,
	ShapingPolicy: resourceToHostNetworkTrafficShapingPolicyExpected,
}

var hostNetworkPolicyToResourceExpected = []interface{}{resourceToHostNetworkPolicyInput}

var resourceToHostVirtualSwitchSpecInput = map[string]interface{}{
	"bridge":          hostVirtualSwitchBondBridgeToResourceExpected,
	"mtu":             9000,
	"number_of_ports": 256,
	"policy":          hostNetworkPolicyToResourceExpected,
}

var resourceToHostVirtualSwitchSpecExpected = &types.HostVirtualSwitchSpec{
	Bridge:   resourceToHostVirtualSwitchBondBridgeExpected,
	Mtu:      9000,
	NumPorts: 256,
	Policy:   resourceToHostNetworkPolicyExpected,
}

var hostVirtualSwitchSpecToResourceExpected = []interface{}{resourceToHostVirtualSwitchSpecInput}

func TestResourceToHostVirtualSwitchBeaconConfig(t *testing.T) {
	in := resourceToHostVirtualSwitchBeaconConfigInput
	expected := resourceToHostVirtualSwitchBeaconConfigExpected
	actual := resourceToHostVirtualSwitchBeaconConfig(in)
	if !reflect.DeepEqual(expected, actual) {
		t.Fatalf("\nExpected:\n\n%s\n\ngot:\n\n%s", spew.Sdump(expected), spew.Sdump(actual))
	}
}

func TestResourceToLinkDiscoveryProtocolConfig(t *testing.T) {
	in := resourceToLinkDiscoveryProtocolConfigInput
	expected := resourceToLinkDiscoveryProtocolConfigExpected
	actual := resourceToLinkDiscoveryProtocolConfig(in)
	if !reflect.DeepEqual(expected, actual) {
		t.Fatalf("\nExpected:\n\n%s\n\ngot:\n\n%s", spew.Sdump(expected), spew.Sdump(actual))
	}
}

func TestResourceToHostVirtualSwitchBondBridge(t *testing.T) {
	in := resourceToHostVirtualSwitchBondBridgeInput
	expected := resourceToHostVirtualSwitchBondBridgeExpected
	actual := resourceToHostVirtualSwitchBondBridge(in)
	if !reflect.DeepEqual(expected, actual) {
		t.Fatalf("\nExpected:\n\n%s\n\ngot:\n\n%s", spew.Sdump(expected), spew.Sdump(actual))
	}
}

func TestResourceToHostNicFailureCriteria(t *testing.T) {
	in := resourceToHostNicFailureCriteriaInput
	expected := resourceToHostNicFailureCriteriaExpected
	actual := resourceToHostNicFailureCriteria(in)
	if !reflect.DeepEqual(expected, actual) {
		t.Fatalf("\nExpected:\n\n%s\n\ngot:\n\n%s", spew.Sdump(expected), spew.Sdump(actual))
	}
}

func TestResourceToHostNicOrderPolicy(t *testing.T) {
	in := resourceToHostNicOrderPolicyInput
	expected := resourceToHostNicOrderPolicyExpected
	actual := resourceToHostNicOrderPolicy(in)
	if !reflect.DeepEqual(expected, actual) {
		t.Fatalf("\nExpected:\n\n%s\n\ngot:\n\n%s", spew.Sdump(expected), spew.Sdump(actual))
	}
}

func TestResourceToHostNicTeamingPolicy(t *testing.T) {
	in := resourceToHostNicTeamingPolicyInput
	expected := resourceToHostNicTeamingPolicyExpected
	actual := resourceToHostNicTeamingPolicy(in)
	if !reflect.DeepEqual(expected, actual) {
		t.Fatalf("\nExpected:\n\n%s\n\ngot:\n\n%s", spew.Sdump(expected), spew.Sdump(actual))
	}
}

func TestResourceToHostNetworkSecurityPolicy(t *testing.T) {
	in := resourceToHostNetworkSecurityPolicyInput
	expected := resourceToHostNetworkSecurityPolicyExpected
	actual := resourceToHostNetworkSecurityPolicy(in)
	if !reflect.DeepEqual(expected, actual) {
		t.Fatalf("\nExpected:\n\n%s\n\ngot:\n\n%s", spew.Sdump(expected), spew.Sdump(actual))
	}
}

func TestResourceToHostNetworkTrafficShapingPolicy(t *testing.T) {
	in := resourceToHostNetworkTrafficShapingPolicyInput
	expected := resourceToHostNetworkTrafficShapingPolicyExpected
	actual := resourceToHostNetworkTrafficShapingPolicy(in)
	if !reflect.DeepEqual(expected, actual) {
		t.Fatalf("\nExpected:\n\n%s\n\ngot:\n\n%s", spew.Sdump(expected), spew.Sdump(actual))
	}
}

func TestResourceToHostNetworkPolicy(t *testing.T) {
	in := resourceToHostNetworkPolicyInput
	expected := resourceToHostNetworkPolicyExpected
	actual := resourceToHostNetworkPolicy(in)
	if !reflect.DeepEqual(expected, actual) {
		t.Fatalf("\nExpected:\n\n%s\n\ngot:\n\n%s", spew.Sdump(expected), spew.Sdump(actual))
	}
}

func TestResourceToHostVirtualSwitchSpec(t *testing.T) {
	in := resourceToHostVirtualSwitchSpecInput
	expected := resourceToHostVirtualSwitchSpecExpected
	actual := resourceToHostVirtualSwitchSpec(in)
	if !reflect.DeepEqual(expected, actual) {
		t.Fatalf("\nExpected:\n\n%s\n\ngot:\n\n%s", spew.Sdump(expected), spew.Sdump(actual))
	}
}

func TestHostVirtualSwitchBeaconConfigToResource(t *testing.T) {
	in := resourceToHostVirtualSwitchBeaconConfigExpected
	expected := hostVirtualSwitchBeaconConfigToResourceExpected
	actual := hostVirtualSwitchBeaconConfigToResource(in)
	if !reflect.DeepEqual(expected, actual) {
		t.Fatalf("\nExpected:\n\n%s\n\ngot:\n\n%s", spew.Sdump(expected), spew.Sdump(actual))
	}
}

func TestLinkDiscoveryProtocolConfigToResource(t *testing.T) {
	in := resourceToLinkDiscoveryProtocolConfigExpected
	expected := linkDiscoveryProtocolConfigToResourceExpected
	actual := linkDiscoveryProtocolConfigToResource(in)
	if !reflect.DeepEqual(expected, actual) {
		t.Fatalf("\nExpected:\n\n%s\n\ngot:\n\n%s", spew.Sdump(expected), spew.Sdump(actual))
	}
}

func TestHostVirtualSwitchBondBridgeToResource(t *testing.T) {
	in := resourceToHostVirtualSwitchBondBridgeExpected
	expected := hostVirtualSwitchBondBridgeToResourceExpected
	actual := hostVirtualSwitchBondBridgeToResource(in)
	if !reflect.DeepEqual(expected, actual) {
		t.Fatalf("\nExpected:\n\n%s\n\ngot:\n\n%s", spew.Sdump(expected), spew.Sdump(actual))
	}
}

func TestHostNicFailureCriteriaToResource(t *testing.T) {
	in := resourceToHostNicFailureCriteriaExpected
	expected := hostNicFailureCriteriaToResourceExpected
	actual := hostNicFailureCriteriaToResource(in)
	if !reflect.DeepEqual(expected, actual) {
		t.Fatalf("\nExpected:\n\n%s\n\ngot:\n\n%s", spew.Sdump(expected), spew.Sdump(actual))
	}
}

func TestHostNicOrderPolicyToResource(t *testing.T) {
	in := resourceToHostNicOrderPolicyExpected
	expected := hostNicOrderPolicyToResourceExpected
	actual := hostNicOrderPolicyToResource(in)
	if !reflect.DeepEqual(expected, actual) {
		t.Fatalf("\nExpected:\n\n%s\n\ngot:\n\n%s", spew.Sdump(expected), spew.Sdump(actual))
	}
}

func TestHostNicTeamingPolicyToResource(t *testing.T) {
	in := resourceToHostNicTeamingPolicyExpected
	expected := hostNicTeamingPolicyToResourceExpected
	actual := hostNicTeamingPolicyToResource(in)
	if !reflect.DeepEqual(expected, actual) {
		t.Fatalf("\nExpected:\n\n%s\n\ngot:\n\n%s", spew.Sdump(expected), spew.Sdump(actual))
	}
}

func TestHostNetworkSecurityPolicyToResource(t *testing.T) {
	in := resourceToHostNetworkSecurityPolicyExpected
	expected := hostNetworkSecurityPolicyToResourceExpected
	actual := hostNetworkSecurityPolicyToResource(in)
	if !reflect.DeepEqual(expected, actual) {
		t.Fatalf("\nExpected:\n\n%s\n\ngot:\n\n%s", spew.Sdump(expected), spew.Sdump(actual))
	}
}

func TestHostNetworkTrafficShapingPolicyToResource(t *testing.T) {
	in := resourceToHostNetworkTrafficShapingPolicyExpected
	expected := hostNetworkTrafficShapingPolicyToResourceExpected
	actual := hostNetworkTrafficShapingPolicyToResource(in)
	if !reflect.DeepEqual(expected, actual) {
		t.Fatalf("\nExpected:\n\n%s\n\ngot:\n\n%s", spew.Sdump(expected), spew.Sdump(actual))
	}
}

func TestHostNetworkPolicyToResource(t *testing.T) {
	in := resourceToHostNetworkPolicyExpected
	expected := hostNetworkPolicyToResourceExpected
	actual := hostNetworkPolicyToResource(in)
	if !reflect.DeepEqual(expected, actual) {
		t.Fatalf("\nExpected:\n\n%s\n\ngot:\n\n%s", spew.Sdump(expected), spew.Sdump(actual))
	}
}

func TestHostVirtualSwitchSpecToResource(t *testing.T) {
	in := resourceToHostVirtualSwitchSpecExpected
	expected := hostVirtualSwitchSpecToResourceExpected
	actual := hostVirtualSwitchSpecToResource(in)
	if !reflect.DeepEqual(expected, actual) {
		t.Fatalf("\nExpected:\n\n%s\n\ngot:\n\n%s", spew.Sdump(expected), spew.Sdump(actual))
	}
}
