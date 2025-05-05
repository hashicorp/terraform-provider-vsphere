// © Broadcom. All Rights Reserved.
// The term "Broadcom" refers to Broadcom Inc. and/or its subsidiaries.
// SPDX-License-Identifier: MPL-2.0

package vsphere

import (
	"github.com/vmware/govmomi/vapi/esx/settings/depots"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func dataSourceVSphereHostBaseImages() *schema.Resource {
	return &schema.Resource{
		Read: dataSourceVSphereHostBaseImagesRead,
		Schema: map[string]*schema.Schema{
			"version": {
				Type:        schema.TypeList,
				Computed:    true,
				Description: "The available ESXi versions.",
				Elem:        &schema.Schema{Type: schema.TypeString},
			},
		},
	}
}

func dataSourceVSphereHostBaseImagesRead(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*Client).restClient

	images, err := depots.NewManager(client).ListBaseImages()
	if err != nil {
		return err
	}

	versions := make([]string, len(images))
	for i, image := range images {
		versions[i] = image.Version
	}

	d.SetId(versions[0])
	return d.Set("version", versions)
}
