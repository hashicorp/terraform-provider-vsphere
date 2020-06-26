package vsphere

import (
	"fmt"
	"os"
	"path"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/terraform"
	"github.com/hashicorp/terraform-provider-vsphere/vsphere/internal/helper/datastore"
	"github.com/hashicorp/terraform-provider-vsphere/vsphere/internal/helper/folder"
	"github.com/hashicorp/terraform-provider-vsphere/vsphere/internal/helper/viapi"
)

func TestAccResourceVSphereVmfsDatastore_basic(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
			testAccResourceVSphereVmfsDatastorePreCheck(t)
		},
		Providers:    testAccProviders,
		CheckDestroy: testAccResourceVSphereVmfsDatastoreExists(false),
		Steps: []resource.TestStep{
			{
				Config: testAccResourceVSphereVmfsDatastoreConfigStaticSingle(),
				Check: resource.ComposeTestCheckFunc(
					testAccResourceVSphereVmfsDatastoreExists(true),
				),
			},
			{
				Config:      testAccResourceVSphereVmfsDatastoreConfigStaticSingle(),
				ImportState: true,
				ImportStateIdFunc: func(s *terraform.State) (string, error) {
					vars, err := testClientVariablesForResource(s, "vsphere_vmfs_datastore.datastore")
					if err != nil {
						return "", err
					}

					return fmt.Sprintf("%s:%s", vars.resourceID, vars.resourceAttributes["host_system_id"]), nil
				},
				ResourceName:            "vsphere_vmfs_datastore.datastore",
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"multiple_host_access"},
			},
		},
	})
}

func TestAccResourceVSphereVmfsDatastore_multiDisk(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
			testAccResourceVSphereVmfsDatastorePreCheck(t)
		},
		Providers:    testAccProviders,
		CheckDestroy: testAccResourceVSphereVmfsDatastoreExists(false),
		Steps: []resource.TestStep{
			{
				Config: testAccResourceVSphereVmfsDatastoreConfigStaticMulti(),
				Check: resource.ComposeTestCheckFunc(
					testAccResourceVSphereVmfsDatastoreExists(true),
				),
			},
		},
	})
}

func TestAccResourceVSphereVmfsDatastore_discoveryViaDatasource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
			testAccResourceVSphereVmfsDatastorePreCheck(t)
		},
		Providers:    testAccProviders,
		CheckDestroy: testAccResourceVSphereVmfsDatastoreExists(false),
		Steps: []resource.TestStep{
			{
				Config: testAccResourceVSphereVmfsDatastoreConfigDiscoverDatasource(),
				Check: resource.ComposeTestCheckFunc(
					testAccResourceVSphereVmfsDatastoreExists(true),
				),
			},
		},
	})
}

func TestAccResourceVSphereVmfsDatastore_addDisksThroughUpdate(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
			testAccResourceVSphereVmfsDatastorePreCheck(t)
		},
		Providers:    testAccProviders,
		CheckDestroy: testAccResourceVSphereVmfsDatastoreExists(false),
		Steps: []resource.TestStep{
			{
				Config: testAccResourceVSphereVmfsDatastoreConfigStaticSingle(),
				Check: resource.ComposeTestCheckFunc(
					testAccResourceVSphereVmfsDatastoreExists(true),
				),
			},
			{
				Config: testAccResourceVSphereVmfsDatastoreConfigStaticMulti(),
				Check: resource.ComposeTestCheckFunc(
					testAccResourceVSphereVmfsDatastoreExists(true),
				),
			},
		},
	})
}

func TestAccResourceVSphereVmfsDatastore_renameDatastore(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
			testAccResourceVSphereVmfsDatastorePreCheck(t)
		},
		Providers:    testAccProviders,
		CheckDestroy: testAccResourceVSphereVmfsDatastoreExists(false),
		Steps: []resource.TestStep{
			{
				Config: testAccResourceVSphereVmfsDatastoreConfigStaticSingle(),
				Check: resource.ComposeTestCheckFunc(
					testAccResourceVSphereVmfsDatastoreExists(true),
				),
			},
			{
				Config: testAccResourceVSphereVmfsDatastoreConfigStaticSingleAltName(),
				Check: resource.ComposeTestCheckFunc(
					testAccResourceVSphereVmfsDatastoreExists(true),
					testAccResourceVSphereVmfsDatastoreHasName("terraform-test-renamed"),
				),
			},
		},
	})
}

func TestAccResourceVSphereVmfsDatastore_withFolder(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
			testAccResourceVSphereVmfsDatastorePreCheck(t)
			// NOTE: This test can't run on ESXi without giving a "dangling
			// resource" error during testing - "move to folder after" hits the
			// error on the same path of the call stack that triggers an error in
			// both create and update and should provide adequate coverage
			// barring manual testing.
			testAccSkipIfEsxi(t)
		},
		Providers:    testAccProviders,
		CheckDestroy: testAccResourceVSphereVmfsDatastoreExists(false),
		Steps: []resource.TestStep{
			{
				Config: testAccResourceVSphereVmfsDatastoreConfigStaticSingleFolder(),
				Check: resource.ComposeTestCheckFunc(
					testAccResourceVSphereVmfsDatastoreExists(true),
					testAccResourceVSphereVmfsDatastoreMatchInventoryPath(os.Getenv("TF_VAR_VSPHERE_DS_FOLDER")),
				),
			},
		},
	})
}

func TestAccResourceVSphereVmfsDatastore_moveToFolderAfter(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
			testAccResourceVSphereVmfsDatastorePreCheck(t)
		},
		Providers:    testAccProviders,
		CheckDestroy: testAccResourceVSphereVmfsDatastoreExists(false),
		Steps: []resource.TestStep{
			{
				Config: testAccResourceVSphereVmfsDatastoreConfigStaticSingle(),
				Check: resource.ComposeTestCheckFunc(
					testAccResourceVSphereVmfsDatastoreExists(true),
				),
			},
			{
				Config:      testAccResourceVSphereVmfsDatastoreConfigStaticSingleFolder(),
				ExpectError: expectErrorIfNotVirtualCenter(),
				Check: resource.ComposeTestCheckFunc(
					testAccResourceVSphereVmfsDatastoreExists(true),
					testAccResourceVSphereVmfsDatastoreMatchInventoryPath(os.Getenv("TF_VAR_VSPHERE_DS_FOLDER")),
				),
			},
		},
	})
}

func TestAccResourceVSphereVmfsDatastore_withDatastoreCluster(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
			testAccResourceVSphereVmfsDatastorePreCheck(t)
			testAccSkipIfEsxi(t)
		},
		Providers:    testAccProviders,
		CheckDestroy: testAccResourceVSphereVmfsDatastoreExists(false),
		Steps: []resource.TestStep{
			{
				Config: testAccResourceVSphereVmfsDatastoreConfigDatastoreCluster(),
				Check: resource.ComposeTestCheckFunc(
					testAccResourceVSphereVmfsDatastoreExists(true),
					testAccResourceVSphereVmfsDatastoreMatchInventoryPath(testAccResourceVSphereDatastoreClusterNameStandard),
				),
			},
		},
	})
}

func TestAccResourceVSphereVmfsDatastore_moveToDatastoreClusterAfter(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
			testAccResourceVSphereVmfsDatastorePreCheck(t)
		},
		Providers:    testAccProviders,
		CheckDestroy: testAccResourceVSphereVmfsDatastoreExists(false),
		Steps: []resource.TestStep{
			{
				Config: testAccResourceVSphereVmfsDatastoreConfigStaticSingle(),
				Check: resource.ComposeTestCheckFunc(
					testAccResourceVSphereVmfsDatastoreExists(true),
				),
			},
			{
				Config: testAccResourceVSphereVmfsDatastoreConfigDatastoreCluster(),
				Check: resource.ComposeTestCheckFunc(
					testAccResourceVSphereVmfsDatastoreExists(true),
					testAccResourceVSphereVmfsDatastoreMatchInventoryPath(testAccResourceVSphereDatastoreClusterNameStandard),
				),
			},
		},
	})
}

func TestAccResourceVSphereVmfsDatastore_singleTag(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
			testAccResourceVSphereVmfsDatastorePreCheck(t)
		},
		Providers:    testAccProviders,
		CheckDestroy: testAccResourceVSphereVmfsDatastoreExists(false),
		Steps: []resource.TestStep{
			{
				Config: testAccResourceVSphereVmfsDatastoreConfigTags(),
				Check: resource.ComposeTestCheckFunc(
					testAccResourceVSphereVmfsDatastoreExists(true),
					testAccResourceVSphereDatastoreCheckTags("vsphere_vmfs_datastore.datastore", "terraform-test-tag"),
				),
			},
		},
	})
}

func TestAccResourceVSphereVmfsDatastore_modifyTags(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
			testAccResourceVSphereVmfsDatastorePreCheck(t)
		},
		Providers:    testAccProviders,
		CheckDestroy: testAccResourceVSphereVmfsDatastoreExists(false),
		Steps: []resource.TestStep{
			{
				Config: testAccResourceVSphereVmfsDatastoreConfigTags(),
				Check: resource.ComposeTestCheckFunc(
					testAccResourceVSphereVmfsDatastoreExists(true),
					testAccResourceVSphereDatastoreCheckTags("vsphere_vmfs_datastore.datastore", "terraform-test-tag"),
				),
			},
			{
				Config: testAccResourceVSphereVmfsDatastoreConfigMultiTags(),
				Check: resource.ComposeTestCheckFunc(
					testAccResourceVSphereVmfsDatastoreExists(true),
					testAccResourceVSphereDatastoreCheckTags("vsphere_vmfs_datastore.datastore", "terraform-test-tags-alt"),
				),
			},
		},
	})
}

func TestAccResourceVSphereVmfsDatastore_badDiskEntry(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
			testAccResourceVSphereVmfsDatastorePreCheck(t)
		},
		Providers:    testAccProviders,
		CheckDestroy: testAccResourceVSphereVmfsDatastoreExists(false),
		Steps: []resource.TestStep{
			{
				Config:      testAccResourceVSphereVmfsDatastoreConfigBadDisk(),
				ExpectError: regexp.MustCompile("empty entry"),
				PlanOnly:    true,
			},
			{
				Config: testAccResourceVSphereEmpty,
				Check:  resource.ComposeTestCheckFunc(),
			},
		},
	})
}

func TestAccResourceVSphereVmfsDatastore_duplicateDiskEntry(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
			testAccResourceVSphereVmfsDatastorePreCheck(t)
		},
		Providers:    testAccProviders,
		CheckDestroy: testAccResourceVSphereVmfsDatastoreExists(false),
		Steps: []resource.TestStep{
			{
				Config:      testAccResourceVSphereVmfsDatastoreConfigDuplicateDisk(),
				ExpectError: regexp.MustCompile("duplicate name"),
				PlanOnly:    true,
			},
			{
				Config: testAccResourceVSphereEmpty,
				Check:  resource.ComposeTestCheckFunc(),
			},
		},
	})
}

func TestAccResourceVSphereVmfsDatastore_singleCustomAttribute(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
			testAccResourceVSphereVmfsDatastorePreCheck(t)
		},
		Providers:    testAccProviders,
		CheckDestroy: testAccResourceVSphereVmfsDatastoreExists(false),
		Steps: []resource.TestStep{
			{
				Config: testAccResourceVSphereVmfsDatastoreConfigCustomAttributes(),
				Check: resource.ComposeTestCheckFunc(
					testAccResourceVSphereVmfsDatastoreExists(true),
					testAccResourceVSphereVmfsDatastoreHasCustomAttributes(),
				),
			},
		},
	})
}

func TestAccResourceVSphereVmfsDatastore_multiCustomAttribute(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
			testAccResourceVSphereVmfsDatastorePreCheck(t)
		},
		Providers:    testAccProviders,
		CheckDestroy: testAccResourceVSphereVmfsDatastoreExists(false),
		Steps: []resource.TestStep{
			{
				Config: testAccResourceVSphereVmfsDatastoreConfigCustomAttributes(),
				Check: resource.ComposeTestCheckFunc(
					testAccResourceVSphereVmfsDatastoreExists(true),
					testAccResourceVSphereVmfsDatastoreHasCustomAttributes(),
				),
			},
			{
				Config: testAccResourceVSphereVmfsDatastoreConfigMultiCustomAttributes(),
				Check: resource.ComposeTestCheckFunc(
					testAccResourceVSphereVmfsDatastoreExists(true),
					testAccResourceVSphereVmfsDatastoreHasCustomAttributes(),
				),
			},
		},
	})
}

func testAccResourceVSphereVmfsDatastorePreCheck(t *testing.T) {
	if os.Getenv("TF_VAR_VSPHERE_NFS_DS_NAME") == "" {
		t.Skip("set TF_VAR_VSPHERE_ESXI_HOST to run vsphere_vmfs_disks acceptance tests")
	}
	if os.Getenv("TF_VAR_VSPHERE_DS_VMFS_DISK0") == "" {
		t.Skip("set TF_VAR_VSPHERE_DS_VMFS_DISK0 to run vsphere_vmfs_datastore acceptance tests")
	}
	if os.Getenv("TF_VAR_VSPHERE_DS_VMFS_DISK1") == "" {
		t.Skip("set TF_VAR_VSPHERE_DS_VMFS_DISK1 to run vsphere_vmfs_datastore acceptance tests")
	}
	if os.Getenv("TF_VAR_VSPHERE_DS_VMFS_DISK2") == "" {
		t.Skip("set TF_VAR_VSPHERE_DS_VMFS_DISK2 to run vsphere_vmfs_datastore acceptance tests")
	}
	if os.Getenv("TF_VAR_VSPHERE_VMFS_REGEXP") == "" {
		t.Skip("set TF_VAR_VSPHERE_VMFS_REGEXP to run vsphere_vmfs_datastore acceptance tests")
	}
	if os.Getenv("TF_VAR_VSPHERE_DS_FOLDER") == "" {
		t.Skip("set TF_VAR_VSPHERE_DS_FOLDER to run vsphere_vmfs_datastore acceptance tests")
	}
}

func testAccResourceVSphereVmfsDatastoreExists(expected bool) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		ds, err := testGetDatastore(s, "vsphere_vmfs_datastore.datastore")
		if err != nil {
			missingState, _ := regexp.MatchString("not found in state", err.Error())
			if viapi.IsManagedObjectNotFoundError(err) && !expected || missingState && !expected {
				// Expected missing
				return nil
			}
			return err
		}
		if !expected {
			return fmt.Errorf("expected datastore %q to be missing", ds.Reference().Value)
		}
		return nil
	}
}

func testAccResourceVSphereVmfsDatastoreHasName(expected string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		ds, err := testGetDatastore(s, "vsphere_vmfs_datastore.datastore")
		if err != nil {
			return err
		}

		props, err := datastore.Properties(ds)
		if err != nil {
			return err
		}

		actual := props.Summary.Name
		if expected != actual {
			return fmt.Errorf("expected datastore name to be %s, got %s", expected, actual)
		}
		return nil
	}
}

func testAccResourceVSphereVmfsDatastoreMatchInventoryPath(expected string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		ds, err := testGetDatastore(s, "vsphere_vmfs_datastore.datastore")
		if err != nil {
			return err
		}

		expected, err = folder.RootPathParticleDatastore.PathFromNewRoot(ds.InventoryPath, folder.RootPathParticleDatastore, expected)
		actual := path.Dir(ds.InventoryPath)
		if err != nil {
			return fmt.Errorf("bad: %s", err)
		}
		if expected != actual {
			return fmt.Errorf("expected path to be %s, got %s", expected, actual)
		}
		return nil
	}
}

func testAccResourceVSphereVmfsDatastoreHasCustomAttributes() resource.TestCheckFunc {
	return func(s *terraform.State) error {
		props, err := testGetDatastoreProperties(s, "vmfs", "datastore")
		if err != nil {
			return err
		}
		return testResourceHasCustomAttributeValues(s, "vsphere_vmfs_datastore", "datastore", props.Entity())
	}
}

func testAccResourceVSphereVmfsDatastoreConfigStaticSingle() string {
	return fmt.Sprintf(`
variable "disk0" {
  type    = "string"
  default = "%s"
}

data "vsphere_datacenter" "datacenter" {
  name = "%s"
}

data "vsphere_host" "esxi_host" {
  name          = "%s"
  datacenter_id = "${data.vsphere_datacenter.datacenter.id}"
}

resource "vsphere_vmfs_datastore" "datastore" {
  name           = "terraform-test"
  host_system_id = "${data.vsphere_host.esxi_host.id}"

  disks = [
    "${var.disk0}",
  ]
}
`, os.Getenv("TF_VAR_VSPHERE_DS_VMFS_DISK0"), os.Getenv("TF_VAR_VSPHERE_DATACENTER"), os.Getenv("TF_VAR_VSPHERE_NFS_DS_NAME"))
}

func testAccResourceVSphereVmfsDatastoreConfigStaticSingleAltName() string {
	return fmt.Sprintf(`
variable "disk0" {
  type    = "string"
  default = "%s"
}

data "vsphere_datacenter" "datacenter" {
  name = "%s"
}

data "vsphere_host" "esxi_host" {
  name          = "%s"
  datacenter_id = "${data.vsphere_datacenter.datacenter.id}"
}

resource "vsphere_vmfs_datastore" "datastore" {
  name           = "terraform-test-renamed"
  host_system_id = "${data.vsphere_host.esxi_host.id}"

  disks = [
    "${var.disk0}",
  ]
}
`, os.Getenv("TF_VAR_VSPHERE_DS_VMFS_DISK0"), os.Getenv("TF_VAR_VSPHERE_DATACENTER"), os.Getenv("TF_VAR_VSPHERE_NFS_DS_NAME"))
}

func testAccResourceVSphereVmfsDatastoreConfigStaticMulti() string {
	return fmt.Sprintf(`
variable "disk0" {
  type    = "string"
  default = "%s"
}

variable "disk1" {
  type    = "string"
  default = "%s"
}

variable "disk2" {
  type    = "string"
  default = "%s"
}

data "vsphere_datacenter" "datacenter" {
  name = "%s"
}

data "vsphere_host" "esxi_host" {
  name          = "%s"
  datacenter_id = "${data.vsphere_datacenter.datacenter.id}"
}

resource "vsphere_vmfs_datastore" "datastore" {
  name           = "terraform-test"
  host_system_id = "${data.vsphere_host.esxi_host.id}"

  disks = [
    "${var.disk0}",
    "${var.disk1}",
    "${var.disk2}",
  ]
}
`, os.Getenv("TF_VAR_VSPHERE_DS_VMFS_DISK0"), os.Getenv("TF_VAR_VSPHERE_DS_VMFS_DISK1"), os.Getenv("TF_VAR_VSPHERE_DS_VMFS_DISK2"), os.Getenv("TF_VAR_VSPHERE_DATACENTER"), os.Getenv("TF_VAR_VSPHERE_NFS_DS_NAME"))
}

func testAccResourceVSphereVmfsDatastoreConfigDiscoverDatasource() string {
	return fmt.Sprintf(`
variable "regexp" {
  type    = "string"
  default = "%s"
}

data "vsphere_datacenter" "datacenter" {
  name = "%s"
}

data "vsphere_host" "esxi_host" {
  name          = "%s"
  datacenter_id = "${data.vsphere_datacenter.datacenter.id}"
}

data "vsphere_vmfs_disks" "available" {
  host_system_id = "${data.vsphere_host.esxi_host.id}"
  rescan         = true
  filter         = "${var.regexp}"
}

resource "vsphere_vmfs_datastore" "datastore" {
  name           = "terraform-test"
  host_system_id = "${data.vsphere_host.esxi_host.id}"

  disks = "${data.vsphere_vmfs_disks.available.disks}"
}
`, os.Getenv("TF_VAR_VSPHERE_VMFS_REGEXP"), os.Getenv("TF_VAR_VSPHERE_DATACENTER"), os.Getenv("TF_VAR_VSPHERE_NFS_DS_NAME"))
}

func testAccResourceVSphereVmfsDatastoreConfigStaticSingleFolder() string {
	return fmt.Sprintf(`
variable "disk0" {
  type    = "string"
  default = "%s"
}

variable "folder" {
  type    = "string"
  default = "%s"
}

data "vsphere_datacenter" "datacenter" {
  name = "%s"
}

data "vsphere_host" "esxi_host" {
  name          = "%s"
  datacenter_id = "${data.vsphere_datacenter.datacenter.id}"
}

resource "vsphere_vmfs_datastore" "datastore" {
  name           = "terraform-test"
  host_system_id = "${data.vsphere_host.esxi_host.id}"
  folder         = "${var.folder}"

  disks = [
    "${var.disk0}",
  ]
}
`, os.Getenv("TF_VAR_VSPHERE_DS_VMFS_DISK0"), os.Getenv("TF_VAR_VSPHERE_DS_FOLDER"), os.Getenv("TF_VAR_VSPHERE_DATACENTER"), os.Getenv("TF_VAR_VSPHERE_NFS_DS_NAME"))
}

func testAccResourceVSphereVmfsDatastoreConfigTags() string {
	return fmt.Sprintf(`
variable "disk0" {
  type    = "string"
  default = "%s"
}

data "vsphere_datacenter" "datacenter" {
  name = "%s"
}

data "vsphere_host" "esxi_host" {
  name          = "%s"
  datacenter_id = "${data.vsphere_datacenter.datacenter.id}"
}

resource "vsphere_tag_category" "terraform-test-category" {
  name        = "terraform-test-tag-category"
  cardinality = "MULTIPLE"

  associable_types = [
    "Datastore",
  ]
}

resource "vsphere_tag" "terraform-test-tag" {
  name        = "terraform-test-tag"
  category_id = "${vsphere_tag_category.terraform-test-category.id}"
}

resource "vsphere_vmfs_datastore" "datastore" {
  name           = "terraform-test"
  host_system_id = "${data.vsphere_host.esxi_host.id}"

  disks = [
    "${var.disk0}",
  ]

  tags = ["${vsphere_tag.terraform-test-tag.id}"]
}
`, os.Getenv("TF_VAR_VSPHERE_DS_VMFS_DISK0"), os.Getenv("TF_VAR_VSPHERE_DATACENTER"), os.Getenv("TF_VAR_VSPHERE_NFS_DS_NAME"))
}

func testAccResourceVSphereVmfsDatastoreConfigMultiTags() string {
	return fmt.Sprintf(`
variable "disk0" {
  type    = "string"
  default = "%s"
}

variable "extra_tags" {
  default = [
    "terraform-test-thing1",
    "terraform-test-thing2",
  ]
}

data "vsphere_datacenter" "datacenter" {
  name = "%s"
}

data "vsphere_host" "esxi_host" {
  name          = "%s"
  datacenter_id = "${data.vsphere_datacenter.datacenter.id}"
}

resource "vsphere_tag_category" "terraform-test-category" {
  name        = "terraform-test-tag-category"
  cardinality = "MULTIPLE"

  associable_types = [
    "Datastore",
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

resource "vsphere_vmfs_datastore" "datastore" {
  name           = "terraform-test"
  host_system_id = "${data.vsphere_host.esxi_host.id}"

  disks = [
    "${var.disk0}",
  ]

  tags = "${vsphere_tag.terraform-test-tags-alt.*.id}"
}
`, os.Getenv("TF_VAR_VSPHERE_DS_VMFS_DISK0"), os.Getenv("TF_VAR_VSPHERE_DATACENTER"), os.Getenv("TF_VAR_VSPHERE_NFS_DS_NAME"))
}

func testAccResourceVSphereVmfsDatastoreConfigBadDisk() string {
	return fmt.Sprintf(`
variable "disk0" {
  type    = "string"
  default = "%s"
}

variable "disk1" {
  type    = "string"
  default = "%s"
}

data "vsphere_datacenter" "datacenter" {
  name = "%s"
}

data "vsphere_host" "esxi_host" {
  name          = "%s"
  datacenter_id = "${data.vsphere_datacenter.datacenter.id}"
}

resource "vsphere_vmfs_datastore" "datastore" {
  name           = "terraform-test"
  host_system_id = "${data.vsphere_host.esxi_host.id}"

  disks = [
    "${var.disk0}",
    "${var.disk1}",
    "",
  ]
}
`, os.Getenv("TF_VAR_VSPHERE_DS_VMFS_DISK0"), os.Getenv("TF_VAR_VSPHERE_DS_VMFS_DISK1"), os.Getenv("TF_VAR_VSPHERE_DATACENTER"), os.Getenv("TF_VAR_VSPHERE_NFS_DS_NAME"))
}

func testAccResourceVSphereVmfsDatastoreConfigDuplicateDisk() string {
	return fmt.Sprintf(`
variable "disk0" {
  type    = "string"
  default = "%s"
}

variable "disk1" {
  type    = "string"
  default = "%s"
}

data "vsphere_datacenter" "datacenter" {
  name = "%s"
}

data "vsphere_host" "esxi_host" {
  name          = "%s"
  datacenter_id = "${data.vsphere_datacenter.datacenter.id}"
}

resource "vsphere_vmfs_datastore" "datastore" {
  name           = "terraform-test"
  host_system_id = "${data.vsphere_host.esxi_host.id}"

  disks = [
    "${var.disk0}",
    "${var.disk1}",
    "${var.disk1}",
  ]
}
`, os.Getenv("TF_VAR_VSPHERE_DS_VMFS_DISK0"), os.Getenv("TF_VAR_VSPHERE_DS_VMFS_DISK1"), os.Getenv("TF_VAR_VSPHERE_DATACENTER"), os.Getenv("TF_VAR_VSPHERE_NFS_DS_NAME"))
}

func testAccResourceVSphereVmfsDatastoreConfigCustomAttributes() string {
	return fmt.Sprintf(`
variable "disk0" {
  type    = "string"
  default = "%s"
}

data "vsphere_datacenter" "datacenter" {
  name = "%s"
}

data "vsphere_host" "esxi_host" {
  name          = "%s"
  datacenter_id = "${data.vsphere_datacenter.datacenter.id}"
}

resource "vsphere_custom_attribute" "terraform-test-attribute" {
  name                = "terraform-test-attribute"
  managed_object_type = "Datastore"
}

locals {
  vmfs_attrs = {
    "${vsphere_custom_attribute.terraform-test-attribute.id}" = "value"
  }
}

resource "vsphere_vmfs_datastore" "datastore" {
  name           = "terraform-test"
  host_system_id = "${data.vsphere_host.esxi_host.id}"

  disks = [
    "${var.disk0}",
  ]

  custom_attributes = "${local.vmfs_attrs}"
}
`, os.Getenv("TF_VAR_VSPHERE_DS_VMFS_DISK0"), os.Getenv("TF_VAR_VSPHERE_DATACENTER"), os.Getenv("TF_VAR_VSPHERE_NFS_DS_NAME"))
}

func testAccResourceVSphereVmfsDatastoreConfigMultiCustomAttributes() string {
	return fmt.Sprintf(`
variable "disk0" {
  type    = "string"
  default = "%s"
}

data "vsphere_datacenter" "datacenter" {
  name = "%s"
}

data "vsphere_host" "esxi_host" {
  name          = "%s"
  datacenter_id = "${data.vsphere_datacenter.datacenter.id}"
}

resource "vsphere_custom_attribute" "terraform-test-attribute" {
  name                = "terraform-test-attribute"
  managed_object_type = "Datastore"
}

resource "vsphere_custom_attribute" "terraform-test-attribute-2" {
  name                = "terraform-test-attribute-2"
  managed_object_type = "Datastore"
}

locals {
  vmfs_attrs = {
    "${vsphere_custom_attribute.terraform-test-attribute.id}" = "value"
    "${vsphere_custom_attribute.terraform-test-attribute-2.id}" = "value-2"
  }
}

resource "vsphere_vmfs_datastore" "datastore" {
  name           = "terraform-test"
  host_system_id = "${data.vsphere_host.esxi_host.id}"

  disks = [
    "${var.disk0}",
  ]

  custom_attributes = "${local.vmfs_attrs}"
}
`, os.Getenv("TF_VAR_VSPHERE_DS_VMFS_DISK0"), os.Getenv("TF_VAR_VSPHERE_DATACENTER"), os.Getenv("TF_VAR_VSPHERE_NFS_DS_NAME"))
}

func testAccResourceVSphereVmfsDatastoreConfigDatastoreCluster() string {
	return fmt.Sprintf(`
variable "disk0" {
  type    = "string"
  default = "%s"
}

variable "folder" {
  type    = "string"
  default = "%s"
}

data "vsphere_datacenter" "datacenter" {
  name = "%s"
}

data "vsphere_host" "esxi_host" {
  name          = "%s"
  datacenter_id = "${data.vsphere_datacenter.datacenter.id}"
}

resource "vsphere_datastore_cluster" "datastore_cluster" {
  name          = "terraform-datastore-cluster-test"
  datacenter_id = "${data.vsphere_datacenter.datacenter.id}"
}

resource "vsphere_vmfs_datastore" "datastore" {
  name                 = "terraform-test"
  host_system_id       = "${data.vsphere_host.esxi_host.id}"
  datastore_cluster_id = "${vsphere_datastore_cluster.datastore_cluster.id}"

  disks = [
    "${var.disk0}",
  ]
}
`, os.Getenv("TF_VAR_VSPHERE_DS_VMFS_DISK0"), os.Getenv("TF_VAR_VSPHERE_DS_FOLDER"), os.Getenv("TF_VAR_VSPHERE_DATACENTER"), os.Getenv("TF_VAR_VSPHERE_NFS_DS_NAME"))
}
