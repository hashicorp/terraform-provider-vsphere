package vsphere

import (
	"context"
	"log"

	"github.com/vmware/govmomi"
	"github.com/vmware/govmomi/object"
)

func virtualMachineFromUUID(client *govmomi.Client, vmUuid string) (*object.VirtualMachine, error) {
	searchIndex := object.NewSearchIndex(client.Client)
	ctx, cancel := context.WithTimeout(context.Background(), defaultAPITimeout)
	defer cancel()
	reference, err := searchIndex.FindByUuid(ctx, nil, vmUuid, true, nil)
	if err != nil {
		log.Printf("[ERROR] " + err.Error())
	}
	managedObjectRef := reference.Reference()
	virtualMachine := object.NewVirtualMachine(client.Client, managedObjectRef)
	return virtualMachine, err
}
