package vsphere

import (
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/terraform"
	"github.com/terraform-providers/terraform-provider-null/null"
	"github.com/terraform-providers/terraform-provider-random/random"
)

var testAccProviders map[string]terraform.ResourceProvider
var testAccProvider *schema.Provider
var testAccNullProvider *schema.Provider
var testAccRandomProvider *schema.Provider

func init() {
	testAccProvider = Provider().(*schema.Provider)
	testAccNullProvider = null.Provider().(*schema.Provider)
	testAccRandomProvider = random.Provider().(*schema.Provider)
	testAccProviders = map[string]terraform.ResourceProvider{
		"vsphere": testAccProvider,
		"null":    testAccNullProvider,
		"random":  testAccRandomProvider,
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
	if v := os.Getenv("TF_VAR_VSPHERE_USER"); v == "" {
		t.Fatal("TF_VAR_VSPHERE_USER must be set for acceptance tests")
	}

	if v := os.Getenv("TF_VAR_VSPHERE_PASSWORD"); v == "" {
		t.Fatal("TF_VAR_VSPHERE_PASSWORD must be set for acceptance tests")
	}

	if v := os.Getenv("TF_VAR_VSPHERE_SERVER"); v == "" {
		t.Fatal("TF_VAR_VSPHERE_SERVER must be set for acceptance tests")
	}
}

func testAccCheckEnvVariables(t *testing.T, variableNames []string) {
	for _, name := range variableNames {
		if v := os.Getenv(name); v == "" {
			t.Skipf("%s must be set for this acceptance test", name)
		}
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
