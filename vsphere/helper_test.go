package vsphere

import (
	"os"
	"testing"
)

// testAccSkipIfNotEsxi skips a test if VSPHERE_TEST_ESXI is not set.
func testAccSkipIfNotEsxi(t *testing.T) {
	if os.Getenv("VSPHERE_TEST_ESXI") == "" {
		t.Skip("set VSPHERE_TEST_ESXI to run ESXi-specific acceptance tests")
	}
}
