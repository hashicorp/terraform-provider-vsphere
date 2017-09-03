package vsphere

import (
	"strings"

	"github.com/hashicorp/terraform/helper/schema"
	"github.com/hashicorp/terraform/helper/validation"
	"github.com/vmware/govmomi/vim25/mo"
	"github.com/vmware/govmomi/vim25/types"
)

var distributedVirtualSwitchNetworkResourceControlVersionAllowedValues = []string{
	string(types.DistributedVirtualSwitchNetworkResourceControlVersionVersion2),
	string(types.DistributedVirtualSwitchNetworkResourceControlVersionVersion3),
}

var configSpecOperationAllowedValues = []string{
	string(types.VirtualDeviceConfigSpecOperationAdd),
	string(types.VirtualDeviceConfigSpecOperationRemove),
	string(types.VirtualDeviceConfigSpecOperationEdit),
}

func schemaDVSContactInfo() *schema.Schema {
	return &schema.Schema{
		Type:     schema.TypeSet,
		Optional: true,
		Elem: &schema.Resource{
			Schema: map[string]*schema.Schema{
				"contact": &schema.Schema{
					Type:        schema.TypeString,
					Optional:    true,
					Description: "The contact information for the person.",
				},
				"name": &schema.Schema{
					Type:        schema.TypeString,
					Optional:    true,
					Description: "The name of the person who is responsible for the switch.",
				},
			},
		},
	}
}

func schemaDVPortSetting() map[string]*schema.Schema {
	// TBD
	return nil
}

func schemaDistributedVirtualSwitchHostMemberPnicBacking() *schema.Schema {
	// TODO maybe a set will fit better to avoid the mistake of putting a nic twice?
	return &schema.Schema{
		Type:     schema.TypeList,
		Optional: true,
		Elem:     &schema.Schema{Type: schema.TypeString},
	}
}

func schemaDistributedVirtualSwitchHostMemberConfigSpec() *schema.Schema {
	se := map[string]*schema.Schema{
		"max_proxy_switch_ports": &schema.Schema{
			Type:         schema.TypeInt,
			Optional:     true,
			Description:  "Maximum number of ports allowed in the HostProxySwitch.",
			ValidateFunc: validation.IntAtLeast(0),
		},
		// The host name should be enough to get a reference to it, which is what we need here
		"host_system_id": &schema.Schema{
			Type:        schema.TypeString,
			Optional:    true,
			Description: "The managed object ID of the host to search for NICs on.",
		},
		"operation": &schema.Schema{
			Type:         schema.TypeInt,
			Optional:     true,
			Description:  "Host member operation type.",
			ValidateFunc: validation.StringInSlice(configSpecOperationAllowedValues, false),
		},
		// DistributedVirtualSwitchHostMemberPnicBacking extends DistributedVirtualSwitchHostMemberBacking
		// which is a base class
		"backing": schemaDistributedVirtualSwitchHostMemberPnicBacking(),
	}
	mergeSchema(se, schemaDistributedVirtualSwitchKeyedOpaqueBlob())

	s := &schema.Schema{
		Type:     schema.TypeList,
		Optional: true,
		Elem: &schema.Resource{
			Schema: se,
		},
	}

	return s
}

func expandDistributedVirtualSwitchHostMemberConfigSpec(d *schema.ResourceData, refs []types.ManagedObjectReference) []types.DistributedVirtualSwitchHostMemberConfigSpec {
	// Configure the host and nic cards used as uplink for the DVS
	var hmc []types.DistributedVirtualSwitchHostMemberConfigSpec

	if hosts, ok := d.GetOk("host"); ok {
		for i, host := range hosts.([]interface{}) {
			hi := host.(map[string]interface{})

			for _, nic := range hi["backing"].([]interface{}) {
				// Get the physical NIC backing
				backing := new(types.DistributedVirtualSwitchHostMemberPnicBacking)
				backing.PnicSpec = append(backing.PnicSpec, types.DistributedVirtualSwitchHostMemberPnicSpec{
					PnicDevice: strings.TrimSpace(nic.(string)),
				})
				h := types.DistributedVirtualSwitchHostMemberConfigSpec{
					Host:      refs[i],
					Backing:   backing,
					Operation: "add", // Options: "add", "edit", "remove"
				}
				hmc = append(hmc, h)
			}
		}
	}

	return hmc
}

func schemaDvsHostInfrastructureTrafficResource() map[string]*schema.Schema {
	// TBD
	return nil
}

func schemaDVSPolicy() map[string]*schema.Schema {
	return map[string]*schema.Schema{
		"auto_pre_install_allowed": &schema.Schema{
			Type:        schema.TypeBool,
			Optional:    true,
			Description: "Whether downloading a new proxy VirtualSwitch module to the host is allowed to be automatically executed by the switch.",
		},
		"auto_upgrade_allowed": &schema.Schema{
			Type:        schema.TypeBool,
			Optional:    true,
			Description: "Whether upgrading of the switch is allowed to be automatically executed by the switch.",
		},
		"partial_upgrade_allowed": &schema.Schema{
			Type:        schema.TypeBool,
			Optional:    true,
			Description: "Whether to allow upgrading a switch when some of the hosts failed to install the needed module.",
		},
	}
}

func schemaDVSUplinkPortPolicy() map[string]*schema.Schema {
	// TBD
	return nil
}

func schemaDistributedVirtualSwitchKeyedOpaqueBlob() map[string]*schema.Schema {
	// TBD should be a map
	return nil
}

func schemaDVSConfiSpec() map[string]*schema.Schema {
	s := map[string]*schema.Schema{
		"config_version": &schema.Schema{
			Type:        schema.TypeString,
			Optional:    true,
			Description: "The version string of the configuration that this spec is trying to change. This property is ignored during switch creation.",
		},
		// nested to avoid having two "name" properties
		"contact": schemaDVSContactInfo(),
		"default_proxy_switch_max_num_ports": &schema.Schema{
			Type:         schema.TypeInt,
			Optional:     true,
			Description:  "The default host proxy switch maximum port number.",
			ValidateFunc: validation.IntAtLeast(0),
		},
		"description": &schema.Schema{
			Type:        schema.TypeString,
			Optional:    true,
			Description: "Set the description string of the switch.",
		},
		"extension_key": &schema.Schema{
			Type:        schema.TypeString,
			Optional:    true,
			Description: "The key of the extension registered by a remote server that controls the switch.",
		},
		"host": schemaDistributedVirtualSwitchHostMemberConfigSpec(),
		"name": &schema.Schema{
			Type:        schema.TypeString,
			Optional:    true,
			Description: "The name of the switch. Must be unique in the parent folder.",
		},
		"network_resource_control_version": &schema.Schema{
			Type:         schema.TypeString,
			Optional:     true,
			Description:  "Indicates the Network Resource Control APIs that are supported on the switch.",
			ValidateFunc: validation.StringInSlice(distributedVirtualSwitchNetworkResourceControlVersionAllowedValues, false),
		},
		"num_standalone_ports": &schema.Schema{
			Type:         schema.TypeInt,
			Optional:     true,
			Description:  "The number of standalone ports in the switch. Standalone ports are ports that do not belong to any portgroup.",
			ValidateFunc: validation.IntAtLeast(0),
		},
		"switch_ip_address": &schema.Schema{
			Type:         schema.TypeString,
			Optional:     true,
			Description:  "IP address for the switch, specified using IPv4 dot notation. IPv6 address is not supported for this property.",
			ValidateFunc: validation.StringInSlice(distributedVirtualSwitchNetworkResourceControlVersionAllowedValues, false),
		},
	}
	//mergeSchema(s, schemaDVSContactInfo())
	mergeSchema(s, schemaDVPortSetting())
	mergeSchema(s, schemaDvsHostInfrastructureTrafficResource())
	mergeSchema(s, schemaDVSPolicy())
	// XXX TBD uplinkPortgroup
	mergeSchema(s, schemaDVSUplinkPortPolicy())
	mergeSchema(s, schemaDistributedVirtualSwitchKeyedOpaqueBlob())

	return s
}

func expandDVSConfigSpec(d *schema.ResourceData, refs []types.ManagedObjectReference) *types.DVSConfigSpec {
	name := d.Get("name").(string)

	obj := &types.DVSConfigSpec{
		Name: name,
		Host: expandDistributedVirtualSwitchHostMemberConfigSpec(d, refs),
	}

	if v, ok := d.GetOkExists("description"); ok {
		obj.Description = v.(string)
	}

	if v, ok := d.GetOkExists("num_standalone_ports"); ok {
		obj.NumStandalonePorts = v.(int32)
	}

	//if v, ok := d.GetOkExists("default_proxy_switch_max_num_ports"); ok {
	//obj.NumStandalonePorts = v.(int32)
	//}

	return obj
}

func flattenDVSConfigSpec(d *schema.ResourceData, obj *mo.DistributedVirtualSwitch) error {
	config := obj.Config.GetDVSConfigInfo()
	d.Set("name", config.Name)
	d.Set("description", config.Description)
	d.Set("num_standalone_ports", config.NumStandalonePorts)
	//d.Set("default_proxy_switch_max_num_ports", config.DefaultProxySwitchMaxNumPorts)

	return nil
}
