// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package vsphere

import (
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestMain(m *testing.M) {
	resource.TestMain(m)
}

func sweepVSphereClient() (*Client, error) {
	config := Config{
		InsecureFlag:    true,
		Debug:           false,
		Persist:         true,
		User:            os.Getenv("VSPHERE_USER"),
		Password:        os.Getenv("VSPHERE_PASSWORD"),
		VSphereServer:   os.Getenv("VSPHERE_SERVER"),
		DebugPath:       "",
		DebugPathRun:    "",
		VimSessionPath:  "",
		RestSessionPath: "",
		KeepAlive:       0,
	}
	return config.Client()
}
