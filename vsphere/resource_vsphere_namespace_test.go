// © Broadcom. All Rights Reserved.
// The term "Broadcom" refers to Broadcom Inc. and/or its subsidiaries.
// SPDX-License-Identifier: MPL-2.0

package vsphere

import (
	"fmt"
	"os"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/vmware/terraform-provider-vsphere/vsphere/internal/helper/testhelper"
)

func TestAccResourceVSphereNamespace_basic(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
			testAccCheckEnvVariables(t, []string{"TF_VAR_SUPERVISOR", "TF_VAR_STORAGE_POLICY"})
		},
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccResourceVSphereNamespaceConfigBasic(),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("vsphere_namespace.namespace", "id"),
					resource.TestCheckResourceAttrSet("vsphere_namespace.namespace", "supervisor"),
				),
			},
		},
	})
}

func TestAccResourceVSphereNamespace_storage(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
			testAccCheckEnvVariables(t, []string{"TF_VAR_SUPERVISOR", "TF_VAR_STORAGE_POLICY"})
		},
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccResourceVSphereNamespaceConfigStorage(0),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("vsphere_namespace.namespace", "id"),
					resource.TestCheckResourceAttrSet("vsphere_namespace.namespace", "supervisor"),
					resource.TestCheckResourceAttr("vsphere_namespace.namespace", "storage.#", "1"),
					resource.TestCheckResourceAttr("vsphere_namespace.namespace", "storage.0.limit", "0"),
				),
			},
			{
				Config: testAccResourceVSphereNamespaceConfigStorage(1024),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("vsphere_namespace.namespace", "storage.#", "1"),
					resource.TestCheckResourceAttr("vsphere_namespace.namespace", "storage.0.limit", "1024"),
				),
			},
			{
				Config: testAccResourceVSphereNamespaceConfigStorage(0),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("vsphere_namespace.namespace", "storage.#", "1"),
					resource.TestCheckResourceAttr("vsphere_namespace.namespace", "storage.0.limit", "0"),
				),
			},
		},
	})
}

func TestAccResourceVSphereNamespace_access(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
			testAccCheckEnvVariables(t, []string{"TF_VAR_SUPERVISOR"})
		},
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccResourceVSphereNamespaceConfigAccess("EDIT"),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("vsphere_namespace.namespace", "access.#", "1"),
					resource.TestCheckTypeSetElemNestedAttrs("vsphere_namespace.namespace", "access.*", map[string]string{
						"subject_type": "USER",
						"subject":      "acc-namespace-user",
						"role":         "EDIT",
					}),
				),
			},
			{
				Config: testAccResourceVSphereNamespaceConfigAccess("VIEW"),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("vsphere_namespace.namespace", "access.#", "1"),
					resource.TestCheckTypeSetElemNestedAttrs("vsphere_namespace.namespace", "access.*", map[string]string{
						"role": "VIEW",
					}),
				),
			},
		},
	})
}

func TestAccResourceVSphereNamespace_zones(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
			testAccCheckEnvVariables(t, []string{"TF_VAR_SUPERVISOR", "TF_VAR_NAMESPACE_ZONE"})
		},
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccResourceVSphereNamespaceConfigZones(),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("vsphere_namespace.namespace", "zone.#", "1"),
					resource.TestCheckTypeSetElemNestedAttrs("vsphere_namespace.namespace", "zone.*", map[string]string{
						"name": os.Getenv("TF_VAR_NAMESPACE_ZONE"),
					}),
				),
			},
		},
	})
}

func TestAccResourceVSphereNamespace_limits(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
			testAccCheckEnvVariables(t, []string{"TF_VAR_SUPERVISOR", "TF_VAR_STORAGE_POLICY", "TF_VAR_NAMESPACE_ZONE"})
		},
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccResourceVSphereNamespaceConfigLimits(),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("vsphere_namespace.namespace", "zone.#", "1"),
					resource.TestCheckTypeSetElemNestedAttrs("vsphere_namespace.namespace", "zone.*", map[string]string{
						"name":               os.Getenv("TF_VAR_NAMESPACE_ZONE"),
						"cpu_limit":          "2000",
						"cpu_reservation":    "1000",
						"memory_limit":       "4096",
						"memory_reservation": "2048",
					}),
					resource.TestCheckResourceAttr("vsphere_namespace.namespace", "storage.#", "1"),
					resource.TestCheckResourceAttr("vsphere_namespace.namespace", "storage.0.limit", "1024"),
				),
			},
		},
	})
}

func TestAccResourceVSphereNamespace_vpcNetwork(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
			testAccCheckEnvVariables(t, []string{"TF_VAR_SUPERVISOR"})
		},
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccResourceVSphereNamespaceConfigVpcNetwork(false),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("vsphere_namespace.namespace", "vpc_network.0.vpc"),
					resource.TestCheckResourceAttr("vsphere_namespace.namespace", "vpc_network.0.default_subnet_size", "64"),
					resource.TestCheckResourceAttr("vsphere_namespace.namespace", "vpc_network.0.vpc_config.0.private_cidrs.#", "1"),
				),
			},
			{
				Config: testAccResourceVSphereNamespaceConfigVpcNetwork(true),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("vsphere_namespace.namespace", "vpc_network.0.default_subnet_size", "64"),
					resource.TestCheckResourceAttr("vsphere_namespace.namespace", "vpc_network.0.vpc_config.0.private_cidrs.#", "2"),
				),
			},
		},
	})
}

func TestAccResourceVSphereNamespace_vpcSharedSubnets(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
			testAccCheckEnvVariables(t, []string{
				"TF_VAR_SUPERVISOR",
				"TF_VAR_NAMESPACE_VPC_PATH",
				"TF_VAR_NAMESPACE_SHARED_SUBNET_PATH",
			})
		},
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccResourceVSphereNamespaceConfigVpcSharedSubnets(true),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("vsphere_namespace.namespace", "vpc_network.0.vpc", os.Getenv("TF_VAR_NAMESPACE_VPC_PATH")),
					resource.TestCheckResourceAttr("vsphere_namespace.namespace", "vpc_network.0.shared_subnet.#", "1"),
					resource.TestCheckTypeSetElemNestedAttrs("vsphere_namespace.namespace", "vpc_network.0.shared_subnet.*", map[string]string{
						"path":        os.Getenv("TF_VAR_NAMESPACE_SHARED_SUBNET_PATH"),
						"pod_default": "true",
						"vm_default":  "true",
					}),
				),
			},
			{
				Config: testAccResourceVSphereNamespaceConfigVpcSharedSubnets(false),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckTypeSetElemNestedAttrs("vsphere_namespace.namespace", "vpc_network.0.shared_subnet.*", map[string]string{
						"pod_default": "true",
						"vm_default":  "false",
					}),
				),
			},
		},
	})
}

func TestAccResourceVSphereNamespace_vsphereNetwork(t *testing.T) {
	// Cannot run on the same supervisor as the VPC-based configurations
	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
			testAccCheckEnvVariables(t, []string{"TF_VAR_SUPERVISOR"})
		},
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccResourceVSphereNamespaceConfigVsphereNetwork(),
				Check: resource.ComposeTestCheckFunc(
					resource.TestMatchResourceAttr("data.vsphere_supervisor.supervisor", "workload_network.1.portgroup", regexp.MustCompile(`^dvportgroup-`)),
					resource.TestCheckResourceAttrPair("vsphere_namespace.namespace", "vsphere_network.0.default_workload_network", "data.vsphere_supervisor.supervisor", "workload_network.0.network"),
					resource.TestCheckResourceAttr("vsphere_namespace.namespace", "vsphere_network.0.networks.#", "2"),
					resource.TestCheckTypeSetElemAttrPair("vsphere_namespace.namespace", "vsphere_network.0.networks.*", "data.vsphere_supervisor.supervisor", "workload_network.0.network"),
					resource.TestCheckTypeSetElemAttrPair("vsphere_namespace.namespace", "vsphere_network.0.networks.*", "data.vsphere_supervisor.supervisor", "workload_network.1.network"),
				),
			},
		},
	})
}

func TestAccResourceVSphereNamespace_vmService(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
			testAccCheckEnvVariables(t, []string{"TF_VAR_SUPERVISOR", "TF_VAR_STORAGE_POLICY"})
		},
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccResourceVSphereNamespaceConfigVmService(),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("vsphere_namespace.namespace", "id"),
					resource.TestCheckResourceAttrSet("vsphere_namespace.namespace", "supervisor"),
					resource.TestCheckResourceAttrSet("vsphere_namespace.namespace", "storage.#"),
					resource.TestCheckResourceAttrSet("vsphere_namespace.namespace", "vm_service.0.vm_classes.#"),
					resource.TestCheckResourceAttrSet("vsphere_namespace.namespace", "vm_service.0.content_libraries.#"),
				),
			},
		},
	})
}

func TestAccResourceVSphereNamespace_update(t *testing.T) {
	// consecutively apply some of the creation configurations to verify the work via the update path too
	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
			testAccCheckEnvVariables(t, []string{"TF_VAR_SUPERVISOR", "TF_VAR_STORAGE_POLICY"})
		},
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccResourceVSphereNamespaceConfigBasic(),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("vsphere_namespace.namespace", "id"),
					resource.TestCheckResourceAttrSet("vsphere_namespace.namespace", "supervisor"),
				),
			},
			{
				Config: testAccResourceVSphereNamespaceConfigStorage(0),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("vsphere_namespace.namespace", "id"),
					resource.TestCheckResourceAttrSet("vsphere_namespace.namespace", "supervisor"),
					resource.TestCheckResourceAttrSet("vsphere_namespace.namespace", "storage.#"),
				),
			},
			{
				Config: testAccResourceVSphereNamespaceConfigVmService(),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("vsphere_namespace.namespace", "id"),
					resource.TestCheckResourceAttrSet("vsphere_namespace.namespace", "supervisor"),
					resource.TestCheckResourceAttrSet("vsphere_namespace.namespace", "storage.#"),
					resource.TestCheckResourceAttrSet("vsphere_namespace.namespace", "vm_service.0.vm_classes.#"),
					resource.TestCheckResourceAttrSet("vsphere_namespace.namespace", "vm_service.0.content_libraries.#"),
				),
			},
		},
	})
}

func testAccResourceVSphereNamespaceConfigBasic() string {
	return fmt.Sprintf(`
%s

resource vsphere_namespace "namespace" {
  name       = "test-acc-namespace"
  supervisor = data.vsphere_supervisor.supervisor.id
}
`, testAccDataSourceVSphereSupervisorConfigName())
}

func testAccResourceVSphereNamespaceConfigStorage(limit int) string {
	limitAttr := ""
	if limit > 0 {
		limitAttr = fmt.Sprintf("limit  = %d", limit)
	}
	return fmt.Sprintf(`
%s

data vsphere_storage_policy image_policy {
  name = "%s"
}

resource vsphere_namespace "namespace" {
  name       = "test-acc-namespace"
  supervisor = data.vsphere_supervisor.supervisor.id

  storage {
    policy = data.vsphere_storage_policy.image_policy.id
    %s
  }
}
`,
		testAccDataSourceVSphereSupervisorConfigName(),
		os.Getenv("TF_VAR_STORAGE_POLICY"),
		limitAttr)
}

func testAccResourceVSphereNamespaceConfigAccess(role string) string {
	return fmt.Sprintf(`
%s

resource vsphere_sso_user "user" {
  name     = "acc-namespace-user"
  password = "AccTestP@ssw0rd!"
}

resource vsphere_namespace "namespace" {
  name       = "test-acc-namespace"
  supervisor = data.vsphere_supervisor.supervisor.id

  access {
    subject_type = "USER"
    subject      = vsphere_sso_user.user.name
    domain       = vsphere_sso_user.user.domain
    role         = "%s"
  }
}
`,
		testAccDataSourceVSphereSupervisorConfigName(),
		role)
}

func testAccResourceVSphereNamespaceConfigZones() string {
	return fmt.Sprintf(`
%s

resource vsphere_namespace "namespace" {
  name       = "test-acc-namespace"
  supervisor = data.vsphere_supervisor.supervisor.id

  zone {
    name = "%s"
  }
}
`,
		testAccDataSourceVSphereSupervisorConfigName(),
		os.Getenv("TF_VAR_NAMESPACE_ZONE"))
}

func testAccResourceVSphereNamespaceConfigLimits() string {
	return fmt.Sprintf(`
%s

data vsphere_storage_policy image_policy {
  name = "%s"
}

resource vsphere_namespace "namespace" {
  name       = "test-acc-namespace"
  supervisor = data.vsphere_supervisor.supervisor.id

  zone {
    name               = "%s"
    cpu_limit          = 2000
    cpu_reservation    = 1000
    memory_limit       = 4096
    memory_reservation = 2048
  }

  storage {
    policy = data.vsphere_storage_policy.image_policy.id
    limit  = 1024
  }
}
`,
		testAccDataSourceVSphereSupervisorConfigName(),
		os.Getenv("TF_VAR_STORAGE_POLICY"),
		os.Getenv("TF_VAR_NAMESPACE_ZONE"))
}

func testAccResourceVSphereNamespaceConfigVpcNetwork(withSecondCidr bool) string {
	secondCidr := ""
	if withSecondCidr {
		secondCidr = `
        private_cidrs {
          address = "10.245.0.0"
          prefix  = 24
        }`
	}
	return fmt.Sprintf(`
%s

resource vsphere_namespace "namespace" {
  name       = "test-acc-namespace"
  supervisor = data.vsphere_supervisor.supervisor.id

  vpc_network {
    default_subnet_size = 64

    vpc_config {
      private_cidrs {
        address = "10.244.0.0"
        prefix  = 24
      }

%s
    }
  }
}
`,
		testAccResourceVSphereNamespaceSupervisorConfig("TF_VAR_SUPERVISOR"),
		secondCidr)
}

func testAccResourceVSphereNamespaceConfigVpcSharedSubnets(vmDefault bool) string {
	return fmt.Sprintf(`
%s

resource vsphere_namespace "namespace" {
  name       = "test-acc-namespace"
  supervisor = data.vsphere_supervisor.supervisor.id

  vpc_network {
    vpc = "%s"

    shared_subnet {
      path        = "%s"
      pod_default = true
      vm_default  = %t
    }
  }
}
`,
		testAccResourceVSphereNamespaceSupervisorConfig("TF_VAR_SUPERVISOR"),
		os.Getenv("TF_VAR_NAMESPACE_VPC_PATH"),
		os.Getenv("TF_VAR_NAMESPACE_SHARED_SUBNET_PATH"),
		vmDefault)
}

func testAccResourceVSphereNamespaceConfigVsphereNetwork() string {
	return fmt.Sprintf(`
%s

resource vsphere_namespace "namespace" {
  name       = "test-acc-namespace"
  supervisor = data.vsphere_supervisor.supervisor.id

  vsphere_network {
    default_workload_network = data.vsphere_supervisor.supervisor.workload_network.0.network
    networks = [
      data.vsphere_supervisor.supervisor.workload_network.0.network,
      data.vsphere_supervisor.supervisor.workload_network.1.network,
    ]
  }
}
`, testAccResourceVSphereNamespaceSupervisorConfig("TF_VAR_SUPERVISOR"))
}

func testAccResourceVSphereNamespaceSupervisorConfig(supervisorEnvVar string) string {
	return fmt.Sprintf(`
data vsphere_supervisor "supervisor" {
  name = "%s"
}
`, os.Getenv(supervisorEnvVar))
}

func testAccResourceVSphereNamespaceConfigVmService() string {
	return fmt.Sprintf(`
%s

data vsphere_storage_policy image_policy {
  name = "%s"
}

resource "vsphere_virtual_machine_class" "vm_class" {
  name               = "acc-test-vmclass"
  cpus               = 4
  memory             = 4096
  memory_reservation = 100
}

resource "vsphere_content_library" "library_published" {
  name            = "testacc_published"
  storage_backing = [data.vsphere_datastore.rootds1.id]
  publication {
    authentication_method = "BASIC"
    username              = "vcsp"
    password              = "Password123!"
    published             = true
  }
}

resource "vsphere_content_library" "library_subscribed" {
  name            = "testacc_subscribed"
  storage_backing = [data.vsphere_datastore.rootds1.id]
  subscription {
    authentication_method = "BASIC"
    username              = "vcsp"
    password              = "Password123!"
    subscription_url      = vsphere_content_library.library_published.publication.0.publish_url
  }
}

resource vsphere_namespace "namespace" {
  name       = "test-acc-namespace"
  supervisor = data.vsphere_supervisor.supervisor.id

  vm_service {
    content_libraries = [vsphere_content_library.library_subscribed.id]
    vm_classes = [vsphere_virtual_machine_class.vm_class.id]
  }

  storage {
    policy = data.vsphere_storage_policy.image_policy.id
  }
}
`,
		testhelper.CombineConfigs(
			testAccResourceVSphereNamespaceConfigBase(),
			testAccDataSourceVSphereSupervisorConfigName()),
		os.Getenv("TF_VAR_STORAGE_POLICY"))
}

func testAccResourceVSphereNamespaceConfigBase() string {
	return testhelper.CombineConfigs(testhelper.ConfigDataRootDC1(), testhelper.ConfigDataRootDS1())
}
