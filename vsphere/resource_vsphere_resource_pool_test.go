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

func TestAccResourceVSphereResourcePool_basic(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
			testAccResourceVSphereResourcePoolPreCheck(t)
		},
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccResourceVSphereResourcePoolConfigBasic(),
				Check: resource.ComposeTestCheckFunc(
					testAccResourceVSphereResourcePoolCheckExists(true),
				),
			},
		},
	})
}

func TestAccResourceVSphereResourcePool_updateToCustom(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
			testAccResourceVSphereResourcePoolPreCheck(t)
		},
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccResourceVSphereResourcePoolConfigBasic(),
				Check: resource.ComposeTestCheckFunc(
					testAccResourceVSphereResourcePoolCheckExists(true),
				),
			},
			{
				Config: testAccResourceVSphereResourcePoolConfigNonDefault(),
				Check: resource.ComposeTestCheckFunc(
					testAccResourceVSphereResourcePoolCheckExists(true),
				),
			},
		},
	})
}

func TestAccResourceVSphereResourcePool_updateToDefaults(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
			testAccResourceVSphereResourcePoolPreCheck(t)
		},
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccResourceVSphereResourcePoolConfigNonDefault(),
				Check: resource.ComposeTestCheckFunc(
					testAccResourceVSphereResourcePoolCheckExists(true),
				),
			},
			{
				Config: testAccResourceVSphereResourcePoolConfigBasic(),
				Check: resource.ComposeTestCheckFunc(
					testAccResourceVSphereResourcePoolCheckExists(true),
				),
			},
		},
	})
}

func TestAccResourceVSphereResourcePool_esxiHost(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
			testAccResourceVSphereResourcePoolPreCheck(t)
		},
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccResourceVSphereResourcePoolConfigEsxiHost(),
				Check: resource.ComposeTestCheckFunc(
					testAccResourceVSphereResourcePoolCheckExists(true),
				),
			},
		},
	})
}

func testAccResourceVSphereResourcePoolPreCheck(t *testing.T) {
	if os.Getenv("VSPHERE_DATACENTER") == "" {
		t.Skip("set VSPHERE_DATACENTER to run vsphere_resource_pool acceptance tests")
	}
	if os.Getenv("VSPHERE_CLUSTER") == "" {
		t.Skip("set VSPHERE_CLUSTER to run vsphere_resource_pool acceptance tests")
	}
	if os.Getenv("VSPHERE_ESXI_HOST7") == "" {
		t.Skip("set VSPHERE_ESXI_HOST7 to run vsphere_resource_pool acceptance tests")
	}
}

func testAccResourceVSphereResourcePoolCheckExists(expected bool) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		_, err := testGetResourcePool(s, "resource_pool")
		if err != nil {
			if viapi.IsManagedObjectNotFoundError(err) && expected == false {
				// Expected missing
				return nil
			}
			return err
		}
		if !expected {
			return errors.New("expected resource pool to be missing")
		}
		return nil
	}
}

func testAccResourceVSphereResourcePoolConfigNonDefault() string {
	return fmt.Sprintf(`
variable "datacenter" {
  default = "%s"
}

variable "cluster" {
  default = "%s"
}

data "vsphere_datacenter" "dc" {
  name = "${var.datacenter}"
}

data "vsphere_compute_cluster" "cluster" {
  name          = "${var.cluster}"
  datacenter_id = "${data.vsphere_datacenter.dc.id}"
}

resource "vsphere_resource_pool" "resource_pool" {
  name                  = "terraform-resource-pool-test"
  root_resource_pool_id = "${data.vsphere_compute_cluster.cluster.resource_pool_id}"
  cpu_share_level       = "custom"
  cpu_shares            = 10
  cpu_reservation       = 10
  cpu_expandable        = false
  cpu_limit             = 20
  memory_share_level    = "custom"
  memory_shares         = 10
  memory_reservation    = 10
  memory_expandable     = false
  memory_limit          = 20
}
`,
		os.Getenv("VSPHERE_DATACENTER"),
		os.Getenv("VSPHERE_CLUSTER"),
	)
}

func testAccResourceVSphereResourcePoolConfigEsxiHost() string {
	return fmt.Sprintf(`
variable "datacenter" {
  default = "%s"
}

variable "host" {
  default = "%s"
}

data "vsphere_datacenter" "dc" {
  name = "${var.datacenter}"
}

data "vsphere_host" "host" {
  name          = "${var.host}"
  datacenter_id = "${data.vsphere_datacenter.dc.id}"
}

resource "vsphere_resource_pool" "resource_pool" {
  name                  = "terraform-resource-pool-test"
  root_resource_pool_id = "${data.vsphere_host.host.resource_pool_id}"
}
`,
		os.Getenv("VSPHERE_DATACENTER"),
		os.Getenv("VSPHERE_ESXI_HOST7"),
	)
}

func testAccResourceVSphereResourcePoolConfigBasic() string {
	return fmt.Sprintf(`
variable "datacenter" {
  default = "%s"
}

variable "cluster" {
  default = "%s"
}

data "vsphere_datacenter" "dc" {
  name = "${var.datacenter}"
}

data "vsphere_compute_cluster" "cluster" {
  name          = "${var.cluster}"
  datacenter_id = "${data.vsphere_datacenter.dc.id}"
}

resource "vsphere_resource_pool" "resource_pool" {
  name                  = "terraform-resource-pool-test"
  root_resource_pool_id = "${data.vsphere_compute_cluster.cluster.resource_pool_id}"
}
`,
		os.Getenv("VSPHERE_DATACENTER"),
		os.Getenv("VSPHERE_CLUSTER"),
	)
}
