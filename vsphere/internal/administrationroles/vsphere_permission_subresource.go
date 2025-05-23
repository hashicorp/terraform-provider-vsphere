// © Broadcom. All Rights Reserved.
// The term "Broadcom" refers to Broadcom Inc. and/or its subsidiaries.
// SPDX-License-Identifier: MPL-2.0

package administrationroles

import (
	"strings"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func VspherePermissionSchema() map[string]*schema.Schema {
	return map[string]*schema.Schema{
		"user_or_group": {
			Type:        schema.TypeString,
			Required:    true,
			Description: "User or group receiving access.",
			DiffSuppressFunc: func(_, old, newValue string, _ *schema.ResourceData) bool {
				return strings.EqualFold(old, newValue)
			},
		},
		"propagate": {
			Type:        schema.TypeBool,
			Required:    true,
			Description: "Whether or not this permission propagates down the hierarchy to sub-entities.",
		},
		"is_group": {
			Type:        schema.TypeBool,
			Required:    true,
			Description: "Whether user_or_group field refers to a user or a group. True for a group and false for a user.",
		},
		"role_id": {
			Type:        schema.TypeString,
			Required:    true,
			Description: "Reference to the role providing the access.",
		},
	}
}
