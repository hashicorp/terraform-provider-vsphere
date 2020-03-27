package computepolicy

import (
	"context"
	"log"
	"strings"

	"github.com/terraform-providers/terraform-provider-vsphere/vsphere/internal/helper/provider"
	"github.com/vmware/govmomi/vapi/compute"
	"github.com/vmware/govmomi/vapi/rest"
)

// CreatePolicy creates a Compute Policy.
func CreatePolicy(c *rest.Client, name string, description string, capability string, vmtag string, hosttag string) (string, error) {
	log.Printf("[DEBUG] computepolicy.CreatePolicy: Creating compute policy %s", name)
	cpm := compute.NewPolicyManager(c)
	ctx := context.TODO()
	policy := compute.Policy{
		Name:        name,
		Capability:  capability,
		Description: description,
		VMTag:       vmtag,
		HostTag:     hosttag,
	}
	id, err := cpm.Create(ctx, policy)
	if err != nil {
		return "", provider.ProviderError(name, "CreatePolicy", err)
	}
	log.Printf("[DEBUG] computepolicy.CreatePolicy: Compute policy %s successfully created", name)
	return id, nil
}

// ReadPolicy reads a Compute Policy.
func ReadPolicy(c *rest.Client, id string) (*compute.Policy, error) {
	log.Printf("[DEBUG] computepolicy.ReadPolicy: Reading compute policy %s", id)
	cpm := compute.NewPolicyManager(c)
	ctx := context.TODO()
	policy, err := cpm.Get(ctx, id)
	if err != nil {
		return nil, provider.ProviderError(id, "ReadPolicy", err)
	}
	log.Printf("[DEBUG] computepolicy.ReadPolicy: Compute policy %s read complete", id)
	return policy, nil
}

// ListPolicy lists all Compute Policies.
func ListPolicy(c *rest.Client) ([]compute.Policy, error) {
	log.Printf("[DEBUG] computepolicy.ListPolicy: Listing all compute policies")
	cpm := compute.NewPolicyManager(c)
	ctx := context.TODO()
	policyList, err := cpm.List(ctx)
	if err != nil {
		return nil, provider.ProviderError("", "ListPolicy", err)
	}
	log.Printf("[DEBUG] computepolicy.ListPolicy: Listing all compute policies complete")
	return policyList, nil
}

// DeletePolicy deletes a Compute Policy.
func DeletePolicy(c *rest.Client, id string) error {
	log.Printf("[DEBUG] computepolicy.DeletePolicy: Deleting compute policy %s", id)
	cpm := compute.NewPolicyManager(c)
	ctx := context.TODO()
	err := cpm.Delete(ctx, id)
	if err != nil {
		return provider.ProviderError(id, "DeletePolicy", err)
	}
	log.Printf("[DEBUG] computepolicy.DeletePolicy: Compute policy %s deleted", id)
	return nil
}

// PolicyTypeToCapability converts policy type to full capability prop name used in API
func PolicyTypeToCapability(policyType string) string {
	return "com.vmware.vcenter.compute.policies.capabilities." + policyType
}

// CapabilityToPolicyType converts capability to user friendly policy type value
func CapabilityToPolicyType(capability string) string {
	tokens := strings.Split(capability, ".")
	return tokens[len(tokens)-1]
}
