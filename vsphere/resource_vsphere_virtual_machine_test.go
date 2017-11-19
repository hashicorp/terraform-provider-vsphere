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
	"github.com/terraform-providers/terraform-provider-vsphere/vsphere/internal/helper/folder"
	"github.com/terraform-providers/terraform-provider-vsphere/vsphere/internal/helper/structure"
	"github.com/terraform-providers/terraform-provider-vsphere/vsphere/internal/virtualdevice"
	"github.com/vmware/govmomi/object"
	"github.com/vmware/govmomi/vim25/types"
)

const (
	testAccResourceVSphereVirtualMachineDiskNameEager     = "terraform-test-extra-eager.vmdk"
	testAccResourceVSphereVirtualMachineDiskNameLazy      = "terraform-test-extra-lazy.vmdk"
	testAccResourceVSphereVirtualMachineDiskNameThin      = "terraform-test-extra-thin.vmdk"
	testAccResourceVSphereVirtualMachineDiskNameExtraVmdk = "terraform-test-vm-extra-disk.vmdk"
	testAccResourceVSphereVirtualMachineStaticMacAddr     = "06:5c:89:2b:a0:64"
	testAccResourceVSphereVirtualMachineAnnotation        = "Managed by Terraform"
)

func TestAccResourceVSphereVirtualMachine(t *testing.T) {
	var tp *testing.T
	var state *terraform.State
	testAccResourceVSphereVirtualMachineCases := []struct {
		name     string
		testCase resource.TestCase
	}{
		{
			"basic",
			resource.TestCase{
				PreCheck: func() {
					testAccPreCheck(tp)
					testAccResourceVSphereVirtualMachinePreCheck(tp)
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
				},
			},
		},
		{
			"shutdown OK",
			resource.TestCase{
				PreCheck: func() {
					testAccPreCheck(tp)
					testAccResourceVSphereVirtualMachinePreCheck(tp)
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
			},
		},
		{
			"multi-device",
			resource.TestCase{
				PreCheck: func() {
					testAccPreCheck(tp)
					testAccResourceVSphereVirtualMachinePreCheck(tp)
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
			},
		},
		{
			"add devices",
			resource.TestCase{
				PreCheck: func() {
					testAccPreCheck(tp)
					testAccResourceVSphereVirtualMachinePreCheck(tp)
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
			},
		},
		{
			"remove middle devices",
			resource.TestCase{
				PreCheck: func() {
					testAccPreCheck(tp)
					testAccResourceVSphereVirtualMachinePreCheck(tp)
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
			},
		},
		{
			"remove middle devices, change disk unit",
			resource.TestCase{
				PreCheck: func() {
					testAccPreCheck(tp)
					testAccResourceVSphereVirtualMachinePreCheck(tp)
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
			},
		},
		{
			"cdrom",
			resource.TestCase{
				PreCheck: func() {
					testAccPreCheck(tp)
					testAccResourceVSphereVirtualMachinePreCheck(tp)
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
			},
		},
		{
			"upgrade cpu and ram",
			resource.TestCase{
				PreCheck: func() {
					testAccPreCheck(tp)
					testAccResourceVSphereVirtualMachinePreCheck(tp)
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
			},
		},
		{
			"modify annotation",
			resource.TestCase{
				PreCheck: func() {
					testAccPreCheck(tp)
					testAccResourceVSphereVirtualMachinePreCheck(tp)
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
			},
		},
		{
			"grow disk",
			resource.TestCase{
				PreCheck: func() {
					testAccPreCheck(tp)
					testAccResourceVSphereVirtualMachinePreCheck(tp)
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
			},
		},
		{
			"swap scsi bus",
			resource.TestCase{
				PreCheck: func() {
					testAccPreCheck(tp)
					testAccResourceVSphereVirtualMachinePreCheck(tp)
				},
				Providers:    testAccProviders,
				CheckDestroy: testAccResourceVSphereVirtualMachineCheckExists(false),
				Steps: []resource.TestStep{
					{
						Config: testAccResourceVSphereVirtualMachineConfigBasic(),
						Check: resource.ComposeTestCheckFunc(
							testAccResourceVSphereVirtualMachineCheckExists(true),
							testAccResourceVSphereVirtualMachineCheckSCSIBus(virtualdevice.SubresourceControllerTypeLsiLogic),
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
			},
		},
		{
			"extraconfig",
			resource.TestCase{
				PreCheck: func() {
					testAccPreCheck(tp)
					testAccResourceVSphereVirtualMachinePreCheck(tp)
				},
				Providers:    testAccProviders,
				CheckDestroy: testAccResourceVSphereVirtualMachineCheckExists(false),
				Steps: []resource.TestStep{
					{
						Config: testAccResourceVSphereVirtualMachineConfigExtraConfig(),
						Check: resource.ComposeTestCheckFunc(
							testAccResourceVSphereVirtualMachineCheckExists(true),
							testAccResourceVSphereVirtualMachineCheckExtraConfig("foo", "bar"),
							testAccResourceVSphereVirtualMachineCheckExtraConfig("baz", "qux"),
						),
					},
				},
			},
		},
		{
			"attach existing vmdk",
			resource.TestCase{
				PreCheck: func() {
					testAccPreCheck(tp)
					testAccResourceVSphereVirtualMachinePreCheck(tp)
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
			},
		},
		{
			"in folder",
			resource.TestCase{
				PreCheck: func() {
					testAccPreCheck(tp)
					testAccResourceVSphereVirtualMachinePreCheck(tp)
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
			},
		},
		{
			"static mac",
			resource.TestCase{
				PreCheck: func() {
					testAccPreCheck(tp)
					testAccResourceVSphereVirtualMachinePreCheck(tp)
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
			},
		},
		{
			"single tag",
			resource.TestCase{
				PreCheck: func() {
					testAccPreCheck(tp)
					testAccResourceVSphereVirtualMachinePreCheck(tp)
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
			},
		},
		{
			"multiple tags",
			resource.TestCase{
				PreCheck: func() {
					testAccPreCheck(tp)
					testAccResourceVSphereVirtualMachinePreCheck(tp)
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
			},
		},
		{
			"switch tags",
			resource.TestCase{
				PreCheck: func() {
					testAccPreCheck(tp)
					testAccResourceVSphereVirtualMachinePreCheck(tp)
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
			},
		},
		{
			"clone from template",
			resource.TestCase{
				PreCheck: func() {
					testAccPreCheck(tp)
					testAccResourceVSphereVirtualMachinePreCheck(tp)
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
				},
			},
		},
		{
			"clone with different hostname",
			resource.TestCase{
				PreCheck: func() {
					testAccPreCheck(tp)
					testAccResourceVSphereVirtualMachinePreCheck(tp)
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
			},
		},
		{
			"clone with extra disks",
			resource.TestCase{
				PreCheck: func() {
					testAccPreCheck(tp)
					testAccResourceVSphereVirtualMachinePreCheck(tp)
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
			},
		},
		{
			"cpu hot add",
			resource.TestCase{
				PreCheck: func() {
					testAccPreCheck(tp)
					testAccResourceVSphereVirtualMachinePreCheck(tp)
				},
				Providers:    testAccProviders,
				CheckDestroy: testAccResourceVSphereVirtualMachineCheckExists(false),
				Steps: []resource.TestStep{
					{
						// Starting config
						Config: testAccResourceVSphereVirtualMachineConfigWithHotAdd(2, 1024, true, false, true),
						Check: resource.ComposeTestCheckFunc(
							testAccResourceVSphereVirtualMachineCheckExists(true),
							testAccResourceVSphereVirtualMachineCheckCPUMem(2, 1024),
						),
					},
					{
						// Add CPU w/hot-add
						Config: testAccResourceVSphereVirtualMachineConfigWithHotAdd(4, 1024, true, false, true),
						Check: resource.ComposeTestCheckFunc(
							testAccResourceVSphereVirtualMachineCheckExists(true),
							testAccResourceVSphereVirtualMachineCheckCPUMem(4, 1024),
							testAccResourceVSphereVirtualMachineCheckPowerOffEvent(false),
						),
					},
				},
			},
		},
		{
			"memory hot add",
			resource.TestCase{
				PreCheck: func() {
					testAccPreCheck(tp)
					testAccResourceVSphereVirtualMachinePreCheck(tp)
				},
				Providers:    testAccProviders,
				CheckDestroy: testAccResourceVSphereVirtualMachineCheckExists(false),
				Steps: []resource.TestStep{
					{
						// Starting config
						Config: testAccResourceVSphereVirtualMachineConfigWithHotAdd(2, 1024, true, false, true),
						Check: resource.ComposeTestCheckFunc(
							testAccResourceVSphereVirtualMachineCheckExists(true),
							testAccResourceVSphereVirtualMachineCheckCPUMem(2, 1024),
						),
					},
					{
						// Add memory with hot-add
						Config: testAccResourceVSphereVirtualMachineConfigWithHotAdd(2, 2048, true, false, true),
						Check: resource.ComposeTestCheckFunc(
							testAccResourceVSphereVirtualMachineCheckExists(true),
							testAccResourceVSphereVirtualMachineCheckCPUMem(2, 2048),
							testAccResourceVSphereVirtualMachineCheckPowerOffEvent(false),
						),
					},
				},
			},
		},
		{
			"dual-stack ipv4 and ipv6",
			resource.TestCase{
				PreCheck: func() {
					testAccPreCheck(tp)
					testAccResourceVSphereVirtualMachinePreCheck(tp)
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
						),
					},
				},
			},
		},
		{
			"windows template, customization events and proper IP",
			resource.TestCase{
				PreCheck: func() {
					testAccPreCheck(tp)
					testAccResourceVSphereVirtualMachinePreCheck(tp)
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
			},
		},
		{
			"host vmotion",
			resource.TestCase{
				PreCheck: func() {
					testAccPreCheck(tp)
					testAccResourceVSphereVirtualMachinePreCheck(tp)
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
			},
		},
		{
			"storage vmotion - global setting",
			resource.TestCase{
				PreCheck: func() {
					testAccPreCheck(tp)
					testAccResourceVSphereVirtualMachinePreCheck(tp)
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
			},
		},
		{
			"storage vmotion - single disk",
			resource.TestCase{
				PreCheck: func() {
					testAccPreCheck(tp)
					testAccResourceVSphereVirtualMachinePreCheck(tp)
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
			},
		},
		{
			"storage vmotion - pin datastore",
			resource.TestCase{
				PreCheck: func() {
					testAccPreCheck(tp)
					testAccResourceVSphereVirtualMachinePreCheck(tp)
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
			},
		},
		{
			"import",
			resource.TestCase{
				PreCheck: func() {
					testAccPreCheck(tp)
					testAccResourceVSphereVirtualMachinePreCheck(tp)
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
							"force_power_off",
							"migrate_wait_timeout",
							"shutdown_wait_timeout",
							"wait_for_guest_net_timeout",
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
			},
		},
	}

	for _, tc := range testAccResourceVSphereVirtualMachineCases {
		t.Run(tc.name, func(t *testing.T) {
			tp = t
			resource.Test(t, tc.testCase)
		})
	}
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
			return fmt.Errorf("expected MAC address to be %s, got %s", expected, actual)
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
				return fmt.Errorf("could not locate disk at index %d", n)
			}
		}
		for n, actual := range actualN {
			if actual != expectedN[n] {
				return fmt.Errorf("could not locate network interface at index %d", n)
			}
		}

		return nil
	}
}

func testAccResourceVSphereVirtualMachineCheckCdrom() resource.TestCheckFunc {
	return func(s *terraform.State) error {
		props, err := testGetVirtualMachineProperties(s, "vm")
		if err != nil {
			return err
		}

		for _, dev := range props.Config.Hardware.Device {
			if cdrom, ok := dev.(*types.VirtualCdrom); ok {
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
// test VM's SCSI bus is all of the specified SCSI type.
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
  memory   = 1024
  guest_id = "other3xLinux64Guest"

  network_interface {
    network_id = "${data.vsphere_network.network.id}"
  }

  disk {
    name = "terraform-test.vmdk"
    size = 20
  }
}
`,
		os.Getenv("VSPHERE_DATACENTER"),
		os.Getenv("VSPHERE_RESOURCE_POOL"),
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
  memory   = 1024
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
    name = "terraform-test.vmdk"
    size = 20
  }

  disk {
    name        = "terraform-test_1.vmdk"
    unit_number = 1
    size        = 10
  }

  disk {
    name        = "terraform-test_2.vmdk"
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
  memory   = 1024
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
    name = "terraform-test.vmdk"
    size = 20
  }

  disk {
    name        = "terraform-test_2.vmdk"
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
  memory   = 1024
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
    name = "terraform-test.vmdk"
    size = 20
  }

  disk {
    name        = "terraform-test_2.vmdk"
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
  memory   = 1024
  guest_id = "other3xLinux64Guest"

	wait_for_guest_net_timeout = -1

  network_interface {
    network_id = "${data.vsphere_network.network.id}"
  }

  disk {
    name = "terraform-test.vmdk"
    size = 20
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
    name = "terraform-test.vmdk"
    size = 20
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
  memory   = 1024
  guest_id = "other3xLinux64Guest"
	annotation = "${var.annotation}"

  network_interface {
    network_id = "${data.vsphere_network.network.id}"
  }

  disk {
    name = "terraform-test.vmdk"
    size = 20
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
  memory   = 1024
  guest_id = "other3xLinux64Guest"

  network_interface {
    network_id = "${data.vsphere_network.network.id}"
  }

  disk {
    name = "terraform-test.vmdk"
    size = %d
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
  memory   = 1024
  guest_id = "other3xLinux64Guest"

	scsi_type = "lsilogic-sas"

  network_interface {
    network_id = "${data.vsphere_network.network.id}"
  }

  disk {
    name = "terraform-test.vmdk"
    size = 20
  }
}
`,
		os.Getenv("VSPHERE_DATACENTER"),
		os.Getenv("VSPHERE_RESOURCE_POOL"),
		os.Getenv("VSPHERE_NETWORK_LABEL_PXE"),
		os.Getenv("VSPHERE_DATASTORE"),
	)
}

func testAccResourceVSphereVirtualMachineConfigExtraConfig() string {
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
  memory   = 1024
  guest_id = "other3xLinux64Guest"

  extra_config {
    "foo" = "bar"
    "baz" = "qux"
  }

  network_interface {
    network_id = "${data.vsphere_network.network.id}"
  }

  disk {
    name = "terraform-test.vmdk"
    size = 20
  }
}
`,
		os.Getenv("VSPHERE_DATACENTER"),
		os.Getenv("VSPHERE_RESOURCE_POOL"),
		os.Getenv("VSPHERE_NETWORK_LABEL_PXE"),
		os.Getenv("VSPHERE_DATASTORE"),
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
  memory   = 1024
  guest_id = "other3xLinux64Guest"

  network_interface {
    network_id = "${data.vsphere_network.network.id}"
  }

  disk {
    name = "terraform-test.vmdk"
    size = 20
  }

  disk {
    datastore_id = "${data.vsphere_datastore.datastore.id}"
    name         = "${vsphere_virtual_disk.disk.vmdk_path}"
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
  memory   = 1024
  guest_id = "other3xLinux64Guest"

  network_interface {
    network_id = "${data.vsphere_network.network.id}"
  }

  disk {
    name = "terraform-test.vmdk"
    size = 20
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
  memory   = 1024
  guest_id = "other3xLinux64Guest"

  network_interface {
    network_id     = "${data.vsphere_network.network.id}"
    use_static_mac = true
    mac_address    = "%s"
  }

  disk {
    name = "terraform-test.vmdk"
    size = 20
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
  memory   = 1024
  guest_id = "other3xLinux64Guest"

  network_interface {
    network_id = "${data.vsphere_network.network.id}"
  }

  disk {
    name = "terraform-test.vmdk"
    size = 20
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
  memory   = 1024
  guest_id = "other3xLinux64Guest"

  network_interface {
    network_id = "${data.vsphere_network.network.id}"
  }

  disk {
    name = "terraform-test.vmdk"
    size = 20
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
  memory   = 1024
  guest_id = "${data.vsphere_virtual_machine.template.guest_id}"

  network_interface {
    network_id   = "${data.vsphere_network.network.id}"
    adapter_type = "${data.vsphere_virtual_machine.template.network_interface_types[0]}"
  }

  disk {
    name = "terraform-test.vmdk"
    size = "${data.vsphere_virtual_machine.template.disk_sizes[0]}"
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
		os.Getenv("VSPHERE_DATASTORE"),
		os.Getenv("VSPHERE_TEMPLATE"),
		os.Getenv("VSPHERE_USE_LINKED_CLONE"),
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
  memory   = 1024
  guest_id = "${data.vsphere_virtual_machine.template.guest_id}"

  network_interface {
    network_id   = "${data.vsphere_network.network.id}"
    adapter_type = "${data.vsphere_virtual_machine.template.network_interface_types[0]}"
  }

  disk {
    name = "terraform-test.vmdk"
    size = "${data.vsphere_virtual_machine.template.disk_sizes[0]}"
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
		os.Getenv("VSPHERE_DATASTORE"),
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
    name = "terraform-test.vmdk"
    size = "${data.vsphere_virtual_machine.template.disk_sizes[0]}"
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
  memory   = 1024
  guest_id = "${data.vsphere_virtual_machine.template.guest_id}"

  network_interface {
    network_id   = "${data.vsphere_network.network.id}"
    adapter_type = "${data.vsphere_virtual_machine.template.network_interface_types[0]}"
  }

  disk {
    name = "terraform-test.vmdk"
    size = "${data.vsphere_virtual_machine.template.disk_sizes[0]}"
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

      ipv4_gateway = "${var.ipv4_gateway}"
			ipv6_gateway = "fd00::1"
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
  memory   = 1024
  guest_id = "ubuntu64Guest"

  network_interface {
    network_id   = "${data.vsphere_network.network.id}"
    adapter_type = "${data.vsphere_virtual_machine.template.network_interface_types[0]}"
  }

  disk {
    name = "terraform-test.vmdk"
    size = "${data.vsphere_virtual_machine.template.disk_sizes[0]}"
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
		os.Getenv("VSPHERE_DATASTORE"),
		os.Getenv("VSPHERE_TEMPLATE"),
		os.Getenv("VSPHERE_USE_LINKED_CLONE"),
		host,
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
  memory   = 1024
  guest_id = "ubuntu64Guest"

  network_interface {
    network_id   = "${data.vsphere_network.network.id}"
    adapter_type = "${data.vsphere_virtual_machine.template.network_interface_types[0]}"
  }

  disk {
    name = "terraform-test.vmdk"
    size = "${data.vsphere_virtual_machine.template.disk_sizes[0]}"
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
  memory   = 1024
  guest_id = "ubuntu64Guest"

  network_interface {
    network_id   = "${data.vsphere_network.network.id}"
    adapter_type = "${data.vsphere_virtual_machine.template.network_interface_types[0]}"
  }

  disk {
    name = "terraform-test.vmdk"
    size = "${data.vsphere_virtual_machine.template.disk_sizes[0]}"
  }
  
	disk {
		datastore_id = "${data.vsphere_datastore.disk_datastore.id}"
    name         = "terraform-test_1.vmdk"
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
  memory   = 1024
  guest_id = "ubuntu64Guest"

  network_interface {
    network_id   = "${data.vsphere_network.network.id}"
    adapter_type = "${data.vsphere_virtual_machine.template.network_interface_types[0]}"
  }

  disk {
    name = "terraform-test.vmdk"
    size = "${data.vsphere_virtual_machine.template.disk_sizes[0]}"
  }
  
	disk {
		datastore_id = "${data.vsphere_datastore.disk_datastore.id}"
    name         = "terraform-test_1.vmdk"
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
		datastore,
		os.Getenv("VSPHERE_TEMPLATE"),
		os.Getenv("VSPHERE_USE_LINKED_CLONE"),
		os.Getenv("VSPHERE_DATASTORE"),
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

variable "disk_name_eager" {
  default = "%s"
}

variable "disk_name_lazy" {
  default = "%s"
}

variable "disk_name_thin" {
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
  memory   = 1024
  guest_id = "${data.vsphere_virtual_machine.template.guest_id}"

  network_interface {
    network_id   = "${data.vsphere_network.network.id}"
    adapter_type = "${data.vsphere_virtual_machine.template.network_interface_types[0]}"
  }

  disk {
    name = "terraform-test.vmdk"
    size = "${data.vsphere_virtual_machine.template.disk_sizes[0]}"
  }

  disk {
    name             = "${var.disk_name_eager}"
    size             = 1
    unit_number      = 1
    thin_provisioned = false
    eagerly_scrub    = true
  }

  disk {
    name             = "${var.disk_name_lazy}"
    size             = 1
    unit_number      = 2
    thin_provisioned = false
    eagerly_scrub    = false
  }

  disk {
    name        = "${var.disk_name_thin}"
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
		os.Getenv("VSPHERE_DATASTORE"),
		os.Getenv("VSPHERE_TEMPLATE"),
		os.Getenv("VSPHERE_DS_VMFS_DISK0"),
		os.Getenv("VSPHERE_DS_VMFS_DISK1"),
		os.Getenv("VSPHERE_DS_VMFS_DISK2"),
		os.Getenv("VSPHERE_ESXI_HOST"),
		testAccResourceVSphereVirtualMachineDiskNameEager,
		testAccResourceVSphereVirtualMachineDiskNameLazy,
		testAccResourceVSphereVirtualMachineDiskNameThin,
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
    name = "terraform-test.vmdk"
    size = "${data.vsphere_virtual_machine.template.disk_sizes[0]}"
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
        ipv4_address = "${var.ipv4_address}"
        ipv4_netmask = "${var.ipv4_netmask}"
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
		os.Getenv("VSPHERE_DATASTORE"),
		os.Getenv("VSPHERE_TEMPLATE_WINDOWS"),
		os.Getenv("VSPHERE_USE_LINKED_CLONE"),
	)
}
