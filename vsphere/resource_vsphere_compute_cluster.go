package vsphere

import (
	"log"

	"github.com/hashicorp/terraform/helper/schema"
	"github.com/hashicorp/terraform/helper/validation"
	"github.com/terraform-providers/terraform-provider-vsphere/vsphere/internal/helper/customattribute"
	"github.com/terraform-providers/terraform-provider-vsphere/vsphere/internal/helper/folder"
	"github.com/terraform-providers/terraform-provider-vsphere/vsphere/internal/helper/structure"
	"github.com/vmware/govmomi/vim25/types"
)

const resourceVSphereComputeClusterName = "vsphere_compute_cluster"

const (
	clusterAdmissionControlTypeResourcePercentage = "resourcePercentage"
	clusterAdmissionControlTypeSlotPolicy         = "slotPolicy"
	clusterAdmissionControlTypeFailoverHosts      = "failoverHosts"
	clusterAdmissionControlTypeDisabled           = "disabled"
)

var clusterAdmissionControlTypeAllowedValues = []string{
	clusterAdmissionControlTypeResourcePercentage,
	clusterAdmissionControlTypeSlotPolicy,
	clusterAdmissionControlTypeFailoverHosts,
	clusterAdmissionControlTypeDisabled,
}

var drsBehaviorAllowedValues = []string{
	string(types.DrsBehaviorManual),
	string(types.DrsBehaviorPartiallyAutomated),
	string(types.DrsBehaviorFullyAutomated),
}

var dpmBehaviorAllowedValues = []string{
	string(types.DpmBehaviorManual),
	string(types.DpmBehaviorAutomated),
}

var clusterDasConfigInfoServiceStateAllowedValues = []string{
	string(types.ClusterDasConfigInfoServiceStateEnabled),
	string(types.ClusterDasConfigInfoServiceStateDisabled),
}

var computeClusterDasConfigInfoServiceStateAllowedValues = []string{
	string(types.ClusterDasVmSettingsRestartPriorityLowest),
	string(types.ClusterDasVmSettingsRestartPriorityLow),
	string(types.ClusterDasVmSettingsRestartPriorityMedium),
	string(types.ClusterDasVmSettingsRestartPriorityHigh),
	string(types.ClusterDasVmSettingsRestartPriorityHighest),
}

var computeClusterVMReadinessReadyConditionAllowedValues = []string{
	string(types.ClusterVmReadinessReadyConditionNone),
	string(types.ClusterVmReadinessReadyConditionPoweredOn),
	string(types.ClusterVmReadinessReadyConditionGuestHbStatusGreen),
	string(types.ClusterVmReadinessReadyConditionAppHbStatusGreen),
}

var computeClusterDasVMSettingsIsolationResponseAllowedValues = []string{
	string(types.ClusterDasVmSettingsIsolationResponseNone),
	string(types.ClusterDasVmSettingsIsolationResponsePowerOff),
	string(types.ClusterDasVmSettingsIsolationResponseShutdown),
}

var computeClusterVMStorageProtectionForPDLAllowedValues = []string{
	string(types.ClusterVmComponentProtectionSettingsStorageVmReactionDisabled),
	string(types.ClusterVmComponentProtectionSettingsStorageVmReactionWarning),
	string(types.ClusterVmComponentProtectionSettingsStorageVmReactionRestartAggressive),
}

var computeClusterVMStorageProtectionForAPDAllowedValues = []string{
	string(types.ClusterVmComponentProtectionSettingsStorageVmReactionDisabled),
	string(types.ClusterVmComponentProtectionSettingsStorageVmReactionWarning),
	string(types.ClusterVmComponentProtectionSettingsStorageVmReactionRestartConservative),
	string(types.ClusterVmComponentProtectionSettingsStorageVmReactionRestartAggressive),
}

var computeClusterVMReactionOnAPDClearedAllowedValues = []string{
	string(types.ClusterVmComponentProtectionSettingsVmReactionOnAPDClearedNone),
	string(types.ClusterVmComponentProtectionSettingsVmReactionOnAPDClearedReset),
}

var clusterDasConfigInfoVMMonitoringStateAllowedValues = []string{
	string(types.ClusterDasConfigInfoVmMonitoringStateVmMonitoringDisabled),
	string(types.ClusterDasConfigInfoVmMonitoringStateVmMonitoringOnly),
	string(types.ClusterDasConfigInfoVmMonitoringStateVmAndAppMonitoring),
}

var clusterDasConfigInfoHBDatastoreCandidateAllowedValues = []string{
	string(types.ClusterDasConfigInfoHBDatastoreCandidateUserSelectedDs),
	string(types.ClusterDasConfigInfoHBDatastoreCandidateAllFeasibleDs),
	string(types.ClusterDasConfigInfoHBDatastoreCandidateAllFeasibleDsWithUserPreference),
}

var clusterInfraUpdateHaConfigInfoBehaviorTypeAllowedValues = []string{
	string(types.ClusterInfraUpdateHaConfigInfoBehaviorTypeManual),
	string(types.ClusterInfraUpdateHaConfigInfoBehaviorTypeAutomated),
}

var clusterInfraUpdateHaConfigInfoRemediationTypeAllowedValues = []string{
	string(types.ClusterInfraUpdateHaConfigInfoRemediationTypeMaintenanceMode),
	string(types.ClusterInfraUpdateHaConfigInfoRemediationTypeQuarantineMode),
}

func resourceVSphereComputeCluster() *schema.Resource {
	return &schema.Resource{
		Create: resourceVSphereComputeClusterCreate,
		Read:   resourceVSphereComputeClusterRead,
		Update: resourceVSphereComputeClusterUpdate,
		Delete: resourceVSphereComputeClusterDelete,

		Schema: map[string]*schema.Schema{
			"name": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "Name for the new cluster.",
			},
			"datacenter_id": {
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
				Description: "The managed object ID of the datacenter to put the cluster in.",
			},
			"folder": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "The name of the folder to locate the cluster in.",
				StateFunc:   folder.NormalizePath,
			},
			// DRS - General/automation
			"drs_enabled": {
				Type:        schema.TypeBool,
				Optional:    true,
				Default:     false,
				Description: "Enable DRS for this cluster.",
			},
			"drs_automation_level": {
				Type:         schema.TypeString,
				Optional:     true,
				Default:      string(types.DrsBehaviorManual),
				Description:  "The default automation level for all virtual machines in this cluster.",
				ValidateFunc: validation.StringInSlice(drsBehaviorAllowedValues, false),
			},
			"drs_migration_threshold": {
				Type:         schema.TypeInt,
				Optional:     true,
				Default:      3,
				Description:  "A value between 1 to 5 indicating the threshold of imbalance tolerated between hosts. A lower setting will tolerate more imbalance while a higher setting will tolerate less.",
				ValidateFunc: validation.IntBetween(1, 5),
			},
			"drs_enable_vm_overrides": {
				Type:        schema.TypeBool,
				Optional:    true,
				Default:     true,
				Description: "When true, allows individual VM overrides within this cluster to be set.",
			},
			"drs_enable_predictive_drs": {
				Type:        schema.TypeBool,
				Optional:    true,
				Description: "When true, enables DRS to use data from vRealize Operations Manager to make proactive DRS recommendations.",
			},
			// DRS - DPM
			"dpm_enabled": {
				Type:        schema.TypeBool,
				Optional:    true,
				Default:     false,
				Description: "Enable DPM support for DRS. This allows you to dynamically control the power of hosts depending on the needs of virtual machines in the cluster. Requires that DRS be enabled.",
			},
			"dpm_automation_level": {
				Type:         schema.TypeString,
				Optional:     true,
				Default:      string(types.DpmBehaviorManual),
				Description:  "The automation level for host power operations in this cluster.",
				ValidateFunc: validation.StringInSlice(dpmBehaviorAllowedValues, false),
			},
			"dpm_threshold": {
				Type:         schema.TypeInt,
				Optional:     true,
				Default:      3,
				Description:  "A value between 1 to 5 indicating the threshold of load within the cluster that influences host power operations. This affects both power on and power off operations - a lower setting will tolerate more of a surplus/deficit than a higher setting.",
				ValidateFunc: validation.IntBetween(1, 5),
			},
			// DRS - Advanced options
			"drs_advanced_options": {
				Type:        schema.TypeMap,
				Optional:    true,
				Description: "Advanced configuration options for DRS and DPM.",
			},
			// HA - General
			"ha_enabled": {
				Type:        schema.TypeBool,
				Optional:    true,
				Default:     false,
				Description: "Enable vSphere HA for this cluster.",
			},
			// HA - Host monitoring settings
			"ha_host_monitoring": {
				Type:         schema.TypeString,
				Optional:     true,
				Default:      string(types.ClusterDasConfigInfoServiceStateEnabled),
				Description:  "Global setting that controls whether vSphere HA remediates VMs on host failure. Can be one of enabled or disabled.",
				ValidateFunc: validation.StringInSlice(clusterDasConfigInfoServiceStateAllowedValues, false),
			},
			// Host monitoring - VM restarts
			"ha_default_vm_restart_priority": {
				Type:         schema.TypeString,
				Optional:     true,
				Default:      string(types.ClusterDasVmSettingsRestartPriorityMedium),
				Description:  "The default restart priority for affected VMs when vSphere detects a host failure. Can be one of lowest, low, medium, high, or highest.",
				ValidateFunc: validation.StringInSlice(computeClusterDasConfigInfoServiceStateAllowedValues, false),
			},
			"ha_vm_dependency_restart_condition": {
				Type:         schema.TypeString,
				Optional:     true,
				Default:      string(types.ClusterDasVmSettingsRestartPriorityMedium),
				Description:  "The condition used to determine whether or not VMs in a certain restart priority class are online, allowing HA to move on to restarting VMs on the next priority. Can be one of none, poweredOn, guestHbStatusGreen, or appHbStatusGreen.",
				ValidateFunc: validation.StringInSlice(computeClusterVMReadinessReadyConditionAllowedValues, false),
			},
			"ha_vm_restart_additional_delay": {
				Type:        schema.TypeInt,
				Optional:    true,
				Description: "Additional delay in seconds after ready condition is met. A VM is considered ready at this point.",
			},
			"ha_default_vm_restart_timeout": {
				Type:        schema.TypeInt,
				Optional:    true,
				Default:     600,
				Description: "The maximum time, in seconds, that vSphere HA will wait for virtual machines in one priority to be ready before proceeding with the next priority.",
			},
			// Host monitoring - host isolation
			"ha_host_isolation_response": {
				Type:         schema.TypeString,
				Optional:     true,
				Default:      string(types.ClusterDasVmSettingsIsolationResponseNone),
				Description:  "The action to take on virtual machines when a host has detected that it has been isolated from the rest of the cluster. Can be one of none, powerOff, or shutdown.",
				ValidateFunc: validation.StringInSlice(computeClusterDasVMSettingsIsolationResponseAllowedValues, false),
			},
			// Datastore monitoring - Permanent Device Loss
			"ha_datastore_pdl_response": {
				Type:         schema.TypeString,
				Optional:     true,
				Default:      string(types.ClusterVmComponentProtectionSettingsStorageVmReactionDisabled),
				Description:  "The action to take on virtual machines when the cluster had detected a permanent device loss to a relevant datastore. Can be one of none, warning, or restartAggressive.",
				ValidateFunc: validation.StringInSlice(computeClusterVMStorageProtectionForPDLAllowedValues, false),
			},
			// Datastore monitoring - All Paths Down
			"ha_datastore_apd_response": {
				Type:         schema.TypeString,
				Optional:     true,
				Default:      string(types.ClusterVmComponentProtectionSettingsStorageVmReactionDisabled),
				Description:  "The action to take on virtual machines when the cluster had detected loss to all paths to a relevant datastore. Can be one of none, warning, restartConservative, or restartAggressive.",
				ValidateFunc: validation.StringInSlice(computeClusterVMStorageProtectionForAPDAllowedValues, false),
			},
			"ha_datastore_apd_recovery_action": {
				Type:         schema.TypeString,
				Optional:     true,
				Default:      string(types.ClusterVmComponentProtectionSettingsVmReactionOnAPDClearedNone),
				Description:  "The action to take on virtual machines if an APD status on an affected datastore clears in the middle of an APD event. Can be one of none or reset.",
				ValidateFunc: validation.StringInSlice(computeClusterVMReactionOnAPDClearedAllowedValues, false),
			},
			"ha_datastore_apd_response_delay": {
				Type:        schema.TypeInt,
				Optional:    true,
				Default:     3,
				Description: "The delay in minutes to wait after an APD timeout event to execute the response action defined in ha_datastore_apd_response.",
			},
			// VM monitoring
			"ha_vm_monitoring": {
				Type:         schema.TypeString,
				Optional:     true,
				Default:      string(types.ClusterDasConfigInfoVmMonitoringStateVmMonitoringDisabled),
				Description:  "The type of virtual machine monitoring to use when HA is enabled in the cluster. Can be one of vmMonitoringDisabled, vmMonitoringOnly, or vmAndAppMonitoring.",
				ValidateFunc: validation.StringInSlice(clusterDasConfigInfoVMMonitoringStateAllowedValues, false),
			},
			"ha_vm_failure_interval": {
				Type:        schema.TypeInt,
				Optional:    true,
				Default:     30,
				Description: "If a heartbeat from a virtual machine is not received within this configured interval, the virtual machine is marked as failed. The value is in seconds.",
			},
			"ha_vm_minimum_uptime": {
				Type:        schema.TypeInt,
				Optional:    true,
				Default:     120,
				Description: "The time, in seconds, that HA waits after powering on a virtual machine before monitoring for heartbeats.",
			},
			"ha_vm_maximum_resets": {
				Type:        schema.TypeInt,
				Optional:    true,
				Default:     3,
				Description: "The maximum number of resets that HA will perform to a virtual machine when responding to a failure event.",
			},
			"ha_vm_maximum_failure_window": {
				Type:        schema.TypeInt,
				Optional:    true,
				Default:     -1,
				Description: "The length of the reset window in which ha_vm_maximum_resets can operate. When this window expires, no more resets are attempted regardless of the setting configured in ha_vm_maximum_resets. -1 means no window, meaning an unlimited reset time is allotted.",
			},
			// Admission control
			"ha_admission_control_policy": {
				Type:         schema.TypeString,
				Optional:     true,
				Default:      clusterAdmissionControlTypeResourcePercentage,
				Description:  "The type of admission control policy to use with vSphere HA, which controls whether or specific VM operations are permitted in the cluster in order to protect the reliability of the cluster. Can be one of resourcePercentage, slotPolicy, failoverHosts, or disabled. Note that disabling admission control is not recommended and can lead to service issues.",
				ValidateFunc: validation.StringInSlice(clusterAdmissionControlTypeAllowedValues, false),
			},
			"ha_admission_control_host_failure_tolerance": {
				Type:        schema.TypeInt,
				Optional:    true,
				Default:     1,
				Description: "The maximum number of failed hosts that admission control tolerates when making decisions on whether to permit virtual machine operations. The maximum is one less than the number of hosts in the cluster.",
			},
			"ha_admission_control_performace_tolerance": {
				Type:         schema.TypeInt,
				Optional:     true,
				Default:      100,
				Description:  "The percentage of resource reduction that a cluster of VMs can tolerate in case of a failover. A value of 0 produces warnings only, whereas a value of 100 disables the setting.",
				ValidateFunc: validation.IntBetween(0, 100),
			},
			"ha_admission_control_resource_percentage_auto_compute": {
				Type:        schema.TypeBool,
				Optional:    true,
				Default:     true,
				Description: "When ha_admission_control_policy is resourcePercentage, automatically determine available resource percentages by subtracting the average number of host resources represented by the ha_admission_control_host_failure_tolerance setting from the total amount of resources in the cluster. Disable to supply user-defined values.",
			},
			"ha_admission_control_resource_percentage_cpu": {
				Type:         schema.TypeInt,
				Optional:     true,
				Default:      100,
				Description:  "When ha_admission_control_policy is resourcePercentage, this controls the user-defined percentage of CPU resources in the cluster to reserve for failover.",
				ValidateFunc: validation.IntBetween(1, 100),
			},
			"ha_admission_control_resource_percentage_memory": {
				Type:         schema.TypeInt,
				Optional:     true,
				Default:      100,
				Description:  "When ha_admission_control_policy is resourcePercentage, this controls the user-defined percentage of CPU resources in the cluster to reserve for failover.",
				ValidateFunc: validation.IntBetween(1, 100),
			},
			"ha_admission_control_slot_policy_use_explicit_size": {
				Type:        schema.TypeBool,
				Optional:    true,
				Description: "When ha_admission_control_policy is slotPolicy, this setting controls whether or not you wish to supply explicit values to CPU and memory slot sizes. The default is to gather a automatic average based on all powered-on virtual machines currently in the cluster.",
			},
			"ha_admission_control_slot_policy_explicit_cpu": {
				Type:        schema.TypeInt,
				Optional:    true,
				Default:     32,
				Description: "When ha_admission_control_policy is slotPolicy, this controls the user-defined CPU slot size, in MHz.",
			},
			"ha_admission_control_slot_policy_explicit_memory": {
				Type:        schema.TypeInt,
				Optional:    true,
				Default:     100,
				Description: "When ha_admission_control_policy is slotPolicy, this controls the user-defined memory slot size, in MB.",
			},
			"ha_admission_control_failover_host_system_ids": {
				Type:        schema.TypeSet,
				Optional:    true,
				Description: "When ha_admission_control_policy is failoverHosts, this defines the managed object IDs of hosts to use as dedicated failover hosts. These hosts are kept as available as possible - admission control will block access to the host, and DRS will ignore the host when making recommendations.",
				Elem:        &schema.Schema{Type: schema.TypeString},
			},
			// HA - datastores
			"ha_heartbeat_datastore_policy": {
				Type:         schema.TypeString,
				Optional:     true,
				Default:      string(types.ClusterDasConfigInfoHBDatastoreCandidateAllFeasibleDsWithUserPreference),
				Description:  "The selection policy for HA heartbeat datastores. Can be one of allFeasibleDs, userSelectedDs, or allFeasibleDsWithUserPreference.",
				ValidateFunc: validation.StringInSlice(clusterDasConfigInfoHBDatastoreCandidateAllowedValues, false),
			},
			"ha_heartbeat_datastores": {
				Type:        schema.TypeSet,
				Optional:    true,
				Description: "The list of managed object IDs for preferred datastores to use for HA heartbeating. This setting is only useful when ha_heartbeat_datastore_policy is set to either userSelectedDs or allFeasibleDsWithUserPreference.",
				Elem:        &schema.Schema{Type: schema.TypeString},
			},
			// HA - Advanced options
			"ha_advanced_options": {
				Type:        schema.TypeMap,
				Optional:    true,
				Description: "Advanced configuration options for vSphere HA.",
			},
			// Proactive HA
			"proactive_ha_enabled": {
				Type:        schema.TypeBool,
				Optional:    true,
				Description: "Enables proactive HA, allowing for vSphere to get HA data from external providers and use DRS to perform remediation.",
			},
			"proactive_ha_behavior": {
				Type:         schema.TypeString,
				Optional:     true,
				Default:      string(types.ClusterInfraUpdateHaConfigInfoBehaviorTypeManual),
				Description:  "The DRS behavior for proactive HA recommendations. Can be one of Automated or Manual.",
				ValidateFunc: validation.StringInSlice(clusterInfraUpdateHaConfigInfoBehaviorTypeAllowedValues, false),
			},
			"proactive_ha_moderate_remediation": {
				Type:         schema.TypeString,
				Optional:     true,
				Default:      string(types.ClusterInfraUpdateHaConfigInfoRemediationTypeQuarantineMode),
				Description:  "The configured remediation for moderately degraded hosts. Can be one of MaintenanceMode or QuarantineMode. Note that this cannot be set to MaintenanceMode when proactive_ha_severe_remediation is set to QuarantineMode.",
				ValidateFunc: validation.StringInSlice(clusterInfraUpdateHaConfigInfoRemediationTypeAllowedValues, false),
			},
			"proactive_ha_severe_remediation": {
				Type:         schema.TypeString,
				Optional:     true,
				Default:      string(types.ClusterInfraUpdateHaConfigInfoRemediationTypeQuarantineMode),
				Description:  "The configured remediation for severely degraded hosts. Can be one of MaintenanceMode or QuarantineMode. Note that this cannot be set to QuarantineMode when proactive_ha_moderate_remediation is set to MaintenanceMode.",
				ValidateFunc: validation.StringInSlice(clusterInfraUpdateHaConfigInfoRemediationTypeAllowedValues, false),
			},
			"proactive_ha_provider_ids": {
				Type:        schema.TypeSet,
				Optional:    true,
				Description: "The list of IDs for health update providers configured for this cluster.",
				Elem:        &schema.Schema{Type: schema.TypeString},
			},

			vSphereTagAttributeKey:    tagsSchema(),
			customattribute.ConfigKey: customattribute.ConfigSchema(),
		},
	}
}

func resourceVSphereComputeClusterCreate(d *schema.ResourceData, meta interface{}) error {
	log.Printf("[DEBUG] %s: Beginning create", resourceVSphereComputeClusterIDString(d))

	log.Printf("[DEBUG] %s: Create finished successfully", resourceVSphereComputeClusterIDString(d))
	return resourceVSphereComputeClusterRead(d, meta)
}

func resourceVSphereComputeClusterRead(d *schema.ResourceData, meta interface{}) error {
	log.Printf("[DEBUG] %s: Beginning read", resourceVSphereComputeClusterIDString(d))

	log.Printf("[DEBUG] %s: Read completed successfully", resourceVSphereComputeClusterIDString(d))
	return nil
}

func resourceVSphereComputeClusterUpdate(d *schema.ResourceData, meta interface{}) error {
	log.Printf("[DEBUG] %s: Beginning update", resourceVSphereComputeClusterIDString(d))

	log.Printf("[DEBUG] %s: Update finished successfully", resourceVSphereComputeClusterIDString(d))
	return resourceVSphereComputeClusterRead(d, meta)
}

func resourceVSphereComputeClusterDelete(d *schema.ResourceData, meta interface{}) error {
	resourceIDString := resourceVSphereComputeClusterIDString(d)
	log.Printf("[DEBUG] %s: Beginning delete", resourceIDString)

	log.Printf("[DEBUG] %s: Deleted successfully", resourceIDString)
	return nil
}

// resourceVSphereComputeClusterIDString prints a friendly string for the
// vsphere_compute_cluster resource.
func resourceVSphereComputeClusterIDString(d structure.ResourceIDStringer) string {
	return structure.ResourceIDString(d, resourceVSphereComputeClusterName)
}
