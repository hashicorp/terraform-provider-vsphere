package vsphere

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"reflect"
	"testing"

	"github.com/davecgh/go-spew/spew"
	"github.com/hashicorp/terraform-plugin-sdk/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/terraform"
	"github.com/hashicorp/terraform-provider-vsphere/vsphere/internal/helper/structure"
	"github.com/hashicorp/terraform-provider-vsphere/vsphere/internal/helper/viapi"
	"github.com/vmware/govmomi/vim25/types"
)

func TestAccResourceVSphereStorageDrsVMOverride_basic(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
			testAccResourceVSphereStorageDrsVMOverridePreCheck(t)
		},
		Providers:    testAccProviders,
		CheckDestroy: testAccResourceVSphereStorageDrsVMOverrideExists(false),
		Steps: []resource.TestStep{
			{
				Config: testAccResourceVSphereStorageDrsVMOverrideConfigBasic(),
				Check: resource.ComposeTestCheckFunc(
					testAccResourceVSphereStorageDrsVMOverrideExists(true),
					testAccResourceVSphereStorageDrsVMOverrideMatch("", structure.BoolPtr(false), nil),
				),
			},
			{
				ResourceName:      "vsphere_storage_drs_vm_override.drs_vm_override",
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateIdFunc: func(s *terraform.State) (string, error) {
					pod, err := testGetDatastoreCluster(s, "datastore_cluster")
					if err != nil {
						return "", err
					}
					vm, err := testGetVirtualMachine(s, "vm")
					if err != nil {
						return "", err
					}

					m := make(map[string]string)
					m["datastore_cluster_path"] = pod.InventoryPath
					m["virtual_machine_path"] = vm.InventoryPath
					b, err := json.Marshal(m)
					if err != nil {
						return "", err
					}

					return string(b), nil
				},
				Config: testAccResourceVSphereStorageDrsVMOverrideConfigBasic(),
				Check: resource.ComposeTestCheckFunc(
					testAccResourceVSphereStorageDrsVMOverrideExists(true),
					testAccResourceVSphereStorageDrsVMOverrideMatch("", structure.BoolPtr(false), nil),
				),
			},
		},
	})
}

func TestAccResourceVSphereStorageDrsVMOverride_overrides(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
			testAccResourceVSphereStorageDrsVMOverridePreCheck(t)
		},
		Providers:    testAccProviders,
		CheckDestroy: testAccResourceVSphereStorageDrsVMOverrideExists(false),
		Steps: []resource.TestStep{
			{
				Config: testAccResourceVSphereStorageDrsVMOverrideConfigOverrides(),
				Check: resource.ComposeTestCheckFunc(
					testAccResourceVSphereStorageDrsVMOverrideExists(true),
					testAccResourceVSphereStorageDrsVMOverrideMatch("automated", nil, structure.BoolPtr(false)),
				),
			},
		},
	})
}

func TestAccResourceVSphereStorageDrsVMOverride_update(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
			testAccResourceVSphereStorageDrsVMOverridePreCheck(t)
		},
		Providers:    testAccProviders,
		CheckDestroy: testAccResourceVSphereStorageDrsVMOverrideExists(false),
		Steps: []resource.TestStep{
			{
				Config: testAccResourceVSphereStorageDrsVMOverrideConfigBasic(),
				Check: resource.ComposeTestCheckFunc(
					testAccResourceVSphereStorageDrsVMOverrideExists(true),
					testAccResourceVSphereStorageDrsVMOverrideMatch("", structure.BoolPtr(false), nil),
				),
			},
			{
				Config: testAccResourceVSphereStorageDrsVMOverrideConfigOverrides(),
				Check: resource.ComposeTestCheckFunc(
					testAccResourceVSphereStorageDrsVMOverrideExists(true),
					testAccResourceVSphereStorageDrsVMOverrideMatch("automated", nil, structure.BoolPtr(false)),
				),
			},
		},
	})
}

func testAccResourceVSphereStorageDrsVMOverridePreCheck(t *testing.T) {
	if os.Getenv("TF_VAR_VSPHERE_DATACENTER") == "" {
		t.Skip("set TF_VAR_VSPHERE_DATACENTER to run vsphere_storage_drs_vm_override acceptance tests")
	}
	if os.Getenv("TF_VAR_VSPHERE_NAS_HOST") == "" {
		t.Skip("set TF_VAR_VSPHERE_NAS_HOST to run vsphere_storage_drs_vm_override acceptance tests")
	}
	if os.Getenv("TF_VAR_VSPHERE_NFS_PATH") == "" {
		t.Skip("set TF_VAR_VSPHERE_NFS_PATH to run vsphere_storage_drs_vm_override acceptance tests")
	}
	if os.Getenv("TF_VAR_VSPHERE_NFS_DS_NAME") == "" {
		t.Skip("set TF_VAR_VSPHERE_ESXI_HOST to run vsphere_storage_drs_vm_override acceptance tests")
	}
	if os.Getenv("TF_VAR_VSPHERE_ESXI_HOST2") == "" {
		t.Skip("set TF_VAR_VSPHERE_ESXI_HOST2 to run vsphere_storage_drs_vm_override acceptance tests")
	}
	if os.Getenv("TF_VAR_VSPHERE_ESXI_HOST3") == "" {
		t.Skip("set TF_VAR_VSPHERE_ESXI_HOST3 to run vsphere_storage_drs_vm_override acceptance tests")
	}
	if os.Getenv("TF_VAR_VSPHERE_RESOURCE_POOL") == "" {
		t.Skip("set TF_VAR_VSPHERE_RESOURCE_POOL to run vsphere_storage_drs_vm_override acceptance tests")
	}
	if os.Getenv("TF_VAR_VSPHERE_PG_NAME") == "" {
		t.Skip("set TF_VAR_VSPHERE_PG_NAME to run vsphere_storage_drs_vm_override acceptance tests")
	}
}

func testAccResourceVSphereStorageDrsVMOverrideExists(expected bool) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		info, err := testGetDatastoreClusterSDRSVMConfig(s, "drs_vm_override")
		if err != nil {
			if viapi.IsManagedObjectNotFoundError(err) && expected == false {
				// This is not necessarily a missing override, but more than likely a
				// missing datastore cluster, which happens during destroy as the
				// dependent resources will be missing as well, so want to treat this
				// as a deleted override as well.
				return nil
			}
			return err
		}

		switch {
		case info == nil && !expected:
			// Expected missing
			return nil
		case info == nil && expected:
			// Expected to exist
			return errors.New("storage DRS VM override missing when expected to exist")
		case !expected:
			return errors.New("storage DRS VM override still present when expected to be missing")
		}

		return nil
	}
}

func testAccResourceVSphereStorageDrsVMOverrideMatch(behavior string, enabled, intraVMAffinity *bool) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		actual, err := testGetDatastoreClusterSDRSVMConfig(s, "drs_vm_override")
		if err != nil {
			return err
		}

		if actual == nil {
			return errors.New("storage DRS VM override missing")
		}

		expected := &types.StorageDrsVmConfigInfo{
			Behavior:        behavior,
			Enabled:         enabled,
			IntraVmAffinity: intraVMAffinity,
			Vm:              actual.Vm,
		}

		if !reflect.DeepEqual(expected, actual) {
			return spew.Errorf("expected %#v got %#v", expected, actual)
		}

		return nil
	}
}

func testAccResourceVSphereStorageDrsVMOverrideConfigBasic() string {
	return fmt.Sprintf(`
variable "datacenter" {
  default = "%s"
}

variable "nfs_host" {
  default = "%s"
}

variable "nfs_path" {
  default = "%s"
}

variable "esxi_hosts" {
  default = [
    "%s",
    "%s",
    "%s",
  ]
}

variable "resource_pool" {
  default = "%s"
}

variable "network_label" {
  default = "%s"
}

data "vsphere_datacenter" "dc" {
  name = "${var.datacenter}"
}

data "vsphere_resource_pool" "pool" {
  name          = "${var.resource_pool}"
  datacenter_id = "${data.vsphere_datacenter.dc.id}"
}

data "vsphere_network" "network" {
  name          = "${var.network_label}"
  datacenter_id = "${data.vsphere_datacenter.dc.id}"
}

data "vsphere_host" "esxi_hosts" {
  count         = "${length(var.esxi_hosts)}"
  name          = "${var.esxi_hosts[count.index]}"
  datacenter_id = "${data.vsphere_datacenter.dc.id}"
}

resource "vsphere_datastore_cluster" "datastore_cluster" {
  name          = "terraform-datastore-cluster-test"
  datacenter_id = "${data.vsphere_datacenter.dc.id}"
  sdrs_enabled  = true
}

resource "vsphere_nas_datastore" "datastore" {
  name                 = "terraform-test-nas"
  host_system_ids      = "${data.vsphere_host.esxi_hosts.*.id}"
  datastore_cluster_id = "${vsphere_datastore_cluster.datastore_cluster.id}"

  type         = "NFS"
  remote_hosts = ["${var.nfs_host}"]
  remote_path  = "${var.nfs_path}"
}

resource "vsphere_virtual_machine" "vm" {
  name             = "terraform-test"
  resource_pool_id = "${data.vsphere_resource_pool.pool.id}"
  datastore_id     = "${vsphere_nas_datastore.datastore.id}"

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

resource "vsphere_storage_drs_vm_override" "drs_vm_override" {
  datastore_cluster_id = "${vsphere_datastore_cluster.datastore_cluster.id}"
  virtual_machine_id   = "${vsphere_virtual_machine.vm.id}"
  sdrs_enabled         = false
}
`,
		os.Getenv("TF_VAR_VSPHERE_DATACENTER"),
		os.Getenv("TF_VAR_VSPHERE_NAS_HOST"),
		os.Getenv("TF_VAR_VSPHERE_NFS_PATH"),
		os.Getenv("TF_VAR_VSPHERE_NFS_DS_NAME"),
		os.Getenv("TF_VAR_VSPHERE_ESXI1"),
		os.Getenv("TF_VAR_VSPHERE_ESXI2"),
		os.Getenv("TF_VAR_VSPHERE_RESOURCE_POOL"),
		os.Getenv("TF_VAR_VSPHERE_PG_NAME"),
	)
}

func testAccResourceVSphereStorageDrsVMOverrideConfigOverrides() string {
	return fmt.Sprintf(`
variable "datacenter" {
  default = "%s"
}

variable "nfs_host" {
  default = "%s"
}

variable "nfs_path" {
  default = "%s"
}

variable "esxi_hosts" {
  default = [
    "%s",
    "%s",
    "%s",
  ]
}

variable "resource_pool" {
  default = "%s"
}

variable "network_label" {
  default = "%s"
}

data "vsphere_datacenter" "dc" {
  name = "${var.datacenter}"
}

data "vsphere_resource_pool" "pool" {
  name          = "${var.resource_pool}"
  datacenter_id = "${data.vsphere_datacenter.dc.id}"
}

data "vsphere_network" "network" {
  name          = "${var.network_label}"
  datacenter_id = "${data.vsphere_datacenter.dc.id}"
}

data "vsphere_host" "esxi_hosts" {
  count         = "${length(var.esxi_hosts)}"
  name          = "${var.esxi_hosts[count.index]}"
  datacenter_id = "${data.vsphere_datacenter.dc.id}"
}

resource "vsphere_datastore_cluster" "datastore_cluster" {
  name          = "terraform-datastore-cluster-test"
  datacenter_id = "${data.vsphere_datacenter.dc.id}"
  sdrs_enabled  = true
}

resource "vsphere_nas_datastore" "datastore" {
  name                 = "terraform-test-nas"
  host_system_ids      = "${data.vsphere_host.esxi_hosts.*.id}"
  datastore_cluster_id = "${vsphere_datastore_cluster.datastore_cluster.id}"

  type         = "NFS"
  remote_hosts = ["${var.nfs_host}"]
  remote_path  = "${var.nfs_path}"
}

resource "vsphere_virtual_machine" "vm" {
  name                 = "terraform-test"
  resource_pool_id     = "${data.vsphere_resource_pool.pool.id}"
  datastore_cluster_id = "${vsphere_datastore_cluster.datastore_cluster.id}"

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

  depends_on = ["vsphere_nas_datastore.datastore"]
}

resource "vsphere_storage_drs_vm_override" "drs_vm_override" {
  datastore_cluster_id   = "${vsphere_datastore_cluster.datastore_cluster.id}"
  virtual_machine_id     = "${vsphere_virtual_machine.vm.id}"
  sdrs_automation_level  = "automated"
  sdrs_intra_vm_affinity = false
}
`,
		os.Getenv("TF_VAR_VSPHERE_DATACENTER"),
		os.Getenv("TF_VAR_VSPHERE_NAS_HOST"),
		os.Getenv("TF_VAR_VSPHERE_NFS_PATH"),
		os.Getenv("TF_VAR_VSPHERE_NFS_DS_NAME"),
		os.Getenv("TF_VAR_VSPHERE_ESXI_HOST2"),
		os.Getenv("TF_VAR_VSPHERE_ESXI_HOST3"),
		os.Getenv("TF_VAR_VSPHERE_RESOURCE_POOL"),
		os.Getenv("TF_VAR_VSPHERE_PG_NAME"),
	)
}
