package vsphere

import (
	"context"
	"fmt"
	"os"
	"testing"

	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
	"github.com/vmware/govmomi"
	"github.com/vmware/govmomi/find"
	"github.com/vmware/govmomi/vim25/mo"
)

func testBasicPreCheckSnapshotRevert(t *testing.T) {
	testAccPreCheck(t)

}

func TestAccVmSnapshotRevert_Basic(t *testing.T) {
	var vmId, snapshotId, suppressPower string
	if v := os.Getenv("VSPHERE_VM_ID"); v != "" {
		vmId = v
	}
	if v := os.Getenv("VSPHERE_VM_SNAPSHOT_ID"); v != "" {
		snapshotId = v
	}
	if v := os.Getenv("VSPHERE_SUPPRESS_POWER_ON"); v != "" {
		suppressPower = v
	}
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckVmSnapshotRevertDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccCheckVSphereVMSnapshotRevertConfig_basic(vmId, snapshotId, suppressPower),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckVmCurrentSnapshot("vsphere_virtual_machine_snapshot_revert.Test_terraform_cases", snapshotId),
				),
			},
		},
	})
}

func testAccCheckVmSnapshotRevertDestroy(s *terraform.State) error {

	return nil
}

func testAccCheckVmCurrentSnapshot(n, snapshot_name string) resource.TestCheckFunc {
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

		var vm_object mo.VirtualMachine

		err = vm.Properties(context.TODO(), vm.Reference(), []string{"snapshot"}, &vm_object)

		if err != nil {
			return nil
		}
		current_snap := vm_object.Snapshot.CurrentSnapshot
		snapshot, err := vm.FindSnapshot(context.TODO(), snapshot_name)

		if err != nil {
			return fmt.Errorf("Error while getting the snapshot %v", snapshot)
		}
		if fmt.Sprintf("<%s>", snapshot) == fmt.Sprintf("<%s>", current_snap) {
			return nil
		}

		return fmt.Errorf("Test Case failed for revert snapshot. Current snapshot does not match to reverted snapshot")
	}
}

func testAccCheckVSphereVMSnapshotRevertConfig_basic(vmId, snapshotId, suppressPowerOn string) string {
	return fmt.Sprintf(`
	resource "vsphere_virtual_machine_snapshot_revert" "Test_terraform_cases"{
		vm_id = "%s"
		snapshot_id = "%s"
		suppress_power_on = %s
	}`, vmId, snapshotId, suppressPowerOn)
}
