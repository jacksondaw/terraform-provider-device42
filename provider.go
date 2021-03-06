package main

import (
	"crypto/tls"
	"fmt"

	"github.com/go-resty/resty/v2"
	"github.com/hashicorp/terraform/helper/schema"
)

// Environment variables the provider recognizes for configuration
const (
	// Environment variable to configure the device42 api host
	HostEnv string = "D42_HOST"
	// Environment variable to configure the device42 api username attribute
	UsernameEnv string = "D42_USER"
	// Environment variable to configure the device42 api password attribute
	PasswordEnv string = "D42_PASS"
)

// Provider -- main device42 provider structure
func Provider() *schema.Provider {
	return &schema.Provider{
		Schema: map[string]*schema.Schema{
			// -- API Interaction Definitions --
			"host": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				DefaultFunc: schema.EnvDefaultFunc(
					HostEnv,
					"",
				),
				Description: "The device42 server to interact with.",
			},
			"password": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				DefaultFunc: schema.EnvDefaultFunc(
					PasswordEnv,
					"",
				),
				Description: "The password to authenticate with Device42.",
			},
			"username": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				DefaultFunc: schema.EnvDefaultFunc(
					UsernameEnv,
					"",
				),
				Description: "The username to authenticate with Device42.",
			},
			"client_tls_insecure": &schema.Schema{
				Type:     schema.TypeBool,
				Optional: true,
				Default:  false,
				Description: "Whether to perform TLS cert verification on the server's certificate. " +
					"Defaults to `false`.",
			},
		},
		ResourcesMap: map[string]*schema.Resource{
			"device42_device": resourceDevice42Device(),
		},
		ConfigureFunc: providerConfigure,
	}
}

func providerConfigure(d *schema.ResourceData) (interface{}, error) {
	host := d.Get("host").(string)
	username := d.Get("username").(string)
	password := d.Get("password").(string)
	tlsInsecure := d.Get("client_tls_insecure").(bool)

	if host == "" {
		return nil, fmt.Errorf("no Device42 host was provided")
	}

	client := resty.New()
	client.SetTLSClientConfig(&tls.Config{InsecureSkipVerify: tlsInsecure})
	client.SetHostURL(fmt.Sprintf("https://%s/api", host))
	client.SetBasicAuth(username, password)

	return client, nil
}
