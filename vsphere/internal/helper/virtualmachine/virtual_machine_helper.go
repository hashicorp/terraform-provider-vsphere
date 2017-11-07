package virtualmachine

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net"
	"time"

	"github.com/terraform-providers/terraform-provider-vsphere/vsphere/internal/helper/folder"
	"github.com/terraform-providers/terraform-provider-vsphere/vsphere/internal/helper/provider"
	"github.com/terraform-providers/terraform-provider-vsphere/vsphere/internal/helper/structure"
	"github.com/vmware/govmomi"
	"github.com/vmware/govmomi/find"
	"github.com/vmware/govmomi/object"
	"github.com/vmware/govmomi/property"
	"github.com/vmware/govmomi/vim25/mo"
	"github.com/vmware/govmomi/vim25/types"
)

var errGuestShutdownTimeout = errors.New("the VM did not power off within the specified amount of time")

// FromUUID locates a virtualMachine by its UUID.
func FromUUID(client *govmomi.Client, uuid string) (*object.VirtualMachine, error) {
	log.Printf("[DEBUG] Locating virtual machine with UUID %q", uuid)
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
	log.Printf("[DEBUG] VM %q found for UUID %q", vm.(*object.VirtualMachine).InventoryPath, uuid)
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
	log.Printf("[DEBUG] Fetching properties for VM %q", vm.InventoryPath)
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
		log.Printf("[DEBUG] Skipping network waiter for VM %q", vm.InventoryPath)
		return nil
	}
	log.Printf("[DEBUG] Waiting for routeable address on VM %q (timeout = %dm)", vm.InventoryPath, timeout)
	var v4gw, v6gw net.IP

	p := client.PropertyCollector()
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute*time.Duration(timeout))
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

	log.Printf("[DEBUG] Routeable address available for VM %q", vm.InventoryPath)
	return nil
}

// Create wraps the creation of a virtual machine and the subsequent waiting of
// the task. A higher-level virtual machine object is returned.
func Create(c *govmomi.Client, f *object.Folder, s types.VirtualMachineConfigSpec, p *object.ResourcePool, h *object.HostSystem) (*object.VirtualMachine, error) {
	log.Printf("[DEBUG] Creating virtual machine %q", fmt.Sprintf("%s/%s", f.InventoryPath, s.Name))
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
	log.Printf("[DEBUG] Virtual machine %q: creation complete (MOID: %q)", fmt.Sprintf("%s/%s", f.InventoryPath, s.Name), result.Result.(types.ManagedObjectReference).Value)
	return FromMOID(c, result.Result.(types.ManagedObjectReference).Value)
}

// PowerOn wraps powering on a VM and the waiting for the subsequent task.
func PowerOn(vm *object.VirtualMachine) error {
	log.Printf("[DEBUG] Powering on virtual machine %q", vm.InventoryPath)
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
	log.Printf("[DEBUG] Forcing power off of virtual machine of %q", vm.InventoryPath)
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

// ShutdownGuest wraps the graceful shutdown of a guest VM, and then waiting an
// appropriate amount of time for the guest power state to go to powered off.
// If the VM does not power off in the shutdown period specified by timeout (in
// minutes), an error is returned.
//
// The minimum value for timeout is 1 minute - setting to a 0 or negative value
// is not allowed and will just reset the timeout to the minimum.
func ShutdownGuest(client *govmomi.Client, vm *object.VirtualMachine, timeout int) error {
	log.Printf("[DEBUG] Attempting guest shutdown of virtual machine %q", vm.InventoryPath)
	sctx, scancel := context.WithTimeout(context.Background(), provider.DefaultAPITimeout)
	defer scancel()
	if err := vm.ShutdownGuest(sctx); err != nil {
		return err
	}

	// We now wait on VM power state to be powerOff, via a property collector that waits on power state.
	p := client.PropertyCollector()
	if timeout < 1 {
		timeout = 1
	}
	pctx, pcancel := context.WithTimeout(context.Background(), time.Minute*time.Duration(timeout))
	defer pcancel()

	err := property.Wait(pctx, p, vm.Reference(), []string{"runtime.powerState"}, func(pc []types.PropertyChange) bool {
		for _, c := range pc {
			if c.Op != types.PropertyChangeOpAssign {
				continue
			}

			switch v := c.Val.(type) {
			case types.VirtualMachinePowerState:
				if v == types.VirtualMachinePowerStatePoweredOff {
					return true
				}
			}
		}

		return false
	})

	if err != nil {
		// Provide a friendly error message if we timed out waiting for a shutdown.
		if pctx.Err() == context.DeadlineExceeded {
			return errGuestShutdownTimeout
		}
		return err
	}
	return nil
}

// GracefulPowerOff is a meta-operation that handles powering down of virtual
// machines. A graceful shutdown is attempted first if possible (VMware tools
// is installed, and the guest state is not suspended), and then, if allowed, a
// power-off is forced if that fails.
func GracefulPowerOff(client *govmomi.Client, vm *object.VirtualMachine, timeout int, force bool) error {
	vprops, err := Properties(vm)
	if err != nil {
		return err
	}
	// First we attempt a guest shutdown if we have VMware tools and if the VM is
	// actually powered on (we don't expect that a graceful shutdown would
	// complete on a suspended VM, so there's really no point in trying).
	if vprops.Runtime.PowerState == types.VirtualMachinePowerStatePoweredOn && vprops.Guest != nil && vprops.Guest.ToolsRunningStatus == string(types.VirtualMachineToolsRunningStatusGuestToolsRunning) {
		if err := ShutdownGuest(client, vm, timeout); err != nil {
			if err == errGuestShutdownTimeout && !force {
				return err
			}
		} else {
			return nil
		}
	}
	// If the guest shutdown failed (and we were allowed to proceed), or
	// conditions did not satisfy the criteria for a graceful shutdown, do a full
	// power-off of the VM.
	return PowerOff(vm)
}

// MoveToFolder moves a virtual machine to the specified folder.
func MoveToFolder(client *govmomi.Client, vm *object.VirtualMachine, relative string) error {
	log.Printf("[DEBUG] Moving virtual %q to VM path %q", vm.InventoryPath, relative)
	f, err := folder.VirtualMachineFolderFromObject(client, vm, relative)
	if err != nil {
		return err
	}
	return folder.MoveObjectTo(vm.Reference(), f)
}

// Reconfigure wraps the Reconfigure task and the subsequent waiting for
// the task to complete.
func Reconfigure(vm *object.VirtualMachine, spec types.VirtualMachineConfigSpec) error {
	log.Printf("[DEBUG] Reconfiguring virtual machine %q", vm.InventoryPath)
	ctx, cancel := context.WithTimeout(context.Background(), provider.DefaultAPITimeout)
	defer cancel()
	task, err := vm.Reconfigure(ctx, spec)
	if err != nil {
		return err
	}
	tctx, tcancel := context.WithTimeout(context.Background(), provider.DefaultAPITimeout)
	defer tcancel()
	return task.Wait(tctx)
}

// Destroy wraps the Destroy task and the subsequent waiting for the task to
// complete.
func Destroy(vm *object.VirtualMachine) error {
	log.Printf("[DEBUG] Deleting virtual machine %q", vm.InventoryPath)
	ctx, cancel := context.WithTimeout(context.Background(), provider.DefaultAPITimeout)
	defer cancel()
	task, err := vm.Destroy(ctx)
	if err != nil {
		return err
	}
	tctx, tcancel := context.WithTimeout(context.Background(), provider.DefaultAPITimeout)
	defer tcancel()
	return task.Wait(tctx)
}
