// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package vsphere

import (
	"context"
	"errors"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/vmware/govmomi/vapi/cis/tasks"
	"github.com/vmware/govmomi/vapi/esx/settings/depots"
)

func resourceVsphereOfflineSoftwareDepot() *schema.Resource {
	s := map[string]*schema.Schema{
		"location": {
			Type:        schema.TypeString,
			Description: "The remote location where the contents for this depot are served.",
			Required:    true,
			ForceNew:    true,
		},
		"component": {
			Type:        schema.TypeList,
			Description: "The list of components in this depot.",
			Computed:    true,
			Elem: &schema.Resource{
				Schema: map[string]*schema.Schema{
					"key": {
						Type:        schema.TypeString,
						Description: "The key of the component.",
						Computed:    true,
					},
					"display_name": {
						Type:        schema.TypeString,
						Description: "The name of the component.",
						Computed:    true,
					},
					"version": {
						Type:        schema.TypeList,
						Description: "The list of versions of the component.",
						Computed:    true,
						Elem:        &schema.Schema{Type: schema.TypeString},
					},
				},
			},
		},
	}

	return &schema.Resource{
		Create: resourceVsphereOfflineSoftwareDepotCreate,
		Read:   resourceVsphereOfflineSoftwareDepotRead,
		Delete: resourceVsphereOfflineSoftwareDepotDelete,
		Schema: s,
	}
}

func resourceVsphereOfflineSoftwareDepotCreate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*Client).restClient

	location := d.Get("location").(string)

	spec := depots.SettingsDepotsOfflineCreateSpec{
		SourceType: string(depots.SourceTypePull),
		Location:   location,
	}

	m := depots.NewManager(client)

	if taskId, err := m.CreateOfflineDepot(spec); err != nil {
		return err
	} else if _, err = tasks.NewManager(client).WaitForCompletion(context.Background(), taskId); err != nil {
		return err
	} else if offlineDepots, err := m.GetOfflineDepots(); err != nil {
		return err
	} else {
		for id, depot := range offlineDepots {
			if depot.Location == location {
				d.SetId(id)
				return resourceVsphereOfflineSoftwareDepotRead(d, meta)
			}
		}

		return errors.New("failed to create offline software depot")
	}
}

func resourceVsphereOfflineSoftwareDepotRead(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*Client).restClient
	m := depots.NewManager(client)

	if data, err := m.GetOfflineDepotContent(d.Id()); err != nil {
		return err
	} else {
		d.SetId(d.Id())
		return d.Set("component", readComponents(data))
	}
}

func resourceVsphereOfflineSoftwareDepotDelete(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*Client).restClient
	m := depots.NewManager(client)

	if taskId, err := m.DeleteOfflineDepot(d.Id()); err != nil {
		return err
	} else if _, err = tasks.NewManager(client).WaitForCompletion(context.Background(), taskId); err != nil {
		return err
	} else {
		return nil
	}
}

func readComponents(data depots.SettingsDepotsOfflineContentInfo) []map[string]interface{} {
	components := make([]map[string]interface{}, 0)
	for _, srcBundles := range data.MetadataBundles {
		for _, srcBundle := range srcBundles {
			components = append(components, readIndependentComponents(srcBundle.IndependentComponents)...)
		}
	}

	return components
}

func readIndependentComponents(data map[string]depots.SettingsDepotsComponentSummary) []map[string]interface{} {
	independentComponents := make([]map[string]interface{}, len(data))
	count := 0
	for key, srcComponent := range data {
		component := make(map[string]interface{})
		component["key"] = key
		component["display_name"] = srcComponent.DisplayName
		versions := make([]string, len(srcComponent.Versions))
		component["version"] = versions
		for i, version := range srcComponent.Versions {
			versions[i] = version.Version
		}
		independentComponents[count] = component
		count++
	}

	return independentComponents
}
