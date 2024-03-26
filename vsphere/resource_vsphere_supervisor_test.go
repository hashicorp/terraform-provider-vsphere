// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package vsphere

import (
	"fmt"
	"github.com/hashicorp/terraform-provider-vsphere/vsphere/internal/helper/testhelper"
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestAccResourceVSphereSupervisor_basic(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccCheckEnvVariables(t, []string{
				"TF_VAR_STORAGE_POLICY",
				"TF_VAR_MANAGEMENT_NETWORK",
				"TF_VAR_CONTENT_LIBRARY",
				"TF_VAR_DISTRIBUTED_SWITCH",
				"TF_VAR_EDGE_CLUSTER",
			})
		},
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				// You can change the network settings in the configuration
				// so that they fit your environment
				Config: testAccVSphereSupervisorConfig(),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("vsphere_supervisor.supervisor", "id"),
				),
			},
		},
	})
}

func testAccVSphereSupervisorConfig() string {
	return fmt.Sprintf(`
%s

data vsphere_storage_policy image_policy {
	name = "%s"
}

data vsphere_network mgmt_net {
	name = "%s"
	datacenter_id = "${data.vsphere_datacenter.rootdc1.id}"
}

data vsphere_content_library subscribed_lib {
	name = "%s"
}

data vsphere_distributed_virtual_switch dvs {
	name = "%s"
	datacenter_id = "${data.vsphere_datacenter.rootdc1.id}"
}

resource "vsphere_supervisor" "supervisor" {
	cluster = "${data.vsphere_compute_cluster.rootcompute_cluster1.id}"
	storage_policy = "${data.vsphere_storage_policy.image_policy.id}"
	content_library = "${data.vsphere_content_library.subscribed_lib.id}"
	main_dns = "10.0.0.250"
	worker_dns = "10.0.0.250"
	edge_cluster = "%s"
	dvs_uuid = "${data.vsphere_distributed_virtual_switch.dvs.id}"
	sizing_hint = "MEDIUM"
	
	management_network {
		network = "${data.vsphere_network.mgmt_net.id}"
		subnet_mask = "255.255.255.0"
		starting_address = "10.0.0.150"
		gateway = "10.0.0.250"
		address_count = 5
	}

	ingress_cidr {
		address = "10.10.10.0"
		prefix = 24
	}

	egress_cidr {
		address = "10.10.11.0"
		prefix = 24
	}

	pod_cidr {
		address = "10.244.10.0"
		prefix = 23
	}

	service_cidr {
		address = "10.10.12.0"
		prefix = 24
	}

	search_domains = [ "vrack.vsphere.local" ]

	namespace {
		name = "test-namespace-03"
		content_libraries = [ "${data.vsphere_content_library.subscribed_lib.id}" ]
		vm_class {
			id = "test-class-04"
			cpus = 4
			memory = 4096
		}
		vm_class {
			id = "test-class-05"
			cpus = 4
			memory = 4096
		}
	}
}
`,
		testAccConfigBase(),
		os.Getenv("TF_VAR_STORAGE_POLICY"),
		os.Getenv("TF_VAR_MANAGEMENT_NETWORK"),
		os.Getenv("TF_VAR_CONTENT_LIBRARY"),
		os.Getenv("TF_VAR_DISTRIBUTED_SWITCH"),
		os.Getenv("TF_VAR_EDGE_CLUSTER"))
}

func testAccConfigBase() string {
	return testhelper.CombineConfigs(testhelper.ConfigDataRootDC1(), testhelper.ConfigDataRootComputeCluster1())
}
