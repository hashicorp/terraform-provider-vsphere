// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package vsphere

import (
	"fmt"
	"strconv"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
	"github.com/hashicorp/terraform-provider-vsphere/vsphere/internal/helper/testhelper"
	"github.com/hashicorp/terraform-provider-vsphere/vsphere/internal/helper/viapi"
	"github.com/vmware/govmomi/vim25/types"
)

func TestAccResourceVSphereDistributedVirtualSwitchPvlanMapping_basic(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			RunSweepers()
			testAccPreCheck(t)
			testAccResourceVSphereDistributedVirtualSwitchPreCheck(t)
		},
		Providers:    testAccProviders,
		CheckDestroy: testAccResourceVSphereDistributedVirtualSwitchPvlanMappingExists(false),
		Steps: []resource.TestStep{
			{
				Config: testAccResourceVSphereDistributedVirtualSwitchPvlanMappingConfig(),
				Check: resource.ComposeTestCheckFunc(
					testAccResourceVSphereDistributedVirtualSwitchPvlanMappingExists(true),
				),
			},
		},
	})
}

func testAccResourceVSphereDistributedVirtualSwitchPvlanMappingConfig() string {
	return fmt.Sprintf(`
%s

resource "vsphere_distributed_virtual_switch_pvlan_mapping" "mapping" {
  distributed_virtual_switch_id = vsphere_distributed_virtual_switch.dvs.id
  
  primary_vlan_id = 1005
  secondary_vlan_id = 1005
  pvlan_type = "promiscuous"
}

resource "vsphere_distributed_virtual_switch" "dvs" {
  name          = "testacc-dvs2"
  datacenter_id = "${data.vsphere_datacenter.rootdc1.id}"
  ignore_other_pvlan_mappings = true

  host {
    host_system_id = data.vsphere_host.roothost2.id
    devices = ["%s"]
  }
}
`,
		testhelper.CombineConfigs(testhelper.ConfigDataRootDC1(), testhelper.ConfigDataRootHost2()),
		testhelper.HostNic0,
	)
}

func testAccResourceVSphereDistributedVirtualSwitchPvlanMappingExists(expected bool) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		props, err := testGetDVSProperties(s, "dvs")
		if err != nil {
			if viapi.IsAnyNotFoundError(err) && expected == false {
				// Expected missing
				return nil
			}

			return err
		}

		mappingToSearchFor, err := testClientVariablesForResource(s, "vsphere_distributed_virtual_switch_pvlan_mapping.mapping")
		if err != nil {
			return fmt.Errorf("could not find pvlan mapping resource: %s", err)
		}

		primary_vlan_id, err := strconv.Atoi(mappingToSearchFor.resourceAttributes["primary_vlan_id"])
		if err != nil {
			return err
		}
		secondary_vlan_id, err := strconv.Atoi(mappingToSearchFor.resourceAttributes["secondary_vlan_id"])
		if err != nil {
			return err
		}
		pvlan_type := mappingToSearchFor.resourceAttributes["pvlan_type"]

		for _, mapping := range props.Config.(*types.VMwareDVSConfigInfo).PvlanConfig {
			if mapping.PrimaryVlanId == int32(primary_vlan_id) && mapping.SecondaryVlanId == int32(secondary_vlan_id) && mapping.PvlanType == pvlan_type {
				if !expected {
					return fmt.Errorf("found PVLAN mapping when not expecting to")
				}
				return nil
			}
		}

		if expected {
			return fmt.Errorf("could not find PVLAN mapping")
		}

		return nil
	}
}
