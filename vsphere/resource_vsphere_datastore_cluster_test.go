package vsphere

import (
	"errors"
	"fmt"
	"os"
	"testing"

	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
	"github.com/terraform-providers/terraform-provider-vsphere/vsphere/internal/helper/viapi"
)

func TestAccResourceVSphereDatastoreCluster_basic(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},
		Providers:    testAccProviders,
		CheckDestroy: testAccResourceVSphereDatastoreClusterCheckExists(false),
		Steps: []resource.TestStep{
			{
				Config: testAccResourceVSphereDatastoreClusterConfigBasic(),
				Check: resource.ComposeTestCheckFunc(
					testAccResourceVSphereDatastoreClusterCheckExists(true),
					testAccResourceVSphereDatastoreClusterCheckSDRSEnabled(false),
				),
			},
		},
	})
}

func TestAccResourceVSphereDatastoreCluster_sdrsEnabled(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},
		Providers:    testAccProviders,
		CheckDestroy: testAccResourceVSphereDatastoreClusterCheckExists(false),
		Steps: []resource.TestStep{
			{
				Config: testAccResourceVSphereDatastoreClusterConfigSDRSBasic(),
				Check: resource.ComposeTestCheckFunc(
					testAccResourceVSphereDatastoreClusterCheckExists(true),
					testAccResourceVSphereDatastoreClusterCheckSDRSEnabled(true),
				),
			},
		},
	})
}

func testAccResourceVSphereDatastoreClusterCheckExists(expected bool) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		_, err := testGetDatastoreCluster(s, "datastore_cluster")
		if err != nil {
			if viapi.IsManagedObjectNotFoundError(err) && expected == false {
				// Expected missing
				return nil
			}
			return err
		}
		if !expected {
			return errors.New("expected datastore cluster to be missing")
		}
		return nil
	}
}

func testAccResourceVSphereDatastoreClusterCheckSDRSEnabled(expected bool) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		props, err := testGetDatastoreClusterProperties(s, "datastore_cluster")
		if err != nil {
			return err
		}
		actual := props.PodStorageDrsEntry.StorageDrsConfig.PodConfig.Enabled
		if expected != actual {
			return fmt.Errorf("expected enabled to be %t, got %t", expected, actual)
		}
		return nil
	}
}

func testAccResourceVSphereDatastoreClusterConfigBasic() string {
	return fmt.Sprintf(`
variable "datacenter" {
  default = "%s"
}

data "vsphere_datacenter" "dc" {
  name = "${var.datacenter}"
}

resource "vsphere_datastore_cluster" "datastore_cluster" {
  name          = "terraform-datastore-cluster-test"
  datacenter_id = "${data.vsphere_datacenter.dc.id}"
}
`,
		os.Getenv("VSPHERE_DATACENTER"),
	)
}

func testAccResourceVSphereDatastoreClusterConfigSDRSBasic() string {
	return fmt.Sprintf(`
variable "datacenter" {
  default = "%s"
}

data "vsphere_datacenter" "dc" {
  name = "${var.datacenter}"
}

resource "vsphere_datastore_cluster" "datastore_cluster" {
  name          = "terraform-datastore-cluster-test"
  datacenter_id = "${data.vsphere_datacenter.dc.id}"
  sdrs_enabled  = true
}
`,
		os.Getenv("VSPHERE_DATACENTER"),
	)
}
