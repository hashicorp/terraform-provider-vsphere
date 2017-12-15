package vsphere

import (
	"fmt"
	"os"
	"testing"
	"github.com/hashicorp/terraform/terraform"
	"github.com/hashicorp/terraform/helper/resource"
)


const testAccCheckVSphereClusterComputeConfig = `
resource "vsphere_cluster_compute" "%s" {
	cluster = "%s"
	datacenter = "%s"
	drs-enabled = %t
	drs-behavior = "%s"
	drs-recommendations = %d

}
`

func testClusterHasCorrectDrsBahavior(state *terraform.State) (_ error) {
	return nil
}

func testClusterExists(state *terraform.State) (_ error) {
	return nil
}

func TestClusterImport(t *testing.T) {
	testClusterComputeName := "TestClusterImport"
	sourceDatacenter := os.Getenv("VSPHERE_DATACENTER")
	datacenter := sourceDatacenter
	drsEnabled := true
	drsBehavior := "partiallyAutomated"
	drsRecommendations := 0

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		//CheckDestroy: testAccCheckVSphereFileDestroy,
		Steps: []resource.TestStep{
			{
				ImportState: true,
				ImportStateId: testClusterComputeName,
				Config: fmt.Sprintf(
					testAccCheckVSphereClusterComputeConfig,
					testClusterComputeName,
					testClusterComputeName,
					datacenter,
					drsEnabled,
					drsBehavior,
					drsRecommendations,
				),
				Check: resource.ComposeTestCheckFunc(
					testClusterExists,
				),
			},
		},
	})
}

