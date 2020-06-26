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

func TestAccResourceVSphereComputeClusterVMHostRule_basic(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
			testAccResourceVSphereComputeClusterVMHostRulePreCheck(t)
		},
		Providers:    testAccProviders,
		CheckDestroy: testAccResourceVSphereComputeClusterVMHostRuleExists(false),
		Steps: []resource.TestStep{
			{
				Config: testAccResourceVSphereComputeClusterVMHostRuleConfigAffinity(),
				Check: resource.ComposeTestCheckFunc(
					testAccResourceVSphereComputeClusterVMHostRuleExists(true),
					testAccResourceVSphereComputeClusterVMHostRuleMatch(
						true,
						false,
						"terraform-test-cluster-vm-host-rule",
						"terraform-test-cluster-host-group",
						"",
						"terraform-test-cluster-vm-group",
					),
				),
			},
			{
				ResourceName:      "vsphere_compute_cluster_vm_host_rule.cluster_vm_host_rule",
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateIdFunc: func(s *terraform.State) (string, error) {
					cluster, err := testGetComputeCluster(s, "cluster")
					if err != nil {
						return "", err
					}

					rs, ok := s.RootModule().Resources["vsphere_compute_cluster_vm_host_rule.cluster_vm_host_rule"]
					if !ok {
						return "", errors.New("no resource at address vsphere_compute_cluster_vm_host_rule.cluster_vm_host_rule")
					}
					name, ok := rs.Primary.Attributes["name"]
					if !ok {
						return "", errors.New("vsphere_compute_cluster_vm_host_rule.cluster_vm_host_rule has no name attribute")
					}

					m := make(map[string]string)
					m["compute_cluster_path"] = cluster.InventoryPath
					m["name"] = name
					b, err := json.Marshal(m)
					if err != nil {
						return "", err
					}

					return string(b), nil
				},
				Config: testAccResourceVSphereComputeClusterVMHostRuleConfigAffinity(),
				Check: resource.ComposeTestCheckFunc(
					testAccResourceVSphereComputeClusterVMHostRuleExists(true),
					testAccResourceVSphereComputeClusterVMHostRuleMatch(
						true,
						false,
						"terraform-test-cluster-vm-host-rule",
						"terraform-test-cluster-host-group",
						"",
						"terraform-test-cluster-vm-group",
					),
				),
			},
		},
	})
}

func TestAccResourceVSphereComputeClusterVMHostRule_antiAffinity(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
			testAccResourceVSphereComputeClusterVMHostRulePreCheck(t)
		},
		Providers:    testAccProviders,
		CheckDestroy: testAccResourceVSphereComputeClusterVMHostRuleExists(false),
		Steps: []resource.TestStep{
			{
				Config: testAccResourceVSphereComputeClusterVMHostRuleConfigAntiAffinity(),
				Check: resource.ComposeTestCheckFunc(
					testAccResourceVSphereComputeClusterVMHostRuleExists(true),
					testAccResourceVSphereComputeClusterVMHostRuleMatch(
						true,
						false,
						"terraform-test-cluster-vm-host-rule",
						"",
						"terraform-test-cluster-host-group",
						"terraform-test-cluster-vm-group",
					),
				),
			},
		},
	})
}

func TestAccResourceVSphereComputeClusterVMHostRule_updateEnabled(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
			testAccResourceVSphereComputeClusterVMHostRulePreCheck(t)
		},
		Providers:    testAccProviders,
		CheckDestroy: testAccResourceVSphereComputeClusterVMHostRuleExists(false),
		Steps: []resource.TestStep{
			{
				Config: testAccResourceVSphereComputeClusterVMHostRuleConfigAffinity(),
				Check: resource.ComposeTestCheckFunc(
					testAccResourceVSphereComputeClusterVMHostRuleExists(true),
					testAccResourceVSphereComputeClusterVMHostRuleMatch(
						true,
						false,
						"terraform-test-cluster-vm-host-rule",
						"terraform-test-cluster-host-group",
						"",
						"terraform-test-cluster-vm-group",
					),
				),
			},
			{
				Config: testAccResourceVSphereComputeClusterVMHostRuleConfigDisabled(),
				Check: resource.ComposeTestCheckFunc(
					testAccResourceVSphereComputeClusterVMHostRuleExists(true),
					testAccResourceVSphereComputeClusterVMHostRuleMatch(
						false,
						false,
						"terraform-test-cluster-vm-host-rule",
						"terraform-test-cluster-host-group",
						"",
						"terraform-test-cluster-vm-group",
					),
				),
			},
		},
	})
}

func TestAccResourceVSphereComputeClusterVMHostRule_updateAffinity(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
			testAccResourceVSphereComputeClusterVMHostRulePreCheck(t)
		},
		Providers:    testAccProviders,
		CheckDestroy: testAccResourceVSphereComputeClusterVMHostRuleExists(false),
		Steps: []resource.TestStep{
			{
				Config: testAccResourceVSphereComputeClusterVMHostRuleConfigAffinity(),
				Check: resource.ComposeTestCheckFunc(
					testAccResourceVSphereComputeClusterVMHostRuleExists(true),
					testAccResourceVSphereComputeClusterVMHostRuleMatch(
						true,
						false,
						"terraform-test-cluster-vm-host-rule",
						"terraform-test-cluster-host-group",
						"",
						"terraform-test-cluster-vm-group",
					),
				),
			},
			{
				Config: testAccResourceVSphereComputeClusterVMHostRuleConfigAntiAffinity(),
				Check: resource.ComposeTestCheckFunc(
					testAccResourceVSphereComputeClusterVMHostRuleExists(true),
					testAccResourceVSphereComputeClusterVMHostRuleMatch(
						true,
						false,
						"terraform-test-cluster-vm-host-rule",
						"",
						"terraform-test-cluster-host-group",
						"terraform-test-cluster-vm-group",
					),
				),
			},
		},
	})
}

func testAccResourceVSphereComputeClusterVMHostRulePreCheck(t *testing.T) {
	if os.Getenv("TF_VAR_VSPHERE_DATACENTER") == "" {
		t.Skip("set TF_VAR_VSPHERE_DATACENTER to run vsphere_compute_cluster_vm_host_rule acceptance tests")
	}
	if os.Getenv("TF_VAR_VSPHERE_ESXI1") == "" {
		t.Skip("set TF_VAR_VSPHERE_ESXI1 to run vsphere_compute_cluster_vm_host_rule acceptance tests")
	}
	if os.Getenv("TF_VAR_VSPHERE_ESXI2") == "" {
		t.Skip("set TF_VAR_VSPHERE_ESXI2 to run vsphere_compute_cluster_vm_host_rule acceptance tests")
	}
	if os.Getenv("TF_VAR_VSPHERE_NFS_DS_NAME") == "" {
		t.Skip("set TF_VAR_VSPHERE_NFS_DS_NAME to run vsphere_compute_cluster_vm_host_rule acceptance tests")
	}
	if os.Getenv("TF_VAR_VSPHERE_PG_NAME") == "" {
		t.Skip("set TF_VAR_VSPHERE_PG_NAME to run vsphere_compute_cluster_vm_host_rule acceptance tests")
	}
}

func testAccResourceVSphereComputeClusterVMHostRuleExists(expected bool) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		info, err := testGetComputeClusterVMHostRule(s, "cluster_vm_host_rule")
		if err != nil {
			if expected == false {
				if viapi.IsManagedObjectNotFoundError(err) {
					// This is not necessarily a missing rule, but more than likely a
					// missing cluster, which happens during destroy as the dependent
					// resources will be missing as well, so want to treat this as a
					// deleted rule as well.
					return nil
				}
			}
			return err
		}

		switch {
		case info == nil && !expected:
			// Expected missing
			return nil
		case info == nil && expected:
			// Expected to exist
			return errors.New("cluster rule missing when expected to exist")
		case !expected:
			return errors.New("cluster rule still present when expected to be missing")
		}

		return nil
	}
}

func testAccResourceVSphereComputeClusterVMHostRuleMatch(
	enabled bool,
	mandatory bool,
	name string,
	affinityGroup string,
	antiAffinityGroup string,
	vmGroup string,
) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		actual, err := testGetComputeClusterVMHostRule(s, "cluster_vm_host_rule")
		if err != nil {
			return err
		}

		if actual == nil {
			return errors.New("cluster rule missing")
		}

		expected := &types.ClusterVmHostRuleInfo{
			ClusterRuleInfo: types.ClusterRuleInfo{
				Enabled:      structure.BoolPtr(enabled),
				Mandatory:    structure.BoolPtr(mandatory),
				Name:         name,
				UserCreated:  structure.BoolPtr(true),
				InCompliance: actual.InCompliance,
				Key:          actual.Key,
				RuleUuid:     actual.RuleUuid,
				Status:       actual.Status,
			},
			AffineHostGroupName:     affinityGroup,
			AntiAffineHostGroupName: antiAffinityGroup,
			VmGroupName:             vmGroup,
		}

		if !reflect.DeepEqual(expected, actual) {
			return spew.Errorf("expected %#v got %#v", expected, actual)
		}

		return nil
	}
}

func testAccResourceVSphereComputeClusterVMHostRuleConfigAffinity() string {
	return fmt.Sprintf(`
variable "datacenter" {
  default = "%s"
}

variable "hosts" {
  default = [
    "%s",
    "%s",
  ]
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

data "vsphere_host" "hosts" {
	count         = "${length(var.hosts)}"
  name          = "${var.hosts[count.index]}"
  datacenter_id = "${data.vsphere_datacenter.dc.id}"
}

resource "vsphere_compute_cluster" "cluster" {
  name            = "terraform-compute-cluster-test"
  datacenter_id   = "${data.vsphere_datacenter.dc.id}"
  host_system_ids = "${data.vsphere_host.hosts.*.id}"

  force_evacuate_on_destroy = true
}

resource "vsphere_virtual_machine" "vm" {
  name             = "terraform-test"
  resource_pool_id = "${vsphere_compute_cluster.cluster.resource_pool_id}"
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
}

resource "vsphere_compute_cluster_host_group" "cluster_host_group" {
  name               = "terraform-test-cluster-host-group"
  compute_cluster_id = "${vsphere_compute_cluster.cluster.id}"
  host_system_ids    = "${data.vsphere_host.hosts.*.id}"
}

resource "vsphere_compute_cluster_vm_group" "cluster_vm_group" {
  name                = "terraform-test-cluster-vm-group"
  compute_cluster_id  = "${vsphere_compute_cluster.cluster.id}"
  virtual_machine_ids = "${vsphere_virtual_machine.vm.*.id}"
}

resource "vsphere_compute_cluster_vm_host_rule" "cluster_vm_host_rule" {
  compute_cluster_id       = "${vsphere_compute_cluster.cluster.id}"
  name                     = "terraform-test-cluster-vm-host-rule"
  vm_group_name            = "${vsphere_compute_cluster_vm_group.cluster_vm_group.name}"
  affinity_host_group_name = "${vsphere_compute_cluster_host_group.cluster_host_group.name}"
}
`,
		os.Getenv("TF_VAR_VSPHERE_DATACENTER"),
		os.Getenv("TF_VAR_VSPHERE_ESXI1"),
		os.Getenv("TF_VAR_VSPHERE_ESXI2"),
		os.Getenv("TF_VAR_VSPHERE_NFS_DS_NAME"),
		os.Getenv("TF_VAR_VSPHERE_PG_NAME"),
	)
}

func testAccResourceVSphereComputeClusterVMHostRuleConfigAntiAffinity() string {
	return fmt.Sprintf(`
variable "datacenter" {
  default = "%s"
}

variable "hosts" {
  default = [
    "%s",
    "%s",
  ]
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

data "vsphere_host" "hosts" {
	count         = "${length(var.hosts)}"
  name          = "${var.hosts[count.index]}"
  datacenter_id = "${data.vsphere_datacenter.dc.id}"
}

resource "vsphere_compute_cluster" "cluster" {
  name            = "terraform-compute-cluster-test"
  datacenter_id   = "${data.vsphere_datacenter.dc.id}"
  host_system_ids = "${data.vsphere_host.hosts.*.id}"

  force_evacuate_on_destroy = true
}

resource "vsphere_virtual_machine" "vm" {
  name             = "terraform-test"
  resource_pool_id = "${vsphere_compute_cluster.cluster.resource_pool_id}"
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
}

resource "vsphere_compute_cluster_host_group" "cluster_host_group" {
  name               = "terraform-test-cluster-host-group"
  compute_cluster_id = "${vsphere_compute_cluster.cluster.id}"
  host_system_ids    = "${data.vsphere_host.hosts.*.id}"
}

resource "vsphere_compute_cluster_vm_group" "cluster_vm_group" {
  name                = "terraform-test-cluster-vm-group"
  compute_cluster_id  = "${vsphere_compute_cluster.cluster.id}"
  virtual_machine_ids = "${vsphere_virtual_machine.vm.*.id}"
}

resource "vsphere_compute_cluster_vm_host_rule" "cluster_vm_host_rule" {
  compute_cluster_id            = "${vsphere_compute_cluster.cluster.id}"
  name                          = "terraform-test-cluster-vm-host-rule"
  vm_group_name                 = "${vsphere_compute_cluster_vm_group.cluster_vm_group.name}"
  anti_affinity_host_group_name = "${vsphere_compute_cluster_host_group.cluster_host_group.name}"
}
`,
		os.Getenv("TF_VAR_VSPHERE_DATACENTER"),
		os.Getenv("TF_VAR_VSPHERE_ESXI1"),
		os.Getenv("TF_VAR_VSPHERE_ESXI2"),
		os.Getenv("TF_VAR_VSPHERE_NFS_DS_NAME"),
		os.Getenv("TF_VAR_VSPHERE_PG_NAME"),
	)
}

func testAccResourceVSphereComputeClusterVMHostRuleConfigDisabled() string {
	return fmt.Sprintf(`
variable "datacenter" {
  default = "%s"
}

variable "hosts" {
  default = [
    "%s",
    "%s",
  ]
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

data "vsphere_host" "hosts" {
	count         = "${length(var.hosts)}"
  name          = "${var.hosts[count.index]}"
  datacenter_id = "${data.vsphere_datacenter.dc.id}"
}

resource "vsphere_compute_cluster" "cluster" {
  name            = "terraform-compute-cluster-test"
  datacenter_id   = "${data.vsphere_datacenter.dc.id}"
  host_system_ids = "${data.vsphere_host.hosts.*.id}"

  force_evacuate_on_destroy = true
}

resource "vsphere_virtual_machine" "vm" {
  name             = "terraform-test"
  resource_pool_id = "${vsphere_compute_cluster.cluster.resource_pool_id}"
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
}

resource "vsphere_compute_cluster_host_group" "cluster_host_group" {
  name               = "terraform-test-cluster-host-group"
  compute_cluster_id = "${vsphere_compute_cluster.cluster.id}"
  host_system_ids    = "${data.vsphere_host.hosts.*.id}"
}

resource "vsphere_compute_cluster_vm_group" "cluster_vm_group" {
  name                = "terraform-test-cluster-vm-group"
  compute_cluster_id  = "${vsphere_compute_cluster.cluster.id}"
  virtual_machine_ids = "${vsphere_virtual_machine.vm.*.id}"
}

resource "vsphere_compute_cluster_vm_host_rule" "cluster_vm_host_rule" {
  compute_cluster_id       = "${vsphere_compute_cluster.cluster.id}"
  name                     = "terraform-test-cluster-vm-host-rule"
  vm_group_name            = "${vsphere_compute_cluster_vm_group.cluster_vm_group.name}"
  affinity_host_group_name = "${vsphere_compute_cluster_host_group.cluster_host_group.name}"
  enabled                  = false
}
`,
		os.Getenv("TF_VAR_VSPHERE_DATACENTER"),
		os.Getenv("TF_VAR_VSPHERE_ESXI1"),
		os.Getenv("TF_VAR_VSPHERE_ESXI2"),
		os.Getenv("TF_VAR_VSPHERE_NFS_DS_NAME"),
		os.Getenv("TF_VAR_VSPHERE_PG_NAME"),
	)
}
