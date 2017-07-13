package main

import (
	"bitbucket.org/crestdatasys/terraform_vsphere/vsphere"
	"github.com/hashicorp/terraform/plugin"
)

func main() {
	plugin.Serve(&plugin.ServeOpts{
		ProviderFunc: vsphere.Provider})
}
