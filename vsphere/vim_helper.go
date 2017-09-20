package vsphere

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/vmware/govmomi"
	"github.com/vmware/govmomi/object"
	"github.com/vmware/govmomi/vim25/methods"
	"github.com/vmware/govmomi/vim25/soap"
	"github.com/vmware/govmomi/vim25/types"
)

// errVirtualCenterOnly is the error message that validateVirtualCenter returns.
const errVirtualCenterOnly = "this operation is only supported on vCenter"

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

// isResourceInUseError checks an error to see if it's of the
// ResourceInUse type.
func isResourceInUseError(err error) bool {
	if f, ok := vimSoapFault(err); ok {
		if _, ok := f.(types.ResourceInUse); ok {
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

// validateVirtualCenter ensures that the client is connected to vCenter.
func validateVirtualCenter(c *govmomi.Client) error {
	if c.ServiceContent.About.ApiType != "VirtualCenter" {
		return errors.New(errVirtualCenterOnly)
	}
	return nil
}

// vSphereVersion represents a version number of a ESXi/vCenter server
// instance.
type vSphereVersion struct {
	// The product name. Example: "VMware vCenter Server", or "VMware ESXi".
	product string

	// The major version. Example: If "6.5.1" is the full version, the major
	// version is "6".
	major int

	// The minor version. Example: If "6.5.1" is the full version, the minor
	// version is "5".
	minor int

	// The patch version. Example: If "6.5.1" is the full version, the patch
	// version is "1".
	patch int

	// The build number. This is usually a lengthy integer. This number should
	// not be used to compare versions on its own.
	build int
}

// parseVersion creates a new vSphereVersion from a parsed version string and
// build number.
func parseVersion(name, version, build string) (vSphereVersion, error) {
	v := vSphereVersion{
		product: name,
	}
	s := strings.Split(version, ".")
	if len(s) > 3 {
		return v, fmt.Errorf("version string %q has more than 3 components", version)
	}
	var err error
	v.major, err = strconv.Atoi(s[0])
	if err != nil {
		return v, fmt.Errorf("could not parse major version %q from version string %q", s[0], version)
	}
	v.minor, err = strconv.Atoi(s[1])
	if err != nil {
		return v, fmt.Errorf("could not parse minor version %q from version string %q", s[1], version)
	}
	v.patch, err = strconv.Atoi(s[2])
	if err != nil {
		return v, fmt.Errorf("could not parse patch version %q from version string %q", s[2], version)
	}
	v.build, err = strconv.Atoi(build)
	if err != nil {
		return v, fmt.Errorf("could not parse build version string %q", build)
	}

	return v, nil
}

// parseVersionFromAboutInfo returns a populated vSphereVersion from an
// AboutInfo data object.
//
// This function panics if it cannot parse the version correctly, as given our
// source of truth is a valid AboutInfo object, such an error is indicative of
// a major issue with our version parsing logic.
func parseVersionFromAboutInfo(info types.AboutInfo) vSphereVersion {
	v, err := parseVersion(info.Name, info.Version, info.Build)
	if err != nil {
		panic(err)
	}
	return v
}

// parseVersionFromClient returns a populated vSphereVersion from a client
// connection.
func parseVersionFromClient(client *govmomi.Client) vSphereVersion {
	return parseVersionFromAboutInfo(client.Client.ServiceContent.About)
}

// String implements stringer for vSphereVersion.
func (v vSphereVersion) String() string {
	return fmt.Sprintf("%s %d.%d.%d build-%d", v.product, v.major, v.minor, v.patch, v.build)
}

// ProductEqual returns true if this version's product name is the same as the
// supplied version's name.
func (v vSphereVersion) ProductEqual(other vSphereVersion) bool {
	return v.product == other.product
}

// newerVersion checks the major/minor/patch part of the version to see it's
// higher than the version supplied in other. This is broken off from the main
// test so that it can be checked in Older before the build number is compared.
func (v vSphereVersion) newerVersion(other vSphereVersion) bool {
	if v.major > other.major {
		return true
	}
	if v.minor > other.minor {
		return true
	}
	if v.patch > other.patch {
		return true
	}
	return false
}

// Newer returns true if this version's product is the same, and composite of
// the version and build numbers, are newer than the supplied version's
// information.
func (v vSphereVersion) Newer(other vSphereVersion) bool {
	if !v.ProductEqual(other) {
		return false
	}
	if v.newerVersion(other) {
		return true
	}

	// Double check this version is not actually older by version number before
	// moving on to the build number
	if v.olderVersion(other) {
		return false
	}

	if v.build > other.build {
		return true
	}
	return false
}

// olderVersion checks the major/minor/patch part of the version to see it's
// older than the version supplied in other. This is broken off from the main
// test so that it can be checked in Newer before the build number is compared.
func (v vSphereVersion) olderVersion(other vSphereVersion) bool {
	if v.major < other.major {
		return true
	}
	if v.minor < other.minor {
		return true
	}
	if v.patch < other.patch {
		return true
	}
	return false
}

// Older returns true if this version's product is the same, and composite of
// the version and build numbers, are older than the supplied version's
// information.
func (v vSphereVersion) Older(other vSphereVersion) bool {
	if !v.ProductEqual(other) {
		return false
	}
	if v.olderVersion(other) {
		return true
	}

	// Double check this version is not actually newer by version number before
	// moving on to the build number
	if v.newerVersion(other) {
		return false
	}

	if v.build < other.build {
		return true
	}
	return false
}

// Equal returns true if the version is equal to the supplied version.
func (v vSphereVersion) Equal(other vSphereVersion) bool {
	return v.ProductEqual(other) && !v.Older(other) && !v.Newer(other)
}
