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

const testAccCheckVSphereDVSConfigNoUplinks = `
resource "vsphere_distributed_virtual_switch" "testDVS" {
	datacenter = "%s"
  name = "testDVS"
}
`

const testAccCheckVSphereDVSConfigUplinks = `
resource "vsphere_distributed_virtual_switch" "testDVS" {
	datacenter = "%s"
  name = "testDVS"
  host = [{ 
		host = "%s" 
		backing = ["%s"]
	}]
}
`

// Create a distributed virtual switch with no uplinks
func TestAccVSphereDVS_createWithoutUplinks(t *testing.T) {
	resourceName := "vsphere_distributed_virtual_switch.testDVS"
	datacenter := os.Getenv("VSPHERE_DATACENTER")

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckVSphereDVSDestroy,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(testAccCheckVSphereDVSConfigNoUplinks, datacenter),
				Check:  resource.ComposeTestCheckFunc(testAccCheckVSphereDVSExists(resourceName, true)),
			},
		},
	})
}

// Create a distributed virtual switch with uplinks
func TestAccVSphereDVS_createWithUplinks(t *testing.T) {
	resourceName := "vsphere_distributed_virtual_switch.testDVS"
	datacenter := os.Getenv("VSPHERE_DATACENTER")
	host := os.Getenv("VSPHERE_HOST")
	host_pnic := os.Getenv("VSPHERE_HOST_PNIC")

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckVSphereDVSDestroy,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(testAccCheckVSphereDVSConfigUplinks, datacenter, host, host_pnic),
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
