package vsphere

import (
	"fmt"
	"os"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform/helper/resource"
)

func TestAccDataSourceVSphereHost(t *testing.T) {
	var tp *testing.T
	testAccDataSourceVSphereHostCases := []struct {
		name     string
		testCase resource.TestCase
	}{
		{
			"basic",
			resource.TestCase{
				PreCheck: func() {
					testAccPreCheck(tp)
					testAccDataSourceVSphereHostPreCheck(tp)
				},
				Providers: testAccProviders,
				Steps: []resource.TestStep{
					{
						Config: testAccDataSourceVSphereHostConfig(),
						Check: resource.ComposeTestCheckFunc(
							resource.TestMatchResourceAttr(
								"data.vsphere_host.host",
								"id",
								testAccDataSourceVSphereHostExpectedRegexp(),
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
					testAccDataSourceVSphereHostPreCheck(tp)
					testAccSkipIfNotEsxi(tp)
				},
				Providers: testAccProviders,
				Steps: []resource.TestStep{
					{
						Config: testAccDataSourceVSphereHostConfigDefault,
						Check: resource.ComposeTestCheckFunc(
							resource.TestMatchResourceAttr(
								"data.vsphere_host.host",
								"id",
								testAccDataSourceVSphereHostExpectedRegexp(),
							),
						),
					},
				},
			},
		},
	}

	for _, tc := range testAccDataSourceVSphereHostCases {
		t.Run(tc.name, func(t *testing.T) {
			tp = t
			resource.Test(t, tc.testCase)
		})
	}
}

func testAccDataSourceVSphereHostPreCheck(t *testing.T) {
	if os.Getenv("VSPHERE_DATACENTER") == "" {
		t.Skip("set VSPHERE_DATACENTER to run vsphere_host acceptance tests")
	}
	if os.Getenv("VSPHERE_ESXI_HOST") == "" {
		t.Skip("set VSPHERE_ESXI_HOST to run vsphere_host acceptance tests")
	}
}

func testAccDataSourceVSphereHostExpectedRegexp() *regexp.Regexp {
	if os.Getenv("VSPHERE_TEST_ESXI") != "" {
		return regexp.MustCompile("^ha-host$")
	}
	return regexp.MustCompile("^host-")
}

func testAccDataSourceVSphereHostConfig() string {
	return fmt.Sprintf(`
data "vsphere_datacenter" "dc" {
	name = "%s"
}

data "vsphere_host" "host" {
  name          = "%s"
  datacenter_id = "${data.vsphere_datacenter.dc.id}"
}
`, os.Getenv("VSPHERE_DATACENTER"), os.Getenv("VSPHERE_ESXI_HOST"))
}

const testAccDataSourceVSphereHostConfigDefault = `
data "vsphere_datacenter" "dc" {}

data "vsphere_host" "host" {
  datacenter_id = "${data.vsphere_datacenter.dc.id}"
}
`
