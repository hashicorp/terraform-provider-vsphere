// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package vsphere

import (
	"fmt"
	"github.com/hashicorp/terraform-provider-vsphere/vsphere/internal/helper/testhelper"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestAccResourceVSphereSupervisor_basic(t *testing.T) {
	resource.Test(t, resource.TestCase{
		//PreCheck: func() {
		//	RunSweepers()
		//	testAccPreCheck(t)
		//	testAccCheckEnvVariables(t, []string{"ESX_HOSTNAME", "ESX_USERNAME", "ESX_PASSWORD"})
		//},
		Providers: testAccProviders,
		//CheckDestroy: testAccVSphereHostDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccVSphereSupervisorConfig(),
				Check:  resource.ComposeTestCheckFunc(),
			},
		},
	})
}

func testAccVSphereSupervisorConfig() string {
	return fmt.Sprintf(`
%s

data vsphere_storage_policy image_policy {
	name = "vi-cluster1 vSAN Storage Policy"
}

data vsphere_network mgmt_net {
  name = "vi-cluster1-vds-Mgmt-DVPG"
  datacenter_id = "${data.vsphere_datacenter.rootdc1.id}"
}

resource "vsphere_supervisor" "supervisor" {
	cluster = "${data.vsphere_compute_cluster.rootcompute_cluster1.id}"
	storage_policy = "${data.vsphere_storage_policy.image_policy.id}"
	content_library = "22585f3f-68a0-4c3d-8bab-466223c6699e"
	dns = "10.0.0.250"
	edge_cluster = "1c620ec9-66b7-4e8e-b4d9-92bd3c968e91"
	dvs_uuid = "50 05 6c b3 5e 8c a2 a0-0f b5 9a fa 86 2b 91 23"
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
}
`,
		testAccConfigBase())
}

func testAccConfigBase() string {
	return testhelper.CombineConfigs(testhelper.ConfigDataRootDC1(), testhelper.ConfigDataRootComputeCluster1())
}
