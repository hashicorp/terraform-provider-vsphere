package vsphere

import (
	"fmt"
	"testing"
	"time"

	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
	"github.com/vmware/govmomi"
	"github.com/vmware/govmomi/find"
	"golang.org/x/net/context"
)

// Create a datacenter on the root folder
func TestAccVSphereDatacenter_createOnRootFolder(t *testing.T) {
	name := "testDC"
	resourceName := "vsphere_datacenter." + name

	waitForCreation := func(*terraform.State) error {
		// sleep 5 seconds to give it time to create
		time.Sleep(5 * time.Second)
		return nil
	}

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckVSphereDatacenterDestroy,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(
					testAccCheckVSphereDatacenterConfig,
					name,
					name,
				),
				Check: resource.ComposeTestCheckFunc(
					waitForCreation,
					testAccCheckVSphereDatacenterExists(resourceName, true),
				),
			},
		},
	})
}

func testAccCheckVSphereDatacenterDestroy(s *terraform.State) error {
	client := testAccProvider.Meta().(*govmomi.Client)
	finder := find.NewFinder(client.Client, true)

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "vsphere_datacenter" {
			continue
		}

		path := rs.Primary.Attributes["name"]
		if _, ok := rs.Primary.Attributes["folder"]; ok {
			path = rs.Primary.Attributes["folder"] + path
		}
		_, err := finder.Datacenter(context.TODO(), path)
		if err != nil {
			switch err.(type) {
			case *find.NotFoundError:
				fmt.Printf("Expected error received: %s\n", err)
				return nil
			default:
				return err
			}
		} else {
			return fmt.Errorf("Datacenter '%s' still exists", path)
		}
	}

	return nil
}

func testAccCheckVSphereDatacenterExists(n string, exists bool) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("[ERROR] Resource not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("[ERROR] No ID is set")
		}

		client := testAccProvider.Meta().(*govmomi.Client)
		finder := find.NewFinder(client.Client, true)

		path := rs.Primary.Attributes["name"]
		if _, ok := rs.Primary.Attributes["folder"]; ok {
			path = rs.Primary.Attributes["folder"] + path
		}
		_, err := finder.Datacenter(context.TODO(), path)
		if err != nil {
			switch e := err.(type) {
			case *find.NotFoundError:
				if exists {
					return fmt.Errorf("Datacenter does not exist: %s", e.Error())
				}
				fmt.Printf("Expected error received: %s\n", e.Error())
				return nil
			default:
				return err
			}
		}
		return nil
	}
}

const testAccCheckVSphereDatacenterConfig = `
resource "vsphere_datacenter" "%s" {
	name = "%s"
}
`
