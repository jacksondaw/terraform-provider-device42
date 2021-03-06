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

type customField struct {
	Key   string      `json:"key"`
	Notes interface{} `json:"notes"`
	Value interface{} `json:"value"`
}

type apiDeviceReadResponse struct {
	Aliases                 []interface{} `json:"aliases"`
	AssetNo                 string        `json:"asset_no"`
	Category                string        `json:"category"`
	Cpucore                 interface{}   `json:"cpucore"`
	Cpucount                interface{}   `json:"cpucount"`
	Cpuspeed                interface{}   `json:"cpuspeed"`
	CustomFields            []customField `json:"custom_fields"`
	Customer                interface{}   `json:"customer"`
	Datastores              []interface{} `json:"datastores"`
	DeviceExternalLinks     []interface{} `json:"device_external_links"`
	DeviceID                int64         `json:"device_id"`
	DevicePurchaseLineItems []interface{} `json:"device_purchase_line_items"`
	HddDetails              interface{}   `json:"hdd_details"`
	Hddcount                interface{}   `json:"hddcount"`
	Hddraid                 interface{}   `json:"hddraid"`
	HddraidType             interface{}   `json:"hddraid_type"`
	Hddsize                 interface{}   `json:"hddsize"`
	HwDepth                 interface{}   `json:"hw_depth"`
	HwModel                 interface{}   `json:"hw_model"`
	HwModelID               interface{}   `json:"hw_model_id"`
	HwSize                  interface{}   `json:"hw_size"`
	ID                      int64         `json:"id"`
	InService               bool          `json:"in_service"`
	IPAddresses             []interface{} `json:"ip_addresses"`
	IsItBladeHost           string        `json:"is_it_blade_host"`
	IsItSwitch              string        `json:"is_it_switch"`
	IsItVirtualHost         string        `json:"is_it_virtual_host"`
	LastUpdated             string        `json:"last_updated"`
	MacAddresses            []interface{} `json:"mac_addresses"`
	Manufacturer            interface{}   `json:"manufacturer"`
	Name                    string        `json:"name"`
	Nonauthoritativealiases []interface{} `json:"nonauthoritativealiases"`
	Notes                   string        `json:"notes"`
	Os                      interface{}   `json:"os"`
	RAM                     interface{}   `json:"ram"`
	SerialNo                string        `json:"serial_no"`
	ServiceLevel            string        `json:"service_level"`
	Tags                    []interface{} `json:"tags"`
	Type                    string        `json:"type"`
	UcsManager              interface{}   `json:"ucs_manager"`
	UUID                    string        `json:"uuid"`
	VirtualHostName         interface{}   `json:"virtual_host_name"`
	VirtualSubtype          string        `json:"virtual_subtype"`
}

type apiResponse struct {
	Code int64         `json:"code"`
	Msg  []interface{} `json:"msg"`
}

func resourceDevice42Device() *schema.Resource {
	return &schema.Resource{
		Create: resourceDevice42DeviceCreate,
		Read:   resourceDevice42DeviceRead,
		Update: resourceDevice42DeviceUpdate,
		Delete: resourceDevice42DeviceDelete,

		Schema: map[string]*schema.Schema{
			"name": &schema.Schema{
				Type:        schema.TypeString,
				ForceNew:    true,
				Required:    true,
				Description: "The hostname of the device.",
			},
			"device_type": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				Default:  "virtual",
				Description: "The type of the device. " +
					"Valid values are 'physical', 'virtual', 'blade', 'cluster', or 'other'.",
				ValidateFunc: validation.StringInSlice([]string{
					"physical", "virtual", "blade", "cluster", "other",
				}, false),
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
func resourceDevice42DeviceCreate(d *schema.ResourceData, m interface{}) error {
	client := m.(*resty.Client)
	name := d.Get("name").(string)
	deviceType := d.Get("device_type").(string)

	resp, err := client.R().
		SetFormData(map[string]string{
			"name": name,
			"type": deviceType,
		}).
		SetResult(apiResponse{}).
		Post("/device/")

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
		bulkFields := []string{}

		for k, v := range fields {
			bulkFields = append(bulkFields, fmt.Sprintf("%v:%v", k, v))
		}

		resp, err := client.R().
			SetFormData(map[string]string{
				"name":        name,
				"bulk_fields": strings.Join(bulkFields, ","),
			}).
			SetResult(apiResponse{}).
			Put("/1.0/device/custom_field/")

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

func resourceDevice42DeviceRead(d *schema.ResourceData, m interface{}) error {
	client := m.(*resty.Client)
	resp, err := client.R().
		SetResult(apiDeviceReadResponse{}).
		Get(fmt.Sprintf("/1.0/devices/id/%s/", d.Id()))

	if err != nil {
		log.Printf("[WARN] No device found: %s", d.Id())
		d.SetId("")
		return nil
	}

	r := resp.Result().(*apiDeviceReadResponse)
	fields := flattenCustomFields(r.CustomFields)

	d.Set("id", r.ID)
	d.Set("name", r.Name)
	d.Set("device_type", r.Type)
	d.Set("custom_fields", fields)

	return nil
}

func resourceDevice42DeviceUpdate(d *schema.ResourceData, m interface{}) error {
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

func resourceDevice42DeviceDelete(d *schema.ResourceData, m interface{}) error {
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

func setCustomFields(d *schema.ResourceData) map[string]interface{} {
	updatedFields := make(map[string]interface{})
	if d.HasChange("custom_fields") {
		oldRaw, newRaw := d.GetChange("custom_fields")
		old := oldRaw.(map[string]interface{})
		new := newRaw.(map[string]interface{})
		for k, v := range new {
			if old[k] != v {
				log.Printf("[DEBUG] Change to custom field: %s, Old Value: '%s', New Value: '%s'", k, old[k], v)
				updatedFields[k] = v
			}
		}
	}
	return updatedFields
}

func flattenCustomFields(in []customField) map[string]interface{} {
	out := make(map[string]interface{}, len(in))
	for _, x := range in {
		out[x.Key] = x.Value
	}
	return out
}

func suppressCustomFieldsDiffs(k, old, new string, d *schema.ResourceData) bool {
	field := strings.TrimPrefix(k, "custom_fields.")
	setFields := d.Get("custom_fields").(map[string]interface{})
	if _, ok := setFields[field]; ok {
		return false
	}
	return true
}
