package vsphere

import (
	"errors"
	"fmt"
	"os"
	"testing"

	"golang.org/x/net/context"

	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
	"github.com/vmware/govmomi"
	"github.com/vmware/govmomi/license"
)

var (
	testAccLabels = map[string]string{
		"VpxClientLicenseLabel": "Hello World",
		"TestTitle":             "FooBar",
	}

	labelStub = []interface{}{
		map[string]interface{}{
			"key":   "Hello",
			"value": "World",
		},
		map[string]interface{}{
			"key":   "Working",
			"value": "This",
		},
		map[string]interface{}{
			"key":   "Testing",
			"value": "Labels",
		},
	}
)

func testAccVSphereLicenseBasic(t *testing.T) {

	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
			testAccVSpherePreLicenseBasicCheck(t)
		},
		Providers:    testAccProviders,
		CheckDestroy: testAccVSphereLicenseDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccVSphereLicenseBasicCreate(),
				Check: resource.ComposeTestCheckFunc(
					testAccVSphereLicenseExists("vsphere_license.foo"),
				),
			},
		},
	})

}

func testAccVSphereLicenseInvalid(t *testing.T) {

	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccVSphereLicenseInvalidCreate(),
				Check: resource.ComposeTestCheckFunc(
					testAccVSphereLicenseNotExists("vsphere_license.foo"),
				),
			},
		},
	})

}

func testAccVSphereLicenseWithLabels(t *testing.T) {

	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
			testAccVSpherePreLicenseBasicCheck(t)
		},
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccVSphereLicenseWithLabelCreate(testAccLabels),
				Check: resource.ComposeTestCheckFunc(
					testAccVSphereLicenseWithLabelExists("vsphere_license.foo"),
				),
			},
		},
	})

}

func testAccVSphereLicenseInvalidCreate() string {

	// quite sure this key cannot be valid
	return `resource "vsphere_license" "foo" {
  					license_key = "00000-00000-00000-00000-12345"
			}`
}

func testAccVSphereLicenseWithLabelCreate(labels map[string]string) string {

	// precheck already checks if this is present or not
	key := os.Getenv("VSPHERE_LICENSE")

	labelString := labelToString(labels)

	return fmt.Sprintf(`resource "vsphere_license" "foo2" {
					license_key = "%s"

					%s
		}`, key, labelString)
}

func labelToString(labels map[string]string) string {
	val := ""
	for key, value := range labels {
		val += fmt.Sprintf(`
		label {
			key = "%s"
			value = "%s"
		}
		`, key, value)

	}
	return val
}

func testAccVSphereLicenseBasicCreate() string {

	// precheck already checks if this is present or not
	key := os.Getenv("VSPHERE_LICENSE")

	return fmt.Sprintf(`resource "vsphere_license" "foo" {
  		license_key = "%s"
		}
	`, key)

}

func testAccVSphereLicenseDestroy(s *terraform.State) error {

	client := testAccProvider.Meta().(*govmomi.Client)

	manager := license.NewManager(client.Client)

	message := ""
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "vsphere_license" {
			continue
		}

		key := rs.Primary.ID
		if isKeyPresent(key, manager) {
			message += fmt.Sprintf("%s is still present on the server", key)
		}

	}
	if message != "" {
		return errors.New(message)
	}

	return nil
}

func testAccVSphereLicenseExists(name string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[name]

		if !ok {
			return fmt.Errorf("%s key not found on the server", name)
		}

		client := testAccProvider.Meta().(*govmomi.Client)
		manager := license.NewManager(client.Client)

		if !isKeyPresent(rs.Primary.ID, manager) {
			return fmt.Errorf("%s key not found on the server", rs.Primary.ID)
		}

		return nil
	}
}

func testAccVSphereLicenseNotExists(name string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		_, ok := s.RootModule().Resources[name]

		if ok {
			return fmt.Errorf("%s key should not be present on the server", name)
		}

		return nil
	}
}

func testAccVSpherePreLicenseBasicCheck(t *testing.T) {
	if key := os.Getenv("VSPHERE_LICENSE"); key == "" {
		t.Fatal("VSPHERE_LICENSE must be set for acceptance test")
	}
}

func testAccVSphereLicenseWithLabelExists(name string) resource.TestCheckFunc {

	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[name]

		if !ok {
			return fmt.Errorf("%s key not found on the server", name)
		}

		client := testAccProvider.Meta().(*govmomi.Client)
		manager := license.NewManager(client.Client)

		if !isKeyPresent(rs.Primary.ID, manager) {
			return fmt.Errorf("%s key not found on the server", rs.Primary.ID)
		}

		info, err := manager.Decode(context.TODO(), rs.Primary.ID)

		if err != nil {
			return err
		}

		if len(info.Labels) == 0 {
			return fmt.Errorf("The labels were not set for the key %s", info.LicenseKey)
		}

		return nil
	}

}

func TestLabelToMaps(t *testing.T) {

	mapdata, err := labelsToMap(labelStub)

	if err != nil {
		t.Fatal("Error ", err)
	}

	if value, ok := mapdata["Hello"]; !ok || value != "World" {
		t.Fatal("the map data is invalid")
	}

}
