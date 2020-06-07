package vsphere

import (
	"fmt"
	"github.com/hashicorp/terraform-plugin-sdk/terraform"
	"os"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/helper/resource"
)

func TestAccResourceVSphereContentLibraryItem_basic(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
			testAccResourceVSphereContentLibraryItemPreCheck(t)
		},
		Providers:    testAccProviders,
		CheckDestroy: testAccResourceVSphereContentLibraryItemCheckExists(false),
		Steps: []resource.TestStep{
			{
				Config: testAccResourceVSphereContentLibraryItemConfig(),
				Check: resource.ComposeTestCheckFunc(
					resource.TestMatchResourceAttr(
						"vsphere_content_library_item.item", "id", regexp.MustCompile("^[a-f0-9]{8}-[a-f0-9]{4}-[a-f0-9]{4}-[a-f0-9]{4}-[a-f0-9]{12}$"),
					),
					resource.TestMatchResourceAttr(
						"vsphere_content_library_item.item", "description", regexp.MustCompile("Ubuntu Description"),
					),
					resource.TestMatchResourceAttr(
						"vsphere_content_library_item.item", "name", regexp.MustCompile("ubuntu"),
					),
					resource.TestMatchResourceAttr(
						"vsphere_content_library_item.item", "type", regexp.MustCompile("ovf"),
					),
					testAccResourceVSphereContentLibraryItemDescription(regexp.MustCompile("Ubuntu Description")),
					testAccResourceVSphereContentLibraryItemName(regexp.MustCompile("ubuntu")),
					testAccResourceVSphereContentLibraryItemType(regexp.MustCompile("ovf")),
				),
			},
			{
				ResourceName:      "vsphere_content_library_item.item",
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateVerifyIgnore: []string{
					"file_url",
				},
				Config: testAccResourceVSphereContentLibraryItemConfig(),
			},
		},
	})
}

func testAccResourceVSphereContentLibraryItemPreCheck(t *testing.T) {
	if os.Getenv("TF_VAR_VSPHERE_DATACENTER") == "" {
		t.Skip("set TF_VAR_VSPHERE_DATACENTER to run vsphere_content_library acceptance tests")
	}
	if os.Getenv("TF_VAR_VSPHERE_NFS_DS_NAME") == "" {
		t.Skip("set TF_VAR_VSPHERE_NFS_DS_NAME to run vsphere_content_library acceptance tests")
	}
	if os.Getenv("TF_VAR_VSPHERE_CONTENT_LIBRARY_FILES") == "" {
		t.Skip("set TF_VAR_VSPHERE_CONTENT_LIBRARY_FILES to run vsphere_content_library acceptance tests")
	}
}

func testAccResourceVSphereContentLibraryItemDescription(expected *regexp.Regexp) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		library, err := testGetContentLibraryItem(s, "item")
		if err != nil {
			return err
		}
		if !expected.MatchString(library.Description) {
			return fmt.Errorf("Content Library item description does not match. expected: %s, got %s", expected.String(), library.Description)
		}
		return nil
	}
}

func testAccResourceVSphereContentLibraryItemName(expected *regexp.Regexp) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		library, err := testGetContentLibraryItem(s, "item")
		if err != nil {
			return err
		}
		if !expected.MatchString(library.Name) {
			return fmt.Errorf("Content Library item name does not match. expected: %s, got %s", expected.String(), library.Name)
		}
		return nil
	}
}

func testAccResourceVSphereContentLibraryItemType(expected *regexp.Regexp) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		library, err := testGetContentLibraryItem(s, "item")
		if err != nil {
			return err
		}
		if !expected.MatchString(library.Type) {
			return fmt.Errorf("Content Library item type does not match. expected: %s, got %s", expected.String(), library.Type)
		}
		return nil
	}
}

func testAccResourceVSphereContentLibraryItemConfig() string {
	return fmt.Sprintf(`
variable "datacenter" {
  type    = "string"
  default = "%s"
}

variable "datastore" {
  type    = "string"
  default = "%s"
}

variable "file_list" {
  type    = list(string)
  default = %s 
}

data "vsphere_datacenter" "dc" {
  name = var.datacenter
}

data "vsphere_datastore" "ds" {
  datacenter_id = data.vsphere_datacenter.dc.id
  name = var.datastore
}

resource "vsphere_content_library" "library" {
  name            = "ContentLibrary_test"
  storage_backing = [ data.vsphere_datastore.ds.id ]
  description     = "Library Description"
}

resource "vsphere_content_library_item" "item" {
  name        = "ubuntu"
  description = "Ubuntu Description"
  library_id  = vsphere_content_library.library.id
  type        = "ovf"
  file_url    = var.file_list
}
`,
		os.Getenv("TF_VAR_VSPHERE_DATACENTER"),
		os.Getenv("TF_VAR_VSPHERE_NFS_DS_NAME"),
		os.Getenv("TF_VAR_VSPHERE_CONTENT_LIBRARY_FILES"),
	)
}

func testAccResourceVSphereContentLibraryItemCheckExists(expected bool) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		_, err := testGetContentLibraryItem(s, "item")
		if err != nil {
			missingState, _ := regexp.MatchString("not found in state", err.Error())
			missingVSphere, _ := regexp.MatchString("404 Not Found", err.Error())
			if missingState && !expected || missingVSphere && !expected {
				// Expected missing
				return nil
			}
			return err
		}
		if !expected {
			return fmt.Errorf("expected Content Library item to be missing")
		}
		return nil
	}
}
