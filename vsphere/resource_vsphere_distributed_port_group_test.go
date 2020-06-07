package vsphere

import (
	"fmt"
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/terraform"
	"github.com/terraform-providers/terraform-provider-vsphere/vsphere/internal/helper/viapi"
	"github.com/vmware/govmomi/vim25/types"
)

func TestAccResourceVSphereDistributedPortGroup_basic(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
			testAccResourceVSphereDistributedPortGroupPreCheck(t)
		},
		Providers:    testAccProviders,
		CheckDestroy: testAccResourceVSphereDistributedPortGroupExists(false),
		Steps: []resource.TestStep{
			{
				Config: testAccResourceVSphereDistributedPortGroupConfig(),
				Check: resource.ComposeTestCheckFunc(
					testAccResourceVSphereDistributedPortGroupExists(true),
				),
			},
			{
				ResourceName:            "vsphere_distributed_port_group.pg",
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"vlan_range"},
				ImportStateIdFunc: func(s *terraform.State) (string, error) {
					pg, err := testGetDVPortgroup(s, "pg")
					if err != nil {
						return "", err
					}
					return pg.InventoryPath, nil
				},
				Config: testAccResourceVSphereDistributedPortGroupConfig(),
				Check: resource.ComposeTestCheckFunc(
					testAccResourceVSphereDistributedPortGroupExists(true),
				),
			},
		},
	})
}

func TestAccResourceVSphereDistributedPortGroup_inheritPolicyDiffCheck(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
			testAccResourceVSphereDistributedPortGroupPreCheck(t)
		},
		Providers:    testAccProviders,
		CheckDestroy: testAccResourceVSphereDistributedPortGroupExists(false),
		Steps: []resource.TestStep{
			{
				Config: testAccResourceVSphereDistributedPortGroupConfigPolicyInherit(),
				Check: resource.ComposeTestCheckFunc(
					testAccResourceVSphereDistributedPortGroupExists(true),
				),
			},
		},
	})
}

func TestAccResourceVSphereDistributedPortGroup_inheritPolicyDiffCheckVlanRangeTypeSetEdition(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
			testAccResourceVSphereDistributedPortGroupPreCheck(t)
		},
		Providers:    testAccProviders,
		CheckDestroy: testAccResourceVSphereDistributedPortGroupExists(false),
		Steps: []resource.TestStep{
			{
				Config: testAccResourceVSphereDistributedPortGroupConfigPolicyInheritVLANRange(),
				Check: resource.ComposeTestCheckFunc(
					testAccResourceVSphereDistributedPortGroupExists(true),
				),
			},
		},
	})
}

func TestAccResourceVSphereDistributedPortGroup_overrideVlan(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
			testAccResourceVSphereDistributedPortGroupPreCheck(t)
		},
		Providers:    testAccProviders,
		CheckDestroy: testAccResourceVSphereDistributedPortGroupExists(false),
		Steps: []resource.TestStep{
			{
				Config: testAccResourceVSphereDistributedPortGroupConfigOverrideVLAN(),
				Check: resource.ComposeTestCheckFunc(
					testAccResourceVSphereDistributedPortGroupExists(true),
					testAccResourceVSphereDistributedVirtualSwitchHasVlanRange(1000, 1999),
					testAccResourceVSphereDistributedPortGroupHasVlanRange(3000, 3999),
				),
			},
		},
	})
}

func TestAccResourceVSphereDistributedPortGroup_singleTag(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
			testAccResourceVSphereDistributedPortGroupPreCheck(t)
		},
		Providers:    testAccProviders,
		CheckDestroy: testAccResourceVSphereDistributedPortGroupExists(false),
		Steps: []resource.TestStep{
			{
				Config: testAccResourceVSphereDistributedPortGroupConfigSingleTag(),
				Check: resource.ComposeTestCheckFunc(
					testAccResourceVSphereDistributedPortGroupExists(true),
					testAccResourceVSphereDistributedPortGroupCheckTags("terraform-test-tag"),
				),
			},
		},
	})
}

func TestAccResourceVSphereDistributedPortGroup_multiTag(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
			testAccResourceVSphereDistributedPortGroupPreCheck(t)
		},
		Providers:    testAccProviders,
		CheckDestroy: testAccResourceVSphereDistributedPortGroupExists(false),
		Steps: []resource.TestStep{
			{
				Config: testAccResourceVSphereDistributedPortGroupConfigMultiTag(),
				Check: resource.ComposeTestCheckFunc(
					testAccResourceVSphereDistributedPortGroupExists(true),
					testAccResourceVSphereDistributedPortGroupCheckTags("terraform-test-tags-alt"),
				),
			},
		},
	})
}

func TestAccResourceVSphereDistributedPortGroup_singleCustomAttribute(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
			testAccResourceVSphereDistributedPortGroupPreCheck(t)
		},
		Providers:    testAccProviders,
		CheckDestroy: testAccResourceVSphereDistributedPortGroupExists(false),
		Steps: []resource.TestStep{
			{
				Config: testAccResourceVSphereDistributedPortGroupConfigSingleCustomAttribute(),
				Check: resource.ComposeTestCheckFunc(
					testAccResourceVSphereDistributedPortGroupExists(true),
					testAccResourceVSphereDistributedPortGroupCheckCustomAttributes(),
				),
			},
		},
	})
}

func TestAccResourceVSphereDistributedPortGroup_multiCustomAttribute(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
			testAccResourceVSphereDistributedPortGroupPreCheck(t)
		},
		Providers:    testAccProviders,
		CheckDestroy: testAccResourceVSphereDistributedPortGroupExists(false),
		Steps: []resource.TestStep{
			{
				Config: testAccResourceVSphereDistributedPortGroupConfigSingleCustomAttribute(),
				Check: resource.ComposeTestCheckFunc(
					testAccResourceVSphereDistributedPortGroupExists(true),
					testAccResourceVSphereDistributedPortGroupCheckCustomAttributes(),
				),
			},
			{
				Config: testAccResourceVSphereDistributedPortGroupConfigMultiCustomAttribute(),
				Check: resource.ComposeTestCheckFunc(
					testAccResourceVSphereDistributedPortGroupExists(true),
					testAccResourceVSphereDistributedPortGroupCheckCustomAttributes(),
				),
			},
		},
	})
}

func testAccResourceVSphereDistributedPortGroupPreCheck(t *testing.T) {
	if os.Getenv("TF_VAR_VSPHERE_HOST_NIC0") == "" {
		t.Skip("set TF_VAR_VSPHERE_HOST_NIC0 to run vsphere_host_virtual_switch acceptance tests")
	}
	if os.Getenv("TF_VAR_VSPHERE_HOST_NIC1") == "" {
		t.Skip("set TF_VAR_VSPHERE_HOST_NIC1 to run vsphere_host_virtual_switch acceptance tests")
	}
	if os.Getenv("TF_VAR_VSPHERE_NFS_DS_NAME") == "" {
		t.Skip("set TF_VAR_VSPHERE_ESXI_HOST to run vsphere_host_virtual_switch acceptance tests")
	}
	if os.Getenv("TF_VAR_VSPHERE_ESXI_HOST2") == "" {
		t.Skip("set TF_VAR_VSPHERE_ESXI_HOST2 to run vsphere_host_virtual_switch acceptance tests")
	}
	if os.Getenv("TF_VAR_VSPHERE_ESXI_HOST3") == "" {
		t.Skip("set TF_VAR_VSPHERE_ESXI_HOST3 to run vsphere_host_virtual_switch acceptance tests")
	}
}

func testAccResourceVSphereDistributedPortGroupExists(expected bool) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		dvs, err := testGetDVPortgroup(s, "pg")
		if err != nil {
			if viapi.IsAnyNotFoundError(err) && expected == false {
				// Expected missing
				return nil
			}
			return err
		}
		if !expected {
			return fmt.Errorf("expected DVS %s to be missing", dvs.Reference().Value)
		}
		return nil
	}
}

func testAccResourceVSphereDistributedPortGroupHasVlanRange(emin, emax int32) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		props, err := testGetDVPortgroupProperties(s, "pg")
		if err != nil {
			return err
		}
		pc := props.Config.DefaultPortConfig.(*types.VMwareDVSPortSetting)
		ranges := pc.Vlan.(*types.VmwareDistributedVirtualSwitchTrunkVlanSpec).VlanId
		var found bool
		for _, rng := range ranges {
			if rng.Start == emin && rng.End == emax {
				found = true
			}
		}
		if !found {
			return fmt.Errorf("could not find start %d and end %d in %#v", emin, emax, ranges)
		}
		return nil
	}
}

func testAccResourceVSphereDistributedPortGroupCheckTags(tagResName string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		dvs, err := testGetDVPortgroup(s, "pg")
		if err != nil {
			return err
		}
		tagsClient, err := testAccProvider.Meta().(*VSphereClient).TagsManager()
		if err != nil {
			return err
		}
		return testObjectHasTags(s, tagsClient, dvs, tagResName)
	}
}

func testAccResourceVSphereDistributedPortGroupCheckCustomAttributes() resource.TestCheckFunc {
	return func(s *terraform.State) error {
		props, err := testGetDVPortgroupProperties(s, "pg")
		if err != nil {
			return err
		}
		return testResourceHasCustomAttributeValues(s, "vsphere_distributed_port_group", "pg", props.Entity())
	}
}

func testAccResourceVSphereDistributedPortGroupConfig() string {
	return fmt.Sprintf(`
variable "datacenter" {
  default = "%s"
}

data "vsphere_datacenter" "dc" {
  name = "${var.datacenter}"
}

resource "vsphere_distributed_virtual_switch" "dvs" {
  name          = "terraform-test-dvs"
  datacenter_id = "${data.vsphere_datacenter.dc.id}"
}

resource "vsphere_distributed_port_group" "pg" {
  name                            = "terraform-test-pg"
  distributed_virtual_switch_uuid = "${vsphere_distributed_virtual_switch.dvs.id}"
}
`,
		os.Getenv("TF_VAR_VSPHERE_DATACENTER"),
	)
}

func testAccResourceVSphereDistributedPortGroupConfigPolicyInherit() string {
	return fmt.Sprintf(`
variable "datacenter" {
  default = "%s"
}

data "vsphere_datacenter" "dc" {
  name = "${var.datacenter}"
}

resource "vsphere_distributed_virtual_switch" "dvs" {
  name          = "terraform-test-dvs"
  datacenter_id = "${data.vsphere_datacenter.dc.id}"

  vlan_id = 1000

  active_uplinks  = ["uplink1", "uplink2"]
  standby_uplinks = ["uplink3", "uplink4"]
  check_beacon    = true
  failback        = true
  notify_switches = true
  teaming_policy  = "failover_explicit"

  lacp_enabled = true
  lacp_mode    = "active"

  allow_forged_transmits = true
  allow_mac_changes      = true
  allow_promiscuous      = true

  ingress_shaping_enabled           = true
  ingress_shaping_average_bandwidth = 1000000
  ingress_shaping_peak_bandwidth    = 10000000
  ingress_shaping_burst_size        = 5000000

  egress_shaping_enabled           = true
  egress_shaping_average_bandwidth = 1000000
  egress_shaping_peak_bandwidth    = 10000000
  egress_shaping_burst_size        = 5000000

  block_all_ports = true
  netflow_enabled = true
  tx_uplink       = true
}

resource "vsphere_distributed_port_group" "pg" {
  name                            = "terraform-test-pg"
  distributed_virtual_switch_uuid = "${vsphere_distributed_virtual_switch.dvs.id}"
}
`,
		os.Getenv("TF_VAR_VSPHERE_DATACENTER"),
	)
}

func testAccResourceVSphereDistributedPortGroupConfigPolicyInheritVLANRange() string {
	return fmt.Sprintf(`
variable "datacenter" {
  default = "%s"
}

data "vsphere_datacenter" "dc" {
  name = "${var.datacenter}"
}

resource "vsphere_distributed_virtual_switch" "dvs" {
  name          = "terraform-test-dvs"
  datacenter_id = "${data.vsphere_datacenter.dc.id}"

  vlan_range {
		min_vlan = 1000
		max_vlan = 1999
	}
  
	vlan_range {
		min_vlan = 3000
		max_vlan = 3999
	}

  active_uplinks  = ["uplink1", "uplink2"]
  standby_uplinks = ["uplink3", "uplink4"]
  check_beacon    = true
  failback        = true
  notify_switches = true
  teaming_policy  = "failover_explicit"

  lacp_enabled = true
  lacp_mode    = "active"

  allow_forged_transmits = true
  allow_mac_changes      = true
  allow_promiscuous      = true

  ingress_shaping_enabled           = true
  ingress_shaping_average_bandwidth = 1000000
  ingress_shaping_peak_bandwidth    = 10000000
  ingress_shaping_burst_size        = 5000000

  egress_shaping_enabled           = true
  egress_shaping_average_bandwidth = 1000000
  egress_shaping_peak_bandwidth    = 10000000
  egress_shaping_burst_size        = 5000000

  block_all_ports = true
  netflow_enabled = true
  tx_uplink       = true
}

resource "vsphere_distributed_port_group" "pg" {
  name                            = "terraform-test-pg"
  distributed_virtual_switch_uuid = "${vsphere_distributed_virtual_switch.dvs.id}"
}
`,
		os.Getenv("TF_VAR_VSPHERE_DATACENTER"),
	)
}

func testAccResourceVSphereDistributedPortGroupConfigOverrideVLAN() string {
	return fmt.Sprintf(`
variable "datacenter" {
  default = "%s"
}

data "vsphere_datacenter" "dc" {
  name = "${var.datacenter}"
}

resource "vsphere_distributed_virtual_switch" "dvs" {
  name          = "terraform-test-dvs"
  datacenter_id = "${data.vsphere_datacenter.dc.id}"
  
	vlan_range {
		min_vlan = 1000
		max_vlan = 1999
	}
}

resource "vsphere_distributed_port_group" "pg" {
  name                            = "terraform-test-pg"
  distributed_virtual_switch_uuid = "${vsphere_distributed_virtual_switch.dvs.id}"

	vlan_range {
		min_vlan = 3000
		max_vlan = 3999
	}
}
`,
		os.Getenv("TF_VAR_VSPHERE_DATACENTER"),
	)
}

func testAccResourceVSphereDistributedPortGroupConfigSingleTag() string {
	return fmt.Sprintf(`
variable "datacenter" {
  default = "%s"
}

data "vsphere_datacenter" "dc" {
  name = "${var.datacenter}"
}

resource "vsphere_tag_category" "terraform-test-category" {
  name        = "terraform-test-tag-category"
  cardinality = "MULTIPLE"

  associable_types = [
    "VmwareDistributedVirtualPortgroup",
  ]
}

resource "vsphere_tag" "terraform-test-tag" {
  name        = "terraform-test-tag"
  category_id = "${vsphere_tag_category.terraform-test-category.id}"
}

resource "vsphere_distributed_virtual_switch" "dvs" {
  name          = "terraform-test-dvs"
  datacenter_id = "${data.vsphere_datacenter.dc.id}"
}

resource "vsphere_distributed_port_group" "pg" {
  name                            = "terraform-test-pg"
  distributed_virtual_switch_uuid = "${vsphere_distributed_virtual_switch.dvs.id}"
  tags                            = ["${vsphere_tag.terraform-test-tag.id}"]
}
`,
		os.Getenv("TF_VAR_VSPHERE_DATACENTER"),
	)
}

func testAccResourceVSphereDistributedPortGroupConfigMultiTag() string {
	return fmt.Sprintf(`
variable "datacenter" {
  default = "%s"
}

variable "extra_tags" {
  default = [
    "terraform-test-thing1",
    "terraform-test-thing2",
  ]
}

data "vsphere_datacenter" "dc" {
  name = "${var.datacenter}"
}

resource "vsphere_tag_category" "terraform-test-category" {
  name        = "terraform-test-tag-category"
  cardinality = "MULTIPLE"

  associable_types = [
    "VmwareDistributedVirtualPortgroup",
  ]
}

resource "vsphere_tag" "terraform-test-tag" {
  name        = "terraform-test-tag"
  category_id = "${vsphere_tag_category.terraform-test-category.id}"
}

resource "vsphere_tag" "terraform-test-tags-alt" {
  count       = "${length(var.extra_tags)}"
  name        = "${var.extra_tags[count.index]}"
  category_id = "${vsphere_tag_category.terraform-test-category.id}"
}

resource "vsphere_distributed_virtual_switch" "dvs" {
  name          = "terraform-test-dvs"
  datacenter_id = "${data.vsphere_datacenter.dc.id}"
}

resource "vsphere_distributed_port_group" "pg" {
  name                            = "terraform-test-pg"
  distributed_virtual_switch_uuid = "${vsphere_distributed_virtual_switch.dvs.id}"
  tags                            = "${vsphere_tag.terraform-test-tags-alt.*.id}"
}
`,
		os.Getenv("TF_VAR_VSPHERE_DATACENTER"),
	)
}

func testAccResourceVSphereDistributedPortGroupConfigSingleCustomAttribute() string {
	return fmt.Sprintf(`
variable "datacenter" {
  default = "%s"
}

data "vsphere_datacenter" "dc" {
  name = "${var.datacenter}"
}

resource "vsphere_custom_attribute" "terraform-test-attribute" {
  name                = "terraform-test-attribute"
  managed_object_type = "DistributedVirtualPortgroup" 
}

resource "vsphere_distributed_virtual_switch" "dvs" {
  name          = "terraform-test-dvs"
  datacenter_id = "${data.vsphere_datacenter.dc.id}"
}

locals {
  pg_attrs = {
    "${vsphere_custom_attribute.terraform-test-attribute.id}" = "value"
  }
}

resource "vsphere_distributed_port_group" "pg" {
  name                            = "terraform-test-pg"
  distributed_virtual_switch_uuid = "${vsphere_distributed_virtual_switch.dvs.id}"

  custom_attributes = "${local.pg_attrs}"
}
`,
		os.Getenv("TF_VAR_VSPHERE_DATACENTER"),
	)
}

func testAccResourceVSphereDistributedPortGroupConfigMultiCustomAttribute() string {
	return fmt.Sprintf(`
variable "datacenter" {
  default = "%s"
}

variable "custom_attrs" {
  default = [
    "terraform-test-attribute-1",
    "terraform-test-attribute-2"
  ]
}

data "vsphere_datacenter" "dc" {
  name = "${var.datacenter}"
}

resource "vsphere_custom_attribute" "terraform-test-attribute" {
  name                = "terraform-test-attribute"
  managed_object_type = "DistributedVirtualPortgroup"
}

resource "vsphere_custom_attribute" "terraform-test-attribute-alt" {
  count               = "${length(var.custom_attrs)}"
  name                = "${var.custom_attrs[count.index]}"
  managed_object_type = "DistributedVirtualPortgroup"
}

resource "vsphere_distributed_virtual_switch" "dvs" {
  name          = "terraform-test-dvs"
  datacenter_id = "${data.vsphere_datacenter.dc.id}"
}

locals {
  pg_attrs = {
    "${vsphere_custom_attribute.terraform-test-attribute-alt.0.id}" = "value"
    "${vsphere_custom_attribute.terraform-test-attribute-alt.1.id}" = "value-2"
  }
}

resource "vsphere_distributed_port_group" "pg" {
  name                            = "terraform-test-pg"
  distributed_virtual_switch_uuid = "${vsphere_distributed_virtual_switch.dvs.id}"

  custom_attributes = "${local.pg_attrs}"
}
`,
		os.Getenv("TF_VAR_VSPHERE_DATACENTER"),
	)
}
