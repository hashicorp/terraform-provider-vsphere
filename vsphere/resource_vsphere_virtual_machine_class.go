// © Broadcom. All Rights Reserved.
// The term "Broadcom" refers to Broadcom Inc. and/or its subsidiaries.
// SPDX-License-Identifier: MPL-2.0

package vsphere

import (
	"context"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/vmware/govmomi/vapi/namespace"
)

func resourceVsphereVMClass() *schema.Resource {
	return &schema.Resource{
		Create: resourceVsphereVMClassCreate,
		Read:   resourceVsphereVMClassRead,
		Update: resourceVsphereVMClassUpdate,
		Delete: resourceVsphereVMClassDelete,
		Schema: map[string]*schema.Schema{
			"name": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "The name of the virtual machine class.",
			},
			"cpus": {
				Type:        schema.TypeInt,
				Required:    true,
				Description: "The number of CPUs.",
			},
			"memory": {
				Type:        schema.TypeInt,
				Required:    true,
				Description: "The amount of memory (in MB).",
			},
			"cpu_reservation": {
				Type:        schema.TypeInt,
				Optional:    true,
				Description: "The percentage of the available CPU capacity which will be reserved.",
			},
			"memory_reservation": {
				Type:        schema.TypeInt,
				Optional:    true,
				Description: "The percentage of the available memory capacity which will be reserved.",
			},
			"vgpu_devices": {
				Type:        schema.TypeList,
				Optional:    true,
				Description: "A comma-separated list of GPU devices.",
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
			},
		},
	}
}

func resourceVsphereVMClassCreate(d *schema.ResourceData, meta interface{}) error {
	c := meta.(*Client).restClient
	m := namespace.NewManager(c)

	vmClassSpec := namespace.VirtualMachineClassCreateSpec{
		Id:                d.Get("name").(string),
		CpuCount:          int64(d.Get("cpus").(int)),
		MemoryMb:          int64(d.Get("memory").(int)),
		CpuReservation:    int64(d.Get("cpu_reservation").(int)),
		MemoryReservation: int64(d.Get("memory_reservation").(int)),
	}

	vgpuDevices := d.Get("vgpu_devices").([]interface{})
	vmClassSpec.Devices = namespace.VirtualDevices{
		VgpuDevices: make([]namespace.VgpuDevice, len(vgpuDevices)),
	}

	for i, g := range vgpuDevices {
		vmClassSpec.Devices.VgpuDevices[i] = namespace.VgpuDevice{ProfileName: g.(string)}
	}

	if err := m.CreateVmClass(context.Background(), vmClassSpec); err != nil {
		return err
	}

	d.SetId(vmClassSpec.Id)

	return nil
}

func resourceVsphereVMClassRead(d *schema.ResourceData, meta interface{}) error {
	c := meta.(*Client).restClient
	m := namespace.NewManager(c)

	_, err := m.GetVmClass(context.Background(), d.Id())

	return err
}

func resourceVsphereVMClassUpdate(d *schema.ResourceData, meta interface{}) error {
	c := meta.(*Client).restClient
	m := namespace.NewManager(c)

	vmClassSpec := namespace.VirtualMachineClassUpdateSpec{
		Id:                d.Get("name").(string),
		CpuCount:          int64(d.Get("cpus").(int)),
		MemoryMb:          int64(d.Get("memory").(int)),
		CpuReservation:    int64(d.Get("cpu_reservation").(int)),
		MemoryReservation: int64(d.Get("memory_reservation").(int)),
	}

	vgpuDevices := d.Get("vgpu_devices").([]interface{})
	vmClassSpec.Devices = namespace.VirtualDevices{
		VgpuDevices: make([]namespace.VgpuDevice, len(vgpuDevices)),
	}

	for i, g := range vgpuDevices {
		vmClassSpec.Devices.VgpuDevices[i] = namespace.VgpuDevice{ProfileName: g.(string)}
	}

	return m.UpdateVmClass(context.Background(), d.Id(), vmClassSpec)
}

func resourceVsphereVMClassDelete(d *schema.ResourceData, meta interface{}) error {
	c := meta.(*Client).restClient
	m := namespace.NewManager(c)

	return m.DeleteVmClass(context.Background(), d.Id())
}
