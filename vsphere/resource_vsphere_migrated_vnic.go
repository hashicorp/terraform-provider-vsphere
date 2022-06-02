package vsphere

import (
	"context"
	"fmt"
	"log"
	"strconv"
	"strings"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/vmware/govmomi/vim25/types"
)

func resourceVsphereMigratedNic() *schema.Resource {
	return &schema.Resource{
		Create: resourceVsphereMigratedNicUpdate,
		Read:   resourceVsphereNicRead,
		Update: resourceVsphereMigratedNicUpdate,
		Delete: resourceVsphereNicDelete,
		Importer: &schema.ResourceImporter{
			State: resourceVSphereNicImport,
		},
		Schema: vMigratedNicSchema(),
	}
}

func vMigratedNicSchema() map[string]*schema.Schema {
	base := BaseVMKernelSchema()
	base["vnic_id"] = &schema.Schema{
		Type:        schema.TypeString,
		Required:    true,
		Description: "Resource ID of vnic to migrate",
		ForceNew:    true,
	}

	return base
}

func resourceVsphereMigratedNicUpdate(d *schema.ResourceData, meta interface{}) error {
	for _, k := range []string{
		"portgroup", "distributed_switch_port", "distributed_port_group",
		"mac", "mtu", "ipv4", "ipv6", "netstack"} {
		if d.HasChange(k) {
			_, err := updateVMigratedNic(d, meta)
			if err != nil {
				return err
			}
			break
		}
	}
	return resourceVsphereMigratedNicRead(d, meta)
}

func updateVMigratedNic(d *schema.ResourceData, meta interface{}) (string, error) {
	client := meta.(*Client).vimClient
	hostID, nicID := splitHostIDMigratedNicID(d)
	ctx := context.TODO()

	nic, err := getNicSpecFromSchema(d)
	if err != nil {
		return "", err
	}

	hns, err := getHostNetworkSystem(client, hostID)
	if err != nil {
		return "", err
	}

	err = hns.UpdateVirtualNic(ctx, nicID, *nic)
	if err != nil {
		return "", err
	}

	return nicID, nil
}

func splitHostIDMigratedNicID(d *schema.ResourceData) (string, string) {
	id := d.Get("vnic_id").(string)
	idParts := strings.Split(id, "_")
	return idParts[0], idParts[1]
}
