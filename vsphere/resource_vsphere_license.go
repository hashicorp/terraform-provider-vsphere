package vsphere

import (
	"errors"
	"log"

	"github.com/hashicorp/terraform/helper/schema"
	"github.com/vmware/govmomi"
	"github.com/vmware/govmomi/license"
	"github.com/vmware/govmomi/vim25/types"
	"golang.org/x/net/context"
)

var (
	// ErrNoSuchKeyFound is an error primarily thrown by the Read method of the resource.
	// The error doesnt display the key itself for security reasons.
	ErrNoSuchKeyFound = errors.New("The key was not found")
)

func resourceVSphereLicense() *schema.Resource {

	return &schema.Resource{

		SchemaVersion: 1,

		Create: resourceVSphereLicenseCreate,
		Read:   resourceVSphereLicenseRead,
		Update: resourceVSphereLicenseUpdate,
		Delete: resourceVSphereLicenseDelete,

		// None of the other resources have an importer method.
		// The Id of the resource is key itself and as the operation is idempotent, the
		// importer will behave quite similar to creation of a key resource.

		Schema: map[string]*schema.Schema{
			"license_key": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},
			"label": &schema.Schema{
				Type:     schema.TypeList,
				Optional: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"key": &schema.Schema{
							Type:     schema.TypeString,
							Required: true,
							ForceNew: true, // TODO: if the key changes then we have to create
							// a new one. But this might never call Update method.
						},
						"value": &schema.Schema{
							Type:     schema.TypeString,
							Required: true,
						},
					},
				},
			},

			// computed properties returned by the API
			"edition_key": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},
			"name": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},
			"total": &schema.Schema{
				Type:     schema.TypeInt,
				Computed: true,
			},
			"used": &schema.Schema{
				Type:     schema.TypeInt,
				Computed: true,
			},
		},
	}
}

func resourceVSphereLicenseCreate(d *schema.ResourceData, meta interface{}) error {
	log.Println("[G] Running the create method")

	client := meta.(*govmomi.Client)
	manager := license.NewManager(client.Client)

	key := d.Get("license_key").(string)

	log.Println("Reading the key from the resource data")
	var finalLabels map[string]string
	var err error
	if labels, ok := d.GetOk("label"); ok {
		finalLabels, err = labelsToMap(labels)
		if err != nil {
			// If errors are printed by terraform then no need to log errors here
			return err
		}
	}
	info, err := manager.Add(context.TODO(), key, finalLabels)
	if err != nil {
		return err
	}

	// This can be used in the read method to set the computed parameters
	d.SetId(info.LicenseKey)

	return resourceVSphereLicenseRead(d, meta)
}

func resourceVSphereLicenseRead(d *schema.ResourceData, meta interface{}) error {

	log.Println("[G] Running the read method")

	client := meta.(*govmomi.Client)
	manager := license.NewManager(client.Client)

	// the key was never present. Should set the ID to false.
	if _, ok := d.GetOk("license_key"); !ok {
		d.SetId("")
		// TODO: Should this be an error?
		return nil
	}

	if info := getLicenseInfoFromKey(d.Get("license_key").(string), manager); info != nil {
		log.Println("[G] Setting the values")
		d.Set("edition_key", info.EditionKey)
		d.Set("total", info.Total)
		d.Set("used", info.Used)
		d.Set("name", info.Name)
	} else {
		return ErrNoSuchKeyFound
	}

	return nil

}

// resourceVSphereLicenseUpdate check for change in labels of the key and updates them.
// Change in key would remove it from
func resourceVSphereLicenseUpdate(d *schema.ResourceData, meta interface{}) error {

	log.Println("[G] Running the update method")

	client := meta.(*govmomi.Client)
	manager := license.NewManager(client.Client)

	if key, ok := d.GetOk("license_key"); ok {
		keyC := key.(string)
		if !isKeyPresent(keyC, manager) {
			return ErrNoSuchKeyFound
		}

		if d.HasChange("label") {
			mapdata, err := labelsToMap(d.Get("label"))

			if err != nil {
				return err
			}
			_, err = manager.Update(context.TODO(), keyC, mapdata)
			if err != nil {
				return err
			}
		}
	}

	return resourceVSphereLicenseRead(d, meta)
}

func resourceVSphereLicenseDelete(d *schema.ResourceData, meta interface{}) error {

	log.Println("[G] Running the delete method")

	client := meta.(*govmomi.Client)
	manager := license.NewManager(client.Client)

	if key := d.Get("license_key").(string); isKeyPresent(key, manager) {
		d.SetId("")
		err := manager.Remove(context.TODO(), key)
		log.Println("[G] Error removing the key", err)
		return err
	}
	return ErrNoSuchKeyFound

}

// labelsToMap is an adapter method that takes labels and gives a map that
// can be used with the key creation method.
func labelsToMap(labels interface{}) (map[string]string, error) {
	finalLabels := make(map[string]string)
	labelList := labels.([]interface{})
	for _, label := range labelList {
		labelMap := label.(map[string]interface{})
		finalLabels[labelMap["key"].(string)] = finalLabels[labelMap["value"].(string)]
	}
	log.Println("[G]", finalLabels)

	return finalLabels, nil

}
func getLicenseInfoFromKey(key string, manager *license.Manager) *types.LicenseManagerLicenseInfo {

	// TODO: Do we need to handle this error?
	info, _ := manager.Decode(context.TODO(), key)
	return &info

}

// isKeyPresent iterates over the InfoList to check if the license is present or not.
func isKeyPresent(key string, manager *license.Manager) bool {

	// TODO: Manage this error
	infoList, _ := manager.List(context.TODO())

	for _, info := range infoList {
		if info.LicenseKey == key {
			return true
		}
	}

	return false
}
