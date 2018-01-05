package vsphere

import (
	"context"
	"fmt"
	"log"
	"strconv"
	"time"

	"github.com/hashicorp/terraform/helper/schema"
	"github.com/hashicorp/terraform/helper/validation"
	"github.com/terraform-providers/terraform-provider-vsphere/vsphere/internal/helper/computeresource"
	"github.com/vmware/govmomi/find"
	"github.com/vmware/govmomi/object"
	"github.com/vmware/govmomi/vim25/types"
)

var ruleTypeAllowedValues = []string{
	"vmhostaffine",
	"vmhostantiaffine",
	"affinity",
	"antiaffinity",
}

const DefaultAPITimeout = time.Minute * 5

//func boolPtr(b bool) *bool {
//	return &b
//}

type clusterRule struct {
	Id                       int32
	Name                     string
	Mandatory                bool
	Enabled                  bool
	Status                   string
	ClusterRuleType          string
	HostSystemID             string
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
				Type:        schema.TypeBool,
				Compute:     true,
				Description: "The rule status.",
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

func checkExist(ctx context.Context, c *object.ClusterComputeResource, name string) (bool, error) {
	ret, err := getRule(ctx, c, name)
	return ret != nil, err
}

func getRule(ctx context.Context, c *object.ClusterComputeResource, name string) (types.BaseClusterRuleInfo, error) {
	cluserConfig, err := c.Configuration(ctx)
	if err != nil {
		return nil, err
	}
	for _, crule := range cluserConfig.Rule {
		info := crule.GetClusterRuleInfo()
		if info.Name == name {
			return info, nil
		}
	}
	return nil, nil
}

func resourceVSphereClusterRuleCreate(d *schema.ResourceData, meta interface{}) error {
	log.Printf("[INFO] Creating Cluster Rule")
	client := meta.(*VSphereClient).vimClient

	cr, err := createClusterRule(d)
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), DefaultAPITimeout)
	defer cancel()

	dc, err := datacenterFromID(client, cr.DatacenterID)
	if err != nil {
		return err
	}

	finder := find.NewFinder(client.Client, true)
	finder.SetDatacenter(dc)

	var refVMs []types.ManagedObjectReference
	for _, vmNames := range cr.VirtualMachines {
		vms, err := finder.VirtualMachineList(ctx, vmNames)
		if err != nil {
			return err
		}

		for _, vm := range vms {
			ref := types.ManagedObjectReference{
				Type:  "VirtualMachine",
				Value: vm.Reference().Value,
			}
			refVMs = append(refVMs, ref)
		}
	}
	var ruleSpecs []types.ClusterRuleSpec
	var rule types.BaseClusterRuleInfo
	//TODO Add other types
	switch cr.ClusterRuleType {
	case "antiaffinity":
		aaRule := &types.ClusterAntiAffinityRuleSpec{}
		aaRule.Vm = refVMs
		rule = aaRule
	case "affinity":
		aRule := &types.ClusterAffinityRuleSpec{}
		aRule.Vm = refVMs
		rule = aRule
	case "vmhostaffine":
		vmhRule := &types.ClusterVmHostRuleInfo{}
		vmhRule.VmGroupName = cr.VmGroupName
		vmhRule.AffineHostGroupName = cr.HostGroupName
		rule = vmhRule
	case "vmhostantiaffine":
		vmhRule := &types.ClusterVmHostRuleInfo{}
		vmhRule.VmGroupName = cr.VmGroupName
		vmhRule.AffineHostGroupName = cr.HostGroupName
		rule = vmhRule

	}
	ruleInfo := rule.GetClusterRuleInfo()
	ruleInfo.Name = cr.Name
	ruleInfo.Mandatory = &cr.Mandatory
	ruleInfo.Enabled = &cr.Enabled
	spec := types.ClusterRuleSpec{}
	spec.Operation = types.ArrayUpdateOperationAdd
	spec.Info = rule
	ruleSpecs = append(ruleSpecs, spec)

	clusterSpec := &types.ClusterConfigSpecEx{RulesSpec: ruleSpecs}
	ccr, err := computeresource.ClusterFromID(client, cr.ClusterComputeResourceID)
	if err != nil {
		return err
	}

	//Issue github.com/vmware/govmomi/issues/980
	ok, err := checkExist(ctx, ccr, cr.Name)
	if err != nil {
		return err
	}
	if ok {
		return fmt.Errorf("Rule name already exists")
	}

	task, err := ccr.Reconfigure(ctx, clusterSpec, true)
	if err != nil {
		return err
	}
	_, err = task.WaitForResult(ctx, nil)
	if err != nil {
		return err
	}

	//Get rule Key
	resRule, err := getRule(ctx, ccr, cr.Name)
	if err != nil {
		return err
	}
	cr.Id = resRule.GetClusterRuleInfo().Key
	d.SetId(fmt.Sprint(cr.Id))
	return resourceVSphereClusterRuleRead(d, meta)
}

func resourceVSphereClusterRuleRead(d *schema.ResourceData, meta interface{}) error {
	log.Printf("[DEBUG] Reading Cluster Rule.")
	//client := meta.(*VSphereClient).vimClient
	return nil

}

func resourceVSphereClusterRuleUpdate(d *schema.ResourceData, meta interface{}) error {
	log.Printf("[DEBUG] Reading Cluster Rule.")
	//client := meta.(*VSphereClient).vimClient
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

	finder := find.NewFinder(client.Client, true)

	dc, err := datacenterFromID(client, cr.DatacenterID)
	if err != nil {
		return err
	}

	finder.SetDatacenter(dc)

	var ruleSpecs []types.ClusterRuleSpec

	spec := types.ClusterRuleSpec{}
	spec.Operation = types.ArrayUpdateOperationRemove
	spec.RemoveKey = cr.Id
	ruleSpecs = append(ruleSpecs, spec)

	clusterSpec := &types.ClusterConfigSpecEx{RulesSpec: ruleSpecs}
	cluster, err := computeresource.ClusterFromID(client, cr.ClusterComputeResourceID)
	if err != nil {
		return err
	}
	task, err := cluster.Reconfigure(ctx, clusterSpec, true)
	if err != nil {
		return err
	}
	_, err = task.WaitForResult(ctx, nil)
	if err != nil {
		return err
	}

	d.SetId(cr.Name)

	return nil
}
