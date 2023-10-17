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

var VsanClusterFileServiceSystemInstance = vimtypes.ManagedObjectReference{
	Type:  "VsanFileServiceSystem",
	Value: "vsan-cluster-file-service-system",
}

func Reconfigure(vsanClient *vsan.Client, cluster vimtypes.ManagedObjectReference, spec vsantypes.VimVsanReconfigSpec) error {
	ctx := context.TODO()

	task, err := vsanClient.VsanClusterReconfig(ctx, cluster.Reference(), spec)
	if err != nil {
		return err
	}
	return task.Wait(ctx)
}

func GetVsanConfig(vsanClient *vsan.Client, cluster vimtypes.ManagedObjectReference) (*vsantypes.VsanConfigInfoEx, error) {
	ctx, cancel := context.WithTimeout(context.Background(), provider.DefaultAPITimeout)
	defer cancel()

	vsanConfig, err := vsanClient.VsanClusterGetConfig(ctx, cluster.Reference())

	return vsanConfig, err
}

func FindOvfDownloadUrl(vsanClient *vsan.Client, cluster vimtypes.ManagedObjectReference) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), provider.DefaultAPITimeout)
	defer cancel()

	req := vsantypes.VsanFindOvfDownloadUrl{
		This:    VsanClusterFileServiceSystemInstance,
		Cluster: cluster.Reference(),
	}

	res, err := methods.VsanFindOvfDownloadUrl(ctx, vsanClient, &req)
	if err != nil {
		return "", err
	}

	return res.Returnval, err
}

func DownloadFileServiceOvf(vsanClient *vsan.Client, client *govmomi.Client, fileServiceOvfUrl string) error {
	ctx, cancel := context.WithTimeout(context.Background(), provider.DefaultAPITimeout)
	defer cancel()

	req := vsantypes.VsanDownloadFileServiceOvf{
		This:        VsanClusterFileServiceSystemInstance,
		DownloadUrl: fileServiceOvfUrl,
	}

	res, err := methods.VsanDownloadFileServiceOvf(ctx, vsanClient, &req)
	if err != nil {
		return err
	}

	task := object.NewTask(client.Client, res.Returnval)
	return task.Wait(ctx)
}

func CreateFileServiceDomain(vsanClient *vsan.Client, client *govmomi.Client, domainConfig vsantypes.VsanFileServiceDomainConfig, cluster vimtypes.ManagedObjectReference) error {
	ctx, cancel := context.WithTimeout(context.Background(), provider.DefaultAPITimeout)
	defer cancel()

	req := vsantypes.VsanClusterCreateFsDomain{
		This:         VsanClusterFileServiceSystemInstance,
		DomainConfig: domainConfig,
		Cluster:      &cluster,
	}

	res, err := methods.VsanClusterCreateFsDomain(ctx, vsanClient, &req)
	if err != nil {
		return err
	}

	task := object.NewTask(client.Client, res.Returnval)
	return task.Wait(ctx)
}
