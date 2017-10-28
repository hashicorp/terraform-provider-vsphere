package vsphere

import (
	"errors"
	"fmt"
	"net"
	"os"
	"path"
	"regexp"
	"strconv"
	"strings"
	"testing"

	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
	"github.com/terraform-providers/terraform-provider-vsphere/vsphere/internal/helper/folder"
	"github.com/vmware/govmomi/vim25/types"
)

const (
	testAccResourceVSphereVirtualMachineDiskNameEager     = "terraform-test-extra-eager"
	testAccResourceVSphereVirtualMachineDiskNameLazy      = "terraform-test-extra-lazy"
	testAccResourceVSphereVirtualMachineDiskNameThin      = "terraform-test-extra-thin"
	testAccResourceVSphereVirtualMachineDiskNameExtraVmdk = "terraform-test-vm-extra-disk.vmdk"
	testAccResourceVSphereVirtualMachineStaticMacAddr     = "06:5c:89:2b:a0:64"
	testAccResourceVSphereVirtualMachineAnnotation        = "Managed by Terraform"
	testAccResourceVSphereVirtualMachineSlashNetLabel     = "bar/baz"
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
							resource.TestCheckResourceAttr("vsphere_virtual_machine.vm", "name", "terraform-test"),
							resource.TestCheckResourceAttr("vsphere_virtual_machine.vm", "vcpu", "2"),
							resource.TestMatchResourceAttr("vsphere_virtual_machine.vm", "uuid", regexp.MustCompile("[0-9a-f]{8}-([0-9a-f]{4}-){3}[0-9a-f]{12}")),
							resource.TestCheckResourceAttr("vsphere_virtual_machine.vm", "memory", "1024"),
							resource.TestCheckResourceAttr("vsphere_virtual_machine.vm", "disk.#", "1"),
							resource.TestCheckResourceAttr("vsphere_virtual_machine.vm", "network_interface.#", "1"),
							resource.TestCheckResourceAttr("vsphere_virtual_machine.vm", "network_interface.0.label", os.Getenv("VSPHERE_NETWORK_LABEL")),
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
						PlanOnly:           true,
						Config:             testAccResourceVSphereVirtualMachineConfigBasic(),
						ExpectNonEmptyPlan: true,
					},
				},
			},
		},
		{
			"always powered on",
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
						Config: testAccResourceVSphereVirtualMachineConfigBasic(),
						Check: resource.ComposeTestCheckFunc(
							testAccResourceVSphereVirtualMachineCheckPowerState(types.VirtualMachinePowerStatePoweredOn),
							testAccResourceVSphereVirtualMachineCheckExists(true),
						),
					},
				},
			},
		},
		{
			"different hostname",
			resource.TestCase{
				PreCheck: func() {
					testAccPreCheck(tp)
					testAccResourceVSphereVirtualMachinePreCheck(tp)
				},
				Providers:    testAccProviders,
				CheckDestroy: testAccResourceVSphereVirtualMachineCheckExists(false),
				Steps: []resource.TestStep{
					{
						Config: testAccResourceVSphereVirtualMachineConfigSeparateHostname(),
						Check: resource.ComposeTestCheckFunc(
							testAccResourceVSphereVirtualMachineCheckExists(true),
							testAccResourceVSphereVirtualMachineCheckHostname("terraform-test-renamed"),
						),
					},
				},
			},
		},
		{
			"extra disks",
			resource.TestCase{
				PreCheck: func() {
					testAccPreCheck(tp)
					testAccResourceVSphereVirtualMachinePreCheck(tp)
				},
				Providers:    testAccProviders,
				CheckDestroy: testAccResourceVSphereVirtualMachineCheckExists(false),
				Steps: []resource.TestStep{
					{
						Config: testAccResourceVSphereVirtualMachineConfigExtraDisks(),
						Check: resource.ComposeTestCheckFunc(
							testAccResourceVSphereVirtualMachineCheckExists(true),
							testAccResourceVSphereVirtualMachineCheckExtraDisks(),
						),
					},
				},
			},
		},
		{
			"custom config",
			resource.TestCase{
				PreCheck: func() {
					testAccPreCheck(tp)
					testAccResourceVSphereVirtualMachinePreCheck(tp)
				},
				Providers:    testAccProviders,
				CheckDestroy: testAccResourceVSphereVirtualMachineCheckExists(false),
				Steps: []resource.TestStep{
					{
						Config: testAccResourceVSphereVirtualMachineConfigCustomConfig(),
						Check: resource.ComposeTestCheckFunc(
							testAccResourceVSphereVirtualMachineCheckExists(true),
							resource.TestCheckResourceAttr("vsphere_virtual_machine.vm", "custom_configuration_parameters.foo", "bar"),
							resource.TestCheckResourceAttr("vsphere_virtual_machine.vm", "custom_configuration_parameters.baz", "qux"),
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
			"with annotation",
			resource.TestCase{
				PreCheck: func() {
					testAccPreCheck(tp)
					testAccResourceVSphereVirtualMachinePreCheck(tp)
				},
				Providers:    testAccProviders,
				CheckDestroy: testAccResourceVSphereVirtualMachineCheckExists(false),
				Steps: []resource.TestStep{
					{
						Config: testAccResourceVSphereVirtualMachineConfigWithAnnotation(),
						Check: resource.ComposeTestCheckFunc(
							copyStatePtr(&state),
							testAccResourceVSphereVirtualMachineCheckExists(true),
							testAccResourceVSphereVirtualMachineCheckAnnotation(),
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
			"dhcp only, don't wait for guest net",
			resource.TestCase{
				PreCheck: func() {
					testAccPreCheck(tp)
					testAccResourceVSphereVirtualMachinePreCheck(tp)
				},
				Providers:    testAccProviders,
				CheckDestroy: testAccResourceVSphereVirtualMachineCheckExists(false),
				Steps: []resource.TestStep{
					{
						Config: testAccResourceVSphereVirtualMachineConfigDHCPNoWait(),
						Check: resource.ComposeTestCheckFunc(
							testAccResourceVSphereVirtualMachineCheckExists(true),
							testAccResourceVSphereVirtualMachineCheckCustomizationSucceeded(),
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
						Config: testAccResourceVSphereVirtualMachineConfigWithTag(),
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
						Config: testAccResourceVSphereVirtualMachineConfigWithMultiTags(),
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
						Config: testAccResourceVSphereVirtualMachineConfigWithTag(),
						Check: resource.ComposeTestCheckFunc(
							testAccResourceVSphereVirtualMachineCheckExists(true),
							testAccResourceVSphereVirtualMachineCheckTags("terraform-test-tag"),
						),
					},
					{
						Config: testAccResourceVSphereVirtualMachineConfigWithMultiTags(),
						Check: resource.ComposeTestCheckFunc(
							testAccResourceVSphereVirtualMachineCheckExists(true),
							testAccResourceVSphereVirtualMachineCheckTags("terraform-test-tags-alt"),
						),
					},
				},
			},
		},
		{
			"with slash in network name",
			resource.TestCase{
				PreCheck: func() {
					testAccPreCheck(tp)
					testAccResourceVSphereVirtualMachinePreCheck(tp)
				},
				Providers:    testAccProviders,
				CheckDestroy: testAccResourceVSphereVirtualMachineCheckExists(false),
				Steps: []resource.TestStep{
					{
						Config: testAccResourceVSphereVirtualMachineConfigSlashNetwork(),
						Check: resource.ComposeTestCheckFunc(
							testAccResourceVSphereVirtualMachineCheckExists(true),
							resource.TestCheckResourceAttr("vsphere_virtual_machine.vm", "network_interface.#", "1"),
							resource.TestCheckResourceAttr(
								"vsphere_virtual_machine.vm",
								"network_interface.0.label",
								testAccResourceVSphereVirtualMachineSlashNetLabel,
							),
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

func testAccResourceVSphereVirtualMachineConfigBasic() string {
	return fmt.Sprintf(`
variable "datacenter" {
  default = "%s"
}

variable "cluster" {
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

variable "ipv4_prefix" {
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

resource "vsphere_virtual_machine" "vm" {
  name          = "terraform-test"
  datacenter    = "${var.datacenter}"
  cluster       = "${var.cluster}"
  resource_pool = "${var.resource_pool}"

  vcpu   = 2
  memory = 1024

  network_interface {
    label              = "${var.network_label}"
    ipv4_address       = "${var.ipv4_address}"
    ipv4_prefix_length = "${var.ipv4_prefix}"
    ipv4_gateway       = "${var.ipv4_gateway}"
  }

  disk {
    datastore = "${var.datastore}"
    template  = "${var.template}"
    iops      = 500
  }

  linked_clone = "${var.linked_clone != "" ? "true" : "false" }"
}
`,
		os.Getenv("VSPHERE_DATACENTER"),
		os.Getenv("VSPHERE_CLUSTER"),
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

func testAccResourceVSphereVirtualMachineConfigBeefy() string {
	return fmt.Sprintf(`
variable "datacenter" {
  default = "%s"
}

variable "cluster" {
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

variable "ipv4_prefix" {
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

resource "vsphere_virtual_machine" "vm" {
  name          = "terraform-test"
  datacenter    = "${var.datacenter}"
  cluster       = "${var.cluster}"
  resource_pool = "${var.resource_pool}"

  vcpu   = 4
  memory = 8192

  network_interface {
    label              = "${var.network_label}"
    ipv4_address       = "${var.ipv4_address}"
    ipv4_prefix_length = "${var.ipv4_prefix}"
    ipv4_gateway       = "${var.ipv4_gateway}"
  }

  disk {
    datastore = "${var.datastore}"
    template  = "${var.template}"
    iops      = 500
  }

  linked_clone = "${var.linked_clone != "" ? "true" : "false" }"
}
`,
		os.Getenv("VSPHERE_DATACENTER"),
		os.Getenv("VSPHERE_CLUSTER"),
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

func testAccResourceVSphereVirtualMachineConfigSeparateHostname() string {
	return fmt.Sprintf(`
variable "datacenter" {
  default = "%s"
}

variable "cluster" {
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

variable "ipv4_prefix" {
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

resource "vsphere_virtual_machine" "vm" {
  name          = "terraform-test"
  datacenter    = "${var.datacenter}"
  cluster       = "${var.cluster}"
  resource_pool = "${var.resource_pool}"

  vcpu   = 2
  memory = 1024

  hostname = "terraform-test-renamed"

  network_interface {
    label              = "${var.network_label}"
    ipv4_address       = "${var.ipv4_address}"
    ipv4_prefix_length = "${var.ipv4_prefix}"
    ipv4_gateway       = "${var.ipv4_gateway}"
  }

  disk {
    datastore = "${var.datastore}"
    template  = "${var.template}"
    iops      = 500
  }

  linked_clone = "${var.linked_clone != "" ? "true" : "false" }"
}
`,
		os.Getenv("VSPHERE_DATACENTER"),
		os.Getenv("VSPHERE_CLUSTER"),
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

func testAccResourceVSphereVirtualMachineConfigExtraDisks() string {
	return fmt.Sprintf(`
variable "datacenter" {
  default = "%s"
}

variable "cluster" {
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

variable "ipv4_prefix" {
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

variable "disk_name_eager" {
  default = "%s"
}

variable "disk_name_lazy" {
  default = "%s"
}

variable "disk_name_thin" {
  default = "%s"
}

resource "vsphere_virtual_machine" "vm" {
  name          = "terraform-test"
  datacenter    = "${var.datacenter}"
  cluster       = "${var.cluster}"
  resource_pool = "${var.resource_pool}"

  vcpu   = 2
  memory = 1024

  network_interface {
    label              = "${var.network_label}"
    ipv4_address       = "${var.ipv4_address}"
    ipv4_prefix_length = "${var.ipv4_prefix}"
    ipv4_gateway       = "${var.ipv4_gateway}"
  }

  disk {
    datastore = "${var.datastore}"
    template  = "${var.template}"
    iops      = 500
  }

  disk {
    size = 1
    type = "eager_zeroed"
    name = "${var.disk_name_eager}"
  }

  disk {
    size = 1
    type = "lazy"
    name = "${var.disk_name_lazy}"
  }
  
	disk {
    size = 1
    type = "thin"
    name = "${var.disk_name_thin}"
  }

  linked_clone = "${var.linked_clone != "" ? "true" : "false" }"
}
`,
		os.Getenv("VSPHERE_DATACENTER"),
		os.Getenv("VSPHERE_CLUSTER"),
		os.Getenv("VSPHERE_RESOURCE_POOL"),
		os.Getenv("VSPHERE_NETWORK_LABEL"),
		os.Getenv("VSPHERE_IPV4_ADDRESS"),
		os.Getenv("VSPHERE_IPV4_PREFIX"),
		os.Getenv("VSPHERE_IPV4_GATEWAY"),
		os.Getenv("VSPHERE_DATASTORE"),
		os.Getenv("VSPHERE_TEMPLATE"),
		os.Getenv("VSPHERE_USE_LINKED_CLONE"),
		testAccResourceVSphereVirtualMachineDiskNameEager,
		testAccResourceVSphereVirtualMachineDiskNameLazy,
		testAccResourceVSphereVirtualMachineDiskNameThin,
	)
}

func testAccResourceVSphereVirtualMachineConfigCustomConfig() string {
	return fmt.Sprintf(`
variable "datacenter" {
  default = "%s"
}

variable "cluster" {
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

variable "ipv4_prefix" {
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

resource "vsphere_virtual_machine" "vm" {
  name          = "terraform-test"
  datacenter    = "${var.datacenter}"
  cluster       = "${var.cluster}"
  resource_pool = "${var.resource_pool}"

  vcpu   = 2
  memory = 1024

  network_interface {
    label              = "${var.network_label}"
    ipv4_address       = "${var.ipv4_address}"
    ipv4_prefix_length = "${var.ipv4_prefix}"
    ipv4_gateway       = "${var.ipv4_gateway}"
  }

  custom_configuration_parameters {
    "foo" = "bar"
    "baz" = "qux"
  }

  disk {
    datastore = "${var.datastore}"
    template  = "${var.template}"
    iops      = 500
  }

  linked_clone = "${var.linked_clone != "" ? "true" : "false" }"
}
`,
		os.Getenv("VSPHERE_DATACENTER"),
		os.Getenv("VSPHERE_CLUSTER"),
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

func testAccResourceVSphereVirtualMachineConfigInFolder() string {
	return fmt.Sprintf(`
variable "datacenter" {
  default = "%s"
}

variable "cluster" {
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

variable "ipv4_prefix" {
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

data "vsphere_datacenter" "datacenter" {
  name = "${var.datacenter}"
}

resource "vsphere_folder" "folder" {
  path          = "terraform-test-vms"
  type          = "vm"
  datacenter_id = "${data.vsphere_datacenter.datacenter.id}"
}

resource "vsphere_virtual_machine" "vm" {
  name          = "terraform-test"
  datacenter    = "${var.datacenter}"
  cluster       = "${var.cluster}"
  resource_pool = "${var.resource_pool}"
  folder        = "${vsphere_folder.folder.path}"

  vcpu   = 2
  memory = 1024

  network_interface {
    label              = "${var.network_label}"
    ipv4_address       = "${var.ipv4_address}"
    ipv4_prefix_length = "${var.ipv4_prefix}"
    ipv4_gateway       = "${var.ipv4_gateway}"
  }

  disk {
    datastore = "${var.datastore}"
    template  = "${var.template}"
    iops      = 500
  }

  linked_clone = "${var.linked_clone != "" ? "true" : "false" }"
}
`,
		os.Getenv("VSPHERE_DATACENTER"),
		os.Getenv("VSPHERE_CLUSTER"),
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

func testAccResourceVSphereVirtualMachineConfigExistingVmdk() string {
	return fmt.Sprintf(`
variable "datacenter" {
  default = "%s"
}

variable "cluster" {
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

variable "ipv4_prefix" {
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

variable "extra_vmdk_name" {
  default = "%s"
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
  name          = "terraform-test"
  datacenter    = "${var.datacenter}"
  cluster       = "${var.cluster}"
  resource_pool = "${var.resource_pool}"

  vcpu   = 2
  memory = 1024

  network_interface {
    label              = "${var.network_label}"
    ipv4_address       = "${var.ipv4_address}"
    ipv4_prefix_length = "${var.ipv4_prefix}"
    ipv4_gateway       = "${var.ipv4_gateway}"
  }

  disk {
    datastore = "${var.datastore}"
    template  = "${var.template}"
    iops      = 500
  }

  disk {
    datastore      = "${var.datastore}"
    vmdk           = "${vsphere_virtual_disk.disk.vmdk_path}"
    keep_on_remove = true
  }

  linked_clone = "${var.linked_clone != "" ? "true" : "false" }"
}
`,
		os.Getenv("VSPHERE_DATACENTER"),
		os.Getenv("VSPHERE_CLUSTER"),
		os.Getenv("VSPHERE_RESOURCE_POOL"),
		os.Getenv("VSPHERE_NETWORK_LABEL"),
		os.Getenv("VSPHERE_IPV4_ADDRESS"),
		os.Getenv("VSPHERE_IPV4_PREFIX"),
		os.Getenv("VSPHERE_IPV4_GATEWAY"),
		os.Getenv("VSPHERE_DATASTORE"),
		os.Getenv("VSPHERE_TEMPLATE"),
		os.Getenv("VSPHERE_USE_LINKED_CLONE"),
		testAccResourceVSphereVirtualMachineDiskNameExtraVmdk,
	)
}

func testAccResourceVSphereVirtualMachineConfigDualStack() string {
	return fmt.Sprintf(`
variable "datacenter" {
  default = "%s"
}

variable "cluster" {
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

variable "ipv4_prefix" {
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

resource "vsphere_virtual_machine" "vm" {
  name          = "terraform-test"
  datacenter    = "${var.datacenter}"
  cluster       = "${var.cluster}"
  resource_pool = "${var.resource_pool}"

  vcpu   = 2
  memory = 1024

  network_interface {
    label              = "${var.network_label}"
    ipv4_address       = "${var.ipv4_address}"
    ipv4_prefix_length = "${var.ipv4_prefix}"
    ipv4_gateway       = "${var.ipv4_gateway}"
    ipv6_address       = "fd00::2"
    ipv6_prefix_length = "32"
    ipv6_gateway       = "fd00::1"
  }

  disk {
    datastore = "${var.datastore}"
    template  = "${var.template}"
    iops      = 500
  }

  linked_clone = "${var.linked_clone != "" ? "true" : "false" }"
}
`,
		os.Getenv("VSPHERE_DATACENTER"),
		os.Getenv("VSPHERE_CLUSTER"),
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

func testAccResourceVSphereVirtualMachineConfigStaticMAC() string {
	return fmt.Sprintf(`
variable "datacenter" {
  default = "%s"
}

variable "cluster" {
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

variable "ipv4_prefix" {
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

variable "static_mac_addr" {
  default = "%s"
}

resource "vsphere_virtual_machine" "vm" {
  name          = "terraform-test"
  datacenter    = "${var.datacenter}"
  cluster       = "${var.cluster}"
  resource_pool = "${var.resource_pool}"

  vcpu   = 2
  memory = 1024

  network_interface {
    label              = "${var.network_label}"
    mac_address        = "${var.static_mac_addr}"
    ipv4_address       = "${var.ipv4_address}"
    ipv4_prefix_length = "${var.ipv4_prefix}"
    ipv4_gateway       = "${var.ipv4_gateway}"
  }

  disk {
    datastore = "${var.datastore}"
    template  = "${var.template}"
    iops      = 500
  }

  linked_clone = "${var.linked_clone != "" ? "true" : "false" }"
}
`,
		os.Getenv("VSPHERE_DATACENTER"),
		os.Getenv("VSPHERE_CLUSTER"),
		os.Getenv("VSPHERE_RESOURCE_POOL"),
		os.Getenv("VSPHERE_NETWORK_LABEL"),
		os.Getenv("VSPHERE_IPV4_ADDRESS"),
		os.Getenv("VSPHERE_IPV4_PREFIX"),
		os.Getenv("VSPHERE_IPV4_GATEWAY"),
		os.Getenv("VSPHERE_DATASTORE"),
		os.Getenv("VSPHERE_TEMPLATE"),
		os.Getenv("VSPHERE_USE_LINKED_CLONE"),
		testAccResourceVSphereVirtualMachineStaticMacAddr,
	)
}

func testAccResourceVSphereVirtualMachineConfigWithAnnotation() string {
	return fmt.Sprintf(`
variable "datacenter" {
  default = "%s"
}

variable "cluster" {
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

variable "ipv4_prefix" {
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

variable "annotation" {
	default = "%s"
}

resource "vsphere_virtual_machine" "vm" {
  name          = "terraform-test"
  datacenter    = "${var.datacenter}"
  cluster       = "${var.cluster}"
  resource_pool = "${var.resource_pool}"
  annotation    = "${var.annotation}"

  vcpu   = 2
  memory = 1024

  network_interface {
    label              = "${var.network_label}"
    ipv4_address       = "${var.ipv4_address}"
    ipv4_prefix_length = "${var.ipv4_prefix}"
    ipv4_gateway       = "${var.ipv4_gateway}"
  }

  disk {
    datastore = "${var.datastore}"
    template  = "${var.template}"
    iops      = 500
  }

  linked_clone = "${var.linked_clone != "" ? "true" : "false" }"
}
`,
		os.Getenv("VSPHERE_DATACENTER"),
		os.Getenv("VSPHERE_CLUSTER"),
		os.Getenv("VSPHERE_RESOURCE_POOL"),
		os.Getenv("VSPHERE_NETWORK_LABEL"),
		os.Getenv("VSPHERE_IPV4_ADDRESS"),
		os.Getenv("VSPHERE_IPV4_PREFIX"),
		os.Getenv("VSPHERE_IPV4_GATEWAY"),
		os.Getenv("VSPHERE_DATASTORE"),
		os.Getenv("VSPHERE_TEMPLATE"),
		os.Getenv("VSPHERE_USE_LINKED_CLONE"),
		testAccResourceVSphereVirtualMachineAnnotation,
	)
}

func testAccResourceVSphereVirtualMachineConfigWindows() string {
	return fmt.Sprintf(`
variable "datacenter" {
  default = "%s"
}

variable "cluster" {
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

variable "ipv4_prefix" {
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

resource "vsphere_virtual_machine" "vm" {
  name          = "terraform-test"
  datacenter    = "${var.datacenter}"
  cluster       = "${var.cluster}"
  resource_pool = "${var.resource_pool}"

  vcpu   = 4
  memory = 4096

  network_interface {
    label              = "${var.network_label}"
    ipv4_address       = "${var.ipv4_address}"
    ipv4_prefix_length = "${var.ipv4_prefix}"
    ipv4_gateway       = "${var.ipv4_gateway}"
  }

  disk {
    datastore = "${var.datastore}"
    template  = "${var.template}"
    iops      = 500
  }

  windows_opt_config {
    admin_password = "VMw4re"
  }

  linked_clone = "${var.linked_clone != "" ? "true" : "false" }"
}
`,
		os.Getenv("VSPHERE_DATACENTER"),
		os.Getenv("VSPHERE_CLUSTER"),
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

func testAccResourceVSphereVirtualMachineConfigDHCPNoWait() string {
	return fmt.Sprintf(`
variable "datacenter" {
  default = "%s"
}

variable "cluster" {
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

resource "vsphere_virtual_machine" "vm" {
  name          = "terraform-test"
  datacenter    = "${var.datacenter}"
  cluster       = "${var.cluster}"
  resource_pool = "${var.resource_pool}"

  vcpu   = 2
  memory = 1024

  network_interface {
    label = "${var.network_label}"
  }

  wait_for_guest_net = false

  disk {
    datastore = "${var.datastore}"
    template  = "${var.template}"
    iops      = 500
  }

  linked_clone = "${var.linked_clone != "" ? "true" : "false" }"
}
`,
		os.Getenv("VSPHERE_DATACENTER"),
		os.Getenv("VSPHERE_CLUSTER"),
		os.Getenv("VSPHERE_RESOURCE_POOL"),
		os.Getenv("VSPHERE_NETWORK_LABEL"),
		os.Getenv("VSPHERE_DATASTORE"),
		os.Getenv("VSPHERE_TEMPLATE"),
		os.Getenv("VSPHERE_USE_LINKED_CLONE"),
	)
}

func testAccResourceVSphereVirtualMachineConfigWithTag() string {
	return fmt.Sprintf(`
variable "datacenter" {
  default = "%s"
}

variable "cluster" {
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

variable "ipv4_prefix" {
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
  name          = "terraform-test"
  datacenter    = "${var.datacenter}"
  cluster       = "${var.cluster}"
  resource_pool = "${var.resource_pool}"

  vcpu   = 2
  memory = 1024

  network_interface {
    label              = "${var.network_label}"
    ipv4_address       = "${var.ipv4_address}"
    ipv4_prefix_length = "${var.ipv4_prefix}"
    ipv4_gateway       = "${var.ipv4_gateway}"
  }

  disk {
    datastore = "${var.datastore}"
    template  = "${var.template}"
    iops      = 500
  }

  linked_clone = "${var.linked_clone != "" ? "true" : "false" }"

  tags = [
    "${vsphere_tag.terraform-test-tag.id}",
  ]
}
`,
		os.Getenv("VSPHERE_DATACENTER"),
		os.Getenv("VSPHERE_CLUSTER"),
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

func testAccResourceVSphereVirtualMachineConfigWithMultiTags() string {
	return fmt.Sprintf(`
variable "datacenter" {
  default = "%s"
}

variable "cluster" {
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

variable "ipv4_prefix" {
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

variable "extra_tags" {
  default = [
    "terraform-test-thing1",
    "terraform-test-thing2",
  ]
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
  name          = "terraform-test"
  datacenter    = "${var.datacenter}"
  cluster       = "${var.cluster}"
  resource_pool = "${var.resource_pool}"

  vcpu   = 2
  memory = 1024

  network_interface {
    label              = "${var.network_label}"
    ipv4_address       = "${var.ipv4_address}"
    ipv4_prefix_length = "${var.ipv4_prefix}"
    ipv4_gateway       = "${var.ipv4_gateway}"
  }

  disk {
    datastore = "${var.datastore}"
    template  = "${var.template}"
    iops      = 500
  }

  linked_clone = "${var.linked_clone != "" ? "true" : "false" }"

  tags = ["${vsphere_tag.terraform-test-tags-alt.*.id}"]
}
`,
		os.Getenv("VSPHERE_DATACENTER"),
		os.Getenv("VSPHERE_CLUSTER"),
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

func testAccResourceVSphereVirtualMachineConfigSlashNetwork() string {
	return fmt.Sprintf(`
variable "datacenter" {
  default = "%s"
}

variable "hosts" {
  default = [
    "%s",
    "%s",
    "%s",
  ]
}

variable "switch_nic" {
  default = "%s"
}

variable "cluster" {
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

variable "ipv4_prefix" {
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

data "vsphere_datacenter" "datacenter" {
  name = "${var.datacenter}"
}

data "vsphere_host" "host" {
  count         = "${length(var.hosts)}"
  name          = "${var.hosts[count.index]}"
  datacenter_id = "${data.vsphere_datacenter.datacenter.id}"
}

resource "vsphere_host_virtual_switch" "switch" {
  count          = "${length(data.vsphere_host.host.*.id)}"
  name           = "vSwitchTerraformTest"
  host_system_id = "${data.vsphere_host.host.*.id[count.index]}"

  network_adapters = ["${var.switch_nic}"]

  active_nics  = ["${var.switch_nic}"]
  standby_nics = []
}

resource "vsphere_host_port_group" "pg" {
  count               = "${length(data.vsphere_host.host.*.id)}"
  name                = "${var.network_label}"
  host_system_id      = "${data.vsphere_host.host.*.id[count.index]}"
  virtual_switch_name = "${vsphere_host_virtual_switch.switch.*.name[count.index]}"
}

resource "vsphere_virtual_machine" "vm" {
  name          = "terraform-test"
  datacenter    = "${var.datacenter}"
  cluster       = "${var.cluster}"
  resource_pool = "${var.resource_pool}"

  vcpu   = 2
  memory = 1024

  network_interface {
    label              = "${vsphere_host_port_group.pg.0.name}"
    ipv4_address       = "${var.ipv4_address}"
    ipv4_prefix_length = "${var.ipv4_prefix}"
    ipv4_gateway       = "${var.ipv4_gateway}"
  }

  disk {
    datastore = "${var.datastore}"
    template  = "${var.template}"
    iops      = 500
  }

  depends_on = ["vsphere_host_port_group.pg"]

  linked_clone = "${var.linked_clone != "" ? "true" : "false" }"
}
`,
		os.Getenv("VSPHERE_DATACENTER"),
		os.Getenv("VSPHERE_ESXI_HOST"),
		os.Getenv("VSPHERE_ESXI_HOST2"),
		os.Getenv("VSPHERE_ESXI_HOST3"),
		os.Getenv("VSPHERE_HOST_NIC0"),
		os.Getenv("VSPHERE_CLUSTER"),
		os.Getenv("VSPHERE_RESOURCE_POOL"),
		testAccResourceVSphereVirtualMachineSlashNetLabel,
		os.Getenv("VSPHERE_IPV4_ADDRESS"),
		os.Getenv("VSPHERE_IPV4_PREFIX"),
		os.Getenv("VSPHERE_IPV4_GATEWAY"),
		os.Getenv("VSPHERE_DATASTORE"),
		os.Getenv("VSPHERE_TEMPLATE"),
		os.Getenv("VSPHERE_USE_LINKED_CLONE"),
	)
}
