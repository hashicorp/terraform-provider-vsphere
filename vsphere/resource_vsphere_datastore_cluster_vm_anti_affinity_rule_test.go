package vsphere

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"reflect"
	"sort"
	"testing"

	"github.com/davecgh/go-spew/spew"
	"github.com/hashicorp/terraform-plugin-sdk/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/terraform"
	"github.com/hashicorp/terraform-provider-vsphere/vsphere/internal/helper/structure"
	"github.com/hashicorp/terraform-provider-vsphere/vsphere/internal/helper/viapi"
	"github.com/hashicorp/terraform-provider-vsphere/vsphere/internal/helper/virtualmachine"
	"github.com/vmware/govmomi/vim25/types"
)

func TestAccResourceVSphereDatastoreClusterVMAntiAffinityRule_basic(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
			testAccResourceVSphereDatastoreClusterVMAntiAffinityRulePreCheck(t)
		},
		Providers:    testAccProviders,
		CheckDestroy: testAccResourceVSphereDatastoreClusterVMAntiAffinityRuleExists(false),
		Steps: []resource.TestStep{
			{
				Config: testAccResourceVSphereDatastoreClusterVMAntiAffinityRuleConfig(2, true),
				Check: resource.ComposeTestCheckFunc(
					testAccResourceVSphereDatastoreClusterVMAntiAffinityRuleExists(true),
					testAccResourceVSphereDatastoreClusterVMAntiAffinityRuleMatchBase(
						true,
						false,
						"terraform-test-datastore-cluster-anti-affinity-rule",
					),
					testAccResourceVSphereDatastoreClusterVMAntiAffinityRuleMatchMembership(),
				),
			},
			{
				ResourceName:      "vsphere_datastore_cluster_vm_anti_affinity_rule.cluster_vm_anti_affinity_rule",
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateIdFunc: func(s *terraform.State) (string, error) {
					pod, err := testGetDatastoreCluster(s, "datastore_cluster")
					if err != nil {
						return "", err
					}

					rs, ok := s.RootModule().Resources["vsphere_datastore_cluster_vm_anti_affinity_rule.cluster_vm_anti_affinity_rule"]
					if !ok {
						return "", errors.New("no resource at address vsphere_datastore_cluster_vm_anti_affinity_rule.cluster_vm_anti_affinity_rule")
					}
					name, ok := rs.Primary.Attributes["name"]
					if !ok {
						return "", errors.New("vsphere_datastore_cluster_vm_anti_affinity_rule.cluster_vm_anti_affinity_rule has no name attribute")
					}

					m := make(map[string]string)
					m["datastore_cluster_path"] = pod.InventoryPath
					m["name"] = name
					b, err := json.Marshal(m)
					if err != nil {
						return "", err
					}

					return string(b), nil
				},
				Config: testAccResourceVSphereDatastoreClusterVMAntiAffinityRuleConfig(2, true),
				Check: resource.ComposeTestCheckFunc(
					testAccResourceVSphereDatastoreClusterVMAntiAffinityRuleExists(true),
					testAccResourceVSphereDatastoreClusterVMAntiAffinityRuleMatchMembership(),
				),
			},
		},
	})
}

func TestAccResourceVSphereDatastoreClusterVMAntiAffinityRule_updateEnabled(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
			testAccResourceVSphereDatastoreClusterVMAntiAffinityRulePreCheck(t)
		},
		Providers:    testAccProviders,
		CheckDestroy: testAccResourceVSphereDatastoreClusterVMAntiAffinityRuleExists(false),
		Steps: []resource.TestStep{
			{
				Config: testAccResourceVSphereDatastoreClusterVMAntiAffinityRuleConfig(2, true),
				Check: resource.ComposeTestCheckFunc(
					testAccResourceVSphereDatastoreClusterVMAntiAffinityRuleExists(true),
					testAccResourceVSphereDatastoreClusterVMAntiAffinityRuleMatchBase(
						true,
						false,
						"terraform-test-datastore-cluster-anti-affinity-rule",
					),
					testAccResourceVSphereDatastoreClusterVMAntiAffinityRuleMatchMembership(),
				),
			},
			{
				Config: testAccResourceVSphereDatastoreClusterVMAntiAffinityRuleConfig(2, false),
				Check: resource.ComposeTestCheckFunc(
					testAccResourceVSphereDatastoreClusterVMAntiAffinityRuleExists(true),
					testAccResourceVSphereDatastoreClusterVMAntiAffinityRuleMatchBase(
						false,
						false,
						"terraform-test-datastore-cluster-anti-affinity-rule",
					),
					testAccResourceVSphereDatastoreClusterVMAntiAffinityRuleMatchMembership(),
				),
			},
		},
	})
}

func TestAccResourceVSphereDatastoreClusterVMAntiAffinityRule_updateCount(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
			testAccResourceVSphereDatastoreClusterVMAntiAffinityRulePreCheck(t)
		},
		Providers:    testAccProviders,
		CheckDestroy: testAccResourceVSphereDatastoreClusterVMAntiAffinityRuleExists(false),
		Steps: []resource.TestStep{
			{
				Config: testAccResourceVSphereDatastoreClusterVMAntiAffinityRuleConfig(2, true),
				Check: resource.ComposeTestCheckFunc(
					testAccResourceVSphereDatastoreClusterVMAntiAffinityRuleExists(true),
					testAccResourceVSphereDatastoreClusterVMAntiAffinityRuleMatchBase(
						true,
						false,
						"terraform-test-datastore-cluster-anti-affinity-rule",
					),
					testAccResourceVSphereDatastoreClusterVMAntiAffinityRuleMatchMembership(),
				),
			},
			{
				Config: testAccResourceVSphereDatastoreClusterVMAntiAffinityRuleConfig(3, true),
				Check: resource.ComposeTestCheckFunc(
					testAccResourceVSphereDatastoreClusterVMAntiAffinityRuleExists(true),
					testAccResourceVSphereDatastoreClusterVMAntiAffinityRuleMatchBase(
						true,
						false,
						"terraform-test-datastore-cluster-anti-affinity-rule",
					),
					testAccResourceVSphereDatastoreClusterVMAntiAffinityRuleMatchMembership(),
				),
			},
		},
	})
}

func testAccResourceVSphereDatastoreClusterVMAntiAffinityRulePreCheck(t *testing.T) {
	if os.Getenv("TF_VAR_VSPHERE_DATACENTER") == "" {
		t.Skip("set TF_VAR_VSPHERE_DATACENTER to run vsphere_datastore_cluster_vm_anti_affinity_rule acceptance tests")
	}
	if os.Getenv("TF_VAR_VSPHERE_NAS_HOST") == "" {
		t.Skip("set TF_VAR_VSPHERE_NAS_HOST to run vsphere_datastore_cluster_vm_anti_affinity_rule acceptance tests")
	}
	if os.Getenv("TF_VAR_VSPHERE_NFS_PATH") == "" {
		t.Skip("set TF_VAR_VSPHERE_NFS_PATH to run vsphere_datastore_cluster_vm_anti_affinity_rule acceptance tests")
	}
	if os.Getenv("TF_VAR_VSPHERE_NFS_DS_NAME") == "" {
		t.Skip("set TF_VAR_VSPHERE_ESXI_HOST to run vsphere_datastore_cluster_vm_anti_affinity_rule acceptance tests")
	}
	if os.Getenv("TF_VAR_VSPHERE_ESXI_HOST2") == "" {
		t.Skip("set TF_VAR_VSPHERE_ESXI_HOST2 to run vsphere_datastore_cluster_vm_anti_affinity_rule acceptance tests")
	}
	if os.Getenv("TF_VAR_VSPHERE_ESXI_HOST3") == "" {
		t.Skip("set TF_VAR_VSPHERE_ESXI_HOST3 to run vsphere_datastore_cluster_vm_anti_affinity_rule acceptance tests")
	}
	if os.Getenv("TF_VAR_VSPHERE_CLUSTER") == "" {
		t.Skip("set TF_VAR_VSPHERE_CLUSTER to run vsphere_datastore_cluster_vm_anti_affinity_rule acceptance tests")
	}
	if os.Getenv("TF_VAR_VSPHERE_PG_NAME") == "" {
		t.Skip("set TF_VAR_VSPHERE_PG_NAME to run vsphere_datastore_cluster_vm_anti_affinity_rule acceptance tests")
	}
}

func testAccResourceVSphereDatastoreClusterVMAntiAffinityRuleExists(expected bool) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		info, err := testGetDatastoreClusterVMAntiAffinityRule(s, "cluster_vm_anti_affinity_rule")
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

func testAccResourceVSphereDatastoreClusterVMAntiAffinityRuleMatchBase(
	enabled bool,
	mandatory bool,
	name string,
) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		actual, err := testGetDatastoreClusterVMAntiAffinityRule(s, "cluster_vm_anti_affinity_rule")
		if err != nil {
			return err
		}

		if actual == nil {
			return errors.New("cluster rule missing")
		}

		expected := &types.ClusterAntiAffinityRuleSpec{
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
			Vm: actual.Vm,
		}

		if !reflect.DeepEqual(expected, actual) {
			return spew.Errorf("expected %#v got %#v", expected, actual)
		}

		return nil
	}
}

func testAccResourceVSphereDatastoreClusterVMAntiAffinityRuleMatchMembership() resource.TestCheckFunc {
	return func(s *terraform.State) error {
		actual, err := testGetDatastoreClusterVMAntiAffinityRule(s, "cluster_vm_anti_affinity_rule")
		if err != nil {
			return err
		}

		if actual == nil {
			return errors.New("cluster rule missing")
		}

		vms, err := testAccResourceVSphereDatastoreClusterVMAntiAffinityRuleMatchMembershipVMIDs(s)
		if err != nil {
			return err
		}

		expectedSort := structure.MoRefSorter(vms)
		sort.Sort(expectedSort)

		expected := &types.ClusterAntiAffinityRuleSpec{
			ClusterRuleInfo: actual.ClusterRuleInfo,
			Vm:              actual.Vm,
		}

		actualSort := structure.MoRefSorter(actual.Vm)
		sort.Sort(actualSort)
		actual.Vm = []types.ManagedObjectReference(actualSort)

		if !reflect.DeepEqual(expected, actual) {
			return spew.Errorf("expected %#v got %#v", expected, actual)
		}

		return nil
	}
}

func testAccResourceVSphereDatastoreClusterVMAntiAffinityRuleMatchMembershipVMIDs(s *terraform.State) ([]types.ManagedObjectReference, error) {
	var ids []string
	if rs, ok := s.RootModule().Resources["vsphere_virtual_machine.vm"]; ok {
		ids = []string{rs.Primary.ID}
	} else {
		ids = testAccResourceVSphereDatastoreClusterVMAntiAffinityRuleGetMultiple(s)
	}

	results, err := virtualmachine.MOIDsForUUIDs(testAccProvider.Meta().(*VSphereClient).vimClient, ids)
	if err != nil {
		return nil, err
	}
	return results.ManagedObjectReferences(), nil
}

func testAccResourceVSphereDatastoreClusterVMAntiAffinityRuleGetMultiple(s *terraform.State) []string {
	var i int
	var ids []string
	for {
		rs, ok := s.RootModule().Resources[fmt.Sprintf("vsphere_virtual_machine.vm.%d", i)]
		if !ok {
			break
		}
		ids = append(ids, rs.Primary.ID)
		i++
	}
	return ids
}

func testAccResourceVSphereDatastoreClusterVMAntiAffinityRuleConfig(count int, enabled bool) string {
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
  ]
}

variable "cluster" {
  default = "%s"
}

variable "network_label" {
  default = "%s"
}

variable "vm_count" {
  default = "%d"
}

data "vsphere_datacenter" "dc" {
  name = "${var.datacenter}"
}

data "vsphere_host" "esxi_hosts" {
  count         = "${length(var.esxi_hosts)}"
  name          = "${var.esxi_hosts[count.index]}"
  datacenter_id = "${data.vsphere_datacenter.dc.id}"
}

data "vsphere_compute_cluster" "cluster" {
  name          = "${var.cluster}"
  datacenter_id = "${data.vsphere_datacenter.dc.id}"
}

data "vsphere_network" "network" {
  name          = "${var.network_label}"
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
  count                = "${var.vm_count}"
  name                 = "terraform-test-${count.index}"
  resource_pool_id     = "${data.vsphere_compute_cluster.cluster.resource_pool_id}"
  datastore_cluster_id = "${vsphere_datastore_cluster.datastore_cluster.id}"

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
  
	depends_on = ["vsphere_nas_datastore.datastore"]
}

resource "vsphere_datastore_cluster_vm_anti_affinity_rule" "cluster_vm_anti_affinity_rule" {
  name                 = "terraform-test-datastore-cluster-anti-affinity-rule"
  datastore_cluster_id = "${vsphere_datastore_cluster.datastore_cluster.id}"
  virtual_machine_ids  = "${vsphere_virtual_machine.vm.*.id}"
  enabled              = %t
}
`,
		os.Getenv("TF_VAR_VSPHERE_DATACENTER"),
		os.Getenv("TF_VAR_VSPHERE_NAS_HOST"),
		os.Getenv("TF_VAR_VSPHERE_NFS_PATH"),
		os.Getenv("TF_VAR_VSPHERE_ESXI1"),
		os.Getenv("TF_VAR_VSPHERE_ESXI2"),
		os.Getenv("TF_VAR_VSPHERE_CLUSTER"),
		os.Getenv("TF_VAR_VSPHERE_PG_NAME"),
		count,
		enabled,
	)
}
