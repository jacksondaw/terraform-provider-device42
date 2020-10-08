package main

import (
	"fmt"
	"log"
	"strconv"
	"strings"

	"github.com/go-resty/resty/v2"
	"github.com/hashicorp/terraform/helper/schema"
	"github.com/hashicorp/terraform/helper/validation"
)

type password struct {
	Id   int64       `json:"id"`
	Password   string       `json:"password"`
	LastPasswordChange   string       `json:"last_pw_change"`
	Notes   string       `json:"notes"`
	Label   string       `json:"label"`
	FirstAdded   string       `json:"first_added"`
	Username   string       `json:"username"`
}

type apiPasswordReadReponse struct {
	Passwords                 []interface{} `json:"passwords"`
}

func resourceDevice42Password() *schema.Resource {
	return &schema.Resource{
		Create: resourceDevice42PasswordCreate,
		Read:   resourceDevice42PasswordRead,
		Update: resourceDevice42PasswordUpdate,
		Delete: resourceDevice42PasswordDelete,

		Schema: map[string]*schema.Schema{
			"username": &schema.Schema{
				Type:        schema.TypeString,
				ForceNew:    false,
				Required:    true,
				Description: "The username associated with the account.",
			},
			"password": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				Description: "Password associated with the account",
			},
			"label": &schema.Schema{
				Type:             schema.TypeString,
				Optional:         true,
				Description:      "Labels associated with the password",
			},
			"category": &schema.Schema{
				Type:             schema.TypeString,
				Optional:         true,
				Description:      "Category associated with the password",
			},
			"device": &schema.Schema{
				Type:             schema.TypeString,
				Optional:         true,
				Description:      "Device associated with the password",
			},
			"appcomp": &schema.Schema{
				Type:             schema.TypeString,
				Optional:         true,
				Description:      "Application Component associated with the password",
			},
			"id": &schema.Schema{
				Type:             schema.TypeInt,
				Optional:         true,
				Description:      "Id of the password",
			},
			"plain_text": &schema.Schema{
				Type:             schema.TypeInt,
				Optional:         true,
				Description:      "Include the Password in Plain Text in the Response valid options are yes or no",
				Default: "yes",
				ValidateFunc: validation.StringInSlice([]string{
					"yes", "no",
				}, false),
			},
			"notes": &schema.Schema{
				Type:             schema.TypeString,
				Optional:         true,
				Description:      "Any notes associated with the password",
			},
			"custom_fields": &schema.Schema{
				Type:             schema.TypeMap,
				Optional:         true,
				Computed:         true,
				Description:      "Any custom fields that will be used in device42.",
				DiffSuppressFunc: suppressCustomFieldsDiffs,
			},
		},
	}
}

// -----------------------------------------------------------------------------
// Resource CRUD Operations
// -----------------------------------------------------------------------------
func resourceDevice42PasswordCreate(d *schema.ResourceData, m interface{}) error {
	client := m.(*resty.Client)
	username := d.Get("username").(string)
	password := d.Get("password").(string)
	label := d.Get("label").(string)
	notes := d.Get("notes").(string)

	resp, err := client.R().
		SetFormData(map[string]string{
			"username": username,
			"password": password,
			"label": label,
			"notes": notes,
		}).
		SetResult(apiResponse{}).
		Post("/1.0/passwords/")

	if err != nil {
		return err
	}

	r := resp.Result().(*apiResponse)

	if r.Code != 0 {
		return fmt.Errorf("API returned code %d", r.Code)
	}

	log.Printf("[DEBUG] Result: %#v", r)
	id := int(r.Msg[1].(float64))

	if d.Get("custom_fields") != nil {
		fields := d.Get("custom_fields").(map[string]interface{})
		key := fields["key"].(string)
		bulkFields := []string{}
		for k, v := range fields {
			bulkFields = append(bulkFields, fmt.Sprintf("%v:%v", k, v))
		}

		resp, err := client.R().
			SetFormData(map[string]string{
				"username":        username,
				"key":        key,
				"bulk_fields": strings.Join(bulkFields, ","),
			}).
			SetResult(apiResponse{}).
			Put("/1.0/custom_fields/password")

		if err != nil {
			return err
		}

		r := resp.Result().(*apiResponse)

		if r.Code != 0 {
			return fmt.Errorf("API returned code %d", r.Code)
		}
	}

	// Only set ID after all conditions are successfull
	d.SetId(strconv.Itoa(id))

	return resourceDevice42DeviceRead(d, m)
}

func resourceDevice42PasswordRead(d *schema.ResourceData, m interface{}) error {
	id := d.Get("id").(string)
	plain_text := d.Get("plain_text").(string)
	client := m.(*resty.Client)
	resp, err := client.R().
		SetResult(apiPasswordReadReponse{}).
		Get(fmt.Sprintf("/1.0/passwords/?plain_text=%s&id=%s", 
			plain_text, id ))

	if err != nil {
		log.Printf("[WARN] No device found: %s", d.Id())
		d.SetId("")
		return nil
	}

	r := resp.Result().(*apiPasswordReadReponse)

	if len(r.Passwords) == 0 {
		return nil
	}

	p := r.Passwords[0].(map[string]string)

	d.SetId(id)
	d.Set("username", p["username"])
	d.Set("category", p["category"])
	d.Set("label", p["label"])
	d.Set("password", p["password"])
	d.Set("device", p["device"])
	d.Set("appcomp", p["appcomp"])
	d.Set("notes", p["notes"])

	return nil
}

func resourceDevice42PasswordUpdate(d *schema.ResourceData, m interface{}) error {
	client := m.(*resty.Client)
	name := d.Get("name").(string)
	if d.HasChange("custom_fields") {
		updateList := setCustomFields(d)
		for k, v := range updateList {
			resp, err := client.R().
				SetFormData(map[string]string{
					"name":  name,
					"key":   k,
					"value": v.(string),
				}).
				SetResult(apiResponse{}).
				Put("/1.0/device/custom_field/")

			if err != nil {
				return err
			}

			r := resp.Result().(*apiResponse)
			log.Printf("[DEBUG] Result: %#v", r)
		}
	}
	return resourceDevice42DeviceRead(d, m)
}

func resourceDevice42PasswordDelete(d *schema.ResourceData, m interface{}) error {
	client := m.(*resty.Client)
	log.Printf("Deleting device %s (UUID: %s)", d.Get("name"), d.Id())

	url := fmt.Sprintf("/1.0/devices/%s/", d.Id())

	resp, err := client.R().
		SetResult(apiResponse{}).
		Delete(url)

	if err != nil {
		return err
	}

	r := resp.Result().(*apiResponse)
	log.Printf("[DEBUG] Result: %#v", r)
	return nil
}