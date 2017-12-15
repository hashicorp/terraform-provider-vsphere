package vsphere

import (
	"fmt"
	"log"
	"errors"
	"strings"
	"golang.org/x/net/context"
	"github.com/hashicorp/terraform/helper/schema"
	"github.com/vmware/govmomi"
	"github.com/vmware/govmomi/find"
	"github.com/vmware/govmomi/object"
	"github.com/vmware/govmomi/property"
	"github.com/vmware/govmomi/vim25/methods"
	"github.com/vmware/govmomi/vim25/mo"
	"github.com/vmware/govmomi/vim25/types"
)

var (
	DrsBehaviors = []types.DrsBehavior{
		types.DrsBehaviorManual,
		types.DrsBehaviorPartiallyAutomated,
		types.DrsBehaviorFullyAutomated,
	}
)

func resourceVSphereClusterCompute() *schema.Resource {
	return &schema.Resource{
		Create: resourceVSphereClusterComputeCreate,
		Read: resourceVSphereClusterComputeRead,
		Update: resourceVSphereClusterComputeUpdate,
		Delete: resourceVSphereClusterComputeDelete,
		Schema: map[string]*schema.Schema{
			"cluster": &schema.Schema{
				Type:     schema.TypeString,
				ForceNew: false,
				Required: true,
			},
			"datacenter": &schema.Schema{
				Type:     schema.TypeString,
				ForceNew: false,
				Required: true,
			},
			"drs-enabled": &schema.Schema{
				Type:     schema.TypeBool,
				ForceNew: false,
				Required: true,
			},
			"drs-behavior": &schema.Schema{
				Type:     schema.TypeString,
				ForceNew: false,
				Optional: true,
				ValidateFunc: validDrsBehavior,
			},
			"drs-recommendations": &schema.Schema{
				Type:     schema.TypeInt,
				ForceNew: false,
				Optional: true,
			},
		},
		Importer: &schema.ResourceImporter{
			State: importCluster,
		},
	}
}

func importCluster(d *schema.ResourceData, meta interface{}) ([]*schema.ResourceData, error){
	log.Printf("[DEBUG] Import Data (%s)\n", d)
	client := meta.(*govmomi.Client)
	ccr, err := getClusterComputeResource(d, client)
	if err != nil {
		return nil, err
	}
	ret := make([]*schema.ResourceData, 1)
	ccrmo, err := getClusterComputeResourceManagedObject(client, ccr)
	if err != nil {
		return nil, err
	}
	ret[0] = updateResourceFromCluster(d, ccr, ccrmo)
	return ret, nil
}

func validDrsBehavior(meta interface{}, _ string) ([]string, []error) {
	arg := meta.(string)
	for _, behavior := range DrsBehaviors {
		if arg == string(behavior) {
			return nil, nil
		}
	}
	return nil, []error{
		errors.New(fmt.Sprintf("Invlalid drs-behavior attribute. Must be one of (%s)", DrsBehaviors)),
	}
}

func getClusterComputeResource(d *schema.ResourceData, client *govmomi.Client) (_ *object.ClusterComputeResource, err error) {
	finder := find.NewFinder(client.Client, true)

	var name string;
	if n, ok := d.GetOk("cluster"); ok {
		name = n.(string)
	} else {
		log.Printf("[INFO] getCluster - Couldn't get cluster. Using resource ID instead: (%s)\n", name)
		name = d.Id()
	}

	dcs, err := finder.DatacenterList(context.TODO(), "*")
	if err != nil {
		return nil, fmt.Errorf("error %s", err)
	}

	var ccr *object.ClusterComputeResource
	for _, dc := range dcs {
		finder.SetDatacenter(dc)
		ccr, err = finder.ClusterComputeResource(context.TODO(), name)
		if err == nil && ccr != nil {
			break
		}
		log.Printf("[DEBUG] getCluster - cluster (%s) not found in datacenter (%s)", name, dc.Name())
	}
	if ccr == nil {
		return nil, fmt.Errorf("[ERROR] getCluster - Error finding cluster: %v", err)
	}
	log.Printf("[DEBUG] getCluser - CLUSTER: %s", ccr)
	return ccr, err
}

func getClusterComputeResourceManagedObject(client *govmomi.Client, cluster *object.ClusterComputeResource) (_ *mo.ClusterComputeResource, err error) {
	var ccrmo mo.ClusterComputeResource
	collector := property.DefaultCollector(client.Client)
	err = collector.RetrieveOne(context.TODO(), cluster.Reference(), []string{"configuration", "drsRecommendation"}, &ccrmo);
	if err != nil {
		return nil, err
	}
	log.Printf("[DEBUG]: getClusterData - CLUSTER %s", ccrmo.Configuration.DrsConfig)
	return &ccrmo, nil
}

func updateResourceFromCluster(d *schema.ResourceData, c *object.ClusterComputeResource, cd *mo.ClusterComputeResource) (_ *schema.ResourceData) {
	d.Set("cluster", c.Name())
	d.Set("datacenter", GetDatacenterFromInventoryPath(c.InventoryPath))
	d.Set("drs-enabled", cd.Configuration.DrsConfig.Enabled)
	d.Set("drs-behavior", cd.Configuration.DrsConfig.DefaultVmBehavior)
	d.Set("drs-recommendations", len(cd.DrsRecommendation))
	return d
}

func GetDatacenterFromInventoryPath(s string) (_ string){
	for _, s := range strings.Split(s, "/") {
		if(len(s) > 0) {
			return s
		}
	}
	return s
}

func resourceVSphereClusterComputeCreate(d *schema.ResourceData, meta interface{}) (err error) {
	return
}

func resourceVSphereClusterComputeRead(d *schema.ResourceData, meta interface{}) (err error) {
	log.Print("[INFO] READING CLUSTER")
	client := meta.(*govmomi.Client)
	ccr, err := getClusterComputeResource(d, client)
	if err != nil {
		return
	}
	ccrmo, err := getClusterComputeResourceManagedObject(client, ccr)
	if err != nil {
		return
	}
	d = updateResourceFromCluster(d, ccr, ccrmo)
	log.Printf("[DEBUG] resourceVSphereClusterComputeRead - clusterData %s", ccrmo)
	return
}

func getConfigureSpecFromSchema(d *schema.ResourceData) (_ types.ClusterConfigSpec) {
	enabled := d.Get("drs-enabled").(bool)

	ccs := types.ClusterConfigSpec {
		DrsConfig: &types.ClusterDrsConfigInfo {
			Enabled: &enabled,
		},
	}

	if d, ok := d.GetOk("drs-behavior"); ok {
		log.Printf("[DEBUG] DRS behvaior was specified")
		ccs.DrsConfig.DefaultVmBehavior = types.DrsBehavior(d.(string))
	} else {
		log.Print("[DEBUG] no DRS behavior was supplied.")
	}

	return ccs
}

// "A complete cluster configuration. All fields are defined as optional.
//   In case of a reconfiguration, unset fields are unchanged."
// https://www.vmware.com/support/developer/converter-sdk/conv61_apireference/index.html
func resourceVSphereClusterComputeUpdate(d *schema.ResourceData, meta interface{}) (err error) {
	log.Print("[INFO] UPDATE CLUSTER")

	client := meta.(*govmomi.Client)

	ccr, err := getClusterComputeResource(d, client)
	if err != nil {
		return
	}
	ccrmo, err := getClusterComputeResourceManagedObject(client, ccr)
	if err != nil {
		return
	}

	ccs := getConfigureSpecFromSchema(d)
	task, err := ccr.ReconfigureCluster(context.TODO(), ccs)
	if err != nil {
		log.Print("[ERROR] error creating task to reconfigure cluster.")
		return
	}
	_, err = task.WaitForResult(context.TODO(), nil)
	if err != nil {
		log.Print("[ERROR] error waiting for reconfigure cluster task to complete.")
		return
	}

	for _, rec := range ccrmo.DrsRecommendation {
		log.Printf("[DEBUG] RECOMMENDATION (%s)", rec)
		for _, migrationSuggestion := range rec.MigrationList {
			log.Print("[INFO] applying recommendation")
			ar := &types.ApplyRecommendation{
				This: ccr.Reference(),
				Key: migrationSuggestion.Key,
			}
			methods.ApplyRecommendation(context.TODO(), client.RoundTripper, ar)
		}
	}

	return
}

func resourceVSphereClusterComputeDelete(d *schema.ResourceData, meta interface{}) (err error) {
	return
}

