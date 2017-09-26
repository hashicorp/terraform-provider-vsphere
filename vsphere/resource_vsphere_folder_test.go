package vsphere

import (
	"fmt"
	"os"
	"testing"

	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

const testAccResourceVSphereFolderConfigExpectedName = "terraform-test-folder"
const testAccResourceVSphereFolderConfigExpectedAltName = "terraform-renamed-folder"
const testAccResourceVSphereFolderConfigExpectedParentName = "terraform-test-parent"

func TestAccResourceVSphereFolder(t *testing.T) {
	var tp *testing.T
	testAccResourceVSphereFolderCases := []struct {
		name     string
		testCase resource.TestCase
	}{
		{
			"basic (VM folder)",
			resource.TestCase{
				PreCheck: func() {
					testAccPreCheck(tp)
				},
				Providers:    testAccProviders,
				CheckDestroy: testAccResourceVSphereFolderExists(false),
				Steps: []resource.TestStep{
					{
						Config: testAccResourceVSphereFolderConfigBasic(
							testAccResourceVSphereFolderConfigExpectedName,
							vSphereFolderTypeVM,
						),
						Check: resource.ComposeTestCheckFunc(
							testAccResourceVSphereFolderExists(true),
							testAccResourceVSphereFolderHasName(testAccResourceVSphereFolderConfigExpectedName),
							testAccResourceVSphereFolderHasType(vSphereFolderTypeVM),
						),
					},
				},
			},
		},
		{
			"datastore folder",
			resource.TestCase{
				PreCheck: func() {
					testAccPreCheck(tp)
				},
				Providers:    testAccProviders,
				CheckDestroy: testAccResourceVSphereFolderExists(false),
				Steps: []resource.TestStep{
					{
						Config: testAccResourceVSphereFolderConfigBasic(
							testAccResourceVSphereFolderConfigExpectedName,
							vSphereFolderTypeDatastore,
						),
						Check: resource.ComposeTestCheckFunc(
							testAccResourceVSphereFolderExists(true),
							testAccResourceVSphereFolderHasName(testAccResourceVSphereFolderConfigExpectedName),
							testAccResourceVSphereFolderHasType(vSphereFolderTypeDatastore),
						),
					},
				},
			},
		},
		{
			"network folder",
			resource.TestCase{
				PreCheck: func() {
					testAccPreCheck(tp)
				},
				Providers:    testAccProviders,
				CheckDestroy: testAccResourceVSphereFolderExists(false),
				Steps: []resource.TestStep{
					{
						Config: testAccResourceVSphereFolderConfigBasic(
							testAccResourceVSphereFolderConfigExpectedName,
							vSphereFolderTypeNetwork,
						),
						Check: resource.ComposeTestCheckFunc(
							testAccResourceVSphereFolderExists(true),
							testAccResourceVSphereFolderHasName(testAccResourceVSphereFolderConfigExpectedName),
							testAccResourceVSphereFolderHasType(vSphereFolderTypeNetwork),
						),
					},
				},
			},
		},
		{
			"host folder",
			resource.TestCase{
				PreCheck: func() {
					testAccPreCheck(tp)
				},
				Providers:    testAccProviders,
				CheckDestroy: testAccResourceVSphereFolderExists(false),
				Steps: []resource.TestStep{
					{
						Config: testAccResourceVSphereFolderConfigBasic(
							testAccResourceVSphereFolderConfigExpectedName,
							vSphereFolderTypeHost,
						),
						Check: resource.ComposeTestCheckFunc(
							testAccResourceVSphereFolderExists(true),
							testAccResourceVSphereFolderHasName(testAccResourceVSphereFolderConfigExpectedName),
							testAccResourceVSphereFolderHasType(vSphereFolderTypeHost),
						),
					},
				},
			},
		},
		{
			"datacenter folder",
			resource.TestCase{
				PreCheck: func() {
					testAccPreCheck(tp)
				},
				Providers:    testAccProviders,
				CheckDestroy: testAccResourceVSphereFolderExists(false),
				Steps: []resource.TestStep{
					{
						Config: testAccResourceVSphereFolderConfigBasic(
							testAccResourceVSphereFolderConfigExpectedName,
							vSphereFolderTypeDatacenter,
						),
						Check: resource.ComposeTestCheckFunc(
							testAccResourceVSphereFolderExists(true),
							testAccResourceVSphereFolderHasName(testAccResourceVSphereFolderConfigExpectedName),
							testAccResourceVSphereFolderHasType(vSphereFolderTypeDatacenter),
						),
					},
				},
			},
		},
		{
			"rename",
			resource.TestCase{
				PreCheck: func() {
					testAccPreCheck(tp)
				},
				Providers:    testAccProviders,
				CheckDestroy: testAccResourceVSphereFolderExists(false),
				Steps: []resource.TestStep{
					{
						Config: testAccResourceVSphereFolderConfigBasic(
							testAccResourceVSphereFolderConfigExpectedName,
							vSphereFolderTypeVM,
						),
						Check: resource.ComposeTestCheckFunc(
							testAccResourceVSphereFolderExists(true),
							testAccResourceVSphereFolderHasName(testAccResourceVSphereFolderConfigExpectedName),
							testAccResourceVSphereFolderHasType(vSphereFolderTypeVM),
						),
					},
					{
						Config: testAccResourceVSphereFolderConfigBasic(
							testAccResourceVSphereFolderConfigExpectedAltName,
							vSphereFolderTypeVM,
						),
						Check: resource.ComposeTestCheckFunc(
							testAccResourceVSphereFolderExists(true),
							testAccResourceVSphereFolderHasName(testAccResourceVSphereFolderConfigExpectedAltName),
							testAccResourceVSphereFolderHasType(vSphereFolderTypeVM),
						),
					},
				},
			},
		},
		{
			"subfolder",
			resource.TestCase{
				PreCheck: func() {
					testAccPreCheck(tp)
				},
				Providers:    testAccProviders,
				CheckDestroy: testAccResourceVSphereFolderExists(false),
				Steps: []resource.TestStep{
					{
						Config: testAccResourceVSphereFolderConfigSubFolder(
							testAccResourceVSphereFolderConfigExpectedName,
							vSphereFolderTypeVM,
						),
						Check: resource.ComposeTestCheckFunc(
							testAccResourceVSphereFolderExists(true),
							testAccResourceVSphereFolderHasName(testAccResourceVSphereFolderConfigExpectedName),
							testAccResourceVSphereFolderHasType(vSphereFolderTypeVM),
							testAccResourceVSphereFolderHasParent(false, testAccResourceVSphereFolderConfigExpectedParentName),
						),
					},
				},
			},
		},
		{
			"move to subfolder",
			resource.TestCase{
				PreCheck: func() {
					testAccPreCheck(tp)
				},
				Providers:    testAccProviders,
				CheckDestroy: testAccResourceVSphereFolderExists(false),
				Steps: []resource.TestStep{
					{
						Config: testAccResourceVSphereFolderConfigBasic(
							testAccResourceVSphereFolderConfigExpectedName,
							vSphereFolderTypeVM,
						),
						Check: resource.ComposeTestCheckFunc(
							testAccResourceVSphereFolderExists(true),
							testAccResourceVSphereFolderHasName(testAccResourceVSphereFolderConfigExpectedName),
							testAccResourceVSphereFolderHasType(vSphereFolderTypeVM),
							testAccResourceVSphereFolderHasParent(true, "vm"),
						),
					},
					{
						Config: testAccResourceVSphereFolderConfigSubFolder(
							testAccResourceVSphereFolderConfigExpectedName,
							vSphereFolderTypeVM,
						),
						Check: resource.ComposeTestCheckFunc(
							testAccResourceVSphereFolderExists(true),
							testAccResourceVSphereFolderHasName(testAccResourceVSphereFolderConfigExpectedName),
							testAccResourceVSphereFolderHasType(vSphereFolderTypeVM),
							testAccResourceVSphereFolderHasParent(false, testAccResourceVSphereFolderConfigExpectedParentName),
						),
					},
				},
			},
		},
		{
			"tags",
			resource.TestCase{
				PreCheck: func() {
					testAccPreCheck(tp)
				},
				Providers:    testAccProviders,
				CheckDestroy: testAccResourceVSphereFolderExists(false),
				Steps: []resource.TestStep{
					{
						Config: testAccResourceVSphereFolderConfigTag(),
						Check: resource.ComposeTestCheckFunc(
							testAccResourceVSphereFolderExists(true),
							testAccResourceVSphereFolderHasName(testAccResourceVSphereFolderConfigExpectedName),
							testAccResourceVSphereFolderHasType(vSphereFolderTypeVM),
							testAccResourceVSphereFolderCheckTags("terraform-test-tag"),
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
				},
				Providers:    testAccProviders,
				CheckDestroy: testAccResourceVSphereFolderExists(false),
				Steps: []resource.TestStep{
					{
						Config: testAccResourceVSphereFolderConfigTag(),
						Check: resource.ComposeTestCheckFunc(
							testAccResourceVSphereFolderExists(true),
							testAccResourceVSphereFolderHasName(testAccResourceVSphereFolderConfigExpectedName),
							testAccResourceVSphereFolderHasType(vSphereFolderTypeVM),
							testAccResourceVSphereFolderCheckTags("terraform-test-tag"),
						),
					},
					{
						Config: testAccResourceVSphereFolderConfigMultiTag(),
						Check: resource.ComposeTestCheckFunc(
							testAccResourceVSphereFolderExists(true),
							testAccResourceVSphereFolderHasName(testAccResourceVSphereFolderConfigExpectedName),
							testAccResourceVSphereFolderHasType(vSphereFolderTypeVM),
							testAccResourceVSphereFolderCheckTags("terraform-test-tags-alt"),
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
				},
				Providers:    testAccProviders,
				CheckDestroy: testAccResourceVSphereFolderExists(false),
				Steps: []resource.TestStep{
					{
						Config: testAccResourceVSphereFolderConfigBasic(
							testAccResourceVSphereFolderConfigExpectedName,
							vSphereFolderTypeVM,
						),
						Check: resource.ComposeTestCheckFunc(
							testAccResourceVSphereFolderExists(true),
						),
					},
					{
						ResourceName:      "vsphere_folder.folder",
						ImportState:       true,
						ImportStateVerify: true,
						ImportStateIdFunc: func(s *terraform.State) (string, error) {
							folder, err := testGetFolder(s, "folder")
							if err != nil {
								return "", err
							}
							return folder.InventoryPath, nil
						},
						Config: testAccResourceVSphereFolderConfigBasic(
							testAccResourceVSphereFolderConfigExpectedName,
							vSphereFolderTypeVM,
						),
						Check: resource.ComposeTestCheckFunc(
							testAccResourceVSphereFolderExists(true),
							testAccResourceVSphereFolderHasName(testAccResourceVSphereFolderConfigExpectedName),
							testAccResourceVSphereFolderHasType(vSphereFolderTypeVM),
						),
					},
				},
			},
		},
	}

	for _, tc := range testAccResourceVSphereFolderCases {
		t.Run(tc.name, func(t *testing.T) {
			tp = t
			resource.Test(t, tc.testCase)
		})
	}
}

func testAccResourceVSphereFolderExists(expected bool) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		folder, err := testGetFolder(s, "folder")
		if err != nil {
			if isManagedObjectNotFoundError(err) && expected == false {
				// Expected missing
				return nil
			}
			return err
		}
		if !expected {
			return fmt.Errorf("expected folder %q to be missing", folder.Reference().Value)
		}
		return nil
	}
}

func testAccResourceVSphereFolderHasName(expected string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		props, err := testGetFolderProperties(s, "folder")
		if err != nil {
			return err
		}
		actual := props.Name
		if expected != actual {
			return fmt.Errorf("expected name to be %q, got %q", expected, actual)
		}
		return nil
	}
}

func testAccResourceVSphereFolderHasType(expected vSphereFolderType) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		folder, err := testGetFolder(s, "folder")
		if err != nil {
			return err
		}
		actual, err := findFolderType(folder)
		if err != nil {
			return err
		}
		if expected != actual {
			return fmt.Errorf("expected type to be %q, got %q", expected, actual)
		}
		return nil
	}
}

func testAccResourceVSphereFolderHasParent(expectedRoot bool, expectedName string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		props, err := testGetFolderProperties(s, "folder")
		if err != nil {
			return err
		}
		if props.Parent.Type != "Folder" && !expectedRoot {
			return fmt.Errorf("folder %q is a root folder", props.Name)
		}
		client := testAccProvider.Meta().(*VSphereClient).vimClient
		pfolder, err := folderFromID(client, props.Parent.Value)
		if err != nil {
			return err
		}
		pprops, err := folderProperties(pfolder)
		if err != nil {
			return err
		}

		actual := pprops.Name
		if expectedName != actual {
			return fmt.Errorf("expected parent folder name to be %q, got %q", expectedName, actual)
		}
		return nil
	}
}

// testAccResourceVSphereFolderCheckTags is a check to ensure that any
// tags that have been created with the supplied resource name have been
// attached to the folder.
func testAccResourceVSphereFolderCheckTags(tagResName string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		folder, err := testGetFolder(s, "folder")
		if err != nil {
			return err
		}
		tagsClient, err := testAccProvider.Meta().(*VSphereClient).TagsClient()
		if err != nil {
			return err
		}
		return testObjectHasTags(s, tagsClient, folder, tagResName)
	}
}

func testAccResourceVSphereFolderConfigBasic(name string, ft vSphereFolderType) string {
	return fmt.Sprintf(`
variable "datacenter" {
  default = "%s"
}

variable "folder_name" {
  default = "%s"
}

variable "folder_type" {
  default = "%s"
}

data "vsphere_datacenter" "dc" {
  name = "${var.datacenter}"
}

resource "vsphere_folder" "folder" {
  path          = "${var.folder_name}"
  type          = "${var.folder_type}"
  datacenter_id = "${data.vsphere_datacenter.dc.id}"
}
`,
		os.Getenv("VSPHERE_DATACENTER"),
		name,
		ft,
	)
}

func testAccResourceVSphereFolderConfigSubFolder(name string, ft vSphereFolderType) string {
	return fmt.Sprintf(`
variable "datacenter" {
  default = "%s"
}

variable "folder_name" {
  default = "%s"
}

variable "folder_type" {
  default = "%s"
}

variable "parent_name" {
  default = "%s"
}

data "vsphere_datacenter" "dc" {
  name = "${var.datacenter}"
}

resource "vsphere_folder" "parent" {
  path          = "${var.parent_name}"
  type          = "${var.folder_type}"
  datacenter_id = "${data.vsphere_datacenter.dc.id}"
}

resource "vsphere_folder" "folder" {
  path          = "${vsphere_folder.parent.path}/${var.folder_name}"
  type          = "${var.folder_type}"
  datacenter_id = "${data.vsphere_datacenter.dc.id}"
}
`,
		os.Getenv("VSPHERE_DATACENTER"),
		name,
		ft,
		testAccResourceVSphereFolderConfigExpectedParentName,
	)
}

func testAccResourceVSphereFolderConfigDatacenter() string {
	return fmt.Sprintf(`
variable "folder_name" {
  default = "%s"
}

variable "folder_type" {
  default = "%s"
}

resource "vsphere_folder" "folder" {
  path          = "${var.folder_name}"
  type          = "${var.folder_type}"
}
`,
		testAccResourceVSphereFolderConfigExpectedName,
		vSphereFolderTypeDatacenter,
	)
}

func testAccResourceVSphereFolderConfigTag() string {
	return fmt.Sprintf(`
variable "datacenter" {
  default = "%s"
}

variable "folder_name" {
  default = "%s"
}

variable "folder_type" {
  default = "%s"
}

data "vsphere_datacenter" "dc" {
  name = "${var.datacenter}"
}

resource "vsphere_tag_category" "terraform-test-category" {
  name        = "terraform-test-tag-category"
  cardinality = "MULTIPLE"

  associable_types = [
    "Folder",
  ]
}

resource "vsphere_tag" "terraform-test-tag" {
  name        = "terraform-test-tag"
  category_id = "${vsphere_tag_category.terraform-test-category.id}"
}

resource "vsphere_folder" "folder" {
  path          = "${var.folder_name}"
  type          = "${var.folder_type}"
  datacenter_id = "${data.vsphere_datacenter.dc.id}"
  tags          = ["${vsphere_tag.terraform-test-tag.id}"]
}
`,
		os.Getenv("VSPHERE_DATACENTER"),
		testAccResourceVSphereFolderConfigExpectedName,
		vSphereFolderTypeVM,
	)
}

func testAccResourceVSphereFolderConfigMultiTag() string {
	return fmt.Sprintf(`
variable "datacenter" {
  default = "%s"
}

variable "folder_name" {
  default = "%s"
}

variable "folder_type" {
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
    "Folder",
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

resource "vsphere_folder" "folder" {
  path          = "${var.folder_name}"
  type          = "${var.folder_type}"
  datacenter_id = "${data.vsphere_datacenter.dc.id}"
  tags          = ["${vsphere_tag.terraform-test-tags-alt.*.id}"]
}
`,
		os.Getenv("VSPHERE_DATACENTER"),
		testAccResourceVSphereFolderConfigExpectedName,
		vSphereFolderTypeVM,
	)
}
