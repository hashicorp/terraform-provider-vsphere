// © Broadcom. All Rights Reserved.
// The term "Broadcom" refers to Broadcom Inc. and/or its subsidiaries.
// SPDX-License-Identifier: MPL-2.0

package namespaceinstance

import (
	"errors"
	"fmt"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/vmware/govmomi/vapi/namespace"
	"github.com/vmware/terraform-provider-vsphere/vsphere/internal/helper/viapi"
)

const networkProviderNsxVpc = "NSX_VPC"

func ptrIfNonZero[T comparable](v T) *T {
	var zero T
	if v == zero {
		return nil
	}
	return &v
}

func truePtrOrNil(v bool) *bool {
	if !v {
		return nil
	}
	return &v
}

func ValidateVersionRequirements(version viapi.VSphereVersion, d *schema.ResourceDiff) error {
	var errs []error
	require := func(feature string, major, minor, patch int) {
		required := viapi.VSphereVersion{Product: version.Product, Major: major, Minor: minor, Patch: patch}
		if !version.AtLeast(required) {
			errs = append(errs, fmt.Errorf("%s requires vCenter %d.%d.%d or later (connected to %d.%d.%d)",
				feature, major, minor, patch, version.Major, version.Minor, version.Patch))
		}
	}

	if d.Get("zone").(*schema.Set).Len() > 0 {
		require("zone", 9, 0, 0)
	}

	if vpcNetworks := d.Get("vpc_network").([]interface{}); len(vpcNetworks) > 0 {
		require("vpc_network", 9, 0, 0)

		usesSharedSubnetPath, usesSharedSubnetDefaults := false, false
		for _, subnet := range sharedSubnetsFromVpcNetworks(vpcNetworks) {
			if subnet["path"].(string) != "" {
				usesSharedSubnetPath = true
			}
			if subnet["pod_default"].(bool) || subnet["vm_default"].(bool) || subnet["name"].(string) != "" {
				usesSharedSubnetDefaults = true
			}
		}
		if usesSharedSubnetPath {
			require("vpc_network.shared_subnet.path", 9, 1, 0)
		}
		if usesSharedSubnetDefaults {
			require("vpc_network.shared_subnet pod_default, vm_default and name", 9, 1, 1)
		}
	}

	return errors.Join(errs...)
}

func sharedSubnetsFromVpcNetworks(vpcNetworks []interface{}) []map[string]interface{} {
	vpcNetwork := firstVpcNetwork(vpcNetworks)
	if vpcNetwork == nil {
		return nil
	}
	subnets, _ := vpcNetwork["shared_subnet"].(*schema.Set)
	if subnets == nil {
		return nil
	}
	result := make([]map[string]interface{}, 0, subnets.Len())
	for _, subnet := range subnets.List() {
		result = append(result, subnet.(map[string]interface{}))
	}
	return result
}

func firstVpcNetwork(vpcNetworks []interface{}) map[string]interface{} {
	if len(vpcNetworks) == 0 || vpcNetworks[0] == nil {
		return nil
	}
	return vpcNetworks[0].(map[string]interface{})
}

func ExpandStorageSpecs(d *schema.ResourceData) []namespace.StorageSpec {
	storage := d.Get("storage").([]interface{})
	specs := make([]namespace.StorageSpec, 0, len(storage))
	for _, item := range storage {
		entry := item.(map[string]interface{})
		specs = append(specs, namespace.StorageSpec{
			Policy: entry["policy"].(string),
			Limit:  ptrIfNonZero(int64(entry["limit"].(int))),
		})
	}
	return specs
}

func FlattenStorageSpecs(specs []namespace.StorageSpec) []interface{} {
	result := make([]interface{}, 0, len(specs))
	for _, spec := range specs {
		limit := 0
		if spec.Limit != nil {
			limit = int(*spec.Limit)
		}
		result = append(result, map[string]interface{}{
			"policy": spec.Policy,
			"limit":  limit,
		})
	}
	return result
}

func ExpandAccessList(d *schema.ResourceData) []namespace.Access {
	entries := d.Get("access").(*schema.Set).List()
	accessList := make([]namespace.Access, 0, len(entries))
	for _, item := range entries {
		entry := item.(map[string]interface{})
		accessList = append(accessList, namespace.Access{
			SubjectType:      namespace.AccessSubjectType(entry["subject_type"].(string)),
			Subject:          entry["subject"].(string),
			Domain:           entry["domain"].(string),
			Role:             namespace.AccessRole(entry["role"].(string)),
			IdentityProvider: ptrIfNonZero(entry["identity_provider"].(string)),
		})
	}
	return accessList
}

func FlattenAccessList(accessList []namespace.Access) []interface{} {
	result := make([]interface{}, 0, len(accessList))
	for _, access := range accessList {
		identityProvider := ""
		if access.IdentityProvider != nil {
			identityProvider = *access.IdentityProvider
		}
		result = append(result, map[string]interface{}{
			"subject_type":      string(access.SubjectType),
			"subject":           access.Subject,
			"domain":            access.Domain,
			"role":              string(access.Role),
			"identity_provider": identityProvider,
		})
	}
	return result
}

func ExpandZones(d *schema.ResourceData) []namespace.ZoneSpec {
	entries := d.Get("zone").(*schema.Set).List()
	zones := make([]namespace.ZoneSpec, 0, len(entries))
	for _, item := range entries {
		entry := item.(map[string]interface{})
		zone := namespace.ZoneSpec{
			Name:                 entry["name"].(string),
			CpuLimitMHz:          ptrIfNonZero(int64(entry["cpu_limit"].(int))),
			CpuReservationMHz:    ptrIfNonZero(int64(entry["cpu_reservation"].(int))),
			MemoryLimitMiB:       ptrIfNonZero(int64(entry["memory_limit"].(int))),
			MemoryReservationMiB: ptrIfNonZero(int64(entry["memory_reservation"].(int))),
		}
		zone.DisallowUnreservedDirectPathUsage = truePtrOrNil(entry["disallow_unreserved_direct_path_usage"].(bool))
		if reservations := expandVmReservations(entry["vm_reservation"]); len(reservations) > 0 {
			zone.VmReservations = &reservations
		}
		zones = append(zones, zone)
	}
	return zones
}

func expandVmReservations(value interface{}) []namespace.VirtualMachineClassAllocationSpec {
	set, _ := value.(*schema.Set)
	if set == nil {
		return nil
	}
	reservations := make([]namespace.VirtualMachineClassAllocationSpec, 0, set.Len())
	for _, item := range set.List() {
		entry := item.(map[string]interface{})
		reservations = append(reservations, namespace.VirtualMachineClassAllocationSpec{
			ReservedVmClass: entry["reserved_vm_class"].(string),
			Count:           int64(entry["count"].(int)),
		})
	}
	return reservations
}

var zoneSettingKeys = []string{
	"cpu_limit",
	"cpu_reservation",
	"memory_limit",
	"memory_reservation",
	"disallow_unreserved_direct_path_usage",
}

func FlattenZones(d *schema.ResourceData, zones []namespace.ZoneInfo) []interface{} {
	priorByName := make(map[string]map[string]interface{})
	if prior, ok := d.Get("zone").(*schema.Set); ok {
		for _, item := range prior.List() {
			entry := item.(map[string]interface{})
			priorByName[entry["name"].(string)] = entry
		}
	}

	result := make([]interface{}, 0, len(zones))
	for _, zone := range zones {
		if zone.MarkedForRemoval {
			continue
		}
		flattened := map[string]interface{}{"name": zone.Zone}
		if prior, found := priorByName[zone.Zone]; found {
			for _, key := range zoneSettingKeys {
				flattened[key] = prior[key]
			}
			if reservations, ok := prior["vm_reservation"].(*schema.Set); ok {
				flattened["vm_reservation"] = reservations.List()
			}
		}
		result = append(result, flattened)
	}
	return result
}

func FlattenZoneInfos(zones []namespace.ZoneInfo) []interface{} {
	result := make([]interface{}, 0, len(zones))
	for _, zone := range zones {
		result = append(result, map[string]interface{}{
			"name": zone.Zone,
			"type": zone.Type,
		})
	}
	return result
}

func ExpandNetworkSpecCreate(d *schema.ResourceData) *namespace.NetworkConfigCreateSpec {
	vpcNetwork := firstVpcNetwork(d.Get("vpc_network").([]interface{}))
	if vpcNetwork == nil {
		return nil
	}

	// TODO: select VSPHERE_NETWORK when vsphere_network is added
	vpc := &namespace.VpcNetworkCreateSpec{
		Vpc:               ptrIfNonZero(vpcNetwork["vpc"].(string)),
		DefaultSubnetSize: ptrIfNonZero(int64(vpcNetwork["default_subnet_size"].(int))),
	}
	if cidrs := expandPrivateCidrs(vpcNetwork["vpc_config"]); len(cidrs) > 0 {
		vpc.VpcConfig = &namespace.VpcConfig{PrivateCidrs: cidrs}
	}
	if subnets := expandSharedSubnets(vpcNetwork["shared_subnet"]); len(subnets) > 0 {
		vpc.SharedSubnets = &subnets
	}

	return &namespace.NetworkConfigCreateSpec{
		NetworkProvider: networkProviderNsxVpc,
		VpcNetwork:      vpc,
	}
}

func ExpandNetworkSpecUpdate(d *schema.ResourceData) *namespace.NetworkConfigUpdateSpec {
	vpcNetwork := firstVpcNetwork(d.Get("vpc_network").([]interface{}))
	if vpcNetwork == nil {
		return nil
	}

	// TODO: select VSPHERE_NETWORK when vsphere_network is added
	vpc := &namespace.VpcNetworkUpdateSpec{}
	if d.HasChange("vpc_network.0.default_subnet_size") {
		vpc.DefaultSubnetSize = ptrIfNonZero(int64(vpcNetwork["default_subnet_size"].(int)))
	}
	if d.HasChange("vpc_network.0.vpc_config") {
		vpc.PrivateCidrs = expandPrivateCidrs(vpcNetwork["vpc_config"])
	}
	subnets := expandSharedSubnets(vpcNetwork["shared_subnet"])
	if len(subnets) > 0 || d.HasChange("vpc_network.0.shared_subnet") {
		vpc.SharedSubnets = &subnets
	}

	return &namespace.NetworkConfigUpdateSpec{
		NetworkProvider: networkProviderNsxVpc,
		VpcConfig:       vpc,
	}
}

func expandPrivateCidrs(vpcConfig interface{}) []namespace.Ipv4Cidr {
	configs, _ := vpcConfig.([]interface{})
	if len(configs) == 0 || configs[0] == nil {
		return nil
	}
	cidrs, _ := configs[0].(map[string]interface{})["private_cidrs"].([]interface{})
	result := make([]namespace.Ipv4Cidr, 0, len(cidrs))
	for _, item := range cidrs {
		entry := item.(map[string]interface{})
		result = append(result, namespace.Ipv4Cidr{
			Address: entry["address"].(string),
			Prefix:  entry["prefix"].(int),
		})
	}
	return result
}

func expandSharedSubnets(value interface{}) []namespace.SharedSubnet {
	set, _ := value.(*schema.Set)
	if set == nil {
		return []namespace.SharedSubnet{}
	}
	subnets := make([]namespace.SharedSubnet, 0, set.Len())
	for _, item := range set.List() {
		entry := item.(map[string]interface{})
		subnets = append(subnets, namespace.SharedSubnet{
			Path:       entry["path"].(string),
			PodDefault: truePtrOrNil(entry["pod_default"].(bool)),
			VmDefault:  truePtrOrNil(entry["vm_default"].(bool)),
			Name:       ptrIfNonZero(entry["name"].(string)),
		})
	}
	return subnets
}

func FlattenNetworkSpec(info *namespace.NetworkConfigInfo) []interface{} {
	// TODO: flatten vsphere_network when it is added
	if info == nil || info.VpcNetwork == nil {
		return nil
	}
	vpc := info.VpcNetwork

	cidrs := make([]interface{}, 0, len(vpc.VpcConfig.PrivateCidrs))
	for _, cidr := range vpc.VpcConfig.PrivateCidrs {
		cidrs = append(cidrs, map[string]interface{}{
			"address": cidr.Address,
			"prefix":  cidr.Prefix,
		})
	}

	subnets := make([]interface{}, 0, len(vpc.SharedSubnets))
	for _, subnet := range vpc.SharedSubnets {
		flattened := map[string]interface{}{"path": subnet.Path}
		if subnet.PodDefault != nil {
			flattened["pod_default"] = *subnet.PodDefault
		}
		if subnet.VmDefault != nil {
			flattened["vm_default"] = *subnet.VmDefault
		}
		if subnet.Name != nil {
			flattened["name"] = *subnet.Name
		}
		subnets = append(subnets, flattened)
	}

	vpcNetwork := map[string]interface{}{
		"vpc":                 vpc.Vpc,
		"default_subnet_size": int(vpc.DefaultSubnetSize),
		"vpc_config":          []interface{}{map[string]interface{}{"private_cidrs": cidrs}},
		"shared_subnet":       subnets,
	}
	return []interface{}{vpcNetwork}
}

func FlattenNetworkSpecTrackingNames(d *schema.ResourceData, info *namespace.NetworkConfigInfo) []interface{} {
	vpcNetworks := FlattenNetworkSpec(info)
	vpcNetwork := firstVpcNetwork(vpcNetworks)
	if vpcNetwork == nil {
		return vpcNetworks
	}

	configuredNames := make(map[string]string)
	for _, subnet := range sharedSubnetsFromVpcNetworks(d.Get("vpc_network").([]interface{})) {
		configuredNames[subnet["path"].(string)] = subnet["name"].(string)
	}
	for _, item := range vpcNetwork["shared_subnet"].([]interface{}) {
		subnet := item.(map[string]interface{})
		subnet["name"] = configuredNames[subnet["path"].(string)]
	}
	return vpcNetworks
}
