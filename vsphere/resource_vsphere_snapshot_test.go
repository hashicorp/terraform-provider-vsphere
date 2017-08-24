package vsphere

import (
	"context"
	"fmt"
	"os"
	"testing"

	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
	"github.com/vmware/govmomi"
	"github.com/vmware/govmomi/find")

func testBasicPreCheckSnapshot(t *testing.T) {
	testAccPreCheck(t)
}

func TestAccVmSnapshot_Basic(t *testing.T) {
	var vmId, snapshotName, description, memory, quiesce string
	if v := os.Getenv("VSPHERE_VM_ID"); v != "" {
		vmId = v
	}
	if v := os.Getenv("VSPHERE_VM_SNAPSHOT_NAME"); v != "" {
		snapshotName = v
	}
	if v := os.Getenv("VSPHERE_VM_SNAPSHOT_DESC"); v != "" {
		description = v
	}
	if v := os.Getenv("VSPHERE_VM_SNAPSHOT_MEMORY"); v != "" {
		memory = v
	}
	if v := os.Getenv("VSPHERE_VM_SNAPSHOT_QUIESCE"); v != "" {
		quiesce = v
	}

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckVmSnapshotDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccCheckVSphereVMSnapshotConfig_basic(vmId, snapshotName, description, memory, quiesce),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckVmSnapshotExists("vsphere_virtual_machine_snapshot.Test_terraform_cases"),
					resource.TestCheckResourceAttr(
						"vsphere_virtual_machine_snapshot.Test_terraform_cases", "snapshot_name", snapshotName),
				),
			},
		},
	})
}

func testAccCheckVmSnapshotDestroy(s *terraform.State) error {
	client := testAccProvider.Meta().(*govmomi.Client)

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "vsphere_virtual_machine_snapshot" {
			continue
		}
		dc, err := getDatacenter(client, "")
		if err != nil {
			return fmt.Errorf("error %s", err)
		}
		finder := find.NewFinder(client.Client, true)
		finder = finder.SetDatacenter(dc)
		vm, err := finder.VirtualMachine(context.TODO(), vmPath(rs.Primary.Attributes["folder"], rs.Primary.Attributes["vm_id"]))
		if err != nil {
			return fmt.Errorf("error %s", err)
		}

		snapshot, err := vm.FindSnapshot(context.TODO(), rs.Primary.Attributes["snapshot_name"])
		if err == nil {
			return fmt.Errorf("Vm Snapshot still exists: %v", snapshot)
		}
	}

	return nil
}

func testAccCheckVmSnapshotExists(n string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]

		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No Vm Snapshot ID is set")
		}
		client := testAccProvider.Meta().(*govmomi.Client)

		dc, err := getDatacenter(client, "")
		if err != nil {
			return fmt.Errorf("error %s", err)
		}
		finder := find.NewFinder(client.Client, true)
		finder = finder.SetDatacenter(dc)
		vm, err := finder.VirtualMachine(context.TODO(), os.Getenv("VSPHERE_VM_ID"))
		if err != nil {
			return fmt.Errorf("error %s", err)
		}
		snapshot, err := vm.FindSnapshot(context.TODO(), rs.Primary.ID)
		if err != nil {
			return fmt.Errorf("Error while getting the snapshot %v", snapshot)
		}

		return nil
	}
}

func testAccCheckVSphereVMSnapshotConfig_basic(vmId, snapshotName, description, memory, quiesce string) string {
	return fmt.Sprintf(`
	resource "vsphere_virtual_machine_snapshot" "Test_terraform_cases" {
	  vm_id = "%s"
	  snapshot_name = "%s"
	  description = "%s"
	  memory = %s
	  quiesce = %s
  }`, vmId, snapshotName, description, memory, quiesce)
}
