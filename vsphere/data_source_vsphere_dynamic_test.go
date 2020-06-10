package vsphere

import (
	"fmt"
	"github.com/hashicorp/terraform-plugin-sdk/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/terraform"
	"github.com/terraform-providers/terraform-provider-vsphere/vsphere/internal/helper/testhelper"
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
				Config: testAccDataSourceVSphereConfigRegexAndTag(),
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
				Config: testAccDataSourceVSphereConfigMultiTag(),
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
				Config:      testAccDataSourceVSphereConfigMultiMatch(),
				ExpectError: regexp.MustCompile("multiple object match the supplied criteria"),
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
				Config: testAccDataSourceVSphereConfigType(),
				Check: resource.ComposeTestCheckFunc(
					testMatchDatacenterIds("vsphere_datacenter.dc1", "data.vsphere_dynamic.dyn4"),
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
	return testhelper.CombineConfigs(
		testhelper.ConfigResDC1(),
		testhelper.ConfigResDC2(),
		testhelper.ConfigResTagCat1(),
		testhelper.ConfigResTagCat2(),
		testhelper.ConfigResTag1(),
		testhelper.ConfigResTag2(),
		testhelper.ConfigResTag3(),
	)
}

func testAccDataSourceVSphereConfigRegexAndTag() string {
	conf := fmt.Sprintf(`
data "vsphere_dynamic" "dyn1" {
 filter     = [ vsphere_tag.tag1.id ]
 name_regex = "dc2"
}
`)
	return testhelper.CombineConfigs(
		testAccDataSourceVSphereDynamicConfigBase(),
		conf,
		testhelper.ConfigDataDC2(),
	)
}

func testAccDataSourceVSphereConfigMultiTag() string {
	conf := fmt.Sprintf(`
data "vsphere_dynamic" "dyn2" {
  filter     = [ vsphere_tag.tag1.id, vsphere_tag.tag2.id ]
  name_regex = ""
}
`)
	return testhelper.CombineConfigs(
		testAccDataSourceVSphereDynamicConfigBase(),
		conf,
		testhelper.ConfigDataDC1(),
	)
}

func testAccDataSourceVSphereConfigMultiMatch() string {
	conf := fmt.Sprintf(`
data "vsphere_dynamic" "dyn3" {
  filter     = [ vsphere_tag.tag1.id ]
  name_regex = ""
}
`)
	return testhelper.CombineConfigs(
		testAccDataSourceVSphereDynamicConfigBase(),
		conf,
		testhelper.ConfigDataDC1(),
	)
}

func testAccDataSourceVSphereConfigType() string {
	conf := fmt.Sprintf(`
data "vsphere_dynamic" "dyn4" {
 filter     = [ vsphere_tag.tag1.id, vsphere_tag.tag2.id ]
 name_regex = ""
 type       = "Datacenter"
}
`)
	return testhelper.CombineConfigs(
		testAccDataSourceVSphereDynamicConfigBase(),
		conf,
		testhelper.ConfigDataDC1(),
	)
}
