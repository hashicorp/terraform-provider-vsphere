package vsphere

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path"
	"reflect"
	"regexp"
	"strconv"
	"testing"
	"time"

	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
	"github.com/terraform-providers/terraform-provider-vsphere/vsphere/internal/helper/datastore"
	"github.com/terraform-providers/terraform-provider-vsphere/vsphere/internal/helper/dvportgroup"
	"github.com/terraform-providers/terraform-provider-vsphere/vsphere/internal/helper/folder"
	"github.com/terraform-providers/terraform-provider-vsphere/vsphere/internal/helper/hostsystem"
	"github.com/terraform-providers/terraform-provider-vsphere/vsphere/internal/helper/resourcepool"
	"github.com/terraform-providers/terraform-provider-vsphere/vsphere/internal/helper/storagepod"
	"github.com/terraform-providers/terraform-provider-vsphere/vsphere/internal/helper/viapi"
	"github.com/terraform-providers/terraform-provider-vsphere/vsphere/internal/helper/virtualdisk"
	"github.com/terraform-providers/terraform-provider-vsphere/vsphere/internal/helper/virtualmachine"
	"github.com/terraform-providers/terraform-provider-vsphere/vsphere/internal/virtualdevice"
	"github.com/vmware/govmomi"
	"github.com/vmware/govmomi/object"
	"github.com/vmware/govmomi/vim25/mo"
	"github.com/vmware/govmomi/vim25/types"
	"github.com/vmware/vic/pkg/vsphere/tags"
)

// testAccResourceVSphereEmpty provides an empty provider config to pass some
// error tests with an empty state. This is to ensure there's no dangling
// resources on the destroy check if for some reason some state gets written.
const testAccResourceVSphereEmpty = `
provider vsphere{}
`

// testCheckVariables bundles common variables needed by various test checkers.
type testCheckVariables struct {
	// A client for various operations.
	client *govmomi.Client

	// The client for tagging operations.
	tagsClient *tags.RestClient

	// The subject resource's ID.
	resourceID string

	// The subject resource's attributes.
	resourceAttributes map[string]string

	// The ESXi host that a various API call is directed at.
	esxiHost string

	// The datacenter that a various API call is directed at.
	datacenter string

	// A timeout to pass to various context creation calls.
	timeout time.Duration
}

func testClientVariablesForResource(s *terraform.State, addr string) (testCheckVariables, error) {
	rs, ok := s.RootModule().Resources[addr]
	if !ok {
		return testCheckVariables{}, fmt.Errorf("%s not found in state", addr)
	}

	return testCheckVariables{
		client:             testAccProvider.Meta().(*VSphereClient).vimClient,
		tagsClient:         testAccProvider.Meta().(*VSphereClient).tagsClient,
		resourceID:         rs.Primary.ID,
		resourceAttributes: rs.Primary.Attributes,
		esxiHost:           os.Getenv("VSPHERE_ESXI_HOST"),
		datacenter:         os.Getenv("VSPHERE_DATACENTER"),
		timeout:            time.Minute * 5,
	}, nil
}

// testAccESXiFlagSet returns true if VSPHERE_TEST_ESXI is set.
func testAccESXiFlagSet() bool {
	return os.Getenv("VSPHERE_TEST_ESXI") != ""
}

// testAccSkipIfNotEsxi skips a test if VSPHERE_TEST_ESXI is not set.
func testAccSkipIfNotEsxi(t *testing.T) {
	if !testAccESXiFlagSet() {
		t.Skip("set VSPHERE_TEST_ESXI to run ESXi-specific acceptance tests")
	}
}

// testAccSkipIfEsxi skips a test if VSPHERE_TEST_ESXI is set.
func testAccSkipIfEsxi(t *testing.T) {
	if testAccESXiFlagSet() {
		t.Skip("test skipped as VSPHERE_TEST_ESXI is set")
	}
}

// expectErrorIfNotVirtualCenter returns the error message that
// viapi.ValidateVirtualCenter returns if VSPHERE_TEST_ESXI is set, to allow for test
// cases that will still run on ESXi, but will expect validation failure.
func expectErrorIfNotVirtualCenter() *regexp.Regexp {
	if testAccESXiFlagSet() {
		return regexp.MustCompile(viapi.ErrVirtualCenterOnly)
	}
	return nil
}

// copyStatePtr returns a TestCheckFunc that copies the reference to the test
// run's state to t. This allows access to the state data in later steps where
// it's not normally accessible (ie: in pre-config parts in another test step).
func copyStatePtr(t **terraform.State) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		*t = s
		return nil
	}
}

// copyState returns a TestCheckFunc that returns a deep copy of the state.
// Unlike copyStatePtr, this state has de-coupled from the in-flight state, so
// it will not be modified on subsequent steps and hence will possibly drift.
// It can be used to access values of the state at a certain step.
func copyState(t **terraform.State) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		*t = s.DeepCopy()
		return nil
	}
}

// testGetPortGroup is a convenience method to fetch a static port group
// resource for testing.
func testGetPortGroup(s *terraform.State, resourceName string) (*types.HostPortGroup, error) {
	tVars, err := testClientVariablesForResource(s, fmt.Sprintf("vsphere_host_port_group.%s", resourceName))
	if err != nil {
		return nil, err
	}

	hsID, name, err := splitHostPortGroupID(tVars.resourceID)
	if err != nil {
		return nil, err
	}
	ns, err := hostNetworkSystemFromHostSystemID(tVars.client, hsID)
	if err != nil {
		return nil, fmt.Errorf("error loading host network system: %s", err)
	}

	return hostPortGroupFromName(tVars.client, ns, name)
}

// testGetVirtualMachine is a convenience method to fetch a virtual machine by
// resource name.
func testGetVirtualMachine(s *terraform.State, resourceName string) (*object.VirtualMachine, error) {
	tVars, err := testClientVariablesForResource(s, fmt.Sprintf("vsphere_virtual_machine.%s", resourceName))
	if err != nil {
		return nil, err
	}
	uuid, ok := tVars.resourceAttributes["uuid"]
	if !ok {
		return nil, fmt.Errorf("resource %q has no UUID", resourceName)
	}
	return virtualmachine.FromUUID(tVars.client, uuid)
}

// testGetVirtualMachineProperties is a convenience method that adds an extra
// step to testGetVirtualMachine to get the properties of a virtual machine.
func testGetVirtualMachineProperties(s *terraform.State, resourceName string) (*mo.VirtualMachine, error) {
	vm, err := testGetVirtualMachine(s, resourceName)
	if err != nil {
		return nil, err
	}
	return virtualmachine.Properties(vm)
}

// testGetVirtualMachineHost returns the HostSystem for the host that this
// virtual machine is currently on.
func testGetVirtualMachineHost(s *terraform.State, resourceName string) (*object.HostSystem, error) {
	tVars, err := testClientVariablesForResource(s, fmt.Sprintf("vsphere_virtual_machine.%s", resourceName))
	if err != nil {
		return nil, err
	}
	vprops, err := testGetVirtualMachineProperties(s, resourceName)
	if err != nil {
		return nil, err
	}
	return hostsystem.FromID(tVars.client, vprops.Runtime.Host.Value)
}

// testGetVirtualMachineResourcePool returns the ResourcePool object for the
// resource pool this VM is currently in.
func testGetVirtualMachineResourcePool(s *terraform.State, resourceName string) (*object.ResourcePool, error) {
	tVars, err := testClientVariablesForResource(s, fmt.Sprintf("vsphere_virtual_machine.%s", resourceName))
	if err != nil {
		return nil, err
	}
	vprops, err := testGetVirtualMachineProperties(s, resourceName)
	if err != nil {
		return nil, err
	}
	return resourcepool.FromID(tVars.client, vprops.ResourcePool.Value)
}

// testGetVirtualMachineSCSIBusState reads the SCSI bus state for the supplied
// virtual machine.
func testGetVirtualMachineSCSIBusState(s *terraform.State, resourceName string) (string, error) {
	tVars, err := testClientVariablesForResource(s, fmt.Sprintf("vsphere_virtual_machine.%s", resourceName))
	if err != nil {
		return "", err
	}
	vprops, err := testGetVirtualMachineProperties(s, resourceName)
	if err != nil {
		return "", err
	}
	count, err := strconv.Atoi(tVars.resourceAttributes["scsi_controller_count"])
	if err != nil {
		return "", err
	}
	l := object.VirtualDeviceList(vprops.Config.Hardware.Device)
	return virtualdevice.ReadSCSIBusState(l, count), nil
}

func testGetDatacenter(s *terraform.State, resourceName string) (*object.Datacenter, error) {
	tVars, err := testClientVariablesForResource(s, fmt.Sprintf("vsphere_datacenter.%s", resourceName))
	if err != nil {
		return nil, err
	}
	dcName, ok := tVars.resourceAttributes["name"]
	if !ok {
		return nil, fmt.Errorf("Datacenter resource %q has no name", resourceName)
	}
	return getDatacenter(tVars.client, dcName)
}

func testGetDatacenterCustomAttributes(s *terraform.State, resourceName string) (*mo.Datacenter, error) {
	dc, err := testGetDatacenter(s, resourceName)
	if err != nil {
		return nil, err
	}
	return datacenterCustomAttributes(dc)
}

// testPowerOffVM does an immediate power-off of the supplied virtual machine
// resource defined by the supplied resource address name. It is used to help
// set up a test scenarios where a VM is powered off.
func testPowerOffVM(s *terraform.State, resourceName string) error {
	vm, err := testGetVirtualMachine(s, resourceName)
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), defaultAPITimeout)
	defer cancel()
	task, err := vm.PowerOff(ctx)
	if err != nil {
		return fmt.Errorf("error powering off VM: %s", err)
	}
	tctx, tcancel := context.WithTimeout(context.Background(), defaultAPITimeout)
	defer tcancel()
	if err := task.Wait(tctx); err != nil {
		return fmt.Errorf("error waiting for poweroff: %s", err)
	}
	return nil
}

// testRenameVMFirstDisk renames the first disk in a virtual machine
// configuration and re-attaches it to the virtual machine under the new name.
func testRenameVMFirstDisk(s *terraform.State, resourceName string, new string) error {
	tVars, err := testClientVariablesForResource(s, fmt.Sprintf("vsphere_virtual_machine.%s", resourceName))
	if err != nil {
		return err
	}
	vm, err := testGetVirtualMachine(s, resourceName)
	if err != nil {
		return err
	}
	vprops, err := testGetVirtualMachineProperties(s, resourceName)
	if err != nil {
		return err
	}
	if err := testPowerOffVM(s, resourceName); err != nil {
		return err
	}
	dcp, err := folder.RootPathParticleVM.SplitDatacenter(vm.InventoryPath)
	if err != nil {
		return err
	}
	dc, err := getDatacenter(tVars.client, dcp)
	if err != nil {
		return err
	}

	var dcSpec []types.BaseVirtualDeviceConfigSpec
	for _, d := range vprops.Config.Hardware.Device {
		if oldDisk, ok := d.(*types.VirtualDisk); ok {
			newFileName, err := virtualdisk.Move(
				tVars.client,
				oldDisk.Backing.(*types.VirtualDiskFlatVer2BackingInfo).FileName,
				dc,
				new,
				nil,
			)
			if err != nil {
				return err
			}
			newDisk := &types.VirtualDisk{
				VirtualDevice: types.VirtualDevice{
					Backing: &types.VirtualDiskFlatVer2BackingInfo{
						VirtualDeviceFileBackingInfo: types.VirtualDeviceFileBackingInfo{
							FileName: newFileName,
						},
						ThinProvisioned: oldDisk.Backing.(*types.VirtualDiskFlatVer2BackingInfo).ThinProvisioned,
						EagerlyScrub:    oldDisk.Backing.(*types.VirtualDiskFlatVer2BackingInfo).EagerlyScrub,
						DiskMode:        oldDisk.Backing.(*types.VirtualDiskFlatVer2BackingInfo).DiskMode,
					},
				},
			}
			newDisk.ControllerKey = oldDisk.ControllerKey
			newDisk.UnitNumber = oldDisk.UnitNumber

			dspec, err := object.VirtualDeviceList{oldDisk}.ConfigSpec(types.VirtualDeviceConfigSpecOperationRemove)
			if err != nil {
				return err
			}
			dspec[0].GetVirtualDeviceConfigSpec().FileOperation = ""
			aspec, err := object.VirtualDeviceList{newDisk}.ConfigSpec(types.VirtualDeviceConfigSpecOperationAdd)
			if err != nil {
				return err
			}
			aspec[0].GetVirtualDeviceConfigSpec().FileOperation = ""
			dcSpec = append(dcSpec, dspec...)
			dcSpec = append(dcSpec, aspec...)
			break
		}
	}
	if len(dcSpec) < 1 {
		return fmt.Errorf("could not find a virtual disk on virtual machine %q", vm.InventoryPath)
	}
	spec := types.VirtualMachineConfigSpec{
		DeviceChange: dcSpec,
	}
	return virtualmachine.Reconfigure(vm, spec)
}

// testDeleteVMDisk deletes a VMDK file from the virtual machine directory. It
// doesn't check configuration other than to look for the directory the VMX
// file is in and is mainly meant to serve as a cleanup method.
func testDeleteVMDisk(s *terraform.State, resourceName string, name string) error {
	tVars, err := testClientVariablesForResource(s, "vsphere_virtual_machine.vm")
	if err != nil {
		return err
	}
	vm, err := testGetVirtualMachine(s, "vm")
	if err != nil {
		return err
	}
	props, err := testGetVirtualMachineProperties(s, "vm")
	if err != nil {
		return err
	}
	vmxPath, success := virtualdisk.DatastorePathFromString(props.Config.Files.VmPathName)
	if !success {
		return fmt.Errorf("could not parse VMX path %q", props.Config.Files.VmPathName)
	}
	dcp, err := folder.RootPathParticleVM.SplitDatacenter(vm.InventoryPath)
	if err != nil {
		return err
	}
	dc, err := getDatacenter(tVars.client, dcp)
	if err != nil {
		return err
	}
	p := &object.DatastorePath{
		Datastore: vmxPath.Datastore,
		Path:      path.Join(path.Dir(vmxPath.Path), name),
	}
	return virtualdisk.Delete(tVars.client, p.String(), dc)
}

// testDeleteVM deletes the virtual machine. This is used to test resource
// re-creation if TF cannot locate a VM that is in state any more.
func testDeleteVM(s *terraform.State, resourceName string) error {
	if err := testPowerOffVM(s, resourceName); err != nil {
		return err
	}
	vm, err := testGetVirtualMachine(s, resourceName)
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), defaultAPITimeout)
	defer cancel()
	task, err := vm.Destroy(ctx)
	if err != nil {
		return fmt.Errorf("error destroying virtual machine: %s", err)
	}
	tctx, tcancel := context.WithTimeout(context.Background(), defaultAPITimeout)
	defer tcancel()
	return task.Wait(tctx)
}

// testGetTagCategory gets a tag category by name.
func testGetTagCategory(s *terraform.State, resourceName string) (*tags.Category, error) {
	tVars, err := testClientVariablesForResource(s, fmt.Sprintf("vsphere_tag_category.%s", resourceName))
	if err != nil {
		return nil, err
	}
	ctx, cancel := context.WithTimeout(context.Background(), defaultAPITimeout)
	defer cancel()
	category, err := tVars.tagsClient.GetCategory(ctx, tVars.resourceID)
	if err != nil {
		return nil, fmt.Errorf("could not get tag category for ID %q: %s", tVars.resourceID, err)
	}

	return category, nil
}

// testGetTag gets a tag by name.
func testGetTag(s *terraform.State, resourceName string) (*tags.Tag, error) {
	tVars, err := testClientVariablesForResource(s, fmt.Sprintf("vsphere_tag.%s", resourceName))
	if err != nil {
		return nil, err
	}
	ctx, cancel := context.WithTimeout(context.Background(), defaultAPITimeout)
	defer cancel()
	tag, err := tVars.tagsClient.GetTag(ctx, tVars.resourceID)
	if err != nil {
		return nil, fmt.Errorf("could not get tag for ID %q: %s", tVars.resourceID, err)
	}

	return tag, nil
}

// testObjectHasTags checks an object to see if it has the tags that currently
// exist in the Terrafrom state under the resource with the supplied name.
func testObjectHasTags(s *terraform.State, client *tags.RestClient, obj object.Reference, tagResName string) error {
	var expectedIDs []string
	if tagRS, ok := s.RootModule().Resources[fmt.Sprintf("vsphere_tag.%s", tagResName)]; ok {
		expectedIDs = append(expectedIDs, tagRS.Primary.ID)
	} else {
		var n int
		for {
			multiTagRS, ok := s.RootModule().Resources[fmt.Sprintf("vsphere_tag.%s.%d", tagResName, n)]
			if !ok {
				break
			}
			expectedIDs = append(expectedIDs, multiTagRS.Primary.ID)
			n++
		}
	}
	if len(expectedIDs) < 1 {
		return fmt.Errorf("could not find state for vsphere_tag.%s or vsphere_tag.%s.*", tagResName, tagResName)
	}

	objID := obj.Reference().Value
	objType, err := tagTypeForObject(obj)
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), defaultAPITimeout)
	defer cancel()
	actualIDs, err := client.ListAttachedTags(ctx, objID, objType)
	if err != nil {
		return err
	}

	for _, expectedID := range expectedIDs {
		var found bool
		for _, actualID := range actualIDs {
			if expectedID == actualID {
				found = true
			}
		}
		if !found {
			return fmt.Errorf("could not find expected tag ID %q attached to object %q", expectedID, obj.Reference().Value)
		}
	}

	return nil
}

// testObjectHasNoTags checks to make sure that an object has no tags attached
// to it. The parameters are the same as testObjectHasTags, but no tag resource
// needs to be supplied.
func testObjectHasNoTags(s *terraform.State, client *tags.RestClient, obj object.Reference) error {
	objID := obj.Reference().Value
	objType, err := tagTypeForObject(obj)
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), defaultAPITimeout)
	defer cancel()
	actualIDs, err := client.ListAttachedTags(ctx, objID, objType)
	if err != nil {
		return err
	}
	if len(actualIDs) > 0 {
		return fmt.Errorf("object %q still has tags (%#v)", obj.Reference().Value, actualIDs)
	}
	return nil
}

// testGetDatastore gets the datastore at the supplied full address. This
// function works for multiple datastore resources (example:
// vsphere_nas_datastore and vsphere_vmfs_datastore), hence the need for the
// full resource address including the resource type.
func testGetDatastore(s *terraform.State, resAddr string) (*object.Datastore, error) {
	vars, err := testClientVariablesForResource(s, resAddr)
	if err != nil {
		return nil, err
	}
	return datastore.FromID(vars.client, vars.resourceID)
}

// testGetDatastoreProperties is a convenience method that adds an extra step
// to testGetDatastore to get the properties of a datastore.
func testGetDatastoreProperties(s *terraform.State, datastoreType string, resourceName string) (*mo.Datastore, error) {
	ds, err := testGetDatastore(s, "vsphere_"+datastoreType+"_datastore."+resourceName)
	if err != nil {
		return nil, err
	}
	return datastore.Properties(ds)
}

// testAccResourceVSphereDatastoreCheckTags is a check to ensure that the
// supplied datastore has had the tags that have been created with the supplied
// tag resource name attached.
//
// The full datastore resource address is needed as this functions across
// multiple datastore resource types.
func testAccResourceVSphereDatastoreCheckTags(dsResAddr, tagResName string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		ds, err := testGetDatastore(s, dsResAddr)
		if err != nil {
			return err
		}
		tagsClient, err := testAccProvider.Meta().(*VSphereClient).TagsClient()
		if err != nil {
			return err
		}
		return testObjectHasTags(s, tagsClient, ds, tagResName)
	}
}

// testGetFolder is a convenience method to fetch a folder by resource name.
func testGetFolder(s *terraform.State, resourceName string) (*object.Folder, error) {
	tVars, err := testClientVariablesForResource(s, fmt.Sprintf("vsphere_folder.%s", resourceName))
	if err != nil {
		return nil, err
	}
	return folder.FromID(tVars.client, tVars.resourceID)
}

// testGetFolderProperties is a convenience method that adds an extra step to
// testGetFolder to get the properties of a folder.
func testGetFolderProperties(s *terraform.State, resourceName string) (*mo.Folder, error) {
	f, err := testGetFolder(s, resourceName)
	if err != nil {
		return nil, err
	}
	return folder.Properties(f)
}

// testGetDVS is a convenience method to fetch a DVS by resource name.
func testGetDVS(s *terraform.State, resourceName string) (*object.VmwareDistributedVirtualSwitch, error) {
	tVars, err := testClientVariablesForResource(s, fmt.Sprintf("vsphere_distributed_virtual_switch.%s", resourceName))
	if err != nil {
		return nil, err
	}
	return dvsFromUUID(tVars.client, tVars.resourceID)
}

// testGetDVSProperties is a convenience method that adds an extra step to
// testGetDVS to get the properties of a DVS.
func testGetDVSProperties(s *terraform.State, resourceName string) (*mo.VmwareDistributedVirtualSwitch, error) {
	dvs, err := testGetDVS(s, resourceName)
	if err != nil {
		return nil, err
	}
	return dvsProperties(dvs)
}

// testGetDVPortgroup is a convenience method to fetch a DV portgroup by resource name.
func testGetDVPortgroup(s *terraform.State, resourceName string) (*object.DistributedVirtualPortgroup, error) {
	tVars, err := testClientVariablesForResource(s, fmt.Sprintf("vsphere_distributed_port_group.%s", resourceName))
	if err != nil {
		return nil, err
	}
	dvsID := tVars.resourceAttributes["distributed_virtual_switch_uuid"]
	return dvportgroup.FromKey(tVars.client, dvsID, tVars.resourceID)
}

// testGetDVPortgroupProperties is a convenience method that adds an extra step to
// testGetDVPortgroup to get the properties of a DV portgroup.
func testGetDVPortgroupProperties(s *terraform.State, resourceName string) (*mo.DistributedVirtualPortgroup, error) {
	dvs, err := testGetDVPortgroup(s, resourceName)
	if err != nil {
		return nil, err
	}
	return dvportgroup.Properties(dvs)
}

// testCheckResourceNotAttr is an inverse check of TestCheckResourceAttr. It
// checks to make sure the resource attribute does *not* match a certain value.
func testCheckResourceNotAttr(name, key, value string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		err := resource.TestCheckResourceAttr(name, key, value)(s)
		if err != nil {
			if regexp.MustCompile("[-_.a-zA-Z0-9]\\: Attribute '.*' expected .*, got .*").MatchString(err.Error()) {
				return nil
			}
			return err
		}
		return fmt.Errorf("%s: Attribute '%s' expected to not match %#v", name, key, value)
	}
}

// testGetCustomAttribute gets a custom attribute by name.
func testGetCustomAttribute(s *terraform.State, resourceName string) (*types.CustomFieldDef, error) {
	tVars, err := testClientVariablesForResource(s, fmt.Sprintf("vsphere_custom_attribute.%s", resourceName))
	if err != nil {
		return nil, err
	}

	key, err := strconv.ParseInt(tVars.resourceID, 10, 32)
	if err != nil {
		return nil, err
	}
	fm, err := object.GetCustomFieldsManager(tVars.client.Client)
	if err != nil {
		return nil, err
	}
	ctx, cancel := context.WithTimeout(context.Background(), defaultAPITimeout)
	defer cancel()
	fields, err := fm.Field(ctx)
	if err != nil {
		return nil, err
	}
	field := fields.ByKey(int32(key))

	return field, nil
}

func testResourceHasCustomAttributeValues(s *terraform.State, resourceType string, resourceName string, entity *mo.ManagedEntity) error {
	testVars, err := testClientVariablesForResource(s, fmt.Sprintf("%s.%s", resourceType, resourceName))
	if err != nil {
		return err
	}
	expectedAttrs := make(map[string]string)
	re := regexp.MustCompile(`custom_attributes\.(\d+)`)
	for key, value := range testVars.resourceAttributes {
		if m := re.FindStringSubmatch(key); m != nil {
			expectedAttrs[m[1]] = value
		}
	}

	actualAttrs := make(map[string]string)
	for _, fv := range entity.CustomValue {
		value := fv.(*types.CustomFieldStringValue).Value
		if value != "" {
			actualAttrs[fmt.Sprint(fv.GetCustomFieldValue().Key)] = value
		}
	}

	if !reflect.DeepEqual(expectedAttrs, actualAttrs) {
		return fmt.Errorf("expected custom attributes to be %q, got %q", expectedAttrs, actualAttrs)
	}
	return nil
}

// testDeleteDatastoreFile deletes the specified file from a datastore. If the
// file does not exist, an error is returned.
func testDeleteDatastoreFile(client *govmomi.Client, dsID string, path string) error {
	ds, err := datastore.FromID(client, dsID)
	if err != nil {
		return err
	}
	dc, err := getDatacenter(client, ds.DatacenterPath)
	if err != nil {
		return err
	}
	fm := object.NewFileManager(client.Client)

	ctx, cancel := context.WithTimeout(context.Background(), defaultAPITimeout)
	defer cancel()
	task, err := fm.DeleteDatastoreFile(ctx, path, dc)
	if err != nil {
		return err
	}
	return task.Wait(context.TODO())
}

// testGetDatastoreCluster is a convenience method to fetch a datastore cluster by
// resource name.
func testGetDatastoreCluster(s *terraform.State, resourceName string) (*object.StoragePod, error) {
	vars, err := testClientVariablesForResource(s, fmt.Sprintf("%s.%s", resourceVSphereDatastoreClusterName, resourceName))
	if err != nil {
		return nil, err
	}
	return storagepod.FromID(vars.client, vars.resourceID)
}

// testGetDatastoreClusterProperties is a convenience method that adds an extra
// step to testGetDatastoreCluster to get the properties of a StoragePod.
func testGetDatastoreClusterProperties(s *terraform.State, resourceName string) (*mo.StoragePod, error) {
	pod, err := testGetDatastoreCluster(s, resourceName)
	if err != nil {
		return nil, err
	}
	return storagepod.Properties(pod)
}

// testGetDatastoreClusterSDRSVMConfig is a convenience method to fetch a VM's
// SDRS override in a datastore cluster.
func testGetDatastoreClusterSDRSVMConfig(s *terraform.State, resourceName string) (*types.StorageDrsVmConfigInfo, error) {
	vars, err := testClientVariablesForResource(s, fmt.Sprintf("%s.%s", resourceVSphereStorageDrsVMOverrideName, resourceName))
	if err != nil {
		return nil, err
	}

	if vars.resourceID == "" {
		return nil, errors.New("resource ID is empty")
	}

	podID, vmID, err := resourceVSphereStorageDrsVMOverrideParseID(vars.resourceID)
	if err != nil {
		return nil, err
	}

	pod, err := storagepod.FromID(vars.client, podID)
	if err != nil {
		return nil, err
	}

	vm, err := virtualmachine.FromUUID(vars.client, vmID)
	if err != nil {
		return nil, err
	}

	return resourceVSphereStorageDrsVMOverrideFindEntry(pod, vm)
}
