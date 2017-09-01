package vsphere

import (
	"fmt"
	"os"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform/helper/resource"
)

var testAccDataSourceVSphereDatacenterExpectedRegexp = regexp.MustCompile("^datacenter-")

func TestAccDataSourceVSphereDatacenter(t *testing.T) {
	var tp *testing.T
	testAccDataSourceVSphereDatacenterCases := []struct {
		name     string
		testCase resource.TestCase
	}{
		{
			"basic",
			resource.TestCase{
				PreCheck: func() {
					testAccPreCheck(tp)
					testAccDataSourceVSphereDatacenterPreCheck(tp)
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
			},
		},
		{
			"default",
			resource.TestCase{
				PreCheck: func() {
					testAccPreCheck(tp)
					testAccDataSourceVSphereDatacenterPreCheck(tp)
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
			},
		},
	}

	for _, tc := range testAccDataSourceVSphereDatacenterCases {
		t.Run(tc.name, func(t *testing.T) {
			tp = t
			resource.Test(t, tc.testCase)
		})
	}
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
