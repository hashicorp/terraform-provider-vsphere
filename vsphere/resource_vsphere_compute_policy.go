package vsphere

import (
	"log"
	"strings"

	"github.com/hashicorp/terraform-plugin-sdk/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/helper/validation"
	"github.com/terraform-providers/terraform-provider-vsphere/vsphere/internal/helper/computepolicy"
	"github.com/terraform-providers/terraform-provider-vsphere/vsphere/internal/helper/structure"
)

const resourceVSphereComputePolicyName = "vsphere_compute_policy"

const (
	computePolicyTypeVmHostAffinity     = "vm_host_affinity"
	computePolicyTypeVmHostAntiAffinity = "vm_host_anti_affinity"
	computePolicyTypeVmVmAffinity       = "vm_vm_affinity"
	computePolicyTypeVmVmAntiAffinity   = "vm_vm_anti_affinity"
)

var computePolicyTypeAllowedValues = []string{
	computePolicyTypeVmHostAffinity,
	computePolicyTypeVmHostAntiAffinity,
	computePolicyTypeVmVmAffinity,
	computePolicyTypeVmVmAntiAffinity,
}

func resourceVSphereComputePolicy() *schema.Resource {
	return &schema.Resource{
		Create: resourceVSphereComputePolicyCreate,
		Read:   resourceVSphereComputePolicyRead,
		Delete: resourceVSphereComputePolicyDelete,
		Importer: &schema.ResourceImporter{
			State: resourceVSphereComputePolicyImport,
		},

		Schema: map[string]*schema.Schema{
			"name": {
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
				Description: "Name of the compute policy.",
			},
			"description": {
				Type:        schema.TypeString,
				Optional:    true,
				ForceNew:    true,
				Description: "Description of the compute policy.",
			},
			"policy_type": {
				Type:         schema.TypeString,
				Required:     true,
				ForceNew:     true,
				Description:  "Type of the compute policy.",
				ValidateFunc: validation.StringInSlice(computePolicyTypeAllowedValues, false),
			},
			"vm_tag": {
				Type:        schema.TypeString,
				Description: "The unique identifier of the VM tag.",
				Required:    true,
				ForceNew:    true,
			},
			"host_tag": {
				Type:        schema.TypeString,
				Description: "The unique identifier of the host tag for VM-Host affinity/anti affinity rules",
				Optional:    true,
				ForceNew:    true,
			},
		},
	}
}

func resourceVSphereComputePolicyCreate(d *schema.ResourceData, meta interface{}) error {
	log.Printf("[DEBUG] resourceVSphereComputePolicyCreate : Beginning Compute Policy (%s) create", resourceVSphereComputePolicyIDString(d))
	c := meta.(*VSphereClient).restClient
	name := d.Get("name").(string)
	description := d.Get("description").(string)
	capability := computepolicy.PolicyTypeToCapability(d.Get("policy_type").(string))
	vm_tag := d.Get("vm_tag").(string)
	host_tag := d.Get("host_tag").(string)
	result, err := computepolicy.CreatePolicy(c, name, description, capability, vm_tag, host_tag)
	if err != nil {
		return err
	}

	// All done!
	d.SetId(result)
	log.Printf("[DEBUG] resourceVSphereComputePolicyCreate : Compute policy (%s) creation is complete", resourceVSphereComputePolicyIDString(d))
	return resourceVSphereComputePolicyRead(d, meta)
}

func resourceVSphereComputePolicyRead(d *schema.ResourceData, meta interface{}) error {
	log.Printf("[DEBUG] resourceVSphereComputePolicyRead : Beginning Compute Policy (%s) read", resourceVSphereComputePolicyIDString(d))
	c := meta.(*VSphereClient).restClient
	policyStruct, err := computepolicy.ReadPolicy(c, d.Id())
	if err != nil {
		if strings.Contains(err.Error(), "404 Not Found") {
			d.SetId("")
			return nil
		} else {
			return err
		}
	}

	if err = d.Set("name", policyStruct.Name); err != nil {
		return err
	}
	if err = d.Set("description", policyStruct.Description); err != nil {
		return err
	}
	if err = d.Set("policy_type", computepolicy.CapabilityToPolicyType(policyStruct.Capability)); err != nil {
		return err
	}

	log.Printf("[DEBUG] resourceVSphereComputePolicyRead : Compute Policy (%s) read is complete", d.Id())
	return nil
}

func resourceVSphereComputePolicyDelete(d *schema.ResourceData, meta interface{}) error {
	log.Printf("[DEBUG] resourceVSphereComputePolicyDelete : Deleting Compute Policy (%s)", d.Id())
	c := meta.(*VSphereClient).restClient
	if err := computepolicy.DeletePolicy(c, d.Id()); err != nil {
		return err
	}
	log.Printf("[DEBUG] resourceVSphereComputePolicyDelete : Compute Policy (%s) deleted", d.Id())
	return nil
}

func resourceVSphereComputePolicyImport(d *schema.ResourceData, meta interface{}) ([]*schema.ResourceData, error) {
	// unsupported
	return []*schema.ResourceData{d}, nil
}

// resourceVSphereComputePolicyIDString prints a friendly string for the
// vsphere_compute_policy resource.
func resourceVSphereComputePolicyIDString(d structure.ResourceIDStringer) string {
	return structure.ResourceIDString(d, resourceVSphereComputePolicyName)
}
