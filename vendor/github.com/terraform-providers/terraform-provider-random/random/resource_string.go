package random

import (
	"crypto/rand"

	"github.com/hashicorp/terraform/helper/schema"
)

func resourceString() *schema.Resource {
	return &schema.Resource{
		Create: CreateString,
		Read:   ReadString,
		Delete: schema.RemoveFromState,

		Schema: map[string]*schema.Schema{
			"keepers": {
				Type:     schema.TypeMap,
				Optional: true,
				ForceNew: true,
			},

			"length": {
				Type:     schema.TypeInt,
				Required: true,
				ForceNew: true,
			},

			"special": {
				Type:     schema.TypeBool,
				Optional: true,
				Default:  true,
				ForceNew: true,
			},

			"upper": {
				Type:     schema.TypeBool,
				Optional: true,
				Default:  true,
				ForceNew: true,
			},

			"lower": {
				Type:     schema.TypeBool,
				Optional: true,
				Default:  true,
				ForceNew: true,
			},

			"number": {
				Type:     schema.TypeBool,
				Optional: true,
				Default:  true,
				ForceNew: true,
			},

			"override_special": {
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
			},

			"result": {
				Type:     schema.TypeString,
				Computed: true,
			},
		},
	}
}

func CreateString(d *schema.ResourceData, meta interface{}) error {
	const numChars = "0123456789"
	const lowerChars = "abcdefghijklmnopqrstuvwxyz"
	const upperChars = "ABCDEFGHIJKLMNOPQRSTUVWXYZ"
	var specialChars = "!@#$%&*()-_=+[]{}<>:?"

	length := d.Get("length").(int)
	upper := d.Get("upper").(bool)
	lower := d.Get("lower").(bool)
	number := d.Get("number").(bool)
	special := d.Get("special").(bool)
	overrideSpecial := d.Get("override_special").(string)

	if overrideSpecial != "" {
		specialChars = overrideSpecial
	}

	var chars = string("")
	if upper {
		chars += upperChars
	}
	if lower {
		chars += lowerChars
	}
	if number {
		chars += numChars
	}
	if special {
		chars += specialChars
	}

	var bytes = make([]byte, length)
	var l = byte(len(chars))

	rand.Read(bytes)

	for i, b := range bytes {
		bytes[i] = chars[b%l]
	}
	d.Set("result", string(bytes))
	d.SetId(string(bytes))
	return nil
}

func ReadString(d *schema.ResourceData, meta interface{}) error {
	return nil
}
