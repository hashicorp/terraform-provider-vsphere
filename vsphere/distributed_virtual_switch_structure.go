package vsphere

import (
	"github.com/hashicorp/terraform/helper/schema"
	"github.com/hashicorp/terraform/helper/validation"
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

func schemaDVSContactInfo() map[string]*schema.Schema {
	return map[string]*schema.Schema{
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
	}
}

func schemaDVPortSetting() map[string]*schema.Schema {
	// TBD
}

func schemaDistributedVirtualSwitchHostMemberConfigSpec() map[string]*schema.Schema {
	s := map[string]*schema.Schema{
		"maxProxySwitchPorts": &schema.Schema{
			Type:        schema.TypeInt,
			Optional:    true,
			Description: "Maximum number of ports allowed in the HostProxySwitch.",
			Validation:  validation.IntAtLeast(0),
		},
		"operation": &schema.Schema{
			Type:        schema.TypeInt,
			Optional:    true,
			Description: "Host member operation type.",
			Validation:  validation.StringInSlice(configSpecOperationAllowedValues, false),
		},
	}
	mergeSchema(s, schemaDistributedVirtualSwitchHostMemberBacking())
	// XXX TBD host
	mergeSchema(s, schemaDistributedVirtualSwitchKeyedOpaqueBlob())
}

func schemaDvsHostInfrastructureTrafficResource() map[string]*schema.Schema {
	// TBD
}

func schemaDVSPolicy() map[string]*schema.Schema {
	return map[string]*schema.Schema{
		"autoPreInstallAllowed": &schema.Schema{
			Type:        schema.TypeBool,
			Optional:    true,
			Description: "Whether downloading a new proxy VirtualSwitch module to the host is allowed to be automatically executed by the switch.",
		},
		"autoUpgradeAllowed": &schema.Schema{
			Type:        schema.TypeBool,
			Optional:    true,
			Description: "Whether upgrading of the switch is allowed to be automatically executed by the switch.",
		},
		"partialUpgradeAllowed": &schema.Schema{
			Type:        schema.TypeBool,
			Optional:    true,
			Description: "Whether to allow upgrading a switch when some of the hosts failed to install the needed module.",
		},
	}
}

func schemaDVSUplinkPortPolicy() map[string]*schema.Schema {
	// TBD
}

func schemaDistributedVirtualSwitchKeyedOpaqueBlob() map[string]*schema.Schema {
	// TBD should be a map
}

func schemaDVSConfiSpec() map[string]*schema.Schema {
	s := map[string]*schema.Schema{
		"configVersion": &schema.Schema{
			Type:        schema.TypeString,
			Optional:    true,
			Description: "The version string of the configuration that this spec is trying to change. This property is ignored during switch creation.",
		},
		"defaultProxySwitchMaxNumPorts": &schema.Schema{
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
		"extensionKey": &schema.Schema{
			Type:        schema.TypeString,
			Optional:    true,
			Description: "The key of the extension registered by a remote server that controls the switch.",
		},
		"name": &schema.Schema{
			Type:        schema.TypeString,
			Optional:    true,
			Description: "The name of the switch. Must be unique in the parent folder.",
		},
		"networkResourceControlVersion": &schema.Schema{
			Type:         schema.TypeString,
			Optional:     true,
			Description:  "Indicates the Network Resource Control APIs that are supported on the switch.",
			ValidateFunc: validation.StringInSlice(distributedVirtualSwitchNetworkResourceControlVersionAllowedValues, false),
		},
		"numStandalonePorts": &schema.Schema{
			Type:         schema.TypeInt,
			Optional:     true,
			Description:  "The number of standalone ports in the switch. Standalone ports are ports that do not belong to any portgroup.",
			ValidateFunc: validation.IntAtLeast(0),
		},
		"switchIpAddress": &schema.Schema{
			Type:         schema.TypeString,
			Optional:     true,
			Description:  "IP address for the switch, specified using IPv4 dot notation. IPv6 address is not supported for this property.",
			ValidateFunc: validation.StringInSlice(distributedVirtualSwitchNetworkResourceControlVersionAllowedValues, false),
		},
	}
	mergeSchema(s, schemaDVSContactInfo())
	mergeSchema(s, schemaDVPortSetting())
	mergeSchema(s, schemaDistributedVirtualSwitchHostMemberConfigSpec())
	mergeSchema(s, schemaDvsHostInfrastructureTrafficResource())
	mergeSchema(s, schemaDVSPolicy())
	// XXX TBD uplinkPortgroup
	mergeSchema(s, schemaDVSUplinkPortPolicy())
	mergeSchema(s, schemaDistributedVirtualSwitchKeyedOpaqueBlob())
}
