package vsphere

import (
	"fmt"
	"os"
	"path"
	"testing"

	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

func TestAccResourceVSphereNasDatastore(t *testing.T) {
	var tp *testing.T
	testAccResourceVSphereNasDatastoreCases := []struct {
		name     string
		testCase resource.TestCase
	}{
		{
			"basic",
			resource.TestCase{
				PreCheck: func() {
					testAccPreCheck(tp)
					testAccResourceVSphereNasDatastorePreCheck(tp)
				},
				Providers:    testAccProviders,
				CheckDestroy: testAccResourceVSphereNasDatastoreExists(false),
				Steps: []resource.TestStep{
					{
						Config: testAccResourceVSphereNasDatastoreConfigBasic(),
						Check: resource.ComposeTestCheckFunc(
							testAccResourceVSphereNasDatastoreExists(true),
						),
					},
				},
			},
		},
		{
			"multi-host",
			resource.TestCase{
				PreCheck: func() {
					testAccPreCheck(tp)
					testAccResourceVSphereNasDatastorePreCheck(tp)
					testAccSkipIfEsxi(tp)
				},
				Providers: testAccProviders,
				Steps: []resource.TestStep{
					{
						Config: testAccResourceVSphereNasDatastoreConfigMultiHost(),
						Check: resource.ComposeTestCheckFunc(
							testAccResourceVSphereNasDatastoreExists(true),
						),
					},
				},
			},
		},
		{
			"basic, then multi-host",
			resource.TestCase{
				PreCheck: func() {
					testAccPreCheck(tp)
					testAccResourceVSphereNasDatastorePreCheck(tp)
				},
				Providers: testAccProviders,
				Steps: []resource.TestStep{
					{
						Config: testAccResourceVSphereNasDatastoreConfigBasic(),
						Check: resource.ComposeTestCheckFunc(
							testAccResourceVSphereNasDatastoreExists(true),
						),
					},
					{
						Config:      testAccResourceVSphereNasDatastoreConfigMultiHost(),
						ExpectError: expectErrorIfNotVirtualCenter(),
						Check: resource.ComposeTestCheckFunc(
							testAccResourceVSphereNasDatastoreExists(true),
						),
					},
				},
			},
		},
		{
			"multi-host, then basic",
			resource.TestCase{
				PreCheck: func() {
					testAccPreCheck(tp)
					testAccResourceVSphereNasDatastorePreCheck(tp)
					testAccSkipIfEsxi(tp)
				},
				Providers: testAccProviders,
				Steps: []resource.TestStep{
					{
						Config: testAccResourceVSphereNasDatastoreConfigMultiHost(),
						Check: resource.ComposeTestCheckFunc(
							testAccResourceVSphereNasDatastoreExists(true),
						),
					},
					{
						Config: testAccResourceVSphereNasDatastoreConfigBasic(),
						Check: resource.ComposeTestCheckFunc(
							testAccResourceVSphereNasDatastoreExists(true),
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
					testAccResourceVSphereNasDatastorePreCheck(tp)
				},
				Providers:    testAccProviders,
				CheckDestroy: testAccResourceVSphereNasDatastoreExists(false),
				Steps: []resource.TestStep{
					{
						Config: testAccResourceVSphereNasDatastoreConfigBasic(),
						Check: resource.ComposeTestCheckFunc(
							testAccResourceVSphereNasDatastoreExists(true),
						),
					},
					{
						Config: testAccResourceVSphereNasDatastoreConfigBasicAltName(),
						Check: resource.ComposeTestCheckFunc(
							testAccResourceVSphereNasDatastoreExists(true),
							testAccResourceVSphereNasDatastoreHasName("terraform-test-nas-renamed"),
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
					testAccResourceVSphereNasDatastorePreCheck(tp)
					// NOTE: This test can't run on ESXi without giving a "dangling
					// resource" error during testing - "move to folder after" hits the
					// error on the same path of the call stack that triggers an error in
					// both create and update and should provide adequate coverage
					// barring manual testing.
					testAccSkipIfEsxi(tp)
				},
				Providers:    testAccProviders,
				CheckDestroy: testAccResourceVSphereNasDatastoreExists(false),
				Steps: []resource.TestStep{
					{
						Config: testAccResourceVSphereNasDatastoreConfigBasicFolder(),
						Check: resource.ComposeTestCheckFunc(
							testAccResourceVSphereNasDatastoreExists(true),
							testAccResourceVSphereNasDatastoreMatchInventoryPath(os.Getenv("VSPHERE_DS_FOLDER")),
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
					testAccResourceVSphereNasDatastorePreCheck(tp)
				},
				Providers:    testAccProviders,
				CheckDestroy: testAccResourceVSphereNasDatastoreExists(false),
				Steps: []resource.TestStep{
					{
						Config: testAccResourceVSphereNasDatastoreConfigBasic(),
						Check: resource.ComposeTestCheckFunc(
							testAccResourceVSphereNasDatastoreExists(true),
						),
					},
					{
						Config:      testAccResourceVSphereNasDatastoreConfigBasicFolder(),
						ExpectError: expectErrorIfNotVirtualCenter(),
						Check: resource.ComposeTestCheckFunc(
							testAccResourceVSphereNasDatastoreExists(true),
							testAccResourceVSphereNasDatastoreMatchInventoryPath(os.Getenv("VSPHERE_DS_FOLDER")),
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
					testAccResourceVSphereNasDatastorePreCheck(tp)
				},
				Providers:    testAccProviders,
				CheckDestroy: testAccResourceVSphereNasDatastoreExists(false),
				Steps: []resource.TestStep{
					{
						Config: testAccResourceVSphereNasDatastoreConfigBasicTags(),
						Check: resource.ComposeTestCheckFunc(
							testAccResourceVSphereNasDatastoreExists(true),
							testAccResourceVSphereDatastoreCheckTags("vsphere_nas_datastore.datastore", "terraform-test-tag"),
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
					testAccResourceVSphereNasDatastorePreCheck(tp)
				},
				Providers:    testAccProviders,
				CheckDestroy: testAccResourceVSphereNasDatastoreExists(false),
				Steps: []resource.TestStep{
					{
						Config: testAccResourceVSphereNasDatastoreConfigBasicTags(),
						Check: resource.ComposeTestCheckFunc(
							testAccResourceVSphereNasDatastoreExists(true),
							testAccResourceVSphereDatastoreCheckTags("vsphere_nas_datastore.datastore", "terraform-test-tag"),
						),
					},
					{
						Config: testAccResourceVSphereNasDatastoreConfigMultiTags(),
						Check: resource.ComposeTestCheckFunc(
							testAccResourceVSphereNasDatastoreExists(true),
							testAccResourceVSphereDatastoreCheckTags("vsphere_nas_datastore.datastore", "terraform-test-tags-alt"),
						),
					},
				},
			},
		},
		{
			"import",
			resource.TestCase{
				PreCheck: func() {
					testAccPreCheck(tp)
					testAccResourceVSphereNasDatastorePreCheck(tp)
				},
				Providers:    testAccProviders,
				CheckDestroy: testAccResourceVSphereNasDatastoreExists(false),
				Steps: []resource.TestStep{
					{
						Config: testAccResourceVSphereNasDatastoreConfigBasic(),
						Check: resource.ComposeTestCheckFunc(
							testAccResourceVSphereNasDatastoreExists(true),
						),
					},
					{
						Config:            testAccResourceVSphereNasDatastoreConfigBasic(),
						ImportState:       true,
						ResourceName:      "vsphere_nas_datastore.datastore",
						ImportStateVerify: true,
					},
				},
			},
		},
	}

	for _, tc := range testAccResourceVSphereNasDatastoreCases {
		t.Run(tc.name, func(t *testing.T) {
			tp = t
			resource.Test(t, tc.testCase)
		})
	}
}

func testAccResourceVSphereNasDatastorePreCheck(t *testing.T) {
	if os.Getenv("VSPHERE_ESXI_HOST") == "" {
		t.Skip("set VSPHERE_ESXI_HOST to run vsphere_vmfs_disks acceptance tests")
	}
	if os.Getenv("VSPHERE_ESXI_HOST2") == "" {
		t.Skip("set VSPHERE_ESXI_HOST2 to run vsphere_vmfs_disks acceptance tests")
	}
	if os.Getenv("VSPHERE_ESXI_HOST3") == "" {
		t.Skip("set VSPHERE_ESXI_HOST3 to run vsphere_vmfs_disks acceptance tests")
	}
	if os.Getenv("VSPHERE_NAS_HOST") == "" {
		t.Skip("set VSPHERE_NAS_HOST to run vsphere_nas_datastore acceptance tests")
	}
	if os.Getenv("VSPHERE_NFS_PATH") == "" {
		t.Skip("set VSPHERE_NFS_PATH to run vsphere_nas_datastore acceptance tests")
	}
	if os.Getenv("VSPHERE_DS_FOLDER") == "" {
		t.Skip("set VSPHERE_DS_FOLDER to run vsphere_nas_datastore acceptance tests")
	}
}

func testAccResourceVSphereNasDatastoreExists(expected bool) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		ds, err := testGetDatastore(s, "vsphere_nas_datastore.datastore")
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

func testAccResourceVSphereNasDatastoreHasName(expected string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		ds, err := testGetDatastore(s, "vsphere_nas_datastore.datastore")
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

func testAccResourceVSphereNasDatastoreMatchInventoryPath(expected string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		ds, err := testGetDatastore(s, "vsphere_nas_datastore.datastore")
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

func testAccResourceVSphereNasDatastoreConfigBasic() string {
	return fmt.Sprintf(`
variable "nfs_host" {
  type    = "string"
  default = "%s"
}

variable "nfs_path" {
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

resource "vsphere_nas_datastore" "datastore" {
  name            = "terraform-test-nas"
  host_system_ids = ["${data.vsphere_host.esxi_host.id}"]

  type         = "NFS"
  remote_hosts = ["${var.nfs_host}"]
  remote_path  = "${var.nfs_path}"
}
`, os.Getenv("VSPHERE_NAS_HOST"), os.Getenv("VSPHERE_NFS_PATH"), os.Getenv("VSPHERE_DATACENTER"), os.Getenv("VSPHERE_ESXI_HOST"))
}

func testAccResourceVSphereNasDatastoreConfigMultiHost() string {
	return fmt.Sprintf(`
variable "nfs_host" {
  type    = "string"
  default = "%s"
}

variable "nfs_path" {
  type    = "string"
  default = "%s"
}

variable "esxi_hosts" {
  default = [
    "%s",
    "%s",
    "%s",
  ]
}

data "vsphere_datacenter" "datacenter" {
  name = "%s"
}

data "vsphere_host" "esxi_host" {
  count         = "${length(var.esxi_hosts)}"
  name          = "${var.esxi_hosts[count.index]}"
  datacenter_id = "${data.vsphere_datacenter.datacenter.id}"
}

resource "vsphere_nas_datastore" "datastore" {
  name            = "terraform-test-nas"
  host_system_ids = ["${data.vsphere_host.esxi_host.*.id}"]

  type         = "NFS"
  remote_hosts = ["${var.nfs_host}"]
  remote_path  = "${var.nfs_path}"
}
`, os.Getenv("VSPHERE_NAS_HOST"), os.Getenv("VSPHERE_NFS_PATH"), os.Getenv("VSPHERE_ESXI_HOST"), os.Getenv("VSPHERE_ESXI_HOST2"), os.Getenv("VSPHERE_ESXI_HOST3"), os.Getenv("VSPHERE_DATACENTER"))
}

func testAccResourceVSphereNasDatastoreConfigBasicAltName() string {
	return fmt.Sprintf(`
variable "nfs_host" {
  type    = "string"
  default = "%s"
}

variable "nfs_path" {
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

resource "vsphere_nas_datastore" "datastore" {
  name            = "terraform-test-nas-renamed"
  host_system_ids = ["${data.vsphere_host.esxi_host.id}"]

  type         = "NFS"
  remote_hosts = ["${var.nfs_host}"]
  remote_path  = "${var.nfs_path}"
}
`, os.Getenv("VSPHERE_NAS_HOST"), os.Getenv("VSPHERE_NFS_PATH"), os.Getenv("VSPHERE_DATACENTER"), os.Getenv("VSPHERE_ESXI_HOST"))
}

func testAccResourceVSphereNasDatastoreConfigBasicFolder() string {
	return fmt.Sprintf(`
variable "nfs_host" {
  type    = "string"
  default = "%s"
}

variable "nfs_path" {
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

resource "vsphere_nas_datastore" "datastore" {
  name            = "terraform-test-nas"
  host_system_ids = ["${data.vsphere_host.esxi_host.id}"]
  folder          = "${var.folder}"

  type         = "NFS"
  remote_hosts = ["${var.nfs_host}"]
  remote_path  = "${var.nfs_path}"
}
`, os.Getenv("VSPHERE_NAS_HOST"), os.Getenv("VSPHERE_NFS_PATH"), os.Getenv("VSPHERE_DS_FOLDER"), os.Getenv("VSPHERE_DATACENTER"), os.Getenv("VSPHERE_ESXI_HOST"))
}

func testAccResourceVSphereNasDatastoreConfigBasicTags() string {
	return fmt.Sprintf(`
variable "nfs_host" {
  type    = "string"
  default = "%s"
}

variable "nfs_path" {
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

resource "vsphere_nas_datastore" "datastore" {
  name            = "terraform-test-nas"
  host_system_ids = ["${data.vsphere_host.esxi_host.id}"]

  type         = "NFS"
  remote_hosts = ["${var.nfs_host}"]
  remote_path  = "${var.nfs_path}"

  tags = ["${vsphere_tag.terraform-test-tag.id}"]
}
`, os.Getenv("VSPHERE_NAS_HOST"), os.Getenv("VSPHERE_NFS_PATH"), os.Getenv("VSPHERE_DATACENTER"), os.Getenv("VSPHERE_ESXI_HOST"))
}

func testAccResourceVSphereNasDatastoreConfigMultiTags() string {
	return fmt.Sprintf(`
variable "nfs_host" {
  type    = "string"
  default = "%s"
}

variable "nfs_path" {
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

resource "vsphere_nas_datastore" "datastore" {
  name            = "terraform-test-nas"
  host_system_ids = ["${data.vsphere_host.esxi_host.id}"]

  type         = "NFS"
  remote_hosts = ["${var.nfs_host}"]
  remote_path  = "${var.nfs_path}"

  tags = ["${vsphere_tag.terraform-test-tags-alt.*.id}"]
}
`, os.Getenv("VSPHERE_NAS_HOST"), os.Getenv("VSPHERE_NFS_PATH"), os.Getenv("VSPHERE_DATACENTER"), os.Getenv("VSPHERE_ESXI_HOST"))
}
