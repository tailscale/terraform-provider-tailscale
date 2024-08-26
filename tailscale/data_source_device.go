package tailscale

import (
	"context"
	"fmt"
	"time"

	"github.com/hashicorp/go-cty/cty"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"

	tsclient "github.com/tailscale/tailscale-client-go/v2"
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
	client := m.(*tsclient.Client)

	var filter func(d tsclient.Device) bool
	var filterDesc string

	if name, ok := d.GetOk("name"); ok {
		filter = func(d tsclient.Device) bool {
			return d.Name == name.(string)
		}
		filterDesc = fmt.Sprintf("name=%q", name.(string))
	}

	if hostname, ok := d.GetOk("hostname"); ok {
		filter = func(d tsclient.Device) bool {
			return d.Hostname == hostname.(string)
		}
		filterDesc = fmt.Sprintf("hostname=%q", hostname.(string))
	}

	devices, err := client.Devices().List(ctx)
	if err != nil {
		return diagnosticsError(err, "Failed to fetch devices")
	}

	var selected *tsclient.Device
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
	return setProperties(d, deviceToMap(selected))
}

// deviceToMap converts the given device into a map representing the device as a
// resource in Terraform. This omits the "id" which is expected to be set
// using [schema.ResourceData.SetId].
func deviceToMap(device *tsclient.Device) map[string]any {
	return map[string]any{
		"name":      device.Name,
		"hostname":  device.Hostname,
		"user":      device.User,
		"addresses": device.Addresses,
		"tags":      device.Tags,
	}
}
