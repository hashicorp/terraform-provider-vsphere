package virtualdevice

import (
	"github.com/hashicorp/terraform-plugin-sdk/helper/schema"
)

func VirtualMachineTagRulesSchema() map[string]*schema.Schema {
	return map[string]*schema.Schema{
		"tag_category": {
			Type:        schema.TypeString,
			Required:    true,
			Description: "category",
		},
		"tags": {
			Type:        schema.TypeList,
			Required:    true,
			Description: "tags",
			Elem:        &schema.Schema{Type: schema.TypeString},
		},
		"include_datastores_with_tags": {
			Type:        schema.TypeBool,
			Optional:    true,
			Description: "Whether to include or exclude datastores tagged with the provided tags",
			Default:     true,
		},
	}
}
