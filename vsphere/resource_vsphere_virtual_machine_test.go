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
	testAccResourceVSphereVirtualMachineDiskNameEager      = "terraform-test-extra-eager"
	testAccResourceVSphereVirtualMachineDiskNameLazy       = "terraform-test-extra-lazy"
	testAccResourceVSphereVirtualMachineDiskNameThin       = "terraform-test-extra-thin"
	testAccResourceVSphereVirtualMachineDiskNameExtraVmdk  = "terraform-test-vm-extra-disk.vmdk"
	testAccResourceVSphereVirtualMachineStaticMacAddr      = "06:5c:89:2b:a0:64"
	testAccResourceVSphereVirtualMachineAnnotation         = "Managed by Terraform"
	testAccResourceVSphereVirtualMachineAnnotationExpected = "managed by Terraform"
	testAccResourceVSphereVirtualMachineSlashNetLabel      = "bar/baz"
)

func TestAccResourceVSphereVirtualMachine(t *testing.T) {
	var tp *testing.T
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
						Config: testAccResourceVSphereVirtualMachineConfigWithHotAdd(2, 1024, true, false, false),
						Check: resource.ComposeTestCheckFunc(
							testAccResourceVSphereVirtualMachineCheckExists(true),
							testAccResourceVSphereVirtualMachineCheckCPUMem(2, 1024),
						),
					},
					{
						// Add CPU w/hot-add
						Config: testAccResourceVSphereVirtualMachineConfigWithHotAdd(4, 1024, true, false, false),
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
		expectedEagerName := testAccResourceVSphereVirtualMachineDiskNameEager + ".vmdk"
		expectedLazyName := testAccResourceVSphereVirtualMachineDiskNameLazy + ".vmdk"
		expectedThinName := testAccResourceVSphereVirtualMachineDiskNameThin + ".vmdk"

		for _, dev := range props.Config.Hardware.Device {
			if disk, ok := dev.(*types.VirtualDisk); ok {
				if info, ok := disk.Backing.(*types.VirtualDiskFlatVer2BackingInfo); ok {
					var eager bool
					if info.EagerlyScrub != nil {
						eager = *info.EagerlyScrub
					}
					switch {
					case strings.HasSuffix(info.FileName, expectedEagerName) && eager:
						foundEager = true
					case strings.HasSuffix(info.FileName, expectedLazyName) && !eager:
						foundLazy = true
					case strings.HasSuffix(info.FileName, expectedThinName) && *info.ThinProvisioned:
						foundThin = true
					}
				}
			}
		}

		if !foundEager {
			return fmt.Errorf("could not locate disk: %s", expectedEagerName)
		}
		if !foundLazy {
			return fmt.Errorf("could not locate disk: %s", expectedLazyName)
		}
		if !foundThin {
			return fmt.Errorf("could not locate disk: %s", expectedThinName)
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
		actual := props.Guest.Net[0].MacAddress
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
		props, err := testGetVirtualMachineProperties(s, "vm")
		if err != nil {
			return err
		}
		l := object.VirtualDeviceList(props.Config.Hardware.Device)
		actual := virtualdevice.ReadSCSIBusState(l)
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

  num_cpus                  = %d
  memory                    = %d
  cpu_hot_add_enabled       = %t
  cpu_hot_remove_enabled    = %t
  memory_hot_add_enabled    = %t
  guest_id                  = "other3xLinux64Guest"

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
		nc,
		nm,
		cha,
		chr,
		mha,
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
    network_id = "${data.vsphere_network.network.id}"
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
    network_id = "${data.vsphere_network.network.id}"
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
