package main

import (
	"github.com/hashicorp/terraform/plugin"
	"github.com/terraform-providers/terraform-provider-vsphere/vsphere"
)

func main() {
	plugin.Serve(&plugin.ServeOpts{
		ProviderFunc: vsphere.Provider})
}
