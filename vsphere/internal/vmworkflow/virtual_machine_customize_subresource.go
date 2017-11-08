package vmworkflow

import (
	"github.com/hashicorp/terraform/helper/schema"
)

// VirtualMachineCustomizeSchema returns the schema for VM customization.
func VirtualMachineCustomizeSchema() map[string]*schema.Schema {
	return map[string]*schema.Schema{
		// CustomizationGlobalIPSettings
		"dns_server_list": {
			Type:        schema.TypeList,
			Optional:    true,
			Description: "The list of DNS servers for a virtual network adapter with a static IP address.",
			Elem:        &schema.Schema{Type: schema.TypeString},
		},
		"dns_suffix_list": {
			Type:        schema.TypeList,
			Optional:    true,
			Description: "A list of DNS search domains to add to the DNS configuration on the virtual machine.",
			Elem:        &schema.Schema{Type: schema.TypeString},
		},

		// CustomizationLinuxPrep
		"linux_options": {
			Type:          schema.TypeList,
			Optional:      true,
			MaxItems:      1,
			ConflictsWith: []string{"windows_options", "windows_sysprep_text"},
			Description:   "A list of configuration options specific to Linux virtual machines.",
			Elem: &schema.Resource{Schema: map[string]*schema.Schema{
				"domain": {
					Type:        schema.TypeString,
					Required:    true,
					Description: "The FQDN for this virtual machine.",
				},
				"host_name": {
					Type:        schema.TypeString,
					Required:    true,
					Description: "The host name for this virtual machine.",
				},
				"hw_clock_utc": {
					Type:        schema.TypeBool,
					Optional:    true,
					Description: "Specifies whether or not the hardware clock should be in UTC or not.",
				},
				"time_zone": {
					Type:        schema.TypeString,
					Optional:    true,
					Description: "Customize the time zone on the VM. This should be a time zone-style entry, like America/Los_Angeles.",
				},
			}},
		},

		// CustomizationSysprep
		"windows_options": {
			Type:          schema.TypeList,
			Optional:      true,
			MaxItems:      1,
			ConflictsWith: []string{"linux_options", "windows_sysprep_text"},
			Description:   "A list of configuration options specific to Windows virtual machines.",
			Elem: &schema.Resource{Schema: map[string]*schema.Schema{
				// CustomizationGuiRunOnce
				"run_once_command_list": {
					Type:        schema.TypeList,
					Optional:    true,
					Description: "A list of commands to run at first user logon, after guest customization.",
					Elem:        &schema.Schema{Type: schema.TypeString},
				},
				// CustomizationGuiUnattended
				"auto_logon": {
					Type:        schema.TypeBool,
					Optional:    true,
					Description: "Specifies whether or not the VM automatically logs on as Administrator.",
				},
				"auto_logon_count": {
					Type:        schema.TypeInt,
					Optional:    true,
					Default:     1,
					Description: "Specifies how many times the VM should auto-logon the Administrator account when auto_logon is true.",
				},
				"admin_password": {
					Type:        schema.TypeString,
					Optional:    true,
					Sensitive:   true,
					Description: "The new administrator password for this virtual machine.",
				},
				"time_zone": {
					Type:        schema.TypeInt,
					Optional:    true,
					Default:     85,
					Description: "The new time zone for the virtual machine. This is a sysprep-dictated timezone code.",
				},

				// CustomizationIdentification
				"domain_admin_user": {
					Type:          schema.TypeString,
					Optional:      true,
					ConflictsWith: []string{"join_workgroup"},
					Description:   "The user account of the domain administrator used to join this virtual machine to the domain.",
				},
				"domain_admin_password": {
					Type:          schema.TypeString,
					Optional:      true,
					Sensitive:     true,
					ConflictsWith: []string{"join_workgroup"},
					Description:   "The password of the domain administrator used to join this virtual machine to the domain.",
				},
				"join_domain": {
					Type:          schema.TypeString,
					Optional:      true,
					ConflictsWith: []string{"join_workgroup"},
					Description:   "The domain that the virtual machine should join.",
				},
				"join_workgroup": {
					Type:          schema.TypeString,
					Optional:      true,
					ConflictsWith: []string{"join_domain"},
					Description:   "The workgroup that the virtual machine should join.",
				},

				// CustomizationUserData
				"computer_name": {
					Type:        schema.TypeString,
					Required:    true,
					Description: "The host name for this virtual machine.",
				},
				"full_name": {
					Type:        schema.TypeString,
					Optional:    true,
					Default:     "Administrator",
					Description: "The full name of the user of this virtual machine.",
				},
				"organization_name": {
					Type:        schema.TypeString,
					Optional:    true,
					Default:     "Managed by Terraform",
					Description: "The organization name this virtual machine is being installed for.",
				},
				"product_key": {
					Type:        schema.TypeString,
					Optional:    true,
					Description: "The product key for this virtual machine.",
				},
			}},
		},

		// CustomizationSysprepText
		"windows_sysprep_text": {
			Type:          schema.TypeString,
			Optional:      true,
			ConflictsWith: []string{"linux_options", "windows_options"},
			Description:   "Use this option to specify a windows sysprep file directly.",
		},

		// CustomizationIPSettings
		"network_interface": {
			Type:        schema.TypeList,
			Optional:    true,
			Description: "A specification of network interface configuration options.",
			Elem: &schema.Resource{Schema: map[string]*schema.Schema{
				"dns_server_list": {
					Type:        schema.TypeList,
					Optional:    true,
					Description: "Network-interface specific DNS settings for Windows operating systems. Ignored on Linux.",
					Elem:        &schema.Schema{Type: schema.TypeString},
				},
				"dns_domain": {
					Type:        schema.TypeList,
					Optional:    true,
					Description: "A list of DNS search domains to add to the DNS configuration on the virtual machine.",
					Elem:        &schema.Schema{Type: schema.TypeString},
				},
				"ipv4_address": {
					Type:        schema.TypeString,
					Optional:    true,
					Description: "The IPv4 address assigned to this network adapter. If left blank, DHCP is used.",
				},
				"ipv4_netmask": {
					Type:        schema.TypeInt,
					Optional:    true,
					Description: "The IPv4 CIDR netmask for the supplied IP address. Ignored if DHCP is selected.",
				},
				"ipv6_address": {
					Type:        schema.TypeString,
					Optional:    true,
					Description: "The IPv6 address assigned to this network adapter. If left blank, default auto-configuration is used.",
				},
				"ipv6_netmask": {
					Type:        schema.TypeInt,
					Optional:    true,
					Description: "The IPv6 CIDR netmask for the supplied IP address. Ignored if auto-configuration is selected.",
				},
			}},
		},

		// Base-level settings
		"ipv4_gateway": {
			Type:        schema.TypeString,
			Optional:    true,
			Description: "The IPv4 default gateway when using network_interface customization on the virtual machine. This address must be local to a static IPv4 address configured in an interface sub-resource.",
		},
		"ipv6_gateway": {
			Type:        schema.TypeString,
			Optional:    true,
			Description: "The IPv6 default gateway when using network_interface customization on the virtual machine. This address must be local to a static IPv4 address configured in an interface sub-resource.",
		},
	}
}
