package cluster

import (
	"context"

	"github.com/terraform-providers/terraform-provider-vsphere/vsphere/internal/helper/computeresource"
	"github.com/terraform-providers/terraform-provider-vsphere/vsphere/internal/helper/provider"
	"github.com/vmware/govmomi"
	"github.com/vmware/govmomi/vim25/types"
)

func GetClusterConfigInfo(client *govmomi.Client, clusterComputeResourceID string) (*types.ClusterConfigInfoEx, error) {
	ccr, err := computeresource.ClusterFromID(client, clusterComputeResourceID)
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithTimeout(context.Background(), provider.DefaultAPITimeout)
	defer cancel()
	cluserConfig, err := ccr.Configuration(ctx)
	if err != nil {
		return nil, err
	}
	return cluserConfig, err
}
