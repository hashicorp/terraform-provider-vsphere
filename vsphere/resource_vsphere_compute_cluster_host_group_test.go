package vsphere

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/hashicorp/terraform-provider-vsphere/vsphere/internal/helper/testhelper"
	"os"
	"reflect"
	"sort"
	"testing"

	"github.com/davecgh/go-spew/spew"
	"github.com/hashicorp/terraform-plugin-sdk/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/terraform"
	"github.com/hashicorp/terraform-provider-vsphere/vsphere/internal/helper/structure"
	"github.com/hashicorp/terraform-provider-vsphere/vsphere/internal/helper/viapi"
	"github.com/vmware/govmomi/vim25/types"
)

func TestAccResourceVSphereComputeClusterHostGroup_basic(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			RunSweepers()
			testAccPreCheck(t)
			testAccResourceVSphereComputeClusterHostGroupPreCheck(t)
		},
		Providers:    testAccProviders,
		CheckDestroy: testAccResourceVSphereComputeClusterHostGroupExists(false),
		Steps: []resource.TestStep{
			{
				Config: testAccResourceVSphereComputeClusterHostGroupConfig(2),
				Check: resource.ComposeTestCheckFunc(
					testAccResourceVSphereComputeClusterHostGroupExists(true),
					testAccResourceVSphereComputeClusterHostGroupMatchMembership(),
				),
			},
			{
				ResourceName:      "vsphere_compute_cluster_host_group.cluster_host_group",
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateIdFunc: func(s *terraform.State) (string, error) {
					cluster, err := testGetComputeCluster(s, "cluster")
					if err != nil {
						return "", err
					}

					rs, ok := s.RootModule().Resources["vsphere_compute_cluster_host_group.cluster_host_group"]
					if !ok {
						return "", errors.New("no resource at address vsphere_compute_cluster_host_group.cluster_host_group")
					}
					name, ok := rs.Primary.Attributes["name"]
					if !ok {
						return "", errors.New("vsphere_compute_cluster_host_group.cluster_host_group has no name attribute")
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
				Config: testAccResourceVSphereComputeClusterHostGroupConfig(1),
				Check: resource.ComposeTestCheckFunc(
					testAccResourceVSphereComputeClusterHostGroupExists(true),
					testAccResourceVSphereComputeClusterHostGroupMatchMembership(),
				),
			},
		},
	})
}

func TestAccResourceVSphereComputeClusterHostGroup_update(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			RunSweepers()
			testAccPreCheck(t)
			testAccResourceVSphereComputeClusterHostGroupPreCheck(t)
		},
		Providers:    testAccProviders,
		CheckDestroy: testAccResourceVSphereComputeClusterHostGroupExists(false),
		Steps: []resource.TestStep{
			{
				Config: testAccResourceVSphereComputeClusterHostGroupConfig(1),
				Check: resource.ComposeTestCheckFunc(
					testAccResourceVSphereComputeClusterHostGroupExists(true),
					testAccResourceVSphereComputeClusterHostGroupMatchMembership(),
				),
			},
			{
				Config: testAccResourceVSphereComputeClusterHostGroupConfig(2),
				Check: resource.ComposeTestCheckFunc(
					testAccResourceVSphereComputeClusterHostGroupExists(true),
					testAccResourceVSphereComputeClusterHostGroupMatchMembership(),
				),
			},
		},
	})
}

func testAccResourceVSphereComputeClusterHostGroupPreCheck(t *testing.T) {
	if os.Getenv("TF_VAR_VSPHERE_DATACENTER") == "" {
		t.Skip("set TF_VAR_VSPHERE_DATACENTER to run vsphere_compute_cluster_host_group acceptance tests")
	}
	if os.Getenv("TF_VAR_VSPHERE_ESXI1") == "" {
		t.Skip("set TF_VAR_VSPHERE_ESXI1 to run vsphere_compute_cluster_host_group acceptance tests")
	}
	if os.Getenv("TF_VAR_VSPHERE_ESXI2") == "" {
		t.Skip("set TF_VAR_VSPHERE_ESXI2 to run vsphere_compute_cluster_host_group acceptance tests")
	}
}

func testAccResourceVSphereComputeClusterHostGroupExists(expected bool) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		info, err := testGetComputeClusterHostGroup(s, "cluster_host_group")
		if err != nil {
			if expected == false {
				if viapi.IsManagedObjectNotFoundError(err) {
					// This is not necessarily a missing group, but more than likely a
					// missing cluster, which happens during destroy as the dependent
					// resources will be missing as well, so want to treat this as a
					// deleted override as well.
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
			return errors.New("cluster host group missing when expected to exist")
		case !expected:
			return errors.New("cluster host group still present when expected to be missing")
		}

		return nil
	}
}

func testAccResourceVSphereComputeClusterHostGroupMatchMembership() resource.TestCheckFunc {
	return func(s *terraform.State) error {
		actual, err := testGetComputeClusterHostGroup(s, "cluster_host_group")
		if err != nil {
			return err
		}

		if actual == nil {
			return errors.New("cluster host group missing")
		}

		hosts, err := testAccResourceVSphereComputeClusterHostGroupMatchMembershipHostIDs(s)
		if err != nil {
			return err
		}

		expectedSort := structure.MoRefSorter(hosts)
		sort.Sort(expectedSort)

		expected := &types.ClusterHostGroup{
			ClusterGroupInfo: types.ClusterGroupInfo{
				Name:        actual.Name,
				UserCreated: actual.UserCreated,
			},
			Host: []types.ManagedObjectReference(expectedSort),
		}

		actualSort := structure.MoRefSorter(actual.Host)
		sort.Sort(actualSort)
		actual.Host = []types.ManagedObjectReference(actualSort)

		if !reflect.DeepEqual(expected, actual) {
			return spew.Errorf("expected %#v got %#v", expected, actual)
		}

		return nil
	}
}

func testAccResourceVSphereComputeClusterHostGroupMatchMembershipHostIDs(s *terraform.State) ([]types.ManagedObjectReference, error) {
	var ids []string
	if rs, ok := s.RootModule().Resources["data.vsphere_host.hosts"]; ok {
		ids = []string{rs.Primary.ID}
	} else {
		ids = testAccResourceVSphereComputeClusterHostGroupGetMultiple(s)
	}

	return structure.SliceStringsToManagedObjectReferences(ids, "HostSystem"), nil
}

func testAccResourceVSphereComputeClusterHostGroupGetMultiple(s *terraform.State) []string {
	var i int
	var ids []string
	for {
		rs, ok := s.RootModule().Resources[fmt.Sprintf("data.vsphere_host.hosts.%d", i)]
		if !ok {
			break
		}
		ids = append(ids, rs.Primary.ID)
		i++
	}
	return ids
}

func testAccResourceVSphereComputeClusterHostGroupConfig(count int) string {
	return fmt.Sprintf(`
%s

data "vsphere_host" "hosts" {
  count         = 1
  name          = vsphere_host.nested-esxi1.hostname
  datacenter_id = data.vsphere_datacenter.rootdc1.id
}

resource "vsphere_compute_cluster" "cluster" {
  name            = "testacc-compute-cluster"
  datacenter_id   = "${data.vsphere_datacenter.rootdc1.id}"
  host_system_ids = "${data.vsphere_host.hosts.*.id}"

  force_evacuate_on_destroy = true
}

resource "vsphere_compute_cluster_host_group" "cluster_host_group" {
  name               = "terraform-test-cluster-group"
  compute_cluster_id = "${vsphere_compute_cluster.cluster.id}"
  host_system_ids    = "${data.vsphere_host.hosts.*.id}"
}
`,
		testhelper.CombineConfigs(testhelper.ConfigDataRootDC1(), testhelper.ConfigDataRootPortGroup1(), testhelper.ConfigResNestedEsxi(), testhelper.ConfigDataRootDS1(), testhelper.ConfigDataRootHost2(), testhelper.ConfigDataRootComputeCluster1(), testhelper.ConfigDataRootVMNet()),
	)
}
