package vsphere

import (
	"errors"
	"fmt"

	"github.com/hashicorp/terraform/helper/schema"
	"github.com/terraform-providers/terraform-provider-vsphere/vsphere/internal/helper/structure"
	"github.com/vmware/govmomi/vim25/types"
)

// formatNasDatastoreIDMismatch is a error message format string that is given
// when two NAS datastore IDs mismatch.
const formatNasDatastoreIDMismatch = "datastore ID on host %q (%s) does not original datastore ID (%s)"

func resourceVSphereNasDatastore() *schema.Resource {
	s := map[string]*schema.Schema{
		"name": {
			Type:        schema.TypeString,
			Description: "The name of the datastore.",
			Required:    true,
		},
		"host_system_ids": &schema.Schema{
			Type:        schema.TypeSet,
			Description: "The managed object IDs of the hosts to mount the datastore on.",
			Elem:        &schema.Schema{Type: schema.TypeString},
			MinItems:    1,
			Required:    true,
		},
		"folder": &schema.Schema{
			Type:        schema.TypeString,
			Description: "The path to the datastore folder to put the datastore in.",
			Optional:    true,
			StateFunc:   normalizeFolderPath,
		},
	}
	structure.MergeSchema(s, schemaHostNasVolumeSpec())
	structure.MergeSchema(s, schemaDatastoreSummary())

	// Add tags schema
	s[vSphereTagAttributeKey] = tagsSchema()

	return &schema.Resource{
		Create: resourceVSphereNasDatastoreCreate,
		Read:   resourceVSphereNasDatastoreRead,
		Update: resourceVSphereNasDatastoreUpdate,
		Delete: resourceVSphereNasDatastoreDelete,
		Importer: &schema.ResourceImporter{
			State: resourceVSphereNasDatastoreImport,
		},
		Schema: s,
	}
}

func resourceVSphereNasDatastoreCreate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*VSphereClient).vimClient

	// Load up the tags client, which will validate a proper vCenter before
	// attempting to proceed if we have tags defined.
	tagsClient, err := tagsClientIfDefined(d, meta)
	if err != nil {
		return err
	}

	hosts := structure.SliceInterfacesToStrings(d.Get("host_system_ids").(*schema.Set).List())
	p := &nasDatastoreMountProcessor{
		client:   client,
		oldHSIDs: nil,
		newHSIDs: hosts,
		volSpec:  expandHostNasVolumeSpec(d),
	}
	ds, err := p.processMountOperations()
	if ds != nil {
		d.SetId(ds.Reference().Value)
	}
	if err != nil {
		return fmt.Errorf("error mounting datastore: %s", err)
	}

	// Move the datastore to the correct folder first, if specified.
	folder := d.Get("folder").(string)
	if !pathIsEmpty(folder) {
		if err := moveDatastoreToFolderRelativeHostSystemID(client, ds, hosts[0], folder); err != nil {
			return fmt.Errorf("error moving datastore to folder: %s", err)
		}
	}

	// Apply any pending tags now
	if tagsClient != nil {
		if err := processTagDiff(tagsClient, d, ds); err != nil {
			return err
		}
	}

	// Done
	return resourceVSphereNasDatastoreRead(d, meta)
}

func resourceVSphereNasDatastoreRead(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*VSphereClient).vimClient
	id := d.Id()
	ds, err := datastoreFromID(client, id)
	if err != nil {
		return fmt.Errorf("cannot find datastore: %s", err)
	}
	props, err := datastoreProperties(ds)
	if err != nil {
		return fmt.Errorf("could not get properties for datastore: %s", err)
	}
	if err := flattenDatastoreSummary(d, &props.Summary); err != nil {
		return err
	}

	// Set the folder
	folder, err := rootPathParticleDatastore.SplitRelativeFolder(ds.InventoryPath)
	if err != nil {
		return fmt.Errorf("error parsing datastore path %q: %s", ds.InventoryPath, err)
	}
	d.Set("folder", normalizeFolderPath(folder))

	// Update NAS spec
	if err := flattenHostNasVolume(d, props.Info.(*types.NasDatastoreInfo).Nas); err != nil {
		return err
	}

	// Update mounted hosts
	var mountedHosts []string
	for _, mount := range props.Host {
		mountedHosts = append(mountedHosts, mount.Key.Value)
	}
	if err := d.Set("host_system_ids", mountedHosts); err != nil {
		return err
	}

	// Read tags if we have the ability to do so
	if tagsClient, _ := meta.(*VSphereClient).TagsClient(); tagsClient != nil {
		if err := readTagsForResource(tagsClient, ds, d); err != nil {
			return err
		}
	}

	return nil
}

func resourceVSphereNasDatastoreUpdate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*VSphereClient).vimClient

	// Load up the tags client, which will validate a proper vCenter before
	// attempting to proceed if we have tags defined.
	tagsClient, err := tagsClientIfDefined(d, meta)
	if err != nil {
		return err
	}

	id := d.Id()
	ds, err := datastoreFromID(client, id)
	if err != nil {
		return fmt.Errorf("cannot find datastore: %s", err)
	}

	// Rename this datastore if our name has drifted.
	if d.HasChange("name") {
		if err := renameObject(client, ds.Reference(), d.Get("name").(string)); err != nil {
			return err
		}
	}

	// Update folder if necessary
	if d.HasChange("folder") {
		folder := d.Get("folder").(string)
		if err := moveDatastoreToFolder(client, ds, folder); err != nil {
			return fmt.Errorf("could not move datastore to folder %q: %s", folder, err)
		}
	}

	// Apply any pending tags now
	if tagsClient != nil {
		if err := processTagDiff(tagsClient, d, ds); err != nil {
			return err
		}
	}

	// Process mount/unmount operations.
	o, n := d.GetChange("host_system_ids")

	p := &nasDatastoreMountProcessor{
		client:   client,
		oldHSIDs: structure.SliceInterfacesToStrings(o.(*schema.Set).List()),
		newHSIDs: structure.SliceInterfacesToStrings(n.(*schema.Set).List()),
		volSpec:  expandHostNasVolumeSpec(d),
		ds:       ds,
	}
	// Unmount first
	if err := p.processUnmountOperations(); err != nil {
		return fmt.Errorf("error unmounting hosts: %s", err)
	}
	// Now mount
	if _, err := p.processMountOperations(); err != nil {
		return fmt.Errorf("error mounting hosts: %s", err)
	}

	// Should be done with the update here.
	return resourceVSphereNasDatastoreRead(d, meta)
}

func resourceVSphereNasDatastoreDelete(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*VSphereClient).vimClient
	dsID := d.Id()
	ds, err := datastoreFromID(client, dsID)
	if err != nil {
		return fmt.Errorf("cannot find datastore: %s", err)
	}

	// Unmount the datastore from every host. Once the last host is unmounted we
	// are done and the datastore will delete itself.
	hosts := structure.SliceInterfacesToStrings(d.Get("host_system_ids").(*schema.Set).List())
	p := &nasDatastoreMountProcessor{
		client:   client,
		oldHSIDs: hosts,
		newHSIDs: nil,
		volSpec:  expandHostNasVolumeSpec(d),
		ds:       ds,
	}
	if err := p.processUnmountOperations(); err != nil {
		return fmt.Errorf("error unmounting hosts: %s", err)
	}

	return nil
}

func resourceVSphereNasDatastoreImport(d *schema.ResourceData, meta interface{}) ([]*schema.ResourceData, error) {
	// We support importing a MoRef - so we need to load the datastore and check
	// to make sure 1) it exists, and 2) it's a VMFS datastore. If it is, we are
	// good to go (rest of the stuff will be handled by read on refresh).
	client := meta.(*VSphereClient).vimClient
	id := d.Id()
	ds, err := datastoreFromID(client, id)
	if err != nil {
		return nil, fmt.Errorf("cannot find datastore: %s", err)
	}
	props, err := datastoreProperties(ds)
	if err != nil {
		return nil, fmt.Errorf("could not get properties for datastore: %s", err)
	}

	t := types.HostFileSystemVolumeFileSystemType(props.Summary.Type)
	if !isNasVolume(t) {
		return nil, fmt.Errorf("datastore ID %q is not a NAS datastore", id)
	}

	var accessMode string
	for _, hostMount := range props.Host {
		switch {
		case accessMode == "":
			accessMode = hostMount.MountInfo.AccessMode
		case accessMode != "" && accessMode != hostMount.MountInfo.AccessMode:
			// We don't support selective mount modes across multiple hosts. This
			// should almost never happen (there's no way to do it in the UI so it
			// would need to be done manually). Nonetheless we need to fail here.
			return nil, errors.New("access_mode is inconsistent across configured hosts")
		}
	}
	d.Set("access_mode", accessMode)
	d.Set("type", t)
	return []*schema.ResourceData{d}, nil
}
