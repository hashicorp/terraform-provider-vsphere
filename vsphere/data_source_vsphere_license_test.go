// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package vsphere

import (
	"fmt"
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestAccDataSourceVSphereLicense_basic(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			RunSweepers()
			testAccPreCheck(t)
			testAccDataSourceVSphereLicensePreCheck(t)
		},
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccDataSourceVSphereLicenseConfig(),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(
						"data.vsphere_license.license",
						"id",
						os.Getenv("TF_VAR_VSPHERE_LICENSE"),
					),
				),
			},
		},
	})
}

func testAccDataSourceVSphereLicensePreCheck(t *testing.T) {
	if os.Getenv("TF_VAR_VSPHERE_LICENSE") == "" {
		t.Skip("set TF_VAR_VSPHERE_LICENSE to run vsphere_license acceptance tests")
	}
}

func testAccDataSourceVSphereLicenseConfig() string {
	return fmt.Sprintf(`
data "vsphere_license" "license" {
  license_key = "%s"
}
`, os.Getenv("TF_VAR_VSPHERE_LICENSE"))
}
