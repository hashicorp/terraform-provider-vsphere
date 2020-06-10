package vsphere

import (
	"fmt"
	"github.com/hashicorp/terraform-plugin-sdk/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/terraform"
	"regexp"
	"testing"
)

func TestAccDataSourceVSphereDynamic_regexAndTag(t *testing.T) {
	t.Cleanup(RunSweepers)
	resource.Test(t, resource.TestCase{
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccDataSourceVSphereDynamicConfigBase(),
			},
			{
				Config: testAccDataSourceVSphereDynamicConfig(),
				Check: resource.ComposeTestCheckFunc(
					testMatchDatacenterIds("vsphere_datacenter.dc2", "data.vsphere_dynamic.dyn1"),
				),
			},
			{
				Config: testAccDataSourceVSphereDynamicConfigBase(),
			},
		},
	})
}

func TestAccDataSourceVSphereDynamic_multiTag(t *testing.T) {
	t.Cleanup(RunSweepers)
	resource.Test(t, resource.TestCase{
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccDataSourceVSphereDynamicConfigBase(),
			},
			{
				Config: testAccDataSourceVSphereDynamicConfig(),
				Check: resource.ComposeTestCheckFunc(
					testMatchDatacenterIds("vsphere_datacenter.dc1", "data.vsphere_dynamic.dyn2"),
				),
			},
			{
				Config: testAccDataSourceVSphereDynamicConfigBase(),
			},
		},
	})
}

func TestAccDataSourceVSphereDynamic_multiResult(t *testing.T) {
	t.Cleanup(RunSweepers)
	resource.Test(t, resource.TestCase{
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccDataSourceVSphereDynamicConfigBase(),
			},
			{
				Config: testAccDataSourceVSphereDynamicConfig(),
				Check: resource.ComposeTestCheckFunc(
					resource.TestMatchResourceAttr("data.vsphere_dynamic.dyn3", "id", regexp.MustCompile("datacenter-")),
				),
			},
			{
				Config: testAccDataSourceVSphereDynamicConfigBase(),
			},
		},
	})
}

func TestAccDataSourceVSphereDynamic_sameTagNames(t *testing.T) {
	t.Cleanup(RunSweepers)
	resource.Test(t, resource.TestCase{
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccDataSourceVSphereDynamicConfigBase(),
			},
			{
				Config: testAccDataSourceVSphereDynamicConfig(),
				Check: resource.ComposeTestCheckFunc(
					testMatchDatacenterIds("vsphere_datacenter.dc2", "data.vsphere_dynamic.dyn4"),
				),
			},
			{
				Config: testAccDataSourceVSphereDynamicConfigBase(),
			},
		},
	})
}

func TestAccDataSourceVSphereDynamic_typeFilter(t *testing.T) {
	t.Cleanup(RunSweepers)
	resource.Test(t, resource.TestCase{
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccDataSourceVSphereDynamicConfigBase(),
			},
			{
				Config: testAccDataSourceVSphereDynamicConfig(),
				Check: resource.ComposeTestCheckFunc(
					testMatchDatacenterIds("vsphere_datacenter.dc1", "data.vsphere_dynamic.dyn5"),
				),
			},
			{
				Config: testAccDataSourceVSphereDynamicConfigBase(),
			},
		},
	})
}

func testMatchDatacenterIds(a, b string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		ida := s.RootModule().Resources[a].Primary.Attributes["moid"]
		idb := s.RootModule().Resources[b].Primary.ID
		if ida != idb {
			return fmt.Errorf("unexpected ID. Expected: %s, Got: %s", idb, ida)
		}
		return nil
	}
}

func init() {
	resource.AddTestSweepers("tags", &resource.Sweeper{
		Name:         "tag_cleanup",
		Dependencies: nil,
		F:            tagSweep,
	})
	resource.AddTestSweepers("datacenters", &resource.Sweeper{
		Name:         "datacenter_cleanup",
		Dependencies: nil,
		F:            dcSweep,
	})
}

func testAccDataSourceVSphereDynamicConfigBase() string {
	return fmt.Sprintf(`
resource "vsphere_datacenter" "dc1" {
  name = "testdc1"
  tags = [vsphere_tag.tag1.id, vsphere_tag.tag2.id]
}

resource "vsphere_datacenter" "dc2" {
  name = "testdc2"
  tags = [ vsphere_tag.tag1.id, vsphere_tag.tag3.id ]
}

resource "vsphere_tag_category" "category1" {
  name        = "cat1"
  description = "cat1"
  cardinality = "MULTIPLE"

  associable_types = [
    "Datacenter"
  ]
}

resource "vsphere_tag_category" "category2" {
  name        = "cat2"
  description = "cat2"
  cardinality = "MULTIPLE"

  associable_types = [
    "Datacenter"
  ]
}

resource "vsphere_tag" "tag1" {
  name        = "tag1"
  category_id = vsphere_tag_category.category1.id
}

resource "vsphere_tag" "tag2" {
  name        = "tag2"
  category_id = vsphere_tag_category.category2.id
}

resource "vsphere_tag" "tag3" {
  name        = "tag2"
  category_id = vsphere_tag_category.category1.id
}`)
}

func testAccDataSourceVSphereDynamicConfig() string {
	return fmt.Sprintf(`
resource "vsphere_datacenter" "dc1" {
  name = "testdc1"
  tags = [vsphere_tag.tag1.id, vsphere_tag.tag2.id]
}

resource "vsphere_datacenter" "dc2" {
  name = "testdc2"
  tags = [ vsphere_tag.tag1.id, vsphere_tag.tag3.id ]
}

resource "vsphere_tag_category" "category1" {
  name        = "cat1"
  description = "cat1"
  cardinality = "MULTIPLE"

  associable_types = [
    "Datacenter"
  ]
}

resource "vsphere_tag_category" "category2" {
  name        = "cat2"
  description = "cat2"
  cardinality = "MULTIPLE"

  associable_types = [
    "Datacenter"
  ]
}

resource "vsphere_tag" "tag1" {
  name        = "tag1"
  category_id = vsphere_tag_category.category1.id
}

resource "vsphere_tag" "tag2" {
  name        = "tag2"
  category_id = vsphere_tag_category.category2.id
}

resource "vsphere_tag" "tag3" {
  name        = "tag2"
  category_id = vsphere_tag_category.category1.id
}

data "vsphere_dynamic" "dyn1" {
 filter     = [ vsphere_tag.tag1.id ]
 name_regex = "dc2"
}

data "vsphere_dynamic" "dyn2" {
  filter     = [ vsphere_tag.tag1.id, vsphere_tag.tag2.id ]
  name_regex = ""
}

data "vsphere_dynamic" "dyn3" {
 filter     = [ vsphere_tag.tag1.id ]
 name_regex = ""
}

data "vsphere_dynamic" "dyn4" {
  filter     = [ vsphere_tag.tag3.id ]
  name_regex = ""
}

data "vsphere_dynamic" "dyn5" {
  filter     = [ vsphere_tag.tag1.id, vsphere_tag.tag2.id ]
  name_regex = ""
  type       = "Datacenter"
}

data "vsphere_datacenter" "dc1" {
  name = vsphere_datacenter.dc1.name
}

data "vsphere_datacenter" "dc2" {
  name = vsphere_datacenter.dc2.name
}
`)
}
