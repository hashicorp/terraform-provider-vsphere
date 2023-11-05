// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package vsphere

import (
	"fmt"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-provider-vsphere/vsphere/internal/helper/sso"
)

func dataSourceVsphereSsoUser() *schema.Resource {
	return &schema.Resource{
		Read: dataSourceVSphereSsoUserRead,

		Schema: map[string]*schema.Schema{
			"name": {
				Type:        schema.TypeString,
				Description: "The name of the SSO user.",
				Optional:    false,
				Required:    true,
			},
		},
	}
}

func dataSourceVSphereSsoUserRead(d *schema.ResourceData, meta interface{}) error {

	clientConfig := meta.(*Client).ssoAdminClientConfig
	userName := d.Get("name").(string)

	user, err := sso.AdminPersonUserFromName(clientConfig, userName)

	if err != nil {
		return fmt.Errorf("error fetching sso user: %s", err)
	}

	id := user.Id
	d.SetId(fmt.Sprintf("%s.%s", id.Name, id.Domain))

	return nil
}
