package tailscale

import (
	"context"
	"fmt"
	"time"

	"github.com/hashicorp/go-cty/cty"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"

	"github.com/tailscale/tailscale-client-go/tailscale"
)

func dataSourceDevice() *schema.Resource {
	return &schema.Resource{
		Description: "The device data source describes a single device in a tailnet",
		ReadContext: readWithWaitFor(dataSourceDeviceRead),
		Schema: map[string]*schema.Schema{
			"name": {
				Type:         schema.TypeString,
				Description:  "The full name of the device (e.g. `hostname.domain.ts.net`)",
				Optional:     true,
				ExactlyOneOf: []string{"name", "hostname"},
			},
			"hostname": {
				Type:         schema.TypeString,
				Description:  "The short hostname of the device",
				Optional:     true,
				ExactlyOneOf: []string{"name", "hostname"},
			},
			"user": {
				Type:        schema.TypeString,
				Description: "The user associated with the device",
				Computed:    true,
			},
			"addresses": {
				Type:        schema.TypeList,
				Description: "The list of device's IPs",
				Computed:    true,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
			},
			"tags": {
				Type:        schema.TypeSet,
				Description: "The tags applied to the device",
				Computed:    true,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
			},
			"wait_for": {
				Type:        schema.TypeString,
				Description: "If specified, the provider will make multiple attempts to obtain the data source until the wait_for duration is reached. Retries are made every second so this value should be greater than 1s",
				Optional:    true,
				ValidateDiagFunc: func(i interface{}, path cty.Path) diag.Diagnostics {
					waitFor, err := time.ParseDuration(i.(string))
					switch {
					case err != nil:
						return diagnosticsErrorWithPath(err, "failed to parse wait_for", path)
					case waitFor <= time.Second:
						return diagnosticsErrorWithPath(nil, "wait_for must be greater than 1 second", path)
					default:
						return nil
					}
				},
			},
		},
	}
}

func dataSourceDeviceRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	client := m.(*Clients).V1

	var filter func(d tailscale.Device) bool
	var filterDesc string

	if name, ok := d.GetOk("name"); ok {
		filter = func(d tailscale.Device) bool {
			return d.Name == name.(string)
		}
		filterDesc = fmt.Sprintf("name=%q", name.(string))
	}

	if hostname, ok := d.GetOk("hostname"); ok {
		filter = func(d tailscale.Device) bool {
			return d.Hostname == hostname.(string)
		}
		filterDesc = fmt.Sprintf("hostname=%q", hostname.(string))
	}

	devices, err := client.Devices(ctx)
	if err != nil {
		return diagnosticsError(err, "Failed to fetch devices")
	}

	var selected *tailscale.Device
	for _, device := range devices {
		if filter(device) {
			selected = &device
			break
		}
	}

	if selected == nil {
		return diag.Errorf("Could not find device with %s", filterDesc)
	}

	d.SetId(selected.ID)

	if err = d.Set("name", selected.Name); err != nil {
		return diagnosticsError(err, "Failed to set name")
	}

	if err = d.Set("hostname", selected.Hostname); err != nil {
		return diagnosticsError(err, "Failed to set hostname")
	}

	if err = d.Set("user", selected.User); err != nil {
		return diagnosticsError(err, "Failed to set user")
	}

	if err = d.Set("addresses", selected.Addresses); err != nil {
		return diagnosticsError(err, "Failed to set addresses")
	}

	if err = d.Set("tags", selected.Tags); err != nil {
		return diagnosticsError(err, "Failed to set tags")
	}

	return nil
}
