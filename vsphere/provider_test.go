package vsphere

import (
	"os"
	"testing"

	"github.com/hashicorp/terraform/helper/schema"
	"github.com/hashicorp/terraform/terraform"
	"github.com/terraform-providers/terraform-provider-null/null"
	"github.com/terraform-providers/terraform-provider-random/random"
	"github.com/terraform-providers/terraform-provider-template/template"
)

var testAccProviders map[string]terraform.ResourceProvider
var testAccProvider *schema.Provider
var testAccNullProvider *schema.Provider
var testAccRandomProvider *schema.Provider
var testAccTemplateProvider *schema.Provider

func init() {
	testAccProvider = Provider().(*schema.Provider)
	testAccNullProvider = null.Provider().(*schema.Provider)
	testAccRandomProvider = random.Provider().(*schema.Provider)
	testAccTemplateProvider = template.Provider().(*schema.Provider)
	testAccProviders = map[string]terraform.ResourceProvider{
		"vsphere":  testAccProvider,
		"null":     testAccNullProvider,
		"random":   testAccRandomProvider,
		"template": testAccTemplateProvider,
	}
}

func TestProvider(t *testing.T) {
	if err := Provider().(*schema.Provider).InternalValidate(); err != nil {
		t.Fatalf("err: %s", err)
	}

}

func TestProvider_impl(t *testing.T) {
	var _ terraform.ResourceProvider = Provider()
}

func testAccPreCheck(t *testing.T) {
	if v := os.Getenv("VSPHERE_USER"); v == "" {
		t.Fatal("VSPHERE_USER must be set for acceptance tests")
	}

	if v := os.Getenv("VSPHERE_PASSWORD"); v == "" {
		t.Fatal("VSPHERE_PASSWORD must be set for acceptance tests")
	}

	if v := os.Getenv("VSPHERE_SERVER"); v == "" {
		t.Fatal("VSPHERE_SERVER must be set for acceptance tests")
	}
}

// testAccProviderMeta returns a instantiated VSphereClient for this provider.
// It's useful in state migration tests where a provider connection is actually
// needed, and we don't want to go through the regular provider configure
// channels (so this function doesn't interfere with the testAccProvider
// package global and standard acceptance tests).
//
// Note we lean on environment variables for most of the provider configuration
// here and this will fail if those are missing. A pre-check is not run.
func testAccProviderMeta(t *testing.T) (interface{}, error) {
	t.Helper()
	d := schema.TestResourceDataRaw(t, testAccProvider.Schema, make(map[string]interface{}))
	return providerConfigure(d)
}
