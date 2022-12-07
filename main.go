package main

import (
	"log"

	"github.com/hashicorp/terraform-plugin-sdk/v2/plugin"
	"github.com/hashicorp/terraform-provider-vsphere/vsphere"
)

func main() {
	log.Printf("belle, we using right plugin")
	plugin.Serve(&plugin.ServeOpts{
		ProviderFunc: vsphere.Provider})
}
