package vsphere

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

func TestAccResourceVSphereTag_basic(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},
		Providers:    testAccProviders,
		CheckDestroy: testAccResourceVSphereTagExists(false),
		Steps: []resource.TestStep{
			{
				Config: testAccResourceVSphereTagConfigBasic,
				Check: resource.ComposeTestCheckFunc(
					testAccResourceVSphereTagExists(true),
					testAccResourceVSphereTagHasName("terraform-test-tag"),
					testAccResourceVSphereTagHasDescription("Managed by Terraform"),
					testAccResourceVSphereTagHasCategory(),
				),
			},
		},
	})
}

func TestAccResourceVSphereTag_changeName(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},
		Providers:    testAccProviders,
		CheckDestroy: testAccResourceVSphereTagExists(false),
		Steps: []resource.TestStep{
			{
				Config: testAccResourceVSphereTagConfigBasic,
				Check: resource.ComposeTestCheckFunc(
					testAccResourceVSphereTagExists(true),
				),
			},
			{
				Config: testAccResourceVSphereTagConfigAltName,
				Check: resource.ComposeTestCheckFunc(
					testAccResourceVSphereTagExists(true),
					testAccResourceVSphereTagHasName("terraform-test-tag-renamed"),
				),
			},
		},
	})
}

func TestAccResourceVSphereTag_changeDescription(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},
		Providers:    testAccProviders,
		CheckDestroy: testAccResourceVSphereTagExists(false),
		Steps: []resource.TestStep{
			{
				Config: testAccResourceVSphereTagConfigBasic,
				Check: resource.ComposeTestCheckFunc(
					testAccResourceVSphereTagExists(true),
				),
			},
			{
				Config: testAccResourceVSphereTagConfigAltDescription,
				Check: resource.ComposeTestCheckFunc(
					testAccResourceVSphereTagExists(true),
					testAccResourceVSphereTagHasDescription("Still managed by Terraform"),
				),
			},
		},
	})
}

func TestAccResourceVSphereTag_detachAllTags(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},
		Providers:    testAccProviders,
		CheckDestroy: testAccResourceVSphereTagExists(false),
		Steps: []resource.TestStep{
			{
				Config: testAccResourceVSphereTagConfigOnFolderAttached(),
				Check: resource.ComposeTestCheckFunc(
					testAccResourceVSphereTagExists(true),
				),
			},
			{
				Config: testAccResourceVSphereTagConfigOnFolderNotAttached(),
				Check: resource.ComposeTestCheckFunc(
					testAccResourceVSphereTagExists(true),
					testAccResourceVSphereFolderCheckNoTags(),
				),
			},
		},
	})
}

func TestAccResourceVSphereTag_import(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},
		Providers:    testAccProviders,
		CheckDestroy: testAccResourceVSphereTagExists(false),
		Steps: []resource.TestStep{
			{
				Config: testAccResourceVSphereTagConfigBasic,
				Check: resource.ComposeTestCheckFunc(
					testAccResourceVSphereTagExists(true),
				),
			},
			{
				ResourceName:      "vsphere_tag.terraform-test-tag",
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateIdFunc: func(s *terraform.State) (string, error) {
					cat, err := testGetTagCategory(s, "terraform-test-category")
					if err != nil {
						return "", err
					}
					tag, err := testGetTag(s, "terraform-test-tag")
					if err != nil {
						return "", err
					}
					m := make(map[string]string)
					m["category_name"] = cat.Name
					m["tag_name"] = tag.Name
					b, err := json.Marshal(m)
					if err != nil {
						return "", err
					}

					return string(b), nil
				},
				Config: testAccResourceVSphereTagConfigBasic,
				Check: resource.ComposeTestCheckFunc(
					testAccResourceVSphereTagExists(true),
				),
			},
		},
	})
}

func testAccResourceVSphereTagExists(expected bool) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		_, err := testGetTag(s, "terraform-test-tag")
		if err != nil {
			if strings.Contains(err.Error(), "Status code: 404") && !expected {
				// Expected missing
				return nil
			}
			return err
		}
		if !expected {
			return errors.New("expected tag to be missing")
		}
		return nil
	}
}

func testAccResourceVSphereTagHasName(expected string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		tag, err := testGetTag(s, "terraform-test-tag")
		if err != nil {
			return err
		}
		actual := tag.Name
		if expected != actual {
			return fmt.Errorf("expected name to be %q, got %q", expected, actual)
		}
		return nil
	}
}

func testAccResourceVSphereTagHasDescription(expected string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		tag, err := testGetTag(s, "terraform-test-tag")
		if err != nil {
			return err
		}
		actual := tag.Description
		if expected != actual {
			return fmt.Errorf("expected description to be %q, got %q", expected, actual)
		}
		return nil
	}
}

func testAccResourceVSphereTagHasCategory() resource.TestCheckFunc {
	return func(s *terraform.State) error {
		tag, err := testGetTag(s, "terraform-test-tag")
		if err != nil {
			return err
		}
		category, err := testGetTagCategory(s, "terraform-test-category")
		if err != nil {
			return err
		}

		expected := category.ID
		actual := tag.CategoryID
		if expected != actual {
			return fmt.Errorf("expected ID to be %q, got %q", expected, actual)
		}
		return nil
	}
}

const testAccResourceVSphereTagConfigBasic = `
resource "vsphere_tag_category" "terraform-test-category" {
  name        = "terraform-test-category"
  cardinality = "SINGLE"

  associable_types = [
    "All",
  ]
}

resource "vsphere_tag" "terraform-test-tag" {
  name        = "terraform-test-tag"
  description = "Managed by Terraform"
  category_id = "${vsphere_tag_category.terraform-test-category.id}"
}
`

const testAccResourceVSphereTagConfigAltName = `
resource "vsphere_tag_category" "terraform-test-category" {
  name        = "terraform-test-category"
  cardinality = "SINGLE"

  associable_types = [
    "All",
  ]
}

resource "vsphere_tag" "terraform-test-tag" {
  name        = "terraform-test-tag-renamed"
  description = "Managed by Terraform"
  category_id = "${vsphere_tag_category.terraform-test-category.id}"
}
`

const testAccResourceVSphereTagConfigAltDescription = `
resource "vsphere_tag_category" "terraform-test-category" {
  name        = "terraform-test-category"
  cardinality = "SINGLE"

  associable_types = [
    "All",
  ]
}

resource "vsphere_tag" "terraform-test-tag" {
  name        = "terraform-test-tag"
  description = "Still managed by Terraform"
  category_id = "${vsphere_tag_category.terraform-test-category.id}"
}
`

func testAccResourceVSphereTagConfigOnFolderAttached() string {
	return fmt.Sprintf(`
data "vsphere_datacenter" "dc" {
  name = "%s"
}

resource "vsphere_tag_category" "terraform-test-category" {
  name        = "terraform-test-category"
  cardinality = "SINGLE"

  associable_types = [
    "Folder",
  ]
}

resource "vsphere_tag" "terraform-test-tag" {
  name        = "terraform-test-tag"
  description = "Managed by Terraform"
  category_id = "${vsphere_tag_category.terraform-test-category.id}"
}

resource "vsphere_folder" "folder" {
  path          = "terraform-test-folder"
  type          = "vm"
  datacenter_id = "${data.vsphere_datacenter.dc.id}"

  tags = ["${vsphere_tag.terraform-test-tag.id}"]
}
`,
		os.Getenv("VSPHERE_DATACENTER"),
	)
}

func testAccResourceVSphereTagConfigOnFolderNotAttached() string {
	return fmt.Sprintf(`
data "vsphere_datacenter" "dc" {
  name = "%s"
}

resource "vsphere_tag_category" "terraform-test-category" {
  name        = "terraform-test-category"
  cardinality = "SINGLE"

  associable_types = [
    "Folder",
  ]
}

resource "vsphere_tag" "terraform-test-tag" {
  name        = "terraform-test-tag"
  description = "Managed by Terraform"
  category_id = "${vsphere_tag_category.terraform-test-category.id}"
}

resource "vsphere_folder" "folder" {
  path          = "terraform-test-folder"
  type          = "vm"
  datacenter_id = "${data.vsphere_datacenter.dc.id}"
}
`,
		os.Getenv("VSPHERE_DATACENTER"),
	)
}
