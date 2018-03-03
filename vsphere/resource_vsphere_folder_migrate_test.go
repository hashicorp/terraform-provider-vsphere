package vsphere

import (
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/hashicorp/terraform/terraform"
)

func testAccResourceVSphereFolderMigrateStatePreCheck(t *testing.T) {
	if os.Getenv("TF_ACC") == "" {
		t.Skip("set TF_ACC to run vsphere_folder state migration tests (provider connection is required)")
	}
	if os.Getenv("VSPHERE_FOLDER_V0_PATH") == "" {
		t.Skip("set VSPHERE_FOLDER_V0_PATH to run vsphere_folder state migration tests")
	}
}

func TestAccResourceVSphereFolderMigrateState_basic(t *testing.T) {
	testAccResourceVSphereFolderMigrateStatePreCheck(t)
	testAccPreCheck(t)

	is := &terraform.InstanceState{
		ID: fmt.Sprintf("%v/%v", os.Getenv("VSPHERE_DATACENTER"), os.Getenv("VSPHERE_FOLDER_V0_PATH")),
		Attributes: map[string]string{
			"path": os.Getenv("VSPHERE_FOLDER_V0_PATH"),
		},
	}
	if dc := os.Getenv("VSPHERE_DATACENTER"); dc != "" {
		is.Attributes["datacenter"] = dc
	}
	meta, err := testAccProviderMeta(t)
	if err != nil {
		t.Fatalf("bad: %s", err)
	}
	is, err = resourceVSphereFolderMigrateState(0, is, meta)
	if err != nil {
		t.Fatalf("bad: %s", err)
	}
	if !strings.HasPrefix(is.ID, "group-") {
		t.Fatalf("expected ID to start with \"group-\" got ID as %q", is.ID)
	}
}

func TestAccResourceVSphereFolderMigrateState_empty(t *testing.T) {
	var is *terraform.InstanceState
	var meta interface{}

	testAccResourceVSphereFolderMigrateStatePreCheck(t)
	testAccPreCheck(t)

	// should handle nil
	is, err := resourceVSphereFolderMigrateState(0, is, meta)

	if err != nil {
		t.Fatalf("err: %#v", err)
	}
	if is != nil {
		t.Fatalf("expected nil instancestate, got: %#v", is)
	}

	// should handle non-nil but empty
	is = &terraform.InstanceState{}
	is, err = resourceVSphereFolderMigrateState(0, is, meta)

	if err != nil {
		t.Fatalf("err: %#v", err)
	}
}
