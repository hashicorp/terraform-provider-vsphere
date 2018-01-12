package random

import (
	"fmt"
	"strconv"

	"github.com/hashicorp/terraform/helper/schema"
)

func resourceInteger() *schema.Resource {
	return &schema.Resource{
		Create: CreateInteger,
		Read:   RepopulateInteger,
		Delete: schema.RemoveFromState,

		Schema: map[string]*schema.Schema{
			"keepers": {
				Type:     schema.TypeMap,
				Optional: true,
				ForceNew: true,
			},

			"min": {
				Type:     schema.TypeInt,
				Required: true,
				ForceNew: true,
			},

			"max": {
				Type:     schema.TypeInt,
				Required: true,
				ForceNew: true,
			},

			"seed": {
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
			},

			"result": {
				Type:     schema.TypeInt,
				Computed: true,
			},
		},
	}
}

func CreateInteger(d *schema.ResourceData, meta interface{}) error {
	min := d.Get("min").(int)
	max := d.Get("max").(int)
	seed := d.Get("seed").(string)

	if max <= min {
		return fmt.Errorf("Minimum value needs to be smaller than maximum value")
	}
	rand := NewRand(seed)
	number := rand.Intn((max+1)-min) + min

	d.Set("result", number)
	d.SetId(strconv.Itoa(number))

	return nil
}

func RepopulateInteger(d *schema.ResourceData, _ interface{}) error {
	return nil
}
