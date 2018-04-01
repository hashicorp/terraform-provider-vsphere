package vsphere

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"strings"

	"github.com/hashicorp/terraform/helper/schema"
	"github.com/hashicorp/terraform/helper/validation"
	"github.com/terraform-providers/terraform-provider-vsphere/vsphere/internal/helper/storagepod"
	"github.com/terraform-providers/terraform-provider-vsphere/vsphere/internal/helper/structure"
	"github.com/terraform-providers/terraform-provider-vsphere/vsphere/internal/helper/viapi"
	"github.com/terraform-providers/terraform-provider-vsphere/vsphere/internal/helper/virtualmachine"
	"github.com/vmware/govmomi/object"
	"github.com/vmware/govmomi/vim25/types"
)

const resourceVSphereStorageDrsVMConfigName = "vsphere_storage_drs_vm_config"

func resourceVSphereStorageDrsVMConfig() *schema.Resource {
	return &schema.Resource{
		Create: resourceVSphereStorageDrsVMConfigCreate,
		Read:   resourceVSphereStorageDrsVMConfigRead,
		Update: resourceVSphereStorageDrsVMConfigUpdate,
		Delete: resourceVSphereStorageDrsVMConfigDelete,
		Importer: &schema.ResourceImporter{
			State: resourceVSphereStorageDrsVMConfigImport,
		},

		Schema: map[string]*schema.Schema{
			"datastore_cluster_id": {
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
				Description: "The managed object ID of the datastore cluster.",
			},
			"virtual_machine_id": {
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
				Description: "The managed object ID of the virtual machine.",
			},
			"sdrs_enabled": {
				Type:        schema.TypeBool,
				Optional:    true,
				Description: "Overrides the default Storage DRS setting for this virtual machine.",
			},
			"sdrs_automation_level": {
				Type:         schema.TypeString,
				Optional:     true,
				Default:      string(types.StorageDrsPodConfigInfoBehaviorAutomated),
				Description:  "Overrides any Storage DRS automation levels for this virtual machine.",
				ValidateFunc: validation.StringInSlice(storageDrsPodConfigInfoBehaviorAllowedValues, false),
			},
			"sdrs_intra_vm_affinity": {
				Type:        schema.TypeBool,
				Optional:    true,
				Description: "Overrides the intra-VM affinity setting for this virtual machine.",
			},
		},
	}
}

func resourceVSphereStorageDrsVMConfigCreate(d *schema.ResourceData, meta interface{}) error {
	log.Printf("[DEBUG] %s: Beginning create", resourceVSphereStorageDrsVMConfigIDString(d))

	if err := resourceVSphereStorageDrsVMConfigApplyCreate(d, meta); err != nil {
		return err
	}

	log.Printf("[DEBUG] %s: Create finished successfully", resourceVSphereStorageDrsVMConfigIDString(d))
	return resourceVSphereStorageDrsVMConfigRead(d, meta)
}

func resourceVSphereStorageDrsVMConfigRead(d *schema.ResourceData, meta interface{}) error {
	log.Printf("[DEBUG] %s: Beginning read", resourceVSphereStorageDrsVMConfigIDString(d))
	info, err := resourceVSphereStorageDrsVMConfigGetEntry(d, meta)
	if err != nil {
		return err
	}

	if info == nil {
		// resourceVSphereStorageDrsVMConfigGetEntry return nil when there was no
		// entry, this is a re-creation scenario.
		d.SetId("")
		return nil
	}

	if err := flattenStorageDrsVMConfigInfo(d, info); err != nil {
		return err
	}

	log.Printf("[DEBUG] %s: Read completed successfully", resourceVSphereStorageDrsVMConfigIDString(d))
	return nil
}

func resourceVSphereStorageDrsVMConfigUpdate(d *schema.ResourceData, meta interface{}) error {
	log.Printf("[DEBUG] %s: Beginning update", resourceVSphereStorageDrsVMConfigIDString(d))

	if err := resourceVSphereStorageDrsVMConfigApplyUpdate(d, meta); err != nil {
		return err
	}

	log.Printf("[DEBUG] %s: Update finished successfully", resourceVSphereStorageDrsVMConfigIDString(d))
	return resourceVSphereStorageDrsVMConfigRead(d, meta)
}

func resourceVSphereStorageDrsVMConfigDelete(d *schema.ResourceData, meta interface{}) error {
	resourceIDString := resourceVSphereStorageDrsVMConfigIDString(d)
	log.Printf("[DEBUG] %s: Beginning delete", resourceIDString)

	if err := resourceVSphereStorageDrsVMConfigApplyDelete(d, meta); err != nil {
		return err
	}

	log.Printf("[DEBUG] %s: Deleted successfully", resourceIDString)
	return nil
}

func resourceVSphereStorageDrsVMConfigImport(d *schema.ResourceData, meta interface{}) ([]*schema.ResourceData, error) {
	var data map[string]string
	if err := json.Unmarshal([]byte(d.Id()), &data); err != nil {
		return nil, err
	}
	podPath, ok := data["datastore_cluster_path"]
	if !ok {
		return nil, errors.New("missing datastore_cluster_path in input data")
	}
	vmPath, ok := data["virtual_machine_path"]
	if !ok {
		return nil, errors.New("missing virtual_machine_path in input data")
	}

	client := meta.(*VSphereClient).vimClient
	if err := viapi.ValidateVirtualCenter(client); err != nil {
		return nil, err
	}

	pod, err := storagepod.FromPath(client, podPath, nil)
	if err != nil {
		return nil, fmt.Errorf("cannot locate datastore cluster %q: %s", podPath, err)
	}

	vm, err := virtualmachine.FromPath(client, vmPath, nil)
	if err != nil {
		return nil, fmt.Errorf("cannot locate virtual machine %q: %s", vmPath, err)
	}

	id, err := resourceVSphereStorageDrsVMConfigFlattenID(pod, vm)
	if err != nil {
		return nil, fmt.Errorf("cannot compute ID of imported resource: %s", err)
	}
	d.SetId(id)
	return []*schema.ResourceData{d}, nil
}

// resourceVSphereStorageDrsVMConfigApplyCreate processes the creation part of
// resourceVSphereStorageDrsVMConfigCreate.
func resourceVSphereStorageDrsVMConfigApplyCreate(d *schema.ResourceData, meta interface{}) error {
	log.Printf("[DEBUG] %s: Processing datastore cluster VM setting creation", resourceVSphereStorageDrsVMConfigIDString(d))
	client := meta.(*VSphereClient).vimClient
	if err := viapi.ValidateVirtualCenter(client); err != nil {
		return err
	}

	pod, err := storagepod.FromID(client, d.Get("datastore_cluster_id").(string))
	if err != nil {
		return fmt.Errorf("cannot locate datastore cluster: %s", err)
	}

	vm, err := virtualmachine.FromUUID(client, d.Get("virtual_machine_id").(string))
	if err != nil {
		return fmt.Errorf("cannot locate virtual machine: %s", err)
	}

	spec := types.StorageDrsConfigSpec{
		VmConfigSpec: []types.StorageDrsVmConfigSpec{
			{
				ArrayUpdateSpec: types.ArrayUpdateSpec{
					Operation: types.ArrayUpdateOperationAdd,
				},
				Info: expandStorageDrsVMConfigInfo(d, vm),
			},
		},
	}

	if err = storagepod.ApplyDRSConfiguration(client, pod, spec); err != nil {
		return err
	}

	id, err := resourceVSphereStorageDrsVMConfigFlattenID(pod, vm)
	if err != nil {
		return fmt.Errorf("cannot compute ID of imported resource: %s", err)
	}
	d.SetId(id)

	return nil
}

// resourceVSphereStorageDrsVMConfigApplyUpdate processes the creation part of
// resourceVSphereStorageDrsVMConfigUpdate.
func resourceVSphereStorageDrsVMConfigApplyUpdate(d *schema.ResourceData, meta interface{}) error {
	log.Printf("[DEBUG] %s: Processing datastore cluster VM setting updates", resourceVSphereStorageDrsVMConfigIDString(d))
	client := meta.(*VSphereClient).vimClient
	if err := viapi.ValidateVirtualCenter(client); err != nil {
		return err
	}

	pod, err := storagepod.FromID(client, d.Get("datastore_cluster_id").(string))
	if err != nil {
		return fmt.Errorf("cannot locate datastore cluster: %s", err)
	}

	vm, err := virtualmachine.FromUUID(client, d.Get("virtual_machine_id").(string))
	if err != nil {
		return fmt.Errorf("cannot locate virtual machine: %s", err)
	}

	spec := types.StorageDrsConfigSpec{
		VmConfigSpec: []types.StorageDrsVmConfigSpec{
			{
				ArrayUpdateSpec: types.ArrayUpdateSpec{
					Operation: types.ArrayUpdateOperationEdit,
				},
				Info: expandStorageDrsVMConfigInfo(d, vm),
			},
		},
	}

	if err := storagepod.ApplyDRSConfiguration(client, pod, spec); err != nil {
		return err
	}

	return nil
}

// resourceVSphereStorageDrsVMConfigApplyDelete processes the creation part of
// resourceVSphereStorageDrsVMConfigDelete.
func resourceVSphereStorageDrsVMConfigApplyDelete(d *schema.ResourceData, meta interface{}) error {
	log.Printf("[DEBUG] %s: Processing datastore cluster VM setting removal", resourceVSphereStorageDrsVMConfigIDString(d))
	client := meta.(*VSphereClient).vimClient
	if err := viapi.ValidateVirtualCenter(client); err != nil {
		return err
	}

	pod, err := storagepod.FromID(client, d.Get("datastore_cluster_id").(string))
	if err != nil {
		return fmt.Errorf("cannot locate datastore cluster: %s", err)
	}

	vm, err := virtualmachine.FromUUID(client, d.Get("virtual_machine_id").(string))
	if err != nil {
		return fmt.Errorf("cannot locate virtual machine: %s", err)
	}

	spec := types.StorageDrsConfigSpec{
		VmConfigSpec: []types.StorageDrsVmConfigSpec{
			{
				ArrayUpdateSpec: types.ArrayUpdateSpec{
					Operation: types.ArrayUpdateOperationRemove,
					RemoveKey: vm.Reference(),
				},
			},
		},
	}

	if err := storagepod.ApplyDRSConfiguration(client, pod, spec); err != nil {
		return err
	}

	d.SetId("")

	return nil
}

// resourceVSphereStorageDrsVMConfigGetEntry gets the StorageDrsVmConfigInfo
// entry for the specific StoragePod/VM combination. nil is returned if the
// entry was not found.
func resourceVSphereStorageDrsVMConfigGetEntry(d structure.ResourceIDStringer, meta interface{}) (*types.StorageDrsVmConfigInfo, error) {
	log.Printf("[DEBUG] %s: Fetching config info object from resource ID", resourceVSphereStorageDrsVMConfigIDString(d))
	client := meta.(*VSphereClient).vimClient
	if err := viapi.ValidateVirtualCenter(client); err != nil {
		return nil, err
	}

	podID, vmID, err := resourceVSphereStorageDrsVMConfigParseID(d.Id())
	if err != nil {
		return nil, err
	}

	pod, err := storagepod.FromID(client, podID)
	if err != nil {
		return nil, fmt.Errorf("cannot locate datastore cluster: %s", err)
	}

	vm, err := virtualmachine.FromUUID(client, vmID)
	if err != nil {
		return nil, fmt.Errorf("cannot locate virtual machine: %s", err)
	}

	props, err := storagepod.Properties(pod)
	if err != nil {
		return nil, fmt.Errorf("error fetching datastore cluster properties: %s", err)
	}

	for _, info := range props.PodStorageDrsEntry.StorageDrsConfig.VmConfig {
		if *info.Vm == vm.Reference() {
			log.Printf("[DEBUG] %s: Found storage DRS config info for pod/VM combination", resourceVSphereStorageDrsVMConfigIDString(d))
			return &info, nil
		}
	}

	log.Printf("[DEBUG] %s: No storage DRS config info found for pod/VM combination", resourceVSphereStorageDrsVMConfigIDString(d))
	return nil, nil
}

// expandStorageDrsVMConfigInfo reads certain ResourceData keys and returns a
// StorageDrsVmConfigInfo.
func expandStorageDrsVMConfigInfo(d *schema.ResourceData, vm *object.VirtualMachine) *types.StorageDrsVmConfigInfo {
	obj := &types.StorageDrsVmConfigInfo{
		Behavior:        d.Get("sdrs_automation_level").(string),
		Enabled:         structure.GetBoolPtr(d, "sdrs_enabled"),
		IntraVmAffinity: structure.GetBoolPtr(d, "sdrs_intra_vm_affinity"),
		Vm:              types.NewReference(vm.Reference()),
	}

	return obj
}

// flattenStorageDrsVmConfigInfo saves a StorageDrsVmConfigInfo into the
// supplied ResourceData.
func flattenStorageDrsVMConfigInfo(d *schema.ResourceData, obj *types.StorageDrsVmConfigInfo) error {
	attrs := map[string]interface{}{
		"sdrs_automation_level":  obj.Behavior,
		"sdrs_enabled":           obj.Enabled,
		"sdrs_intra_vm_affinity": obj.IntraVmAffinity,
	}
	for k, p := range attrs {
		if v := structure.DeRef(p); v != nil {
			if err := d.Set(k, v); err != nil {
				return fmt.Errorf("error setting attribute %q: %s", k, err)
			}
		}
	}

	return nil
}

// resourceVSphereStorageDrsVMConfigIDString prints a friendly string for the
// vsphere_storage_drs_vm_config resource.
func resourceVSphereStorageDrsVMConfigIDString(d structure.ResourceIDStringer) string {
	return structure.ResourceIDString(d, resourceVSphereStorageDrsVMConfigName)
}

// resourceVSphereStorageDrsVMConfigFlattenID makes an ID for the
// vsphere_storage_drs_vm_config resource.
func resourceVSphereStorageDrsVMConfigFlattenID(pod *object.StoragePod, vm *object.VirtualMachine) (string, error) {
	podID := pod.Reference().Value
	props, err := virtualmachine.Properties(vm)
	if err != nil {
		return "", fmt.Errorf("cannot compute ID off of properties of virtual machine: %s", err)
	}
	vmID := props.Config.Uuid
	return strings.Join([]string{podID, vmID}, ":"), nil
}

// resourceVSphereStorageDrsVMConfigParseID parses an ID for the
// vsphere_storage_drs_vm_config and outputs its parts.
func resourceVSphereStorageDrsVMConfigParseID(id string) (string, string, error) {
	parts := strings.SplitN(id, ":", 3)
	if len(parts) < 2 {
		return "", "", fmt.Errorf("bad ID %q", id)
	}
	return parts[0], parts[1], nil
}
