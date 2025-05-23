// © Broadcom. All Rights Reserved.
// The term "Broadcom" refers to Broadcom Inc. and/or its subsidiaries.
// SPDX-License-Identifier: MPL-2.0

package virtualdevice

import (
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func VirtualMachineTagRulesSchema() map[string]*schema.Schema {
	return map[string]*schema.Schema{
		"tag_category": {
			Type:        schema.TypeString,
			Required:    true,
			Description: "The tag category to select the tags from.",
		},
		"tags": {
			Type:        schema.TypeList,
			Required:    true,
			MinItems:    1,
			Description: "The tags to use for creating a tag-based vm placement rule.",
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
