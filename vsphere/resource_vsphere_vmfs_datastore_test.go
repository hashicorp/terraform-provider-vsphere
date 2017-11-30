package vsphere

import (
	"fmt"
	"os"
	"path"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
	"github.com/terraform-providers/terraform-provider-vsphere/vsphere/internal/helper/datastore"
	"github.com/terraform-providers/terraform-provider-vsphere/vsphere/internal/helper/folder"
	"github.com/terraform-providers/terraform-provider-vsphere/vsphere/internal/helper/viapi"
)

func TestAccResourceVSphereVmfsDatastore(t *testing.T) {
	var tp *testing.T
	testAccResourceVSphereVmfsDatastoreCases := []struct {
		name     string
		testCase resource.TestCase
	}{
		{
			"basic",
			resource.TestCase{
				PreCheck: func() {
					testAccPreCheck(tp)
					testAccResourceVSphereVmfsDatastorePreCheck(tp)
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
				},
			},
		},
		{
			"multi-disk",
			resource.TestCase{
				PreCheck: func() {
					testAccPreCheck(tp)
					testAccResourceVSphereVmfsDatastorePreCheck(tp)
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
			},
		},
		{
			"discovery via data source",
			resource.TestCase{
				PreCheck: func() {
					testAccPreCheck(tp)
					testAccResourceVSphereVmfsDatastorePreCheck(tp)
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
			},
		},
		{
			"add disks through update",
			resource.TestCase{
				PreCheck: func() {
					testAccPreCheck(tp)
					testAccResourceVSphereVmfsDatastorePreCheck(tp)
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
			},
		},
		{
			"rename datastore",
			resource.TestCase{
				PreCheck: func() {
					testAccPreCheck(tp)
					testAccResourceVSphereVmfsDatastorePreCheck(tp)
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
			},
		},
		{
			"with folder",
			resource.TestCase{
				PreCheck: func() {
					testAccPreCheck(tp)
					testAccResourceVSphereVmfsDatastorePreCheck(tp)
					// NOTE: This test can't run on ESXi without giving a "dangling
					// resource" error during testing - "move to folder after" hits the
					// error on the same path of the call stack that triggers an error in
					// both create and update and should provide adequate coverage
					// barring manual testing.
					testAccSkipIfEsxi(tp)
				},
				Providers:    testAccProviders,
				CheckDestroy: testAccResourceVSphereVmfsDatastoreExists(false),
				Steps: []resource.TestStep{
					{
						Config: testAccResourceVSphereVmfsDatastoreConfigStaticSingleFolder(),
						Check: resource.ComposeTestCheckFunc(
							testAccResourceVSphereVmfsDatastoreExists(true),
							testAccResourceVSphereVmfsDatastoreMatchInventoryPath(os.Getenv("VSPHERE_DS_FOLDER")),
						),
					},
				},
			},
		},
		{
			"move to folder after",
			resource.TestCase{
				PreCheck: func() {
					testAccPreCheck(tp)
					testAccResourceVSphereVmfsDatastorePreCheck(tp)
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
							testAccResourceVSphereVmfsDatastoreMatchInventoryPath(os.Getenv("VSPHERE_DS_FOLDER")),
						),
					},
				},
			},
		},
		{
			"single tag",
			resource.TestCase{
				PreCheck: func() {
					testAccPreCheck(tp)
					testAccResourceVSphereVmfsDatastorePreCheck(tp)
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
			},
		},
		{
			"modify tags",
			resource.TestCase{
				PreCheck: func() {
					testAccPreCheck(tp)
					testAccResourceVSphereVmfsDatastorePreCheck(tp)
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
			},
		},
		{
			"bad disk entry",
			resource.TestCase{
				PreCheck: func() {
					testAccPreCheck(tp)
					testAccResourceVSphereVmfsDatastorePreCheck(tp)
				},
				Providers: testAccProviders,
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
			},
		},
		{
			"duplicate disk entry",
			resource.TestCase{
				PreCheck: func() {
					testAccPreCheck(tp)
					testAccResourceVSphereVmfsDatastorePreCheck(tp)
				},
				Providers: testAccProviders,
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
			},
		},
		{
			"import",
			resource.TestCase{
				PreCheck: func() {
					testAccPreCheck(tp)
					testAccResourceVSphereVmfsDatastorePreCheck(tp)
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
						ResourceName:      "vsphere_vmfs_datastore.datastore",
						ImportStateVerify: true,
					},
				},
			},
		},
	}

	for _, tc := range testAccResourceVSphereVmfsDatastoreCases {
		t.Run(tc.name, func(t *testing.T) {
			tp = t
			resource.Test(t, tc.testCase)
		})
	}
}

func testAccResourceVSphereVmfsDatastorePreCheck(t *testing.T) {
	if os.Getenv("VSPHERE_ESXI_HOST") == "" {
		t.Skip("set VSPHERE_ESXI_HOST to run vsphere_vmfs_disks acceptance tests")
	}
	if os.Getenv("VSPHERE_DS_VMFS_DISK0") == "" {
		t.Skip("set VSPHERE_DS_VMFS_DISK0 to run vsphere_vmfs_datastore acceptance tests")
	}
	if os.Getenv("VSPHERE_DS_VMFS_DISK1") == "" {
		t.Skip("set VSPHERE_DS_VMFS_DISK1 to run vsphere_vmfs_datastore acceptance tests")
	}
	if os.Getenv("VSPHERE_DS_VMFS_DISK2") == "" {
		t.Skip("set VSPHERE_DS_VMFS_DISK2 to run vsphere_vmfs_datastore acceptance tests")
	}
	if os.Getenv("VSPHERE_VMFS_REGEXP") == "" {
		t.Skip("set VSPHERE_VMFS_REGEXP to run vsphere_vmfs_datastore acceptance tests")
	}
	if os.Getenv("VSPHERE_DS_FOLDER") == "" {
		t.Skip("set VSPHERE_DS_FOLDER to run vsphere_vmfs_datastore acceptance tests")
	}
}

func testAccResourceVSphereVmfsDatastoreExists(expected bool) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		ds, err := testGetDatastore(s, "vsphere_vmfs_datastore.datastore")
		if err != nil {
			if viapi.IsManagedObjectNotFoundError(err) && expected == false {
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

		expected, err := folder.RootPathParticleDatastore.PathFromNewRoot(ds.InventoryPath, folder.RootPathParticleDatastore, expected)
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
`, os.Getenv("VSPHERE_DS_VMFS_DISK0"), os.Getenv("VSPHERE_DATACENTER"), os.Getenv("VSPHERE_ESXI_HOST"))
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
`, os.Getenv("VSPHERE_DS_VMFS_DISK0"), os.Getenv("VSPHERE_DATACENTER"), os.Getenv("VSPHERE_ESXI_HOST"))
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
`, os.Getenv("VSPHERE_DS_VMFS_DISK0"), os.Getenv("VSPHERE_DS_VMFS_DISK1"), os.Getenv("VSPHERE_DS_VMFS_DISK2"), os.Getenv("VSPHERE_DATACENTER"), os.Getenv("VSPHERE_ESXI_HOST"))
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

  disks = ["${data.vsphere_vmfs_disks.available.disks}"]
}
`, os.Getenv("VSPHERE_VMFS_REGEXP"), os.Getenv("VSPHERE_DATACENTER"), os.Getenv("VSPHERE_ESXI_HOST"))
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
`, os.Getenv("VSPHERE_DS_VMFS_DISK0"), os.Getenv("VSPHERE_DS_FOLDER"), os.Getenv("VSPHERE_DATACENTER"), os.Getenv("VSPHERE_ESXI_HOST"))
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
`, os.Getenv("VSPHERE_DS_VMFS_DISK0"), os.Getenv("VSPHERE_DATACENTER"), os.Getenv("VSPHERE_ESXI_HOST"))
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

  tags = ["${vsphere_tag.terraform-test-tags-alt.*.id}"]
}
`, os.Getenv("VSPHERE_DS_VMFS_DISK0"), os.Getenv("VSPHERE_DATACENTER"), os.Getenv("VSPHERE_ESXI_HOST"))
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
`, os.Getenv("VSPHERE_DS_VMFS_DISK0"), os.Getenv("VSPHERE_DS_VMFS_DISK1"), os.Getenv("VSPHERE_DATACENTER"), os.Getenv("VSPHERE_ESXI_HOST"))
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
`, os.Getenv("VSPHERE_DS_VMFS_DISK0"), os.Getenv("VSPHERE_DS_VMFS_DISK1"), os.Getenv("VSPHERE_DATACENTER"), os.Getenv("VSPHERE_ESXI_HOST"))
}
