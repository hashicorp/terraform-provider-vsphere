// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package vsphere

import (
	"fmt"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
	"github.com/hashicorp/terraform-provider-vsphere/vsphere/internal/helper/testhelper"
	"os"
	"testing"
)

func TestAccResourceVSphereOfflineSoftwareDepot_basic(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			RunSweepers()
			testAccPreCheck(t)
			testAccResourceVSphereOfflineSoftwareDepotPreCheck(t)
		},
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testhelper.ConfigDataSoftwareDepot(),
				Check: resource.ComposeTestCheckFunc(
					testAccResourceVSphereOfflineSoftwareDepotCheckFunc(),
				),
			},
		},
	})
}

func testAccResourceVSphereOfflineSoftwareDepotCheckFunc() resource.TestCheckFunc {
	return func(s *terraform.State) error {
		if tVars, err := testClientVariablesForResource(s, "vsphere_offline_software_depot.depot"); err != nil {
			return err
		} else {
			location, _ := tVars.resourceAttributes["location"]
			expected := os.Getenv("TF_VAR_VSPHERE_SOFTWARE_DEPOT_LOCATION")
			if location != expected {
				return fmt.Errorf("depot location is incorrect. Expected %s but got %s", expected, location)
			}
		}

		return nil
	}
}

func testAccResourceVSphereOfflineSoftwareDepotPreCheck(t *testing.T) {
	if os.Getenv("TF_VAR_VSPHERE_SOFTWARE_DEPOT_LOCATION") == "" {
		t.Skip("set TF_VAR_VSPHERE_SOFTWARE_DEPOT_LOCATION to run vsphere_offline_software_depot acceptance tests")
	}
}
