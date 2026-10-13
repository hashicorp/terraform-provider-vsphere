// © Broadcom. All Rights Reserved.
// The term "Broadcom" refers to Broadcom Inc. and/or its subsidiaries.
// SPDX-License-Identifier: MPL-2.0

package vsphere

import (
	"fmt"
	"os"
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

func TestAccResourceVSphereNamespace_accessList(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
			testAccCheckEnvVariables(t, []string{
				"TF_VAR_SUPERVISOR",
				"TF_VAR_NAMESPACE_ACCESS_SUBJECT",
				"TF_VAR_NAMESPACE_ACCESS_DOMAIN",
			})
		},
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccResourceVSphereNamespaceConfigAccessList("EDIT"),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("vsphere_namespace.namespace", "access.#", "1"),
					resource.TestCheckTypeSetElemNestedAttrs("vsphere_namespace.namespace", "access.*", map[string]string{
						"subject_type": "USER",
						"subject":      os.Getenv("TF_VAR_NAMESPACE_ACCESS_SUBJECT"),
						"domain":       os.Getenv("TF_VAR_NAMESPACE_ACCESS_DOMAIN"),
						"role":         "EDIT",
					}),
				),
			},
			{
				Config: testAccResourceVSphereNamespaceConfigAccessList("VIEW"),
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
			testAccCheckEnvVariables(t, []string{"TF_VAR_NAMESPACE_ZONE_SUPERVISOR", "TF_VAR_NAMESPACE_ZONE"})
		},
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccResourceVSphereNamespaceConfigZones(""),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("vsphere_namespace.namespace", "zone.#", "1"),
					resource.TestCheckTypeSetElemNestedAttrs("vsphere_namespace.namespace", "zone.*", map[string]string{
						"name":         os.Getenv("TF_VAR_NAMESPACE_ZONE"),
						"cpu_limit":    "2000",
						"memory_limit": "4096",
					}),
				),
			},
			{
				Config: testAccResourceVSphereNamespaceConfigZones(testAccResourceVSphereNamespaceZoneReservations()),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("vsphere_namespace.namespace", "zone.#", "1"),
					resource.TestCheckTypeSetElemNestedAttrs("vsphere_namespace.namespace", "zone.*", map[string]string{
						"name":               os.Getenv("TF_VAR_NAMESPACE_ZONE"),
						"cpu_reservation":    "1000",
						"memory_reservation": "2048",
						"vm_reservation.#":   "1",
					}),
				),
			},
		},
	})
}

func TestAccResourceVSphereNamespace_vpcNetwork(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
			testAccCheckEnvVariables(t, []string{"TF_VAR_NAMESPACE_NSX_VPC_SUPERVISOR"})
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
				"TF_VAR_NAMESPACE_NSX_VPC_SUPERVISOR",
				"TF_VAR_NAMESPACE_VPC",
				"TF_VAR_NAMESPACE_SHARED_SUBNET_PATH",
			})
		},
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccResourceVSphereNamespaceConfigVpcSharedSubnets(true),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("vsphere_namespace.namespace", "vpc_network.0.vpc", os.Getenv("TF_VAR_NAMESPACE_VPC")),
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

func testAccResourceVSphereNamespaceConfigAccessList(role string) string {
	return fmt.Sprintf(`
%s

resource vsphere_namespace "namespace" {
  name       = "test-acc-namespace"
  supervisor = data.vsphere_supervisor.supervisor.id

  access {
    subject_type = "USER"
    subject      = "%s"
    domain       = "%s"
    role         = "%s"
  }
}
`,
		testAccDataSourceVSphereSupervisorConfigName(),
		os.Getenv("TF_VAR_NAMESPACE_ACCESS_SUBJECT"),
		os.Getenv("TF_VAR_NAMESPACE_ACCESS_DOMAIN"),
		role)
}

func testAccResourceVSphereNamespaceZoneReservations() string {
	return `
      cpu_reservation    = 1000
      memory_reservation = 2048

      vm_reservation {
        reserved_vm_class = vsphere_virtual_machine_class.vm_class.id
        count             = 1
      }`
}

func testAccResourceVSphereNamespaceConfigZones(reservations string) string {
	return fmt.Sprintf(`
%s

resource "vsphere_virtual_machine_class" "vm_class" {
  name   = "acc-test-zone-vmclass"
  cpus   = 2
  memory = 2048
}

resource vsphere_namespace "namespace" {
  name       = "test-acc-namespace"
  supervisor = data.vsphere_supervisor.supervisor.id

  vm_service {
    vm_classes = [vsphere_virtual_machine_class.vm_class.id]
  }

  zone {
    name         = "%s"
    cpu_limit    = 2000
    memory_limit = 4096%s
  }
}
`,
		testAccResourceVSphereNamespaceSupervisorConfig("TF_VAR_NAMESPACE_ZONE_SUPERVISOR"),
		os.Getenv("TF_VAR_NAMESPACE_ZONE"),
		reservations)
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
      }%s
    }
  }
}
`,
		testAccResourceVSphereNamespaceSupervisorConfig("TF_VAR_NAMESPACE_NSX_VPC_SUPERVISOR"),
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
		testAccResourceVSphereNamespaceSupervisorConfig("TF_VAR_NAMESPACE_NSX_VPC_SUPERVISOR"),
		os.Getenv("TF_VAR_NAMESPACE_VPC"),
		os.Getenv("TF_VAR_NAMESPACE_SHARED_SUBNET_PATH"),
		vmDefault)
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
		testAccResourceVSphereNamespaceConfigBase(),
		testAccDataSourceVSphereSupervisorConfigName(),
		os.Getenv("TF_VAR_STORAGE_POLICY"))
}

func testAccResourceVSphereNamespaceConfigBase() string {
	return testhelper.CombineConfigs(testhelper.ConfigDataRootDC1(), testhelper.ConfigDataRootDS1())
}
