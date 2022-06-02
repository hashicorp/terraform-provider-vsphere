package vsphere

import (
	"context"
	"fmt"
	"log"
	"strconv"
	"strings"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/vmware/govmomi/vim25/types"
)

func resourceVsphereMigratedNic() *schema.Resource {
	return &schema.Resource{
		Create: resourceVsphereMigratedNicUpdate,
		Read:   resourceVsphereMigratedNicRead,
		Update: resourceVsphereMigratedNicUpdate,
		Delete: resourceVsphereMigratedNicDelete,
		Importer: &schema.ResourceImporter{
			State: resourceVSphereMigratedNicImport,
		},
		Schema: vMigratedNicSchema(),
	}
}

func vMigratedNicSchema() map[string]*schema.Schema {
	base := BaseVMKernelSchema()
	base["vnic_id"] = &schema.Schema{
		Type:        schema.TypeString,
		Required:    true,
		Description: "Resource ID of vnic o migrate",
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
		log.Printf("[DEBUG] MigratedNic (%s) not found. Probably deleted.", nicID)
		d.SetId("")
		return nil
	}

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

func resourceVsphereMigratedNicCreate(d *schema.ResourceData, meta interface{}) error {
	nicID, err := createVMigratedNic(d, meta)
	if err != nil {
		return err
	}
	log.Printf("[DEBUG] Created NIC with ID: %s", nicID)
	hostID := d.Get("host")
	tfMigratedNicID := fmt.Sprintf("%s_%s", hostID, nicID)
	d.SetId(tfMigratedNicID)
	return resourceVsphereMigratedNicRead(d, meta)
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

func resourceVsphereMigratedNicDelete(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*Client).vimClient
	hostID, nicID := splitHostIDMigratedNicID(d)

	err := removeVnic(client, hostID, nicID)
	if err != nil {
		return err
	}
	return resourceVsphereMigratedNicRead(d, meta)
}

func resourceVSphereMigratedNicImport(d *schema.ResourceData, _ interface{}) ([]*schema.ResourceData, error) {
	hostID, _ := splitHostIDMigratedNicID(d)

	err := d.Set("host", hostID)
	if err != nil {
		return []*schema.ResourceData{}, err
	}

	return []*schema.ResourceData{d}, nil
}

func updateVMigratedNic(d *schema.ResourceData, meta interface{}) (string, error) {
	client := meta.(*Client).vimClient
	hostID, nicID := splitHostIDMigratedNicID(d)
	ctx := context.TODO()

	nic, err := getMigratedNicSpecFromSchema(d)
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

func createVMigratedNic(d *schema.ResourceData, meta interface{}) (string, error) {
	client := meta.(*Client).vimClient
	ctx := context.TODO()

	nic, err := getMigratedNicSpecFromSchema(d)
	if err != nil {
		return "", err
	}

	hostID := d.Get("host").(string)
	hns, err := getHostNetworkSystem(client, hostID)
	if err != nil {
		return "", err
	}

	portgroup := d.Get("portgroup").(string)
	nicID, err := hns.AddVirtualNic(ctx, portgroup, *nic)
	if err != nil {
		return "", err
	}
	d.SetId(fmt.Sprintf("%s_%s", hostID, nicID))
	return nicID, nil
}

func getMigratedNicSpecFromSchema(d *schema.ResourceData) (*types.HostVirtualNicSpec, error) {
	portgroup := d.Get("portgroup").(string)
	dvp := d.Get("distributed_switch_port").(string)
	dpg := d.Get("distributed_port_group").(string)
	mac := d.Get("mac").(string)
	mtu := int32(d.Get("mtu").(int))

	if portgroup != "" && dvp != "" {
		return nil, fmt.Errorf("portgroup and distributed_switch_port settings are mutually exclusive")
	}

	var dvpPortConnection *types.DistributedVirtualSwitchPortConnection
	if portgroup != "" {
		dvpPortConnection = nil
	} else {
		dvpPortConnection = &types.DistributedVirtualSwitchPortConnection{
			SwitchUuid:   dvp,
			PortgroupKey: dpg,
		}
	}

	ipConfig := &types.HostIpConfig{}
	routeConfig := &types.HostIpRouteConfig{} // routeConfig := r.IpRouteConfig.GetHostIpRouteConfig()
	if ipv4, ok := d.GetOk("ipv4.0"); ok {
		ipv4Config := ipv4.(map[string]interface{})

		dhcp := ipv4Config["dhcp"].(bool)
		ipv4Address := ipv4Config["ip"].(string)
		ipv4Netmask := ipv4Config["netmask"].(string)
		ipv4Gateway := ipv4Config["gw"].(string)

		if dhcp {
			ipConfig.Dhcp = dhcp
		} else if ipv4Address != "" && ipv4Netmask != "" {
			ipConfig.IpAddress = ipv4Address
			ipConfig.SubnetMask = ipv4Netmask
			routeConfig.DefaultGateway = ipv4Gateway
		}
	}

	if ipv6, ok := d.GetOk("ipv6.0"); ok {
		ipv6Spec := &types.HostIpConfigIpV6AddressConfiguration{}
		ipv6Config := ipv6.(map[string]interface{})

		dhcpv6 := ipv6Config["dhcp"].(bool)
		autoconfig := ipv6Config["autoconfig"].(bool)
		// ipv6addrs := ipv6Config["addresses"].([]interface{})
		ipv6Gateway := ipv6Config["gw"].(string)
		ipv6Spec.DhcpV6Enabled = &dhcpv6
		ipv6Spec.AutoConfigurationEnabled = &autoconfig

		oldAddrsIntf, newAddrsIntf := d.GetChange("ipv6.0.addresses")
		oldAddrs := oldAddrsIntf.([]interface{})
		newAddrs := newAddrsIntf.([]interface{})
		addAddrs := make([]string, len(newAddrs))
		var removeAddrs []string

		// calculate addresses to remove
		for _, old := range oldAddrs {
			addrFound := false
			for _, newAddr := range newAddrs {
				if old == newAddr {
					addrFound = true
					break
				}
			}
			if !addrFound {
				removeAddrs = append(removeAddrs, old.(string))
			}
		}

		// calculate addresses to add
		for _, newAddr := range newAddrs {
			addrFound := false
			for _, old := range oldAddrs {
				if newAddr == old {
					addrFound = true
					break
				}
			}
			if !addrFound {
				addAddrs = append(addAddrs, newAddr.(string))
			}
		}

		if len(removeAddrs) > 0 || len(addAddrs) > 0 {
			addrs := make([]types.HostIpConfigIpV6Address, 0)
			for _, removeAddr := range removeAddrs {
				addrParts := strings.Split(removeAddr, "/")
				addr := addrParts[0]
				prefix, err := strconv.ParseInt(addrParts[1], 0, 32)
				if err != nil {
					return nil, fmt.Errorf("error while parsing IPv6 address")
				}
				tmpAddr := types.HostIpConfigIpV6Address{
					IpAddress:    strings.ToLower(addr),
					PrefixLength: int32(prefix),
					Origin:       "manual",
					Operation:    "remove",
				}
				addrs = append(addrs, tmpAddr)
			}

			for _, newAddr := range newAddrs {
				addrParts := strings.Split(newAddr.(string), "/")
				addr := addrParts[0]
				prefix, err := strconv.ParseInt(addrParts[1], 0, 32)
				if err != nil {
					return nil, fmt.Errorf("error while parsing IPv6 address")
				}
				tmpAddr := types.HostIpConfigIpV6Address{
					IpAddress:    strings.ToLower(addr),
					PrefixLength: int32(prefix),
					Origin:       "manual",
					Operation:    "add",
				}
				addrs = append(addrs, tmpAddr)
			}
			ipv6Spec.IpV6Address = addrs
		}
		routeConfig.IpV6DefaultGateway = ipv6Gateway
		ipConfig.IpV6Config = ipv6Spec
	}

	r := &types.HostVirtualNicIpRouteSpec{
		IpRouteConfig: routeConfig,
	}

	netStackInstance := d.Get("netstack").(string)

	vnic := &types.HostVirtualNicSpec{
		Ip:                     ipConfig,
		Mac:                    mac,
		Mtu:                    mtu,
		Portgroup:              portgroup,
		DistributedVirtualPort: dvpPortConnection,
		IpRouteSpec:            r,
		NetStackInstanceKey:    netStackInstance,
	}
	return vnic, nil
}

func splitHostIDMigratedNicID(d *schema.ResourceData) (string, string) {
	id := d.Get("vnic_id").(string)
	idParts := strings.Split(id, "_")
	return idParts[0], idParts[1]
}
