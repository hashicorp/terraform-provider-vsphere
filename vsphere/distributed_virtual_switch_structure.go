package vsphere

import (
	"log"
	"strings"

	"github.com/hashicorp/terraform/helper/schema"
	"github.com/hashicorp/terraform/helper/validation"
	"github.com/vmware/govmomi"
	"github.com/vmware/govmomi/find"
	"github.com/vmware/govmomi/object"
	"github.com/vmware/govmomi/vim25/mo"
	"github.com/vmware/govmomi/vim25/types"
	"golang.org/x/net/context"
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

/*var distributedVirtualSwitchHostInfrastructureTrafficClass = []string{
	string(types.DistributedVirtualSwitchHostInfrastructureTrafficClassManagement),
	string(types.DistributedVirtualSwitchHostInfrastructureTrafficClassFaultTolerance),
	string(types.DistributedVirtualSwitchHostInfrastructureTrafficClassVmotion),
	string(types.DistributedVirtualSwitchHostInfrastructureTrafficClassVirtualMachine),
	string(types.DistributedVirtualSwitchHostInfrastructureTrafficClassISCSI),
	string(types.DistributedVirtualSwitchHostInfrastructureTrafficClassNfs),
	string(types.DistributedVirtualSwitchHostInfrastructureTrafficClassHbr),
	string(types.DistributedVirtualSwitchHostInfrastructureTrafficClassVsan),
	string(types.DistributedVirtualSwitchHostInfrastructureTrafficClassVdp),
}*/

func schemaDVSContactInfo() map[string]*schema.Schema {
	return map[string]*schema.Schema{
		"contact": &schema.Schema{
			Type:        schema.TypeString,
			Optional:    true,
			Description: "The contact information for the person.",
		},
		"contact_name": &schema.Schema{
			Type:        schema.TypeString,
			Optional:    true,
			Description: "The name of the person who is responsible for the switch.",
		},
	}
}

func expandDVSContactInfo(d *schema.ResourceData) *types.DVSContactInfo {
	dci := &types.DVSContactInfo{}
	if v, ok := d.GetOkExists("contact"); ok {
		dci.Contact = v.(string)
	}

	if v, ok := d.GetOkExists("contact_name"); ok {
		dci.Name = v.(string)
	}
	return dci
}

func flattenDVSContactInfo(d *schema.ResourceData, obj *mo.DistributedVirtualSwitch) {
	config := obj.Config.GetDVSConfigInfo()
	d.Set("contact", config.Contact.Contact)
	d.Set("contact_name", config.Contact.Name)
}

/*func schemaDVPortSetting() map[string]*schema.Schema {
	// TBD
	return nil
}*/

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
		"backing":                schemaDistributedVirtualSwitchHostMemberPnicBacking(),
		"vendor_specific_config": schemaDistributedVirtualSwitchKeyedOpaqueBlob(),
	}

	s := &schema.Schema{
		Type:     schema.TypeList,
		Optional: true,
		Elem: &schema.Resource{
			Schema: se,
		},
	}

	return s
}

func expandDistributedVirtualSwitchHostMemberConfigSpec(client *govmomi.Client, d *schema.ResourceData, dvs *mo.DistributedVirtualSwitch, refs map[string]types.ManagedObjectReference) []types.DistributedVirtualSwitchHostMemberConfigSpec {
	// Configure the host and nic cards used as uplink for the DVS
	var hmc []types.DistributedVirtualSwitchHostMemberConfigSpec

	var hosts []interface{}
	if h, ok := d.GetOk("host"); ok {
		hosts = h.([]interface{})
	}
	// If the DVS exist we go through all the hosts and see which ones
	// we have to delete or modify
	if dvs != nil {
		config := dvs.Config.GetDVSConfigInfo()
		for _, h := range config.Host {
			if host := isHostPartOfDVS(hosts, refs, h.Config.Host); host != nil {
				// Edit
				backing := new(types.DistributedVirtualSwitchHostMemberPnicBacking)
				for _, nic := range host["backing"].([]interface{}) {
					backing.PnicSpec = append(backing.PnicSpec, types.DistributedVirtualSwitchHostMemberPnicSpec{
						PnicDevice: strings.TrimSpace(nic.(string)),
					})
				}
				hcs := types.DistributedVirtualSwitchHostMemberConfigSpec{
					Host:      *h.Config.Host,
					Backing:   backing,
					Operation: "edit", // Options: "add", "edit", "remove"
				}
				hmc = append(hmc, hcs)

				// We take it out from the refs, on the last pass we consider whatever
				// is left as to be added
				delete(refs, host["host_system_id"].(string))
			} else {
				log.Print("[TRACE] Removing host from uplinks")
				// Remove
				backing := new(types.DistributedVirtualSwitchHostMemberPnicBacking)
				finder := find.NewFinder(client.Client, false)

				ds, err := finder.ObjectReference(context.TODO(), *h.Config.Host)
				if err != nil {
					continue
				}
				dso := ds.(*object.HostSystem)
				var mh mo.HostSystem
				err = dso.Properties(context.TODO(), ds.Reference(), []string{"config"}, &mh)
				if err != nil {
					continue
				}

				for _, ps := range mh.Config.Network.ProxySwitch {
					if ps.DvsUuid == d.Id() {
						for _, nic := range ps.Pnic {
							backing.PnicSpec = append(backing.PnicSpec, types.DistributedVirtualSwitchHostMemberPnicSpec{
								PnicDevice: string(nic),
							})
						}
						hcs := types.DistributedVirtualSwitchHostMemberConfigSpec{
							Host:      *h.Config.Host,
							Backing:   backing,
							Operation: "remove", // Options: "add", "edit", "remove"
						}
						hmc = append(hmc, hcs)
					}
				}
			}
		}
	}

	if hosts != nil {
		// Add whatever is left
		for _, host := range hosts {
			hi := host.(map[string]interface{})
			if val, ok := refs[hi["host_system_id"].(string)]; ok {
				backing := new(types.DistributedVirtualSwitchHostMemberPnicBacking)
				for _, nic := range hi["backing"].([]interface{}) {
					backing.PnicSpec = append(backing.PnicSpec, types.DistributedVirtualSwitchHostMemberPnicSpec{
						PnicDevice: strings.TrimSpace(nic.(string)),
					})
				}
				h := types.DistributedVirtualSwitchHostMemberConfigSpec{
					Host:      val,
					Backing:   backing,
					Operation: "add", // Options: "add", "edit", "remove"
				}
				hmc = append(hmc, h)
			}
		}
	}

	return hmc
}

func flattenDistributedVirtualSwitchHostMemberConfigSpec(client *govmomi.Client, d *schema.ResourceData, obj *mo.DistributedVirtualSwitch) {
	log.Printf("[TRACE] Flattening DistributedVirtualSwitchHostMemberConfigSpec %v", d.Get("host"))
	config := obj.Config.GetDVSConfigInfo()
	var hosts []interface{}
	for _, host := range config.Host {
		log.Print("[TRACE] One host")
		hm := make(map[string]interface{})

		hm["host_system_id"] = host.Config.Host

		backing := []string{}

		finder := find.NewFinder(client.Client, false)

		ds, err := finder.ObjectReference(context.TODO(), *host.Config.Host)
		if err != nil {
			continue
		}
		dso := ds.(*object.HostSystem)
		var mh mo.HostSystem
		err = dso.Properties(context.TODO(), ds.Reference(), []string{"config"}, &mh)
		if err != nil {
			continue
		}

		for _, ps := range mh.Config.Network.ProxySwitch {
			log.Print("[TRACE] One proxy switch")
			if ps.DvsUuid == d.Id() {
				log.Print("[TRACE] Found the proxy switch for this DVS")
				for _, nic := range ps.Pnic {
					log.Printf("[TRACE] Pnic %s", nic)
					sn := strings.Split(nic, "-")
					backing = append(backing, sn[len(sn)-1])
				}
			}
		}
		if len(backing) > 0 {
			hm["backing"] = backing
		}
		hosts = append(hosts, hm)
		log.Printf("[TRACE] Host after flattening %v", hosts)
	}
	d.Set("host", hosts)
}

/*func schemaDvsHostInfrastructureTrafficResource() *schema.Schema {
	return &schema.Schema{
		Type:     schema.TypeList,
		Optional: true,
		Elem: &schema.Resource{
			Schema: map[string]*schema.Schema{
				"description": &schema.Schema{
					Type:        schema.TypeString,
					Optional:    true,
					Description: "The description of the host infrastructure resource. This property is ignored for update operation.",
				},
				"key": &schema.Schema{
					Type:         schema.TypeString,
					Optional:     true,
					Description:  "The key of the host infrastructure resource. Possible value can be of DistributedVirtualSwitchHostInfrastructureTrafficClass.",
					ValidateFunc: validation.StringInSlice(distributedVirtualSwitchHostInfrastructureTrafficClass, false),
				},
				//"allocationInfo": TBD
			},
		},
	}
}*/

/*func schemaDVSPolicy() map[string]*schema.Schema {
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
}*/

/*func schemaDVSUplinkPortPolicy() map[string]*schema.Schema {
	// TBD
	return nil
}*/

func schemaDistributedVirtualSwitchKeyedOpaqueBlob() *schema.Schema {
	return &schema.Schema{
		Type:     schema.TypeList,
		Optional: true,
		Elem: &schema.Resource{
			Schema: map[string]*schema.Schema{
				"key": &schema.Schema{
					Type:        schema.TypeString,
					Optional:    true,
					Description: "A key that identifies the opaque binary blob.",
				},
				"opaque_data": &schema.Schema{
					Type:        schema.TypeString,
					Optional:    true,
					Description: "The opaque data. It is recommended that base64 encoding be used for binary data.",
				},
			},
		},
	}
}

func schemaDVSConfiSpec() map[string]*schema.Schema {
	s := map[string]*schema.Schema{
		"config_version": &schema.Schema{
			Type:        schema.TypeString,
			Computed:    true,
			Description: "The version string of the configuration that this spec is trying to change. This property is ignored during switch creation.",
		},
		"default_proxy_switch_max_num_ports": &schema.Schema{
			Type:         schema.TypeInt,
			Optional:     true,
			Default:      512,
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
		//"infrastructure_traffic_resource_config": schemaDvsHostInfrastructureTrafficResource(),
		"name": &schema.Schema{
			Type:        schema.TypeString,
			Optional:    true,
			Description: "The name of the switch. Must be unique in the parent folder.",
		},
		"network_resource_control_version": &schema.Schema{
			Type:         schema.TypeString,
			Optional:     true,
			Default:      "version2",
			Description:  "Indicates the Network Resource Control APIs that are supported on the switch.",
			ValidateFunc: validation.StringInSlice(distributedVirtualSwitchNetworkResourceControlVersionAllowedValues, false),
		},
		"num_standalone_ports": &schema.Schema{
			Type:         schema.TypeInt,
			Optional:     true,
			Default:      512,
			Description:  "The number of standalone ports in the switch. Standalone ports are ports that do not belong to any portgroup.",
			ValidateFunc: validation.IntAtLeast(0),
		},
		"switch_ip_address": &schema.Schema{
			Type:        schema.TypeString,
			Optional:    true,
			Description: "IP address for the switch, specified using IPv4 dot notation. IPv6 address is not supported for this property.",
		},
		"vendor_specific_config": schemaDistributedVirtualSwitchKeyedOpaqueBlob(),
	}
	mergeSchema(s, schemaDVSContactInfo())
	//mergeSchema(s, schemaDVPortSetting())
	//mergeSchema(s, schemaDVSPolicy())
	// XXX TBD uplinkPortgroup
	//mergeSchema(s, schemaDVSUplinkPortPolicy())

	return s
}

func expandDVSConfigSpec(client *govmomi.Client, d *schema.ResourceData, dvs *mo.DistributedVirtualSwitch, refs map[string]types.ManagedObjectReference) *types.DVSConfigSpec {
	obj := &types.DVSConfigSpec{}

	obj.Name = d.Get("name").(string)

	if v, ok := d.GetOkExists("network_resource_control_version"); ok {
		obj.NetworkResourceControlVersion = v.(string)
	}

	if v, ok := d.GetOkExists("config_version"); ok {
		obj.ConfigVersion = v.(string)
	}

	obj.Contact = expandDVSContactInfo(d)

	if v, ok := d.GetOkExists("default_proxy_switch_max_num_ports"); ok {
		obj.NumStandalonePorts = int32(v.(int))
	}

	if v, ok := d.GetOkExists("description"); ok {
		obj.Description = v.(string)
	}

	if v, ok := d.GetOkExists("extension_key"); ok {
		obj.ExtensionKey = v.(string)
	}

	// Always expand since even when removing we will need to mention hosts and nics
	obj.Host = expandDistributedVirtualSwitchHostMemberConfigSpec(client, d, dvs, refs)

	if v, ok := d.GetOkExists("num_standalone_ports"); ok {
		obj.NumStandalonePorts = int32(v.(int))
	}

	if v, ok := d.GetOkExists("switch_ip_addess"); ok {
		obj.NumStandalonePorts = int32(v.(int))
	}

	return obj
}

func flattenDVSConfigSpec(client *govmomi.Client, d *schema.ResourceData, obj *mo.DistributedVirtualSwitch) error {
	config := obj.Config.GetDVSConfigInfo()
	d.Set("config_version", config.ConfigVersion)
	d.Set("description", config.Description)
	d.Set("extension_key", config.ExtensionKey)
	d.Set("name", config.Name)
	d.Set("network_resource_control_version", config.NetworkResourceControlVersion)
	d.Set("description", config.Description)
	d.Set("contact", config.Contact.Contact)
	d.Set("contact_name", config.Contact.Name)
	d.Set("num_standalone_ports", config.NumStandalonePorts)
	d.Set("default_proxy_switch_max_num_ports", config.DefaultProxySwitchMaxNumPorts)
	d.Set("switch_ip_address", config.SwitchIpAddress)
	flattenDVSContactInfo(d, obj)
	flattenDistributedVirtualSwitchHostMemberConfigSpec(client, d, obj)

	return nil
}
