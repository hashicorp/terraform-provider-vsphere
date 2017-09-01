package vsphere

import (
	"context"

	"github.com/vmware/govmomi"
	"github.com/vmware/govmomi/object"
	"github.com/vmware/govmomi/vim25/methods"
	"github.com/vmware/govmomi/vim25/soap"
	"github.com/vmware/govmomi/vim25/types"
)

// soapFault extracts the SOAP fault from an error fault, if it exists. Check
// the returned boolean value to see if you have a SoapFault.
func soapFault(err error) (*soap.Fault, bool) {
	if soap.IsSoapFault(err) {
		return soap.ToSoapFault(err), true
	}
	return nil, false
}

// vimSoapFault extracts the VIM fault Check the returned boolean value to see
// if you have a fault, which will need to be further asserted into the error
// that you are looking for.
func vimSoapFault(err error) (types.AnyType, bool) {
	if sf, ok := soapFault(err); ok {
		return sf.VimFault(), true
	}
	return nil, false
}

// isManagedObjectNotFoundError checks an error to see if it's of the
// ManagedObjectNotFound type.
func isManagedObjectNotFoundError(err error) bool {
	if f, ok := vimSoapFault(err); ok {
		if _, ok := f.(types.ManagedObjectNotFound); ok {
			return true
		}
	}
	return false
}

// renameObject renames a MO and tracks the task to make sure it completes.
func renameObject(client *govmomi.Client, ref types.ManagedObjectReference, new string) error {
	req := types.Rename_Task{
		This:    ref,
		NewName: new,
	}

	rctx, rcancel := context.WithTimeout(context.Background(), defaultAPITimeout)
	defer rcancel()
	res, err := methods.Rename_Task(rctx, client.Client, &req)
	if err != nil {
		return err
	}

	t := object.NewTask(client.Client, res.Returnval)
	tctx, tcancel := context.WithTimeout(context.Background(), defaultAPITimeout)
	defer tcancel()
	return t.Wait(tctx)
}
