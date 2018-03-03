package vsphere

import (
	"errors"
	"fmt"
	"net"
	"os"
	"path"
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"testing"

	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
	"github.com/terraform-providers/terraform-provider-vsphere/vsphere/internal/helper/computeresource"
	"github.com/terraform-providers/terraform-provider-vsphere/vsphere/internal/helper/datastore"
	"github.com/terraform-providers/terraform-provider-vsphere/vsphere/internal/helper/folder"
	"github.com/terraform-providers/terraform-provider-vsphere/vsphere/internal/helper/resourcepool"
	"github.com/terraform-providers/terraform-provider-vsphere/vsphere/internal/helper/structure"
	"github.com/terraform-providers/terraform-provider-vsphere/vsphere/internal/helper/virtualdisk"
	"github.com/terraform-providers/terraform-provider-vsphere/vsphere/internal/virtualdevice"
	"github.com/vmware/govmomi/object"
	"github.com/vmware/govmomi/vim25/types"
)

const (
	testAccResourceVSphereVirtualMachineDiskNameEager     = "terraform-test_1.vmdk"
	testAccResourceVSphereVirtualMachineDiskNameLazy      = "terraform-test_2.vmdk"
	testAccResourceVSphereVirtualMachineDiskNameThin      = "terraform-test_3.vmdk"
	testAccResourceVSphereVirtualMachineDiskNameExtraVmdk = "terraform-test-vm-extra-disk.vmdk"
	testAccResourceVSphereVirtualMachineStaticMacAddr     = "06:5c:89:2b:a0:64"
	testAccResourceVSphereVirtualMachineAnnotation        = "Managed by Terraform"
)

func TestAccResourceVSphereVirtualMachine_basic(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
			testAccResourceVSphereVirtualMachinePreCheck(t)
		},
		Providers:    testAccProviders,
		CheckDestroy: testAccResourceVSphereVirtualMachineCheckExists(false),
		Steps: []resource.TestStep{
			{
				Config: testAccResourceVSphereVirtualMachineConfigBasic(),
				Check: resource.ComposeTestCheckFunc(
					testAccResourceVSphereVirtualMachineCheckExists(true),
					resource.TestMatchResourceAttr("vsphere_virtual_machine.vm", "moid", regexp.MustCompile("^vm-")),
				),
			},
		},
	})
}

func TestAccResourceVSphereVirtualMachine_ESXiOnly(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
			testAccResourceVSphereVirtualMachinePreCheck(t)
			testAccSkipIfNotEsxi(t)
		},
		Providers:    testAccProviders,
		CheckDestroy: testAccResourceVSphereVirtualMachineCheckExists(false),
		Steps: []resource.TestStep{
			{
				Config: testAccResourceVSphereVirtualMachineConfigBasicESXiOnly(),
				Check: resource.ComposeTestCheckFunc(
					testAccResourceVSphereVirtualMachineCheckExists(true),
				),
			},
		},
	})
}

func TestAccResourceVSphereVirtualMachine_shutdownOK(t *testing.T) {
	var state *terraform.State

	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
			testAccResourceVSphereVirtualMachinePreCheck(t)
		},
		Providers:    testAccProviders,
		CheckDestroy: testAccResourceVSphereVirtualMachineCheckExists(false),
		Steps: []resource.TestStep{
			{
				Config: testAccResourceVSphereVirtualMachineConfigBasic(),
				Check: resource.ComposeTestCheckFunc(
					copyStatePtr(&state),
					testAccResourceVSphereVirtualMachineCheckExists(true),
				),
			},
			{
				PreConfig: func() {
					if err := testPowerOffVM(state, "vm"); err != nil {
						panic(err)
					}
				},
				PlanOnly: true,
				Config:   testAccResourceVSphereVirtualMachineConfigBasic(),
			},
		},
	})
}

func TestAccResourceVSphereVirtualMachine_reCreateOnDeletion(t *testing.T) {
	var state *terraform.State

	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
			testAccResourceVSphereVirtualMachinePreCheck(t)
		},
		Providers:    testAccProviders,
		CheckDestroy: testAccResourceVSphereVirtualMachineCheckExists(false),
		Steps: []resource.TestStep{
			{
				Config: testAccResourceVSphereVirtualMachineConfigBasic(),
				Check: resource.ComposeTestCheckFunc(
					copyState(&state),
					testAccResourceVSphereVirtualMachineCheckExists(true),
				),
			},
			{
				PreConfig: func() {
					if err := testDeleteVM(state, "vm"); err != nil {
						panic(err)
					}
				},
				Config: testAccResourceVSphereVirtualMachineConfigBasic(),
				Check: resource.ComposeTestCheckFunc(
					testAccResourceVSphereVirtualMachineCheckExists(true),
					func(s *terraform.State) error {
						oldID := state.RootModule().Resources["vsphere_virtual_machine.vm"].Primary.ID
						return testCheckResourceNotAttr("vsphere_virtual_machine.vm", "id", oldID)(s)
					},
				),
			},
		},
	})
}

func TestAccResourceVSphereVirtualMachine_multiDevice(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
			testAccResourceVSphereVirtualMachinePreCheck(t)
		},
		Providers:    testAccProviders,
		CheckDestroy: testAccResourceVSphereVirtualMachineCheckExists(false),
		Steps: []resource.TestStep{
			{
				Config: testAccResourceVSphereVirtualMachineConfigMultiDevice(),
				Check: resource.ComposeTestCheckFunc(
					testAccResourceVSphereVirtualMachineCheckExists(true),
					testAccResourceVSphereVirtualMachineCheckMultiDevice([]bool{true, true, true}, []bool{true, true, true}),
				),
			},
		},
	})
}

func TestAccResourceVSphereVirtualMachine_addDevices(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
			testAccResourceVSphereVirtualMachinePreCheck(t)
		},
		Providers:    testAccProviders,
		CheckDestroy: testAccResourceVSphereVirtualMachineCheckExists(false),
		Steps: []resource.TestStep{
			{
				Config: testAccResourceVSphereVirtualMachineConfigBasic(),
				Check: resource.ComposeTestCheckFunc(
					testAccResourceVSphereVirtualMachineCheckExists(true),
				),
			},
			{
				Config: testAccResourceVSphereVirtualMachineConfigMultiDevice(),
				Check: resource.ComposeTestCheckFunc(
					testAccResourceVSphereVirtualMachineCheckExists(true),
					testAccResourceVSphereVirtualMachineCheckMultiDevice([]bool{true, true, true}, []bool{true, true, true}),
				),
			},
		},
	})
}

func TestAccResourceVSphereVirtualMachine_removeMiddleDevices(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
			testAccResourceVSphereVirtualMachinePreCheck(t)
		},
		Providers:    testAccProviders,
		CheckDestroy: testAccResourceVSphereVirtualMachineCheckExists(false),
		Steps: []resource.TestStep{
			{
				Config: testAccResourceVSphereVirtualMachineConfigMultiDevice(),
				Check: resource.ComposeTestCheckFunc(
					testAccResourceVSphereVirtualMachineCheckExists(true),
					testAccResourceVSphereVirtualMachineCheckMultiDevice([]bool{true, true, true}, []bool{true, true, true}),
				),
			},
			{
				Config: testAccResourceVSphereVirtualMachineConfigRemoveMiddle(),
				Check: resource.ComposeTestCheckFunc(
					testAccResourceVSphereVirtualMachineCheckExists(true),
					testAccResourceVSphereVirtualMachineCheckMultiDevice([]bool{true, false, true}, []bool{true, false, true}),
				),
			},
		},
	})
}

func TestAccResourceVSphereVirtualMachine_removeMiddleDevicesChangeDiskUnit(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
			testAccResourceVSphereVirtualMachinePreCheck(t)
		},
		Providers:    testAccProviders,
		CheckDestroy: testAccResourceVSphereVirtualMachineCheckExists(false),
		Steps: []resource.TestStep{
			{
				Config: testAccResourceVSphereVirtualMachineConfigMultiDevice(),
				Check: resource.ComposeTestCheckFunc(
					testAccResourceVSphereVirtualMachineCheckExists(true),
					testAccResourceVSphereVirtualMachineCheckMultiDevice([]bool{true, true, true}, []bool{true, true, true}),
				),
			},
			{
				Config: testAccResourceVSphereVirtualMachineConfigRemoveMiddleChangeUnit(),
				Check: resource.ComposeTestCheckFunc(
					testAccResourceVSphereVirtualMachineCheckExists(true),
					testAccResourceVSphereVirtualMachineCheckMultiDevice([]bool{true, false, true}, []bool{true, false, true}),
				),
			},
		},
	})
}

func TestAccResourceVSphereVirtualMachine_highDiskUnitNumbers(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
			testAccResourceVSphereVirtualMachinePreCheck(t)
		},
		Providers:    testAccProviders,
		CheckDestroy: testAccResourceVSphereVirtualMachineCheckExists(false),
		Steps: []resource.TestStep{
			{
				Config: testAccResourceVSphereVirtualMachineConfigMultiHighBus(),
				Check: resource.ComposeTestCheckFunc(
					testAccResourceVSphereVirtualMachineCheckExists(true),
					testAccResourceVSphereVirtualMachineCheckDiskBus("terraform-test.vmdk", 0, 0),
					testAccResourceVSphereVirtualMachineCheckDiskBus("terraform-test_1.vmdk", 1, 0),
					testAccResourceVSphereVirtualMachineCheckDiskBus("terraform-test_2.vmdk", 2, 1),
				),
			},
		},
	})
}

func TestAccResourceVSphereVirtualMachine_highDiskUnitsToRegularSingleController(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
			testAccResourceVSphereVirtualMachinePreCheck(t)
		},
		Providers:    testAccProviders,
		CheckDestroy: testAccResourceVSphereVirtualMachineCheckExists(false),
		Steps: []resource.TestStep{
			{
				Config: testAccResourceVSphereVirtualMachineConfigMultiHighBus(),
				Check: resource.ComposeTestCheckFunc(
					testAccResourceVSphereVirtualMachineCheckExists(true),
					testAccResourceVSphereVirtualMachineCheckDiskBus("terraform-test.vmdk", 0, 0),
					testAccResourceVSphereVirtualMachineCheckDiskBus("terraform-test_1.vmdk", 1, 0),
					testAccResourceVSphereVirtualMachineCheckDiskBus("terraform-test_2.vmdk", 2, 1),
				),
			},
			{
				Config: testAccResourceVSphereVirtualMachineConfigMultiDevice(),
				Check: resource.ComposeTestCheckFunc(
					testAccResourceVSphereVirtualMachineCheckExists(true),
					testAccResourceVSphereVirtualMachineCheckDiskBus("terraform-test.vmdk", 0, 0),
					testAccResourceVSphereVirtualMachineCheckDiskBus("terraform-test_1.vmdk", 0, 1),
					testAccResourceVSphereVirtualMachineCheckDiskBus("terraform-test_2.vmdk", 0, 2),
				),
			},
		},
	})
}

func TestAccResourceVSphereVirtualMachine_cdrom(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
			testAccResourceVSphereVirtualMachinePreCheck(t)
		},
		Providers:    testAccProviders,
		CheckDestroy: testAccResourceVSphereVirtualMachineCheckExists(false),
		Steps: []resource.TestStep{
			{
				Config: testAccResourceVSphereVirtualMachineConfigCdrom(),
				Check: resource.ComposeTestCheckFunc(
					testAccResourceVSphereVirtualMachineCheckExists(true),
					testAccResourceVSphereVirtualMachineCheckCdrom(),
				),
			},
		},
	})
}

func TestAccResourceVSphereVirtualMachine_maximumNumberOfNICs(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
			testAccResourceVSphereVirtualMachinePreCheck(t)
		},
		Providers:    testAccProviders,
		CheckDestroy: testAccResourceVSphereVirtualMachineCheckExists(false),
		Steps: []resource.TestStep{
			{
				Config: testAccResourceVSphereVirtualMachineConfigMaxNIC(),
				Check: resource.ComposeTestCheckFunc(
					testAccResourceVSphereVirtualMachineCheckExists(true),
					testAccResourceVSphereVirtualMachineCheckNICCount(10),
				),
			},
		},
	})
}

func TestAccResourceVSphereVirtualMachine_upgradeCPUAndRam(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
			testAccResourceVSphereVirtualMachinePreCheck(t)
		},
		Providers:    testAccProviders,
		CheckDestroy: testAccResourceVSphereVirtualMachineCheckExists(false),
		Steps: []resource.TestStep{
			{
				Config: testAccResourceVSphereVirtualMachineConfigBasic(),
				Check: resource.ComposeTestCheckFunc(
					testAccResourceVSphereVirtualMachineCheckExists(true),
				),
			},
			{
				Config: testAccResourceVSphereVirtualMachineConfigBeefy(),
				Check: resource.ComposeTestCheckFunc(
					testAccResourceVSphereVirtualMachineCheckExists(true),
					testAccResourceVSphereVirtualMachineCheckCPUMem(4, 8192),
					// Since hot-add should be off, we expect that the VM was powered
					// off as a part of this step. This helps check the functionality
					// of the check for later tests as well.
					testAccResourceVSphereVirtualMachineCheckPowerOffEvent(true),
				),
			},
		},
	})
}

func TestAccResourceVSphereVirtualMachine_modifyAnnotation(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
			testAccResourceVSphereVirtualMachinePreCheck(t)
		},
		Providers:    testAccProviders,
		CheckDestroy: testAccResourceVSphereVirtualMachineCheckExists(false),
		Steps: []resource.TestStep{
			{
				Config: testAccResourceVSphereVirtualMachineConfigBasic(),
				Check: resource.ComposeTestCheckFunc(
					testAccResourceVSphereVirtualMachineCheckExists(true),
				),
			},
			{
				Config: testAccResourceVSphereVirtualMachineConfigBasicAnnotation(),
				Check: resource.ComposeTestCheckFunc(
					testAccResourceVSphereVirtualMachineCheckExists(true),
					testAccResourceVSphereVirtualMachineCheckAnnotation(),
					testAccResourceVSphereVirtualMachineCheckPowerOffEvent(false),
				),
			},
		},
	})
}

func TestAccResourceVSphereVirtualMachine_growDisk(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
			testAccResourceVSphereVirtualMachinePreCheck(t)
		},
		Providers:    testAccProviders,
		CheckDestroy: testAccResourceVSphereVirtualMachineCheckExists(false),
		Steps: []resource.TestStep{
			{
				Config: testAccResourceVSphereVirtualMachineConfigGrowDisk(10),
				Check: resource.ComposeTestCheckFunc(
					testAccResourceVSphereVirtualMachineCheckExists(true),
					testAccResourceVSphereVirtualMachineCheckDiskSize(10),
				),
			},
			{
				Config: testAccResourceVSphereVirtualMachineConfigGrowDisk(20),
				Check: resource.ComposeTestCheckFunc(
					testAccResourceVSphereVirtualMachineCheckExists(true),
					testAccResourceVSphereVirtualMachineCheckDiskSize(20),
				),
			},
		},
	})
}

func TestAccResourceVSphereVirtualMachine_swapSCSIBus(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
			testAccResourceVSphereVirtualMachinePreCheck(t)
		},
		Providers:    testAccProviders,
		CheckDestroy: testAccResourceVSphereVirtualMachineCheckExists(false),
		Steps: []resource.TestStep{
			{
				Config: testAccResourceVSphereVirtualMachineConfigBasic(),
				Check: resource.ComposeTestCheckFunc(
					testAccResourceVSphereVirtualMachineCheckExists(true),
					testAccResourceVSphereVirtualMachineCheckSCSIBus(virtualdevice.SubresourceControllerTypeParaVirtual),
				),
			},
			{
				Config: testAccResourceVSphereVirtualMachineConfigLsiLogicSAS(),
				Check: resource.ComposeTestCheckFunc(
					testAccResourceVSphereVirtualMachineCheckExists(true),
					testAccResourceVSphereVirtualMachineCheckSCSIBus(virtualdevice.SubresourceControllerTypeLsiLogicSAS),
				),
			},
		},
	})
}

func TestAccResourceVSphereVirtualMachine_extraConfig(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
			testAccResourceVSphereVirtualMachinePreCheck(t)
		},
		Providers:    testAccProviders,
		CheckDestroy: testAccResourceVSphereVirtualMachineCheckExists(false),
		Steps: []resource.TestStep{
			{
				Config: testAccResourceVSphereVirtualMachineConfigExtraConfig("foo", "bar"),
				Check: resource.ComposeTestCheckFunc(
					testAccResourceVSphereVirtualMachineCheckExists(true),
					testAccResourceVSphereVirtualMachineCheckExtraConfig("foo", "bar"),
				),
			},
		},
	})
}

func TestAccResourceVSphereVirtualMachine_extraConfigSwapKeys(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
			testAccResourceVSphereVirtualMachinePreCheck(t)
		},
		Providers:    testAccProviders,
		CheckDestroy: testAccResourceVSphereVirtualMachineCheckExists(false),
		Steps: []resource.TestStep{
			{
				Config: testAccResourceVSphereVirtualMachineConfigExtraConfig("foo", "bar"),
				Check: resource.ComposeTestCheckFunc(
					testAccResourceVSphereVirtualMachineCheckExists(true),
					testAccResourceVSphereVirtualMachineCheckExtraConfig("foo", "bar"),
				),
			},
			{
				Config: testAccResourceVSphereVirtualMachineConfigExtraConfig("baz", "qux"),
				Check: resource.ComposeTestCheckFunc(
					testAccResourceVSphereVirtualMachineCheckExists(true),
					testAccResourceVSphereVirtualMachineCheckExtraConfig("baz", "qux"),
					testAccResourceVSphereVirtualMachineCheckExtraConfigKeyMissing("foo"),
				),
			},
		},
	})
}

func TestAccResourceVSphereVirtualMachine_attachExistingVmdk(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
			testAccResourceVSphereVirtualMachinePreCheck(t)
		},
		Providers:    testAccProviders,
		CheckDestroy: testAccResourceVSphereVirtualMachineCheckExists(false),
		Steps: []resource.TestStep{
			{
				Config: testAccResourceVSphereVirtualMachineConfigExistingVmdk(),
				Check: resource.ComposeTestCheckFunc(
					testAccResourceVSphereVirtualMachineCheckExists(true),
					testAccResourceVSphereVirtualMachineCheckExistingVmdk(),
				),
			},
		},
	})
}

func TestAccResourceVSphereVirtualMachine_inFolder(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
			testAccResourceVSphereVirtualMachinePreCheck(t)
		},
		Providers:    testAccProviders,
		CheckDestroy: testAccResourceVSphereVirtualMachineCheckExists(false),
		Steps: []resource.TestStep{
			{
				Config: testAccResourceVSphereVirtualMachineConfigInFolder(),
				Check: resource.ComposeTestCheckFunc(
					testAccResourceVSphereVirtualMachineCheckExists(true),
					testAccResourceVSphereVirtualMachineCheckFolder("terraform-test-vms"),
				),
			},
		},
	})
}

func TestAccResourceVSphereVirtualMachine_moveToFolder(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
			testAccResourceVSphereVirtualMachinePreCheck(t)
		},
		Providers:    testAccProviders,
		CheckDestroy: testAccResourceVSphereVirtualMachineCheckExists(false),
		Steps: []resource.TestStep{
			{
				Config: testAccResourceVSphereVirtualMachineConfigBasic(),
				Check: resource.ComposeTestCheckFunc(
					testAccResourceVSphereVirtualMachineCheckExists(true),
				),
			},
			{
				Config: testAccResourceVSphereVirtualMachineConfigInFolder(),
				Check: resource.ComposeTestCheckFunc(
					testAccResourceVSphereVirtualMachineCheckExists(true),
					testAccResourceVSphereVirtualMachineCheckFolder("terraform-test-vms"),
				),
			},
		},
	})
}

func TestAccResourceVSphereVirtualMachine_staticMAC(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
			testAccResourceVSphereVirtualMachinePreCheck(t)
		},
		Providers:    testAccProviders,
		CheckDestroy: testAccResourceVSphereVirtualMachineCheckExists(false),
		Steps: []resource.TestStep{
			{
				Config: testAccResourceVSphereVirtualMachineConfigStaticMAC(),
				Check: resource.ComposeTestCheckFunc(
					testAccResourceVSphereVirtualMachineCheckExists(true),
					testAccResourceVSphereVirtualMachineCheckStaticMACAddr(),
				),
			},
		},
	})
}

func TestAccResourceVSphereVirtualMachine_singleTag(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
			testAccResourceVSphereVirtualMachinePreCheck(t)
		},
		Providers:    testAccProviders,
		CheckDestroy: testAccResourceVSphereVirtualMachineCheckExists(false),
		Steps: []resource.TestStep{
			{
				Config: testAccResourceVSphereVirtualMachineConfigSingleTag(),
				Check: resource.ComposeTestCheckFunc(
					testAccResourceVSphereVirtualMachineCheckExists(true),
					testAccResourceVSphereVirtualMachineCheckTags("terraform-test-tag"),
				),
			},
		},
	})
}

func TestAccResourceVSphereVirtualMachine_multipleTags(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
			testAccResourceVSphereVirtualMachinePreCheck(t)
		},
		Providers:    testAccProviders,
		CheckDestroy: testAccResourceVSphereVirtualMachineCheckExists(false),
		Steps: []resource.TestStep{
			{
				Config: testAccResourceVSphereVirtualMachineConfigMultiTag(),
				Check: resource.ComposeTestCheckFunc(
					testAccResourceVSphereVirtualMachineCheckExists(true),
					testAccResourceVSphereVirtualMachineCheckTags("terraform-test-tags-alt"),
				),
			},
		},
	})
}

func TestAccResourceVSphereVirtualMachine_switchTags(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
			testAccResourceVSphereVirtualMachinePreCheck(t)
		},
		Providers:    testAccProviders,
		CheckDestroy: testAccResourceVSphereVirtualMachineCheckExists(false),
		Steps: []resource.TestStep{
			{
				Config: testAccResourceVSphereVirtualMachineConfigSingleTag(),
				Check: resource.ComposeTestCheckFunc(
					testAccResourceVSphereVirtualMachineCheckExists(true),
					testAccResourceVSphereVirtualMachineCheckTags("terraform-test-tag"),
				),
			},
			{
				Config: testAccResourceVSphereVirtualMachineConfigMultiTag(),
				Check: resource.ComposeTestCheckFunc(
					testAccResourceVSphereVirtualMachineCheckExists(true),
					testAccResourceVSphereVirtualMachineCheckTags("terraform-test-tags-alt"),
				),
			},
		},
	})
}

func TestAccResourceVSphereVirtualMachine_renamedDiskInPlaceOfExisting(t *testing.T) {
	var state *terraform.State

	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
			testAccResourceVSphereVirtualMachinePreCheck(t)
		},
		Providers:    testAccProviders,
		CheckDestroy: testAccResourceVSphereVirtualMachineCheckExists(false),
		Steps: []resource.TestStep{
			{
				Config: testAccResourceVSphereVirtualMachineConfigBasic(),
				Check: resource.ComposeTestCheckFunc(
					copyStatePtr(&state),
					testAccResourceVSphereVirtualMachineCheckExists(true),
				),
			},
			{
				PreConfig: func() {
					if err := testRenameVMFirstDisk(state, "vm", "foobar.vmdk"); err != nil {
						panic(err)
					}
				},
				Config: testAccResourceVSphereVirtualMachineConfigBasic(),
				Check: resource.ComposeTestCheckFunc(
					testAccResourceVSphereVirtualMachineCheckExists(true),
					// The only real way we can check to see if this is actually
					// functional in the current test framework is by checking that
					// the file we renamed to was not deleted (this is due to a lack
					// of ability to check diff in the test framework right now).
					testCheckVMDiskFileExists("terraform-test.vmdk"),
					testCheckVMDiskFileExists("foobar.vmdk"),
				),
			},
			// The last step is a cleanup step. This assumes the test is
			// functional as the orphaned disk will be now detached and not
			// deleted when the VM is destroyed.
			{
				PreConfig: func() {
					if err := testDeleteVMDisk(state, "vm", "foobar.vmdk"); err != nil {
						panic(err)
					}
				},
				Config: testAccResourceVSphereVirtualMachineConfigBasic(),
				Check:  resource.ComposeTestCheckFunc(),
			},
		},
	})
}

func TestAccResourceVSphereVirtualMachine_blockComputedDiskName(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
			testAccResourceVSphereVirtualMachinePreCheck(t)
		},
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config:      testAccResourceVSphereVirtualMachineConfigComputedDisk(),
				ExpectError: regexp.MustCompile("disk label or name must be defined and cannot be computed"),
				PlanOnly:    true,
			},
			{
				Config: testAccResourceVSphereEmpty,
				Check:  resource.ComposeTestCheckFunc(),
			},
		},
	})
}

func TestAccResourceVSphereVirtualMachine_blockVAppSettingsOnNonClones(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
			testAccResourceVSphereVirtualMachinePreCheck(t)
		},
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config:      testAccResourceVSphereVirtualMachineVAppPropertiesNonClone(),
				ExpectError: regexp.MustCompile("vApp properties can only be set on cloned virtual machines"),
			},
			{
				Config: testAccResourceVSphereEmpty,
				Check:  resource.ComposeTestCheckFunc(),
			},
		},
	})
}

func TestAccResourceVSphereVirtualMachine_blockVAppSettingsOnNonClonesAfterCreation(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
			testAccResourceVSphereVirtualMachinePreCheck(t)
		},
		Providers:    testAccProviders,
		CheckDestroy: testAccResourceVSphereVirtualMachineCheckExists(false),
		Steps: []resource.TestStep{
			{
				Config: testAccResourceVSphereVirtualMachineConfigBasic(),
				Check: resource.ComposeTestCheckFunc(
					testAccResourceVSphereVirtualMachineCheckExists(true),
				),
			},
			{
				Config:      testAccResourceVSphereVirtualMachineVAppPropertiesNonClone(),
				ExpectError: regexp.MustCompile("this VM lacks a vApp configuration and cannot have vApp properties set on it"),
			},
			{
				Config: testAccResourceVSphereVirtualMachineConfigBasic(),
				Check: resource.ComposeTestCheckFunc(
					testAccResourceVSphereVirtualMachineCheckExists(true),
				),
			},
		},
	})
}

func TestAccResourceVSphereVirtualMachine_blockDiskLabelStartingWithOrphanedPrefix(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
			testAccResourceVSphereVirtualMachinePreCheck(t)
		},
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config:      testAccResourceVSphereVirtualMachineConfigBadOrphanedLabel(),
				ExpectError: regexp.MustCompile(regexp.QuoteMeta(`disk label "orphaned_disk_0" cannot start with "orphaned_disk_"`)),
				PlanOnly:    true,
			},
			{
				Config: testAccResourceVSphereEmpty,
				Check:  resource.ComposeTestCheckFunc(),
			},
		},
	})
}

func TestAccResourceVSphereVirtualMachine_createIntoEmptyClusterNoEnvironmentBrowser(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
			testAccResourceVSphereVirtualMachinePreCheck(t)
		},
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config:      testAccResourceVSphereVirtualMachineConfigBasicEmptyCluster(),
				ExpectError: regexp.MustCompile("compute resource .* is missing an Environment Browser\\. Check host, cluster, and vSphere license health of all associated resources and try again"),
			},
		},
	})
}

func TestAccResourceVSphereVirtualMachine_cloneFromTemplate(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
			testAccResourceVSphereVirtualMachinePreCheck(t)
		},
		Providers:    testAccProviders,
		CheckDestroy: testAccResourceVSphereVirtualMachineCheckExists(false),
		Steps: []resource.TestStep{
			{
				Config: testAccResourceVSphereVirtualMachineConfigClone(),
				Check: resource.ComposeTestCheckFunc(
					testAccResourceVSphereVirtualMachineCheckExists(true),
					resource.TestCheckResourceAttr("vsphere_virtual_machine.vm", "default_ip_address", os.Getenv("VSPHERE_IPV4_ADDRESS")),
				),
			},
		},
	})
}

func TestAccResourceVSphereVirtualMachine_cloneModifyDiskAndSCSITypeAtSameTime(t *testing.T) {
	var state *terraform.State

	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
			testAccResourceVSphereVirtualMachinePreCheck(t)
		},
		Providers:    testAccProviders,
		CheckDestroy: testAccResourceVSphereVirtualMachineCheckExists(false),
		Steps: []resource.TestStep{
			{
				Config: testAccResourceVSphereVirtualMachineConfigCloneChangeDiskAndSCSI(),
				Check: resource.ComposeTestCheckFunc(
					copyStatePtr(&state),
					testAccResourceVSphereVirtualMachineCheckExists(true),
					func(s *terraform.State) error {
						oldSize, _ := strconv.Atoi(state.RootModule().Resources["data.vsphere_virtual_machine.template"].Primary.Attributes["disks.0.size"])
						newSize := oldSize * 2
						return resource.TestCheckResourceAttr("vsphere_virtual_machine.vm", "disk.0.size", strconv.Itoa(newSize))(s)
					},
					func(s *terraform.State) error {
						oldBus := state.RootModule().Resources["data.vsphere_virtual_machine.template"].Primary.Attributes["scsi_type"]
						var expected string
						if oldBus == virtualdevice.SubresourceControllerTypeParaVirtual {
							expected = virtualdevice.SubresourceControllerTypeLsiLogicSAS
						} else {
							expected = virtualdevice.SubresourceControllerTypeParaVirtual
						}
						return resource.TestCheckResourceAttr("vsphere_virtual_machine.vm", "scsi_type", expected)(s)
					},
				),
			},
		},
	})
}

func TestAccResourceVSphereVirtualMachine_cloneMultiNICFromSingleNICTemplate(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
			testAccResourceVSphereVirtualMachinePreCheck(t)
		},
		Providers:    testAccProviders,
		CheckDestroy: testAccResourceVSphereVirtualMachineCheckExists(false),
		Steps: []resource.TestStep{
			{
				Config: testAccResourceVSphereVirtualMachineConfigCloneMultiNIC(),
				Check: resource.ComposeTestCheckFunc(
					testAccResourceVSphereVirtualMachineCheckExists(true),
					resource.TestCheckResourceAttr("vsphere_virtual_machine.vm", "default_ip_address", os.Getenv("VSPHERE_IPV4_ADDRESS")),
				),
			},
		},
	})
}

func TestAccResourceVSphereVirtualMachine_cloneWithDifferentTimezone(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
			testAccResourceVSphereVirtualMachinePreCheck(t)
		},
		Providers:    testAccProviders,
		CheckDestroy: testAccResourceVSphereVirtualMachineCheckExists(false),
		Steps: []resource.TestStep{
			{
				Config: testAccResourceVSphereVirtualMachineConfigCloneTimeZone("America/Vancouver"),
				Check: resource.ComposeTestCheckFunc(
					testAccResourceVSphereVirtualMachineCheckExists(true),
				),
			},
		},
	})
}

func TestAccResourceVSphereVirtualMachine_cloneWithBadTimezone(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
			testAccResourceVSphereVirtualMachinePreCheck(t)
		},
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config:      testAccResourceVSphereVirtualMachineConfigCloneTimeZone("Pacific Standard Time"),
				ExpectError: regexp.MustCompile("must be similar to America/Los_Angeles or other Linux/Unix TZ format"),
				PlanOnly:    true,
			},
			{
				Config: testAccResourceVSphereEmpty,
				Check:  resource.ComposeTestCheckFunc(),
			},
		},
	})
}

func TestAccResourceVSphereVirtualMachine_cloneWithBadEagerlyScrubWithLinkedClone(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
			testAccResourceVSphereVirtualMachinePreCheck(t)
		},
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config:      testAccResourceVSphereVirtualMachineConfigBadEager(),
				ExpectError: regexp.MustCompile("must have same value for eagerly_scrub as source when using linked_clone"),
				PlanOnly:    true,
			},
			{
				Config: testAccResourceVSphereEmpty,
				Check:  resource.ComposeTestCheckFunc(),
			},
		},
	})
}

func TestAccResourceVSphereVirtualMachine_cloneWithBadThinProvisionedWithLinkedClone(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
			testAccResourceVSphereVirtualMachinePreCheck(t)
		},
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config:      testAccResourceVSphereVirtualMachineConfigBadThin(),
				ExpectError: regexp.MustCompile("must have same value for thin_provisioned as source when using linked_clone"),
				PlanOnly:    true,
			},
			{
				Config: testAccResourceVSphereEmpty,
				Check:  resource.ComposeTestCheckFunc(),
			},
		},
	})
}

func TestAccResourceVSphereVirtualMachine_cloneWithBadSizeWithLinkedClone(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
			testAccResourceVSphereVirtualMachinePreCheck(t)
		},
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config:      testAccResourceVSphereVirtualMachineConfigBadSizeLinked(),
				ExpectError: regexp.MustCompile("must be the exact size of source when using linked_clone"),
				PlanOnly:    true,
			},
			{
				Config: testAccResourceVSphereEmpty,
				Check:  resource.ComposeTestCheckFunc(),
			},
		},
	})
}

func TestAccResourceVSphereVirtualMachine_cloneWithBadSizeWithoutLinkedClone(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
			testAccResourceVSphereVirtualMachinePreCheck(t)
		},
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config:      testAccResourceVSphereVirtualMachineConfigBadSizeUnlinked(),
				ExpectError: regexp.MustCompile("must be at least the same size of source when cloning"),
				PlanOnly:    true,
			},
			{
				Config: testAccResourceVSphereEmpty,
				Check:  resource.ComposeTestCheckFunc(),
			},
		},
	})
}

func TestAccResourceVSphereVirtualMachine_cloneUnsupportedVAppPropertiesOnCreate(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
			testAccResourceVSphereVirtualMachinePreCheck(t)
		},
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config:      testAccResourceVSphereVirtualMachineConfigCloneBadVAppSettings(),
				ExpectError: regexp.MustCompile("this VM lacks a vApp configuration and cannot have vApp properties set on it"),
			},
		},
	})
}

func TestAccResourceVSphereVirtualMachine_cloneUnsupportedVAppPropertiesOnUpdate(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
			testAccResourceVSphereVirtualMachinePreCheck(t)
		},
		Providers:    testAccProviders,
		CheckDestroy: testAccResourceVSphereVirtualMachineCheckExists(false),
		Steps: []resource.TestStep{
			{
				Config: testAccResourceVSphereVirtualMachineConfigClone(),
				Check: resource.ComposeTestCheckFunc(
					testAccResourceVSphereVirtualMachineCheckExists(true),
				),
			},
			{
				Config:      testAccResourceVSphereVirtualMachineConfigCloneBadVAppSettings(),
				ExpectError: regexp.MustCompile("this VM lacks a vApp configuration and cannot have vApp properties set on it"),
			},
		},
	})
}

func TestAccResourceVSphereVirtualMachine_cloneIntoEmptyCluster(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
			testAccResourceVSphereVirtualMachinePreCheck(t)
		},
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config:      testAccResourceVSphereVirtualMachineConfigCloneEmptyCluster(),
				ExpectError: regexp.MustCompile("compute resource .* is missing an Environment Browser\\. Check host, cluster, and vSphere license health of all associated resources and try again"),
				PlanOnly:    true,
			},
			{
				Config: testAccResourceVSphereEmpty,
				Check:  resource.ComposeTestCheckFunc(),
			},
		},
	})
}

func TestAccResourceVSphereVirtualMachine_cloneWithDifferentHostname(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
			testAccResourceVSphereVirtualMachinePreCheck(t)
		},
		Providers:    testAccProviders,
		CheckDestroy: testAccResourceVSphereVirtualMachineCheckExists(false),
		Steps: []resource.TestStep{
			{
				Config: testAccResourceVSphereVirtualMachineConfigCloneHostname(),
				Check: resource.ComposeTestCheckFunc(
					testAccResourceVSphereVirtualMachineCheckExists(true),
					testAccResourceVSphereVirtualMachineCheckHostname("terraform-test-renamed"),
				),
			},
		},
	})
}

func TestAccResourceVSphereVirtualMachine_cloneWithExtraDisks(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
			testAccResourceVSphereVirtualMachinePreCheck(t)
		},
		Providers:    testAccProviders,
		CheckDestroy: testAccResourceVSphereVirtualMachineCheckExists(false),
		Steps: []resource.TestStep{
			{
				Config: testAccResourceVSphereVirtualMachineConfigCloneExtraDisks(),
				Check: resource.ComposeTestCheckFunc(
					testAccResourceVSphereVirtualMachineCheckExists(true),
					testAccResourceVSphereVirtualMachineCheckExtraDisks(),
				),
			},
		},
	})
}

func TestAccResourceVSphereVirtualMachine_cloneWithCdrom(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
			testAccResourceVSphereVirtualMachinePreCheck(t)
		},
		Providers:    testAccProviders,
		CheckDestroy: testAccResourceVSphereVirtualMachineCheckExists(false),
		Steps: []resource.TestStep{
			{
				Config: testAccResourceVSphereVirtualMachineConfigCloneWithCdrom(),
				Check: resource.ComposeTestCheckFunc(
					testAccResourceVSphereVirtualMachineCheckExists(true),
					testAccResourceVSphereVirtualMachineCheckCdrom(),
				),
			},
		},
	})
}

func TestAccResourceVSphereVirtualMachine_cloneWithVAppProperties(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
			testAccResourceVSphereVirtualMachinePreCheck(t)
		},
		Providers:    testAccProviders,
		CheckDestroy: testAccResourceVSphereVirtualMachineCheckExists(false),
		Steps: []resource.TestStep{
			{
				Config: testAccResourceVSphereVirtualMachineConfigCloneWithVAppProperties(),
				Check: resource.ComposeTestCheckFunc(
					testAccResourceVSphereVirtualMachineCheckExists(true),
					testAccResourceVSphereVirtualMachineCheckVAppConfigKey("guestinfo.hostname", "terraform-test.test.internal"),
					testAccResourceVSphereVirtualMachineCheckVAppConfigKey("guestinfo.dns.server.0", os.Getenv("VSPHERE_DNS")),
				),
			},
			{
				Config: testAccResourceVSphereVirtualMachineConfigCloneUpdatingVAppProperties(),
				Check: resource.ComposeTestCheckFunc(
					testAccResourceVSphereVirtualMachineCheckExists(true),
					testAccResourceVSphereVirtualMachineCheckVAppConfigKey("guestinfo.hostname", "terraform-test.test.internal"),
					testAccResourceVSphereVirtualMachineCheckVAppConfigKey("guestinfo.dns.server.0", os.Getenv("VSPHERE_DNS")),
					testAccResourceVSphereVirtualMachineCheckVAppConfigKey("guestinfo.dns.server.1", "8.8.8.8"),
				),
			},
			{
				Config: testAccResourceVSphereVirtualMachineConfigCloneWithVAppProperties(),
				Check: resource.ComposeTestCheckFunc(
					testAccResourceVSphereVirtualMachineCheckExists(true),
					testAccResourceVSphereVirtualMachineCheckVAppConfigKey("guestinfo.hostname", "terraform-test.test.internal"),
					testAccResourceVSphereVirtualMachineCheckVAppConfigKey("guestinfo.dns.server.0", os.Getenv("VSPHERE_DNS")),
					testAccResourceVSphereVirtualMachineCheckVAppConfigKey("guestinfo.dns.server.1", ""),
				),
			},
		},
	})
}

func TestAccResourceVSphereVirtualMachine_cloneWithBadVAppPropertyOnCreate(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
			testAccResourceVSphereVirtualMachinePreCheck(t)
		},
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config:      testAccResourceVSphereVirtualMachineConfigCloneVAppPropertiesBadKey(),
				ExpectError: regexp.MustCompile(regexp.QuoteMeta("unsupported vApp properties in vapp.properties: [foo]")),
			},
			{
				Config: testAccResourceVSphereEmpty,
				Check:  resource.ComposeTestCheckFunc(),
			},
		},
	})
}

func TestAccResourceVSphereVirtualMachine_cloneWithBadVAppPropertyOnUpdate(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
			testAccResourceVSphereVirtualMachinePreCheck(t)
		},
		Providers:    testAccProviders,
		CheckDestroy: testAccResourceVSphereVirtualMachineCheckExists(false),
		Steps: []resource.TestStep{
			{
				Config: testAccResourceVSphereVirtualMachineConfigCloneWithVAppProperties(),
				Check: resource.ComposeTestCheckFunc(
					testAccResourceVSphereVirtualMachineCheckExists(true),
					testAccResourceVSphereVirtualMachineCheckVAppConfigKey("guestinfo.hostname", "terraform-test.test.internal"),
					testAccResourceVSphereVirtualMachineCheckVAppConfigKey("guestinfo.dns.server.0", os.Getenv("VSPHERE_DNS")),
				),
			},
			{
				Config:      testAccResourceVSphereVirtualMachineConfigCloneVAppPropertiesBadKey(),
				ExpectError: regexp.MustCompile(regexp.QuoteMeta("unsupported vApp properties in vapp.properties: [foo]")),
			},
		},
	})
}

func TestAccResourceVSphereVirtualMachine_cpuHotAdd(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
			testAccResourceVSphereVirtualMachinePreCheck(t)
		},
		Providers:    testAccProviders,
		CheckDestroy: testAccResourceVSphereVirtualMachineCheckExists(false),
		Steps: []resource.TestStep{
			{
				// Starting config
				Config: testAccResourceVSphereVirtualMachineConfigWithHotAdd(2, 2048, true, false, true),
				Check: resource.ComposeTestCheckFunc(
					testAccResourceVSphereVirtualMachineCheckExists(true),
					testAccResourceVSphereVirtualMachineCheckCPUMem(2, 2048),
				),
			},
			{
				// Add CPU w/hot-add
				Config: testAccResourceVSphereVirtualMachineConfigWithHotAdd(4, 2048, true, false, true),
				Check: resource.ComposeTestCheckFunc(
					testAccResourceVSphereVirtualMachineCheckExists(true),
					testAccResourceVSphereVirtualMachineCheckCPUMem(4, 2048),
					testAccResourceVSphereVirtualMachineCheckPowerOffEvent(false),
				),
			},
		},
	})
}

func TestAccResourceVSphereVirtualMachine_memoryHotAdd(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
			testAccResourceVSphereVirtualMachinePreCheck(t)
		},
		Providers:    testAccProviders,
		CheckDestroy: testAccResourceVSphereVirtualMachineCheckExists(false),
		Steps: []resource.TestStep{
			{
				// Starting config
				Config: testAccResourceVSphereVirtualMachineConfigWithHotAdd(2, 2048, true, false, true),
				Check: resource.ComposeTestCheckFunc(
					testAccResourceVSphereVirtualMachineCheckExists(true),
					testAccResourceVSphereVirtualMachineCheckCPUMem(2, 2048),
				),
			},
			{
				// Add memory with hot-add
				Config: testAccResourceVSphereVirtualMachineConfigWithHotAdd(2, 3072, true, false, true),
				Check: resource.ComposeTestCheckFunc(
					testAccResourceVSphereVirtualMachineCheckExists(true),
					testAccResourceVSphereVirtualMachineCheckCPUMem(2, 3072),
					testAccResourceVSphereVirtualMachineCheckPowerOffEvent(false),
				),
			},
		},
	})
}

func TestAccResourceVSphereVirtualMachine_dualStackIPv4AndIPv6(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
			testAccResourceVSphereVirtualMachinePreCheck(t)
		},
		Providers:    testAccProviders,
		CheckDestroy: testAccResourceVSphereVirtualMachineCheckExists(false),
		Steps: []resource.TestStep{
			{
				Config: testAccResourceVSphereVirtualMachineConfigDualStack(),
				Check: resource.ComposeTestCheckFunc(
					testAccResourceVSphereVirtualMachineCheckExists(true),
					testAccResourceVSphereVirtualMachineCheckNet("fd00::2", "32", "fd00::1"),
					testAccResourceVSphereVirtualMachineCheckNet(
						os.Getenv("VSPHERE_IPV4_ADDRESS"),
						os.Getenv("VSPHERE_IPV4_PREFIX"),
						os.Getenv("VSPHERE_IPV4_GATEWAY"),
					),
					resource.TestCheckResourceAttr("vsphere_virtual_machine.vm", "default_ip_address", os.Getenv("VSPHERE_IPV4_ADDRESS")),
				),
			},
		},
	})
}

func TestAccResourceVSphereVirtualMachine_IPv6Only(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
			testAccResourceVSphereVirtualMachinePreCheck(t)
		},
		Providers:    testAccProviders,
		CheckDestroy: testAccResourceVSphereVirtualMachineCheckExists(false),
		Steps: []resource.TestStep{
			{
				Config: testAccResourceVSphereVirtualMachineConfigIPv6Only(),
				Check: resource.ComposeTestCheckFunc(
					testAccResourceVSphereVirtualMachineCheckExists(true),
					testAccResourceVSphereVirtualMachineCheckNet("fd00::2", "32", "fd00::1"),
					resource.TestCheckResourceAttr("vsphere_virtual_machine.vm", "default_ip_address", "fd00::2"),
				),
			},
		},
	})
}

func TestAccResourceVSphereVirtualMachine_windowsTemplateCustomizationEventsAndProperIP(t *testing.T) {
	var state *terraform.State

	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
			testAccResourceVSphereVirtualMachinePreCheck(t)
		},
		Providers:    testAccProviders,
		CheckDestroy: testAccResourceVSphereVirtualMachineCheckExists(false),
		Steps: []resource.TestStep{
			{
				Config: testAccResourceVSphereVirtualMachineConfigWindows(),
				Check: resource.ComposeTestCheckFunc(
					copyStatePtr(&state),
					testAccResourceVSphereVirtualMachineCheckExists(true),
					testAccResourceVSphereVirtualMachineCheckCustomizationSucceeded(),
					testAccResourceVSphereVirtualMachineCheckNet(
						os.Getenv("VSPHERE_IPV4_ADDRESS"),
						os.Getenv("VSPHERE_IPV4_PREFIX"),
						os.Getenv("VSPHERE_IPV4_GATEWAY"),
					),
				),
			},
		},
	})
}

func TestAccResourceVSphereVirtualMachine_hostVMotion(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
			testAccResourceVSphereVirtualMachinePreCheck(t)
		},
		Providers:    testAccProviders,
		CheckDestroy: testAccResourceVSphereVirtualMachineCheckExists(false),
		Steps: []resource.TestStep{
			{
				Config: testAccResourceVSphereVirtualMachineConfigHostVMotion(os.Getenv("VSPHERE_ESXI_HOST")),
				Check: resource.ComposeTestCheckFunc(
					testAccResourceVSphereVirtualMachineCheckExists(true),
					testAccResourceVSphereVirtualMachineCheckHost(os.Getenv("VSPHERE_ESXI_HOST")),
				),
			},
			{
				Config: testAccResourceVSphereVirtualMachineConfigHostVMotion(os.Getenv("VSPHERE_ESXI_HOST2")),
				Check: resource.ComposeTestCheckFunc(
					testAccResourceVSphereVirtualMachineCheckExists(true),
					testAccResourceVSphereVirtualMachineCheckHost(os.Getenv("VSPHERE_ESXI_HOST2")),
				),
			},
		},
	})
}

func TestAccResourceVSphereVirtualMachine_resourcePoolVMotion(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
			testAccResourceVSphereVirtualMachinePreCheck(t)
		},
		Providers:    testAccProviders,
		CheckDestroy: testAccResourceVSphereVirtualMachineCheckExists(false),
		Steps: []resource.TestStep{
			{
				Config: testAccResourceVSphereVirtualMachineConfigResourcePoolVMotion(os.Getenv("VSPHERE_RESOURCE_POOL")),
				Check: resource.ComposeTestCheckFunc(
					testAccResourceVSphereVirtualMachineCheckExists(true),
					testAccResourceVSphereVirtualMachineCheckResourcePool(os.Getenv("VSPHERE_RESOURCE_POOL")),
				),
			},
			{
				Config: testAccResourceVSphereVirtualMachineConfigResourcePoolVMotion(fmt.Sprintf("%s/Resources", os.Getenv("VSPHERE_CLUSTER"))),
				Check: resource.ComposeTestCheckFunc(
					testAccResourceVSphereVirtualMachineCheckExists(true),
					testAccResourceVSphereVirtualMachineCheckResourcePool(fmt.Sprintf("%s/Resources", os.Getenv("VSPHERE_CLUSTER"))),
				),
			},
		},
	})
}

func TestAccResourceVSphereVirtualMachine_storageVMotionGlobalSetting(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
			testAccResourceVSphereVirtualMachinePreCheck(t)
		},
		Providers:    testAccProviders,
		CheckDestroy: testAccResourceVSphereVirtualMachineCheckExists(false),
		Steps: []resource.TestStep{
			{
				Config: testAccResourceVSphereVirtualMachineConfigStorageVMotionGlobal(os.Getenv("VSPHERE_DATASTORE")),
				Check: resource.ComposeTestCheckFunc(
					testAccResourceVSphereVirtualMachineCheckExists(true),
					testAccResourceVSphereVirtualMachineCheckVmxDatastore(os.Getenv("VSPHERE_DATASTORE")),
					testAccResourceVSphereVirtualMachineCheckVmdkDatastore("terraform-test.vmdk", os.Getenv("VSPHERE_DATASTORE")),
				),
			},
			{
				Config: testAccResourceVSphereVirtualMachineConfigStorageVMotionGlobal(os.Getenv("VSPHERE_DATASTORE2")),
				Check: resource.ComposeTestCheckFunc(
					testAccResourceVSphereVirtualMachineCheckExists(true),
					testAccResourceVSphereVirtualMachineCheckVmxDatastore(os.Getenv("VSPHERE_DATASTORE2")),
					testAccResourceVSphereVirtualMachineCheckVmdkDatastore("terraform-test.vmdk", os.Getenv("VSPHERE_DATASTORE2")),
				),
			},
		},
	})
}

func TestAccResourceVSphereVirtualMachine_storageVMotionSingleDisk(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
			testAccResourceVSphereVirtualMachinePreCheck(t)
		},
		Providers:    testAccProviders,
		CheckDestroy: testAccResourceVSphereVirtualMachineCheckExists(false),
		Steps: []resource.TestStep{
			{
				Config: testAccResourceVSphereVirtualMachineConfigStorageVMotionSingleDisk(os.Getenv("VSPHERE_DATASTORE")),
				Check: resource.ComposeTestCheckFunc(
					testAccResourceVSphereVirtualMachineCheckExists(true),
					testAccResourceVSphereVirtualMachineCheckVmxDatastore(os.Getenv("VSPHERE_DATASTORE")),
					testAccResourceVSphereVirtualMachineCheckVmdkDatastore("terraform-test.vmdk", os.Getenv("VSPHERE_DATASTORE")),
					testAccResourceVSphereVirtualMachineCheckVmdkDatastore("terraform-test_1.vmdk", os.Getenv("VSPHERE_DATASTORE")),
				),
			},
			{
				Config: testAccResourceVSphereVirtualMachineConfigStorageVMotionSingleDisk(os.Getenv("VSPHERE_DATASTORE2")),
				Check: resource.ComposeTestCheckFunc(
					testAccResourceVSphereVirtualMachineCheckExists(true),
					testAccResourceVSphereVirtualMachineCheckVmxDatastore(os.Getenv("VSPHERE_DATASTORE")),
					testAccResourceVSphereVirtualMachineCheckVmdkDatastore("terraform-test.vmdk", os.Getenv("VSPHERE_DATASTORE")),
					testAccResourceVSphereVirtualMachineCheckVmdkDatastore("terraform-test_1.vmdk", os.Getenv("VSPHERE_DATASTORE2")),
				),
			},
		},
	})
}

func TestAccResourceVSphereVirtualMachine_storageVMotionPinDatastore(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
			testAccResourceVSphereVirtualMachinePreCheck(t)
		},
		Providers:    testAccProviders,
		CheckDestroy: testAccResourceVSphereVirtualMachineCheckExists(false),
		Steps: []resource.TestStep{
			{
				Config: testAccResourceVSphereVirtualMachineConfigStorageVMotionPinDatastore(os.Getenv("VSPHERE_DATASTORE")),
				Check: resource.ComposeTestCheckFunc(
					testAccResourceVSphereVirtualMachineCheckExists(true),
					testAccResourceVSphereVirtualMachineCheckVmxDatastore(os.Getenv("VSPHERE_DATASTORE")),
					testAccResourceVSphereVirtualMachineCheckVmdkDatastore("terraform-test.vmdk", os.Getenv("VSPHERE_DATASTORE")),
					testAccResourceVSphereVirtualMachineCheckVmdkDatastore("terraform-test_1.vmdk", os.Getenv("VSPHERE_DATASTORE")),
				),
			},
			{
				Config: testAccResourceVSphereVirtualMachineConfigStorageVMotionPinDatastore(os.Getenv("VSPHERE_DATASTORE2")),
				Check: resource.ComposeTestCheckFunc(
					testAccResourceVSphereVirtualMachineCheckExists(true),
					testAccResourceVSphereVirtualMachineCheckVmxDatastore(os.Getenv("VSPHERE_DATASTORE2")),
					testAccResourceVSphereVirtualMachineCheckVmdkDatastore("terraform-test.vmdk", os.Getenv("VSPHERE_DATASTORE2")),
					testAccResourceVSphereVirtualMachineCheckVmdkDatastore("terraform-test_1.vmdk", os.Getenv("VSPHERE_DATASTORE")),
				),
			},
		},
	})
}

func TestAccResourceVSphereVirtualMachine_storageVMotionRenamedVirtualMachine(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
			testAccResourceVSphereVirtualMachinePreCheck(t)
		},
		Providers:    testAccProviders,
		CheckDestroy: testAccResourceVSphereVirtualMachineCheckExists(false),
		Steps: []resource.TestStep{
			{
				Config: testAccResourceVSphereVirtualMachineConfigStorageVMotionRename("terraform-test", os.Getenv("VSPHERE_DATASTORE")),
				Check: resource.ComposeTestCheckFunc(
					testAccResourceVSphereVirtualMachineCheckExists(true),
					testAccResourceVSphereVirtualMachineCheckVmxDatastore(os.Getenv("VSPHERE_DATASTORE")),
					testAccResourceVSphereVirtualMachineCheckVmdkDatastore("terraform-test.vmdk", os.Getenv("VSPHERE_DATASTORE")),
				),
			},
			{
				Config: testAccResourceVSphereVirtualMachineConfigStorageVMotionRename("foobar-test", os.Getenv("VSPHERE_DATASTORE")),
				Check: resource.ComposeTestCheckFunc(
					testAccResourceVSphereVirtualMachineCheckExists(true),
					testAccResourceVSphereVirtualMachineCheckVmxDatastore(os.Getenv("VSPHERE_DATASTORE")),
					testAccResourceVSphereVirtualMachineCheckVmdkDatastore("terraform-test.vmdk", os.Getenv("VSPHERE_DATASTORE")),
				),
			},
			{
				Config: testAccResourceVSphereVirtualMachineConfigStorageVMotionRename("foobar-test", os.Getenv("VSPHERE_DATASTORE2")),
				Check: resource.ComposeTestCheckFunc(
					testAccResourceVSphereVirtualMachineCheckExists(true),
					testAccResourceVSphereVirtualMachineCheckVmxDatastore(os.Getenv("VSPHERE_DATASTORE2")),
					testAccResourceVSphereVirtualMachineCheckVmdkDatastore("foobar-test.vmdk", os.Getenv("VSPHERE_DATASTORE2")),
				),
			},
		},
	})
}

func TestAccResourceVSphereVirtualMachine_storageVMotionLinkedClones(t *testing.T) {
	var state *terraform.State

	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
			testAccResourceVSphereVirtualMachinePreCheck(t)
		},
		Providers:    testAccProviders,
		CheckDestroy: testAccResourceVSphereVirtualMachineCheckExists(false),
		Steps: []resource.TestStep{
			{
				Config: testAccResourceVSphereVirtualMachineConfigStorageVMotionLinkedClone(os.Getenv("VSPHERE_DATASTORE")),
				Check: resource.ComposeTestCheckFunc(
					testAccResourceVSphereVirtualMachineCheckExists(true),
					testAccResourceVSphereVirtualMachineCheckVmxDatastore(os.Getenv("VSPHERE_DATASTORE")),
					testAccResourceVSphereVirtualMachineCheckVmdkDatastore("terraform-test.vmdk", os.Getenv("VSPHERE_DATASTORE")),
				),
			},
			{
				Config: testAccResourceVSphereVirtualMachineConfigStorageVMotionLinkedClone(os.Getenv("VSPHERE_DATASTORE2")),
				Check: resource.ComposeTestCheckFunc(
					copyStatePtr(&state),
					testAccResourceVSphereVirtualMachineCheckExists(true),
					testAccResourceVSphereVirtualMachineCheckVmxDatastore(os.Getenv("VSPHERE_DATASTORE2")),
					func(s *terraform.State) error {
						filename := path.Base(state.RootModule().Resources["vsphere_virtual_machine.vm"].Primary.Attributes["disk.0.path"])
						return testAccResourceVSphereVirtualMachineCheckVmdkDatastore(filename, os.Getenv("VSPHERE_DATASTORE2"))(s)
					},
				),
			},
		},
	})
}

func TestAccResourceVSphereVirtualMachine_storageVMotionBlockExternallyAttachedDisks(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
			testAccResourceVSphereVirtualMachinePreCheck(t)
		},
		Providers:    testAccProviders,
		CheckDestroy: testAccResourceVSphereVirtualMachineCheckExists(false),
		Steps: []resource.TestStep{
			{
				Config: testAccResourceVSphereVirtualMachineConfigStorageVMotionAttachedDisk(os.Getenv("VSPHERE_DATASTORE")),
				Check: resource.ComposeTestCheckFunc(
					testAccResourceVSphereVirtualMachineCheckExists(true),
					testAccResourceVSphereVirtualMachineCheckVmxDatastore(os.Getenv("VSPHERE_DATASTORE")),
					testAccResourceVSphereVirtualMachineCheckVmdkDatastore("terraform-test.vmdk", os.Getenv("VSPHERE_DATASTORE")),
				),
			},
			{
				Config: testAccResourceVSphereVirtualMachineConfigStorageVMotionAttachedDisk(os.Getenv("VSPHERE_DATASTORE2")),
				ExpectError: regexp.MustCompile(regexp.QuoteMeta(
					fmt.Sprintf("externally attached disk %q cannot be migrated", testAccResourceVSphereVirtualMachineDiskNameExtraVmdk),
				)),
			},
		},
	})
}

func TestAccResourceVSphereVirtualMachine_singleCustomAttribute(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
			testAccResourceVSphereVirtualMachinePreCheck(t)
		},
		Providers:    testAccProviders,
		CheckDestroy: testAccResourceVSphereVirtualMachineCheckExists(false),
		Steps: []resource.TestStep{
			{
				Config: testAccResourceVSphereVirtualMachineConfigWithCustomAttribute(),
				Check: resource.ComposeTestCheckFunc(
					testAccResourceVSphereVirtualMachineCheckExists(true),
					testAccResourceVSphereVirtualMachineCheckCustomAttributes(),
				),
			},
		},
	})
}

func TestAccResourceVSphereVirtualMachine_multiCustomAttribute(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
			testAccResourceVSphereVirtualMachinePreCheck(t)
		},
		Providers:    testAccProviders,
		CheckDestroy: testAccResourceVSphereVirtualMachineCheckExists(false),
		Steps: []resource.TestStep{
			{
				Config: testAccResourceVSphereVirtualMachineConfigWithMultiCustomAttribute(),
				Check: resource.ComposeTestCheckFunc(
					testAccResourceVSphereVirtualMachineCheckExists(true),
					testAccResourceVSphereVirtualMachineCheckCustomAttributes(),
				),
			},
		},
	})
}

func TestAccResourceVSphereVirtualMachine_switchCustomAttribute(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
			testAccResourceVSphereVirtualMachinePreCheck(t)
		},
		Providers:    testAccProviders,
		CheckDestroy: testAccResourceVSphereVirtualMachineCheckExists(false),
		Steps: []resource.TestStep{
			{
				Config: testAccResourceVSphereVirtualMachineConfigWithCustomAttribute(),
				Check: resource.ComposeTestCheckFunc(
					testAccResourceVSphereVirtualMachineCheckExists(true),
					testAccResourceVSphereVirtualMachineCheckCustomAttributes(),
				),
			},
			{
				Config: testAccResourceVSphereVirtualMachineConfigWithMultiCustomAttribute(),
				Check: resource.ComposeTestCheckFunc(
					testAccResourceVSphereVirtualMachineCheckExists(true),
					testAccResourceVSphereVirtualMachineCheckCustomAttributes(),
				),
			},
		},
	})
}

func TestAccResourceVSphereVirtualMachine_transitionToLabel(t *testing.T) {
	// TODO: Remove this test in 2.0
	var state *terraform.State

	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
			testAccResourceVSphereVirtualMachinePreCheck(t)
		},
		Providers:    testAccProviders,
		CheckDestroy: testAccResourceVSphereVirtualMachineCheckExists(false),
		Steps: []resource.TestStep{
			{
				Config: testAccResourceVSphereVirtualMachineConfigBasicDiskNameOrLabel("name"),
				Check: resource.ComposeTestCheckFunc(
					copyState(&state),
					testAccResourceVSphereVirtualMachineCheckExists(true),
				),
			},
			{
				Config: testAccResourceVSphereVirtualMachineConfigBasicDiskNameOrLabel("label"),
				Check: resource.ComposeTestCheckFunc(
					testAccResourceVSphereVirtualMachineCheckExists(true),
					func(s *terraform.State) error {
						uuid := state.RootModule().Resources["vsphere_virtual_machine.vm"].Primary.Attributes["disk.0.uuid"]
						return resource.TestCheckResourceAttr("vsphere_virtual_machine.vm", "disk.0.uuid", uuid)(s)
					},
				),
			},
		},
	})
}

func TestAccResourceVSphereVirtualMachine_preventRevertToName(t *testing.T) {
	// TODO: Remove this test in 2.0
	var state *terraform.State

	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
			testAccResourceVSphereVirtualMachinePreCheck(t)
		},
		Providers:    testAccProviders,
		CheckDestroy: testAccResourceVSphereVirtualMachineCheckExists(false),
		Steps: []resource.TestStep{
			{
				Config: testAccResourceVSphereVirtualMachineConfigBasicDiskNameOrLabel("name"),
				Check: resource.ComposeTestCheckFunc(
					copyState(&state),
					testAccResourceVSphereVirtualMachineCheckExists(true),
				),
			},
			{
				Config: testAccResourceVSphereVirtualMachineConfigBasicDiskNameOrLabel("label"),
				Check: resource.ComposeTestCheckFunc(
					testAccResourceVSphereVirtualMachineCheckExists(true),
					func(s *terraform.State) error {
						uuid := state.RootModule().Resources["vsphere_virtual_machine.vm"].Primary.Attributes["disk.0.uuid"]
						return resource.TestCheckResourceAttr("vsphere_virtual_machine.vm", "disk.0.uuid", uuid)(s)
					},
				),
			},
			{
				Config:      testAccResourceVSphereVirtualMachineConfigBasicDiskNameOrLabel("name"),
				ExpectError: regexp.MustCompile("cannot migrate from label to name"),
			},
		},
	})
}

func TestAccResourceVSphereVirtualMachine_transitionToLabelAttachedDisk(t *testing.T) {
	// TODO: Remove this test in 2.0
	var state *terraform.State

	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
			testAccResourceVSphereVirtualMachinePreCheck(t)
		},
		Providers:    testAccProviders,
		CheckDestroy: testAccResourceVSphereVirtualMachineCheckExists(false),
		Steps: []resource.TestStep{
			{
				Config: testAccResourceVSphereVirtualMachineConfigExistingVmdkWithName(),
				Check: resource.ComposeTestCheckFunc(
					copyState(&state),
					testAccResourceVSphereVirtualMachineCheckExists(true),
				),
			},
			{
				Config: testAccResourceVSphereVirtualMachineConfigExistingVmdkWithLabel(),
				Check: resource.ComposeTestCheckFunc(
					testAccResourceVSphereVirtualMachineCheckExists(true),
					func(s *terraform.State) error {
						uuid := state.RootModule().Resources["vsphere_virtual_machine.vm"].Primary.Attributes["disk.1.uuid"]
						if err := resource.TestCheckResourceAttr("vsphere_virtual_machine.vm", "disk.1.uuid", uuid)(s); err != nil {
							return err
						}
						return resource.TestCheckResourceAttr("vsphere_virtual_machine.vm", "disk.1.attach", "true")(s)
					},
				),
			},
		},
	})
}

func TestAccResourceVSphereVirtualMachine_import(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
			testAccResourceVSphereVirtualMachinePreCheck(t)
		},
		Providers:    testAccProviders,
		CheckDestroy: testAccResourceVSphereVirtualMachineCheckExists(false),
		Steps: []resource.TestStep{
			{
				Config: testAccResourceVSphereVirtualMachineConfigBasic(),
				Check: resource.ComposeTestCheckFunc(
					testAccResourceVSphereVirtualMachineCheckExists(true),
				),
			},
			{
				ResourceName:      "vsphere_virtual_machine.vm",
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateVerifyIgnore: []string{
					"disk",
					"imported",
				},
				ImportStateIdFunc: func(s *terraform.State) (string, error) {
					vm, err := testGetVirtualMachine(s, "vm")
					if err != nil {
						return "", err
					}
					return vm.InventoryPath, nil
				},
				Config: testAccResourceVSphereVirtualMachineConfigBasic(),
				Check: resource.ComposeTestCheckFunc(
					testAccResourceVSphereVirtualMachineCheckExists(true),
				),
			},
		},
	})
}

func TestAccResourceVSphereVirtualMachine_importWithMultipleDisksAtDifferentSCSISlots(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
			testAccResourceVSphereVirtualMachinePreCheck(t)
		},
		Providers:    testAccProviders,
		CheckDestroy: testAccResourceVSphereVirtualMachineCheckExists(false),
		Steps: []resource.TestStep{
			{
				Config: testAccResourceVSphereVirtualMachineConfigMultiHighBus(),
				Check: resource.ComposeTestCheckFunc(
					testAccResourceVSphereVirtualMachineCheckExists(true),
				),
			},
			{
				ResourceName:      "vsphere_virtual_machine.vm",
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateVerifyIgnore: []string{
					"disk",
					"imported",
				},
				ImportStateIdFunc: func(s *terraform.State) (string, error) {
					vm, err := testGetVirtualMachine(s, "vm")
					if err != nil {
						return "", err
					}
					return vm.InventoryPath, nil
				},
				Config: testAccResourceVSphereVirtualMachineConfigMultiHighBus(),
				Check: resource.ComposeTestCheckFunc(
					testAccResourceVSphereVirtualMachineCheckExists(true),
				),
			},
		},
	})
}

func testAccResourceVSphereVirtualMachinePreCheck(t *testing.T) {
	// Note that VSPHERE_USE_LINKED_CLONE is also a variable and its presence
	// speeds up tests greatly, but it's not a necessary variable, so we don't
	// enforce it here.
	if os.Getenv("VSPHERE_DATACENTER") == "" {
		t.Skip("set VSPHERE_DATACENTER to run vsphere_virtual_machine acceptance tests")
	}
	if os.Getenv("VSPHERE_CLUSTER") == "" {
		t.Skip("set VSPHERE_CLUSTER to run vsphere_virtual_machine acceptance tests")
	}
	if os.Getenv("VSPHERE_RESOURCE_POOL") == "" {
		t.Skip("set VSPHERE_RESOURCE_POOL to run vsphere_virtual_machine acceptance tests")
	}
	if os.Getenv("VSPHERE_NETWORK_LABEL") == "" {
		t.Skip("set VSPHERE_NETWORK_LABEL to run vsphere_virtual_machine acceptance tests")
	}
	if os.Getenv("VSPHERE_NETWORK_LABEL_PXE") == "" {
		t.Skip("set VSPHERE_NETWORK_LABEL_PXE to run vsphere_virtual_machine acceptance tests")
	}
	if os.Getenv("VSPHERE_IPV4_ADDRESS") == "" {
		t.Skip("set VSPHERE_IPV4_ADDRESS to run vsphere_virtual_machine acceptance tests")
	}
	if os.Getenv("VSPHERE_IPV4_PREFIX") == "" {
		t.Skip("set VSPHERE_IPV4_PREFIX to run vsphere_virtual_machine acceptance tests")
	}
	if os.Getenv("VSPHERE_IPV4_GATEWAY") == "" {
		t.Skip("set VSPHERE_IPV4_GATEWAY to run vsphere_virtual_machine acceptance tests")
	}
	if os.Getenv("VSPHERE_DNS") == "" {
		t.Skip("set VSPHERE_DNS to run vsphere_virtual_machine acceptance tests")
	}
	if os.Getenv("VSPHERE_DATASTORE") == "" {
		t.Skip("set VSPHERE_DATASTORE to run vsphere_virtual_machine acceptance tests")
	}
	if os.Getenv("VSPHERE_DATASTORE2") == "" {
		t.Skip("set VSPHERE_DATASTORE2 to run vsphere_virtual_machine acceptance tests")
	}
	if os.Getenv("VSPHERE_TEMPLATE") == "" {
		t.Skip("set VSPHERE_TEMPLATE to run vsphere_virtual_machine acceptance tests")
	}
	if os.Getenv("VSPHERE_TEMPLATE_WINDOWS") == "" {
		t.Skip("set VSPHERE_TEMPLATE_WINDOWS to run vsphere_virtual_machine acceptance tests")
	}
	if os.Getenv("VSPHERE_TEMPLATE_COREOS") == "" {
		t.Skip("set VSPHERE_TEMPLATE_COREOS to run vsphere_virtual_machine acceptance tests")
	}
	if os.Getenv("VSPHERE_ESXI_HOST") == "" {
		t.Skip("set VSPHERE_ESXI_HOST to run vsphere_virtual_machine acceptance tests")
	}
	if os.Getenv("VSPHERE_ESXI_HOST2") == "" {
		t.Skip("set VSPHERE_ESXI_HOST2 to run vsphere_virtual_machine acceptance tests")
	}
	if os.Getenv("VSPHERE_ESXI_HOST3") == "" {
		t.Skip("set VSPHERE_ESXI_HOST3 to run vsphere_virtual_machine acceptance tests")
	}
	if os.Getenv("VSPHERE_HOST_NIC0") == "" {
		t.Skip("set VSPHERE_HOST_NIC0 to run vsphere_virtual_machine acceptance tests")
	}
	if os.Getenv("VSPHERE_DS_VMFS_DISK0") == "" {
		t.Skip("set VSPHERE_DS_VMFS_DISK0 to run vsphere_virtual_machine acceptance tests")
	}
	if os.Getenv("VSPHERE_DS_VMFS_DISK1") == "" {
		t.Skip("set VSPHERE_DS_VMFS_DISK1 to run vsphere_virtual_machine acceptance tests")
	}
	if os.Getenv("VSPHERE_DS_VMFS_DISK2") == "" {
		t.Skip("set VSPHERE_DS_VMFS_DISK2 to run vsphere_virtual_machine acceptance tests")
	}
}

func testAccResourceVSphereVirtualMachineCheckExists(expected bool) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		_, err := testGetVirtualMachine(s, "vm")
		if err != nil {
			if ok, _ := regexp.MatchString("virtual machine with UUID \"[-a-f0-9]+\" not found", err.Error()); ok && !expected {
				// Expected missing
				return nil
			}
			return err
		}
		if !expected {
			return errors.New("expected VM to be missing")
		}
		return nil
	}
}

func testAccResourceVSphereVirtualMachineCheckVAppConfigKey(key, value string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		props, err := testGetVirtualMachineProperties(s, "vm")
		if err != nil {
			return err
		}
		actual := props.Config.VAppConfig.GetVmConfigInfo().Property
		for _, prop := range actual {
			if prop.Id == key && prop.Value != value {
				return fmt.Errorf("expected vAppConfig property %s to have value %s, got %s", key, value, prop.Value)
			}
		}
		return nil
	}
}

// testAccResourceVSphereVirtualMachineCheckPowerState is a check to check for
// a VirtualMachine's power state.
func testAccResourceVSphereVirtualMachineCheckPowerState(expected types.VirtualMachinePowerState) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		props, err := testGetVirtualMachineProperties(s, "vm")
		if err != nil {
			return err
		}
		actual := props.Runtime.PowerState
		if expected != actual {
			return fmt.Errorf("expected power state to be %s, got %s", expected, actual)
		}
		return nil
	}
}

// testAccResourceVSphereVirtualMachineCheckHostname is a check to check for a
// VirtualMachine's hostname. The check uses guest info, so VMware tools needs
// to be installed.
func testAccResourceVSphereVirtualMachineCheckHostname(expected string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		props, err := testGetVirtualMachineProperties(s, "vm")
		if err != nil {
			return err
		}
		actual := props.Guest.HostName
		if expected != actual {
			return fmt.Errorf("expected hostname to be %s, got %s", expected, actual)
		}
		return nil
	}
}

// testAccResourceVSphereVirtualMachineCheckExtraDisks is a check for proper
// parameters on the vsphere_virtual_machine extra disks test. This is a very
// specific check that checks for the specific disk devices and respective
// backings, and expects them in the exact order outlined in the function.
func testAccResourceVSphereVirtualMachineCheckExtraDisks() resource.TestCheckFunc {
	return func(s *terraform.State) error {
		props, err := testGetVirtualMachineProperties(s, "vm")
		if err != nil {
			return err
		}

		var foundEager, foundLazy, foundThin bool

		for _, dev := range props.Config.Hardware.Device {
			if disk, ok := dev.(*types.VirtualDisk); ok {
				if info, ok := disk.Backing.(*types.VirtualDiskFlatVer2BackingInfo); ok {
					var eager bool
					if info.EagerlyScrub != nil {
						eager = *info.EagerlyScrub
					}
					switch {
					case strings.HasSuffix(info.FileName, testAccResourceVSphereVirtualMachineDiskNameEager) && eager:
						foundEager = true
					case strings.HasSuffix(info.FileName, testAccResourceVSphereVirtualMachineDiskNameLazy) && !eager:
						foundLazy = true
					case strings.HasSuffix(info.FileName, testAccResourceVSphereVirtualMachineDiskNameThin) && *info.ThinProvisioned:
						foundThin = true
					}
				}
			}
		}

		if !foundEager {
			return fmt.Errorf("could not locate disk: %s", testAccResourceVSphereVirtualMachineDiskNameEager)
		}
		if !foundLazy {
			return fmt.Errorf("could not locate disk: %s", testAccResourceVSphereVirtualMachineDiskNameLazy)
		}
		if !foundThin {
			return fmt.Errorf("could not locate disk: %s", testAccResourceVSphereVirtualMachineDiskNameThin)
		}

		return nil
	}
}

// testAccResourceVSphereVirtualMachineCheckDiskBus is a check that looks for a
// disk with a specific name at a specific SCSI bus number and unit number.
func testAccResourceVSphereVirtualMachineCheckDiskBus(name string, expectedBus, expectedUnit int) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		props, err := testGetVirtualMachineProperties(s, "vm")
		if err != nil {
			return err
		}

		for _, dev := range props.Config.Hardware.Device {
			if disk, ok := dev.(*types.VirtualDisk); ok {
				if info, ok := disk.Backing.(*types.VirtualDiskFlatVer2BackingInfo); ok {
					dp := new(object.DatastorePath)
					if ok := dp.FromString(info.FileName); !ok {
						return fmt.Errorf("could not parse datastore path %q", info.FileName)
					}
					if path.Base(dp.Path) != name {
						continue
					}
					l := object.VirtualDeviceList(props.Config.Hardware.Device)
					ctlr := l.FindByKey(disk.ControllerKey)
					if ctlr == nil {
						return fmt.Errorf("could not find controller with key %d for disk %q", disk.ControllerKey, name)
					}
					sc, ok := ctlr.(types.BaseVirtualSCSIController)
					if !ok {
						return fmt.Errorf("disk %q not attached to a SCSI controller (actual: %T)", name, ctlr)
					}
					if sc.GetVirtualSCSIController().BusNumber != int32(expectedBus) {
						return fmt.Errorf("disk %q: Expected controller bus to be %d, got %d", name, expectedBus, sc.GetVirtualSCSIController().BusNumber)
					}
					if disk.UnitNumber == nil {
						return fmt.Errorf("disk %q has no unit number", name)
					}
					if *disk.UnitNumber != int32(expectedUnit) {
						return fmt.Errorf("disk %q: Expected unit number to be %d, got %d", name, expectedUnit, *disk.UnitNumber)
					}
					return nil
				}
			}
		}

		return fmt.Errorf("could not find disk path %q", name)
	}
}

// testAccResourceVSphereVirtualMachineCheckFolder checks to make sure a
// virtual machine's folder matches the folder supplied with expected.
func testAccResourceVSphereVirtualMachineCheckFolder(expected string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		vm, err := testGetVirtualMachine(s, "vm")
		if err != nil {
			return fmt.Errorf("bad: %s", err)
		}
		expected, err := folder.RootPathParticleVM.PathFromNewRoot(vm.InventoryPath, folder.RootPathParticleVM, expected)
		actual := path.Dir(vm.InventoryPath)
		if err != nil {
			return fmt.Errorf("bad: %s", err)
		}
		if expected != actual {
			return fmt.Errorf("expected path to be %s, got %s", expected, actual)
		}
		return nil
	}
}

// testAccResourceVSphereVirtualMachineCheckExistingVmdk is a check to make
// sure that the appropriate disk is attached in the existing VMDK test.
func testAccResourceVSphereVirtualMachineCheckExistingVmdk() resource.TestCheckFunc {
	return func(s *terraform.State) error {
		props, err := testGetVirtualMachineProperties(s, "vm")
		if err != nil {
			return err
		}

		expected := testAccResourceVSphereVirtualMachineDiskNameExtraVmdk

		for _, dev := range props.Config.Hardware.Device {
			if disk, ok := dev.(*types.VirtualDisk); ok {
				if info, ok := disk.Backing.(*types.VirtualDiskFlatVer2BackingInfo); ok {
					if strings.HasSuffix(info.FileName, expected) {
						return nil
					}
				}
			}
		}

		return fmt.Errorf("could not find attached disk matching %q", expected)
	}
}

// testAccResourceVSphereVirtualMachineCheckCPUMem checks the CPU and RAM for a
// VM.
func testAccResourceVSphereVirtualMachineCheckCPUMem(expectedCPU, expectedMem int32) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		props, err := testGetVirtualMachineProperties(s, "vm")
		if err != nil {
			return err
		}
		actualCPU := props.Summary.Config.NumCpu
		actualMem := props.Summary.Config.MemorySizeMB
		if expectedCPU != actualCPU {
			return fmt.Errorf("expected CPU count to be %d, got %d", expectedCPU, actualCPU)
		}
		if expectedMem != actualMem {
			return fmt.Errorf("expected memory size to be %d, got %d", expectedMem, actualMem)
		}
		return nil
	}
}

// testAccResourceVSphereVirtualMachineCheckNet checks to make sure a virtual
// machine's primary NIC has the given IP address and netmask assigned to it,
// and that the appropriate gateway is present.
//
// This uses VMware tools to check this, so it needs to be installed on the
// guest.
func testAccResourceVSphereVirtualMachineCheckNet(expectedAddr, expectedPrefix, expectedGW string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		res, err := strconv.Atoi(expectedPrefix)
		if err != nil {
			return fmt.Errorf("bad: %s", err)
		}
		expectedPrefixInt := int32(res)

		props, err := testGetVirtualMachineProperties(s, "vm")
		if err != nil {
			return err
		}
		var v4gw, v6gw net.IP
		for _, s := range props.Guest.IpStack {
			if s.IpRouteConfig != nil {
				for _, r := range s.IpRouteConfig.IpRoute {
					switch r.Network {
					case "0.0.0.0":
						v4gw = net.ParseIP(r.Gateway.IpAddress)
					case "::":
						v6gw = net.ParseIP(r.Gateway.IpAddress)
					}
				}
			}
		}
		for _, n := range props.Guest.Net {
			if n.IpConfig != nil {
				for _, addr := range n.IpConfig.IpAddress {
					ip := net.ParseIP(addr.IpAddress)
					if !ip.Equal(net.ParseIP(expectedAddr)) {
						continue
					}
					if addr.PrefixLength != expectedPrefixInt {
						continue
					}
					var mask net.IPMask
					if ip.To4() != nil {
						mask = net.CIDRMask(int(addr.PrefixLength), 32)
					} else {
						mask = net.CIDRMask(int(addr.PrefixLength), 128)
					}
					switch {
					case ip.Mask(mask).Equal(v4gw.Mask(mask)):
						if net.ParseIP(expectedGW).Equal(v4gw) {
							return nil
						}
					case ip.Mask(mask).Equal(v6gw.Mask(mask)):
						if net.ParseIP(expectedGW).Equal(v6gw) {
							return nil
						}
					}
				}
			}
		}

		return fmt.Errorf("could not find IP %s/%s, gateway %s", expectedAddr, expectedPrefix, expectedGW)
	}
}

// testAccResourceVSphereVirtualMachineCheckStaticMACAddr is a check to look
// for the MAC address defined in the static MAC address test on the first
// network interface.
func testAccResourceVSphereVirtualMachineCheckStaticMACAddr() resource.TestCheckFunc {
	return func(s *terraform.State) error {
		props, err := testGetVirtualMachineProperties(s, "vm")
		if err != nil {
			return err
		}
		l := object.VirtualDeviceList(props.Config.Hardware.Device)
		devices := l.Select(func(device types.BaseVirtualDevice) bool {
			if _, ok := device.(types.BaseVirtualEthernetCard); ok {
				return true
			}
			return false
		})
		if devices[0].(types.BaseVirtualEthernetCard).GetVirtualEthernetCard().AddressType != string(types.VirtualEthernetCardMacTypeManual) {
			return fmt.Errorf("first network interface is not set to manual address type")
		}
		actual := devices[0].(types.BaseVirtualEthernetCard).GetVirtualEthernetCard().MacAddress
		expected := testAccResourceVSphereVirtualMachineStaticMacAddr
		if expected != actual {
			return fmt.Errorf("expected MAC address to be %q, got %q", expected, actual)
		}
		return nil
	}
}

// testAccResourceVSphereVirtualMachineCheckAnnotation is a check to ensure
// that a VM's annotation is correctly set in the annotation test.
func testAccResourceVSphereVirtualMachineCheckAnnotation() resource.TestCheckFunc {
	return func(s *terraform.State) error {
		props, err := testGetVirtualMachineProperties(s, "vm")
		if err != nil {
			return err
		}
		expected := testAccResourceVSphereVirtualMachineAnnotation
		actual := props.Config.Annotation
		if expected != actual {
			return fmt.Errorf("expected annotation to be %q, got %q", expected, actual)
		}
		return nil
	}
}

// testAccResourceVSphereVirtualMachineCheckCustomizationSucceeded is a check
// to ensure that events have been received for customization success on a VM.
func testAccResourceVSphereVirtualMachineCheckCustomizationSucceeded() resource.TestCheckFunc {
	return func(s *terraform.State) error {
		vm, err := testGetVirtualMachine(s, "vm")
		if err != nil {
			return err
		}
		client := testAccProvider.Meta().(*VSphereClient).vimClient
		actual, err := selectEventsForReference(client, vm.Reference(), []string{eventTypeCustomizationSucceeded})
		if err != nil {
			return err
		}
		if len(actual) < 1 {
			return errors.New("customization success event was not received")
		}
		return nil
	}
}

// testAccResourceVSphereVirtualMachineCheckTags is a check to ensure that any
// tags that have been created with supplied resource name have been attached
// to the virtual machine.
func testAccResourceVSphereVirtualMachineCheckTags(tagResName string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		vm, err := testGetVirtualMachine(s, "vm")
		if err != nil {
			return err
		}
		tagsClient, err := testAccProvider.Meta().(*VSphereClient).TagsClient()
		if err != nil {
			return err
		}
		return testObjectHasTags(s, tagsClient, vm, tagResName)
	}
}

// testAccResourceVSphereVirtualMachineCheckMultiDevice is a check for proper
// parameters on the vsphere_virtual_machine multi-device test. This is a very
// specific check that checks for the specific disk and network devices. The
// configuration that this test asserts should be in the
// testAccResourceVSphereVirtualMachineConfigMultiDevice resource.
func testAccResourceVSphereVirtualMachineCheckMultiDevice(expectedD, expectedN []bool) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		props, err := testGetVirtualMachineProperties(s, "vm")
		if err != nil {
			return err
		}

		actualD := make([]bool, 3)
		actualN := make([]bool, 3)
		expectedDisk0Size := structure.GiBToByte(20)
		expectedDisk1Size := structure.GiBToByte(10)
		expectedDisk2Size := structure.GiBToByte(5)
		expectedNet0Level := types.SharesLevelNormal
		expectedNet1Level := types.SharesLevelHigh
		expectedNet2Level := types.SharesLevelLow

		for _, dev := range props.Config.Hardware.Device {
			if disk, ok := dev.(*types.VirtualDisk); ok {
				switch {
				case disk.CapacityInBytes == expectedDisk0Size:
					actualD[0] = true
				case disk.CapacityInBytes == expectedDisk1Size:
					actualD[1] = true
				case disk.CapacityInBytes == expectedDisk2Size:
					actualD[2] = true
				}
			}
			if bvec, ok := dev.(types.BaseVirtualEthernetCard); ok {
				card := bvec.GetVirtualEthernetCard()
				switch {
				case card.ResourceAllocation.Share.Level == expectedNet0Level:
					actualN[0] = true
				case card.ResourceAllocation.Share.Level == expectedNet1Level:
					actualN[1] = true
				case card.ResourceAllocation.Share.Level == expectedNet2Level:
					actualN[2] = true
				}
			}
		}

		for n, actual := range actualD {
			if actual != expectedD[n] {
				return fmt.Errorf("expected disk at index %d to be %t, got %t", n, expectedD[n], actual)
			}
		}
		for n, actual := range actualN {
			if actual != expectedN[n] {
				return fmt.Errorf("expected network interface at index %d to be %t, got %t", n, expectedN[n], actual)
			}
		}

		return nil
	}
}

// testAccResourceVSphereVirtualMachineCheckCdrom checks to make sure that the
// subject VM has a CDROM device configured and connected.
func testAccResourceVSphereVirtualMachineCheckCdrom() resource.TestCheckFunc {
	return func(s *terraform.State) error {
		props, err := testGetVirtualMachineProperties(s, "vm")
		if err != nil {
			return err
		}

		for _, dev := range props.Config.Hardware.Device {
			if cdrom, ok := dev.(*types.VirtualCdrom); ok {
				if !cdrom.Connectable.Connected {
					return fmt.Errorf("expected CDROM device to be connected")
				}
				if backing, ok := cdrom.Backing.(*types.VirtualCdromIsoBackingInfo); ok {
					expected := &object.DatastorePath{
						Datastore: os.Getenv("VSPHERE_ISO_DATASTORE"),
						Path:      os.Getenv("VSPHERE_ISO_FILE"),
					}
					actual := new(object.DatastorePath)
					actual.FromString(backing.FileName)
					if !reflect.DeepEqual(expected, actual) {
						return fmt.Errorf("expected %#v, got %#v", expected, actual)
					}
					return nil
				}
				return errors.New("could not locate proper backing file on CDROM device")
			}
		}
		return errors.New("could not locate CDROM device on VM")
	}
}

// testAccResourceVSphereVirtualMachineCheckPowerOffEvent is a check to see if
// the VM has been powered off at any point in time.
func testAccResourceVSphereVirtualMachineCheckPowerOffEvent(expected bool) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		vm, err := testGetVirtualMachine(s, "vm")
		if err != nil {
			return err
		}
		client := testAccProvider.Meta().(*VSphereClient).vimClient
		actual, err := selectEventsForReference(client, vm.Reference(), []string{eventTypeVmPoweredOffEvent})
		if err != nil {
			return err
		}
		switch {
		case len(actual) < 1 && expected:
			return errors.New("expected power off, VM was not powered off")
		case len(actual) > 1 && !expected:
			return errors.New("VM was powered off when it should not have been")
		}
		return nil
	}
}

// testAccResourceVSphereVirtualMachineCheckDiskSize checks the first
// VirtualDisk it encounters for a specific size in GiB. It should only be used
// with test configurations with a single disk attached.
func testAccResourceVSphereVirtualMachineCheckDiskSize(expected int) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		props, err := testGetVirtualMachineProperties(s, "vm")
		if err != nil {
			return err
		}

		expectedBytes := structure.GiBToByte(expected)

		for _, dev := range props.Config.Hardware.Device {
			if disk, ok := dev.(*types.VirtualDisk); ok {
				if expectedBytes != disk.CapacityInBytes {
					return fmt.Errorf("expected disk size to be %d, got %d", expectedBytes, disk.CapacityInBytes)
				}
			}
		}

		return nil
	}
}

// testAccResourceVSphereVirtualMachineCheckSCSIBus checks to make sure the
// test VM's SCSI bus is all of the specified SCSI type.
func testAccResourceVSphereVirtualMachineCheckSCSIBus(expected string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		actual, err := testGetVirtualMachineSCSIBusState(s, "vm")
		if err != nil {
			return fmt.Errorf("bad: %s", err)
		}
		if expected != actual {
			return fmt.Errorf("expected SCSI bus to be %s, got %s", expected, actual)
		}
		return nil
	}
}

// testAccResourceVSphereVirtualMachineCheckHost checks to make sure the
// test VM is currently located on a specific host.
func testAccResourceVSphereVirtualMachineCheckHost(expected string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		hs, err := testGetVirtualMachineHost(s, "vm")
		if err != nil {
			return err
		}
		actual := hs.Name()
		if expected != actual {
			return fmt.Errorf("expected host to be %s, got %s", expected, actual)
		}
		return nil
	}
}

// testAccResourceVSphereVirtualMachineCheckResourcePool checks to make sure the
// test VM is currently located in a specific resource pool.
func testAccResourceVSphereVirtualMachineCheckResourcePool(expected string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		pool, err := testGetVirtualMachineResourcePool(s, "vm")
		if err != nil {
			return err
		}

		actual := pool.Name()
		if actual == "Resources" && path.Base(expected) == "Resources" {
			client := testAccProvider.Meta().(*VSphereClient).vimClient
			expectedCluster, err := computeresource.BaseFromPath(client, path.Dir(expected))
			if err != nil {
				return err
			}
			pprops, err := resourcepool.Properties(pool)
			if err != nil {
				return err
			}
			actualCluster, err := computeresource.BaseFromReference(client, *pprops.Parent)
			if err != nil {
				return err
			}
			if expectedCluster.Reference().Value != actualCluster.Reference().Value {
				return fmt.Errorf("expected default resource pool of %q, got default resource pool of %q", expectedCluster.Reference().Value, actualCluster.Reference().Value)
			}
			return nil
		}
		if expected != actual {
			return fmt.Errorf("expected resource pool or to be %q, got %q", expected, actual)
		}
		return nil
	}
}

// testAccResourceVSphereVirtualMachineCheckExtraConfig checks a key/expected
// value combination in a VM's config.
func testAccResourceVSphereVirtualMachineCheckExtraConfig(key, expected string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		props, err := testGetVirtualMachineProperties(s, "vm")
		if err != nil {
			return err
		}
		for _, bov := range props.Config.ExtraConfig {
			ov := bov.GetOptionValue()
			if ov.Key == key {
				if ov.Value.(string) == expected {
					return nil
				}
				return fmt.Errorf("expected key %s to be %s, got %s", key, expected, ov.Value.(string))
			}
		}
		return fmt.Errorf("key %s not found", key)
	}
}

// testAccResourceVSphereVirtualMachineCheckExtraConfigKeyMissing checks to
// make sure that a key is missing in the VM's extraConfig.
func testAccResourceVSphereVirtualMachineCheckExtraConfigKeyMissing(key string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		props, err := testGetVirtualMachineProperties(s, "vm")
		if err != nil {
			return err
		}
		for _, bov := range props.Config.ExtraConfig {
			ov := bov.GetOptionValue()
			if ov.Key == key {
				return fmt.Errorf("expected key %s to be missing", key)
			}
		}
		return nil
	}
}

// testAccResourceVSphereVirtualMachineCheckVmxDatastore checks the datastore
// that the virtual machine's configuration is currently located.
func testAccResourceVSphereVirtualMachineCheckVmxDatastore(expected string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		props, err := testGetVirtualMachineProperties(s, "vm")
		if err != nil {
			return err
		}
		var dsPath object.DatastorePath
		if ok := dsPath.FromString(props.Config.Files.VmPathName); !ok {
			return fmt.Errorf("could not parse datastore path %q", props.Config.Files.VmPathName)
		}
		actual := dsPath.Datastore
		if expected != actual {
			return fmt.Errorf("expected VM configuration to be in datastore %s, got %s", expected, actual)
		}
		return nil
	}
}

// testAccResourceVSphereVirtualMachineCheckVmdkDatastore checks the datastore
// that a specific VMDK file is in.
func testAccResourceVSphereVirtualMachineCheckVmdkDatastore(name, expected string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		props, err := testGetVirtualMachineProperties(s, "vm")
		if err != nil {
			return err
		}
		for _, dev := range props.Config.Hardware.Device {
			if disk, ok := dev.(*types.VirtualDisk); ok {
				if info, ok := disk.Backing.(*types.VirtualDiskFlatVer2BackingInfo); ok {
					var dsPath object.DatastorePath
					if ok := dsPath.FromString(info.FileName); !ok {
						return fmt.Errorf("could not parse datastore path %q", info.FileName)
					}
					if path.Base(dsPath.Path) == name {
						actual := dsPath.Datastore
						if expected == actual {
							return nil
						}
						return fmt.Errorf("expected disk name %q to be on datastore %q, got %q", name, expected, actual)
					}
				}
			}
		}
		return fmt.Errorf("could not find disk %q", name)
	}
}

// testAccResourceVSphereVirtualMachineCheckNICCount checks the number of NICs
// on the virtual machine.
func testAccResourceVSphereVirtualMachineCheckNICCount(expected int) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		props, err := testGetVirtualMachineProperties(s, "vm")
		if err != nil {
			return err
		}

		var actual int
		for _, dev := range props.Config.Hardware.Device {
			if _, ok := dev.(types.BaseVirtualEthernetCard); ok {
				actual++
			}
		}
		if expected != actual {
			return fmt.Errorf("expected %d number of NICs, got %d", expected, actual)
		}
		return nil
	}
}

// testCheckVMDiskFileExists takes a VMDK filename and checks to see if it
// exists within the same directory as the virtual machine's VMX file.
func testCheckVMDiskFileExists(name string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		tVars, err := testClientVariablesForResource(s, "vsphere_virtual_machine.vm")
		if err != nil {
			return err
		}
		vm, err := testGetVirtualMachine(s, "vm")
		if err != nil {
			return err
		}
		props, err := testGetVirtualMachineProperties(s, "vm")
		if err != nil {
			return err
		}
		vmxPath, success := virtualdisk.DatastorePathFromString(props.Config.Files.VmPathName)
		if !success {
			return fmt.Errorf("could not parse VMX path %q", props.Config.Files.VmPathName)
		}
		dcp, err := folder.RootPathParticleVM.SplitDatacenter(vm.InventoryPath)
		if err != nil {
			return err
		}
		dc, err := getDatacenter(tVars.client, dcp)
		if err != nil {
			return err
		}
		ds, err := datastore.FromPath(tVars.client, vmxPath.Datastore, dc)
		if err != nil {
			return err
		}
		p := path.Join(path.Dir(vmxPath.Path), name)
		exists, err := datastore.FileExists(ds, p)
		if err != nil {
			return err
		}
		if !exists {
			return fmt.Errorf("file %q does not exist in datastore %q", p, ds.Name())
		}
		return nil
	}
}

func testAccResourceVSphereVirtualMachineCheckCustomAttributes() resource.TestCheckFunc {
	return func(s *terraform.State) error {
		props, err := testGetVirtualMachineProperties(s, "vm")
		if err != nil {
			return err
		}
		return testResourceHasCustomAttributeValues(s, "vsphere_virtual_machine", "vm", props.Entity())
	}
}

func testAccResourceVSphereVirtualMachineConfigBasic() string {
	return fmt.Sprintf(`
variable "datacenter" {
  default = "%s"
}

variable "resource_pool" {
  default = "%s"
}

variable "network_label" {
  default = "%s"
}

variable "datastore" {
  default = "%s"
}

data "vsphere_datacenter" "dc" {
  name = "${var.datacenter}"
}

data "vsphere_datastore" "datastore" {
  name          = "${var.datastore}"
  datacenter_id = "${data.vsphere_datacenter.dc.id}"
}

data "vsphere_resource_pool" "pool" {
  name          = "${var.resource_pool}"
  datacenter_id = "${data.vsphere_datacenter.dc.id}"
}

data "vsphere_network" "network" {
  name          = "${var.network_label}"
  datacenter_id = "${data.vsphere_datacenter.dc.id}"
}

resource "vsphere_virtual_machine" "vm" {
  name             = "terraform-test"
  resource_pool_id = "${data.vsphere_resource_pool.pool.id}"
  datastore_id     = "${data.vsphere_datastore.datastore.id}"

  num_cpus = 2
  memory   = 2048
  guest_id = "other3xLinux64Guest"

  network_interface {
    network_id = "${data.vsphere_network.network.id}"
  }

  disk {
    label = "disk0"
    size  = 20
  }
}
`,
		os.Getenv("VSPHERE_DATACENTER"),
		os.Getenv("VSPHERE_RESOURCE_POOL"),
		os.Getenv("VSPHERE_NETWORK_LABEL_PXE"),
		os.Getenv("VSPHERE_DATASTORE"),
	)
}

func testAccResourceVSphereVirtualMachineConfigBasicESXiOnly() string {
	return fmt.Sprintf(`
variable "network_label" {
  default = "%s"
}

variable "datastore" {
  default = "%s"
}

data "vsphere_datacenter" "dc" {}

data "vsphere_datastore" "datastore" {
  name          = "${var.datastore}"
  datacenter_id = "${data.vsphere_datacenter.dc.id}"
}

data "vsphere_resource_pool" "pool" {}

data "vsphere_network" "network" {
  name          = "${var.network_label}"
  datacenter_id = "${data.vsphere_datacenter.dc.id}"
}

resource "vsphere_virtual_machine" "vm" {
  name             = "terraform-test"
  resource_pool_id = "${data.vsphere_resource_pool.pool.id}"
  datastore_id     = "${data.vsphere_datastore.datastore.id}"

  num_cpus = 2
  memory   = 2048
  guest_id = "other3xLinux64Guest"

  network_interface {
    network_id = "${data.vsphere_network.network.id}"
  }

  disk {
    label = "disk0"
    size  = 20
  }
}
`,
		os.Getenv("VSPHERE_NETWORK_LABEL_PXE"),
		os.Getenv("VSPHERE_DATASTORE"),
	)
}

func testAccResourceVSphereVirtualMachineConfigMultiDevice() string {
	return fmt.Sprintf(`
variable "datacenter" {
  default = "%s"
}

variable "resource_pool" {
  default = "%s"
}

variable "network_label" {
  default = "%s"
}

variable "datastore" {
  default = "%s"
}

data "vsphere_datacenter" "dc" {
  name = "${var.datacenter}"
}

data "vsphere_datastore" "datastore" {
  name          = "${var.datastore}"
  datacenter_id = "${data.vsphere_datacenter.dc.id}"
}

data "vsphere_resource_pool" "pool" {
  name          = "${var.resource_pool}"
  datacenter_id = "${data.vsphere_datacenter.dc.id}"
}

data "vsphere_network" "network" {
  name          = "${var.network_label}"
  datacenter_id = "${data.vsphere_datacenter.dc.id}"
}

resource "vsphere_virtual_machine" "vm" {
  name             = "terraform-test"
  resource_pool_id = "${data.vsphere_resource_pool.pool.id}"
  datastore_id     = "${data.vsphere_datastore.datastore.id}"

  num_cpus = 2
  memory   = 2048
  guest_id = "other3xLinux64Guest"

  network_interface {
    network_id            = "${data.vsphere_network.network.id}"
    bandwidth_share_level = "normal"
  }

  network_interface {
    network_id            = "${data.vsphere_network.network.id}"
    bandwidth_share_level = "high"
  }

  network_interface {
    network_id            = "${data.vsphere_network.network.id}"
    bandwidth_share_level = "low"
  }

  disk {
    label = "disk0"
    size  = 20
  }

  disk {
    label       = "disk1"
    unit_number = 1
    size        = 10
  }

  disk {
    label       = "disk2"
    unit_number = 2
    size        = 5
  }
}
`,
		os.Getenv("VSPHERE_DATACENTER"),
		os.Getenv("VSPHERE_RESOURCE_POOL"),
		os.Getenv("VSPHERE_NETWORK_LABEL_PXE"),
		os.Getenv("VSPHERE_DATASTORE"),
	)
}

func testAccResourceVSphereVirtualMachineConfigMultiHighBus() string {
	return fmt.Sprintf(`
variable "datacenter" {
  default = "%s"
}

variable "resource_pool" {
  default = "%s"
}

variable "network_label" {
  default = "%s"
}

variable "datastore" {
  default = "%s"
}

data "vsphere_datacenter" "dc" {
  name = "${var.datacenter}"
}

data "vsphere_datastore" "datastore" {
  name          = "${var.datastore}"
  datacenter_id = "${data.vsphere_datacenter.dc.id}"
}

data "vsphere_resource_pool" "pool" {
  name          = "${var.resource_pool}"
  datacenter_id = "${data.vsphere_datacenter.dc.id}"
}

data "vsphere_network" "network" {
  name          = "${var.network_label}"
  datacenter_id = "${data.vsphere_datacenter.dc.id}"
}

resource "vsphere_virtual_machine" "vm" {
  name             = "terraform-test"
  resource_pool_id = "${data.vsphere_resource_pool.pool.id}"
  datastore_id     = "${data.vsphere_datastore.datastore.id}"

  scsi_controller_count = 3

  num_cpus = 2
  memory   = 2048
  guest_id = "other3xLinux64Guest"

  network_interface {
    network_id            = "${data.vsphere_network.network.id}"
    bandwidth_share_level = "normal"
  }

  network_interface {
    network_id            = "${data.vsphere_network.network.id}"
    bandwidth_share_level = "high"
  }

  network_interface {
    network_id            = "${data.vsphere_network.network.id}"
    bandwidth_share_level = "low"
  }

  disk {
    label = "disk0"
    size  = 20
  }

  disk {
    label       = "disk1"
    unit_number = 15
    size        = 10
  }

  disk {
    label       = "disk2"
    unit_number = 31
    size        = 5
  }
}
`,
		os.Getenv("VSPHERE_DATACENTER"),
		os.Getenv("VSPHERE_RESOURCE_POOL"),
		os.Getenv("VSPHERE_NETWORK_LABEL_PXE"),
		os.Getenv("VSPHERE_DATASTORE"),
	)
}

func testAccResourceVSphereVirtualMachineConfigRemoveMiddle() string {
	return fmt.Sprintf(`
variable "datacenter" {
  default = "%s"
}

variable "resource_pool" {
  default = "%s"
}

variable "network_label" {
  default = "%s"
}

variable "datastore" {
  default = "%s"
}

data "vsphere_datacenter" "dc" {
  name = "${var.datacenter}"
}

data "vsphere_datastore" "datastore" {
  name          = "${var.datastore}"
  datacenter_id = "${data.vsphere_datacenter.dc.id}"
}

data "vsphere_resource_pool" "pool" {
  name          = "${var.resource_pool}"
  datacenter_id = "${data.vsphere_datacenter.dc.id}"
}

data "vsphere_network" "network" {
  name          = "${var.network_label}"
  datacenter_id = "${data.vsphere_datacenter.dc.id}"
}

resource "vsphere_virtual_machine" "vm" {
  name             = "terraform-test"
  resource_pool_id = "${data.vsphere_resource_pool.pool.id}"
  datastore_id     = "${data.vsphere_datastore.datastore.id}"

  num_cpus = 2
  memory   = 2048
  guest_id = "other3xLinux64Guest"

  network_interface {
    network_id            = "${data.vsphere_network.network.id}"
    bandwidth_share_level = "normal"
  }

  network_interface {
    network_id            = "${data.vsphere_network.network.id}"
    bandwidth_share_level = "low"
  }

  disk {
    label = "disk0"
    size  = 20
  }

  disk {
    label       = "disk2"
    unit_number = 2
    size        = 5
  }
}
`,
		os.Getenv("VSPHERE_DATACENTER"),
		os.Getenv("VSPHERE_RESOURCE_POOL"),
		os.Getenv("VSPHERE_NETWORK_LABEL_PXE"),
		os.Getenv("VSPHERE_DATASTORE"),
	)
}

func testAccResourceVSphereVirtualMachineConfigRemoveMiddleChangeUnit() string {
	return fmt.Sprintf(`
variable "datacenter" {
  default = "%s"
}

variable "resource_pool" {
  default = "%s"
}

variable "network_label" {
  default = "%s"
}

variable "datastore" {
  default = "%s"
}

data "vsphere_datacenter" "dc" {
  name = "${var.datacenter}"
}

data "vsphere_datastore" "datastore" {
  name          = "${var.datastore}"
  datacenter_id = "${data.vsphere_datacenter.dc.id}"
}

data "vsphere_resource_pool" "pool" {
  name          = "${var.resource_pool}"
  datacenter_id = "${data.vsphere_datacenter.dc.id}"
}

data "vsphere_network" "network" {
  name          = "${var.network_label}"
  datacenter_id = "${data.vsphere_datacenter.dc.id}"
}

resource "vsphere_virtual_machine" "vm" {
  name             = "terraform-test"
  resource_pool_id = "${data.vsphere_resource_pool.pool.id}"
  datastore_id     = "${data.vsphere_datastore.datastore.id}"

  num_cpus = 2
  memory   = 2048
  guest_id = "other3xLinux64Guest"

  network_interface {
    network_id            = "${data.vsphere_network.network.id}"
    bandwidth_share_level = "normal"
  }

  network_interface {
    network_id            = "${data.vsphere_network.network.id}"
    bandwidth_share_level = "low"
  }

  disk {
    label = "disk0"
    size  = 20
  }

  disk {
    label       = "disk2"
    unit_number = 1
    size        = 5
  }
}
`,
		os.Getenv("VSPHERE_DATACENTER"),
		os.Getenv("VSPHERE_RESOURCE_POOL"),
		os.Getenv("VSPHERE_NETWORK_LABEL_PXE"),
		os.Getenv("VSPHERE_DATASTORE"),
	)
}

func testAccResourceVSphereVirtualMachineConfigCdrom() string {
	return fmt.Sprintf(`
variable "datacenter" {
  default = "%s"
}

variable "resource_pool" {
  default = "%s"
}

variable "network_label" {
  default = "%s"
}

variable "datastore" {
  default = "%s"
}

variable "iso_datastore" {
  default = "%s"
}

variable "iso_path" {
  default = "%s"
}

data "vsphere_datacenter" "dc" {
  name = "${var.datacenter}"
}

data "vsphere_datastore" "datastore" {
  name          = "${var.datastore}"
  datacenter_id = "${data.vsphere_datacenter.dc.id}"
}

data "vsphere_datastore" "iso_datastore" {
  name          = "${var.iso_datastore}"
  datacenter_id = "${data.vsphere_datacenter.dc.id}"
}

data "vsphere_resource_pool" "pool" {
  name          = "${var.resource_pool}"
  datacenter_id = "${data.vsphere_datacenter.dc.id}"
}

data "vsphere_network" "network" {
  name          = "${var.network_label}"
  datacenter_id = "${data.vsphere_datacenter.dc.id}"
}

resource "vsphere_virtual_machine" "vm" {
  name             = "terraform-test"
  resource_pool_id = "${data.vsphere_resource_pool.pool.id}"
  datastore_id     = "${data.vsphere_datastore.datastore.id}"

  num_cpus = 2
  memory   = 2048
  guest_id = "other3xLinux64Guest"

  wait_for_guest_net_timeout = -1

  network_interface {
    network_id = "${data.vsphere_network.network.id}"
  }

  disk {
    label = "disk0"
    size  = 20
  }

  cdrom {
    datastore_id = "${data.vsphere_datastore.iso_datastore.id}"
    path         = "${var.iso_path}"
  }
}
`,
		os.Getenv("VSPHERE_DATACENTER"),
		os.Getenv("VSPHERE_RESOURCE_POOL"),
		os.Getenv("VSPHERE_NETWORK_LABEL_PXE"),
		os.Getenv("VSPHERE_DATASTORE"),
		os.Getenv("VSPHERE_ISO_DATASTORE"),
		os.Getenv("VSPHERE_ISO_FILE"),
	)
}

func testAccResourceVSphereVirtualMachineConfigBeefy() string {
	return fmt.Sprintf(`
variable "datacenter" {
  default = "%s"
}

variable "resource_pool" {
  default = "%s"
}

variable "network_label" {
  default = "%s"
}

variable "datastore" {
  default = "%s"
}

data "vsphere_datacenter" "dc" {
  name = "${var.datacenter}"
}

data "vsphere_datastore" "datastore" {
  name          = "${var.datastore}"
  datacenter_id = "${data.vsphere_datacenter.dc.id}"
}

data "vsphere_resource_pool" "pool" {
  name          = "${var.resource_pool}"
  datacenter_id = "${data.vsphere_datacenter.dc.id}"
}

data "vsphere_network" "network" {
  name          = "${var.network_label}"
  datacenter_id = "${data.vsphere_datacenter.dc.id}"
}

resource "vsphere_virtual_machine" "vm" {
  name             = "terraform-test"
  resource_pool_id = "${data.vsphere_resource_pool.pool.id}"
  datastore_id     = "${data.vsphere_datastore.datastore.id}"

  num_cpus = 4
  memory   = 8192
  guest_id = "other3xLinux64Guest"

  network_interface {
    network_id = "${data.vsphere_network.network.id}"
  }

  disk {
    label = "disk0"
    size  = 20
  }
}
`,
		os.Getenv("VSPHERE_DATACENTER"),
		os.Getenv("VSPHERE_RESOURCE_POOL"),
		os.Getenv("VSPHERE_NETWORK_LABEL_PXE"),
		os.Getenv("VSPHERE_DATASTORE"),
	)
}

func testAccResourceVSphereVirtualMachineConfigMaxNIC() string {
	return fmt.Sprintf(`
variable "datacenter" {
  default = "%s"
}

variable "resource_pool" {
  default = "%s"
}

variable "network_label" {
  default = "%s"
}

variable "datastore" {
  default = "%s"
}

data "vsphere_datacenter" "dc" {
  name = "${var.datacenter}"
}

data "vsphere_datastore" "datastore" {
  name          = "${var.datastore}"
  datacenter_id = "${data.vsphere_datacenter.dc.id}"
}

data "vsphere_resource_pool" "pool" {
  name          = "${var.resource_pool}"
  datacenter_id = "${data.vsphere_datacenter.dc.id}"
}

data "vsphere_network" "network" {
  name          = "${var.network_label}"
  datacenter_id = "${data.vsphere_datacenter.dc.id}"
}

resource "vsphere_virtual_machine" "vm" {
  name             = "terraform-test"
  resource_pool_id = "${data.vsphere_resource_pool.pool.id}"
  datastore_id     = "${data.vsphere_datastore.datastore.id}"

  num_cpus = 2
  memory   = 2048
  guest_id = "other3xLinux64Guest"

  network_interface {
    network_id = "${data.vsphere_network.network.id}"
  }

  network_interface {
    network_id = "${data.vsphere_network.network.id}"
  }

  network_interface {
    network_id = "${data.vsphere_network.network.id}"
  }

  network_interface {
    network_id = "${data.vsphere_network.network.id}"
  }

  network_interface {
    network_id = "${data.vsphere_network.network.id}"
  }

  network_interface {
    network_id = "${data.vsphere_network.network.id}"
  }

  network_interface {
    network_id = "${data.vsphere_network.network.id}"
  }

  network_interface {
    network_id = "${data.vsphere_network.network.id}"
  }

  network_interface {
    network_id = "${data.vsphere_network.network.id}"
  }

  network_interface {
    network_id = "${data.vsphere_network.network.id}"
  }

  disk {
    label = "disk0"
    size  = 20
  }
}
`,
		os.Getenv("VSPHERE_DATACENTER"),
		os.Getenv("VSPHERE_RESOURCE_POOL"),
		os.Getenv("VSPHERE_NETWORK_LABEL_PXE"),
		os.Getenv("VSPHERE_DATASTORE"),
	)
}

func testAccResourceVSphereVirtualMachineConfigBasicAnnotation() string {
	return fmt.Sprintf(`
variable "datacenter" {
  default = "%s"
}

variable "resource_pool" {
  default = "%s"
}

variable "network_label" {
  default = "%s"
}

variable "datastore" {
  default = "%s"
}

variable "annotation" {
  default = "%s"
}

data "vsphere_datacenter" "dc" {
  name = "${var.datacenter}"
}

data "vsphere_datastore" "datastore" {
  name          = "${var.datastore}"
  datacenter_id = "${data.vsphere_datacenter.dc.id}"
}

data "vsphere_resource_pool" "pool" {
  name          = "${var.resource_pool}"
  datacenter_id = "${data.vsphere_datacenter.dc.id}"
}

data "vsphere_network" "network" {
  name          = "${var.network_label}"
  datacenter_id = "${data.vsphere_datacenter.dc.id}"
}

resource "vsphere_virtual_machine" "vm" {
  name             = "terraform-test"
  resource_pool_id = "${data.vsphere_resource_pool.pool.id}"
  datastore_id     = "${data.vsphere_datastore.datastore.id}"

  num_cpus   = 2
  memory     = 2048
  guest_id   = "other3xLinux64Guest"
  annotation = "${var.annotation}"

  network_interface {
    network_id = "${data.vsphere_network.network.id}"
  }

  disk {
    label = "disk0"
    size  = 20
  }
}
`,
		os.Getenv("VSPHERE_DATACENTER"),
		os.Getenv("VSPHERE_RESOURCE_POOL"),
		os.Getenv("VSPHERE_NETWORK_LABEL_PXE"),
		os.Getenv("VSPHERE_DATASTORE"),
		testAccResourceVSphereVirtualMachineAnnotation,
	)
}

func testAccResourceVSphereVirtualMachineConfigGrowDisk(size int) string {
	return fmt.Sprintf(`
variable "datacenter" {
  default = "%s"
}

variable "resource_pool" {
  default = "%s"
}

variable "network_label" {
  default = "%s"
}

variable "datastore" {
  default = "%s"
}

data "vsphere_datacenter" "dc" {
  name = "${var.datacenter}"
}

data "vsphere_datastore" "datastore" {
  name = "${var.datastore}"
  datacenter_id = "${data.vsphere_datacenter.dc.id}"
}

data "vsphere_resource_pool" "pool" {
  name          = "${var.resource_pool}"
  datacenter_id = "${data.vsphere_datacenter.dc.id}"
}

data "vsphere_network" "network" {
  name          = "${var.network_label}"
  datacenter_id = "${data.vsphere_datacenter.dc.id}"
}

resource "vsphere_virtual_machine" "vm" {
  name             = "terraform-test"
  resource_pool_id = "${data.vsphere_resource_pool.pool.id}"
  datastore_id     = "${data.vsphere_datastore.datastore.id}"

  num_cpus = 2
  memory   = 2048
  guest_id = "other3xLinux64Guest"

  network_interface {
    network_id = "${data.vsphere_network.network.id}"
  }

  disk {
    label = "disk0"
    size  = %d
  }
}
`,
		os.Getenv("VSPHERE_DATACENTER"),
		os.Getenv("VSPHERE_RESOURCE_POOL"),
		os.Getenv("VSPHERE_NETWORK_LABEL_PXE"),
		os.Getenv("VSPHERE_DATASTORE"),
		size,
	)
}

func testAccResourceVSphereVirtualMachineConfigLsiLogicSAS() string {
	return fmt.Sprintf(`
variable "datacenter" {
  default = "%s"
}

variable "resource_pool" {
  default = "%s"
}

variable "network_label" {
  default = "%s"
}

variable "datastore" {
  default = "%s"
}

data "vsphere_datacenter" "dc" {
  name = "${var.datacenter}"
}

data "vsphere_datastore" "datastore" {
  name          = "${var.datastore}"
  datacenter_id = "${data.vsphere_datacenter.dc.id}"
}

data "vsphere_resource_pool" "pool" {
  name          = "${var.resource_pool}"
  datacenter_id = "${data.vsphere_datacenter.dc.id}"
}

data "vsphere_network" "network" {
  name          = "${var.network_label}"
  datacenter_id = "${data.vsphere_datacenter.dc.id}"
}

resource "vsphere_virtual_machine" "vm" {
  name             = "terraform-test"
  resource_pool_id = "${data.vsphere_resource_pool.pool.id}"
  datastore_id     = "${data.vsphere_datastore.datastore.id}"

  num_cpus = 2
  memory   = 2048
  guest_id = "other3xLinux64Guest"

  scsi_type = "lsilogic-sas"

  network_interface {
    network_id = "${data.vsphere_network.network.id}"
  }

  disk {
    label = "disk0"
    size  = 20
  }
}
`,
		os.Getenv("VSPHERE_DATACENTER"),
		os.Getenv("VSPHERE_RESOURCE_POOL"),
		os.Getenv("VSPHERE_NETWORK_LABEL_PXE"),
		os.Getenv("VSPHERE_DATASTORE"),
	)
}

func testAccResourceVSphereVirtualMachineConfigExtraConfig(k, v string) string {
	return fmt.Sprintf(`
variable "datacenter" {
  default = "%s"
}

variable "resource_pool" {
  default = "%s"
}

variable "network_label" {
  default = "%s"
}

variable "datastore" {
  default = "%s"
}

data "vsphere_datacenter" "dc" {
  name = "${var.datacenter}"
}

data "vsphere_datastore" "datastore" {
  name          = "${var.datastore}"
  datacenter_id = "${data.vsphere_datacenter.dc.id}"
}

data "vsphere_resource_pool" "pool" {
  name          = "${var.resource_pool}"
  datacenter_id = "${data.vsphere_datacenter.dc.id}"
}

data "vsphere_network" "network" {
  name          = "${var.network_label}"
  datacenter_id = "${data.vsphere_datacenter.dc.id}"
}

resource "vsphere_virtual_machine" "vm" {
  name             = "terraform-test"
  resource_pool_id = "${data.vsphere_resource_pool.pool.id}"
  datastore_id     = "${data.vsphere_datastore.datastore.id}"

  num_cpus = 2
  memory   = 2048
  guest_id = "other3xLinux64Guest"

  extra_config {
    "%s" = "%s"
  }

  network_interface {
    network_id = "${data.vsphere_network.network.id}"
  }

  disk {
    label = "disk0"
    size  = 20
  }
}
`,
		os.Getenv("VSPHERE_DATACENTER"),
		os.Getenv("VSPHERE_RESOURCE_POOL"),
		os.Getenv("VSPHERE_NETWORK_LABEL_PXE"),
		os.Getenv("VSPHERE_DATASTORE"),
		k, v,
	)
}

func testAccResourceVSphereVirtualMachineConfigExistingVmdk() string {
	return fmt.Sprintf(`
variable "datacenter" {
  default = "%s"
}

variable "resource_pool" {
  default = "%s"
}

variable "network_label" {
  default = "%s"
}

variable "datastore" {
  default = "%s"
}

variable "extra_vmdk_name" {
  default = "%s"
}

data "vsphere_datacenter" "dc" {
  name = "${var.datacenter}"
}

data "vsphere_datastore" "datastore" {
  name          = "${var.datastore}"
  datacenter_id = "${data.vsphere_datacenter.dc.id}"
}

data "vsphere_resource_pool" "pool" {
  name          = "${var.resource_pool}"
  datacenter_id = "${data.vsphere_datacenter.dc.id}"
}

data "vsphere_network" "network" {
  name          = "${var.network_label}"
  datacenter_id = "${data.vsphere_datacenter.dc.id}"
}

resource "vsphere_virtual_disk" "disk" {
  size         = 1
  vmdk_path    = "${var.extra_vmdk_name}"
  datacenter   = "${var.datacenter}"
  datastore    = "${var.datastore}"
  type         = "thin"
  adapter_type = "lsiLogic"
}

resource "vsphere_virtual_machine" "vm" {
  name             = "terraform-test"
  resource_pool_id = "${data.vsphere_resource_pool.pool.id}"
  datastore_id     = "${data.vsphere_datastore.datastore.id}"

  num_cpus = 2
  memory   = 2048
  guest_id = "other3xLinux64Guest"

  network_interface {
    network_id = "${data.vsphere_network.network.id}"
  }

  disk {
    label = "disk0"
    size  = 20
  }

  disk {
    label        = "disk1"
    datastore_id = "${data.vsphere_datastore.datastore.id}"
    path         = "${vsphere_virtual_disk.disk.vmdk_path}"
    disk_mode    = "independent_persistent"
    attach       = true
    unit_number  = 1
  }
}
`,
		os.Getenv("VSPHERE_DATACENTER"),
		os.Getenv("VSPHERE_RESOURCE_POOL"),
		os.Getenv("VSPHERE_NETWORK_LABEL_PXE"),
		os.Getenv("VSPHERE_DATASTORE"),
		testAccResourceVSphereVirtualMachineDiskNameExtraVmdk,
	)
}

func testAccResourceVSphereVirtualMachineConfigInFolder() string {
	return fmt.Sprintf(`
variable "datacenter" {
  default = "%s"
}

variable "resource_pool" {
  default = "%s"
}

variable "network_label" {
  default = "%s"
}

variable "datastore" {
  default = "%s"
}

data "vsphere_datacenter" "dc" {
  name = "${var.datacenter}"
}

data "vsphere_datastore" "datastore" {
  name          = "${var.datastore}"
  datacenter_id = "${data.vsphere_datacenter.dc.id}"
}

data "vsphere_resource_pool" "pool" {
  name          = "${var.resource_pool}"
  datacenter_id = "${data.vsphere_datacenter.dc.id}"
}

data "vsphere_network" "network" {
  name          = "${var.network_label}"
  datacenter_id = "${data.vsphere_datacenter.dc.id}"
}

resource "vsphere_folder" "folder" {
  path          = "terraform-test-vms"
  type          = "vm"
  datacenter_id = "${data.vsphere_datacenter.dc.id}"
}

resource "vsphere_virtual_machine" "vm" {
  name             = "terraform-test"
  resource_pool_id = "${data.vsphere_resource_pool.pool.id}"
  datastore_id     = "${data.vsphere_datastore.datastore.id}"
  folder           = "${vsphere_folder.folder.path}"

  num_cpus = 2
  memory   = 2048
  guest_id = "other3xLinux64Guest"

  network_interface {
    network_id = "${data.vsphere_network.network.id}"
  }

  disk {
    label = "disk0"
    size  = 20
  }
}
`,
		os.Getenv("VSPHERE_DATACENTER"),
		os.Getenv("VSPHERE_RESOURCE_POOL"),
		os.Getenv("VSPHERE_NETWORK_LABEL_PXE"),
		os.Getenv("VSPHERE_DATASTORE"),
	)
}

func testAccResourceVSphereVirtualMachineConfigStaticMAC() string {
	return fmt.Sprintf(`
variable "datacenter" {
  default = "%s"
}

variable "resource_pool" {
  default = "%s"
}

variable "network_label" {
  default = "%s"
}

variable "datastore" {
  default = "%s"
}

data "vsphere_datacenter" "dc" {
  name = "${var.datacenter}"
}

data "vsphere_datastore" "datastore" {
  name          = "${var.datastore}"
  datacenter_id = "${data.vsphere_datacenter.dc.id}"
}

data "vsphere_resource_pool" "pool" {
  name          = "${var.resource_pool}"
  datacenter_id = "${data.vsphere_datacenter.dc.id}"
}

data "vsphere_network" "network" {
  name          = "${var.network_label}"
  datacenter_id = "${data.vsphere_datacenter.dc.id}"
}

resource "vsphere_virtual_machine" "vm" {
  name             = "terraform-test"
  resource_pool_id = "${data.vsphere_resource_pool.pool.id}"
  datastore_id     = "${data.vsphere_datastore.datastore.id}"

  num_cpus = 2
  memory   = 2048
  guest_id = "other3xLinux64Guest"

  network_interface {
    network_id     = "${data.vsphere_network.network.id}"
    use_static_mac = true
    mac_address    = "%s"
  }

  disk {
    label = "disk0"
    size  = 20
  }
}
`,
		os.Getenv("VSPHERE_DATACENTER"),
		os.Getenv("VSPHERE_RESOURCE_POOL"),
		os.Getenv("VSPHERE_NETWORK_LABEL_PXE"),
		os.Getenv("VSPHERE_DATASTORE"),
		testAccResourceVSphereVirtualMachineStaticMacAddr,
	)
}

func testAccResourceVSphereVirtualMachineConfigSingleTag() string {
	return fmt.Sprintf(`
variable "datacenter" {
  default = "%s"
}

variable "resource_pool" {
  default = "%s"
}

variable "network_label" {
  default = "%s"
}

variable "datastore" {
  default = "%s"
}

data "vsphere_datacenter" "dc" {
  name = "${var.datacenter}"
}

data "vsphere_datastore" "datastore" {
  name          = "${var.datastore}"
  datacenter_id = "${data.vsphere_datacenter.dc.id}"
}

data "vsphere_resource_pool" "pool" {
  name          = "${var.resource_pool}"
  datacenter_id = "${data.vsphere_datacenter.dc.id}"
}

data "vsphere_network" "network" {
  name          = "${var.network_label}"
  datacenter_id = "${data.vsphere_datacenter.dc.id}"
}

resource "vsphere_tag_category" "terraform-test-category" {
  name        = "terraform-test-tag-category"
  cardinality = "MULTIPLE"

  associable_types = [
    "VirtualMachine",
  ]
}

resource "vsphere_tag" "terraform-test-tag" {
  name        = "terraform-test-tag"
  category_id = "${vsphere_tag_category.terraform-test-category.id}"
}

resource "vsphere_virtual_machine" "vm" {
  name             = "terraform-test"
  resource_pool_id = "${data.vsphere_resource_pool.pool.id}"
  datastore_id     = "${data.vsphere_datastore.datastore.id}"

  num_cpus = 2
  memory   = 2048
  guest_id = "other3xLinux64Guest"

  network_interface {
    network_id = "${data.vsphere_network.network.id}"
  }

  disk {
    label = "disk0"
    size  = 20
  }

  tags = [
    "${vsphere_tag.terraform-test-tag.id}",
  ]
}
`,
		os.Getenv("VSPHERE_DATACENTER"),
		os.Getenv("VSPHERE_RESOURCE_POOL"),
		os.Getenv("VSPHERE_NETWORK_LABEL_PXE"),
		os.Getenv("VSPHERE_DATASTORE"),
	)
}

func testAccResourceVSphereVirtualMachineConfigMultiTag() string {
	return fmt.Sprintf(`
variable "datacenter" {
  default = "%s"
}

variable "resource_pool" {
  default = "%s"
}

variable "network_label" {
  default = "%s"
}

variable "datastore" {
  default = "%s"
}

variable "extra_tags" {
  default = [
    "terraform-test-thing1",
    "terraform-test-thing2",
  ]
}

data "vsphere_datacenter" "dc" {
  name = "${var.datacenter}"
}

data "vsphere_datastore" "datastore" {
  name          = "${var.datastore}"
  datacenter_id = "${data.vsphere_datacenter.dc.id}"
}

data "vsphere_resource_pool" "pool" {
  name          = "${var.resource_pool}"
  datacenter_id = "${data.vsphere_datacenter.dc.id}"
}

data "vsphere_network" "network" {
  name          = "${var.network_label}"
  datacenter_id = "${data.vsphere_datacenter.dc.id}"
}

resource "vsphere_tag_category" "terraform-test-category" {
  name        = "terraform-test-tag-category"
  cardinality = "MULTIPLE"

  associable_types = [
    "VirtualMachine",
  ]
}

resource "vsphere_tag" "terraform-test-tag" {
  name        = "terraform-test-tag"
  category_id = "${vsphere_tag_category.terraform-test-category.id}"
}

resource "vsphere_tag" "terraform-test-tags-alt" {
  count       = "${length(var.extra_tags)}"
  name        = "${var.extra_tags[count.index]}"
  category_id = "${vsphere_tag_category.terraform-test-category.id}"
}

resource "vsphere_virtual_machine" "vm" {
  name             = "terraform-test"
  resource_pool_id = "${data.vsphere_resource_pool.pool.id}"
  datastore_id     = "${data.vsphere_datastore.datastore.id}"

  num_cpus = 2
  memory   = 2048
  guest_id = "other3xLinux64Guest"

  network_interface {
    network_id = "${data.vsphere_network.network.id}"
  }

  disk {
    label = "disk0"
    size  = 20
  }

  tags = ["${vsphere_tag.terraform-test-tags-alt.*.id}"]
}
`,
		os.Getenv("VSPHERE_DATACENTER"),
		os.Getenv("VSPHERE_RESOURCE_POOL"),
		os.Getenv("VSPHERE_NETWORK_LABEL_PXE"),
		os.Getenv("VSPHERE_DATASTORE"),
	)
}

func testAccResourceVSphereVirtualMachineConfigComputedDisk() string {
	return fmt.Sprintf(`
variable "datacenter" {
  default = "%s"
}

variable "resource_pool" {
  default = "%s"
}

variable "network_label" {
  default = "%s"
}

variable "datastore" {
  default = "%s"
}

data "vsphere_datacenter" "dc" {
  name = "${var.datacenter}"
}

data "vsphere_datastore" "datastore" {
  name          = "${var.datastore}"
  datacenter_id = "${data.vsphere_datacenter.dc.id}"
}

data "vsphere_resource_pool" "pool" {
  name          = "${var.resource_pool}"
  datacenter_id = "${data.vsphere_datacenter.dc.id}"
}

data "vsphere_network" "network" {
  name          = "${var.network_label}"
  datacenter_id = "${data.vsphere_datacenter.dc.id}"
}

resource "random_pet" "pet" {}

resource "vsphere_virtual_machine" "vm" {
  name             = "terraform-test-${random_pet.pet.id}"
  resource_pool_id = "${data.vsphere_resource_pool.pool.id}"
  datastore_id     = "${data.vsphere_datastore.datastore.id}"

  num_cpus = 2
  memory   = 2048
  guest_id = "other3xLinux64Guest"

  network_interface {
    network_id = "${data.vsphere_network.network.id}"
  }

  disk {
    label = "terraform-test-${random_pet.pet.id}.vmdk"
    size  = 20
  }
}
`,
		os.Getenv("VSPHERE_DATACENTER"),
		os.Getenv("VSPHERE_RESOURCE_POOL"),
		os.Getenv("VSPHERE_NETWORK_LABEL_PXE"),
		os.Getenv("VSPHERE_DATASTORE"),
	)
}

func testAccResourceVSphereVirtualMachineVAppPropertiesNonClone() string {
	return fmt.Sprintf(`
variable "datacenter" {
  default = "%s"
}

variable "resource_pool" {
  default = "%s"
}

variable "network_label" {
  default = "%s"
}

variable "datastore" {
  default = "%s"
}

data "vsphere_datacenter" "dc" {
  name = "${var.datacenter}"
}

data "vsphere_datastore" "datastore" {
  name          = "${var.datastore}"
  datacenter_id = "${data.vsphere_datacenter.dc.id}"
}

data "vsphere_resource_pool" "pool" {
  name          = "${var.resource_pool}"
  datacenter_id = "${data.vsphere_datacenter.dc.id}"
}

data "vsphere_network" "network" {
  name          = "${var.network_label}"
  datacenter_id = "${data.vsphere_datacenter.dc.id}"
}

resource "vsphere_virtual_machine" "vm" {
  name             = "terraform-test"
  resource_pool_id = "${data.vsphere_resource_pool.pool.id}"
  datastore_id     = "${data.vsphere_datastore.datastore.id}"

  num_cpus = 2
  memory   = 2048
  guest_id = "other3xLinux64Guest"

  network_interface {
    network_id = "${data.vsphere_network.network.id}"
  }

  disk {
    label = "disk0"
    size  = 20
  }

  vapp {
    properties {
      "foo" = "bar"
    }
  }
}
`,
		os.Getenv("VSPHERE_DATACENTER"),
		os.Getenv("VSPHERE_RESOURCE_POOL"),
		os.Getenv("VSPHERE_NETWORK_LABEL_PXE"),
		os.Getenv("VSPHERE_DATASTORE"),
	)
}

func testAccResourceVSphereVirtualMachineConfigBadOrphanedLabel() string {
	return fmt.Sprintf(`
variable "datacenter" {
  default = "%s"
}

variable "resource_pool" {
  default = "%s"
}

variable "network_label" {
  default = "%s"
}

variable "datastore" {
  default = "%s"
}

data "vsphere_datacenter" "dc" {
  name = "${var.datacenter}"
}

data "vsphere_datastore" "datastore" {
  name          = "${var.datastore}"
  datacenter_id = "${data.vsphere_datacenter.dc.id}"
}

data "vsphere_resource_pool" "pool" {
  name          = "${var.resource_pool}"
  datacenter_id = "${data.vsphere_datacenter.dc.id}"
}

data "vsphere_network" "network" {
  name          = "${var.network_label}"
  datacenter_id = "${data.vsphere_datacenter.dc.id}"
}

resource "vsphere_virtual_machine" "vm" {
  name             = "terraform-test"
  resource_pool_id = "${data.vsphere_resource_pool.pool.id}"
  datastore_id     = "${data.vsphere_datastore.datastore.id}"

  num_cpus = 2
  memory   = 2048
  guest_id = "other3xLinux64Guest"

  network_interface {
    network_id = "${data.vsphere_network.network.id}"
  }

  disk {
    label = "orphaned_disk_0"
    size  = 20
  }
}
`,
		os.Getenv("VSPHERE_DATACENTER"),
		os.Getenv("VSPHERE_RESOURCE_POOL"),
		os.Getenv("VSPHERE_NETWORK_LABEL_PXE"),
		os.Getenv("VSPHERE_DATASTORE"),
	)
}

func testAccResourceVSphereVirtualMachineConfigClone() string {
	return fmt.Sprintf(`
variable "datacenter" {
  default = "%s"
}

variable "resource_pool" {
  default = "%s"
}

variable "network_label" {
  default = "%s"
}

variable "ipv4_address" {
  default = "%s"
}

variable "ipv4_netmask" {
  default = "%s"
}

variable "ipv4_gateway" {
  default = "%s"
}

variable "dns_server" {
  default = "%s"
}

variable "datastore" {
  default = "%s"
}

variable "template" {
  default = "%s"
}

variable "linked_clone" {
  default = "%s"
}

data "vsphere_datacenter" "dc" {
  name = "${var.datacenter}"
}

data "vsphere_datastore" "datastore" {
  name          = "${var.datastore}"
  datacenter_id = "${data.vsphere_datacenter.dc.id}"
}

data "vsphere_resource_pool" "pool" {
  name          = "${var.resource_pool}"
  datacenter_id = "${data.vsphere_datacenter.dc.id}"
}

data "vsphere_network" "network" {
  name          = "${var.network_label}"
  datacenter_id = "${data.vsphere_datacenter.dc.id}"
}

data "vsphere_virtual_machine" "template" {
  name          = "${var.template}"
  datacenter_id = "${data.vsphere_datacenter.dc.id}"
}

resource "vsphere_virtual_machine" "vm" {
  name             = "terraform-test"
  resource_pool_id = "${data.vsphere_resource_pool.pool.id}"
  datastore_id     = "${data.vsphere_datastore.datastore.id}"

  num_cpus = 2
  memory   = 2048
  guest_id = "${data.vsphere_virtual_machine.template.guest_id}"

  network_interface {
    network_id   = "${data.vsphere_network.network.id}"
    adapter_type = "${data.vsphere_virtual_machine.template.network_interface_types[0]}"
  }

  disk {
    label            = "disk0"
    size             = "${data.vsphere_virtual_machine.template.disks.0.size}"
    eagerly_scrub    = "${data.vsphere_virtual_machine.template.disks.0.eagerly_scrub}"
    thin_provisioned = "${data.vsphere_virtual_machine.template.disks.0.thin_provisioned}"
  }

  clone {
    template_uuid = "${data.vsphere_virtual_machine.template.id}"
    linked_clone  = "${var.linked_clone != "" ? "true" : "false" }"

    customize {
      linux_options {
        host_name = "terraform-test"
        domain    = "test.internal"
      }

      network_interface {
        ipv4_address = "${var.ipv4_address}"
        ipv4_netmask = "${var.ipv4_netmask}"
      }

      ipv4_gateway    = "${var.ipv4_gateway}"
      dns_server_list = ["${var.dns_server}"]
      dns_suffix_list = ["test.internal"]
    }
  }
}
`,
		os.Getenv("VSPHERE_DATACENTER"),
		os.Getenv("VSPHERE_RESOURCE_POOL"),
		os.Getenv("VSPHERE_NETWORK_LABEL"),
		os.Getenv("VSPHERE_IPV4_ADDRESS"),
		os.Getenv("VSPHERE_IPV4_PREFIX"),
		os.Getenv("VSPHERE_IPV4_GATEWAY"),
		os.Getenv("VSPHERE_DNS"),
		os.Getenv("VSPHERE_DATASTORE"),
		os.Getenv("VSPHERE_TEMPLATE"),
		os.Getenv("VSPHERE_USE_LINKED_CLONE"),
	)
}

func testAccResourceVSphereVirtualMachineConfigBadEager() string {
	return fmt.Sprintf(`
variable "datacenter" {
  default = "%s"
}

variable "resource_pool" {
  default = "%s"
}

variable "network_label" {
  default = "%s"
}

variable "ipv4_address" {
  default = "%s"
}

variable "ipv4_netmask" {
  default = "%s"
}

variable "ipv4_gateway" {
  default = "%s"
}

variable "dns_server" {
  default = "%s"
}

variable "datastore" {
  default = "%s"
}

variable "template" {
  default = "%s"
}

variable "linked_clone" {
  default = "%s"
}

data "vsphere_datacenter" "dc" {
  name = "${var.datacenter}"
}

data "vsphere_datastore" "datastore" {
  name          = "${var.datastore}"
  datacenter_id = "${data.vsphere_datacenter.dc.id}"
}

data "vsphere_resource_pool" "pool" {
  name          = "${var.resource_pool}"
  datacenter_id = "${data.vsphere_datacenter.dc.id}"
}

data "vsphere_network" "network" {
  name          = "${var.network_label}"
  datacenter_id = "${data.vsphere_datacenter.dc.id}"
}

data "vsphere_virtual_machine" "template" {
  name          = "${var.template}"
  datacenter_id = "${data.vsphere_datacenter.dc.id}"
}

resource "vsphere_virtual_machine" "vm" {
  name             = "terraform-test"
  resource_pool_id = "${data.vsphere_resource_pool.pool.id}"
  datastore_id     = "${data.vsphere_datastore.datastore.id}"

  num_cpus = 2
  memory   = 2048
  guest_id = "${data.vsphere_virtual_machine.template.guest_id}"

  network_interface {
    network_id   = "${data.vsphere_network.network.id}"
    adapter_type = "${data.vsphere_virtual_machine.template.network_interface_types[0]}"
  }

  disk {
    label            = "disk0"
    size             = "${data.vsphere_virtual_machine.template.disks.0.size}"
    eagerly_scrub    = "${data.vsphere_virtual_machine.template.disks.0.eagerly_scrub == "true" ? "false" : "true"}"
    thin_provisioned = "${data.vsphere_virtual_machine.template.disks.0.thin_provisioned}"
  }

  clone {
    template_uuid = "${data.vsphere_virtual_machine.template.id}"
    linked_clone  = true

    customize {
      linux_options {
        host_name = "terraform-test"
        domain    = "test.internal"
      }

      network_interface {
        ipv4_address = "${var.ipv4_address}"
        ipv4_netmask = "${var.ipv4_netmask}"
      }

      ipv4_gateway    = "${var.ipv4_gateway}"
      dns_server_list = ["${var.dns_server}"]
      dns_suffix_list = ["test.internal"]
    }
  }
}
`,
		os.Getenv("VSPHERE_DATACENTER"),
		os.Getenv("VSPHERE_RESOURCE_POOL"),
		os.Getenv("VSPHERE_NETWORK_LABEL"),
		os.Getenv("VSPHERE_IPV4_ADDRESS"),
		os.Getenv("VSPHERE_IPV4_PREFIX"),
		os.Getenv("VSPHERE_IPV4_GATEWAY"),
		os.Getenv("VSPHERE_DNS"),
		os.Getenv("VSPHERE_DATASTORE"),
		os.Getenv("VSPHERE_TEMPLATE"),
		os.Getenv("VSPHERE_USE_LINKED_CLONE"),
	)
}

func testAccResourceVSphereVirtualMachineConfigBadThin() string {
	return fmt.Sprintf(`
variable "datacenter" {
  default = "%s"
}

variable "resource_pool" {
  default = "%s"
}

variable "network_label" {
  default = "%s"
}

variable "ipv4_address" {
  default = "%s"
}

variable "ipv4_netmask" {
  default = "%s"
}

variable "ipv4_gateway" {
  default = "%s"
}

variable "dns_server" {
  default = "%s"
}

variable "datastore" {
  default = "%s"
}

variable "template" {
  default = "%s"
}

variable "linked_clone" {
  default = "%s"
}

data "vsphere_datacenter" "dc" {
  name = "${var.datacenter}"
}

data "vsphere_datastore" "datastore" {
  name          = "${var.datastore}"
  datacenter_id = "${data.vsphere_datacenter.dc.id}"
}

data "vsphere_resource_pool" "pool" {
  name          = "${var.resource_pool}"
  datacenter_id = "${data.vsphere_datacenter.dc.id}"
}

data "vsphere_network" "network" {
  name          = "${var.network_label}"
  datacenter_id = "${data.vsphere_datacenter.dc.id}"
}

data "vsphere_virtual_machine" "template" {
  name          = "${var.template}"
  datacenter_id = "${data.vsphere_datacenter.dc.id}"
}

resource "vsphere_virtual_machine" "vm" {
  name             = "terraform-test"
  resource_pool_id = "${data.vsphere_resource_pool.pool.id}"
  datastore_id     = "${data.vsphere_datastore.datastore.id}"

  num_cpus = 2
  memory   = 2048
  guest_id = "${data.vsphere_virtual_machine.template.guest_id}"

  network_interface {
    network_id   = "${data.vsphere_network.network.id}"
    adapter_type = "${data.vsphere_virtual_machine.template.network_interface_types[0]}"
  }

  disk {
    label            = "disk0"
    size             = "${data.vsphere_virtual_machine.template.disks.0.size}"
    eagerly_scrub    = "${data.vsphere_virtual_machine.template.disks.0.eagerly_scrub}"
    thin_provisioned = "${data.vsphere_virtual_machine.template.disks.0.thin_provisioned == "true" ? "false" : "true"}"
  }

  clone {
    template_uuid = "${data.vsphere_virtual_machine.template.id}"
    linked_clone  = true

    customize {
      linux_options {
        host_name = "terraform-test"
        domain    = "test.internal"
      }

      network_interface {
        ipv4_address = "${var.ipv4_address}"
        ipv4_netmask = "${var.ipv4_netmask}"
      }

      ipv4_gateway    = "${var.ipv4_gateway}"
      dns_server_list = ["${var.dns_server}"]
      dns_suffix_list = ["test.internal"]
    }
  }
}
`,
		os.Getenv("VSPHERE_DATACENTER"),
		os.Getenv("VSPHERE_RESOURCE_POOL"),
		os.Getenv("VSPHERE_NETWORK_LABEL"),
		os.Getenv("VSPHERE_IPV4_ADDRESS"),
		os.Getenv("VSPHERE_IPV4_PREFIX"),
		os.Getenv("VSPHERE_IPV4_GATEWAY"),
		os.Getenv("VSPHERE_DNS"),
		os.Getenv("VSPHERE_DATASTORE"),
		os.Getenv("VSPHERE_TEMPLATE"),
		os.Getenv("VSPHERE_USE_LINKED_CLONE"),
	)
}

func testAccResourceVSphereVirtualMachineConfigBadSizeLinked() string {
	return fmt.Sprintf(`
variable "datacenter" {
  default = "%s"
}

variable "resource_pool" {
  default = "%s"
}

variable "network_label" {
  default = "%s"
}

variable "ipv4_address" {
  default = "%s"
}

variable "ipv4_netmask" {
  default = "%s"
}

variable "ipv4_gateway" {
  default = "%s"
}

variable "dns_server" {
  default = "%s"
}

variable "datastore" {
  default = "%s"
}

variable "template" {
  default = "%s"
}

variable "linked_clone" {
  default = "%s"
}

data "vsphere_datacenter" "dc" {
  name = "${var.datacenter}"
}

data "vsphere_datastore" "datastore" {
  name          = "${var.datastore}"
  datacenter_id = "${data.vsphere_datacenter.dc.id}"
}

data "vsphere_resource_pool" "pool" {
  name          = "${var.resource_pool}"
  datacenter_id = "${data.vsphere_datacenter.dc.id}"
}

data "vsphere_network" "network" {
  name          = "${var.network_label}"
  datacenter_id = "${data.vsphere_datacenter.dc.id}"
}

data "vsphere_virtual_machine" "template" {
  name          = "${var.template}"
  datacenter_id = "${data.vsphere_datacenter.dc.id}"
}

resource "vsphere_virtual_machine" "vm" {
  name             = "terraform-test"
  resource_pool_id = "${data.vsphere_resource_pool.pool.id}"
  datastore_id     = "${data.vsphere_datastore.datastore.id}"

  num_cpus = 2
  memory   = 2048
  guest_id = "${data.vsphere_virtual_machine.template.guest_id}"

  network_interface {
    network_id   = "${data.vsphere_network.network.id}"
    adapter_type = "${data.vsphere_virtual_machine.template.network_interface_types[0]}"
  }

  disk {
    label            = "disk0"
    size             = 999
    eagerly_scrub    = "${data.vsphere_virtual_machine.template.disks.0.eagerly_scrub}"
    thin_provisioned = "${data.vsphere_virtual_machine.template.disks.0.thin_provisioned}"
  }

  clone {
    template_uuid = "${data.vsphere_virtual_machine.template.id}"
    linked_clone  = true

    customize {
      linux_options {
        host_name = "terraform-test"
        domain    = "test.internal"
      }

      network_interface {
        ipv4_address = "${var.ipv4_address}"
        ipv4_netmask = "${var.ipv4_netmask}"
      }

      ipv4_gateway    = "${var.ipv4_gateway}"
      dns_server_list = ["${var.dns_server}"]
      dns_suffix_list = ["test.internal"]
    }
  }
}
`,
		os.Getenv("VSPHERE_DATACENTER"),
		os.Getenv("VSPHERE_RESOURCE_POOL"),
		os.Getenv("VSPHERE_NETWORK_LABEL"),
		os.Getenv("VSPHERE_IPV4_ADDRESS"),
		os.Getenv("VSPHERE_IPV4_PREFIX"),
		os.Getenv("VSPHERE_IPV4_GATEWAY"),
		os.Getenv("VSPHERE_DNS"),
		os.Getenv("VSPHERE_DATASTORE"),
		os.Getenv("VSPHERE_TEMPLATE"),
		os.Getenv("VSPHERE_USE_LINKED_CLONE"),
	)
}

func testAccResourceVSphereVirtualMachineConfigBadSizeUnlinked() string {
	return fmt.Sprintf(`
variable "datacenter" {
  default = "%s"
}

variable "resource_pool" {
  default = "%s"
}

variable "network_label" {
  default = "%s"
}

variable "ipv4_address" {
  default = "%s"
}

variable "ipv4_netmask" {
  default = "%s"
}

variable "ipv4_gateway" {
  default = "%s"
}

variable "dns_server" {
  default = "%s"
}

variable "datastore" {
  default = "%s"
}

variable "template" {
  default = "%s"
}

variable "linked_clone" {
  default = "%s"
}

data "vsphere_datacenter" "dc" {
  name = "${var.datacenter}"
}

data "vsphere_datastore" "datastore" {
  name          = "${var.datastore}"
  datacenter_id = "${data.vsphere_datacenter.dc.id}"
}

data "vsphere_resource_pool" "pool" {
  name          = "${var.resource_pool}"
  datacenter_id = "${data.vsphere_datacenter.dc.id}"
}

data "vsphere_network" "network" {
  name          = "${var.network_label}"
  datacenter_id = "${data.vsphere_datacenter.dc.id}"
}

data "vsphere_virtual_machine" "template" {
  name          = "${var.template}"
  datacenter_id = "${data.vsphere_datacenter.dc.id}"
}

resource "vsphere_virtual_machine" "vm" {
  name             = "terraform-test"
  resource_pool_id = "${data.vsphere_resource_pool.pool.id}"
  datastore_id     = "${data.vsphere_datastore.datastore.id}"

  num_cpus = 2
  memory   = 2048
  guest_id = "${data.vsphere_virtual_machine.template.guest_id}"

  network_interface {
    network_id   = "${data.vsphere_network.network.id}"
    adapter_type = "${data.vsphere_virtual_machine.template.network_interface_types[0]}"
  }

  disk {
    label            = "disk0"
    size             = 1
    eagerly_scrub    = "${data.vsphere_virtual_machine.template.disks.0.eagerly_scrub}"
    thin_provisioned = "${data.vsphere_virtual_machine.template.disks.0.thin_provisioned}"
  }

  clone {
    template_uuid = "${data.vsphere_virtual_machine.template.id}"

    customize {
      linux_options {
        host_name = "terraform-test"
        domain    = "test.internal"
      }

      network_interface {
        ipv4_address = "${var.ipv4_address}"
        ipv4_netmask = "${var.ipv4_netmask}"
      }

      ipv4_gateway    = "${var.ipv4_gateway}"
      dns_server_list = ["${var.dns_server}"]
      dns_suffix_list = ["test.internal"]
    }
  }
}
`,
		os.Getenv("VSPHERE_DATACENTER"),
		os.Getenv("VSPHERE_RESOURCE_POOL"),
		os.Getenv("VSPHERE_NETWORK_LABEL"),
		os.Getenv("VSPHERE_IPV4_ADDRESS"),
		os.Getenv("VSPHERE_IPV4_PREFIX"),
		os.Getenv("VSPHERE_IPV4_GATEWAY"),
		os.Getenv("VSPHERE_DNS"),
		os.Getenv("VSPHERE_DATASTORE"),
		os.Getenv("VSPHERE_TEMPLATE"),
		os.Getenv("VSPHERE_USE_LINKED_CLONE"),
	)
}

func testAccResourceVSphereVirtualMachineConfigCloneMultiNIC() string {
	return fmt.Sprintf(`
variable "datacenter" {
  default = "%s"
}

variable "resource_pool" {
  default = "%s"
}

variable "network_label" {
  default = "%s"
}

variable "ipv4_address" {
  default = "%s"
}

variable "ipv4_netmask" {
  default = "%s"
}

variable "ipv4_gateway" {
  default = "%s"
}

variable "dns_server" {
  default = "%s"
}

variable "datastore" {
  default = "%s"
}

variable "template" {
  default = "%s"
}

variable "linked_clone" {
  default = "%s"
}

data "vsphere_datacenter" "dc" {
  name = "${var.datacenter}"
}

data "vsphere_datastore" "datastore" {
  name          = "${var.datastore}"
  datacenter_id = "${data.vsphere_datacenter.dc.id}"
}

data "vsphere_resource_pool" "pool" {
  name          = "${var.resource_pool}"
  datacenter_id = "${data.vsphere_datacenter.dc.id}"
}

data "vsphere_network" "network" {
  name          = "${var.network_label}"
  datacenter_id = "${data.vsphere_datacenter.dc.id}"
}

data "vsphere_virtual_machine" "template" {
  name          = "${var.template}"
  datacenter_id = "${data.vsphere_datacenter.dc.id}"
}

resource "vsphere_virtual_machine" "vm" {
  name             = "terraform-test"
  resource_pool_id = "${data.vsphere_resource_pool.pool.id}"
  datastore_id     = "${data.vsphere_datastore.datastore.id}"

  num_cpus = 2
  memory   = 2048
  guest_id = "${data.vsphere_virtual_machine.template.guest_id}"

  network_interface {
    network_id = "${data.vsphere_network.network.id}"
  }

  network_interface {
    network_id = "${data.vsphere_network.network.id}"
  }

  disk {
    label            = "disk0"
    size             = "${data.vsphere_virtual_machine.template.disks.0.size}"
    eagerly_scrub    = "${data.vsphere_virtual_machine.template.disks.0.eagerly_scrub}"
    thin_provisioned = "${data.vsphere_virtual_machine.template.disks.0.thin_provisioned}"
  }

  clone {
    template_uuid = "${data.vsphere_virtual_machine.template.id}"
    linked_clone  = "${var.linked_clone != "" ? "true" : "false" }"

    customize {
      linux_options {
        host_name = "terraform-test"
        domain    = "test.internal"
      }

      network_interface {
        ipv4_address = "${var.ipv4_address}"
        ipv4_netmask = "${var.ipv4_netmask}"
      }

      network_interface {}

      ipv4_gateway    = "${var.ipv4_gateway}"
      dns_server_list = ["${var.dns_server}"]
      dns_suffix_list = ["test.internal"]
    }
  }
}
`,
		os.Getenv("VSPHERE_DATACENTER"),
		os.Getenv("VSPHERE_RESOURCE_POOL"),
		os.Getenv("VSPHERE_NETWORK_LABEL"),
		os.Getenv("VSPHERE_IPV4_ADDRESS"),
		os.Getenv("VSPHERE_IPV4_PREFIX"),
		os.Getenv("VSPHERE_IPV4_GATEWAY"),
		os.Getenv("VSPHERE_DNS"),
		os.Getenv("VSPHERE_DATASTORE"),
		os.Getenv("VSPHERE_TEMPLATE"),
		os.Getenv("VSPHERE_USE_LINKED_CLONE"),
	)
}

func testAccResourceVSphereVirtualMachineConfigCloneTimeZone(zone string) string {
	return fmt.Sprintf(`
variable "datacenter" {
  default = "%s"
}

variable "resource_pool" {
  default = "%s"
}

variable "network_label" {
  default = "%s"
}

variable "ipv4_address" {
  default = "%s"
}

variable "ipv4_netmask" {
  default = "%s"
}

variable "ipv4_gateway" {
  default = "%s"
}

variable "dns_server" {
  default = "%s"
}

variable "datastore" {
  default = "%s"
}

variable "template" {
  default = "%s"
}

variable "linked_clone" {
  default = "%s"
}

variable "time_zone" {
  default = "%s"
}

data "vsphere_datacenter" "dc" {
  name = "${var.datacenter}"
}

data "vsphere_datastore" "datastore" {
  name          = "${var.datastore}"
  datacenter_id = "${data.vsphere_datacenter.dc.id}"
}

data "vsphere_resource_pool" "pool" {
  name          = "${var.resource_pool}"
  datacenter_id = "${data.vsphere_datacenter.dc.id}"
}

data "vsphere_network" "network" {
  name          = "${var.network_label}"
  datacenter_id = "${data.vsphere_datacenter.dc.id}"
}

data "vsphere_virtual_machine" "template" {
  name          = "${var.template}"
  datacenter_id = "${data.vsphere_datacenter.dc.id}"
}

resource "vsphere_virtual_machine" "vm" {
  name             = "terraform-test"
  resource_pool_id = "${data.vsphere_resource_pool.pool.id}"
  datastore_id     = "${data.vsphere_datastore.datastore.id}"

  num_cpus = 2
  memory   = 2048
  guest_id = "${data.vsphere_virtual_machine.template.guest_id}"

  network_interface {
    network_id   = "${data.vsphere_network.network.id}"
    adapter_type = "${data.vsphere_virtual_machine.template.network_interface_types[0]}"
  }

  disk {
    label            = "disk0"
    size             = "${data.vsphere_virtual_machine.template.disks.0.size}"
    eagerly_scrub    = "${data.vsphere_virtual_machine.template.disks.0.eagerly_scrub}"
    thin_provisioned = "${data.vsphere_virtual_machine.template.disks.0.thin_provisioned}"
  }

  clone {
    template_uuid = "${data.vsphere_virtual_machine.template.id}"
    linked_clone  = "${var.linked_clone != "" ? "true" : "false" }"

    customize {
      linux_options {
        host_name = "terraform-test"
        domain    = "test.internal"
        time_zone = "${var.time_zone}"
      }

      network_interface {
        ipv4_address = "${var.ipv4_address}"
        ipv4_netmask = "${var.ipv4_netmask}"
      }

      ipv4_gateway    = "${var.ipv4_gateway}"
      dns_server_list = ["${var.dns_server}"]
      dns_suffix_list = ["test.internal"]
    }
  }
}
`,
		os.Getenv("VSPHERE_DATACENTER"),
		os.Getenv("VSPHERE_RESOURCE_POOL"),
		os.Getenv("VSPHERE_NETWORK_LABEL"),
		os.Getenv("VSPHERE_IPV4_ADDRESS"),
		os.Getenv("VSPHERE_IPV4_PREFIX"),
		os.Getenv("VSPHERE_IPV4_GATEWAY"),
		os.Getenv("VSPHERE_DNS"),
		os.Getenv("VSPHERE_DATASTORE"),
		os.Getenv("VSPHERE_TEMPLATE"),
		os.Getenv("VSPHERE_USE_LINKED_CLONE"),
		zone,
	)
}

func testAccResourceVSphereVirtualMachineConfigCloneHostname() string {
	return fmt.Sprintf(`
variable "datacenter" {
  default = "%s"
}

variable "resource_pool" {
  default = "%s"
}

variable "network_label" {
  default = "%s"
}

variable "ipv4_address" {
  default = "%s"
}

variable "ipv4_netmask" {
  default = "%s"
}

variable "ipv4_gateway" {
  default = "%s"
}

variable "dns_server" {
  default = "%s"
}

variable "datastore" {
  default = "%s"
}

variable "template" {
  default = "%s"
}

variable "linked_clone" {
  default = "%s"
}

data "vsphere_datacenter" "dc" {
  name = "${var.datacenter}"
}

data "vsphere_datastore" "datastore" {
  name          = "${var.datastore}"
  datacenter_id = "${data.vsphere_datacenter.dc.id}"
}

data "vsphere_resource_pool" "pool" {
  name          = "${var.resource_pool}"
  datacenter_id = "${data.vsphere_datacenter.dc.id}"
}

data "vsphere_network" "network" {
  name          = "${var.network_label}"
  datacenter_id = "${data.vsphere_datacenter.dc.id}"
}

data "vsphere_virtual_machine" "template" {
  name          = "${var.template}"
  datacenter_id = "${data.vsphere_datacenter.dc.id}"
}

resource "vsphere_virtual_machine" "vm" {
  name             = "terraform-test"
  resource_pool_id = "${data.vsphere_resource_pool.pool.id}"
  datastore_id     = "${data.vsphere_datastore.datastore.id}"

  num_cpus = 2
  memory   = 2048
  guest_id = "${data.vsphere_virtual_machine.template.guest_id}"

  network_interface {
    network_id   = "${data.vsphere_network.network.id}"
    adapter_type = "${data.vsphere_virtual_machine.template.network_interface_types[0]}"
  }

  disk {
    label            = "disk0"
    size             = "${data.vsphere_virtual_machine.template.disks.0.size}"
    eagerly_scrub    = "${data.vsphere_virtual_machine.template.disks.0.eagerly_scrub}"
    thin_provisioned = "${data.vsphere_virtual_machine.template.disks.0.thin_provisioned}"
  }

  clone {
    template_uuid = "${data.vsphere_virtual_machine.template.id}"
    linked_clone  = "${var.linked_clone != "" ? "true" : "false" }"

    customize {
      linux_options {
        host_name = "terraform-test-renamed"
        domain    = "test.internal"
      }

      network_interface {
        ipv4_address = "${var.ipv4_address}"
        ipv4_netmask = "${var.ipv4_netmask}"
      }

      ipv4_gateway    = "${var.ipv4_gateway}"
      dns_server_list = ["${var.dns_server}"]
      dns_suffix_list = ["test.internal"]
    }
  }
}
`,
		os.Getenv("VSPHERE_DATACENTER"),
		os.Getenv("VSPHERE_RESOURCE_POOL"),
		os.Getenv("VSPHERE_NETWORK_LABEL"),
		os.Getenv("VSPHERE_IPV4_ADDRESS"),
		os.Getenv("VSPHERE_IPV4_PREFIX"),
		os.Getenv("VSPHERE_IPV4_GATEWAY"),
		os.Getenv("VSPHERE_DNS"),
		os.Getenv("VSPHERE_DATASTORE"),
		os.Getenv("VSPHERE_TEMPLATE"),
		os.Getenv("VSPHERE_USE_LINKED_CLONE"),
	)
}

func testAccResourceVSphereVirtualMachineConfigCloneWithCdrom() string {
	return fmt.Sprintf(`
variable "datacenter" {
  default = "%s"
}

variable "resource_pool" {
  default = "%s"
}

variable "network_label" {
  default = "%s"
}

variable "ipv4_address" {
  default = "%s"
}

variable "ipv4_netmask" {
  default = "%s"
}

variable "ipv4_gateway" {
  default = "%s"
}

variable "dns_server" {
  default = "%s"
}

variable "datastore" {
  default = "%s"
}

variable "iso_datastore" {
  default = "%s"
}

variable "iso_path" {
  default = "%s"
}

variable "template" {
  default = "%s"
}

variable "linked_clone" {
  default = "%s"
}

data "vsphere_datacenter" "dc" {
  name = "${var.datacenter}"
}

data "vsphere_datastore" "datastore" {
  name          = "${var.datastore}"
  datacenter_id = "${data.vsphere_datacenter.dc.id}"
}

data "vsphere_datastore" "iso_datastore" {
  name          = "${var.iso_datastore}"
  datacenter_id = "${data.vsphere_datacenter.dc.id}"
}

data "vsphere_resource_pool" "pool" {
  name          = "${var.resource_pool}"
  datacenter_id = "${data.vsphere_datacenter.dc.id}"
}

data "vsphere_network" "network" {
  name          = "${var.network_label}"
  datacenter_id = "${data.vsphere_datacenter.dc.id}"
}

data "vsphere_virtual_machine" "template" {
  name          = "${var.template}"
  datacenter_id = "${data.vsphere_datacenter.dc.id}"
}

resource "vsphere_virtual_machine" "vm" {
  name             = "terraform-test"
  resource_pool_id = "${data.vsphere_resource_pool.pool.id}"
  datastore_id     = "${data.vsphere_datastore.datastore.id}"

  num_cpus = 2
  memory   = 2048
  guest_id = "${data.vsphere_virtual_machine.template.guest_id}"

  network_interface {
    network_id   = "${data.vsphere_network.network.id}"
    adapter_type = "${data.vsphere_virtual_machine.template.network_interface_types[0]}"
  }

  disk {
    label            = "disk0"
    size             = "${data.vsphere_virtual_machine.template.disks.0.size}"
    eagerly_scrub    = "${data.vsphere_virtual_machine.template.disks.0.eagerly_scrub}"
    thin_provisioned = "${data.vsphere_virtual_machine.template.disks.0.thin_provisioned}"
  }

  cdrom {
    datastore_id = "${data.vsphere_datastore.iso_datastore.id}"
    path         = "${var.iso_path}"
  }

  clone {
    template_uuid = "${data.vsphere_virtual_machine.template.id}"
    linked_clone  = "${var.linked_clone != "" ? "true" : "false" }"

    customize {
      linux_options {
        host_name = "terraform-test"
        domain    = "test.internal"
      }

      network_interface {
        ipv4_address = "${var.ipv4_address}"
        ipv4_netmask = "${var.ipv4_netmask}"
      }

      ipv4_gateway    = "${var.ipv4_gateway}"
      dns_server_list = ["${var.dns_server}"]
      dns_suffix_list = ["test.internal"]
    }
  }
}
`,
		os.Getenv("VSPHERE_DATACENTER"),
		os.Getenv("VSPHERE_RESOURCE_POOL"),
		os.Getenv("VSPHERE_NETWORK_LABEL"),
		os.Getenv("VSPHERE_IPV4_ADDRESS"),
		os.Getenv("VSPHERE_IPV4_PREFIX"),
		os.Getenv("VSPHERE_IPV4_GATEWAY"),
		os.Getenv("VSPHERE_DNS"),
		os.Getenv("VSPHERE_DATASTORE"),
		os.Getenv("VSPHERE_ISO_DATASTORE"),
		os.Getenv("VSPHERE_ISO_FILE"),
		os.Getenv("VSPHERE_TEMPLATE"),
		os.Getenv("VSPHERE_USE_LINKED_CLONE"),
	)
}

func testAccResourceVSphereVirtualMachineConfigWithHotAdd(nc, nm int, cha, chr, mha bool) string {
	return fmt.Sprintf(`
variable "datacenter" {
  default = "%s"
}

variable "resource_pool" {
  default = "%s"
}

variable "network_label" {
  default = "%s"
}

variable "ipv4_address" {
  default = "%s"
}

variable "ipv4_netmask" {
  default = "%s"
}

variable "ipv4_gateway" {
  default = "%s"
}

variable "dns_server" {
  default = "%s"
}

variable "datastore" {
  default = "%s"
}

variable "template" {
  default = "%s"
}

variable "linked_clone" {
  default = "%s"
}

data "vsphere_datacenter" "dc" {
  name = "${var.datacenter}"
}

data "vsphere_datastore" "datastore" {
  name          = "${var.datastore}"
  datacenter_id = "${data.vsphere_datacenter.dc.id}"
}

data "vsphere_resource_pool" "pool" {
  name          = "${var.resource_pool}"
  datacenter_id = "${data.vsphere_datacenter.dc.id}"
}

data "vsphere_network" "network" {
  name          = "${var.network_label}"
  datacenter_id = "${data.vsphere_datacenter.dc.id}"
}

data "vsphere_virtual_machine" "template" {
  name          = "${var.template}"
  datacenter_id = "${data.vsphere_datacenter.dc.id}"
}

resource "vsphere_virtual_machine" "vm" {
  name             = "terraform-test"
  resource_pool_id = "${data.vsphere_resource_pool.pool.id}"
  datastore_id     = "${data.vsphere_datastore.datastore.id}"

  num_cpus                  = %d
  memory                    = %d
  cpu_hot_add_enabled       = %t
  cpu_hot_remove_enabled    = %t
  memory_hot_add_enabled    = %t
  guest_id = "${data.vsphere_virtual_machine.template.guest_id}"

  network_interface {
    network_id   = "${data.vsphere_network.network.id}"
    adapter_type = "${data.vsphere_virtual_machine.template.network_interface_types[0]}"
  }

  disk {
    label            = "disk0"
    size             = "${data.vsphere_virtual_machine.template.disks.0.size}"
    eagerly_scrub    = "${data.vsphere_virtual_machine.template.disks.0.eagerly_scrub}"
    thin_provisioned = "${data.vsphere_virtual_machine.template.disks.0.thin_provisioned}"
  }

  clone {
    template_uuid = "${data.vsphere_virtual_machine.template.id}"
    linked_clone  = "${var.linked_clone != "" ? "true" : "false" }"

    customize {
      linux_options {
        host_name = "terraform-test"
        domain    = "test.internal"
      }

      network_interface {
        ipv4_address = "${var.ipv4_address}"
        ipv4_netmask = "${var.ipv4_netmask}"
      }

      ipv4_gateway    = "${var.ipv4_gateway}"
      dns_server_list = ["${var.dns_server}"]
      dns_suffix_list = ["test.internal"]
    }
  }
}
`,
		os.Getenv("VSPHERE_DATACENTER"),
		os.Getenv("VSPHERE_RESOURCE_POOL"),
		os.Getenv("VSPHERE_NETWORK_LABEL"),
		os.Getenv("VSPHERE_IPV4_ADDRESS"),
		os.Getenv("VSPHERE_IPV4_PREFIX"),
		os.Getenv("VSPHERE_IPV4_GATEWAY"),
		os.Getenv("VSPHERE_DNS"),
		os.Getenv("VSPHERE_DATASTORE"),
		os.Getenv("VSPHERE_TEMPLATE"),
		os.Getenv("VSPHERE_USE_LINKED_CLONE"),
		nc,
		nm,
		cha,
		chr,
		mha,
	)
}

func testAccResourceVSphereVirtualMachineConfigDualStack() string {
	return fmt.Sprintf(`
variable "datacenter" {
  default = "%s"
}

variable "resource_pool" {
  default = "%s"
}

variable "network_label" {
  default = "%s"
}

variable "ipv4_address" {
  default = "%s"
}

variable "ipv4_netmask" {
  default = "%s"
}

variable "ipv4_gateway" {
  default = "%s"
}

variable "dns_server" {
  default = "%s"
}

variable "datastore" {
  default = "%s"
}

variable "template" {
  default = "%s"
}

variable "linked_clone" {
  default = "%s"
}

data "vsphere_datacenter" "dc" {
  name = "${var.datacenter}"
}

data "vsphere_datastore" "datastore" {
  name          = "${var.datastore}"
  datacenter_id = "${data.vsphere_datacenter.dc.id}"
}

data "vsphere_resource_pool" "pool" {
  name          = "${var.resource_pool}"
  datacenter_id = "${data.vsphere_datacenter.dc.id}"
}

data "vsphere_network" "network" {
  name          = "${var.network_label}"
  datacenter_id = "${data.vsphere_datacenter.dc.id}"
}

data "vsphere_virtual_machine" "template" {
  name          = "${var.template}"
  datacenter_id = "${data.vsphere_datacenter.dc.id}"
}

resource "vsphere_virtual_machine" "vm" {
  name             = "terraform-test"
  resource_pool_id = "${data.vsphere_resource_pool.pool.id}"
  datastore_id     = "${data.vsphere_datastore.datastore.id}"

  num_cpus = 2
  memory   = 2048
  guest_id = "${data.vsphere_virtual_machine.template.guest_id}"

  network_interface {
    network_id   = "${data.vsphere_network.network.id}"
    adapter_type = "${data.vsphere_virtual_machine.template.network_interface_types[0]}"
  }

  disk {
    label            = "disk0"
    size             = "${data.vsphere_virtual_machine.template.disks.0.size}"
    eagerly_scrub    = "${data.vsphere_virtual_machine.template.disks.0.eagerly_scrub}"
    thin_provisioned = "${data.vsphere_virtual_machine.template.disks.0.thin_provisioned}"
  }

  clone {
    template_uuid = "${data.vsphere_virtual_machine.template.id}"
    linked_clone  = "${var.linked_clone != "" ? "true" : "false" }"

    customize {
      linux_options {
        host_name = "terraform-test"
        domain    = "test.internal"
      }

      network_interface {
        ipv4_address = "${var.ipv4_address}"
        ipv4_netmask = "${var.ipv4_netmask}"
        ipv6_address = "fd00::2"
        ipv6_netmask = "32"
      }

      ipv4_gateway    = "${var.ipv4_gateway}"
      ipv6_gateway    = "fd00::1"
      dns_server_list = ["${var.dns_server}"]
      dns_suffix_list = ["test.internal"]
    }
  }
}
`,
		os.Getenv("VSPHERE_DATACENTER"),
		os.Getenv("VSPHERE_RESOURCE_POOL"),
		os.Getenv("VSPHERE_NETWORK_LABEL"),
		os.Getenv("VSPHERE_IPV4_ADDRESS"),
		os.Getenv("VSPHERE_IPV4_PREFIX"),
		os.Getenv("VSPHERE_IPV4_GATEWAY"),
		os.Getenv("VSPHERE_DNS"),
		os.Getenv("VSPHERE_DATASTORE"),
		os.Getenv("VSPHERE_TEMPLATE"),
		os.Getenv("VSPHERE_USE_LINKED_CLONE"),
	)
}

func testAccResourceVSphereVirtualMachineConfigIPv6Only() string {
	return fmt.Sprintf(`
variable "datacenter" {
  default = "%s"
}

variable "resource_pool" {
  default = "%s"
}

variable "network_label" {
  default = "%s"
}

variable "datastore" {
  default = "%s"
}

variable "template" {
  default = "%s"
}

variable "linked_clone" {
  default = "%s"
}

data "vsphere_datacenter" "dc" {
  name = "${var.datacenter}"
}

data "vsphere_datastore" "datastore" {
  name          = "${var.datastore}"
  datacenter_id = "${data.vsphere_datacenter.dc.id}"
}

data "vsphere_resource_pool" "pool" {
  name          = "${var.resource_pool}"
  datacenter_id = "${data.vsphere_datacenter.dc.id}"
}

data "vsphere_network" "network" {
  name          = "${var.network_label}"
  datacenter_id = "${data.vsphere_datacenter.dc.id}"
}

data "vsphere_virtual_machine" "template" {
  name          = "${var.template}"
  datacenter_id = "${data.vsphere_datacenter.dc.id}"
}

resource "vsphere_virtual_machine" "vm" {
  name             = "terraform-test"
  resource_pool_id = "${data.vsphere_resource_pool.pool.id}"
  datastore_id     = "${data.vsphere_datastore.datastore.id}"

  num_cpus = 2
  memory   = 2048
  guest_id = "${data.vsphere_virtual_machine.template.guest_id}"

  wait_for_guest_net_timeout = 10

  network_interface {
    network_id   = "${data.vsphere_network.network.id}"
    adapter_type = "${data.vsphere_virtual_machine.template.network_interface_types[0]}"
  }

  disk {
    label            = "disk0"
    size             = "${data.vsphere_virtual_machine.template.disks.0.size}"
    eagerly_scrub    = "${data.vsphere_virtual_machine.template.disks.0.eagerly_scrub}"
    thin_provisioned = "${data.vsphere_virtual_machine.template.disks.0.thin_provisioned}"
  }

  clone {
    template_uuid = "${data.vsphere_virtual_machine.template.id}"
    linked_clone  = "${var.linked_clone != "" ? "true" : "false" }"

    customize {
      linux_options {
        host_name = "terraform-test"
        domain    = "test.internal"
      }

      network_interface {
        ipv6_address = "fd00::2"
        ipv6_netmask = "32"
      }

      ipv6_gateway = "fd00::1"
    }
  }
}
`,
		os.Getenv("VSPHERE_DATACENTER"),
		os.Getenv("VSPHERE_RESOURCE_POOL"),
		os.Getenv("VSPHERE_NETWORK_LABEL"),
		os.Getenv("VSPHERE_DATASTORE"),
		os.Getenv("VSPHERE_TEMPLATE"),
		os.Getenv("VSPHERE_USE_LINKED_CLONE"),
	)
}

func testAccResourceVSphereVirtualMachineConfigHostVMotion(host string) string {
	return fmt.Sprintf(`
variable "datacenter" {
  default = "%s"
}

variable "resource_pool" {
  default = "%s"
}

variable "network_label" {
  default = "%s"
}

variable "ipv4_address" {
  default = "%s"
}

variable "ipv4_netmask" {
  default = "%s"
}

variable "ipv4_gateway" {
  default = "%s"
}

variable "dns_server" {
  default = "%s"
}

variable "datastore" {
  default = "%s"
}

variable "template" {
  default = "%s"
}

variable "linked_clone" {
  default = "%s"
}

variable "host" {
  default = "%s"
}

data "vsphere_datacenter" "dc" {
  name = "${var.datacenter}"
}

data "vsphere_datastore" "datastore" {
  name          = "${var.datastore}"
  datacenter_id = "${data.vsphere_datacenter.dc.id}"
}

data "vsphere_resource_pool" "pool" {
  name          = "${var.resource_pool}"
  datacenter_id = "${data.vsphere_datacenter.dc.id}"
}

data "vsphere_host" "host" {
  name          = "${var.host}"
  datacenter_id = "${data.vsphere_datacenter.dc.id}"
}

data "vsphere_network" "network" {
  name          = "${var.network_label}"
  datacenter_id = "${data.vsphere_datacenter.dc.id}"
}

data "vsphere_virtual_machine" "template" {
  name          = "${var.template}"
  datacenter_id = "${data.vsphere_datacenter.dc.id}"
}

resource "vsphere_virtual_machine" "vm" {
  name             = "terraform-test"
  resource_pool_id = "${data.vsphere_resource_pool.pool.id}"
  host_system_id   = "${data.vsphere_host.host.id}"
  datastore_id     = "${data.vsphere_datastore.datastore.id}"

  num_cpus = 2
  memory   = 2048
  guest_id = "ubuntu64Guest"

  network_interface {
    network_id   = "${data.vsphere_network.network.id}"
    adapter_type = "${data.vsphere_virtual_machine.template.network_interface_types[0]}"
  }

  disk {
    label            = "disk0"
    size             = "${data.vsphere_virtual_machine.template.disks.0.size}"
    eagerly_scrub    = "${data.vsphere_virtual_machine.template.disks.0.eagerly_scrub}"
    thin_provisioned = "${data.vsphere_virtual_machine.template.disks.0.thin_provisioned}"
  }

  clone {
    template_uuid = "${data.vsphere_virtual_machine.template.id}"
    linked_clone  = "${var.linked_clone != "" ? "true" : "false" }"

    customize {
      linux_options {
        host_name = "terraform-test"
        domain    = "test.internal"
      }

      network_interface {
        ipv4_address = "${var.ipv4_address}"
        ipv4_netmask = "${var.ipv4_netmask}"
      }

      ipv4_gateway    = "${var.ipv4_gateway}"
      dns_server_list = ["${var.dns_server}"]
      dns_suffix_list = ["test.internal"]
    }
  }
}
`,
		os.Getenv("VSPHERE_DATACENTER"),
		os.Getenv("VSPHERE_RESOURCE_POOL"),
		os.Getenv("VSPHERE_NETWORK_LABEL"),
		os.Getenv("VSPHERE_IPV4_ADDRESS"),
		os.Getenv("VSPHERE_IPV4_PREFIX"),
		os.Getenv("VSPHERE_IPV4_GATEWAY"),
		os.Getenv("VSPHERE_DNS"),
		os.Getenv("VSPHERE_DATASTORE"),
		os.Getenv("VSPHERE_TEMPLATE"),
		os.Getenv("VSPHERE_USE_LINKED_CLONE"),
		host,
	)
}

func testAccResourceVSphereVirtualMachineConfigResourcePoolVMotion(pool string) string {
	return fmt.Sprintf(`
variable "datacenter" {
  default = "%s"
}

variable "resource_pool" {
  default = "%s"
}

variable "network_label" {
  default = "%s"
}

variable "ipv4_address" {
  default = "%s"
}

variable "ipv4_netmask" {
  default = "%s"
}

variable "ipv4_gateway" {
  default = "%s"
}

variable "dns_server" {
  default = "%s"
}

variable "datastore" {
  default = "%s"
}

variable "template" {
  default = "%s"
}

variable "linked_clone" {
  default = "%s"
}

data "vsphere_datacenter" "dc" {
  name = "${var.datacenter}"
}

data "vsphere_datastore" "datastore" {
  name          = "${var.datastore}"
  datacenter_id = "${data.vsphere_datacenter.dc.id}"
}

data "vsphere_resource_pool" "pool" {
  name          = "${var.resource_pool}"
  datacenter_id = "${data.vsphere_datacenter.dc.id}"
}

data "vsphere_network" "network" {
  name          = "${var.network_label}"
  datacenter_id = "${data.vsphere_datacenter.dc.id}"
}

data "vsphere_virtual_machine" "template" {
  name          = "${var.template}"
  datacenter_id = "${data.vsphere_datacenter.dc.id}"
}

resource "vsphere_virtual_machine" "vm" {
  name             = "terraform-test"
  resource_pool_id = "${data.vsphere_resource_pool.pool.id}"
  datastore_id     = "${data.vsphere_datastore.datastore.id}"

  num_cpus = 2
  memory   = 2048
  guest_id = "ubuntu64Guest"

  network_interface {
    network_id   = "${data.vsphere_network.network.id}"
    adapter_type = "${data.vsphere_virtual_machine.template.network_interface_types[0]}"
  }

  disk {
    label            = "disk0"
    size             = "${data.vsphere_virtual_machine.template.disks.0.size}"
    eagerly_scrub    = "${data.vsphere_virtual_machine.template.disks.0.eagerly_scrub}"
    thin_provisioned = "${data.vsphere_virtual_machine.template.disks.0.thin_provisioned}"
  }

  clone {
    template_uuid = "${data.vsphere_virtual_machine.template.id}"
    linked_clone  = "${var.linked_clone != "" ? "true" : "false" }"

    customize {
      linux_options {
        host_name = "terraform-test"
        domain    = "test.internal"
      }

      network_interface {
        ipv4_address = "${var.ipv4_address}"
        ipv4_netmask = "${var.ipv4_netmask}"
      }

      ipv4_gateway    = "${var.ipv4_gateway}"
      dns_server_list = ["${var.dns_server}"]
      dns_suffix_list = ["test.internal"]
    }
  }
}
`,
		os.Getenv("VSPHERE_DATACENTER"),
		pool,
		os.Getenv("VSPHERE_NETWORK_LABEL"),
		os.Getenv("VSPHERE_IPV4_ADDRESS"),
		os.Getenv("VSPHERE_IPV4_PREFIX"),
		os.Getenv("VSPHERE_IPV4_GATEWAY"),
		os.Getenv("VSPHERE_DNS"),
		os.Getenv("VSPHERE_DATASTORE"),
		os.Getenv("VSPHERE_TEMPLATE"),
		os.Getenv("VSPHERE_USE_LINKED_CLONE"),
	)
}

func testAccResourceVSphereVirtualMachineConfigStorageVMotionGlobal(datastore string) string {
	return fmt.Sprintf(`
variable "datacenter" {
  default = "%s"
}

variable "resource_pool" {
  default = "%s"
}

variable "network_label" {
  default = "%s"
}

variable "ipv4_address" {
  default = "%s"
}

variable "ipv4_netmask" {
  default = "%s"
}

variable "ipv4_gateway" {
  default = "%s"
}

variable "dns_server" {
  default = "%s"
}

variable "datastore" {
  default = "%s"
}

variable "template" {
  default = "%s"
}

variable "linked_clone" {
  default = "%s"
}

data "vsphere_datacenter" "dc" {
  name = "${var.datacenter}"
}

data "vsphere_datastore" "datastore" {
  name          = "${var.datastore}"
  datacenter_id = "${data.vsphere_datacenter.dc.id}"
}

data "vsphere_resource_pool" "pool" {
  name          = "${var.resource_pool}"
  datacenter_id = "${data.vsphere_datacenter.dc.id}"
}

data "vsphere_network" "network" {
  name          = "${var.network_label}"
  datacenter_id = "${data.vsphere_datacenter.dc.id}"
}

data "vsphere_virtual_machine" "template" {
  name          = "${var.template}"
  datacenter_id = "${data.vsphere_datacenter.dc.id}"
}

resource "vsphere_virtual_machine" "vm" {
  name             = "terraform-test"
  resource_pool_id = "${data.vsphere_resource_pool.pool.id}"
  datastore_id     = "${data.vsphere_datastore.datastore.id}"

  num_cpus = 2
  memory   = 2048
  guest_id = "ubuntu64Guest"

  network_interface {
    network_id   = "${data.vsphere_network.network.id}"
    adapter_type = "${data.vsphere_virtual_machine.template.network_interface_types[0]}"
  }

  disk {
    label            = "disk0"
    size             = "${data.vsphere_virtual_machine.template.disks.0.size}"
    eagerly_scrub    = "${data.vsphere_virtual_machine.template.disks.0.eagerly_scrub}"
    thin_provisioned = "${data.vsphere_virtual_machine.template.disks.0.thin_provisioned}"
  }

  clone {
    template_uuid = "${data.vsphere_virtual_machine.template.id}"

    customize {
      linux_options {
        host_name = "terraform-test"
        domain    = "test.internal"
      }

      network_interface {
        ipv4_address = "${var.ipv4_address}"
        ipv4_netmask = "${var.ipv4_netmask}"
      }

      ipv4_gateway    = "${var.ipv4_gateway}"
      dns_server_list = ["${var.dns_server}"]
      dns_suffix_list = ["test.internal"]
    }
  }
}
`,
		os.Getenv("VSPHERE_DATACENTER"),
		os.Getenv("VSPHERE_RESOURCE_POOL"),
		os.Getenv("VSPHERE_NETWORK_LABEL"),
		os.Getenv("VSPHERE_IPV4_ADDRESS"),
		os.Getenv("VSPHERE_IPV4_PREFIX"),
		os.Getenv("VSPHERE_IPV4_GATEWAY"),
		os.Getenv("VSPHERE_DNS"),
		datastore,
		os.Getenv("VSPHERE_TEMPLATE"),
		os.Getenv("VSPHERE_USE_LINKED_CLONE"),
	)
}

func testAccResourceVSphereVirtualMachineConfigStorageVMotionSingleDisk(datastore string) string {
	return fmt.Sprintf(`
variable "datacenter" {
  default = "%s"
}

variable "resource_pool" {
  default = "%s"
}

variable "network_label" {
  default = "%s"
}

variable "ipv4_address" {
  default = "%s"
}

variable "ipv4_netmask" {
  default = "%s"
}

variable "ipv4_gateway" {
  default = "%s"
}

variable "dns_server" {
  default = "%s"
}

variable "datastore" {
  default = "%s"
}

variable "template" {
  default = "%s"
}

variable "linked_clone" {
  default = "%s"
}

variable "disk_datastore" {
  default = "%s"
}

data "vsphere_datacenter" "dc" {
  name = "${var.datacenter}"
}

data "vsphere_datastore" "datastore" {
  name          = "${var.datastore}"
  datacenter_id = "${data.vsphere_datacenter.dc.id}"
}

data "vsphere_datastore" "disk_datastore" {
  name          = "${var.disk_datastore}"
  datacenter_id = "${data.vsphere_datacenter.dc.id}"
}

data "vsphere_resource_pool" "pool" {
  name          = "${var.resource_pool}"
  datacenter_id = "${data.vsphere_datacenter.dc.id}"
}

data "vsphere_network" "network" {
  name          = "${var.network_label}"
  datacenter_id = "${data.vsphere_datacenter.dc.id}"
}

data "vsphere_virtual_machine" "template" {
  name          = "${var.template}"
  datacenter_id = "${data.vsphere_datacenter.dc.id}"
}

resource "vsphere_virtual_machine" "vm" {
  name             = "terraform-test"
  resource_pool_id = "${data.vsphere_resource_pool.pool.id}"
  datastore_id     = "${data.vsphere_datastore.datastore.id}"

  num_cpus = 2
  memory   = 2048
  guest_id = "ubuntu64Guest"

  network_interface {
    network_id   = "${data.vsphere_network.network.id}"
    adapter_type = "${data.vsphere_virtual_machine.template.network_interface_types[0]}"
  }

  disk {
    label            = "disk0"
    size             = "${data.vsphere_virtual_machine.template.disks.0.size}"
    eagerly_scrub    = "${data.vsphere_virtual_machine.template.disks.0.eagerly_scrub}"
    thin_provisioned = "${data.vsphere_virtual_machine.template.disks.0.thin_provisioned}"
  }

  disk {
    label        = "disk1"
    datastore_id = "${data.vsphere_datastore.disk_datastore.id}"
    size         = 1
    unit_number  = 1
  }

  clone {
    template_uuid = "${data.vsphere_virtual_machine.template.id}"

    customize {
      linux_options {
        host_name = "terraform-test"
        domain    = "test.internal"
      }

      network_interface {
        ipv4_address = "${var.ipv4_address}"
        ipv4_netmask = "${var.ipv4_netmask}"
      }

      ipv4_gateway    = "${var.ipv4_gateway}"
      dns_server_list = ["${var.dns_server}"]
      dns_suffix_list = ["test.internal"]
    }
  }
}
`,
		os.Getenv("VSPHERE_DATACENTER"),
		os.Getenv("VSPHERE_RESOURCE_POOL"),
		os.Getenv("VSPHERE_NETWORK_LABEL"),
		os.Getenv("VSPHERE_IPV4_ADDRESS"),
		os.Getenv("VSPHERE_IPV4_PREFIX"),
		os.Getenv("VSPHERE_IPV4_GATEWAY"),
		os.Getenv("VSPHERE_DNS"),
		os.Getenv("VSPHERE_DATASTORE"),
		os.Getenv("VSPHERE_TEMPLATE"),
		os.Getenv("VSPHERE_USE_LINKED_CLONE"),
		datastore,
	)
}

func testAccResourceVSphereVirtualMachineConfigStorageVMotionPinDatastore(datastore string) string {
	return fmt.Sprintf(`
variable "datacenter" {
  default = "%s"
}

variable "resource_pool" {
  default = "%s"
}

variable "network_label" {
  default = "%s"
}

variable "ipv4_address" {
  default = "%s"
}

variable "ipv4_netmask" {
  default = "%s"
}

variable "ipv4_gateway" {
  default = "%s"
}

variable "dns_server" {
  default = "%s"
}

variable "datastore" {
  default = "%s"
}

variable "template" {
  default = "%s"
}

variable "linked_clone" {
  default = "%s"
}

variable "disk_datastore" {
  default = "%s"
}

data "vsphere_datacenter" "dc" {
  name = "${var.datacenter}"
}

data "vsphere_datastore" "datastore" {
  name          = "${var.datastore}"
  datacenter_id = "${data.vsphere_datacenter.dc.id}"
}

data "vsphere_datastore" "disk_datastore" {
  name          = "${var.disk_datastore}"
  datacenter_id = "${data.vsphere_datacenter.dc.id}"
}

data "vsphere_resource_pool" "pool" {
  name          = "${var.resource_pool}"
  datacenter_id = "${data.vsphere_datacenter.dc.id}"
}

data "vsphere_network" "network" {
  name          = "${var.network_label}"
  datacenter_id = "${data.vsphere_datacenter.dc.id}"
}

data "vsphere_virtual_machine" "template" {
  name          = "${var.template}"
  datacenter_id = "${data.vsphere_datacenter.dc.id}"
}

resource "vsphere_virtual_machine" "vm" {
  name             = "terraform-test"
  resource_pool_id = "${data.vsphere_resource_pool.pool.id}"
  datastore_id     = "${data.vsphere_datastore.datastore.id}"

  num_cpus = 2
  memory   = 2048
  guest_id = "ubuntu64Guest"

  network_interface {
    network_id   = "${data.vsphere_network.network.id}"
    adapter_type = "${data.vsphere_virtual_machine.template.network_interface_types[0]}"
  }

  disk {
    label            = "disk0"
    size             = "${data.vsphere_virtual_machine.template.disks.0.size}"
    eagerly_scrub    = "${data.vsphere_virtual_machine.template.disks.0.eagerly_scrub}"
    thin_provisioned = "${data.vsphere_virtual_machine.template.disks.0.thin_provisioned}"
  }

  disk {
    label        = "disk1"
    datastore_id = "${data.vsphere_datastore.disk_datastore.id}"
    size         = 1
    unit_number  = 1
  }

  clone {
    template_uuid = "${data.vsphere_virtual_machine.template.id}"

    customize {
      linux_options {
        host_name = "terraform-test"
        domain    = "test.internal"
      }

      network_interface {
        ipv4_address = "${var.ipv4_address}"
        ipv4_netmask = "${var.ipv4_netmask}"
      }

      ipv4_gateway    = "${var.ipv4_gateway}"
      dns_server_list = ["${var.dns_server}"]
      dns_suffix_list = ["test.internal"]
    }
  }
}
`,
		os.Getenv("VSPHERE_DATACENTER"),
		os.Getenv("VSPHERE_RESOURCE_POOL"),
		os.Getenv("VSPHERE_NETWORK_LABEL"),
		os.Getenv("VSPHERE_IPV4_ADDRESS"),
		os.Getenv("VSPHERE_IPV4_PREFIX"),
		os.Getenv("VSPHERE_IPV4_GATEWAY"),
		os.Getenv("VSPHERE_DNS"),
		datastore,
		os.Getenv("VSPHERE_TEMPLATE"),
		os.Getenv("VSPHERE_USE_LINKED_CLONE"),
		os.Getenv("VSPHERE_DATASTORE"),
	)
}

func testAccResourceVSphereVirtualMachineConfigStorageVMotionRename(name, datastore string) string {
	return fmt.Sprintf(`
variable "datacenter" {
  default = "%s"
}

variable "resource_pool" {
  default = "%s"
}

variable "network_label" {
  default = "%s"
}

variable "ipv4_address" {
  default = "%s"
}

variable "ipv4_netmask" {
  default = "%s"
}

variable "ipv4_gateway" {
  default = "%s"
}

variable "dns_server" {
  default = "%s"
}

variable "vm_name" {
  default = "%s"
}

variable "datastore" {
  default = "%s"
}

variable "template" {
  default = "%s"
}

variable "linked_clone" {
  default = "%s"
}

data "vsphere_datacenter" "dc" {
  name = "${var.datacenter}"
}

data "vsphere_datastore" "datastore" {
  name          = "${var.datastore}"
  datacenter_id = "${data.vsphere_datacenter.dc.id}"
}

data "vsphere_resource_pool" "pool" {
  name          = "${var.resource_pool}"
  datacenter_id = "${data.vsphere_datacenter.dc.id}"
}

data "vsphere_network" "network" {
  name          = "${var.network_label}"
  datacenter_id = "${data.vsphere_datacenter.dc.id}"
}

data "vsphere_virtual_machine" "template" {
  name          = "${var.template}"
  datacenter_id = "${data.vsphere_datacenter.dc.id}"
}

resource "vsphere_virtual_machine" "vm" {
  name             = "${var.vm_name}"
  resource_pool_id = "${data.vsphere_resource_pool.pool.id}"
  datastore_id     = "${data.vsphere_datastore.datastore.id}"

  num_cpus = 2
  memory   = 2048
  guest_id = "ubuntu64Guest"

  network_interface {
    network_id   = "${data.vsphere_network.network.id}"
    adapter_type = "${data.vsphere_virtual_machine.template.network_interface_types[0]}"
  }

  disk {
    label            = "disk0"
    size             = "${data.vsphere_virtual_machine.template.disks.0.size}"
    eagerly_scrub    = "${data.vsphere_virtual_machine.template.disks.0.eagerly_scrub}"
    thin_provisioned = "${data.vsphere_virtual_machine.template.disks.0.thin_provisioned}"
  }

  clone {
    template_uuid = "${data.vsphere_virtual_machine.template.id}"

    customize {
      linux_options {
        host_name = "terraform-test"
        domain    = "test.internal"
      }

      network_interface {
        ipv4_address = "${var.ipv4_address}"
        ipv4_netmask = "${var.ipv4_netmask}"
      }

      ipv4_gateway    = "${var.ipv4_gateway}"
      dns_server_list = ["${var.dns_server}"]
      dns_suffix_list = ["test.internal"]
    }
  }
}
`,
		os.Getenv("VSPHERE_DATACENTER"),
		os.Getenv("VSPHERE_RESOURCE_POOL"),
		os.Getenv("VSPHERE_NETWORK_LABEL"),
		os.Getenv("VSPHERE_IPV4_ADDRESS"),
		os.Getenv("VSPHERE_IPV4_PREFIX"),
		os.Getenv("VSPHERE_IPV4_GATEWAY"),
		os.Getenv("VSPHERE_DNS"),
		name,
		datastore,
		os.Getenv("VSPHERE_TEMPLATE"),
		os.Getenv("VSPHERE_USE_LINKED_CLONE"),
	)
}

func testAccResourceVSphereVirtualMachineConfigStorageVMotionLinkedClone(datastore string) string {
	return fmt.Sprintf(`
variable "datacenter" {
  default = "%s"
}

variable "resource_pool" {
  default = "%s"
}

variable "network_label" {
  default = "%s"
}

variable "ipv4_address" {
  default = "%s"
}

variable "ipv4_netmask" {
  default = "%s"
}

variable "ipv4_gateway" {
  default = "%s"
}

variable "dns_server" {
  default = "%s"
}

variable "datastore" {
  default = "%s"
}

variable "template" {
  default = "%s"
}

variable "linked_clone" {
  default = "%s"
}

data "vsphere_datacenter" "dc" {
  name = "${var.datacenter}"
}

data "vsphere_datastore" "datastore" {
  name          = "${var.datastore}"
  datacenter_id = "${data.vsphere_datacenter.dc.id}"
}

data "vsphere_resource_pool" "pool" {
  name          = "${var.resource_pool}"
  datacenter_id = "${data.vsphere_datacenter.dc.id}"
}

data "vsphere_network" "network" {
  name          = "${var.network_label}"
  datacenter_id = "${data.vsphere_datacenter.dc.id}"
}

data "vsphere_virtual_machine" "template" {
  name          = "${var.template}"
  datacenter_id = "${data.vsphere_datacenter.dc.id}"
}

resource "vsphere_virtual_machine" "vm" {
  name             = "terraform-test"
  resource_pool_id = "${data.vsphere_resource_pool.pool.id}"
  datastore_id     = "${data.vsphere_datastore.datastore.id}"

  num_cpus = 2
  memory   = 2048
  guest_id = "ubuntu64Guest"

  network_interface {
    network_id   = "${data.vsphere_network.network.id}"
    adapter_type = "${data.vsphere_virtual_machine.template.network_interface_types[0]}"
  }

  disk {
    label            = "disk0"
    size             = "${data.vsphere_virtual_machine.template.disks.0.size}"
    eagerly_scrub    = "${data.vsphere_virtual_machine.template.disks.0.eagerly_scrub}"
    thin_provisioned = "${data.vsphere_virtual_machine.template.disks.0.thin_provisioned}"
  }

  clone {
    template_uuid = "${data.vsphere_virtual_machine.template.id}"
    linked_clone  = true

    customize {
      linux_options {
        host_name = "terraform-test"
        domain    = "test.internal"
      }

      network_interface {
        ipv4_address = "${var.ipv4_address}"
        ipv4_netmask = "${var.ipv4_netmask}"
      }

      ipv4_gateway    = "${var.ipv4_gateway}"
      dns_server_list = ["${var.dns_server}"]
      dns_suffix_list = ["test.internal"]
    }
  }
}
`,
		os.Getenv("VSPHERE_DATACENTER"),
		os.Getenv("VSPHERE_RESOURCE_POOL"),
		os.Getenv("VSPHERE_NETWORK_LABEL"),
		os.Getenv("VSPHERE_IPV4_ADDRESS"),
		os.Getenv("VSPHERE_IPV4_PREFIX"),
		os.Getenv("VSPHERE_IPV4_GATEWAY"),
		os.Getenv("VSPHERE_DNS"),
		datastore,
		os.Getenv("VSPHERE_TEMPLATE"),
		os.Getenv("VSPHERE_USE_LINKED_CLONE"),
	)
}

func testAccResourceVSphereVirtualMachineConfigStorageVMotionAttachedDisk(datastore string) string {
	return fmt.Sprintf(`
variable "datacenter" {
  default = "%s"
}

variable "resource_pool" {
  default = "%s"
}

variable "network_label" {
  default = "%s"
}

variable "ipv4_address" {
  default = "%s"
}

variable "ipv4_netmask" {
  default = "%s"
}

variable "ipv4_gateway" {
  default = "%s"
}

variable "dns_server" {
  default = "%s"
}

variable "datastore" {
  default = "%s"
}

variable "template" {
  default = "%s"
}

variable "linked_clone" {
  default = "%s"
}

variable "extra_vmdk_name" {
  default = "%s"
}

data "vsphere_datacenter" "dc" {
  name = "${var.datacenter}"
}

data "vsphere_datastore" "datastore" {
  name          = "${var.datastore}"
  datacenter_id = "${data.vsphere_datacenter.dc.id}"
}

data "vsphere_resource_pool" "pool" {
  name          = "${var.resource_pool}"
  datacenter_id = "${data.vsphere_datacenter.dc.id}"
}

data "vsphere_network" "network" {
  name          = "${var.network_label}"
  datacenter_id = "${data.vsphere_datacenter.dc.id}"
}

data "vsphere_virtual_machine" "template" {
  name          = "${var.template}"
  datacenter_id = "${data.vsphere_datacenter.dc.id}"
}

resource "vsphere_virtual_disk" "disk" {
  size         = 1
  vmdk_path    = "${var.extra_vmdk_name}"
  datacenter   = "${var.datacenter}"
  datastore    = "${var.datastore}"
  type         = "thin"
  adapter_type = "lsiLogic"
}

resource "vsphere_virtual_machine" "vm" {
  name             = "terraform-test"
  resource_pool_id = "${data.vsphere_resource_pool.pool.id}"
  datastore_id     = "${data.vsphere_datastore.datastore.id}"

  num_cpus = 2
  memory   = 2048
  guest_id = "ubuntu64Guest"

  network_interface {
    network_id   = "${data.vsphere_network.network.id}"
    adapter_type = "${data.vsphere_virtual_machine.template.network_interface_types[0]}"
  }

  disk {
    label            = "disk0"
    size             = "${data.vsphere_virtual_machine.template.disks.0.size}"
    eagerly_scrub    = "${data.vsphere_virtual_machine.template.disks.0.eagerly_scrub}"
    thin_provisioned = "${data.vsphere_virtual_machine.template.disks.0.thin_provisioned}"
  }

  disk {
    label        = "disk1"
    datastore_id = "${data.vsphere_datastore.datastore.id}"
    path         = "${vsphere_virtual_disk.disk.vmdk_path}"
    disk_mode    = "independent_persistent"
    attach       = true
    unit_number  = 1
  }

  clone {
    template_uuid = "${data.vsphere_virtual_machine.template.id}"

    customize {
      linux_options {
        host_name = "terraform-test"
        domain    = "test.internal"
      }

      network_interface {
        ipv4_address = "${var.ipv4_address}"
        ipv4_netmask = "${var.ipv4_netmask}"
      }

      ipv4_gateway    = "${var.ipv4_gateway}"
      dns_server_list = ["${var.dns_server}"]
      dns_suffix_list = ["test.internal"]
    }
  }
}
`,
		os.Getenv("VSPHERE_DATACENTER"),
		os.Getenv("VSPHERE_RESOURCE_POOL"),
		os.Getenv("VSPHERE_NETWORK_LABEL"),
		os.Getenv("VSPHERE_IPV4_ADDRESS"),
		os.Getenv("VSPHERE_IPV4_PREFIX"),
		os.Getenv("VSPHERE_IPV4_GATEWAY"),
		os.Getenv("VSPHERE_DNS"),
		datastore,
		os.Getenv("VSPHERE_TEMPLATE"),
		os.Getenv("VSPHERE_USE_LINKED_CLONE"),
		testAccResourceVSphereVirtualMachineDiskNameExtraVmdk,
	)
}

func testAccResourceVSphereVirtualMachineConfigCloneExtraDisks() string {
	return fmt.Sprintf(`
variable "datacenter" {
  default = "%s"
}

variable "resource_pool" {
  default = "%s"
}

variable "network_label" {
  default = "%s"
}

variable "ipv4_address" {
  default = "%s"
}

variable "ipv4_netmask" {
  default = "%s"
}

variable "ipv4_gateway" {
  default = "%s"
}

variable "dns_server" {
  default = "%s"
}

variable "datastore" {
  default = "%s"
}

variable "template" {
  default = "%s"
}

variable "disk0" {
  type    = "string"
  default = "%s"
}

variable "disk1" {
  type    = "string"
  default = "%s"
}

variable "disk2" {
  type    = "string"
  default = "%s"
}

variable "host" {
  type    = "string"
  default = "%s"
}

data "vsphere_datacenter" "dc" {
  name = "${var.datacenter}"
}

data "vsphere_host" "esxi_host" {
  name          = "${var.host}"
  datacenter_id = "${data.vsphere_datacenter.dc.id}"
}

data "vsphere_resource_pool" "pool" {
  name          = "${var.resource_pool}"
  datacenter_id = "${data.vsphere_datacenter.dc.id}"
}

data "vsphere_network" "network" {
  name          = "${var.network_label}"
  datacenter_id = "${data.vsphere_datacenter.dc.id}"
}

data "vsphere_virtual_machine" "template" {
  name          = "${var.template}"
  datacenter_id = "${data.vsphere_datacenter.dc.id}"
}

resource "vsphere_vmfs_datastore" "datastore" {
  name           = "terraform-test"
  host_system_id = "${data.vsphere_host.esxi_host.id}"

  disks = [
    "${var.disk0}",
    "${var.disk1}",
    "${var.disk2}",
  ]
}

resource "vsphere_virtual_machine" "vm" {
  name             = "terraform-test"
  resource_pool_id = "${data.vsphere_resource_pool.pool.id}"
  datastore_id     = "${vsphere_vmfs_datastore.datastore.id}"

  num_cpus = 2
  memory   = 2048
  guest_id = "${data.vsphere_virtual_machine.template.guest_id}"

  network_interface {
    network_id   = "${data.vsphere_network.network.id}"
    adapter_type = "${data.vsphere_virtual_machine.template.network_interface_types[0]}"
  }

  disk {
    label            = "disk0"
    size             = "${data.vsphere_virtual_machine.template.disks.0.size}"
    eagerly_scrub    = "${data.vsphere_virtual_machine.template.disks.0.eagerly_scrub}"
    thin_provisioned = "${data.vsphere_virtual_machine.template.disks.0.thin_provisioned}"
  }

  disk {
    label            = "disk1"
    size             = 1
    unit_number      = 1
    thin_provisioned = false
    eagerly_scrub    = true
  }

  disk {
    label            = "disk2"
    size             = 1
    unit_number      = 2
    thin_provisioned = false
    eagerly_scrub    = false
  }

  disk {
    label       = "disk3"
    size        = 1
    unit_number = 3
  }

  clone {
    template_uuid = "${data.vsphere_virtual_machine.template.id}"
    linked_clone  = false

    customize {
      linux_options {
        host_name = "terraform-test"
        domain    = "test.internal"
      }

      network_interface {
        ipv4_address = "${var.ipv4_address}"
        ipv4_netmask = "${var.ipv4_netmask}"
      }

      ipv4_gateway    = "${var.ipv4_gateway}"
      dns_server_list = ["${var.dns_server}"]
      dns_suffix_list = ["test.internal"]
    }
  }
}
`,
		os.Getenv("VSPHERE_DATACENTER"),
		os.Getenv("VSPHERE_RESOURCE_POOL"),
		os.Getenv("VSPHERE_NETWORK_LABEL"),
		os.Getenv("VSPHERE_IPV4_ADDRESS"),
		os.Getenv("VSPHERE_IPV4_PREFIX"),
		os.Getenv("VSPHERE_IPV4_GATEWAY"),
		os.Getenv("VSPHERE_DNS"),
		os.Getenv("VSPHERE_DATASTORE"),
		os.Getenv("VSPHERE_TEMPLATE"),
		os.Getenv("VSPHERE_DS_VMFS_DISK0"),
		os.Getenv("VSPHERE_DS_VMFS_DISK1"),
		os.Getenv("VSPHERE_DS_VMFS_DISK2"),
		os.Getenv("VSPHERE_ESXI_HOST"),
	)
}

func testAccResourceVSphereVirtualMachineConfigWindows() string {
	return fmt.Sprintf(`
variable "datacenter" {
  default = "%s"
}

variable "resource_pool" {
  default = "%s"
}

variable "network_label" {
  default = "%s"
}

variable "ipv4_address" {
  default = "%s"
}

variable "ipv4_netmask" {
  default = "%s"
}

variable "ipv4_gateway" {
  default = "%s"
}

variable "dns_server" {
  default = "%s"
}

variable "datastore" {
  default = "%s"
}

variable "template" {
  default = "%s"
}

variable "linked_clone" {
  default = "%s"
}

data "vsphere_datacenter" "dc" {
  name = "${var.datacenter}"
}

data "vsphere_datastore" "datastore" {
  name          = "${var.datastore}"
  datacenter_id = "${data.vsphere_datacenter.dc.id}"
}

data "vsphere_resource_pool" "pool" {
  name          = "${var.resource_pool}"
  datacenter_id = "${data.vsphere_datacenter.dc.id}"
}

data "vsphere_network" "network" {
  name          = "${var.network_label}"
  datacenter_id = "${data.vsphere_datacenter.dc.id}"
}

data "vsphere_virtual_machine" "template" {
  name          = "${var.template}"
  datacenter_id = "${data.vsphere_datacenter.dc.id}"
}

resource "vsphere_virtual_machine" "vm" {
  name             = "terraform-test"
  resource_pool_id = "${data.vsphere_resource_pool.pool.id}"
  datastore_id     = "${data.vsphere_datastore.datastore.id}"

  num_cpus = 4
  memory   = 4096
  guest_id = "${data.vsphere_virtual_machine.template.guest_id}"

  scsi_type = "${data.vsphere_virtual_machine.template.scsi_type}"

  network_interface {
    network_id   = "${data.vsphere_network.network.id}"
    adapter_type = "${data.vsphere_virtual_machine.template.network_interface_types[0]}"
  }

  disk {
    label            = "disk0"
    size             = "${data.vsphere_virtual_machine.template.disks.0.size}"
    eagerly_scrub    = "${data.vsphere_virtual_machine.template.disks.0.eagerly_scrub}"
    thin_provisioned = "${data.vsphere_virtual_machine.template.disks.0.thin_provisioned}"
  }

  clone {
    template_uuid = "${data.vsphere_virtual_machine.template.id}"
    linked_clone  = "${var.linked_clone != "" ? "true" : "false" }"

    customize {
      windows_options {
        computer_name  = "terraform-test"
        workgroup      = "test"
        admin_password = "VMw4re"
      }

      network_interface {
        ipv4_address    = "${var.ipv4_address}"
        ipv4_netmask    = "${var.ipv4_netmask}"
        dns_server_list = ["${var.dns_server}"]
        dns_domain      = "test.internal"
      }

      ipv4_gateway = "${var.ipv4_gateway}"
    }
  }
}
`,
		os.Getenv("VSPHERE_DATACENTER"),
		os.Getenv("VSPHERE_RESOURCE_POOL"),
		os.Getenv("VSPHERE_NETWORK_LABEL"),
		os.Getenv("VSPHERE_IPV4_ADDRESS"),
		os.Getenv("VSPHERE_IPV4_PREFIX"),
		os.Getenv("VSPHERE_IPV4_GATEWAY"),
		os.Getenv("VSPHERE_DNS"),
		os.Getenv("VSPHERE_DATASTORE"),
		os.Getenv("VSPHERE_TEMPLATE_WINDOWS"),
		os.Getenv("VSPHERE_USE_LINKED_CLONE"),
	)
}

func testAccResourceVSphereVirtualMachineConfigWithCustomAttribute() string {
	return fmt.Sprintf(`
variable "datacenter" {
  default = "%s"
}

variable "resource_pool" {
  default = "%s"
}

variable "network_label" {
  default = "%s"
}

variable "ipv4_address" {
  default = "%s"
}

variable "ipv4_netmask" {
  default = "%s"
}

variable "ipv4_gateway" {
  default = "%s"
}

variable "dns_server" {
  default = "%s"
}

variable "datastore" {
  default = "%s"
}

variable "template" {
  default = "%s"
}

variable "linked_clone" {
  default = "%s"
}

data "vsphere_datacenter" "dc" {
  name = "${var.datacenter}"
}

data "vsphere_datastore" "datastore" {
  name          = "${var.datastore}"
  datacenter_id = "${data.vsphere_datacenter.dc.id}"
}

data "vsphere_resource_pool" "pool" {
  name          = "${var.resource_pool}"
  datacenter_id = "${data.vsphere_datacenter.dc.id}"
}

data "vsphere_network" "network" {
  name          = "${var.network_label}"
  datacenter_id = "${data.vsphere_datacenter.dc.id}"
}

data "vsphere_virtual_machine" "template" {
  name          = "${var.template}"
  datacenter_id = "${data.vsphere_datacenter.dc.id}"
}

resource "vsphere_custom_attribute" "terraform-test-attribute" {
  name                = "terraform-test-attribute"
  managed_object_type = "VirtualMachine"
}

locals {
  vm_attrs = {
    "${vsphere_custom_attribute.terraform-test-attribute.id}" = "value"
  }
}

resource "vsphere_virtual_machine" "vm" {
  name             = "terraform-test"
  resource_pool_id = "${data.vsphere_resource_pool.pool.id}"
  datastore_id     = "${data.vsphere_datastore.datastore.id}"
  num_cpus         = 2
  memory           = 2048
  guest_id         = "${data.vsphere_virtual_machine.template.guest_id}"
  scsi_type        = "${data.vsphere_virtual_machine.template.scsi_type}"

  network_interface {
    network_id   = "${data.vsphere_network.network.id}"
    adapter_type = "${data.vsphere_virtual_machine.template.network_interface_types[0]}"
  }

  disk {
    label = "disk0"
    size  = "${data.vsphere_virtual_machine.template.disks.0.size}"
  }

  clone {
    template_uuid = "${data.vsphere_virtual_machine.template.id}"
    linked_clone  = "${var.linked_clone != "" ? "true" : "false" }"

    customize {
      linux_options {
        host_name = "terraform-test"
        domain    = "test.internal"
      }

      network_interface {
        ipv4_address = "${var.ipv4_address}"
        ipv4_netmask = "${var.ipv4_netmask}"
      }

      ipv4_gateway    = "${var.ipv4_gateway}"
      dns_server_list = ["${var.dns_server}"]
      dns_suffix_list = ["test.internal"]
    }
  }

  custom_attributes = "${local.vm_attrs}"
}
`,
		os.Getenv("VSPHERE_DATACENTER"),
		os.Getenv("VSPHERE_RESOURCE_POOL"),
		os.Getenv("VSPHERE_NETWORK_LABEL"),
		os.Getenv("VSPHERE_IPV4_ADDRESS"),
		os.Getenv("VSPHERE_IPV4_PREFIX"),
		os.Getenv("VSPHERE_IPV4_GATEWAY"),
		os.Getenv("VSPHERE_DNS"),
		os.Getenv("VSPHERE_DATASTORE"),
		os.Getenv("VSPHERE_TEMPLATE"),
		os.Getenv("VSPHERE_USE_LINKED_CLONE"),
	)
}

func testAccResourceVSphereVirtualMachineConfigWithMultiCustomAttribute() string {
	return fmt.Sprintf(`
variable "datacenter" {
  default = "%s"
}

variable "resource_pool" {
  default = "%s"
}

variable "network_label" {
  default = "%s"
}

variable "ipv4_address" {
  default = "%s"
}

variable "ipv4_netmask" {
  default = "%s"
}

variable "ipv4_gateway" {
  default = "%s"
}

variable "dns_server" {
  default = "%s"
}

variable "datastore" {
  default = "%s"
}

variable "template" {
  default = "%s"
}

variable "linked_clone" {
  default = "%s"
}

variable "custom_attrs" {
  default = [
    "terraform-test-attribute-1",
    "terraform-test-attriubute-2",
  ]
}

data "vsphere_datacenter" "dc" {
  name = "${var.datacenter}"
}

data "vsphere_datastore" "datastore" {
  name          = "${var.datastore}"
  datacenter_id = "${data.vsphere_datacenter.dc.id}"
}

data "vsphere_resource_pool" "pool" {
  name          = "${var.resource_pool}"
  datacenter_id = "${data.vsphere_datacenter.dc.id}"
}

data "vsphere_network" "network" {
  name          = "${var.network_label}"
  datacenter_id = "${data.vsphere_datacenter.dc.id}"
}

data "vsphere_virtual_machine" "template" {
  name          = "${var.template}"
  datacenter_id = "${data.vsphere_datacenter.dc.id}"
}

resource "vsphere_custom_attribute" "terraform-test-attribute" {
  name                = "terraform-test-attribute"
  managed_object_type = "VirtualMachine"
}

resource "vsphere_custom_attribute" "terraform-test-attribute-alt" {
  count               = "${length(var.custom_attrs)}"
  name                = "${var.custom_attrs[count.index]}"
  managed_object_type = "VirtualMachine"
}

locals {
  vm_attrs = {
    "${vsphere_custom_attribute.terraform-test-attribute-alt.0.id}" = "value"
    "${vsphere_custom_attribute.terraform-test-attribute-alt.1.id}" = "value-2"
  }
}

resource "vsphere_virtual_machine" "vm" {
  name             = "terraform-test"
  resource_pool_id = "${data.vsphere_resource_pool.pool.id}"
  datastore_id     = "${data.vsphere_datastore.datastore.id}"
  num_cpus         = 2
  memory           = 2048
  guest_id         = "${data.vsphere_virtual_machine.template.guest_id}"
  scsi_type        = "${data.vsphere_virtual_machine.template.scsi_type}"

  network_interface {
    network_id   = "${data.vsphere_network.network.id}"
    adapter_type = "${data.vsphere_virtual_machine.template.network_interface_types[0]}"
  }

  disk {
    label = "disk0"
    size  = "${data.vsphere_virtual_machine.template.disks.0.size}"
  }

  clone {
    template_uuid = "${data.vsphere_virtual_machine.template.id}"
    linked_clone  = "${var.linked_clone != "" ? "true" : "false" }"

    customize {
      linux_options {
        host_name = "terraform-test"
        domain    = "test.internal"
      }

      network_interface {
        ipv4_address = "${var.ipv4_address}"
        ipv4_netmask = "${var.ipv4_netmask}"
      }

      ipv4_gateway    = "${var.ipv4_gateway}"
      dns_server_list = ["${var.dns_server}"]
      dns_suffix_list = ["test.internal"]
    }
  }

  custom_attributes = "${local.vm_attrs}"
}
`,
		os.Getenv("VSPHERE_DATACENTER"),
		os.Getenv("VSPHERE_RESOURCE_POOL"),
		os.Getenv("VSPHERE_NETWORK_LABEL"),
		os.Getenv("VSPHERE_IPV4_ADDRESS"),
		os.Getenv("VSPHERE_IPV4_PREFIX"),
		os.Getenv("VSPHERE_IPV4_GATEWAY"),
		os.Getenv("VSPHERE_DNS"),
		os.Getenv("VSPHERE_DATASTORE"),
		os.Getenv("VSPHERE_TEMPLATE"),
		os.Getenv("VSPHERE_USE_LINKED_CLONE"),
	)
}

func testAccResourceVSphereVirtualMachineConfigCloneWithVAppProperties() string {
	return fmt.Sprintf(`
variable "datacenter" {
  default = "%s"
}

variable "resource_pool" {
  default = "%s"
}

variable "network_label" {
  default = "%s"
}

variable "ipv4_address" {
  default = "%s"
}

variable "ipv4_netmask" {
  default = "%s"
}

variable "ipv4_gateway" {
  default = "%s"
}

variable "dns_server" {
  default = "%s"
}

variable "datastore" {
  default = "%s"
}

variable "template" {
  default = "%s"
}

variable "linked_clone" {
  default = "%s"
}

data "vsphere_datacenter" "dc" {
  name = "${var.datacenter}"
}

data "vsphere_datastore" "datastore" {
  name          = "${var.datastore}"
  datacenter_id = "${data.vsphere_datacenter.dc.id}"
}

data "vsphere_resource_pool" "pool" {
  name          = "${var.resource_pool}"
  datacenter_id = "${data.vsphere_datacenter.dc.id}"
}

data "vsphere_network" "network" {
  name          = "${var.network_label}"
  datacenter_id = "${data.vsphere_datacenter.dc.id}"
}

data "vsphere_virtual_machine" "template" {
  name          = "${var.template}"
  datacenter_id = "${data.vsphere_datacenter.dc.id}"
}

resource "vsphere_virtual_machine" "vm" {
  name             = "terraform-test"
  resource_pool_id = "${data.vsphere_resource_pool.pool.id}"
  datastore_id     = "${data.vsphere_datastore.datastore.id}"
  num_cpus         = 2
  memory           = 2048
  guest_id         = "${data.vsphere_virtual_machine.template.guest_id}"

  network_interface {
    network_id   = "${data.vsphere_network.network.id}"
    adapter_type = "${data.vsphere_virtual_machine.template.network_interface_types[0]}"
  }

  disk {
    label = "disk0"
    size  = "${data.vsphere_virtual_machine.template.disks.0.size}"
  }

  vapp {
    properties {
      "guestinfo.hostname"                        = "terraform-test.test.internal"
      "guestinfo.interface.0.name"                = "ens192"
      "guestinfo.interface.0.ip.0.address"        = "${var.ipv4_address}/${var.ipv4_netmask}"
      "guestinfo.interface.0.route.0.gateway"     = "${var.ipv4_gateway}"
      "guestinfo.interface.0.route.0.destination" = "0.0.0.0/0"
      "guestinfo.dns.server.0"                    = "${var.dns_server}"
    }
  }

  clone {
    template_uuid = "${data.vsphere_virtual_machine.template.id}"
    linked_clone  = "${var.linked_clone}"
  }
}
`,

		os.Getenv("VSPHERE_DATACENTER"),
		os.Getenv("VSPHERE_RESOURCE_POOL"),
		os.Getenv("VSPHERE_NETWORK_LABEL"),
		os.Getenv("VSPHERE_IPV4_ADDRESS"),
		os.Getenv("VSPHERE_IPV4_PREFIX"),
		os.Getenv("VSPHERE_IPV4_GATEWAY"),
		os.Getenv("VSPHERE_DNS"),
		os.Getenv("VSPHERE_DATASTORE"),
		os.Getenv("VSPHERE_TEMPLATE_COREOS"),
		os.Getenv("VSPHERE_USE_LINKED_CLONE"),
	)
}

func testAccResourceVSphereVirtualMachineConfigCloneUpdatingVAppProperties() string {
	return fmt.Sprintf(`
variable "datacenter" {
  default = "%s"
}

variable "resource_pool" {
  default = "%s"
}

variable "network_label" {
  default = "%s"
}

variable "ipv4_address" {
  default = "%s"
}

variable "ipv4_netmask" {
  default = "%s"
}

variable "ipv4_gateway" {
  default = "%s"
}

variable "dns_server" {
  default = "%s"
}

variable "datastore" {
  default = "%s"
}

variable "template" {
  default = "%s"
}

variable "linked_clone" {
  default = "%s"
}

data "vsphere_datacenter" "dc" {
  name = "${var.datacenter}"
}

data "vsphere_datastore" "datastore" {
  name          = "${var.datastore}"
  datacenter_id = "${data.vsphere_datacenter.dc.id}"
}

data "vsphere_resource_pool" "pool" {
  name          = "${var.resource_pool}"
  datacenter_id = "${data.vsphere_datacenter.dc.id}"
}

data "vsphere_network" "network" {
  name          = "${var.network_label}"
  datacenter_id = "${data.vsphere_datacenter.dc.id}"
}

data "vsphere_virtual_machine" "template" {
  name          = "${var.template}"
  datacenter_id = "${data.vsphere_datacenter.dc.id}"
}

resource "vsphere_virtual_machine" "vm" {
  name             = "terraform-test"
  resource_pool_id = "${data.vsphere_resource_pool.pool.id}"
  datastore_id     = "${data.vsphere_datastore.datastore.id}"
  num_cpus         = 2
  memory           = 2048
  guest_id         = "${data.vsphere_virtual_machine.template.guest_id}"

  network_interface {
    network_id   = "${data.vsphere_network.network.id}"
    adapter_type = "${data.vsphere_virtual_machine.template.network_interface_types[0]}"
  }

  disk {
    label = "disk0"
    size  = "${data.vsphere_virtual_machine.template.disks.0.size}"
  }

  vapp {
    properties {
      "guestinfo.hostname"                        = "terraform-test.test.internal"
      "guestinfo.interface.0.name"                = "ens192"
      "guestinfo.interface.0.ip.0.address"        = "${var.ipv4_address}/${var.ipv4_netmask}"
      "guestinfo.interface.0.route.0.gateway"     = "${var.ipv4_gateway}"
      "guestinfo.interface.0.route.0.destination" = "0.0.0.0/0"
      "guestinfo.dns.server.0"                    = "${var.dns_server}"
      "guestinfo.dns.server.1"                    = "8.8.8.8"
    }
  }

  clone {
    template_uuid = "${data.vsphere_virtual_machine.template.id}"
    linked_clone  = "${var.linked_clone}"
  }
}
`,

		os.Getenv("VSPHERE_DATACENTER"),
		os.Getenv("VSPHERE_RESOURCE_POOL"),
		os.Getenv("VSPHERE_NETWORK_LABEL"),
		os.Getenv("VSPHERE_IPV4_ADDRESS"),
		os.Getenv("VSPHERE_IPV4_PREFIX"),
		os.Getenv("VSPHERE_IPV4_GATEWAY"),
		os.Getenv("VSPHERE_DNS"),
		os.Getenv("VSPHERE_DATASTORE"),
		os.Getenv("VSPHERE_TEMPLATE_COREOS"),
		os.Getenv("VSPHERE_USE_LINKED_CLONE"),
	)
}

func testAccResourceVSphereVirtualMachineConfigCloneVAppPropertiesBadKey() string {
	return fmt.Sprintf(`
variable "datacenter" {
  default = "%s"
}

variable "resource_pool" {
  default = "%s"
}

variable "network_label" {
  default = "%s"
}

variable "ipv4_address" {
  default = "%s"
}

variable "ipv4_netmask" {
  default = "%s"
}

variable "ipv4_gateway" {
  default = "%s"
}

variable "dns_server" {
  default = "%s"
}

variable "datastore" {
  default = "%s"
}

variable "template" {
  default = "%s"
}

variable "linked_clone" {
  default = "%s"
}

data "vsphere_datacenter" "dc" {
  name = "${var.datacenter}"
}

data "vsphere_datastore" "datastore" {
  name          = "${var.datastore}"
  datacenter_id = "${data.vsphere_datacenter.dc.id}"
}

data "vsphere_resource_pool" "pool" {
  name          = "${var.resource_pool}"
  datacenter_id = "${data.vsphere_datacenter.dc.id}"
}

data "vsphere_network" "network" {
  name          = "${var.network_label}"
  datacenter_id = "${data.vsphere_datacenter.dc.id}"
}

data "vsphere_virtual_machine" "template" {
  name          = "${var.template}"
  datacenter_id = "${data.vsphere_datacenter.dc.id}"
}

resource "vsphere_virtual_machine" "vm" {
  name             = "terraform-test"
  resource_pool_id = "${data.vsphere_resource_pool.pool.id}"
  datastore_id     = "${data.vsphere_datastore.datastore.id}"
  num_cpus         = 2
  memory           = 2048
  guest_id         = "${data.vsphere_virtual_machine.template.guest_id}"

  network_interface {
    network_id   = "${data.vsphere_network.network.id}"
    adapter_type = "${data.vsphere_virtual_machine.template.network_interface_types[0]}"
  }

  disk {
    label = "disk0"
    size  = "${data.vsphere_virtual_machine.template.disks.0.size}"
  }

  vapp {
    properties {
      "guestinfo.hostname"                        = "terraform-test.test.internal"
      "guestinfo.interface.0.name"                = "ens192"
      "guestinfo.interface.0.ip.0.address"        = "${var.ipv4_address}/${var.ipv4_netmask}"
      "guestinfo.interface.0.route.0.gateway"     = "${var.ipv4_gateway}"
      "guestinfo.interface.0.route.0.destination" = "0.0.0.0/0"
      "guestinfo.dns.server.0"                    = "${var.dns_server}"
      "foo"                                       = "bar"
    }
  }

  clone {
    template_uuid = "${data.vsphere_virtual_machine.template.id}"
    linked_clone  = "${var.linked_clone}"
  }
}
`,

		os.Getenv("VSPHERE_DATACENTER"),
		os.Getenv("VSPHERE_RESOURCE_POOL"),
		os.Getenv("VSPHERE_NETWORK_LABEL"),
		os.Getenv("VSPHERE_IPV4_ADDRESS"),
		os.Getenv("VSPHERE_IPV4_PREFIX"),
		os.Getenv("VSPHERE_IPV4_GATEWAY"),
		os.Getenv("VSPHERE_DNS"),
		os.Getenv("VSPHERE_DATASTORE"),
		os.Getenv("VSPHERE_TEMPLATE_COREOS"),
		os.Getenv("VSPHERE_USE_LINKED_CLONE"),
	)
}

func testAccResourceVSphereVirtualMachineConfigCloneBadVAppSettings() string {
	return fmt.Sprintf(`
variable "datacenter" {
  default = "%s"
}

variable "resource_pool" {
  default = "%s"
}

variable "network_label" {
  default = "%s"
}

variable "ipv4_address" {
  default = "%s"
}

variable "ipv4_netmask" {
  default = "%s"
}

variable "ipv4_gateway" {
  default = "%s"
}

variable "dns_server" {
  default = "%s"
}

variable "datastore" {
  default = "%s"
}

variable "template" {
  default = "%s"
}

variable "linked_clone" {
  default = "%s"
}

data "vsphere_datacenter" "dc" {
  name = "${var.datacenter}"
}

data "vsphere_datastore" "datastore" {
  name          = "${var.datastore}"
  datacenter_id = "${data.vsphere_datacenter.dc.id}"
}

data "vsphere_resource_pool" "pool" {
  name          = "${var.resource_pool}"
  datacenter_id = "${data.vsphere_datacenter.dc.id}"
}

data "vsphere_network" "network" {
  name          = "${var.network_label}"
  datacenter_id = "${data.vsphere_datacenter.dc.id}"
}

data "vsphere_virtual_machine" "template" {
  name          = "${var.template}"
  datacenter_id = "${data.vsphere_datacenter.dc.id}"
}

resource "vsphere_virtual_machine" "vm" {
  name             = "terraform-test"
  resource_pool_id = "${data.vsphere_resource_pool.pool.id}"
  datastore_id     = "${data.vsphere_datastore.datastore.id}"

  num_cpus = 2
  memory   = 2048
  guest_id = "${data.vsphere_virtual_machine.template.guest_id}"

  network_interface {
    network_id   = "${data.vsphere_network.network.id}"
    adapter_type = "${data.vsphere_virtual_machine.template.network_interface_types[0]}"
  }

  disk {
    label            = "disk0"
    size             = "${data.vsphere_virtual_machine.template.disks.0.size}"
    eagerly_scrub    = "${data.vsphere_virtual_machine.template.disks.0.eagerly_scrub}"
    thin_provisioned = "${data.vsphere_virtual_machine.template.disks.0.thin_provisioned}"
  }

  clone {
    template_uuid = "${data.vsphere_virtual_machine.template.id}"
    linked_clone  = "${var.linked_clone != "" ? "true" : "false" }"

    customize {
      linux_options {
        host_name = "terraform-test"
        domain    = "test.internal"
      }

      network_interface {
        ipv4_address = "${var.ipv4_address}"
        ipv4_netmask = "${var.ipv4_netmask}"
      }

      ipv4_gateway    = "${var.ipv4_gateway}"
      dns_server_list = ["${var.dns_server}"]
      dns_suffix_list = ["test.internal"]
    }
  }

  vapp {
    properties {
      "foo" = "bar"
    }
  }
}
`,
		os.Getenv("VSPHERE_DATACENTER"),
		os.Getenv("VSPHERE_RESOURCE_POOL"),
		os.Getenv("VSPHERE_NETWORK_LABEL"),
		os.Getenv("VSPHERE_IPV4_ADDRESS"),
		os.Getenv("VSPHERE_IPV4_PREFIX"),
		os.Getenv("VSPHERE_IPV4_GATEWAY"),
		os.Getenv("VSPHERE_DNS"),
		os.Getenv("VSPHERE_DATASTORE"),
		os.Getenv("VSPHERE_TEMPLATE"),
		os.Getenv("VSPHERE_USE_LINKED_CLONE"),
	)
}

// TODO: Remove this fixture in 2.0
func testAccResourceVSphereVirtualMachineConfigBasicDiskNameOrLabel(nameKey string) string {
	return fmt.Sprintf(`
variable "datacenter" {
  default = "%s"
}

variable "resource_pool" {
  default = "%s"
}

variable "network_label" {
  default = "%s"
}

variable "datastore" {
  default = "%s"
}

data "vsphere_datacenter" "dc" {
  name = "${var.datacenter}"
}

data "vsphere_datastore" "datastore" {
  name          = "${var.datastore}"
  datacenter_id = "${data.vsphere_datacenter.dc.id}"
}

data "vsphere_resource_pool" "pool" {
  name          = "${var.resource_pool}"
  datacenter_id = "${data.vsphere_datacenter.dc.id}"
}

data "vsphere_network" "network" {
  name          = "${var.network_label}"
  datacenter_id = "${data.vsphere_datacenter.dc.id}"
}

resource "vsphere_virtual_machine" "vm" {
  name             = "terraform-test"
  resource_pool_id = "${data.vsphere_resource_pool.pool.id}"
  datastore_id     = "${data.vsphere_datastore.datastore.id}"

  num_cpus = 2
  memory   = 2048
  guest_id = "other3xLinux64Guest"

  network_interface {
    network_id = "${data.vsphere_network.network.id}"
  }

  disk {
    %s    = "terraform-test.vmdk"
    size  = 20
  }
}
`,
		os.Getenv("VSPHERE_DATACENTER"),
		os.Getenv("VSPHERE_RESOURCE_POOL"),
		os.Getenv("VSPHERE_NETWORK_LABEL_PXE"),
		os.Getenv("VSPHERE_DATASTORE"),
		nameKey,
	)
}

// TODO: Remove this fixture in 2.0
func testAccResourceVSphereVirtualMachineConfigExistingVmdkWithName() string {
	return fmt.Sprintf(`
variable "datacenter" {
  default = "%s"
}

variable "resource_pool" {
  default = "%s"
}

variable "network_label" {
  default = "%s"
}

variable "datastore" {
  default = "%s"
}

variable "extra_vmdk_name" {
  default = "%s"
}

data "vsphere_datacenter" "dc" {
  name = "${var.datacenter}"
}

data "vsphere_datastore" "datastore" {
  name          = "${var.datastore}"
  datacenter_id = "${data.vsphere_datacenter.dc.id}"
}

data "vsphere_resource_pool" "pool" {
  name          = "${var.resource_pool}"
  datacenter_id = "${data.vsphere_datacenter.dc.id}"
}

data "vsphere_network" "network" {
  name          = "${var.network_label}"
  datacenter_id = "${data.vsphere_datacenter.dc.id}"
}

resource "vsphere_virtual_disk" "disk" {
  size         = 1
  vmdk_path    = "${var.extra_vmdk_name}"
  datacenter   = "${var.datacenter}"
  datastore    = "${var.datastore}"
  type         = "thin"
  adapter_type = "lsiLogic"
}

resource "vsphere_virtual_machine" "vm" {
  name             = "terraform-test"
  resource_pool_id = "${data.vsphere_resource_pool.pool.id}"
  datastore_id     = "${data.vsphere_datastore.datastore.id}"

  num_cpus = 2
  memory   = 2048
  guest_id = "other3xLinux64Guest"

  network_interface {
    network_id = "${data.vsphere_network.network.id}"
  }

  disk {
    name  = "terraform-test.vmdk"
    size  = 20
  }

  disk {
    name         = "${vsphere_virtual_disk.disk.vmdk_path}"
    datastore_id = "${data.vsphere_datastore.datastore.id}"
    disk_mode    = "independent_persistent"
    attach       = true
    unit_number  = 1
  }
}
`,
		os.Getenv("VSPHERE_DATACENTER"),
		os.Getenv("VSPHERE_RESOURCE_POOL"),
		os.Getenv("VSPHERE_NETWORK_LABEL_PXE"),
		os.Getenv("VSPHERE_DATASTORE"),
		testAccResourceVSphereVirtualMachineDiskNameExtraVmdk,
	)
}

// TODO: Remove this fixture in 2.0
func testAccResourceVSphereVirtualMachineConfigExistingVmdkWithLabel() string {
	return fmt.Sprintf(`
variable "datacenter" {
  default = "%s"
}

variable "resource_pool" {
  default = "%s"
}

variable "network_label" {
  default = "%s"
}

variable "datastore" {
  default = "%s"
}

variable "extra_vmdk_name" {
  default = "%s"
}

data "vsphere_datacenter" "dc" {
  name = "${var.datacenter}"
}

data "vsphere_datastore" "datastore" {
  name          = "${var.datastore}"
  datacenter_id = "${data.vsphere_datacenter.dc.id}"
}

data "vsphere_resource_pool" "pool" {
  name          = "${var.resource_pool}"
  datacenter_id = "${data.vsphere_datacenter.dc.id}"
}

data "vsphere_network" "network" {
  name          = "${var.network_label}"
  datacenter_id = "${data.vsphere_datacenter.dc.id}"
}

resource "vsphere_virtual_disk" "disk" {
  size         = 1
  vmdk_path    = "${var.extra_vmdk_name}"
  datacenter   = "${var.datacenter}"
  datastore    = "${var.datastore}"
  type         = "thin"
  adapter_type = "lsiLogic"
}

resource "vsphere_virtual_machine" "vm" {
  name             = "terraform-test"
  resource_pool_id = "${data.vsphere_resource_pool.pool.id}"
  datastore_id     = "${data.vsphere_datastore.datastore.id}"

  num_cpus = 2
  memory   = 2048
  guest_id = "other3xLinux64Guest"

  network_interface {
    network_id = "${data.vsphere_network.network.id}"
  }

  disk {
    label = "terraform-test.vmdk"
    size  = 20
  }

  disk {
    label        = "${vsphere_virtual_disk.disk.vmdk_path}"
    path         = "${vsphere_virtual_disk.disk.vmdk_path}"
    datastore_id = "${data.vsphere_datastore.datastore.id}"
    disk_mode    = "independent_persistent"
    attach       = true
    unit_number  = 1
  }
}
`,
		os.Getenv("VSPHERE_DATACENTER"),
		os.Getenv("VSPHERE_RESOURCE_POOL"),
		os.Getenv("VSPHERE_NETWORK_LABEL_PXE"),
		os.Getenv("VSPHERE_DATASTORE"),
		testAccResourceVSphereVirtualMachineDiskNameExtraVmdk,
	)
}

func testAccResourceVSphereVirtualMachineConfigCloneChangeDiskAndSCSI() string {
	return fmt.Sprintf(`
variable "datacenter" {
  default = "%s"
}

variable "resource_pool" {
  default = "%s"
}

variable "network_label" {
  default = "%s"
}

variable "ipv4_address" {
  default = "%s"
}

variable "ipv4_netmask" {
  default = "%s"
}

variable "ipv4_gateway" {
  default = "%s"
}

variable "dns_server" {
  default = "%s"
}

variable "datastore" {
  default = "%s"
}

variable "template" {
  default = "%s"
}

data "vsphere_datacenter" "dc" {
  name = "${var.datacenter}"
}

data "vsphere_datastore" "datastore" {
  name          = "${var.datastore}"
  datacenter_id = "${data.vsphere_datacenter.dc.id}"
}

data "vsphere_resource_pool" "pool" {
  name          = "${var.resource_pool}"
  datacenter_id = "${data.vsphere_datacenter.dc.id}"
}

data "vsphere_network" "network" {
  name          = "${var.network_label}"
  datacenter_id = "${data.vsphere_datacenter.dc.id}"
}

data "vsphere_virtual_machine" "template" {
  name          = "${var.template}"
  datacenter_id = "${data.vsphere_datacenter.dc.id}"
}

resource "vsphere_virtual_machine" "vm" {
  name             = "terraform-test"
  resource_pool_id = "${data.vsphere_resource_pool.pool.id}"
  datastore_id     = "${data.vsphere_datastore.datastore.id}"

  num_cpus = 2
  memory   = 2048
  guest_id = "${data.vsphere_virtual_machine.template.guest_id}"

  scsi_type = "${data.vsphere_virtual_machine.template.scsi_type == "pvscsi" ? "lsilogic-sas" : "pvscsi"}"

  network_interface {
    network_id   = "${data.vsphere_network.network.id}"
    adapter_type = "${data.vsphere_virtual_machine.template.network_interface_types[0]}"
  }

  disk {
    label = "disk0"
    size  = "${data.vsphere_virtual_machine.template.disks.0.size * 2}"
  }

  clone {
    template_uuid = "${data.vsphere_virtual_machine.template.id}"

    customize {
      linux_options {
        host_name = "terraform-test"
        domain    = "test.internal"
      }

      network_interface {
        ipv4_address = "${var.ipv4_address}"
        ipv4_netmask = "${var.ipv4_netmask}"
      }

      ipv4_gateway    = "${var.ipv4_gateway}"
      dns_server_list = ["${var.dns_server}"]
      dns_suffix_list = ["test.internal"]
    }
  }
}
`,
		os.Getenv("VSPHERE_DATACENTER"),
		os.Getenv("VSPHERE_RESOURCE_POOL"),
		os.Getenv("VSPHERE_NETWORK_LABEL"),
		os.Getenv("VSPHERE_IPV4_ADDRESS"),
		os.Getenv("VSPHERE_IPV4_PREFIX"),
		os.Getenv("VSPHERE_IPV4_GATEWAY"),
		os.Getenv("VSPHERE_DNS"),
		os.Getenv("VSPHERE_DATASTORE"),
		os.Getenv("VSPHERE_TEMPLATE"),
	)
}

func testAccResourceVSphereVirtualMachineConfigBasicEmptyCluster() string {
	return fmt.Sprintf(`
variable "datacenter" {
  default = "%s"
}

variable "resource_pool" {
  default = "%s/Resources"
}

variable "network_label" {
  default = "%s"
}

variable "datastore" {
  default = "%s"
}

data "vsphere_datacenter" "dc" {
  name = "${var.datacenter}"
}

data "vsphere_datastore" "datastore" {
  name          = "${var.datastore}"
  datacenter_id = "${data.vsphere_datacenter.dc.id}"
}

data "vsphere_resource_pool" "pool" {
  name          = "${var.resource_pool}"
  datacenter_id = "${data.vsphere_datacenter.dc.id}"
}

data "vsphere_network" "network" {
  name          = "${var.network_label}"
  datacenter_id = "${data.vsphere_datacenter.dc.id}"
}

resource "vsphere_virtual_machine" "vm" {
  name             = "terraform-test"
  resource_pool_id = "${data.vsphere_resource_pool.pool.id}"
  datastore_id     = "${data.vsphere_datastore.datastore.id}"

  num_cpus = 2
  memory   = 2048
  guest_id = "other3xLinux64Guest"

  network_interface {
    network_id = "${data.vsphere_network.network.id}"
  }

  disk {
    label = "disk0"
    size  = 20
  }
}
`,
		os.Getenv("VSPHERE_DATACENTER"),
		os.Getenv("VSPHERE_EMPTY_CLUSTER"),
		os.Getenv("VSPHERE_NETWORK_LABEL_PXE"),
		os.Getenv("VSPHERE_DATASTORE"),
	)
}

func testAccResourceVSphereVirtualMachineConfigCloneEmptyCluster() string {
	return fmt.Sprintf(`
variable "datacenter" {
  default = "%s"
}

variable "resource_pool" {
  default = "%s/Resources"
}

variable "network_label" {
  default = "%s"
}

variable "ipv4_address" {
  default = "%s"
}

variable "ipv4_netmask" {
  default = "%s"
}

variable "ipv4_gateway" {
  default = "%s"
}

variable "dns_server" {
  default = "%s"
}

variable "datastore" {
  default = "%s"
}

variable "template" {
  default = "%s"
}

variable "linked_clone" {
  default = "%s"
}

data "vsphere_datacenter" "dc" {
  name = "${var.datacenter}"
}

data "vsphere_datastore" "datastore" {
  name          = "${var.datastore}"
  datacenter_id = "${data.vsphere_datacenter.dc.id}"
}

data "vsphere_resource_pool" "pool" {
  name          = "${var.resource_pool}"
  datacenter_id = "${data.vsphere_datacenter.dc.id}"
}

data "vsphere_network" "network" {
  name          = "${var.network_label}"
  datacenter_id = "${data.vsphere_datacenter.dc.id}"
}

data "vsphere_virtual_machine" "template" {
  name          = "${var.template}"
  datacenter_id = "${data.vsphere_datacenter.dc.id}"
}

resource "vsphere_virtual_machine" "vm" {
  name             = "terraform-test"
  resource_pool_id = "${data.vsphere_resource_pool.pool.id}"
  datastore_id     = "${data.vsphere_datastore.datastore.id}"

  num_cpus = 2
  memory   = 2048
  guest_id = "${data.vsphere_virtual_machine.template.guest_id}"

  network_interface {
    network_id   = "${data.vsphere_network.network.id}"
    adapter_type = "${data.vsphere_virtual_machine.template.network_interface_types[0]}"
  }

  disk {
    label            = "disk0"
    size             = "${data.vsphere_virtual_machine.template.disks.0.size}"
    eagerly_scrub    = "${data.vsphere_virtual_machine.template.disks.0.eagerly_scrub}"
    thin_provisioned = "${data.vsphere_virtual_machine.template.disks.0.thin_provisioned}"
  }

  clone {
    template_uuid = "${data.vsphere_virtual_machine.template.id}"
    linked_clone  = "${var.linked_clone != "" ? "true" : "false" }"

    customize {
      linux_options {
        host_name = "terraform-test"
        domain    = "test.internal"
      }

      network_interface {
        ipv4_address = "${var.ipv4_address}"
        ipv4_netmask = "${var.ipv4_netmask}"
      }

      ipv4_gateway    = "${var.ipv4_gateway}"
      dns_server_list = ["${var.dns_server}"]
      dns_suffix_list = ["test.internal"]
    }
  }
}
`,
		os.Getenv("VSPHERE_DATACENTER"),
		os.Getenv("VSPHERE_EMPTY_CLUSTER"),
		os.Getenv("VSPHERE_NETWORK_LABEL"),
		os.Getenv("VSPHERE_IPV4_ADDRESS"),
		os.Getenv("VSPHERE_IPV4_PREFIX"),
		os.Getenv("VSPHERE_IPV4_GATEWAY"),
		os.Getenv("VSPHERE_DNS"),
		os.Getenv("VSPHERE_DATASTORE"),
		os.Getenv("VSPHERE_TEMPLATE"),
		os.Getenv("VSPHERE_USE_LINKED_CLONE"),
	)
}
