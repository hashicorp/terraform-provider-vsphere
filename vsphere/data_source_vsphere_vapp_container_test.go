package vsphere

import (
	"fmt"
	"os"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/helper/resource"
)

func TestAccDataSourceVSphereVAppContainer_basic(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
			testAccDataSourceVSphereVAppContainerPreCheck(t)
			testAccSkipIfEsxi(t)
		},
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccDataSourceVSphereVAppContainerConfig(),
				Check: resource.ComposeTestCheckFunc(
					resource.TestMatchResourceAttr("data.vsphere_vapp_container.container", "id", regexp.MustCompile("^resgroup-")),
				),
			},
		},
	})
}

func TestAccDataSourceVSphereVAppContainer_path(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
			testAccDataSourceVSphereVAppContainerPreCheck(t)
			testAccSkipIfEsxi(t)
		},
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccDataSourceVSphereVAppContainerPathConfig(),
				Check: resource.ComposeTestCheckFunc(
					resource.TestMatchResourceAttr("data.vsphere_vapp_container.container", "id", regexp.MustCompile("^resgroup-")),
				),
			},
		},
	})
}

func testAccDataSourceVSphereVAppContainerPreCheck(t *testing.T) {
	if os.Getenv("TF_VAR_VSPHERE_DATACENTER") == "" {
		t.Skip("set TF_VAR_VSPHERE_DATACENTER to run vsphere_vapp_container acceptance tests")
	}
	if os.Getenv("TF_VAR_VSPHERE_CLUSTER") == "" {
		t.Skip("set TF_VAR_VSPHERE_CLUSTER to run vsphere_vapp_container acceptance tests")
	}
}

func testAccDataSourceVSphereVAppContainerConfig() string {
	return fmt.Sprintf(`
variable "cluster" {
  default = "%s"
}

variable "datacenter" {
  default = "%s"
}

data "vsphere_datacenter" "dc" {
  name = var.datacenter
}

data "vsphere_compute_cluster" "cluster" {
  datacenter_id = data.vsphere_datacenter.dc.id
  name          = var.cluster
}

resource "vsphere_vapp_container" "vapp" {
  name                 = "vapp-test"
  parent_resource_pool = data.vsphere_compute_cluster.cluster.resource_pool_id
}

data "vsphere_vapp_container" "container" {
  name          = vsphere_vapp_container.vapp.name
  datacenter_id = data.vsphere_datacenter.dc.id
}
`,
		os.Getenv("TF_VAR_VSPHERE_CLUSTER"),
		os.Getenv("TF_VAR_VSPHERE_DATACENTER"),
	)
}

func testAccDataSourceVSphereVAppContainerPathConfig() string {
	return fmt.Sprintf(`
variable "datacenter" {
  default = "%s"
}

variable "cluster" {
  default = "%s"
}

data "vsphere_datacenter" "dc" {
  name = var.datacenter
}

data "vsphere_compute_cluster" "cluster" {
  datacenter_id = data.vsphere_datacenter.dc.id
  name          = var.cluster
}

resource "vsphere_vapp_container" "vapp" {
  name                 = "vapp-test"
  parent_resource_pool = data.vsphere_compute_cluster.cluster.resource_pool_id
}

data "vsphere_vapp_container" "container" {
  name          = "/${var.datacenter}/vm/vapp-test"
  datacenter_id = data.vsphere_datacenter.dc.id
}
`,
		os.Getenv("TF_VAR_VSPHERE_DATACENTER"),
		os.Getenv("TF_VAR_VSPHERE_CLUSTER"),
	)
}
