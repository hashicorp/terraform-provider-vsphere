package vsphere

import (
	"context"
	"strings"
	"fmt"
	"log"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func resourceVsphereMigratedNic() *schema.Resource {
	return &schema.Resource{
		Create: resourceVsphereMigratedNicUpdate,
		Read:   resourceVsphereMigratedNicRead,
		Update: resourceVsphereMigratedNicUpdate,
		Delete: resourceVsphereMigratedNicDelete,
		Importer: &schema.ResourceImporter{
			State: resourceVSphereNicImport,
		},
		Schema: vMigratedNicSchema(),
	}
}

func vMigratedNicSchema() map[string]*schema.Schema {
	base := BaseVMKernelSchema()
	base["vnic_id"] = &schema.Schema{
		Type:        schema.TypeString,
		Required:    true,
		Description: "Resource ID of vnic to migrate",
		ForceNew:    true,
	}

	return base
}

func resourceVsphereMigratedNicRead(d *schema.ResourceData, meta interface{}) error {
	log.Printf("[DEBUG] starting resource_vnic")
	ctx := context.TODO()
	client := meta.(*Client).vimClient

	hostID, nicID := splitHostIDMigratedNicID(d)

	vnic, err := getVnicFromHost(ctx, client, hostID, nicID)
	if err != nil {
		log.Printf("[DEBUG] Nic (%s) not found. Probably deleted.", nicID)
		d.SetId("")
		return nil
	}
	d.SetId(fmt.Sprintf("%s_%s", hostID, nicID))

	_ = d.Set("netstack", vnic.Spec.NetStackInstanceKey)
	_ = d.Set("portgroup", vnic.Portgroup)
	if vnic.Spec.DistributedVirtualPort != nil {
		_ = d.Set("distributed_switch_port", vnic.Spec.DistributedVirtualPort.SwitchUuid)
		_ = d.Set("distributed_port_group", vnic.Spec.DistributedVirtualPort.PortgroupKey)
	}
	_ = d.Set("mtu", vnic.Spec.Mtu)
	_ = d.Set("mac", vnic.Spec.Mac)

	// Do we have any ipv4 config ?
	// IpAddress will be an empty string if ipv4 is off
	if vnic.Spec.Ip.IpAddress != "" {
		// if DHCP is true then we should ignore whatever addresses are set here.
		ipv4dict := make(map[string]interface{})
		ipv4dict["dhcp"] = vnic.Spec.Ip.Dhcp
		if !vnic.Spec.Ip.Dhcp {
			ipv4dict["ip"] = vnic.Spec.Ip.IpAddress
			ipv4dict["netmask"] = vnic.Spec.Ip.SubnetMask
			if vnic.Spec.IpRouteSpec != nil {
				ipv4dict["gw"] = vnic.Spec.IpRouteSpec.IpRouteConfig.GetHostIpRouteConfig().DefaultGateway
			}
		}
		err = d.Set("ipv4", []map[string]interface{}{ipv4dict})
		if err != nil {
			return err
		}
	}

	// Do we have any ipv6 config ?
	// IpV6Config will be nil if ipv6 is off
	if vnic.Spec.Ip.IpV6Config != nil {
		ipv6dict := map[string]interface{}{
			"dhcp":       *vnic.Spec.Ip.IpV6Config.DhcpV6Enabled,
			"autoconfig": *vnic.Spec.Ip.IpV6Config.AutoConfigurationEnabled,
		}

		// First we need to filter out addresses that were configured via dhcp or autoconfig
		// or link local or any other mechanism
		addrList := make([]string, 0)
		for _, addr := range vnic.Spec.Ip.IpV6Config.IpV6Address {
			if addr.Origin == "manual" {
				addrList = append(addrList, fmt.Sprintf("%s/%d", addr.IpAddress, addr.PrefixLength))
			}
		}
		if (len(addrList) == 0) && !*vnic.Spec.Ip.IpV6Config.DhcpV6Enabled && !*vnic.Spec.Ip.IpV6Config.AutoConfigurationEnabled {
			_ = d.Set("ipv6", nil)
		} else {
			ipv6dict["addresses"] = addrList

			if vnic.Spec.IpRouteSpec != nil {
				ipv6dict["gw"] = vnic.Spec.IpRouteSpec.IpRouteConfig.GetHostIpRouteConfig().IpV6DefaultGateway
			} else if _, ok := d.GetOk("ipv6.0.gw"); ok {
				// There is a gw set in the config, but none set on the Host.
				ipv6dict["gw"] = ""
			}
			err = d.Set("ipv6", []map[string]interface{}{ipv6dict})
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func resourceVsphereMigratedNicUpdate(d *schema.ResourceData, meta interface{}) error {
	for _, k := range []string{
		"portgroup", "distributed_switch_port", "distributed_port_group",
		"mac", "mtu", "ipv4", "ipv6", "netstack"} {
		if d.HasChange(k) {
			_, err := updateVMigratedNic(d, meta)
			if err != nil {
				return err
			}
			break
		}
	}
	return resourceVsphereMigratedNicRead(d, meta)
}

func updateVMigratedNic(d *schema.ResourceData, meta interface{}) (string, error) {
	client := meta.(*Client).vimClient
	hostID, nicID := splitHostIDMigratedNicID(d)
	ctx := context.TODO()

	nic, err := getNicSpecFromSchema(d)
	if err != nil {
		return "", err
	}

	hns, err := getHostNetworkSystem(client, hostID)
	if err != nil {
		return "", err
	}

	err = hns.UpdateVirtualNic(ctx, nicID, *nic)
	if err != nil {
		return "", err
	}

	return nicID, nil
}

func splitHostIDMigratedNicID(d *schema.ResourceData) (string, string) {
	id := d.Get("vnic_id").(string)
	idParts := strings.Split(id, "_")
	log.Printf("vnic_id=%s", id)
	log.Printf("idParts=%v", idParts)
	return idParts[0], idParts[1]
}

func resourceVsphereMigratedNicDelete(d *schema.ResourceData, meta interface{}) error {
	d.SetId("")
	return nil
}
