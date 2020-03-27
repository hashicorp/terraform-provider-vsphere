package vsphere

import (
	"github.com/hashicorp/terraform-plugin-sdk/helper/schema"
	"github.com/terraform-providers/terraform-provider-vsphere/vsphere/internal/helper/computepolicy"
)

func dataSourceVSphereComputePolicy() *schema.Resource {
	return &schema.Resource{
		Read: dataSourceVSphereComputePolicyRead,

		Schema: map[string]*schema.Schema{
			"name": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "Name of the compute policy.",
			},
			"description": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Description of the compute policy.",
			},
			"policy_type": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Type of the compute policy.",
			},
			"vm_tag": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The unique identifier of the VM tag.",
			},
			"host_tag": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The unique identifier of the host tag for VM-Host affinity/anti affinity rules",
			},
		},
	}
}

func dataSourceVSphereComputePolicyRead(d *schema.ResourceData, meta interface{}) error {
	c := meta.(*VSphereClient).restClient
	policyName := d.Get("name").(string)

	policies, err := computepolicy.ListPolicy(c)
	if err != nil {
		return err
	}

	for _, policy := range policies {
		if policy.Name == policyName {
			d.SetId(policy.Policy)
			d.Set("name", policy.Name)
			d.Set("description", policy.Description)
			d.Set("policy_type", computepolicy.CapabilityToPolicyType(policy.Capability))
			return nil
		}
	}
	return nil
}
