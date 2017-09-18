package vsphere

import (
	"errors"
	"fmt"
	"reflect"
	"regexp"
	"sort"
	"strings"
	"testing"

	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

func TestAccResourceVSphereTagCategory(t *testing.T) {
	var tp *testing.T
	testAccResourceVSphereTagCategoryCases := []struct {
		name     string
		testCase resource.TestCase
	}{
		{
			"basic",
			resource.TestCase{
				PreCheck: func() {
					testAccPreCheck(tp)
				},
				Providers:    testAccProviders,
				CheckDestroy: testAccResourceVSphereTagCategoryExists(false),
				Steps: []resource.TestStep{
					{
						Config: testAccResourceVSphereTagCategoryConfigBasic,
						Check: resource.ComposeTestCheckFunc(
							testAccResourceVSphereTagCategoryExists(true),
							testAccResourceVSphereTagCategoryHasName("terraform-test-category"),
							testAccResourceVSphereTagCategoryHasCardinality(vSphereTagCategoryCardinalitySingle),
							testAccResourceVSphereTagCategoryHasTypes([]string{
								vSphereTagCategoryAssociableTypeVirtualMachine,
							}),
						),
					},
				},
			},
		},
		{
			"add type",
			resource.TestCase{
				PreCheck: func() {
					testAccPreCheck(tp)
				},
				Providers:    testAccProviders,
				CheckDestroy: testAccResourceVSphereTagCategoryExists(false),
				Steps: []resource.TestStep{
					{
						Config: testAccResourceVSphereTagCategoryConfigBasic,
						Check: resource.ComposeTestCheckFunc(
							testAccResourceVSphereTagCategoryExists(true),
							testAccResourceVSphereTagCategoryHasTypes([]string{
								vSphereTagCategoryAssociableTypeVirtualMachine,
							}),
						),
					},
					{
						Config: testAccResourceVSphereTagCategoryConfigMultiType,
						Check: resource.ComposeTestCheckFunc(
							testAccResourceVSphereTagCategoryExists(true),
							testAccResourceVSphereTagCategoryHasTypes([]string{
								vSphereTagCategoryAssociableTypeVirtualMachine,
								vSphereTagCategoryAssociableTypeDatastore,
							}),
						),
					},
				},
			},
		},
		{
			"remove type, should error",
			resource.TestCase{
				PreCheck: func() {
					testAccPreCheck(tp)
				},
				Providers:    testAccProviders,
				CheckDestroy: testAccResourceVSphereTagCategoryExists(false),
				Steps: []resource.TestStep{
					{
						Config: testAccResourceVSphereTagCategoryConfigMultiType,
						Check: resource.ComposeTestCheckFunc(
							testAccResourceVSphereTagCategoryExists(true),
						),
					},
					{
						Config:      testAccResourceVSphereTagCategoryConfigBasic,
						ExpectError: regexp.MustCompile("removal of associable types is not supported"),
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
				CheckDestroy: testAccResourceVSphereTagCategoryExists(false),
				Steps: []resource.TestStep{
					{
						Config: testAccResourceVSphereTagCategoryConfigBasic,
						Check: resource.ComposeTestCheckFunc(
							testAccResourceVSphereTagCategoryExists(true),
						),
					},
					{
						Config: testAccResourceVSphereTagCategoryConfigAltName,
						Check: resource.ComposeTestCheckFunc(
							testAccResourceVSphereTagCategoryExists(true),
							testAccResourceVSphereTagCategoryHasName("terraform-test-category-renamed"),
						),
					},
				},
			},
		},
		{
			"single cardinality",
			resource.TestCase{
				PreCheck: func() {
					testAccPreCheck(tp)
				},
				Providers:    testAccProviders,
				CheckDestroy: testAccResourceVSphereTagCategoryExists(false),
				Steps: []resource.TestStep{
					{
						Config: testAccResourceVSphereTagCategoryConfigSingleCardinality,
						Check: resource.ComposeTestCheckFunc(
							testAccResourceVSphereTagCategoryExists(true),
							testAccResourceVSphereTagCategoryHasCardinality(vSphereTagCategoryCardinalitySingle),
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
				CheckDestroy: testAccResourceVSphereTagCategoryExists(false),
				Steps: []resource.TestStep{
					{
						Config: testAccResourceVSphereTagCategoryConfigBasic,
						Check: resource.ComposeTestCheckFunc(
							testAccResourceVSphereTagCategoryExists(true),
						),
					},
					{
						ResourceName:      "vsphere_tag_category.terraform-test-category",
						ImportState:       true,
						ImportStateVerify: true,
						ImportStateIdFunc: func(s *terraform.State) (string, error) {
							cat, err := testGetTagCategory(s, "terraform-test-category")
							if err != nil {
								return "", err
							}
							return cat.Name, nil
						},
						Config: testAccResourceVSphereTagCategoryConfigBasic,
						Check: resource.ComposeTestCheckFunc(
							testAccResourceVSphereTagCategoryExists(true),
						),
					},
				},
			},
		},
	}

	for _, tc := range testAccResourceVSphereTagCategoryCases {
		t.Run(tc.name, func(t *testing.T) {
			tp = t
			resource.Test(t, tc.testCase)
		})
	}
}

func testAccResourceVSphereTagCategoryExists(expected bool) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		_, err := testGetTagCategory(s, "terraform-test-category")
		if err != nil {
			if strings.Contains(err.Error(), "Status code: 404") && !expected {
				// Expected missing
				return nil
			}
			return err
		}
		if !expected {
			return errors.New("expected tag category to be missing")
		}
		return nil
	}
}

func testAccResourceVSphereTagCategoryHasTypes(expected []string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		cat, err := testGetTagCategory(s, "terraform-test-category")
		if err != nil {
			return err
		}
		// We use a *schema.Set for types, so the list may not be in the order that
		// we expect it in. Sort both lists to make sure.
		actual := cat.AssociableTypes
		sort.Strings(expected)
		sort.Strings(actual)
		if !reflect.DeepEqual(expected, actual) {
			return fmt.Errorf("expected types list to be %#v, got %#v", expected, actual)
		}
		return nil
	}
}

func testAccResourceVSphereTagCategoryHasName(expected string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		cat, err := testGetTagCategory(s, "terraform-test-category")
		if err != nil {
			return err
		}
		actual := cat.Name
		if expected != actual {
			return fmt.Errorf("expected name to be %q, got %q", expected, actual)
		}
		return nil
	}
}

func testAccResourceVSphereTagCategoryHasCardinality(expected string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		cat, err := testGetTagCategory(s, "terraform-test-category")
		if err != nil {
			return err
		}
		actual := cat.Cardinality
		if expected != actual {
			return fmt.Errorf("expected cardinality to be %q, got %q", expected, actual)
		}
		return nil
	}
}

const testAccResourceVSphereTagCategoryConfigBasic = `
resource "vsphere_tag_category" "terraform-test-category" {
  name        = "terraform-test-category"
  description = "Managed by Terraform"
  cardinality = "SINGLE"

  associable_types = [
    "VirtualMachine",
  ]
}
`

const testAccResourceVSphereTagCategoryConfigMultiType = `
resource "vsphere_tag_category" "terraform-test-category" {
  name        = "terraform-test-category"
  description = "Managed by Terraform"
  cardinality = "SINGLE"

  associable_types = [
    "VirtualMachine",
    "Datastore",
  ]
}
`

const testAccResourceVSphereTagCategoryConfigMultiCardinality = `
resource "vsphere_tag_category" "terraform-test-category" {
  name        = "terraform-test-category"
  description = "Managed by Terraform"
  cardinality = "MULTIPLE"

  associable_types = [
    "VirtualMachine",
  ]
}
`

const testAccResourceVSphereTagCategoryConfigAltName = `
resource "vsphere_tag_category" "terraform-test-category" {
  name        = "terraform-test-category-renamed"
  description = "Managed by Terraform"
  cardinality = "MULTIPLE"

  associable_types = [
    "VirtualMachine",
  ]
}
`

const testAccResourceVSphereTagCategoryConfigSingleCardinality = `
resource "vsphere_tag_category" "terraform-test-category" {
  name        = "terraform-test-category"
  description = "Managed by Terraform"
  cardinality = "SINGLE"

  associable_types = [
    "VirtualMachine",
  ]
}
`
