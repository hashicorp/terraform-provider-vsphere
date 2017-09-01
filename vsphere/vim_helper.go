package vsphere

import (
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
