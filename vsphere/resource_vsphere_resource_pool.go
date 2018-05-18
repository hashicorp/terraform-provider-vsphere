package vsphere

import (
	//"fmt"
	"log"

	"github.com/hashicorp/terraform/helper/schema"
	"github.com/hashicorp/terraform/helper/validation"
	"github.com/terraform-providers/terraform-provider-vsphere/vsphere/internal/helper/resourcepool"
	"github.com/terraform-providers/terraform-provider-vsphere/vsphere/internal/helper/structure"
	"github.com/terraform-providers/terraform-provider-vsphere/vsphere/internal/virtualdevice"
	"github.com/vmware/govmomi/vim25/types"
)

const resourceVSphereResourcePoolName = "vsphere_resource_pool"

func resourceVSphereResourcePool() *schema.Resource {
	s := map[string]*schema.Schema{
		"name": {
			Type:        schema.TypeString,
			Required:    true,
			Description: "Name of resource pool.",
		},
		"root_resource_pool_id": {
			Type:        schema.TypeString,
			Description: "The ID of the root resource pool of the compute resource the resource pool is in.",
			Required:    true,
		},
		"cpu_share_level": {
			Type:         schema.TypeString,
			Description:  "The allocation level. The level is a simplified view of shares. Levels map to a pre-determined set of numeric values for shares. Can be one of low, normal, or high. Default: normal",
			Optional:     true,
			ValidateFunc: validation.StringInSlice(virtualdevice.SharesLevelAllowedValues, false),
			Default:      "normal",
		},
		"cpu_shares": {
			Type:        schema.TypeInt,
			Description: "The number of shares allocated. Used to determine resource allocation in case of resource contention. If this is set, cpu_share_level must be custom",
			Computed:    true,
			Optional:    true,
		},
		"cpu_reservation": {
			Type:        schema.TypeInt,
			Description: "Amount of CPU (MHz) that is guaranteed available to the resource pool. Default: 0",
			Optional:    true,
			Default:     0,
		},
		"cpu_expandable": {
			Type:        schema.TypeBool,
			Description: "Determines if the reservation on a resource pool can grow beyond the specified value, if the parent resource pool has unreserved resources. Default: true",
			Optional:    true,
			Default:     true,
		},
		"cpu_limit": {
			Type:        schema.TypeInt,
			Description: "The utilization of a resource pool will not exceed this limit, even if there are available resources. Set to -1 for unlimited. Default: -1",
			Optional:    true,
			Default:     -1,
		},
		"memory_share_level": {
			Type:         schema.TypeString,
			Description:  "The allocation level. The level is a simplified view of shares. Levels map to a pre-determined set of numeric values for shares. Can be one of low, normal, high, or custom. Default: normal",
			Optional:     true,
			ValidateFunc: validation.StringInSlice(virtualdevice.SharesLevelAllowedValues, false),
			Default:      "normal",
		},
		"memory_shares": {
			Type:        schema.TypeInt,
			Description: "The number of shares allocated. Used to determine resource allocation in case of resource contention. If this is set, memory_share_level must be custom",
			Computed:    true,
			Optional:    true,
		},
		"memory_reservation": {
			Type:        schema.TypeInt,
			Description: "Amount of memory (MB) that is guaranteed available to the resource pool. Default: 0",
			Optional:    true,
			Default:     0,
		},
		"memory_expandable": {
			Type:        schema.TypeBool,
			Description: "Determines if the reservation on a resource pool can grow beyond the specified value, if the parent resource pool has unreserved resources. Default: true",
			Optional:    true,
			Default:     true,
		},
		"memory_limit": {
			Type:        schema.TypeInt,
			Description: "The utilization of a resource pool will not exceed this limit, even if there are available resources. Set to -1 for unlimited. Default: -1",
			Optional:    true,
			Default:     -1,
		},
	}
	return &schema.Resource{
		Create: resourceVSphereResourcePoolCreate,
		Read:   resourceVSphereResourcePoolRead,
		Update: resourceVSphereResourcePoolUpdate,
		Delete: resourceVSphereResourcePoolDelete,
		Importer: &schema.ResourceImporter{
			State: resourceVSphereResourcePoolImport,
		},
		SchemaVersion: 3,
		Schema:        s,
	}
}

func resourceVSphereResourcePoolImport(d *schema.ResourceData, meta interface{}) ([]*schema.ResourceData, error) {
	client := meta.(*VSphereClient).vimClient
	dc, _ := datacenterFromID(client, "hashi-dc")
	rp, err := resourcepool.FromPathOrDefault(client, d.Id(), dc)
	if err != nil {
		return nil, err
	}
	d.SetId(rp.Reference().Value)
	rpProps, err := resourcepool.Properties(rp)
	if err != nil {
		return nil, err
	}
	err = flattenResourcePoolConfigSpec(d, &rpProps.Config)
	if err != nil {
		return nil, err
	}
	return []*schema.ResourceData{d}, nil
}

func resourceVSphereResourcePoolCreate(d *schema.ResourceData, meta interface{}) error {
	log.Printf("[DEBUG] %s: Beginning create", resourceVSphereResourcePoolIDString(d))
	client := meta.(*VSphereClient).vimClient
	rrp, err := resourcepool.FromID(client, d.Get("root_resource_pool_id").(string))
	if err != nil {
		return err
	}
	rpSpec := expandResourcePoolConfigSpec(d)
	rp, err := resourcepool.Create(rrp, d.Get("name").(string), rpSpec)
	if err != nil {
		return err
	}
	d.SetId(rp.Reference().Value)
	log.Printf("[DEBUG] %s: Create finished successfully", resourceVSphereResourcePoolIDString(d))
	return nil
}

func resourceVSphereResourcePoolRead(d *schema.ResourceData, meta interface{}) error {
	log.Printf("[DEBUG] %s: Beginning read", resourceVSphereResourcePoolIDString(d))
	client := meta.(*VSphereClient).vimClient
	rp, err := resourcepool.FromID(client, d.Id())
	if err != nil {
		return err
	}
	_ = d.Set("name", rp.Name())
	rpProps, err := resourcepool.Properties(rp)
	if err != nil {
		return err
	}
	err = flattenResourcePoolConfigSpec(d, &rpProps.Config)
	if err != nil {
		return err
	}
	log.Printf("[DEBUG] %s: Read finishes successfully", resourceVSphereResourcePoolIDString(d))
	return nil
}

func resourceVSphereResourcePoolUpdate(d *schema.ResourceData, meta interface{}) error {
	log.Printf("[DEBUG] %s: Beginning update", resourceVSphereResourcePoolIDString(d))
	client := meta.(*VSphereClient).vimClient
	rp, err := resourcepool.FromID(client, d.Id())
	if err != nil {
		return err
	}
	rpSpec := expandResourcePoolConfigSpec(d)
	err = resourcepool.Update(rp, d.Get("name").(string), rpSpec)
	if err != nil {
		return err
	}
	log.Printf("[DEBUG] %s: Update finished successfully", resourceVSphereResourcePoolIDString(d))
	return nil
}

func resourceVSphereResourcePoolDelete(d *schema.ResourceData, meta interface{}) error {
	log.Printf("[DEBUG] %s: Beginning delete", resourceVSphereResourcePoolIDString(d))
	client := meta.(*VSphereClient).vimClient
	rp, err := resourcepool.FromID(client, d.Id())
	if err != nil {
		return err
	}
	err = resourcepool.Delete(rp)
	if err != nil {
		return err
	}
	log.Printf("[DEBUG] %s: Deleted successfully", resourceVSphereResourcePoolIDString(d))
	return nil
}

// resourceVSphereResourcePoolIDString prints a friendly string for the
// vsphere_virtual_machine resource.
func resourceVSphereResourcePoolIDString(d structure.ResourceIDStringer) string {
	return structure.ResourceIDString(d, "vsphere_resource_pool")
}

func flattenResourcePoolConfigSpec(d *schema.ResourceData, obj *types.ResourceConfigSpec) error {
	err := flattenResourcePoolMemoryAllocation(d, obj.MemoryAllocation)
	if err != nil {
		return err
	}
	return flattenResourcePoolCPUAllocation(d, obj.CpuAllocation)
}

func flattenResourcePoolCPUAllocation(d *schema.ResourceData, obj types.ResourceAllocationInfo) error {
	return structure.SetBatch(d, map[string]interface{}{
		"cpu_reservation": obj.Reservation,
		"cpu_expandable":  obj.ExpandableReservation,
		"cpu_limit":       obj.Limit,
		"cpu_shares":      obj.Shares.Shares,
		"cpu_share_level": obj.Shares.Level,
	})
}

func flattenResourcePoolMemoryAllocation(d *schema.ResourceData, obj types.ResourceAllocationInfo) error {
	return structure.SetBatch(d, map[string]interface{}{
		"memory_reservation": obj.Reservation,
		"memory_expandable":  obj.ExpandableReservation,
		"memory_limit":       obj.Limit,
		"memory_shares":      obj.Shares.Shares,
		"memory_share_level": obj.Shares.Level,
	})
}

func expandResourcePoolConfigSpec(d *schema.ResourceData) *types.ResourceConfigSpec {
	return &types.ResourceConfigSpec{
		CpuAllocation:    *expandResourcePoolCPUAllocation(d),
		MemoryAllocation: *expandResourcePoolMemoryAllocation(d),
	}
}

func expandResourcePoolCPUAllocation(d *schema.ResourceData) *types.ResourceAllocationInfo {
	return &types.ResourceAllocationInfo{
		Reservation:           structure.GetInt64Ptr(d, "cpu_reservation"),
		ExpandableReservation: structure.GetBoolPtr(d, "cpu_expandable"),
		Limit: structure.GetInt64Ptr(d, "cpu_limit"),
		Shares: &types.SharesInfo{
			Level:  types.SharesLevel(d.Get("cpu_share_level").(string)),
			Shares: int32(d.Get("cpu_shares").(int)),
		},
	}
}

func expandResourcePoolMemoryAllocation(d *schema.ResourceData) *types.ResourceAllocationInfo {
	return &types.ResourceAllocationInfo{
		Reservation:           structure.GetInt64Ptr(d, "memory_reservation"),
		ExpandableReservation: structure.GetBoolPtr(d, "memory_expandable"),
		Limit: structure.GetInt64Ptr(d, "memory_limit"),
		Shares: &types.SharesInfo{
			Shares: int32(d.Get("memory_shares").(int)),
			Level:  types.SharesLevel(d.Get("memory_share_level").(string)),
		},
	}
}
