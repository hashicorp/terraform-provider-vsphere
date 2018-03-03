package vsphere

import (
	"fmt"
	"os"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform/helper/resource"
)

var testAccDataSourceVSphereDatacenterExpectedRegexp = regexp.MustCompile("^datacenter-")

func TestAccDataSourceVSphereDatacenter_basic(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
			testAccDataSourceVSphereDatacenterPreCheck(t)
		},
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccDataSourceVSphereDatacenterConfig(),
				Check: resource.ComposeTestCheckFunc(
					resource.TestMatchResourceAttr(
						"data.vsphere_datacenter.dc",
						"id",
						testAccDataSourceVSphereDatacenterExpectedRegexp,
					),
				),
			},
		},
	})
}

func TestAccDataSourceVSphereDatacenter_defaultDatacenter(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
			testAccDataSourceVSphereDatacenterPreCheck(t)
		},
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccDataSourceVSphereDatacenterConfigDefault,
				Check: resource.ComposeTestCheckFunc(
					resource.TestMatchResourceAttr(
						"data.vsphere_datacenter.dc",
						"id",
						testAccDataSourceVSphereDatacenterExpectedRegexp,
					),
				),
			},
		},
	})
}

func testAccDataSourceVSphereDatacenterPreCheck(t *testing.T) {
	if os.Getenv("VSPHERE_DATACENTER") == "" {
		t.Skip("set VSPHERE_DATACENTER to run vsphere_datacenter acceptance tests")
	}
}

func testAccDataSourceVSphereDatacenterConfig() string {
	return fmt.Sprintf(`
data "vsphere_datacenter" "dc" {
  name = "%s"
}
`, os.Getenv("VSPHERE_DATACENTER"))
}

const testAccDataSourceVSphereDatacenterConfigDefault = `
data "vsphere_datacenter" "dc" {}
`
