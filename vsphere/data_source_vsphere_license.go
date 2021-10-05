package vsphere

import (
	"log"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/vmware/govmomi/license"
)

func dataSourceVSphereLicense() *schema.Resource {
	return &schema.Resource{
		Read: dataSourceVSphereLicenseRead,

		Schema: map[string]*schema.Schema{
			"license_key": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"labels": {
				Type:     schema.TypeMap,
				Optional: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
			},
			"id": {
				Type:     schema.TypeString,
				Computed: true,
			},

			// computed properties returned by the API
			"edition_key": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"name": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"total": {
				Type:     schema.TypeInt,
				Computed: true,
			},
			"used": {
				Type:     schema.TypeInt,
				Computed: true,
			},
		},
	}
}

func dataSourceVSphereLicenseRead(d *schema.ResourceData, meta interface{}) error {

	client := meta.(*Client).vimClient
	manager := license.NewManager(client.Client)
	licenseKey := d.Get("license_key").(string)
	if info := getLicenseInfoFromKey(d.Get("license_key").(string), manager); info != nil {
		log.Println("[INFO] Setting the values")
		_ = d.Set("edition_key", info.EditionKey)
		_ = d.Set("total", info.Total)
		_ = d.Set("used", info.Used)
		_ = d.Set("name", info.Name)
		_ = d.Set("labels", keyValuesToMap(info.Labels))
		d.SetId(licenseKey)

	} else {
		return ErrNoSuchKeyFound
	}

	return nil
}
