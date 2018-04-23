package vsphere

import (
	"errors"
	"fmt"
	"os"
	"path"
	"testing"

	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
	"github.com/terraform-providers/terraform-provider-vsphere/vsphere/internal/helper/folder"
	"github.com/terraform-providers/terraform-provider-vsphere/vsphere/internal/helper/viapi"
	"github.com/vmware/govmomi/vim25/types"
)

const (
	testAccResourceVSphereComputeClusterNameStandard = "terraform-datastore-cluster-test"
	testAccResourceVSphereComputeClusterNameRenamed  = "terraform-datastore-cluster-test-renamed"
	testAccResourceVSphereComputeClusterFolder       = "datastore-cluster-folder-test"
)

func TestAccResourceVSphereComputeCluster_basic(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
			testAccResourceVSphereComputeClusterPreCheck(t)
		},
		Providers:    testAccProviders,
		CheckDestroy: testAccResourceVSphereComputeClusterCheckExists(false),
		Steps: []resource.TestStep{
			{
				Config: testAccResourceVSphereComputeClusterConfigBasic(),
				Check: resource.ComposeTestCheckFunc(
					testAccResourceVSphereComputeClusterCheckExists(true),
					testAccResourceVSphereComputeClusterCheckDRSEnabled(false),
				),
			},
		},
	})
}

func TestAccResourceVSphereComputeCluster_drsHAEnabled(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
			testAccResourceVSphereComputeClusterPreCheck(t)
		},
		Providers:    testAccProviders,
		CheckDestroy: testAccResourceVSphereComputeClusterCheckExists(false),
		Steps: []resource.TestStep{
			{
				Config: testAccResourceVSphereComputeClusterConfigDRSHABasic(),
				Check: resource.ComposeTestCheckFunc(
					testAccResourceVSphereComputeClusterCheckExists(true),
					testAccResourceVSphereComputeClusterCheckDRSEnabled(true),
					testAccResourceVSphereComputeClusterCheckHAEnabled(true),
				),
			},
		},
	})
}

func TestAccResourceVSphereComputeCluster_rename(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
			testAccResourceVSphereComputeClusterPreCheck(t)
		},
		Providers:    testAccProviders,
		CheckDestroy: testAccResourceVSphereComputeClusterCheckExists(false),
		Steps: []resource.TestStep{
			{
				Config: testAccResourceVSphereComputeClusterConfigWithName(testAccResourceVSphereComputeClusterNameStandard),
				Check: resource.ComposeTestCheckFunc(
					testAccResourceVSphereComputeClusterCheckExists(true),
					testAccResourceVSphereComputeClusterCheckName(testAccResourceVSphereComputeClusterNameStandard),
				),
			},
			{
				Config: testAccResourceVSphereComputeClusterConfigWithName(testAccResourceVSphereComputeClusterNameRenamed),
				Check: resource.ComposeTestCheckFunc(
					testAccResourceVSphereComputeClusterCheckExists(true),
					testAccResourceVSphereComputeClusterCheckName(testAccResourceVSphereComputeClusterNameRenamed),
				),
			},
		},
	})
}

func TestAccResourceVSphereComputeCluster_inFolder(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
			testAccResourceVSphereComputeClusterPreCheck(t)
		},
		Providers:    testAccProviders,
		CheckDestroy: testAccResourceVSphereComputeClusterCheckExists(false),
		Steps: []resource.TestStep{
			{
				Config: testAccResourceVSphereComputeClusterConfigWithFolder(testAccResourceVSphereComputeClusterFolder),
				Check: resource.ComposeTestCheckFunc(
					testAccResourceVSphereComputeClusterCheckExists(true),
					testAccResourceVSphereComputeClusterMatchInventoryPath(testAccResourceVSphereComputeClusterFolder),
				),
			},
		},
	})
}

func TestAccResourceVSphereComputeCluster_moveToFolder(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
			testAccResourceVSphereComputeClusterPreCheck(t)
		},
		Providers:    testAccProviders,
		CheckDestroy: testAccResourceVSphereComputeClusterCheckExists(false),
		Steps: []resource.TestStep{
			{
				Config: testAccResourceVSphereComputeClusterConfigBasic(),
				Check: resource.ComposeTestCheckFunc(
					testAccResourceVSphereComputeClusterCheckExists(true),
					testAccResourceVSphereComputeClusterMatchInventoryPath(""),
				),
			},
			{
				Config: testAccResourceVSphereComputeClusterConfigWithFolder(testAccResourceVSphereComputeClusterFolder),
				Check: resource.ComposeTestCheckFunc(
					testAccResourceVSphereComputeClusterCheckExists(true),
					testAccResourceVSphereComputeClusterMatchInventoryPath(testAccResourceVSphereComputeClusterFolder),
				),
			},
		},
	})
}

func TestAccResourceVSphereComputeCluster_singleTag(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
			testAccResourceVSphereComputeClusterPreCheck(t)
		},
		Providers:    testAccProviders,
		CheckDestroy: testAccResourceVSphereComputeClusterCheckExists(false),
		Steps: []resource.TestStep{
			{
				Config: testAccResourceVSphereComputeClusterConfigSingleTag(),
				Check: resource.ComposeTestCheckFunc(
					testAccResourceVSphereComputeClusterCheckExists(true),
					testAccResourceVSphereComputeClusterCheckTags("terraform-test-tag"),
				),
			},
		},
	})
}

func TestAccResourceVSphereComputeCluster_multipleTags(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
			testAccResourceVSphereComputeClusterPreCheck(t)
		},
		Providers:    testAccProviders,
		CheckDestroy: testAccResourceVSphereComputeClusterCheckExists(false),
		Steps: []resource.TestStep{
			{
				Config: testAccResourceVSphereComputeClusterConfigMultiTag(),
				Check: resource.ComposeTestCheckFunc(
					testAccResourceVSphereComputeClusterCheckExists(true),
					testAccResourceVSphereComputeClusterCheckTags("terraform-test-tags-alt"),
				),
			},
		},
	})
}

func TestAccResourceVSphereComputeCluster_switchTags(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
			testAccResourceVSphereComputeClusterPreCheck(t)
		},
		Providers:    testAccProviders,
		CheckDestroy: testAccResourceVSphereComputeClusterCheckExists(false),
		Steps: []resource.TestStep{
			{
				Config: testAccResourceVSphereComputeClusterConfigSingleTag(),
				Check: resource.ComposeTestCheckFunc(
					testAccResourceVSphereComputeClusterCheckExists(true),
					testAccResourceVSphereComputeClusterCheckTags("terraform-test-tag"),
				),
			},
			{
				Config: testAccResourceVSphereComputeClusterConfigMultiTag(),
				Check: resource.ComposeTestCheckFunc(
					testAccResourceVSphereComputeClusterCheckExists(true),
					testAccResourceVSphereComputeClusterCheckTags("terraform-test-tags-alt"),
				),
			},
		},
	})
}

func TestAccResourceVSphereComputeCluster_singleCustomAttribute(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
			testAccResourceVSphereComputeClusterPreCheck(t)
		},
		Providers:    testAccProviders,
		CheckDestroy: testAccResourceVSphereComputeClusterCheckExists(false),
		Steps: []resource.TestStep{
			{
				Config: testAccResourceVSphereComputeClusterConfigSingleCustomAttribute(),
				Check: resource.ComposeTestCheckFunc(
					testAccResourceVSphereComputeClusterCheckExists(true),
					testAccResourceVSphereComputeClusterCheckCustomAttributes(),
				),
			},
		},
	})
}

func TestAccResourceVSphereComputeCluster_multipleCustomAttribute(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
			testAccResourceVSphereComputeClusterPreCheck(t)
		},
		Providers:    testAccProviders,
		CheckDestroy: testAccResourceVSphereComputeClusterCheckExists(false),
		Steps: []resource.TestStep{
			{
				Config: testAccResourceVSphereComputeClusterConfigMultiCustomAttributes(),
				Check: resource.ComposeTestCheckFunc(
					testAccResourceVSphereComputeClusterCheckExists(true),
					testAccResourceVSphereComputeClusterCheckCustomAttributes(),
				),
			},
		},
	})
}

func TestAccResourceVSphereComputeCluster_switchCustomAttribute(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
			testAccResourceVSphereComputeClusterPreCheck(t)
		},
		Providers:    testAccProviders,
		CheckDestroy: testAccResourceVSphereComputeClusterCheckExists(false),
		Steps: []resource.TestStep{
			{
				Config: testAccResourceVSphereComputeClusterConfigSingleCustomAttribute(),
				Check: resource.ComposeTestCheckFunc(
					testAccResourceVSphereComputeClusterCheckExists(true),
					testAccResourceVSphereComputeClusterCheckCustomAttributes(),
				),
			},
			{
				Config: testAccResourceVSphereComputeClusterConfigMultiCustomAttributes(),
				Check: resource.ComposeTestCheckFunc(
					testAccResourceVSphereComputeClusterCheckExists(true),
					testAccResourceVSphereComputeClusterCheckCustomAttributes(),
				),
			},
		},
	})
}

func TestAccResourceVSphereComputeCluster_import(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
			testAccResourceVSphereComputeClusterPreCheck(t)
		},
		Providers:    testAccProviders,
		CheckDestroy: testAccResourceVSphereComputeClusterCheckExists(false),
		Steps: []resource.TestStep{
			{
				Config: testAccResourceVSphereComputeClusterConfigBasic(),
				Check: resource.ComposeTestCheckFunc(
					testAccResourceVSphereComputeClusterCheckExists(true),
					testAccResourceVSphereComputeClusterCheckDRSEnabled(false),
				),
			},
			{
				ResourceName:            "vsphere_compute_cluster.compute_cluster",
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"datacenter_id", "sdrs_free_space_threshold"},
				ImportStateIdFunc: func(s *terraform.State) (string, error) {
					cluster, err := testGetComputeCluster(s, "compute_cluster")
					if err != nil {
						return "", err
					}
					return cluster.InventoryPath, nil
				},
				Config: testAccResourceVSphereComputeClusterConfigBasic(),
				Check: resource.ComposeTestCheckFunc(
					testAccResourceVSphereComputeClusterCheckExists(true),
					testAccResourceVSphereComputeClusterCheckDRSEnabled(false),
				),
			},
		},
	})
}

func testAccResourceVSphereComputeClusterPreCheck(t *testing.T) {
	if os.Getenv("VSPHERE_DATACENTER") == "" {
		t.Skip("set VSPHERE_DATACENTER to run vsphere_compute_cluster acceptance tests")
	}
	if os.Getenv("VSPHERE_ESXI_HOST5") == "" {
		t.Skip("set VSPHERE_ESXI_HOST5 to run vsphere_compute_cluster acceptance tests")
	}
	if os.Getenv("VSPHERE_ESXI_HOST6") == "" {
		t.Skip("set VSPHERE_ESXI_HOST6 to run vsphere_compute_cluster acceptance tests")
	}
	if os.Getenv("VSPHERE_ESXI_HOST7") == "" {
		t.Skip("set VSPHERE_ESXI_HOST7 to run vsphere_compute_cluster acceptance tests")
	}
}

func testAccResourceVSphereComputeClusterCheckExists(expected bool) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		_, err := testGetComputeCluster(s, "compute_cluster")
		if err != nil {
			if viapi.IsManagedObjectNotFoundError(err) && expected == false {
				// Expected missing
				return nil
			}
			return err
		}
		if !expected {
			return errors.New("expected datastore cluster to be missing")
		}
		return nil
	}
}

func testAccResourceVSphereComputeClusterCheckDRSEnabled(expected bool) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		props, err := testGetComputeClusterProperties(s, "compute_cluster")
		if err != nil {
			return err
		}
		actual := *props.ConfigurationEx.(*types.ClusterConfigInfoEx).DrsConfig.Enabled
		if expected != actual {
			return fmt.Errorf("expected enabled to be %t, got %t", expected, actual)
		}
		return nil
	}
}

func testAccResourceVSphereComputeClusterCheckHAEnabled(expected bool) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		props, err := testGetComputeClusterProperties(s, "compute_cluster")
		if err != nil {
			return err
		}
		actual := *props.ConfigurationEx.(*types.ClusterConfigInfoEx).DasConfig.Enabled
		if expected != actual {
			return fmt.Errorf("expected enabled to be %t, got %t", expected, actual)
		}
		return nil
	}
}

func testAccResourceVSphereComputeClusterCheckName(expected string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		cluster, err := testGetComputeCluster(s, "compute_cluster")
		if err != nil {
			return err
		}
		actual := cluster.Name()
		if expected != actual {
			return fmt.Errorf("expected name to be %q, got %q", expected, actual)
		}
		return nil
	}
}

func testAccResourceVSphereComputeClusterMatchInventoryPath(expected string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		cluster, err := testGetComputeCluster(s, "compute_cluster")
		if err != nil {
			return err
		}

		expected, err = folder.RootPathParticleHost.PathFromNewRoot(cluster.InventoryPath, folder.RootPathParticleHost, expected)
		actual := path.Dir(cluster.InventoryPath)
		if err != nil {
			return fmt.Errorf("bad: %s", err)
		}
		if expected != actual {
			return fmt.Errorf("expected path to be %s, got %s", expected, actual)
		}
		return nil
	}
}

func testAccResourceVSphereComputeClusterCheckDRSDefaultAutomationLevel(expected types.DrsBehavior) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		props, err := testGetComputeClusterProperties(s, "compute_cluster")
		if err != nil {
			return err
		}
		actual := props.ConfigurationEx.(*types.ClusterConfigInfoEx).DrsConfig.DefaultVmBehavior
		if expected != actual {
			return fmt.Errorf("expected default automation level to be %q got %q", expected, actual)
		}
		return nil
	}
}

func testAccResourceVSphereComputeClusterCheckTags(tagResName string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		cluster, err := testGetComputeCluster(s, "compute_cluster")
		if err != nil {
			return err
		}
		tagsClient, err := testAccProvider.Meta().(*VSphereClient).TagsClient()
		if err != nil {
			return err
		}
		return testObjectHasTags(s, tagsClient, cluster, tagResName)
	}
}

func testAccResourceVSphereComputeClusterCheckCustomAttributes() resource.TestCheckFunc {
	return func(s *terraform.State) error {
		props, err := testGetComputeClusterProperties(s, "compute_cluster")
		if err != nil {
			return err
		}
		return testResourceHasCustomAttributeValues(s, "vsphere_compute_cluster", "compute_cluster", props.Entity())
	}
}

func testAccResourceVSphereComputeClusterConfigEmpty() string {
	return fmt.Sprintf(`
variable "datacenter" {
  default = "%s"
}

data "vsphere_datacenter" "dc" {
  name = "${var.datacenter}"
}

resource "vsphere_compute_cluster" "compute_cluster" {
  name            = "terraform-compute-cluster-test"
  datacenter_id   = "${data.vsphere_datacenter.dc.id}"
}
`,
		os.Getenv("VSPHERE_DATACENTER"),
	)
}

func testAccResourceVSphereComputeClusterConfigBasic() string {
	return fmt.Sprintf(`
variable "datacenter" {
  default = "%s"
}

variable "hosts" {
  default = [
    "%s",
    "%s",
    "%s",
  ]
}

data "vsphere_datacenter" "dc" {
  name = "${var.datacenter}"
}

data "vsphere_host" "hosts" {
  count         = "${length(var.hosts)}"
  name          = "${var.hosts[count.index]}"
  datacenter_id = "${data.vsphere_datacenter.dc.id}"
}

resource "vsphere_compute_cluster" "compute_cluster" {
  name            = "terraform-compute-cluster-test"
  datacenter_id   = "${data.vsphere_datacenter.dc.id}"
  host_system_ids = ["${data.vsphere_host.hosts.*.id}"]

  force_evacuate_on_destroy = true
}
`,
		os.Getenv("VSPHERE_DATACENTER"),
		os.Getenv("VSPHERE_ESXI_HOST5"),
		os.Getenv("VSPHERE_ESXI_HOST6"),
		os.Getenv("VSPHERE_ESXI_HOST7"),
	)
}

func testAccResourceVSphereComputeClusterConfigDRSHABasic() string {
	return fmt.Sprintf(`
variable "datacenter" {
  default = "%s"
}

variable "hosts" {
  default = [
    "%s",
    "%s",
    "%s",
  ]
}

data "vsphere_datacenter" "dc" {
  name = "${var.datacenter}"
}

data "vsphere_host" "hosts" {
  count         = "${length(var.hosts)}"
  name          = "${var.hosts[count.index]}"
  datacenter_id = "${data.vsphere_datacenter.dc.id}"
}

resource "vsphere_compute_cluster" "compute_cluster" {
  name            = "terraform-compute-cluster-test"
  datacenter_id   = "${data.vsphere_datacenter.dc.id}"
  host_system_ids = ["${data.vsphere_host.hosts.*.id}"]

  drs_enabled          = true
  drs_automation_level = "fullyAutomated"

  ha_enabled = true
  
	force_evacuate_on_destroy = true
}
`,
		os.Getenv("VSPHERE_DATACENTER"),
		os.Getenv("VSPHERE_ESXI_HOST5"),
		os.Getenv("VSPHERE_ESXI_HOST6"),
		os.Getenv("VSPHERE_ESXI_HOST7"),
	)
}

func testAccResourceVSphereComputeClusterConfigWithName(name string) string {
	return fmt.Sprintf(`
variable "datacenter" {
  default = "%s"
}

data "vsphere_datacenter" "dc" {
  name = "${var.datacenter}"
}

resource "vsphere_compute_cluster" "compute_cluster" {
  name          = "%s"
  datacenter_id = "${data.vsphere_datacenter.dc.id}"
}
`,
		os.Getenv("VSPHERE_DATACENTER"),
		name,
	)
}

func testAccResourceVSphereComputeClusterConfigWithFolder(f string) string {
	return fmt.Sprintf(`
variable "datacenter" {
  default = "%s"
}

variable "folder" {
  default = "%s"
}

data "vsphere_datacenter" "dc" {
  name = "${var.datacenter}"
}

resource "vsphere_folder" "compute_cluster_folder" {
  path          = "${var.folder}"
  type          = "datastore"
  datacenter_id = "${data.vsphere_datacenter.dc.id}"
}

resource "vsphere_compute_cluster" "compute_cluster" {
  name          = "terraform-datastore-cluster-test"
  datacenter_id = "${data.vsphere_datacenter.dc.id}"
  folder        = "${vsphere_folder.compute_cluster_folder.path}"
}
`,
		os.Getenv("VSPHERE_DATACENTER"),
		f,
	)
}

func testAccResourceVSphereComputeClusterConfigSDRSOverrides() string {
	return fmt.Sprintf(`
variable "datacenter" {
  default = "%s"
}

data "vsphere_datacenter" "dc" {
  name = "${var.datacenter}"
}

resource "vsphere_compute_cluster" "compute_cluster" {
  name                                     = "terraform-datastore-cluster-test"
  datacenter_id                            = "${data.vsphere_datacenter.dc.id}"
  sdrs_enabled                             = true
  sdrs_automation_level                    = "manual"
  sdrs_space_balance_automation_level      = "automated"
  sdrs_io_balance_automation_level         = "automated"
  sdrs_rule_enforcement_automation_level   = "automated"
  sdrs_policy_enforcement_automation_level = "automated"
  sdrs_vm_evacuation_automation_level      = "automated"
}
`,
		os.Getenv("VSPHERE_DATACENTER"),
	)
}

func testAccResourceVSphereComputeClusterConfigSDRSMiscTweaks() string {
	return fmt.Sprintf(`
variable "datacenter" {
  default = "%s"
}

data "vsphere_datacenter" "dc" {
  name = "${var.datacenter}"
}

resource "vsphere_compute_cluster" "compute_cluster" {
  name                             = "terraform-datastore-cluster-test"
  datacenter_id                    = "${data.vsphere_datacenter.dc.id}"
  sdrs_enabled                     = true
  sdrs_default_intra_vm_affinity   = false
  sdrs_io_latency_threshold        = 5
  sdrs_space_utilization_threshold = 50
}
`,
		os.Getenv("VSPHERE_DATACENTER"),
	)
}

func testAccResourceVSphereComputeClusterConfigReservableIopsManual() string {
	return fmt.Sprintf(`
variable "datacenter" {
  default = "%s"
}

data "vsphere_datacenter" "dc" {
  name = "${var.datacenter}"
}

resource "vsphere_compute_cluster" "compute_cluster" {
  name                              = "terraform-datastore-cluster-test"
  datacenter_id                     = "${data.vsphere_datacenter.dc.id}"
  sdrs_enabled                      = true
  sdrs_io_reservable_threshold_mode = "manual"
  sdrs_io_reservable_iops_threshold = 5000
}
`,
		os.Getenv("VSPHERE_DATACENTER"),
	)
}

func testAccResourceVSphereComputeClusterConfigReservableIopsAutomatic() string {
	return fmt.Sprintf(`
variable "datacenter" {
  default = "%s"
}

data "vsphere_datacenter" "dc" {
  name = "${var.datacenter}"
}

resource "vsphere_compute_cluster" "compute_cluster" {
  name                                 = "terraform-datastore-cluster-test"
  datacenter_id                        = "${data.vsphere_datacenter.dc.id}"
  sdrs_enabled                         = true
  sdrs_io_reservable_percent_threshold = 40
}
`,
		os.Getenv("VSPHERE_DATACENTER"),
	)
}

func testAccResourceVSphereComputeClusterConfigSpaceManual() string {
	return fmt.Sprintf(`
variable "datacenter" {
  default = "%s"
}

data "vsphere_datacenter" "dc" {
  name = "${var.datacenter}"
}

resource "vsphere_compute_cluster" "compute_cluster" {
  name                           = "terraform-datastore-cluster-test"
  datacenter_id                  = "${data.vsphere_datacenter.dc.id}"
  sdrs_enabled                   = true
  sdrs_free_space_threshold_mode = "freeSpace"
  sdrs_free_space_threshold      = 500
}
`,
		os.Getenv("VSPHERE_DATACENTER"),
	)
}

func testAccResourceVSphereComputeClusterConfigSingleTag() string {
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
    "StoragePod",
  ]
}

resource "vsphere_tag" "terraform-test-tag" {
  name        = "terraform-test-tag"
  category_id = "${vsphere_tag_category.terraform-test-category.id}"
}

resource "vsphere_compute_cluster" "compute_cluster" {
  name          = "terraform-datastore-cluster-test"
  datacenter_id = "${data.vsphere_datacenter.dc.id}"

  tags = [
    "${vsphere_tag.terraform-test-tag.id}",
  ]
}
`,
		os.Getenv("VSPHERE_DATACENTER"),
	)
}

func testAccResourceVSphereComputeClusterConfigMultiTag() string {
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
    "StoragePod",
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

resource "vsphere_compute_cluster" "compute_cluster" {
  name          = "terraform-datastore-cluster-test"
  datacenter_id = "${data.vsphere_datacenter.dc.id}"

  tags = ["${vsphere_tag.terraform-test-tags-alt.*.id}"]
}
`,
		os.Getenv("VSPHERE_DATACENTER"),
	)
}

func testAccResourceVSphereComputeClusterConfigSingleCustomAttribute() string {
	return fmt.Sprintf(`
variable "datacenter" {
  default = "%s"
}

data "vsphere_datacenter" "dc" {
  name = "${var.datacenter}"
}

resource "vsphere_custom_attribute" "terraform-test-attribute" {
  name                = "terraform-test-attribute"
  managed_object_type = "StoragePod"
}

locals {
  attrs = {
    "${vsphere_custom_attribute.terraform-test-attribute.id}" = "value"
  }
}

resource "vsphere_compute_cluster" "compute_cluster" {
  name          = "terraform-datastore-cluster-test"
  datacenter_id = "${data.vsphere_datacenter.dc.id}"

  custom_attributes = "${local.attrs}"
}
`,
		os.Getenv("VSPHERE_DATACENTER"),
	)
}

func testAccResourceVSphereComputeClusterConfigMultiCustomAttributes() string {
	return fmt.Sprintf(`
variable "datacenter" {
  default = "%s"
}

data "vsphere_datacenter" "dc" {
  name = "${var.datacenter}"
}

resource "vsphere_custom_attribute" "terraform-test-attribute" {
  name                = "terraform-test-attribute"
  managed_object_type = "StoragePod"
}

resource "vsphere_custom_attribute" "terraform-test-attribute-2" {
  name                = "terraform-test-attribute-2"
  managed_object_type = "StoragePod"
}

locals {
  attrs = {
    "${vsphere_custom_attribute.terraform-test-attribute.id}" = "value"
    "${vsphere_custom_attribute.terraform-test-attribute-2.id}" = "value-2"
  }
}

resource "vsphere_compute_cluster" "compute_cluster" {
  name          = "terraform-datastore-cluster-test"
  datacenter_id = "${data.vsphere_datacenter.dc.id}"

  custom_attributes = "${local.attrs}"
}
`,
		os.Getenv("VSPHERE_DATACENTER"),
	)
}
