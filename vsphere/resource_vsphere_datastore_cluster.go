package vsphere

import (
	"fmt"
	"log"

	"github.com/hashicorp/terraform/helper/schema"
	"github.com/hashicorp/terraform/helper/validation"
	"github.com/terraform-providers/terraform-provider-vsphere/vsphere/internal/helper/customattribute"
	"github.com/terraform-providers/terraform-provider-vsphere/vsphere/internal/helper/folder"
	"github.com/terraform-providers/terraform-provider-vsphere/vsphere/internal/helper/storagepod"
	"github.com/terraform-providers/terraform-provider-vsphere/vsphere/internal/helper/structure"
	"github.com/terraform-providers/terraform-provider-vsphere/vsphere/internal/helper/viapi"
	"github.com/vmware/govmomi/object"
	"github.com/vmware/govmomi/vim25/types"
)

var storageDrsPodConfigInfoBehaviorAllowedValues = []string{
	string(types.StorageDrsPodConfigInfoBehaviorManual),
	string(types.StorageDrsPodConfigInfoBehaviorAutomated),
}

var storageDrsSpaceLoadBalanceConfigSpaceThresholdModeAllowedValues = []string{
	string(types.StorageDrsSpaceLoadBalanceConfigSpaceThresholdModeUtilization),
	string(types.StorageDrsSpaceLoadBalanceConfigSpaceThresholdModeFreeSpace),
}

func resourceVSphereDatastoreCluster() *schema.Resource {
	return &schema.Resource{
		Create: resourceVSphereDatastoreClusterCreate,
		Read:   resourceVSphereDatastoreClusterRead,
		Update: resourceVSphereDatastoreClusterUpdate,
		Delete: resourceVSphereDatastoreClusterDelete,

		Schema: map[string]*schema.Schema{
			"name": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "Name for the new storage pod.",
				StateFunc:   folder.NormalizePath,
			},
			"datacenter_id": {
				Type:        schema.TypeString,
				Description: "The managed object ID of the datacenter to put the datastore cluster in.",
				Required:    true,
			},
			"folder": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "The name of the folder to locate the datastore cluster in.",
				StateFunc:   folder.NormalizePath,
			},
			"sdrs_enabled": {
				Type:        schema.TypeBool,
				Optional:    true,
				Default:     false,
				Description: "Enable storage DRS for this datastore cluster.",
			},
			"sdrs_automation_level": {
				Type:         schema.TypeString,
				Optional:     true,
				Default:      string(types.StorageDrsPodConfigInfoBehaviorAutomated),
				Description:  "The default automation level for all virtual machines in this storage cluster.",
				ValidateFunc: validation.StringInSlice(storageDrsPodConfigInfoBehaviorAllowedValues, false),
			},
			"sdrs_space_balance_automation_level": {
				Type:         schema.TypeString,
				Optional:     true,
				Description:  "Overrides the default automation settings when correcting disk space imbalances.",
				ValidateFunc: validation.StringInSlice(storageDrsPodConfigInfoBehaviorAllowedValues, false),
			},
			"sdrs_io_balance_automation_level": {
				Type:         schema.TypeString,
				Optional:     true,
				Description:  "Overrides the default automation settings when correcting I/O load imbalances.",
				ValidateFunc: validation.StringInSlice(storageDrsPodConfigInfoBehaviorAllowedValues, false),
			},
			"sdrs_rule_enforcement_automation_level": {
				Type:         schema.TypeString,
				Optional:     true,
				Description:  "Overrides the default automation settings when correcting affinity rule violations.",
				ValidateFunc: validation.StringInSlice(storageDrsPodConfigInfoBehaviorAllowedValues, false),
			},
			"sdrs_policy_enforcement_automation_level": {
				Type:         schema.TypeString,
				Optional:     true,
				Description:  "Overrides the default automation settings when correcting storage and VM policy violations.",
				ValidateFunc: validation.StringInSlice(storageDrsPodConfigInfoBehaviorAllowedValues, false),
			},
			"sdrs_vm_evacuation_automation_level": {
				Type:         schema.TypeString,
				Optional:     true,
				Description:  "Overrides the default automation settings when generating recommendations for datastore evacuation.",
				ValidateFunc: validation.StringInSlice(storageDrsPodConfigInfoBehaviorAllowedValues, false),
			},
			"sdrs_io_load_balance_enabled": {
				Type:        schema.TypeBool,
				Optional:    true,
				Default:     true,
				Description: "Enable I/O load balancing for this datastore cluster.",
			},
			"sdrs_default_intra_vm_affinity": {
				Type:        schema.TypeBool,
				Optional:    true,
				Default:     true,
				Description: "When true, storage DRS keeps VMDKs for individual VMs on the same datastore by default.",
			},
			"sdrs_io_latency_threshold": {
				Type:         schema.TypeInt,
				Optional:     true,
				Default:      15,
				Description:  "The I/O latency threshold, in milliseconds, that storage DRS uses to make recommendations to move disks from this datastore.",
				ValidateFunc: validation.IntBetween(5, 100),
			},
			"sdrs_io_load_imbalance_threshold": {
				Type:         schema.TypeInt,
				Optional:     true,
				Default:      5,
				Description:  "The difference between load in datastores in the cluster before storage DRS makes recommendations to balance the load.",
				ValidateFunc: validation.IntBetween(1, 100),
			},
			"sdrs_io_reservable_iops_threshold": {
				Type:        schema.TypeInt,
				Optional:    true,
				Description: "The threshold of reservable IOPS of all virtual machines on the datastore before storage DRS makes recommendations to move VMs off of a datastore.",
			},
			"sdrs_io_reservable_percent_threshold": {
				Type:         schema.TypeInt,
				Optional:     true,
				Default:      60,
				Description:  "The threshold, in percent, of actual estimated performance of the datastore (in IOPS) that storage DRS uses to make recommendations to move VMs off of a datastore when the total reservable IOPS exceeds the threshold.",
				ValidateFunc: validation.IntBetween(30, 100),
			},
			"sdrs_io_reservable_threshold_mode": {
				Type:         schema.TypeString,
				Optional:     true,
				Default:      string(types.StorageDrsPodConfigInfoBehaviorAutomated),
				Description:  "The reservable IOPS threshold to use, percent in the event of automatic, or manual threshold in the event of manual.",
				ValidateFunc: validation.StringInSlice(storageDrsPodConfigInfoBehaviorAllowedValues, false),
			},
			"sdrs_load_balance_interval": {
				Type:         schema.TypeInt,
				Optional:     true,
				Default:      480,
				Description:  "The storage DRS poll interval, in minutes.",
				ValidateFunc: validation.IntBetween(60, 2505600),
			},
			"sdrs_free_space_threshold": {
				Type:         schema.TypeInt,
				Optional:     true,
				Default:      50,
				Description:  "The threshold, in GB, that storage DRS uses to make decisions to migrate VMs out of a datastore.",
				ValidateFunc: validation.IntAtLeast(1),
			},
			"sdrs_free_space_utilization_difference": {
				Type:         schema.TypeInt,
				Optional:     true,
				Default:      5,
				Description:  "The threshold, in percent, of difference between space utilization in datastores before storage DRS makes decisions to balance the space.",
				ValidateFunc: validation.IntBetween(1, 50),
			},
			"sdrs_space_utilization_threshold": {
				Type:         schema.TypeInt,
				Optional:     true,
				Default:      80,
				Description:  "The threshold, in percent of used space, that storage DRS uses to make decisions to migrate VMs out of a datastore.",
				ValidateFunc: validation.IntAtLeast(1),
			},
			"sdrs_free_space_threshold_mode": {
				Type:         schema.TypeString,
				Optional:     true,
				Default:      string(types.StorageDrsSpaceLoadBalanceConfigSpaceThresholdModeUtilization),
				Description:  "The free space threshold to use. When set to utilization, drs_space_utilization_threshold is used, and when set to freeSpace, drs_free_space_threshold is used.",
				ValidateFunc: validation.StringInSlice(storageDrsSpaceLoadBalanceConfigSpaceThresholdModeAllowedValues, false),
			},
			"sdrs_advanced_options": {
				Type:        schema.TypeMap,
				Optional:    true,
				Description: "Advanced configuration options for storage DRS.",
			},
			vSphereTagAttributeKey:    tagsSchema(),
			customattribute.ConfigKey: customattribute.ConfigSchema(),
		},
	}
}

func resourceVSphereDatastoreClusterCreate(d *schema.ResourceData, meta interface{}) error {
	log.Printf("[DEBUG] %s: Beginning create", resourceVSphereDatastoreClusterIDString(d))

	pod, err := resourceVSphereDatastoreClusterApplyCreate(d, meta)
	if err != nil {
		return err
	}

	if err := resourceVSphereDatastoreClusterApplyTags(d, meta, pod); err != nil {
		return err
	}

	if err := resourceVSphereDatastoreClusterApplyCustomAttributes(d, meta, pod); err != nil {
		return err
	}

	if err := resourceVSphereDatastoreClusterApplySDRSConfig(d, meta, pod); err != nil {
		return err
	}

	log.Printf("[DEBUG] %s: Create finished successfully", resourceVSphereDatastoreClusterIDString(d))
	return resourceVSphereDatastoreClusterRead(d, meta)
}

func resourceVSphereDatastoreClusterRead(d *schema.ResourceData, meta interface{}) error {
	log.Printf("[DEBUG] %s: Beginning read", resourceVSphereDatastoreClusterIDString(d))
	pod, err := resourceVSphereDatastoreClusterGetPod(d, meta)
	if err != nil {
		return err
	}

	if err := resourceVSphereDatastoreClusterFlattenSDRSData(d, meta, pod); err != nil {
		return err
	}

	return nil
}

func resourceVSphereDatastoreClusterUpdate(d *schema.ResourceData, meta interface{}) error {
	return resourceVSphereDatastoreClusterRead(d, meta)
}

func resourceVSphereDatastoreClusterDelete(d *schema.ResourceData, meta interface{}) error {
	return nil
}

// resourceVSphereDatastoreClusterApplyCreate processes the creation part of
// resourceVSphereDatastoreClusterCreate.
func resourceVSphereDatastoreClusterApplyCreate(d *schema.ResourceData, meta interface{}) (*object.StoragePod, error) {
	log.Printf("[DEBUG] %s: Processing datastore cluster creation", resourceVSphereDatastoreClusterIDString(d))
	client := meta.(*VSphereClient).vimClient
	if err := viapi.ValidateVirtualCenter(client); err != nil {
		return nil, err
	}

	dc, err := datacenterFromID(client, d.Get("datacenter_id").(string))
	if err != nil {
		return nil, fmt.Errorf("cannot locate datacenter: %s", err)
	}

	// Find the folder based off the path to the datacenter. This is where we
	// create the datastore cluster.
	f, err := folder.DatastoreFolderFromObject(client, dc, d.Get("folder").(string))
	if err != nil {
		return nil, fmt.Errorf("cannot locate folder: %s", err)
	}

	// Create the storage pod (datastore cluster).
	pod, err := storagepod.Create(f, d.Get("name").(string))
	if err != nil {
		return nil, fmt.Errorf("error creating datastore cluster: %s", err)
	}

	// Set the ID now before proceeding with tags, custom attributes, and DRS.
	// This ensures that we can recover from a problem with any of these
	// operations.
	d.SetId(pod.Reference().Value)

	return pod, nil
}

// resourceVSphereDatastoreClusterApplyTags processes the tags step for both
// create and update for vsphere_datastore_cluster.
func resourceVSphereDatastoreClusterApplyTags(d *schema.ResourceData, meta interface{}, pod *object.StoragePod) error {
	tagsClient, err := tagsClientIfDefined(d, meta)
	if err != nil {
		return err
	}

	// Apply any pending tags now
	if tagsClient == nil {
		log.Printf("[DEBUG] %s: Tags unsupported on this connection, skipping", resourceVSphereDatastoreClusterIDString(d))
		return nil
	}

	log.Printf("[DEBUG] %s: Applying any pending tags", resourceVSphereDatastoreClusterIDString(d))
	return processTagDiff(tagsClient, d, pod)
}

// resourceVSphereDatastoreClusterReadTags reads the tags for
// vsphere_datastore_cluster.
func resourceVSphereDatastoreClusterReadTags(d *schema.ResourceData, meta interface{}, pod *object.StoragePod) error {
	if tagsClient, _ := meta.(*VSphereClient).TagsClient(); tagsClient != nil {
		log.Printf("[DEBUG] %s: Reading tags", resourceVSphereDatastoreClusterIDString(d))
		if err := readTagsForResource(tagsClient, pod, d); err != nil {
			return err
		}
	} else {
		log.Printf("[DEBUG] %s: Tags unsupported on this connection, skipping tag read", resourceVSphereDatastoreClusterIDString(d))
	}
	return nil
}

// resourceVSphereDatastoreClusterApplyCustomAttributes processes the custom
// attributes step for both create and update for vsphere_datastore_cluster.
func resourceVSphereDatastoreClusterApplyCustomAttributes(d *schema.ResourceData, meta interface{}, pod *object.StoragePod) error {
	client := meta.(*VSphereClient).vimClient
	// Verify a proper vCenter before proceeding if custom attributes are defined
	attrsProcessor, err := customattribute.GetDiffProcessorIfAttributesDefined(client, d)
	if err != nil {
		return err
	}

	if attrsProcessor == nil {
		log.Printf("[DEBUG] %s: Custom attributes unsupported on this connection, skipping", resourceVSphereDatastoreClusterIDString(d))
		return nil
	}

	log.Printf("[DEBUG] %s: Applying any pending custom attributes", resourceVSphereDatastoreClusterIDString(d))
	return attrsProcessor.ProcessDiff(pod)
}

// resourceVSphereDatastoreClusterReadCustomAttributes reads the custom
// attributes for vsphere_datastore_cluster.
func resourceVSphereDatastoreClusterReadCustomAttributes(d *schema.ResourceData, meta interface{}, pod *object.StoragePod) error {
	client := meta.(*VSphereClient).vimClient
	// Read custom attributes
	if customattribute.IsSupported(client) {
		log.Printf("[DEBUG] %s: Reading custom attributes", resourceVSphereDatastoreClusterIDString(d))
		props, err := storagepod.Properties(pod)
		if err != nil {
			return err
		}
		customattribute.ReadFromResource(client, props.Entity(), d)
	} else {
		log.Printf("[DEBUG] %s: Custom attributes unsupported on this connection, skipping", resourceVSphereDatastoreClusterIDString(d))
	}

	return nil
}

// resourceVSphereDatastoreClusterApplySDRSConfig applies the SDRS configuration to a datastore cluster.
func resourceVSphereDatastoreClusterApplySDRSConfig(d *schema.ResourceData, meta interface{}, pod *object.StoragePod) error {
	log.Printf("[DEBUG] %s: Applying SDRS configuration", resourceVSphereDatastoreClusterIDString(d))
	client := meta.(*VSphereClient).vimClient
	if err := viapi.ValidateVirtualCenter(client); err != nil {
		return err
	}

	// Get the version of the vSphere connection to help determine what
	// attributes we need to set
	version := viapi.ParseVersionFromClient(client)

	// Expand the SDRS configuration.
	spec := types.StorageDrsConfigSpec{
		PodConfigSpec: expandStorageDrsPodConfigSpec(d, version),
	}

	return storagepod.ApplyDRSConfiguration(client, pod, spec)
}

// resourceVSphereDatastoreClusterGetPod gets the StoragePod from the ID in the
// supplied ResourceData.
func resourceVSphereDatastoreClusterGetPod(d structure.ResourceIDStringer, meta interface{}) (*object.StoragePod, error) {
	log.Printf("[DEBUG] %s: Fetching StoragePod object from resource ID", resourceVSphereDatastoreClusterIDString(d))
	client := meta.(*VSphereClient).vimClient
	if err := viapi.ValidateVirtualCenter(client); err != nil {
		return nil, err
	}

	return storagepod.FromID(client, d.Id())
}

// resourceVSphereDatastoreClusterSaveNameAndPath saves the name and path of a
// StoragePod into the supplied ResourceData.
func resourceVSphereDatastoreClusterSaveNameAndPath(d *schema.ResourceData, pod *object.StoragePod) error {
	log.Printf(
		"[DEBUG] %s: Saving name and path data for datastore cluster %q",
		resourceVSphereDatastoreClusterIDString(d),
		pod.InventoryPath,
	)

	if err := d.Set("name", pod.Name()); err != nil {
		return fmt.Errorf("error saving name: %s", err)
	}

	f, err := folder.RootPathParticleDatastore.SplitRelativeFolder(pod.InventoryPath)
	if err != nil {
		return fmt.Errorf("error parsing datastore cluster path %q: %s", pod.InventoryPath, err)
	}
	if err := d.Set("folder", folder.NormalizePath(f)); err != nil {
		return fmt.Errorf("error saving folder: %s", err)
	}
	return nil
}

// resourceVSphereDatastoreClusterFlattenSDRSData saves the DRS attributes from
// a StoragePod into the supplied ResourceData.
//
// Note that other functions handle non-SDRS related items, such as path, name,
// tags, and custom attributes.
func resourceVSphereDatastoreClusterFlattenSDRSData(d *schema.ResourceData, meta interface{}, pod *object.StoragePod) error {
	log.Printf("[DEBUG] %s: Saving datastore cluster attributes", resourceVSphereDatastoreClusterIDString(d))
	client := meta.(*VSphereClient).vimClient
	if err := viapi.ValidateVirtualCenter(client); err != nil {
		return err
	}

	// Get the version of the vSphere connection to help determine what
	// attributes we need to set
	version := viapi.ParseVersionFromClient(client)

	props, err := storagepod.Properties(pod)
	if err != nil {
		return err
	}

	return flattenStorageDrsPodConfigInfo(d, props.PodStorageDrsEntry.StorageDrsConfig.PodConfig, version)
}

// expandStorageDrsPodConfigSpec reads certain ResourceData keys and returns a
// StorageDrsPodConfigSpec.
func expandStorageDrsPodConfigSpec(d *schema.ResourceData, version viapi.VSphereVersion) *types.StorageDrsPodConfigSpec {
	obj := &types.StorageDrsPodConfigSpec{
		DefaultIntraVmAffinity: structure.GetBool(d, "sdrs_default_intra_vm_affinity"),
		DefaultVmBehavior:      d.Get("sdrs_automation_level").(string),
		Enabled:                structure.GetBool(d, "sdrs_enabled"),
		IoLoadBalanceConfig:    expandStorageDrsIoLoadBalanceConfig(d, version),
		IoLoadBalanceEnabled:   structure.GetBool(d, "sdrs_io_load_balance_enabled"),
		LoadBalanceInterval:    int32(d.Get("sdrs_load_balance_interval").(int)),
		SpaceLoadBalanceConfig: expandStorageDrsSpaceLoadBalanceConfig(d, version),
		Option:                 expandStorageDrsOptionSpec(d),
	}

	if version.Newer(viapi.VSphereVersion{Product: version.Product, Major: 6}) {
		obj.AutomationOverrides = expandStorageDrsAutomationConfig(d)
	}

	return obj
}

// flattenStorageDrsPodConfigInfo saves a StorageDrsPodConfigInfo into the supplied ResourceData.
func flattenStorageDrsPodConfigInfo(d *schema.ResourceData, obj types.StorageDrsPodConfigInfo, version viapi.VSphereVersion) error {
	attrs := map[string]interface{}{
		"sdrs_default_intra_vm_affinity": obj.DefaultIntraVmAffinity,
		"sdrs_automation_level":          obj.DefaultVmBehavior,
		"sdrs_enabled":                   obj.Enabled,
		"sdrs_io_load_balance_enabled":   obj.IoLoadBalanceEnabled,
		"sdrs_load_balance_interval":     obj.LoadBalanceInterval,
	}
	for k, v := range attrs {
		if err := d.Set(k, v); err != nil {
			return fmt.Errorf("error setting attribute %q: %s", k, err)
		}
	}

	return nil
}

// expandStorageDrsAutomationConfig reads certain ResourceData keys and returns
// a StorageDrsAutomationConfig.
func expandStorageDrsAutomationConfig(d *schema.ResourceData) *types.StorageDrsAutomationConfig {
	obj := &types.StorageDrsAutomationConfig{
		IoLoadBalanceAutomationMode:     d.Get("sdrs_io_balance_automation_level").(string),
		PolicyEnforcementAutomationMode: d.Get("sdrs_policy_enforcement_automation_level").(string),
		RuleEnforcementAutomationMode:   d.Get("sdrs_rule_enforcement_automation_level").(string),
		SpaceLoadBalanceAutomationMode:  d.Get("sdrs_space_balance_automation_level").(string),
		VmEvacuationAutomationMode:      d.Get("sdrs_vm_evacuation_automation_level").(string),
	}
	return obj
}

// expandStorageDrsIoLoadBalanceConfig reads certain ResourceData keys and returns
// a StorageDrsIoLoadBalanceConfig.
func expandStorageDrsIoLoadBalanceConfig(d *schema.ResourceData, version viapi.VSphereVersion) *types.StorageDrsIoLoadBalanceConfig {
	obj := &types.StorageDrsIoLoadBalanceConfig{
		IoLatencyThreshold:       int32(d.Get("sdrs_io_latency_threshold").(int)),
		IoLoadImbalanceThreshold: int32(d.Get("sdrs_io_load_imbalance_threshold").(int)),
	}

	if version.Newer(viapi.VSphereVersion{Product: version.Product, Major: 6}) {
		obj.ReservableIopsThreshold = int32(d.Get("sdrs_io_reservable_iops_threshold").(int))
		obj.ReservablePercentThreshold = int32(d.Get("sdrs_io_reservable_percent_threshold").(int))
		obj.ReservableThresholdMode = d.Get("sdrs_io_reservable_threshold_mode").(string)
	}

	return obj
}

// expandStorageDrsSpaceLoadBalanceConfig reads certain ResourceData keys and returns
// a StorageDrsSpaceLoadBalanceConfig.
func expandStorageDrsSpaceLoadBalanceConfig(
	d *schema.ResourceData,
	version viapi.VSphereVersion,
) *types.StorageDrsSpaceLoadBalanceConfig {
	obj := &types.StorageDrsSpaceLoadBalanceConfig{
		MinSpaceUtilizationDifference: int32(d.Get("sdrs_free_space_utilization_difference").(int)),
		SpaceUtilizationThreshold:     int32(d.Get("sdrs_space_utilization_threshold").(int)),
	}

	if version.Newer(viapi.VSphereVersion{Product: version.Product, Major: 6}) {
		obj.FreeSpaceThresholdGB = int32(d.Get("sdrs_free_space_threshold").(int))
		obj.SpaceThresholdMode = d.Get("sdrs_free_space_threshold_mode").(string)
	}

	return obj
}

// expandStorageDrsOptionSpec reads certain ResourceData keys and returns
// a StorageDrsOptionSpec.
func expandStorageDrsOptionSpec(d *schema.ResourceData) []types.StorageDrsOptionSpec {
	var opts []types.StorageDrsOptionSpec

	m := d.Get("sdrs_advanced_options").(map[string]interface{})
	for k, v := range m {
		opts = append(opts, types.StorageDrsOptionSpec{
			Option: &types.OptionValue{
				Key:   k,
				Value: types.AnyType(v),
			},
		})
	}
	return opts
}

// resourceVSphereDatastoreClusterIDString prints a friendly string for the
// vsphere_datastore_cluster resource.
func resourceVSphereDatastoreClusterIDString(d structure.ResourceIDStringer) string {
	return structure.ResourceIDString(d, "vsphere_datastore_cluster")
}
