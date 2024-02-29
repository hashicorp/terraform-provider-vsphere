// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package vsanclient

import (
	"context"

	"github.com/hashicorp/terraform-provider-vsphere/vsphere/internal/helper/provider"
	"github.com/vmware/govmomi"
	"github.com/vmware/govmomi/object"
	vimtypes "github.com/vmware/govmomi/vim25/types"
	"github.com/vmware/govmomi/vsan"
	"github.com/vmware/govmomi/vsan/methods"
	vsantypes "github.com/vmware/govmomi/vsan/types"
)

func Reconfigure(vsanClient *vsan.Client, cluster vimtypes.ManagedObjectReference, spec vsantypes.VimVsanReconfigSpec) error {
	ctx := context.TODO()

	task, err := vsanClient.VsanClusterReconfig(ctx, cluster.Reference(), spec)
	if err != nil {
		return err
	}
	return task.WaitEx(ctx)
}

func GetVsanConfig(vsanClient *vsan.Client, cluster vimtypes.ManagedObjectReference) (*vsantypes.VsanConfigInfoEx, error) {
	ctx, cancel := context.WithTimeout(context.Background(), provider.DefaultAPITimeout)
	defer cancel()

	vsanConfig, err := vsanClient.VsanClusterGetConfig(ctx, cluster.Reference())

	return vsanConfig, err
}

func ConvertToStretchedCluster(vsanClient *vsan.Client, client *govmomi.Client, req vsantypes.VSANVcConvertToStretchedCluster) error {
	ctx, cancel := context.WithTimeout(context.Background(), provider.DefaultAPITimeout)
	defer cancel()

	res, err := methods.VSANVcConvertToStretchedCluster(ctx, vsanClient, &req)

	if err != nil {
		return err
	}

	task := object.NewTask(client.Client, res.Returnval)
	return task.WaitEx(ctx)
}

// removing the witness host automatically disables stretched cluster.
func RemoveWitnessHost(vsanClient *vsan.Client, client *govmomi.Client, req vsantypes.VSANVcRemoveWitnessHost) error {
	ctx, cancel := context.WithTimeout(context.Background(), provider.DefaultAPITimeout)
	defer cancel()

	res, err := methods.VSANVcRemoveWitnessHost(ctx, vsanClient, &req)

	if err != nil {
		return err
	}

	task := object.NewTask(client.Client, res.Returnval)
	return task.WaitEx(ctx)
}

func GetWitnessHosts(vsanClient *vsan.Client, cluster vimtypes.ManagedObjectReference) (*vsantypes.VSANVcGetWitnessHostsResponse, error) {
	ctx, cancel := context.WithTimeout(context.Background(), provider.DefaultAPITimeout)
	defer cancel()

	req := vsantypes.VSANVcGetWitnessHosts{
		This:    vsan.VsanVcStretchedClusterSystem,
		Cluster: cluster.Reference(),
	}

	res, err := methods.VSANVcGetWitnessHosts(ctx, vsanClient, &req)
	if err != nil {
		return nil, err
	}

	return res, err
}
