package vsphere

import (
	"fmt"
	"os"
	"testing"

	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
	"github.com/vmware/govmomi"
	//	"github.com/vmware/govmomi/find"
	//  "golang.org/x/net/context"
)

func testAccCheckVSphereDVSConfigNoUplinks() string {
	return fmt.Sprintf(`
data "vsphere_datacenter" "datacenter" {
	  name = "%s"
}

resource "vsphere_distributed_virtual_switch" "testDVS" {
	datacenter_id = "${data.vsphere_datacenter.datacenter.id}"
  name = "testDVS"
	contact = "dvsmanager@yourcompany.com"
	contact_name = "John Doe"
	description = "Test DVS"
}
`, os.Getenv("VSPHERE_DATACENTER"))
}

func testAccCheckVSphereDVSConfigUplinks(uplinks bool, multiple bool) string {
	if uplinks {
		conf := `
data "vsphere_datacenter" "datacenter" {
	  name = "%s"
}

data "vsphere_host" "esxi_host" {
	  name = "%s"
    datacenter_id = "${data.vsphere_datacenter.datacenter.id}"
}

resource "vsphere_distributed_virtual_switch" "testDVS" {
	datacenter_id = "${data.vsphere_datacenter.datacenter.id}"
  name = "testDVS"
  host = [{ 
		host_system_id = "${data.vsphere_host.esxi_host.id}" 
		backing = ["%s"]
	}]
}
`
		if multiple {
			backing := fmt.Sprintf("%s\",\"%s", os.Getenv("VSPHERE_HOST_NIC0"), os.Getenv("VSPHERE_HOST_NIC1"))
			return fmt.Sprintf(conf, os.Getenv("VSPHERE_DATACENTER"), os.Getenv("VSPHERE_ESXI_HOST"), backing)
		} else {
			return fmt.Sprintf(conf, os.Getenv("VSPHERE_DATACENTER"), os.Getenv("VSPHERE_ESXI_HOST"), os.Getenv("VSPHERE_HOST_NIC0"))
		}
	} else {
		return fmt.Sprintf(`
data "vsphere_datacenter" "datacenter" {
	  name = "%s"
}

data "vsphere_host" "esxi_host" {
	  name = "%s"
    datacenter_id = "${data.vsphere_datacenter.datacenter.id}"
}

resource "vsphere_distributed_virtual_switch" "testDVS" {
	datacenter_id = "${data.vsphere_datacenter.datacenter.id}"
  name = "testDVS"
}
`, os.Getenv("VSPHERE_DATACENTER"), os.Getenv("VSPHERE_ESXI_HOST"))
	}
}

func testAccResourceVSphereDistributedVirtualSwitchPreCheck(t *testing.T) {
	if os.Getenv("VSPHERE_DATACENTER") == "" {
		t.Skip("set VSPHERE_DATACENTER to run vsphere_distributed_virtual_switch acceptance tests")
	}
	if os.Getenv("VSPHERE_HOST_NIC0") == "" {
		t.Skip("set VSPHERE_HOST_NIC0 to run vsphere_distributed_virtual_switch acceptance tests")
	}
	if os.Getenv("VSPHERE_HOST_NIC1") == "" {
		t.Skip("set VSPHERE_HOST_NIC0 to run vsphere_distributed_virtual_switch acceptance tests")
	}
	if os.Getenv("VSPHERE_ESXI_HOST") == "" {
		t.Skip("set VSPHERE_HOST_NIC0 to run vsphere_distributed_virtual_switch acceptance tests")
	}
}

// Create a distributed virtual switch with no uplinks
func TestAccVSphereDVS_createWithoutUplinks(t *testing.T) {
	resourceName := "vsphere_distributed_virtual_switch.testDVS"

	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
			testAccResourceVSphereDistributedVirtualSwitchPreCheck(t)
		},
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckVSphereDVSDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccCheckVSphereDVSConfigNoUplinks(),
				Check:  resource.ComposeTestCheckFunc(testAccCheckVSphereDVSExists(resourceName, 0)),
			},
		},
	})
}

// Create a distributed virtual switch with one uplink
func TestAccVSphereDVS_createWithUplinks(t *testing.T) {
	resourceName := "vsphere_distributed_virtual_switch.testDVS"

	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
			testAccResourceVSphereDistributedVirtualSwitchPreCheck(t)
		},
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckVSphereDVSDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccCheckVSphereDVSConfigUplinks(true, false),
				Check:  resource.ComposeTestCheckFunc(testAccCheckVSphereDVSExists(resourceName, 1)),
			},
		},
	})
}

// Create a distributed virtual switch with an uplink, delete it and add it again
func TestAccVSphereDVS_createAndUpdateWithUplinks(t *testing.T) {
	resourceName := "vsphere_distributed_virtual_switch.testDVS"

	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
			testAccResourceVSphereDistributedVirtualSwitchPreCheck(t)
		},
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckVSphereDVSDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccCheckVSphereDVSConfigUplinks(true, false),
				Check:  resource.ComposeTestCheckFunc(testAccCheckVSphereDVSExists(resourceName, 1)),
			},
			{
				Config: testAccCheckVSphereDVSConfigUplinks(false, false),
				Check:  resource.ComposeTestCheckFunc(testAccCheckVSphereDVSExists(resourceName, 0)),
			},
			{
				Config: testAccCheckVSphereDVSConfigUplinks(true, false),
				Check:  resource.ComposeTestCheckFunc(testAccCheckVSphereDVSExists(resourceName, 1)),
			},
			{
				Config: testAccCheckVSphereDVSConfigUplinks(false, false),
				Check:  resource.ComposeTestCheckFunc(testAccCheckVSphereDVSExists(resourceName, 0)),
			},
			{
				Config: testAccCheckVSphereDVSConfigUplinks(true, true),
				Check:  resource.ComposeTestCheckFunc(testAccCheckVSphereDVSExists(resourceName, 2)),
			},
		},
	})
}

func testAccCheckVSphereDVSDestroy(s *terraform.State) error {
	client := testAccProvider.Meta().(*govmomi.Client)

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "vsphere_distributed_virtual_switch" {
			continue
		}

		id := rs.Primary.ID
		_, err := dvsFromUuid(client, id)
		if err != nil {
			fmt.Printf("Expected error received: %s\n", err)
			return nil
		} else {
			return fmt.Errorf("distributed virtual switch '%s' still exists", id)
		}
	}
	return nil
}

func testAccCheckVSphereDVSExists(name string, n int) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		client := testAccProvider.Meta().(*govmomi.Client)

		for _, rs := range s.RootModule().Resources {
			if rs.Type != "vsphere_distributed_virtual_switch" {
				continue
			}
			if rs.Primary.Attributes["name"] != name {
				continue
			}

			id := rs.Primary.ID
			dvs, err := dvsFromUuid(client, id)
			if err != nil {
				return fmt.Errorf("distributed virtual switch '%s' doesn't exists", id)
			} else {
				config := dvs.Config.GetDVSConfigInfo()
				if len(config.Host) != n {
					return fmt.Errorf("expected '%d' uplinks, found '%d'", n, len(config.Host))
				}
				fmt.Println("DVS exists and has the correct number of uplinks")
				return nil
			}
		}
		return nil
	}
}
