package vsphere

import (
	"errors"
	"fmt"
	"os"
	"reflect"
	"testing"

	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
	"github.com/terraform-providers/terraform-provider-vsphere/vsphere/internal/helper/structure"
	"github.com/vmware/govmomi/object"
	"github.com/vmware/govmomi/vim25/types"
)

const testAccResourceVSphereVirtualMachineV2AnnotationExpected = "managed by Terraform"

func TestAccResourceVSphereVirtualMachineV2(t *testing.T) {
	var tp *testing.T
	testAccResourceVSphereVirtualMachineV2Cases := []struct {
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
						Config: testAccResourceVSphereVirtualMachineV2ConfigBasic(),
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
						Config: testAccResourceVSphereVirtualMachineV2ConfigMultiDevice(),
						Check: resource.ComposeTestCheckFunc(
							testAccResourceVSphereVirtualMachineCheckExists(true),
							testAccResourceVSphereVirtualMachineV2CheckMultiDevice([]bool{true, true, true}, []bool{true, true, true}),
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
						Config: testAccResourceVSphereVirtualMachineV2ConfigBasic(),
						Check: resource.ComposeTestCheckFunc(
							testAccResourceVSphereVirtualMachineCheckExists(true),
						),
					},
					{
						Config: testAccResourceVSphereVirtualMachineV2ConfigMultiDevice(),
						Check: resource.ComposeTestCheckFunc(
							testAccResourceVSphereVirtualMachineCheckExists(true),
							testAccResourceVSphereVirtualMachineV2CheckMultiDevice([]bool{true, true, true}, []bool{true, true, true}),
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
						Config: testAccResourceVSphereVirtualMachineV2ConfigMultiDevice(),
						Check: resource.ComposeTestCheckFunc(
							testAccResourceVSphereVirtualMachineCheckExists(true),
							testAccResourceVSphereVirtualMachineV2CheckMultiDevice([]bool{true, true, true}, []bool{true, true, true}),
						),
					},
					{
						Config: testAccResourceVSphereVirtualMachineV2ConfigRemoveMiddle(),
						Check: resource.ComposeTestCheckFunc(
							testAccResourceVSphereVirtualMachineCheckExists(true),
							testAccResourceVSphereVirtualMachineV2CheckMultiDevice([]bool{true, false, true}, []bool{true, false, true}),
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
						Config: testAccResourceVSphereVirtualMachineV2ConfigCdrom(),
						Check: resource.ComposeTestCheckFunc(
							testAccResourceVSphereVirtualMachineCheckExists(true),
							testAccResourceVSphereVirtualMachineV2CheckCdrom(),
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
						Config: testAccResourceVSphereVirtualMachineV2ConfigBasic(),
						Check: resource.ComposeTestCheckFunc(
							testAccResourceVSphereVirtualMachineCheckExists(true),
						),
					},
					{
						Config: testAccResourceVSphereVirtualMachineV2ConfigBeefy(),
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
						Config: testAccResourceVSphereVirtualMachineV2ConfigWithHotAdd(2, 1024, true, false, false),
						Check: resource.ComposeTestCheckFunc(
							testAccResourceVSphereVirtualMachineCheckExists(true),
							testAccResourceVSphereVirtualMachineCheckCPUMem(2, 1024),
						),
					},
					{
						// Add CPU w/hot-add
						Config: testAccResourceVSphereVirtualMachineV2ConfigWithHotAdd(4, 1024, true, false, false),
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
						Config: testAccResourceVSphereVirtualMachineV2ConfigBasic(),
						Check: resource.ComposeTestCheckFunc(
							testAccResourceVSphereVirtualMachineCheckExists(true),
						),
					},
					{
						Config: testAccResourceVSphereVirtualMachineV2ConfigBasicAnnotation(),
						Check: resource.ComposeTestCheckFunc(
							testAccResourceVSphereVirtualMachineCheckExists(true),
							testAccResourceVSphereVirtualMachineCheckAnnotation(),
							testAccResourceVSphereVirtualMachineCheckPowerOffEvent(false),
						),
					},
				},
			},
		},
	}

	for _, tc := range testAccResourceVSphereVirtualMachineV2Cases {
		t.Run(tc.name, func(t *testing.T) {
			tp = t
			resource.Test(t, tc.testCase)
		})
	}
}

// testAccResourceVSphereVirtualMachineV2CheckMultiDevice is a check for proper
// parameters on the vsphere_virtual_machine multi-device test. This is a very
// specific check that checks for the specific disk and network devices. The
// configuration that this test asserts should be in the
// testAccResourceVSphereVirtualMachineV2ConfigMultiDevice resource.
func testAccResourceVSphereVirtualMachineV2CheckMultiDevice(expectedD, expectedN []bool) resource.TestCheckFunc {
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

func testAccResourceVSphereVirtualMachineV2CheckCdrom() resource.TestCheckFunc {
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

// testAccResourceVSphereVirtualMachineV2CheckDiskSize checks the first
// VirtualDisk it encounters for a specific size in GiB. It should only be used
// with test configurations with a single disk attached.
func testAccResourceVSphereVirtualMachineV2CheckDiskSize(expected int) resource.TestCheckFunc {
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

func testAccResourceVSphereVirtualMachineV2ConfigBasic() string {
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

resource "vsphere_virtual_machine_v2" "vm" {
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
    unit_number = 0
    size        = 20
  }
}
`,
		os.Getenv("VSPHERE_DATACENTER"),
		os.Getenv("VSPHERE_RESOURCE_POOL"),
		os.Getenv("VSPHERE_NETWORK_LABEL_PXE"),
		os.Getenv("VSPHERE_DATASTORE"),
	)
}

func testAccResourceVSphereVirtualMachineV2ConfigMultiDevice() string {
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

resource "vsphere_virtual_machine_v2" "vm" {
  name             = "terraform-test"
  resource_pool_id = "${data.vsphere_resource_pool.pool.id}"
  datastore_id     = "${data.vsphere_datastore.datastore.id}"

  num_cpus = 2
  memory   = 1024
  guest_id = "other3xLinux64Guest"

  network_interface {
    index                 = 0
    network_id            = "${data.vsphere_network.network.id}"
    bandwidth_share_level = "normal"
  }

  network_interface {
    index                 = 1
    network_id            = "${data.vsphere_network.network.id}"
    bandwidth_share_level = "high"
  }

  network_interface {
    index                 = 2
    network_id            = "${data.vsphere_network.network.id}"
    bandwidth_share_level = "low"
  }

  disk {
    index = 0
    size  = 20
  }

  disk {
    index = 1
    size  = 10
  }

  disk {
    index = 2
    size  = 5
  }
}
`,
		os.Getenv("VSPHERE_DATACENTER"),
		os.Getenv("VSPHERE_RESOURCE_POOL"),
		os.Getenv("VSPHERE_NETWORK_LABEL_PXE"),
		os.Getenv("VSPHERE_DATASTORE"),
	)
}

func testAccResourceVSphereVirtualMachineV2ConfigRemoveMiddle() string {
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

resource "vsphere_virtual_machine_v2" "vm" {
  name             = "terraform-test"
  resource_pool_id = "${data.vsphere_resource_pool.pool.id}"
  datastore_id     = "${data.vsphere_datastore.datastore.id}"

  num_cpus = 2
  memory   = 1024
  guest_id = "other3xLinux64Guest"

  network_interface {
    index                 = 0
    network_id            = "${data.vsphere_network.network.id}"
    bandwidth_share_level = "normal"
  }

  network_interface {
    index                 = 2
    network_id            = "${data.vsphere_network.network.id}"
    bandwidth_share_level = "low"
  }

  disk {
    index = 0
    size  = 20
  }

  disk {
    index = 2
    size  = 5
  }
}
`,
		os.Getenv("VSPHERE_DATACENTER"),
		os.Getenv("VSPHERE_RESOURCE_POOL"),
		os.Getenv("VSPHERE_NETWORK_LABEL_PXE"),
		os.Getenv("VSPHERE_DATASTORE"),
	)
}

func testAccResourceVSphereVirtualMachineV2ConfigCdrom() string {
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

resource "vsphere_virtual_machine_v2" "vm" {
  name             = "terraform-test"
  resource_pool_id = "${data.vsphere_resource_pool.pool.id}"
  datastore_id     = "${data.vsphere_datastore.datastore.id}"

  num_cpus                   = 2
  memory                     = 1024
  guest_id                   = "other3xLinux64Guest"
  wait_for_guest_net_timeout = -1

  network_interface {
    index      = 0
    network_id = "${data.vsphere_network.network.id}"
  }

  disk {
    index = 0
    size  = 20
  }

  cdrom {
    index        = 0
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

func testAccResourceVSphereVirtualMachineV2ConfigBeefy() string {
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

resource "vsphere_virtual_machine_v2" "vm" {
  name             = "terraform-test"
  resource_pool_id = "${data.vsphere_resource_pool.pool.id}"
  datastore_id     = "${data.vsphere_datastore.datastore.id}"

  num_cpus = 4
  memory   = 8192
  guest_id = "other3xLinux64Guest"

  network_interface {
    index      = 0
    network_id = "${data.vsphere_network.network.id}"
  }

  disk {
    index = 0
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

func testAccResourceVSphereVirtualMachineV2ConfigWithHotAdd(nc, nm int, cha, chr, mha bool) string {
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

resource "vsphere_virtual_machine_v2" "vm" {
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
    index      = 0
    network_id = "${data.vsphere_network.network.id}"
  }

  disk {
    index = 0
    size  = 20
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

func testAccResourceVSphereVirtualMachineV2ConfigBasicAnnotation() string {
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

resource "vsphere_virtual_machine_v2" "vm" {
  name             = "terraform-test"
  resource_pool_id = "${data.vsphere_resource_pool.pool.id}"
  datastore_id     = "${data.vsphere_datastore.datastore.id}"

  num_cpus = 2
  memory   = 1024
  guest_id = "other3xLinux64Guest"
	annotation = "${var.annotation}"

  network_interface {
    index      = 0
    network_id = "${data.vsphere_network.network.id}"
  }

  disk {
    index = 0
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

func testAccResourceVSphereVirtualMachineV2ConfigGrowDisk(size int) string {
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

resource "vsphere_virtual_machine_v2" "vm" {
  name             = "terraform-test"
  resource_pool_id = "${data.vsphere_resource_pool.pool.id}"
  datastore_id     = "${data.vsphere_datastore.datastore.id}"

  num_cpus = 2
  memory   = 1024
  guest_id = "other3xLinux64Guest"

  network_interface {
    index      = 0
    network_id = "${data.vsphere_network.network.id}"
  }

  disk {
    index = 0
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
