package vsphere

import (
	"context"
	"fmt"
	"log"
	"strconv"
	"time"

	"github.com/hashicorp/terraform/helper/schema"
	"github.com/hashicorp/terraform/helper/validation"
	"github.com/terraform-providers/terraform-provider-vsphere/vsphere/internal/helper/cluster"
	"github.com/terraform-providers/terraform-provider-vsphere/vsphere/internal/helper/clusterrule"
	"github.com/terraform-providers/terraform-provider-vsphere/vsphere/internal/helper/computeresource"
	"github.com/terraform-providers/terraform-provider-vsphere/vsphere/internal/helper/virtualmachine"
	"github.com/vmware/govmomi/vim25/types"
)

var ruleTypeAllowedValues = []string{
	"vmhostaffine",
	"vmhostantiaffine",
	"affinity",
	"antiaffinity",
}

const DefaultAPITimeout = time.Minute * 5

type clusterRule struct {
	Id                       int32
	Name                     string
	Mandatory                bool
	Enabled                  bool
	Status                   string
	ClusterRuleType          string
	DatacenterID             string
	ClusterComputeResourceID string
	VirtualMachines          []string
	VmGroupName              string
	HostGroupName            string
}

// Define Cluster Rule
func resourceVSphereClusterRule() *schema.Resource {
	return &schema.Resource{
		Create: resourceVSphereClusterRuleCreate,
		Read:   resourceVSphereClusterRuleRead,
		Update: resourceVSphereClusterRuleUpdate,

		Delete: resourceVSphereClusterRuleDelete,

		Schema: map[string]*schema.Schema{
			"name": &schema.Schema{
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
				Description: "The name for the cluster rule.",
			},
			"type": &schema.Schema{
				Type:         schema.TypeString,
				Required:     true,
				ForceNew:     true,
				Description:  "The type for the cluster rule.",
				ValidateFunc: validation.StringInSlice(ruleTypeAllowedValues, false),
			},
			"mandatory": &schema.Schema{
				Type:        schema.TypeBool,
				Optional:    true,
				Default:     false,
				Description: "Use this option to set this rule is mandatory or optional.",
			},
			"enabled": &schema.Schema{
				Type:        schema.TypeBool,
				Optional:    true,
				Default:     true,
				Description: "Use this option to enable the rule.",
			},
			"status": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},

			"datacenter_id": &schema.Schema{
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
				Description: "The datacenter name in vSphere",
			},
			"cluster_compute_resource_id": &schema.Schema{
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
				Description: "The cluster name in vSphere",
			},
			"virtual_machines": &schema.Schema{
				Type:        schema.TypeList,
				Optional:    true,
				Description: "The list of virtual machines for the affinity",
				Elem:        &schema.Schema{Type: schema.TypeString},
			},
			"vm_group_name": &schema.Schema{
				Type:        schema.TypeString,
				Optional:    true,
				Description: "Use this option only for type 'vmhost'. The virtual machine group name",
				Elem:        &schema.Schema{Type: schema.TypeString},
			},
			"host_group_name": &schema.Schema{
				Type:        schema.TypeString,
				Optional:    true,
				Description: "Use this option only for type 'vmhost'. The host group name.",
			},
		},
	}
}

//Fork from terraform-provider-aws/aws/structure.go
func expandStringList(configured []interface{}) []string {
	vs := make([]string, 0, len(configured))
	for _, v := range configured {
		val, ok := v.(string)
		if ok && val != "" {
			vs = append(vs, v.(string))
		}
	}
	return vs
}

func createClusterRule(d *schema.ResourceData) (*clusterRule, error) {
	var cr clusterRule

	if d.Id() != "" {
		id, err := strconv.Atoi(d.Id())
		if err != nil {
			return nil, fmt.Errorf("Unable to convert Id to int32.")
		}
		cr.Id = int32(id)
	}
	if name, ok := d.GetOk("name"); ok {
		cr.Name = name.(string)
	}
	if crType, ok := d.GetOk("type"); ok {
		cr.ClusterRuleType = crType.(string)
	}
	if datacenter_id, ok := d.GetOk("datacenter_id"); ok {
		cr.DatacenterID = datacenter_id.(string)
	}
	if ccrID, ok := d.GetOk("cluster_compute_resource_id"); ok {
		cr.ClusterComputeResourceID = ccrID.(string)
	}
	if vms, ok := d.GetOk("virtual_machines"); ok {
		cr.VirtualMachines = expandStringList(vms.([]interface{}))
	}
	if vmgn, ok := d.GetOk("vm_group_name"); ok {
		cr.VmGroupName = vmgn.(string)
	}
	if hgn, ok := d.GetOk("host_group_name"); ok {
		cr.HostGroupName = hgn.(string)
	}
	if m, ok := d.GetOk("enabled"); ok {
		cr.Enabled = m.(bool)
	}
	if e, ok := d.GetOk("mandatory"); ok {
		cr.Mandatory = e.(bool)
	}
	if s, ok := d.GetOk("status"); ok {
		cr.Status = s.(string)
	}
	return &cr, nil
}

func resourceVSphereClusterRuleCreate(d *schema.ResourceData, meta interface{}) error {
	log.Printf("[INFO] Create Cluster Rule")
	client := meta.(*VSphereClient).vimClient

	cr, err := createClusterRule(d)
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), DefaultAPITimeout)
	defer cancel()
	//Get datacenter
	dc, err := datacenterFromID(client, cr.DatacenterID)
	if err != nil {
		return err
	}
	//Get cluster
	ccr, err := computeresource.ClusterFromID(client, cr.ClusterComputeResourceID)
	if err != nil {
		return err
	}

	//Check rule existance
	//Issue github.com/vmware/govmomi/issues/980
	ok, err := clusterrule.CheckExist(ctx, ccr, cr.Name)
	if err != nil {
		return err
	}

	if ok {
		return fmt.Errorf("Rule name already exists")
	}

	//create
	var rule types.BaseClusterRuleInfo
	switch cr.ClusterRuleType {
	case "antiaffinity":
		aarule := &types.ClusterAntiAffinityRuleSpec{}
		refvms, err := virtualmachine.GetVmsRefFromPaths(client, cr.VirtualMachines, dc)
		if err != nil {
			return err
		}
		aarule.Vm = refvms
		rule = aarule
	case "affinity":
		arule := &types.ClusterAffinityRuleSpec{}
		refvms, err := virtualmachine.GetVmsRefFromPaths(client, cr.VirtualMachines, dc)
		if err != nil {
			return err
		}
		arule.Vm = refvms
		rule = arule
	case "vmhostaffine":
		vmhrule := &types.ClusterVmHostRuleInfo{}
		vmhrule.VmGroupName = cr.VmGroupName
		vmhrule.AffineHostGroupName = cr.HostGroupName
		rule = vmhrule
	case "vmhostantiaffine":
		vmhrule := &types.ClusterVmHostRuleInfo{}
		vmhrule.VmGroupName = cr.VmGroupName
		vmhrule.AntiAffineHostGroupName = cr.HostGroupName
		rule = vmhrule

	}

	ruleinfo := rule.GetClusterRuleInfo()
	ruleinfo.Name = cr.Name
	ruleinfo.Mandatory = &cr.Mandatory
	ruleinfo.Enabled = &cr.Enabled

	spec := types.ClusterRuleSpec{}
	spec.Operation = types.ArrayUpdateOperationAdd
	spec.Info = rule

	clusterSpec := &types.ClusterConfigSpecEx{
		RulesSpec: []types.ClusterRuleSpec{
			spec,
		},
	}

	task, err := ccr.Reconfigure(ctx, clusterSpec, true)
	if err != nil {
		return err
	}
	_, err = task.WaitForResult(ctx, nil)
	if err != nil {
		return err
	}

	return resourceVSphereClusterRuleRead(d, meta)
}

func resourceVSphereClusterRuleUpdate(d *schema.ResourceData, meta interface{}) error {
	log.Printf("[DEBUG] Update Cluster Rule.")
	client := meta.(*VSphereClient).vimClient

	cr, err := createClusterRule(d)
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), DefaultAPITimeout)
	defer cancel()

	//Get datacenter
	dc, err := datacenterFromID(client, cr.DatacenterID)
	if err != nil {
		return err
	}

	//Get cluster
	ccr, err := computeresource.ClusterFromID(client, cr.ClusterComputeResourceID)
	if err != nil {
		return err
	}

	//Update
	cluserConfig, err := cluster.GetClusterConfigInfo(client, cr.ClusterComputeResourceID)
	if err != nil {
		return err
	}

	var crule types.BaseClusterRuleInfo
	for _, crule = range cluserConfig.Rule {
		info := crule.GetClusterRuleInfo()
		if cr.Id == info.Key {
			info.Name = cr.Name
			info.Mandatory = &cr.Mandatory
			info.Enabled = &cr.Enabled
			break
			//Update and push
		}
	}
	if crule == nil {
		return fmt.Errorf("Could not find rule key : %v\n", cr.Id)
	}

	switch crule.(type) {
	case *types.ClusterAntiAffinityRuleSpec:
		refvms, err := virtualmachine.GetVmsRefFromPaths(client, cr.VirtualMachines, dc)
		if err != nil {
			return err
		}
		crule.(*types.ClusterAntiAffinityRuleSpec).Vm = refvms

	case *types.ClusterAffinityRuleSpec:
		refvms, err := virtualmachine.GetVmsRefFromPaths(client, cr.VirtualMachines, dc)
		if err != nil {
			return err
		}
		crule.(*types.ClusterAffinityRuleSpec).Vm = refvms

	case *types.ClusterVmHostRuleInfo:
		crule.(*types.ClusterVmHostRuleInfo).VmGroupName = cr.VmGroupName
		crule.(*types.ClusterVmHostRuleInfo).AffineHostGroupName = cr.HostGroupName
		crule.(*types.ClusterVmHostRuleInfo).AntiAffineHostGroupName = cr.HostGroupName
	}

	spec := types.ClusterRuleSpec{}
	spec.Operation = types.ArrayUpdateOperationEdit
	spec.Info = crule

	clusterSpec := &types.ClusterConfigSpecEx{
		RulesSpec: []types.ClusterRuleSpec{
			spec,
		},
	}

	task, err := ccr.Reconfigure(ctx, clusterSpec, true)
	if err != nil {
		return err
	}
	_, err = task.WaitForResult(ctx, nil)
	if err != nil {
		return err
	}

	return resourceVSphereClusterRuleRead(d, meta)
}

func resourceVSphereClusterRuleRead(d *schema.ResourceData, meta interface{}) error {
	log.Printf("[DEBUG] Reading Cluster Rule.")
	client := meta.(*VSphereClient).vimClient

	cr, err := createClusterRule(d)
	if err != nil {
		return err
	}

	cluserConfig, err := cluster.GetClusterConfigInfo(client, cr.ClusterComputeResourceID)
	if err != nil {
		return err
	}

	for _, crule := range cluserConfig.Rule {
		info := crule.GetClusterRuleInfo()
		if cr.Name == info.Name {
			d.SetId(fmt.Sprint(info.Key))
			d.Set("name", info.Name)
			d.Set("mandatory", *info.Mandatory)
			d.Set("enabled", *info.Enabled)
			d.Set("status", info.Status)
			switch v := crule.(type) {
			case *types.ClusterAntiAffinityRuleSpec:
				d.Set("type", "antiaffinity")
				vmNames, err := virtualmachine.ConvertManagedObjectReferenceToName(client, crule.(*types.ClusterAntiAffinityRuleSpec).Vm)
				if err != nil {
					return err
				}
				d.Set("virtual_machines", vmNames)

			case *types.ClusterAffinityRuleSpec:
				d.Set("type", "affinity")
				vmNames, err := virtualmachine.ConvertManagedObjectReferenceToName(client, crule.(*types.ClusterAffinityRuleSpec).Vm)
				if err != nil {
					return err
				}

				d.Set("virtual_machines", vmNames)

			case *types.ClusterVmHostRuleInfo:
				cri := crule.(*types.ClusterVmHostRuleInfo)
				d.Set("vm_group_name", cri.VmGroupName)
				if cri.AntiAffineHostGroupName != "" && cri.AffineHostGroupName == "" {
					d.Set("type", "vmhostantiaffine")
					d.Set("host_group_name", cri.AntiAffineHostGroupName)
				}
				if cri.AntiAffineHostGroupName == "" && cri.AffineHostGroupName != "" {
					d.Set("type", "vmhostaffine")
					d.Set("host_group_name", cri.AffineHostGroupName)
				}
			default:
				return fmt.Errorf("Error during reading ClusterRule type : unknown %v", v)

			}
		}
	}

	return nil

}
func resourceVSphereClusterRuleDelete(d *schema.ResourceData, meta interface{}) error {
	log.Printf("[INFO] Deleting Cluster Rule")
	client := meta.(*VSphereClient).vimClient

	cr, err := createClusterRule(d)
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), DefaultAPITimeout)
	defer cancel()

	spec := types.ClusterRuleSpec{}
	spec.Operation = types.ArrayUpdateOperationRemove
	spec.RemoveKey = cr.Id

	clusterSpec := &types.ClusterConfigSpecEx{
		RulesSpec: []types.ClusterRuleSpec{
			spec,
		},
	}

	ccr, err := computeresource.ClusterFromID(client, cr.ClusterComputeResourceID)
	if err != nil {
		return err
	}
	task, err := ccr.Reconfigure(ctx, clusterSpec, true)
	if err != nil {
		return err
	}
	_, err = task.WaitForResult(ctx, nil)
	if err != nil {
		return err
	}

	d.SetId("")

	return nil
}
