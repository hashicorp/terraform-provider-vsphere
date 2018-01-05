package clusterrule

import (
	"context"

	"github.com/terraform-providers/terraform-provider-vsphere/vsphere/internal/helper/provider"
	"github.com/vmware/govmomi/object"
	"github.com/vmware/govmomi/vim25/types"
)

func CheckExist(ctx context.Context, c *object.ClusterComputeResource, name string) (bool, error) {
	ret, err := GetRuleByName(c, name)
	return ret != nil, err
}

func GetRuleByName(c *object.ClusterComputeResource, name string) (types.BaseClusterRuleInfo, error) {
	ctx, cancel := context.WithTimeout(context.Background(), provider.DefaultAPITimeout)
	defer cancel()

	cluserConfig, err := c.Configuration(ctx)
	if err != nil {
		return nil, err
	}
	for _, crule := range cluserConfig.Rule {
		info := crule.GetClusterRuleInfo()
		if info.Name == name {
			return info, nil
		}
	}
	return nil, nil
}
