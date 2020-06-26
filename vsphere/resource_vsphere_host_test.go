package vsphere

import (
	"errors"
	"fmt"
	"os"
	"regexp"
	"strconv"
	"testing"

	"github.com/vmware/govmomi/vim25/types"

	"github.com/vmware/govmomi"

	"github.com/hashicorp/terraform-provider-vsphere/vsphere/internal/helper/hostsystem"
	"github.com/hashicorp/terraform-provider-vsphere/vsphere/internal/helper/viapi"

	"github.com/hashicorp/terraform-plugin-sdk/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/terraform"
)

func TestAccResourceVSphereHost_basic(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
			testAccCheckEnvVariables(t, []string{"ESX_HOSTNAME", "ESX_USERNAME", "ESX_PASSWORD"})
		},
		Providers:    testAccProviders,
		CheckDestroy: testAccVSphereHostDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccVSphereHostConfig(),
				Check: resource.ComposeTestCheckFunc(
					testAccVSphereHostExists("vsphere_host.h1"),
				),
			},
			{
				Config: testAccVSphereHostConfig_import(),
				Check: resource.ComposeTestCheckFunc(
					testAccVSphereHostExists("vsphere_host.h1"),
				),
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})

}

func TestAccResourceVSphereHost_rootFolder(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
			testAccCheckEnvVariables(t, []string{"ESX_HOSTNAME", "ESX_USERNAME", "ESX_PASSWORD"})
		},
		Providers:    testAccProviders,
		CheckDestroy: testAccVSphereHostDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccVSphereHostConfig_rootFolder(),
				Check: resource.ComposeTestCheckFunc(
					testAccVSphereHostExists("vsphere_host.h1"),
				),
			},
		},
	})

}

func TestAccResourceVSphereHost_connection(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
			testAccCheckEnvVariables(t, []string{"ESX_HOSTNAME", "ESX_USERNAME", "ESX_PASSWORD"})
		},
		Providers:    testAccProviders,
		CheckDestroy: testAccVSphereHostDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccVSphereHostConfig_connection(false),
				Check: resource.ComposeTestCheckFunc(
					testAccVSphereHostExists("vsphere_host.h1"),
					testAccVSphereHostConnected("vsphere_host.h1", false),
				),
			},
			{
				Config: testAccVSphereHostConfig_connection(true),
				Check: resource.ComposeTestCheckFunc(
					testAccVSphereHostExists("vsphere_host.h1"),
					testAccVSphereHostConnected("vsphere_host.h1", true),
				),
			},
		},
	})

}

func TestAccResourceVSphereHost_maintenance(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
			testAccCheckEnvVariables(t, []string{"ESX_HOSTNAME", "ESX_USERNAME", "ESX_PASSWORD"})
		},
		Providers:    testAccProviders,
		CheckDestroy: testAccVSphereHostDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccVSphereHostConfig_maintenance(true),
				Check: resource.ComposeTestCheckFunc(
					testAccVSphereHostExists("vsphere_host.h1"),
					testAccVSphereHostMaintenanceState("vsphere_host.h1", true),
				),
			},
			{
				Config: testAccVSphereHostConfig_maintenance(false),
				Check: resource.ComposeTestCheckFunc(
					testAccVSphereHostExists("vsphere_host.h1"),
					testAccVSphereHostMaintenanceState("vsphere_host.h1", false),
				),
			},
		},
	})

}

func TestAccResourceVSphereHost_lockdown(t *testing.T) {

	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
			testAccCheckEnvVariables(t, []string{"ESX_HOSTNAME", "ESX_USERNAME", "ESX_PASSWORD"})
		},
		Providers:    testAccProviders,
		CheckDestroy: testAccVSphereHostDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccVSphereHostConfig_lockdown("strict"),
				Check: resource.ComposeTestCheckFunc(
					testAccVSphereHostExists("vsphere_host.h1"),
					testAccVSphereHostLockdownState("vsphere_host.h1", "strict"),
				),
			},
			{
				Config: testAccVSphereHostConfig_lockdown("normal"),
				Check: resource.ComposeTestCheckFunc(
					testAccVSphereHostExists("vsphere_host.h1"),
					testAccVSphereHostLockdownState("vsphere_host.h1", "normal"),
				),
			},
			{
				Config: testAccVSphereHostConfig_lockdown("disabled"),
				Check: resource.ComposeTestCheckFunc(
					testAccVSphereHostExists("vsphere_host.h1"),
					testAccVSphereHostLockdownState("vsphere_host.h1", "disabled"),
				),
			},
		},
	})

}

func TestAccResourceVSphereHost_lockdown_invalid(t *testing.T) {

	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
			testAccCheckEnvVariables(t, []string{"ESX_HOSTNAME", "ESX_USERNAME", "ESX_PASSWORD"})
		},
		Providers:    testAccProviders,
		CheckDestroy: testAccVSphereHostDestroy,
		Steps: []resource.TestStep{
			{
				Config:      testAccVSphereHostConfig_lockdown("invalidvalue"),
				ExpectError: regexp.MustCompile("be one of \\[disabled normal strict\\], got invalidvalue"),
			},
		},
	})

}

func TestAccResourceVSphereHost_emptyLicense(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
			testAccCheckEnvVariables(t, []string{"ESX_HOSTNAME", "ESX_USERNAME", "ESX_PASSWORD"})
		},
		Providers:    testAccProviders,
		CheckDestroy: testAccVSphereHostDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccVSphereHostConfig_emptyLicense(),
				Check: resource.ComposeTestCheckFunc(
					testAccVSphereHostExists("vsphere_host.h1"),
				),
			},
		},
	})

}

func testAccVSphereHostExists(name string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[name]

		if !ok {
			return fmt.Errorf("%s key not found on the server", name)
		}
		hostID := rs.Primary.ID
		client := testAccProvider.Meta().(*VSphereClient).vimClient
		res, err := hostExists(client, hostID)
		if err != nil {
			return err
		}

		if !res {
			return fmt.Errorf("Host with ID %s not found", hostID)
		}

		return nil
	}
}

func testAccVSphereHostConnected(name string, shouldBeConnected bool) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[name]

		if !ok {
			return fmt.Errorf("%s key not found on the server", name)
		}
		hostID := rs.Primary.ID
		client := testAccProvider.Meta().(*VSphereClient).vimClient
		res, err := hostConnected(client, hostID)
		if err != nil {
			return err
		}

		if res != shouldBeConnected {
			return fmt.Errorf("Host with ID %s connection: %t, expected %t", hostID, res, shouldBeConnected)
		}

		return nil
	}
}

func testAccVSphereHostMaintenanceState(name string, inMaintenance bool) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[name]

		if !ok {
			return fmt.Errorf("%s key not found on the server", name)
		}
		hostID := rs.Primary.ID
		client := testAccProvider.Meta().(*VSphereClient).vimClient
		res, err := hostInMaintenance(client, hostID)
		if err != nil {
			return err
		}

		if res != inMaintenance {
			return fmt.Errorf("Host with ID %s in maintenance : %t, expected %t", hostID, res, inMaintenance)
		}

		return nil
	}
}

func testAccVSphereHostLockdownState(name string, lockdown string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[name]

		if !ok {
			return fmt.Errorf("%s key not found on the server", name)
		}
		hostID := rs.Primary.ID
		client := testAccProvider.Meta().(*VSphereClient).vimClient
		res, err := checkHostLockdown(client, hostID, lockdown)
		if err != nil {
			return err
		}

		if !res {
			return fmt.Errorf("Host with ID %s not in desired lockdown state. Current state: %s", hostID, lockdown)
		}

		return nil
	}
}

func testAccVSphereHostDestroy(s *terraform.State) error {
	message := ""
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "vsphere_host" {
			continue
		}
		hostID := rs.Primary.ID
		client := testAccProvider.Meta().(*VSphereClient).vimClient
		res, err := hostExists(client, hostID)
		if err != nil {
			return err
		}

		if res {
			message += fmt.Sprintf("Host with ID %s was found", hostID)
		}
	}
	if message != "" {
		return errors.New(message)
	}
	return nil
}

func hostExists(client *govmomi.Client, hostID string) (bool, error) {
	hs, err := hostsystem.FromID(client, hostID)
	if err != nil {
		if viapi.IsManagedObjectNotFoundError(err) {
			return false, nil
		}
		return false, err
	}

	if hs.Reference().Value != hostID {
		return false, nil
	}
	return true, nil
}

func hostConnected(client *govmomi.Client, hostID string) (bool, error) {
	hs, err := hostsystem.FromID(client, hostID)
	if err != nil {
		return false, err
	}

	connectionState, err := hostsystem.GetConnectionState(hs)
	if err != nil {
		return false, err
	}
	return (connectionState == types.HostSystemConnectionStateConnected), nil
}

func hostInMaintenance(client *govmomi.Client, hostID string) (bool, error) {
	hs, err := hostsystem.FromID(client, hostID)
	if err != nil {
		return false, err
	}

	maintenanceState, err := hostsystem.HostInMaintenance(hs)
	if err != nil {
		return false, err
	}
	return maintenanceState, nil
}

func checkHostLockdown(client *govmomi.Client, hostID, lockdownMode string) (bool, error) {
	host, err := hostsystem.FromID(client, hostID)
	if err != nil {
		return false, err
	}
	hostProps, err := hostsystem.Properties(host)
	if err != nil {
		return false, err
	}

	lockdownModes := map[types.HostLockdownMode]string{
		types.HostLockdownModeLockdownDisabled: "disabled",
		types.HostLockdownModeLockdownNormal:   "normal",
		types.HostLockdownModeLockdownStrict:   "strict",
	}

	modeString, ok := lockdownModes[hostProps.Config.LockdownMode]
	if !ok {
		return false, fmt.Errorf("Unknown lockdown mode found: %s", hostProps.Config.LockdownMode)
	}

	return (modeString == lockdownMode), nil
}

func testAccVSphereHostConfig() string {
	return fmt.Sprintf(`
	data "vsphere_datacenter" "dc" {
	  name = "%s"
	}

	resource "vsphere_compute_cluster" "c1" {
	  name = "%s"
	  datacenter_id = data.vsphere_datacenter.dc.id
	}

	resource "vsphere_host" "h1" {
	  # Useful only for connection
	  hostname = "%s"
	  username = "%s"
	  password = "%s"
	  thumbprint = "%s"

	  # Makes sense to update
	  license = "%s"
	  cluster = vsphere_compute_cluster.c1.id
	}
	`, os.Getenv("TF_VAR_VSPHERE_DATACENTER"),
		"TestCluster",
		os.Getenv("ESX_HOSTNAME"),
		os.Getenv("ESX_USERNAME"),
		os.Getenv("ESX_PASSWORD"),
		os.Getenv("ESX_THUMBPRINT"),
		os.Getenv("TF_VAR_VSPHERE_LICENSE"))
}

func testAccVSphereHostConfig_rootFolder() string {
	return fmt.Sprintf(`
	data "vsphere_datacenter" "dc" {
	  name = "%s"
	}

	resource "vsphere_host" "h1" {
	  # Useful only for connection
	  hostname = "%s"
	  username = "%s"
	  password = "%s"
	  thumbprint = "%s"

	  # Makes sense to update
	  license = "%s"
	  datacenter = data.vsphere_datacenter.dc.id
	}
	`, os.Getenv("TF_VAR_VSPHERE_DATACENTER"),
		os.Getenv("ESX_HOSTNAME"),
		os.Getenv("ESX_USERNAME"),
		os.Getenv("ESX_PASSWORD"),
		os.Getenv("ESX_THUMBPRINT"),
		os.Getenv("TF_VAR_VSPHERE_LICENSE"))
}

func testAccVSphereHostConfig_emptyLicense() string {
	return fmt.Sprintf(`
	data "vsphere_datacenter" "dc" {
	  name = "%s"
	}

	resource "vsphere_host" "h1" {
	  # Useful only for connection
	  hostname = "%s"
	  username = "%s"
	  password = "%s"
	  thumbprint = "%s"

	  # Makes sense to update
	  datacenter = data.vsphere_datacenter.dc.id
	}
	`, os.Getenv("TF_VAR_VSPHERE_DATACENTER"),
		os.Getenv("ESX_HOSTNAME"),
		os.Getenv("ESX_USERNAME"),
		os.Getenv("ESX_PASSWORD"),
		os.Getenv("ESX_THUMBPRINT"))
}

func testAccVSphereHostConfig_import() string {
	return fmt.Sprintf(`
	data "vsphere_datacenter" "dc" {
	  name = "%s"
	}
		
	resource "vsphere_compute_cluster" "c1" {
	  name = "%s"
	  datacenter_id = data.vsphere_datacenter.dc.id
	}
		
	resource "vsphere_host" "h1" {
	  # Useful only for connection
	  hostname = "%s"
	  username = "%s"
	  password = "%s"
	  thumbprint = "%s"
	
	  # Makes sense to update
	  license = "%s"
	  cluster = vsphere_compute_cluster.c1.id
	}	  
	`, os.Getenv("TF_VAR_VSPHERE_DATACENTER"),
		"TestCluster",
		os.Getenv("ESX_HOSTNAME"),
		os.Getenv("ESX_USERNAME"),
		os.Getenv("ESX_PASSWORD"),
		os.Getenv("ESX_THUMBPRINT"),
		os.Getenv("TF_VAR_VSPHERE_LICENSE"))
}

func testAccVSphereHostConfig_connection(connection bool) string {
	return fmt.Sprintf(`
	data "vsphere_datacenter" "dc" {
	  name = "%s"
	}
		
	resource "vsphere_compute_cluster" "c1" {
	  name = "%s"
	  datacenter_id = data.vsphere_datacenter.dc.id
	}
		
	resource "vsphere_host" "h1" {
	  hostname = "%s"
	  username = "%s"
	  password = "%s"
	  thumbprint = "%s"
	
	  license = "%s"
	  connected = "%s"
	  cluster = vsphere_compute_cluster.c1.id
	}	  
	`, os.Getenv("TF_VAR_VSPHERE_DATACENTER"),
		"TestCluster",
		os.Getenv("ESX_HOSTNAME"),
		os.Getenv("ESX_USERNAME"),
		os.Getenv("ESX_PASSWORD"),
		os.Getenv("ESX_THUMBPRINT"),
		os.Getenv("TF_VAR_VSPHERE_LICENSE"),
		strconv.FormatBool(connection))
}

func testAccVSphereHostConfig_maintenance(maintenance bool) string {
	return fmt.Sprintf(`
	data "vsphere_datacenter" "dc" {
	  name = "%s"
	}
		
	resource "vsphere_compute_cluster" "c1" {
	  name = "%s"
	  datacenter_id = data.vsphere_datacenter.dc.id
	}
		
	resource "vsphere_host" "h1" {
	  hostname = "%s"
	  username = "%s"
	  password = "%s"
	  thumbprint = "%s"
	
	  license = "%s"
	  connected = "true"
	  maintenance = "%s"
	  cluster = vsphere_compute_cluster.c1.id
	}	  
	`, os.Getenv("TF_VAR_VSPHERE_DATACENTER"),
		"TestCluster",
		os.Getenv("ESX_HOSTNAME"),
		os.Getenv("ESX_USERNAME"),
		os.Getenv("ESX_PASSWORD"),
		os.Getenv("ESX_THUMBPRINT"),
		os.Getenv("TF_VAR_VSPHERE_LICENSE"),
		strconv.FormatBool(maintenance))
}

func testAccVSphereHostConfig_lockdown(lockdown string) string {
	return fmt.Sprintf(`
	data "vsphere_datacenter" "dc" {
	  name = "%s"
	}
		
	resource "vsphere_compute_cluster" "c1" {
	  name = "%s"
	  datacenter_id = data.vsphere_datacenter.dc.id
	}
		
	resource "vsphere_host" "h1" {
	  hostname = "%s"
	  username = "%s"
	  password = "%s"
	  thumbprint = "%s"
	
	  license = "%s"
	  connected = "true"
	  maintenance = "false"
	  lockdown = "%s"
	  cluster = vsphere_compute_cluster.c1.id
	}	  
	`, os.Getenv("TF_VAR_VSPHERE_DATACENTER"),
		"TestCluster",
		os.Getenv("ESX_HOSTNAME"),
		os.Getenv("ESX_USERNAME"),
		os.Getenv("ESX_PASSWORD"),
		os.Getenv("ESX_THUMBPRINT"),
		os.Getenv("TF_VAR_VSPHERE_LICENSE"),
		lockdown)
}
