package vsphere

import (
	"errors"
	"fmt"
	"log"

	"context"

	"github.com/hashicorp/terraform/helper/schema"
	"github.com/vmware/govmomi"
	"github.com/vmware/govmomi/license"
	"github.com/vmware/govmomi/vim25/methods"
	"github.com/vmware/govmomi/vim25/types"
)

var (
	// ErrNoSuchKeyFound is an error primarily thrown by the Read method of the resource.
	// The error doesn't display the key itself for security reasons.
	ErrNoSuchKeyFound = errors.New("The key was not found")
	// ErrKeyCannotBeDeleted is an error which occurs when a key that is used by VMs is
	// being removed
	ErrKeyCannotBeDeleted = errors.New("The key wasn't deleted")
)

func resourceVSphereLicense() *schema.Resource {

	return &schema.Resource{

		SchemaVersion: 1,

		Create: resourceVSphereLicenseCreate,
		Read:   resourceVSphereLicenseRead,
		Update: resourceVSphereLicenseUpdate,
		Delete: resourceVSphereLicenseDelete,

		Schema: map[string]*schema.Schema{
			"license_key": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"label": &schema.Schema{
				Type:     schema.TypeList,
				Optional: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"key": &schema.Schema{
							Type:     schema.TypeString,
							Required: true,
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

	log.Println("[INFO] Running the create method")

	client := meta.(*govmomi.Client)
	manager := license.NewManager(client.Client)

	key := d.Get("license_key").(string)

	log.Println(" [INFO] Reading the key from the resource data")
	var finalLabels map[string]string
	var err error
	if labels, ok := d.GetOk("label"); ok {
		finalLabels, err = labelsToMap(labels)
		if err != nil {
			// If errors are printed by terraform then no need to log errors here
			return err
		}
	}

	var info types.LicenseManagerLicenseInfo

	switch t := client.ServiceContent.About.ApiType; t {
	case "HostAgent":
		info, err = manager.Update(context.TODO(), key, finalLabels)
	case "VirtualCenter":
		info, err = manager.Add(context.TODO(), key, finalLabels)
	default:
		return fmt.Errorf("unsupported ApiType: %s", t)
	}

	if err != nil {
		return err
	}

	if err = DecodeError(info); err != nil {
		return err
	}

	// This can be used in the read method to set the computed parameters
	d.SetId(info.LicenseKey)

	return resourceVSphereLicenseRead(d, meta)
}

func resourceVSphereLicenseRead(d *schema.ResourceData, meta interface{}) error {

	log.Println("[INFO] Running the read method")

	client := meta.(*govmomi.Client)
	manager := license.NewManager(client.Client)

	if info := getLicenseInfoFromKey(d.Get("license_key").(string), manager); info != nil {
		log.Println("[INFO] Setting the values")
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
func resourceVSphereLicenseUpdate(d *schema.ResourceData, meta interface{}) error {

	log.Println("[INFO] Running the update method")

	client := meta.(*govmomi.Client)
	manager := license.NewManager(client.Client)

	if key, ok := d.GetOk("license_key"); ok {
		licenseKey := key.(string)
		if !isKeyPresent(licenseKey, manager) {
			return ErrNoSuchKeyFound
		}

		if d.HasChange("label") {
			labelMap, err := labelsToMap(d.Get("label"))

			if err != nil {
				return err
			}
			for key, value := range labelMap {
				err := UpdateLabel(context.TODO(), manager, licenseKey, key, value)
				if err != nil {
					return err
				}
			}
			if err != nil {
				return err
			}
		}
	}

	return resourceVSphereLicenseRead(d, meta)
}

func resourceVSphereLicenseDelete(d *schema.ResourceData, meta interface{}) error {

	log.Println("[INFO] Running the delete method")

	client := meta.(*govmomi.Client)
	manager := license.NewManager(client.Client)

	if key := d.Get("license_key").(string); isKeyPresent(key, manager) {

		err := manager.Remove(context.TODO(), key)

		if err != nil {
			return err

		}

		// if the key is still present
		if isKeyPresent(key, manager) {
			return ErrKeyCannotBeDeleted
		}
		d.SetId("")
		return nil
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
		finalLabels[labelMap["key"].(string)] = labelMap["value"].(string)
	}

	return finalLabels, nil

}

func getLicenseInfoFromKey(key string, manager *license.Manager) *types.LicenseManagerLicenseInfo {

	// Use of decode is not returning labels so using list instead
	// Issue - https://github.com/vmware/govmomi/issues/797
	infoList, _ := manager.List(context.TODO())
	for _, info := range infoList {
		if info.LicenseKey == key {
			return &info
		}
	}
	return nil

}

// isKeyPresent iterates over the InfoList to check if the license is present or not.
func isKeyPresent(key string, manager *license.Manager) bool {

	infoList, _ := manager.List(context.TODO())

	for _, info := range infoList {
		if info.LicenseKey == key {
			return true
		}
	}

	return false
}

// UpdateLabel provides a wrapper around the UpdateLabel data objects
func UpdateLabel(ctx context.Context, m *license.Manager, licenseKey string, key string, val string) error {

	req := types.UpdateLicenseLabel{
		This:       m.Reference(),
		LicenseKey: licenseKey,
		LabelKey:   key,
		LabelValue: val,
	}

	_, err := methods.UpdateLicenseLabel(ctx, m.Client(), &req)
	return err
}

// DecodeError tries to find a specific error which occurs when an invalid key is passed
// to the server
func DecodeError(info types.LicenseManagerLicenseInfo) error {

	for _, property := range info.Properties {
		if property.Key == "localizedDiagnostic" {
			if message, ok := property.Value.(types.LocalizableMessage); ok {
				if message.Key == "com.vmware.vim.vc.license.error.decode" {
					return errors.New(message.Message)
				}
			}
		}
	}

	return nil

}
