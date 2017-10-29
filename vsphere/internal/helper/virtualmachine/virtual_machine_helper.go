package virtualmachine

import (
	"context"
	"errors"
	"fmt"
	"net"

	"github.com/terraform-providers/terraform-provider-vsphere/vsphere/internal/helper/provider"
	"github.com/terraform-providers/terraform-provider-vsphere/vsphere/internal/helper/structure"
	"github.com/vmware/govmomi"
	"github.com/vmware/govmomi/find"
	"github.com/vmware/govmomi/object"
	"github.com/vmware/govmomi/property"
	"github.com/vmware/govmomi/vim25/mo"
	"github.com/vmware/govmomi/vim25/types"
)

// FromUUID locates a virtualMachine by its UUID.
func FromUUID(client *govmomi.Client, uuid string) (*object.VirtualMachine, error) {
	search := object.NewSearchIndex(client.Client)

	ctx, cancel := context.WithTimeout(context.Background(), provider.DefaultAPITimeout)
	defer cancel()
	result, err := search.FindByUuid(ctx, nil, uuid, true, structure.BoolPtr(false))
	if err != nil {
		return nil, err
	}

	if result == nil {
		return nil, fmt.Errorf("virtual machine with UUID %q not found", uuid)
	}

	// We need to filter our object through finder to ensure that the
	// InventoryPath field is populated, or else functions that depend on this
	// being present will fail.
	finder := find.NewFinder(client.Client, false)

	rctx, rcancel := context.WithTimeout(context.Background(), provider.DefaultAPITimeout)
	defer rcancel()
	vm, err := finder.ObjectReference(rctx, result.Reference())
	if err != nil {
		return nil, err
	}

	// Should be safe to return here. If our reference returned here and is not a
	// VM, then we have bigger problems and to be honest we should be panicking
	// anyway.
	return vm.(*object.VirtualMachine), nil
}

// FromMOID locates a virtualMachine by its managed
// object reference ID.
func FromMOID(client *govmomi.Client, id string) (*object.VirtualMachine, error) {
	finder := find.NewFinder(client.Client, false)

	ref := types.ManagedObjectReference{
		Type:  "VirtualMachine",
		Value: id,
	}

	ctx, cancel := context.WithTimeout(context.Background(), provider.DefaultAPITimeout)
	defer cancel()
	vm, err := finder.ObjectReference(ctx, ref)
	if err != nil {
		return nil, err
	}
	// Should be safe to return here. If our reference returned here and is not a
	// VM, then we have bigger problems and to be honest we should be panicking
	// anyway.
	return vm.(*object.VirtualMachine), nil
}

// Properties is a convenience method that wraps fetching the
// VirtualMachine MO from its higher-level object.
func Properties(vm *object.VirtualMachine) (*mo.VirtualMachine, error) {
	ctx, cancel := context.WithTimeout(context.Background(), provider.DefaultAPITimeout)
	defer cancel()
	var props mo.VirtualMachine
	if err := vm.Properties(ctx, vm.Reference(), nil, &props); err != nil {
		return nil, err
	}
	return &props, nil
}

// WaitForGuestNet waits for a virtual machine to have routeable network
// access. This is denoted as a gateway, and at least one IP address that can
// reach that gateway. This function supports both IPv4 and IPv6, and returns
// the moment either stack is routeable - it doesn't wait for both.
//
// The timeout is specified in minutes. If zero or a negative value is passed,
// the waiter returns without error immediately.
func WaitForGuestNet(client *govmomi.Client, vm *object.VirtualMachine, timeout int) error {
	if timeout < 1 {
		return nil
	}
	var v4gw, v6gw net.IP

	p := client.PropertyCollector()
	ctx, cancel := context.WithTimeout(context.Background(), provider.DefaultAPITimeout)
	defer cancel()

	err := property.Wait(ctx, p, vm.Reference(), []string{"guest.net", "guest.ipStack"}, func(pc []types.PropertyChange) bool {
		for _, c := range pc {
			if c.Op != types.PropertyChangeOpAssign {
				continue
			}

			switch v := c.Val.(type) {
			case types.ArrayOfGuestStackInfo:
				for _, s := range v.GuestStackInfo {
					if s.IpRouteConfig != nil {
						for _, r := range s.IpRouteConfig.IpRoute {
							switch r.Network {
							case "0.0.0.0":
								v4gw = net.ParseIP(r.Gateway.IpAddress)
							case "::":
								v6gw = net.ParseIP(r.Gateway.IpAddress)
							}
						}
					}
				}
			case types.ArrayOfGuestNicInfo:
				for _, n := range v.GuestNicInfo {
					if n.IpConfig != nil {
						for _, addr := range n.IpConfig.IpAddress {
							ip := net.ParseIP(addr.IpAddress)
							var mask net.IPMask
							if ip.To4() != nil {
								mask = net.CIDRMask(int(addr.PrefixLength), 32)
							} else {
								mask = net.CIDRMask(int(addr.PrefixLength), 128)
							}
							if ip.Mask(mask).Equal(v4gw.Mask(mask)) || ip.Mask(mask).Equal(v6gw.Mask(mask)) {
								return true
							}
						}
					}
				}
			}
		}

		return false
	})

	if err != nil {
		// Provide a friendly error message if we timed out waiting for a routeable IP.
		if ctx.Err() == context.DeadlineExceeded {
			return errors.New("timeout waiting for a routeable interface")
		}
		return err
	}

	return nil
}

// Create wraps the creation of a virtual machine and the subsequent waiting of
// the task. A higher-level virtual machine object is returned.
func Create(c *govmomi.Client, f *object.Folder, s types.VirtualMachineConfigSpec, p *object.ResourcePool, h *object.HostSystem) (*object.VirtualMachine, error) {
	ctx, cancel := context.WithTimeout(context.Background(), provider.DefaultAPITimeout)
	defer cancel()
	task, err := f.CreateVM(ctx, s, p, h)
	if err != nil {
		return nil, err
	}
	tctx, tcancel := context.WithTimeout(context.Background(), provider.DefaultAPITimeout)
	defer tcancel()
	result, err := task.WaitForResult(tctx, nil)
	if err != nil {
		return nil, err
	}
	return FromMOID(c, result.Result.(types.ManagedObjectReference).Value)
}

// PowerOn wraps powering on a VM and the waiting for the subsequent task.
func PowerOn(vm *object.VirtualMachine) error {
	ctx, cancel := context.WithTimeout(context.Background(), provider.DefaultAPITimeout)
	defer cancel()
	task, err := vm.PowerOn(ctx)
	if err != nil {
		return err
	}
	tctx, tcancel := context.WithTimeout(context.Background(), provider.DefaultAPITimeout)
	defer tcancel()
	return task.Wait(tctx)
}

// PowerOff wraps powering off a VM and the waiting for the subsequent task.
func PowerOff(vm *object.VirtualMachine) error {
	ctx, cancel := context.WithTimeout(context.Background(), provider.DefaultAPITimeout)
	defer cancel()
	task, err := vm.PowerOff(ctx)
	if err != nil {
		return err
	}
	tctx, tcancel := context.WithTimeout(context.Background(), provider.DefaultAPITimeout)
	defer tcancel()
	return task.Wait(tctx)
}
