package vsphere

import (
	"context"
	"fmt"

	"github.com/vmware/govmomi"
	"github.com/vmware/govmomi/object"
	"github.com/vmware/govmomi/vim25/mo"
)

// virtualMachineFromUUID locates a virtualMachine by its UUID.
func virtualMachineFromUUID(client *govmomi.Client, uuid string) (*object.VirtualMachine, error) {
	search := object.NewSearchIndex(client.Client)

	ctx, cancel := context.WithTimeout(context.Background(), defaultAPITimeout)
	defer cancel()
	vm, err := search.FindByUuid(ctx, nil, uuid, true, boolPtr(false))
	if err != nil {
		return nil, err
	}

	if vm == nil {
		return nil, fmt.Errorf("virtual machine with UUID %q not found", uuid)
	}

	// Should be safe to return here. If our reference returned here and is not a
	// VM, then we have bigger problems and to be honest we should be panicking
	// anyway.
	return vm.(*object.VirtualMachine), nil
}

// virtualMachineProperties is a convenience method that wraps fetching the
// VirtualMachine MO from its higher-level object.
func virtualMachineProperties(vm *object.VirtualMachine) (*mo.VirtualMachine, error) {
	ctx, cancel := context.WithTimeout(context.Background(), defaultAPITimeout)
	defer cancel()
	var props mo.VirtualMachine
	if err := vm.Properties(ctx, vm.Reference(), nil, &props); err != nil {
		return nil, err
	}
	return &props, nil
}
