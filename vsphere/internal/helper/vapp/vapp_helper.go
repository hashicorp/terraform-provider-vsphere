package vapp

import (
	"fmt"
	"log"
	"regexp"

	"github.com/hashicorp/terraform/helper/schema"
	"github.com/terraform-providers/terraform-provider-vsphere/vsphere/internal/helper/virtualmachine"
	"github.com/vmware/govmomi"
	"github.com/vmware/govmomi/object"
	"github.com/vmware/govmomi/vim25/types"
)

// IsVAppCdrom takes VirtualCdrom and determines if it is needed for vApp ISO
// transport. It does this by first checking if it has an ISO inserted that
// matches the vApp ISO naming pattern. If it does, then the next step is to
// see if vApp ISO transport is supported on the VM. If both of those
// conditions are met, then the CDROM is considered in use for vApp transport.
func VerifyVAppCdrom(d *schema.ResourceData, device *types.VirtualCdrom, l object.VirtualDeviceList, c *govmomi.Client) (bool, error) {
	log.Printf("[DEBUG] IsVAppCdrom: Checking if CDROM is using a vApp ISO")
	// If the CDROM is using VirtualCdromIsoBackingInfo and matches the ISO
	// naming pattern, it has been used as a vApp CDROM, and we can move on to
	// checking if the parent VM supports ISO transport.
	if backing, ok := device.Backing.(*types.VirtualCdromIsoBackingInfo); ok {
		dp := &object.DatastorePath{}
		if ok := dp.FromString(backing.FileName); !ok {
			// If the ISO path can not be read, we can't tell if a vApp ISO is
			// connected.
			log.Printf("[DEBUG] IsVAppCdrom: Cannot read ISO path, cannot determine if CDROM is used for vApp")
			return false, nil
		}
		// The pattern used for vApp ISO naming is
		// "<vmname>/_ovfenv-<vmname>.iso"
		re := regexp.MustCompile(".*/_ovfenv-.*.iso")
		if !re.MatchString(dp.Path) {
			log.Printf("[DEBUG] IsVAppCdrom: ISO is name does not match vApp ISO naming pattern (<vmname>/_ovfenv-<vmname>.iso): %s", dp.Path)
			return false, nil
		}
	} else {
		// vApp CDROMs must be backed by an ISO.
		log.Printf("[DEBUG] IsVAppCdrom: CDROM is not backed by an ISO")
		return false, nil
	}
	log.Printf("[DEBUG] IsVAppCdrom: CDROM has a vApp ISO inserted")
	// Set the vApp transport methods
	tm, err := vAppTransportMethods(d.Id(), c)
	if err != nil {
		return false, err
	}
	for _, t := range tm {
		if t == "iso" {
			log.Printf("[DEBUG] IsVAppCdrom: vApp ISO transport is supported")
			return true, nil
		}
	}
	log.Printf("[DEBUG] IsVAppCdrom: vApp ISO transport is not required")
	return false, nil
}

// VerifyVAppTransport validates that all the required components are included in
// the virtual machine configuration if vApp properties are set.
func VerifyVAppTransport(d *schema.ResourceDiff, c *govmomi.Client) error {
	log.Printf("[DEBUG] VAppDiffOperation: Verifying configuration meets requirements for vApp transport")
	vApp := d.Get("vapp")
	if len(vApp.([]interface{})) == 0 || len(vApp.([]interface{})[0].(map[string]interface{})["properties"].(map[string]interface{})) == 0 {
		// Properties are not set, so no additional configuration checks are needed.
		log.Printf("[DEBUG] VAppDiffOperation: No vApp properties are set, so no additional checks are required")
		return nil
	}
	// Get the template info so we can check the supported vApp transport methods.
	t := d.Get("clone").([]interface{})
	tm, _ := vAppTransportMethods(t[0].(map[string]interface{})["template_uuid"].(string), c)
	// Check if there is a client CDROM device configured.
	cl := d.Get("cdrom")
	for _, c := range cl.([]interface{}) {
		if c.(map[string]interface{})["client_device"].(bool) == true {
			// There is a device configured that can support vApp ISO transport if needed
			log.Printf("[DEBUG] VAppDiffOperation: Client CDROM device exists which can support ISO transport")
			return nil
		}
	}
	// Iterate over each transport and see if ISO transport is supported.
	for _, m := range tm {
		if m == "iso" && len(tm) == 1 {
			return fmt.Errorf("this virtual machine requires a client CDROM device to deliver vApp properties")
		}
	}
	log.Printf("[DEBUG] VAppDiffOperation: ISO transport is not supported on this virtual machine or multiple transport options exist")
	return nil
}

// vAppTransportMethods returns a list of transport methods supported for
// delivering vApp configuration to a virtual machine.
func vAppTransportMethods(id string, c *govmomi.Client) (tm []string, err error) {
	// Get virtual machine so we can check read ovf environment
	vm, err := virtualmachine.FromUUID(c, id)
	if err != nil {
		return tm, err
	}
	vprops, err := virtualmachine.Properties(vm)
	if err != nil {
		return tm, err
	}
	vconfig := vprops.Config.VAppConfig
	// If the VM doesn't support vApp properties, this will be nil.
	if vconfig != nil {
		tm = vconfig.GetVmConfigInfo().OvfEnvironmentTransport
	}
	return tm, nil
}
