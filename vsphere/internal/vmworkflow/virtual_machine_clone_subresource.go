package vmworkflow

import (
	"github.com/hashicorp/terraform/helper/schema"
)

// VirtualMachineCloneSchema represents the schema for the VM clone sub-resource.
//
// This is a workflow for vsphere_virtual_machine that facilitates the creation
// of a virtual machine through cloning from an existing template.
// Customization is nested here, even though it exists in its own workflow.
func VirtualMachineCloneSchema() map[string]*schema.Schema {
	return map[string]*schema.Schema{
		"template": {
			Type:        schema.TypeString,
			Required:    true,
			ForceNew:    true,
			Description: "The source for the clone. This can be a virtual machine or template.",
		},
		"linked_clone": {
			Type:        schema.TypeBool,
			Optional:    true,
			ForceNew:    true,
			Description: "Whether or not to create a linked clone when cloning. When this option is used, the source VM must have a single snapshot associated with it.",
		},
		"customize": {
			Type:        schema.TypeList,
			Optional:    true,
			MaxItems:    1,
			Description: "The customization spec for this clone. This allows the user to configure the virtual machine post-clone.",
		},
	}
}
