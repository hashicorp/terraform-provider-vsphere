package vsphere

import (
	"os"
	"path"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform/terraform"
	"github.com/terraform-providers/terraform-provider-vsphere/vsphere/internal/helper/virtualmachine"
)

func testAccResourceVSphereVirtualMachineMigrateStatePreCheck(t *testing.T) {
	if os.Getenv("TF_ACC") == "" {
		t.Skip("set TF_ACC to run vsphere_virtual_machine state migration tests (provider connection is required)")
	}
	if os.Getenv("VSPHERE_VM_V1_PATH") == "" {
		t.Skip("set VSPHERE_VM_V1_PATH to run vsphere_virtual_machine state migration tests")
	}
}

func TestVSphereVirtualMachineMigrateStateV1(t *testing.T) {
	cases := map[string]struct {
		Attributes map[string]string
		Expected   map[string]string
	}{
		"skip_customization before 0.6.16": {
			Attributes: map[string]string{},
			Expected: map[string]string{
				"skip_customization": "false",
			},
		},
		"enable_disk_uuid before 0.6.16": {
			Attributes: map[string]string{},
			Expected: map[string]string{
				"enable_disk_uuid": "false",
			},
		},
		"disk controller_type": {
			Attributes: map[string]string{
				"disk.1234.size":            "0",
				"disk.5678.size":            "0",
				"disk.9999.size":            "0",
				"disk.9999.controller_type": "ide",
			},
			Expected: map[string]string{
				"disk.1234.size":            "0",
				"disk.1234.controller_type": "scsi",
				"disk.5678.size":            "0",
				"disk.5678.controller_type": "scsi",
				"disk.9999.size":            "0",
				"disk.9999.controller_type": "ide",
			},
		},
	}

	for tn, tc := range cases {
		is := &terraform.InstanceState{
			ID:         "i-abc123",
			Attributes: tc.Attributes,
		}
		if err := migrateVSphereVirtualMachineStateV1(is, nil); err != nil {
			t.Fatalf("bad: %s, err: %#v", tn, err)
		}

		for k, v := range tc.Expected {
			if is.Attributes[k] != v {
				t.Fatalf(
					"bad: %s\n\n expected: %#v -> %#v\n got: %#v -> %#v\n in: %#v",
					tn, k, v, k, is.Attributes[k], is.Attributes)
			}
		}
	}
}

func TestAccResourceVSphereVirtualMachineMigrateStateV2(t *testing.T) {
	testAccResourceVSphereVirtualMachineMigrateStatePreCheck(t)
	testAccPreCheck(t)

	meta, err := testAccProviderMeta(t)
	if err != nil {
		t.Fatalf("bad: %s", err)
	}

	client := meta.(*VSphereClient).vimClient
	pth := os.Getenv("VSPHERE_VM_V1_PATH")
	name := path.Base(pth)
	vm, err := virtualmachine.FromPath(client, pth, nil)
	if err != nil {
		t.Fatalf("error fetching virtual machine: %s", err)
	}
	props, err := virtualmachine.Properties(vm)
	if err != nil {
		t.Fatalf("error fetching virtual machine properties: %s", err)
	}

	is := &terraform.InstanceState{
		ID: name,
		Attributes: map[string]string{
			"uuid": props.Config.Uuid,
		},
	}
	// Start this at version 0 so we know it go through the whole path. There's
	// currently nothing from v0 to v2 that should hinder this.
	is, err = resourceVSphereVirtualMachineMigrateState(0, is, meta)
	if err != nil {
		t.Fatalf("bad: %s", err)
	}
	if matched, _ := regexp.Match("^[a-f0-9]{8}-[a-f0-9]{4}-[a-f0-9]{4}-[a-f0-9]{4}-[a-f0-9]{12}$", []byte(is.ID)); !matched {
		t.Fatalf("expected ID to be a UUID, got ID as %q", is.ID)
	}
	if is.Attributes["imported"] != "true" {
		t.Fatalf("expected imported to be true")
	}
}

func TestComputeInstanceMigrateState_empty(t *testing.T) {
	var is *terraform.InstanceState
	var meta interface{}

	// should handle nil
	is, err := resourceVSphereVirtualMachineMigrateState(0, is, meta)

	if err != nil {
		t.Fatalf("err: %#v", err)
	}
	if is != nil {
		t.Fatalf("expected nil instancestate, got: %#v", is)
	}

	// should handle non-nil but empty
	is = &terraform.InstanceState{}
	is, err = resourceVSphereVirtualMachineMigrateState(0, is, meta)

	if err != nil {
		t.Fatalf("err: %#v", err)
	}
}
