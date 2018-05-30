package vsphere

import (
	"errors"
	"fmt"
	"os"
	"testing"

	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
	"github.com/terraform-providers/terraform-provider-vsphere/vsphere/internal/helper/vappcontainer"
	"github.com/terraform-providers/terraform-provider-vsphere/vsphere/internal/helper/viapi"
	"github.com/vmware/govmomi/object"
)

func TestAccResourceVSphereVAppContainer_basic(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
			testAccResourceVSphereVAppContainerPreCheck(t)
		},
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccResourceVSphereVAppContainerConfigBasic(),
				Check: resource.ComposeTestCheckFunc(
					testAccResourceVSphereVAppContainerCheckExists(true),
					testAccResourceVSphereVAppContainerCheckFolder("parent_folder"),
				),
			},
		},
	})
}

func TestAccResourceVSphereVAppContainer_import(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
			testAccResourceVSphereVAppContainerPreCheck(t)
		},
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccResourceVSphereVAppContainerConfigBasic(),
				Check: resource.ComposeTestCheckFunc(
					testAccResourceVSphereVAppContainerCheckExists(true),
					testAccResourceVSphereVAppContainerCheckFolder("parent_folder"),
				),
			},
			{
				Config: testAccResourceVSphereVAppContainerConfigBasic(),
				Check: resource.ComposeTestCheckFunc(
					testAccResourceVSphereVAppContainerCheckExists(true),
				),
			},
			{
				ResourceName:      "vsphere_vapp_container.vapp_container",
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateIdFunc: func(s *terraform.State) (string, error) {
					vc, err := testGetVAppContainer(s, "vapp_container")
					if err != nil {
						return "", err
					}
					return fmt.Sprintf("/%s/host/%s/Resources/resource-pool-parent/%s",
						os.Getenv("VSPHERE_DATACENTER"),
						os.Getenv("VSPHERE_CLUSTER"),
						vc.Name(),
					), nil
				},
				Config: testAccResourceVSphereVAppContainerConfigBasic(),
				Check: resource.ComposeTestCheckFunc(
					testAccResourceVSphereVAppContainerCheckExists(true),
				),
			},
		},
	})
}

func TestAccResourceVSphereVAppContainer_vmBasic(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
			testAccResourceVSphereVAppContainerPreCheck(t)
		},
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccResourceVSphereVAppContainerConfigVM(),
				Check: resource.ComposeTestCheckFunc(
					testAccResourceVSphereVAppContainerCheckExists(true),
					testAccResourceVSphereVAppContainerContainsVM("vm"),
				),
			},
		},
	})
}

func TestAccResourceVSphereVAppContainer_vmMoveIntoVApp(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
			testAccResourceVSphereVAppContainerPreCheck(t)
		},
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccResourceVSphereVAppContainerConfigVMOutsideVApp(),
				Check: resource.ComposeTestCheckFunc(
					testAccResourceVSphereVAppContainerCheckExists(true),
				),
			},
			{
				Config: testAccResourceVSphereVAppContainerConfigVM(),
				Check: resource.ComposeTestCheckFunc(
					testAccResourceVSphereVAppContainerCheckExists(true),
					testAccResourceVSphereVAppContainerContainsVM("vm"),
				),
			},
		},
	})
}

func TestAccResourceVSphereVAppContainer_vmSDRS(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
			testAccResourceVSphereVAppContainerPreCheck(t)
		},
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccResourceVSphereVAppContainerConfigVmSdrs(),
				Check: resource.ComposeTestCheckFunc(
					testAccResourceVSphereVAppContainerCheckExists(true),
					testAccResourceVSphereVAppContainerContainsVM("vm"),
				),
			},
		},
	})
}

func TestAccResourceVSphereVAppContainer_vmClone(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
			testAccResourceVSphereVAppContainerPreCheck(t)
		},
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccResourceVSphereVAppContainerConfigVmClone(),
				Check: resource.ComposeTestCheckFunc(
					testAccResourceVSphereVAppContainerCheckExists(true),
					testAccResourceVSphereVAppContainerContainsVM("vm"),
				),
			},
		},
	})
}

func TestAccResourceVSphereVAppContainer_vmCloneSDRS(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
			testAccResourceVSphereVAppContainerPreCheck(t)
		},
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccResourceVSphereVAppContainerConfigVmSdrsClone(),
				Check: resource.ComposeTestCheckFunc(
					testAccResourceVSphereVAppContainerCheckExists(true),
					testAccResourceVSphereVAppContainerContainsVM("vm"),
				),
			},
		},
	})
}

func TestAccResourceVSphereVAppContainer_vmMoveIntoVAppSDRS(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
			testAccResourceVSphereVAppContainerPreCheck(t)
		},
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccResourceVSphereVAppContainerConfigVmSdrsNoVApp(),
				Check: resource.ComposeTestCheckFunc(
					testAccResourceVSphereVAppContainerCheckExists(true),
					testAccResourceVSphereVAppContainerContainsVM("vm"),
				),
			},
			{
				Config: testAccResourceVSphereVAppContainerConfigVmSdrs(),
				Check: resource.ComposeTestCheckFunc(
					testAccResourceVSphereVAppContainerCheckExists(true),
					testAccResourceVSphereVAppContainerContainsVM("vm"),
				),
			},
		},
	})
}

func testAccResourceVSphereVAppContainerPreCheck(t *testing.T) {
	if os.Getenv("VSPHERE_DATACENTER") == "" {
		t.Skip("set VSPHERE_DATACENTER to run vsphere_vapp_container acceptance tests")
	}
	if os.Getenv("VSPHERE_CLUSTER") == "" {
		t.Skip("set VSPHERE_CLUSTER to run vsphere_vapp_container acceptance tests")
	}
	if os.Getenv("VSPHERE_NETWORK_LABEL_PXE") == "" {
		t.Skip("set VSPHERE_NETWORK_LABEL_PXE to run vsphere_vapp_container acceptance tests")
	}
	if os.Getenv("VSPHERE_DATASTORE") == "" {
		t.Skip("set VSPHERE_DATASTORE to run vsphere_vapp_container acceptance tests")
	}
	if os.Getenv("VSPHERE_NFS_PATH") == "" {
		t.Skip("set VSPHERE_NFS_PATH to run vsphere_vapp_container acceptance tests")
	}
	if os.Getenv("VSPHERE_TEMPLATE") == "" {
		t.Skip("set VSPHERE_TEMPLATE to run vsphere_vapp_container acceptance tests")
	}
}

func testAccResourceVSphereVAppContainerCheckExists(expected bool) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		_, err := testGetVAppContainer(s, "vapp_container")
		if err != nil {
			if viapi.IsManagedObjectNotFoundError(err) && expected == false {
				// Expected missing
				return nil
			}
			return err
		}
		if !expected {
			return errors.New("expected vapp_container to be missing")
		}
		return nil
	}
}

func testAccResourceVSphereVAppContainerCheckFolder(expected string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		vc, err := testGetVAppContainer(s, "vapp_container")
		if err != nil {
			return err
		}
		f, err := testGetFolder(s, expected)
		if err != nil {
			return err
		}
		vcp, err := vappcontainer.Properties(vc)
		if err != nil {
			return err
		}
		if *vcp.ParentFolder != f.Reference() {
			return fmt.Errorf("expected path to be %s, got %s", expected, vcp.ParentFolder)
		}
		return nil
	}
}

func testAccResourceVSphereVAppContainerContainsVM(vmName string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		vm, err := testGetVirtualMachineProperties(s, vmName)
		if err != nil {
			return err
		}
		vc, err := testGetVAppContainer(s, "vapp_container")
		if err != nil {
			return err
		}
		if vm.ParentVApp != nil && vm.ParentVApp.Reference() != vc.Reference() {
			return fmt.Errorf("VM is not a part of vApp container")
		}
		return nil
	}
}

func testGetVAppContainer(s *terraform.State, resourceName string) (*object.VirtualApp, error) {
	vars, err := testClientVariablesForResource(s, fmt.Sprintf("%s.%s", resourceVSphereVAppContainerName, resourceName))
	if err != nil {
		return nil, err
	}
	return vappcontainer.FromID(vars.client, vars.resourceID)
}

func testAccResourceVSphereVAppContainerConfigBasic() string {
	return fmt.Sprintf(`
variable "datacenter" {
  default = "%s"
}

variable "cluster" {
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

data "vsphere_compute_cluster" "cluster" {
  name          = "${var.cluster}"
  datacenter_id = "${data.vsphere_datacenter.dc.id}"
}

resource "vsphere_resource_pool" "parent_resource_pool" {
  name                    = "resource-pool-parent"
  parent_resource_pool_id = "${data.vsphere_compute_cluster.cluster.resource_pool_id}"
}

resource "vsphere_folder" "parent_folder" {
  path          = "terraform-test-parent-folder"
  type          = "vm"
  datacenter_id = "${data.vsphere_datacenter.dc.id}"
}

resource "vsphere_vapp_container" "vapp_container" {
  name                    = "vapp-container-test"
  parent_resource_pool_id = "${vsphere_resource_pool.parent_resource_pool.id}"
  parent_folder_id        = "${vsphere_folder.parent_folder.id}"
}
`,
		os.Getenv("VSPHERE_DATACENTER"),
		os.Getenv("VSPHERE_CLUSTER"),
		os.Getenv("VSPHERE_DATASTORE"),
	)
}

func testAccResourceVSphereVAppContainerConfigVmSdrsNoVApp() string {
	return fmt.Sprintf(`
variable "datacenter" {
  default = "%s"
}

variable "cluster" {
  default = "%s"
}

variable "nfs_path" {
  default = "%s"
}

variable "nas_host" {
  default = "%s"
}

variable "network_label" {
  default = "%s"
}

variable "hosts" {
  default = [
    "n-esxi1.vsphere.hashicorptest.internal",
    "n-esxi2.vsphere.hashicorptest.internal",
    "n-esxi3.vsphere.hashicorptest.internal",
  ]
}

data "vsphere_datacenter" "dc" {
  name = "${var.datacenter}"
}

data "vsphere_network" "network" {
  name          = "${var.network_label}"
  datacenter_id = "${data.vsphere_datacenter.dc.id}"
}

data "vsphere_compute_cluster" "cluster" {
  name          = "${var.cluster}"
  datacenter_id = "${data.vsphere_datacenter.dc.id}"
}

data "vsphere_host" "esxi_hosts" {
  count         = "${length(var.hosts)}"
  name          = "${var.hosts[count.index]}"
  datacenter_id = "${data.vsphere_datacenter.dc.id}"
}

resource "vsphere_datastore_cluster" "datastore_cluster" {
  name          = "terraform-datastore-cluster-test"
  datacenter_id = "${data.vsphere_datacenter.dc.id}"
  sdrs_enabled  = true
}

resource "vsphere_nas_datastore" "datastore1" {
  name                 = "terraform-datastore-test1"
  host_system_ids      = ["${data.vsphere_host.esxi_hosts.*.id}"]
  datastore_cluster_id = "${vsphere_datastore_cluster.datastore_cluster.id}"

  type         = "NFS"
  remote_hosts = ["${var.nas_host}"]
  remote_path  = "${var.nfs_path}"
}

resource "vsphere_resource_pool" "parent_resource_pool" {
  name                    = "resource-pool-parent"
  parent_resource_pool_id = "${data.vsphere_compute_cluster.cluster.resource_pool_id}"
}

resource "vsphere_folder" "parent_folder" {
  path          = "terraform-test-parent-folder"
  type          = "vm"
  datacenter_id = "${data.vsphere_datacenter.dc.id}"
}

resource "vsphere_vapp_container" "vapp_container" {
  name                    = "vapp-container-test"
  parent_resource_pool_id = "${vsphere_resource_pool.parent_resource_pool.id}"
  parent_folder_id        = "${vsphere_folder.parent_folder.id}"
}

resource "vsphere_virtual_machine" "vm" {
  name                 = "terraform-virtual-machine-test"
  resource_pool_id     = "${vsphere_resource_pool.parent_resource_pool.id}"
  datastore_cluster_id = "${vsphere_datastore_cluster.datastore_cluster.id}"

  num_cpus                   = 2
  memory                     = 2048
  guest_id                   = "other3xLinux64Guest"
  wait_for_guest_net_timeout = -1
  depends_on                 = ["vsphere_nas_datastore.datastore1"]

  disk {
    label = "disk0"
    size  = "1"
  }

  network_interface {
    network_id = "${data.vsphere_network.network.id}"
  }
}
`,
		os.Getenv("VSPHERE_DATACENTER"),
		os.Getenv("VSPHERE_CLUSTER"),
		os.Getenv("VSPHERE_NFS_PATH"),
		os.Getenv("VSPHERE_NAS_HOST"),
		os.Getenv("VSPHERE_NETWORK_LABEL_PXE"),
	)
}

func testAccResourceVSphereVAppContainerConfigVmSdrs() string {
	return fmt.Sprintf(`
variable "datacenter" {
  default = "%s"
}

variable "cluster" {
  default = "%s"
}

variable "nfs_path" {
  default = "%s"
}

variable "nas_host" {
  default = "%s"
}

variable "network_label" {
  default = "%s"
}

variable "hosts" {
  default = [
    "n-esxi1.vsphere.hashicorptest.internal",
    "n-esxi2.vsphere.hashicorptest.internal",
    "n-esxi3.vsphere.hashicorptest.internal",
  ]
}

data "vsphere_datacenter" "dc" {
  name = "${var.datacenter}"
}

data "vsphere_network" "network" {
  name          = "${var.network_label}"
  datacenter_id = "${data.vsphere_datacenter.dc.id}"
}

data "vsphere_compute_cluster" "cluster" {
  name          = "${var.cluster}"
  datacenter_id = "${data.vsphere_datacenter.dc.id}"
}

data "vsphere_host" "esxi_hosts" {
  count         = "${length(var.hosts)}"
  name          = "${var.hosts[count.index]}"
  datacenter_id = "${data.vsphere_datacenter.dc.id}"
}

resource "vsphere_datastore_cluster" "datastore_cluster" {
  name          = "terraform-datastore-cluster-test"
  datacenter_id = "${data.vsphere_datacenter.dc.id}"
  sdrs_enabled  = true
}

resource "vsphere_nas_datastore" "datastore1" {
  name                 = "terraform-datastore-test1"
  host_system_ids      = ["${data.vsphere_host.esxi_hosts.*.id}"]
  datastore_cluster_id = "${vsphere_datastore_cluster.datastore_cluster.id}"

  type         = "NFS"
  remote_hosts = ["${var.nas_host}"]
  remote_path  = "${var.nfs_path}"
}

resource "vsphere_resource_pool" "parent_resource_pool" {
  name                    = "resource-pool-parent"
  parent_resource_pool_id = "${data.vsphere_compute_cluster.cluster.resource_pool_id}"
}

resource "vsphere_folder" "parent_folder" {
  path          = "terraform-test-parent-folder"
  type          = "vm"
  datacenter_id = "${data.vsphere_datacenter.dc.id}"
}

resource "vsphere_vapp_container" "vapp_container" {
  name                    = "vapp-container-test"
  parent_resource_pool_id = "${vsphere_resource_pool.parent_resource_pool.id}"
  parent_folder_id        = "${vsphere_folder.parent_folder.id}"
}

resource "vsphere_virtual_machine" "vm" {
  name                 = "terraform-virtual-machine-test"
  resource_pool_id     = "${vsphere_vapp_container.vapp_container.id}"
  datastore_cluster_id = "${vsphere_datastore_cluster.datastore_cluster.id}"

  num_cpus                   = 2
  memory                     = 2048
  guest_id                   = "other3xLinux64Guest"
  wait_for_guest_net_timeout = -1
  depends_on                 = ["vsphere_nas_datastore.datastore1"]

  disk {
    label = "disk0"
    size  = "1"
  }

  network_interface {
    network_id = "${data.vsphere_network.network.id}"
  }
}
`,
		os.Getenv("VSPHERE_DATACENTER"),
		os.Getenv("VSPHERE_CLUSTER"),
		os.Getenv("VSPHERE_NFS_PATH"),
		os.Getenv("VSPHERE_NAS_HOST"),
		os.Getenv("VSPHERE_NETWORK_LABEL_PXE"),
	)
}

func testAccResourceVSphereVAppContainerConfigVmSdrsClone() string {
	return fmt.Sprintf(`
variable "datacenter" {
  default = "%s"
}

variable "cluster" {
  default = "%s"
}

variable "nfs_path" {
  default = "%s"
}

variable "nas_host" {
  default = "%s"
}

variable "network_label" {
  default = "%s"
}

variable "hosts" {
  default = [
    "n-esxi1.vsphere.hashicorptest.internal",
    "n-esxi2.vsphere.hashicorptest.internal",
    "n-esxi3.vsphere.hashicorptest.internal",
  ]
}

variable "template" {
  default = "%s"
}

data "vsphere_datacenter" "dc" {
  name = "${var.datacenter}"
}

data "vsphere_network" "network" {
  name          = "${var.network_label}"
  datacenter_id = "${data.vsphere_datacenter.dc.id}"
}

data "vsphere_compute_cluster" "cluster" {
  name          = "${var.cluster}"
  datacenter_id = "${data.vsphere_datacenter.dc.id}"
}

data "vsphere_host" "esxi_hosts" {
  count         = "${length(var.hosts)}"
  name          = "${var.hosts[count.index]}"
  datacenter_id = "${data.vsphere_datacenter.dc.id}"
}

data "vsphere_virtual_machine" "template" {
  name          = "${var.template}"
  datacenter_id = "${data.vsphere_datacenter.dc.id}"
}

resource "vsphere_datastore_cluster" "datastore_cluster" {
  name          = "terraform-datastore-cluster-test"
  datacenter_id = "${data.vsphere_datacenter.dc.id}"
  sdrs_enabled  = true
}

resource "vsphere_nas_datastore" "datastore1" {
  name                 = "terraform-datastore-test1"
  host_system_ids      = ["${data.vsphere_host.esxi_hosts.*.id}"]
  datastore_cluster_id = "${vsphere_datastore_cluster.datastore_cluster.id}"

  type         = "NFS"
  remote_hosts = ["${var.nas_host}"]
  remote_path  = "${var.nfs_path}"
}

resource "vsphere_resource_pool" "parent_resource_pool" {
  name                    = "resource-pool-parent"
  parent_resource_pool_id = "${data.vsphere_compute_cluster.cluster.resource_pool_id}"
}

resource "vsphere_folder" "parent_folder" {
  path          = "terraform-test-parent-folder"
  type          = "vm"
  datacenter_id = "${data.vsphere_datacenter.dc.id}"
}

resource "vsphere_vapp_container" "vapp_container" {
  name                    = "vapp-container-test"
  parent_resource_pool_id = "${vsphere_resource_pool.parent_resource_pool.id}"
  parent_folder_id        = "${vsphere_folder.parent_folder.id}"
}

resource "vsphere_virtual_machine" "vm" {
  name                 = "terraform-virtual-machine-test"
  resource_pool_id     = "${vsphere_vapp_container.vapp_container.id}"
  datastore_cluster_id = "${vsphere_datastore_cluster.datastore_cluster.id}"

  num_cpus                   = 2
  memory                     = 2048
  guest_id                   = "ubuntu64Guest"
  wait_for_guest_net_timeout = -1
  depends_on                 = ["vsphere_nas_datastore.datastore1"]

  clone {
    template_uuid = "${data.vsphere_virtual_machine.template.id}"
  }

  disk {
    label = "disk0"
    size  = "32"
  }

  network_interface {
    network_id = "${data.vsphere_network.network.id}"
  }
}
`,
		os.Getenv("VSPHERE_DATACENTER"),
		os.Getenv("VSPHERE_CLUSTER"),
		os.Getenv("VSPHERE_NFS_PATH"),
		os.Getenv("VSPHERE_NAS_HOST"),
		os.Getenv("VSPHERE_NETWORK_LABEL_PXE"),
		os.Getenv("VSPHERE_TEMPLATE"),
	)
}

func testAccResourceVSphereVAppContainerConfigVmClone() string {
	return fmt.Sprintf(`
variable "datacenter" {
  default = "%s"
}

variable "cluster" {
  default = "%s"
}

variable "datastore" {
  default = "%s"
}

variable "network_label" {
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

data "vsphere_network" "network" {
  name          = "${var.network_label}"
  datacenter_id = "${data.vsphere_datacenter.dc.id}"
}

data "vsphere_compute_cluster" "cluster" {
  name          = "${var.cluster}"
  datacenter_id = "${data.vsphere_datacenter.dc.id}"
}

data "vsphere_virtual_machine" "template" {
  name          = "${var.template}"
  datacenter_id = "${data.vsphere_datacenter.dc.id}"
}

resource "vsphere_resource_pool" "parent_resource_pool" {
  name                    = "resource-pool-parent"
  parent_resource_pool_id = "${data.vsphere_compute_cluster.cluster.resource_pool_id}"
}

resource "vsphere_folder" "parent_folder" {
  path          = "terraform-test-parent-folder"
  type          = "vm"
  datacenter_id = "${data.vsphere_datacenter.dc.id}"
}

resource "vsphere_vapp_container" "vapp_container" {
  name                    = "vapp-container-test"
  parent_resource_pool_id = "${vsphere_resource_pool.parent_resource_pool.id}"
  parent_folder_id        = "${vsphere_folder.parent_folder.id}"
}

resource "vsphere_virtual_machine" "vm" {
  name             = "terraform-virtual-machine-test"
  resource_pool_id = "${vsphere_vapp_container.vapp_container.id}"
  datastore_id     = "${data.vsphere_datastore.datastore.id}"

  num_cpus                   = 2
  memory                     = 2048
  guest_id                   = "ubuntu64Guest"
  wait_for_guest_net_timeout = -1

  clone {
    template_uuid = "${data.vsphere_virtual_machine.template.id}"
    linked_clone  = true
  }

  disk {
    label = "disk0"
    size  = "32"
  }

  network_interface {
    network_id = "${data.vsphere_network.network.id}"
  }
}
`,
		os.Getenv("VSPHERE_DATACENTER"),
		os.Getenv("VSPHERE_CLUSTER"),
		os.Getenv("VSPHERE_DATASTORE"),
		os.Getenv("VSPHERE_NETWORK_LABEL_PXE"),
		os.Getenv("VSPHERE_TEMPLATE"),
	)
}

func testAccResourceVSphereVAppContainerConfigVM() string {
	return fmt.Sprintf(`
variable "datacenter" {
  default = "%s"
}

variable "cluster" {
  default = "%s"
}

variable "datastore" {
  default = "%s"
}

variable "network_label" {
  default = "%s"
}

data "vsphere_datacenter" "dc" {
  name = "${var.datacenter}"
}

data "vsphere_datastore" "datastore" {
  name          = "${var.datastore}"
  datacenter_id = "${data.vsphere_datacenter.dc.id}"
}

data "vsphere_network" "network" {
  name          = "${var.network_label}"
  datacenter_id = "${data.vsphere_datacenter.dc.id}"
}

data "vsphere_compute_cluster" "cluster" {
  name          = "${var.cluster}"
  datacenter_id = "${data.vsphere_datacenter.dc.id}"
}

resource "vsphere_resource_pool" "parent_resource_pool" {
  name                    = "resource-pool-parent"
  parent_resource_pool_id = "${data.vsphere_compute_cluster.cluster.resource_pool_id}"
}

resource "vsphere_folder" "parent_folder" {
  path          = "terraform-test-parent-folder"
  type          = "vm"
  datacenter_id = "${data.vsphere_datacenter.dc.id}"
}

resource "vsphere_vapp_container" "vapp_container" {
  name                    = "vapp-container-test"
  parent_resource_pool_id = "${vsphere_resource_pool.parent_resource_pool.id}"
  parent_folder_id        = "${vsphere_folder.parent_folder.id}"
}

resource "vsphere_virtual_machine" "vm" {
  name             = "terraform-virtual-machine-test"
  resource_pool_id = "${vsphere_vapp_container.vapp_container.id}"
  datastore_id     = "${data.vsphere_datastore.datastore.id}"

  num_cpus                   = 2
  memory                     = 2048
  guest_id                   = "other3xLinux64Guest"
  wait_for_guest_net_timeout = -1

  disk {
    label = "disk0"
    size  = "1"
  }

  network_interface {
    network_id = "${data.vsphere_network.network.id}"
  }
}
`,
		os.Getenv("VSPHERE_DATACENTER"),
		os.Getenv("VSPHERE_CLUSTER"),
		os.Getenv("VSPHERE_DATASTORE"),
		os.Getenv("VSPHERE_NETWORK_LABEL_PXE"),
	)
}

func testAccResourceVSphereVAppContainerConfigVMOutsideVApp() string {
	return fmt.Sprintf(`
variable "datacenter" {
  default = "%s"
}

variable "cluster" {
  default = "%s"
}

variable "datastore" {
  default = "%s"
}

variable "network_label" {
  default = "%s"
}

data "vsphere_datacenter" "dc" {
  name = "${var.datacenter}"
}

data "vsphere_datastore" "datastore" {
  name          = "${var.datastore}"
  datacenter_id = "${data.vsphere_datacenter.dc.id}"
}

data "vsphere_network" "network" {
  name          = "${var.network_label}"
  datacenter_id = "${data.vsphere_datacenter.dc.id}"
}

data "vsphere_compute_cluster" "cluster" {
  name          = "${var.cluster}"
  datacenter_id = "${data.vsphere_datacenter.dc.id}"
}

resource "vsphere_resource_pool" "parent_resource_pool" {
  name                    = "resource-pool-parent"
  parent_resource_pool_id = "${data.vsphere_compute_cluster.cluster.resource_pool_id}"
}

resource "vsphere_folder" "parent_folder" {
  path          = "terraform-test-parent-folder"
  type          = "vm"
  datacenter_id = "${data.vsphere_datacenter.dc.id}"
}

resource "vsphere_vapp_container" "vapp_container" {
  name                    = "vapp-container-test"
  parent_resource_pool_id = "${vsphere_resource_pool.parent_resource_pool.id}"
  parent_folder_id        = "${vsphere_folder.parent_folder.id}"
}

resource "vsphere_virtual_machine" "vm" {
  name             = "terraform-virtual-machine-test"
  resource_pool_id = "${vsphere_resource_pool.parent_resource_pool.id}"
  datastore_id     = "${data.vsphere_datastore.datastore.id}"

  num_cpus                   = 2
  memory                     = 2048
  guest_id                   = "other3xLinux64Guest"
  wait_for_guest_net_timeout = -1

  disk {
    label = "disk0"
    size  = "1"
  }

  network_interface {
    network_id = "${data.vsphere_network.network.id}"
  }
}
`,
		os.Getenv("VSPHERE_DATACENTER"),
		os.Getenv("VSPHERE_CLUSTER"),
		os.Getenv("VSPHERE_DATASTORE"),
		os.Getenv("VSPHERE_NETWORK_LABEL_PXE"),
	)
}
