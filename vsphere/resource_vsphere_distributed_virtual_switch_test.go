package vsphere

import (
	"fmt"
	"os"
	"testing"

	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
	"github.com/vmware/govmomi"
	"github.com/vmware/govmomi/find"
	"golang.org/x/net/context"
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

func testAccCheckVSphereDVSConfigUplinks(uplinks bool) string {
	if uplinks {
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
  host = [{ 
		host_system_id = "${data.vsphere_host.esxi_host.id}" 
		backing = ["%s"]
	}]
}
`, os.Getenv("VSPHERE_DATACENTER"), os.Getenv("VSPHERE_ESXI_HOST"), os.Getenv("VSPHERE_HOST_NIC0"))
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

// Create a distributed virtual switch with no uplinks
func TestAccVSphereDVS_createWithoutUplinks(t *testing.T) {
	resourceName := "vsphere_distributed_virtual_switch.testDVS"

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckVSphereDVSDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccCheckVSphereDVSConfigNoUplinks(),
				Check:  resource.ComposeTestCheckFunc(testAccCheckVSphereDVSExists(resourceName, true)),
			},
		},
	})
}

// Create a distributed virtual switch with uplinks
func TestAccVSphereDVS_createWithUplinks(t *testing.T) {
	resourceName := "vsphere_distributed_virtual_switch.testDVS"

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckVSphereDVSDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccCheckVSphereDVSConfigUplinks(true),
				Check:  resource.ComposeTestCheckFunc(testAccCheckVSphereDVSExists(resourceName, true)),
			},
		},
	})
}

// Create a distributed virtual switch with an uplink, delete it and add it again
func TestAccVSphereDVS_createAndUpdateWithUplinks(t *testing.T) {
	resourceName := "vsphere_distributed_virtual_switch.testDVS"

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckVSphereDVSDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccCheckVSphereDVSConfigUplinks(true),
				Check:  resource.ComposeTestCheckFunc(testAccCheckVSphereDVSExists(resourceName, true)),
			}, // XXX checks here need to be more thorough
			{
				Config: testAccCheckVSphereDVSConfigUplinks(false),
				Check:  resource.ComposeTestCheckFunc(testAccCheckVSphereDVSExists(resourceName, true)),
			},
			{
				Config: testAccCheckVSphereDVSConfigUplinks(true),
				Check:  resource.ComposeTestCheckFunc(testAccCheckVSphereDVSExists(resourceName, true)),
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

		datacenter := rs.Primary.Attributes["datacenter"]
		dc, err := getDatacenter(client, datacenter)
		if err != nil {
			return err
		}

		finder := find.NewFinder(client.Client, true)
		finder = finder.SetDatacenter(dc)

		name := rs.Primary.Attributes["name"]
		_, err = finder.NetworkList(context.TODO(), name)
		if err != nil {
			switch err.(type) {
			case *find.NotFoundError:
				fmt.Printf("Expected error received: %s\n", err)
				return nil
			default:
				return err
			}
		} else {
			return fmt.Errorf("distributed virtual switch '%s' still exists", name)
		}
	}

	return nil
}

func testAccCheckVSphereDVSExists(n string, exists bool) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("resource not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("no ID is set")
		}

		client := testAccProvider.Meta().(*govmomi.Client)

		datacenter := rs.Primary.Attributes["datacenter"]
		dc, err := getDatacenter(client, datacenter)
		if err != nil {
			return err
		}

		finder := find.NewFinder(client.Client, true)
		finder = finder.SetDatacenter(dc)

		name := rs.Primary.Attributes["name"]
		_, err = finder.NetworkList(context.TODO(), name)
		if err != nil {
			switch err.(type) {
			case *find.NotFoundError:
				fmt.Printf("Expected error received: %s\n", err)
				return nil
			default:
				return err
			}
		}
		return nil
	}
}
