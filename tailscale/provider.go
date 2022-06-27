// Package tailscale describes the resources and data sources provided by the terraform provider. Each resource
// or data source is described within its own file.
package tailscale

import (
	"context"
	"fmt"
	"time"

	"github.com/hashicorp/go-cty/cty"
	"github.com/hashicorp/go-uuid"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"

	"github.com/davidsbond/tailscale-client-go/tailscale"
)

type ProviderOption func(p *schema.Provider)

// Provider returns the *schema.Provider instance that implements the terraform provider.
func Provider(options ...ProviderOption) *schema.Provider {
	provider := &schema.Provider{
		ConfigureContextFunc: providerConfigure,
		Schema: map[string]*schema.Schema{
			"api_key": {
				Type:        schema.TypeString,
				DefaultFunc: schema.EnvDefaultFunc("TAILSCALE_API_KEY", nil),
				Required:    true,
				Description: "The API key to use for authenticating requests to the API. Can be set via the TAILSCALE_API_KEY environment variable.",
				Sensitive:   true,
			},
			"tailnet": {
				Type:        schema.TypeString,
				DefaultFunc: schema.EnvDefaultFunc("TAILSCALE_TAILNET", nil),
				Required:    true,
				Description: "The Tailnet to perform actions in. Can be set via the TAILSCALE_TAILNET environment variable.",
			},
			"base_url": {
				Type:        schema.TypeString,
				DefaultFunc: schema.EnvDefaultFunc("TAILSCALE_BASE_URL", "https://api.tailscale.com"),
				Optional:    true,
				Description: "The base URL of the Tailscale API. Defaults to https://api.tailscale.com. Can be set via the TAILSCALE_BASE_URL environment variable.",
			},
		},
		ResourcesMap: map[string]*schema.Resource{
			"tailscale_acl":                  resourceACL(),
			"tailscale_dns_nameservers":      resourceDNSNameservers(),
			"tailscale_dns_preferences":      resourceDNSPreferences(),
			"tailscale_dns_search_paths":     resourceDNSSearchPaths(),
			"tailscale_device_subnet_routes": resourceDeviceSubnetRoutes(),
			"tailscale_device_authorization": resourceDeviceAuthorization(),
			"tailscale_tailnet_key":          resourceTailnetKey(),
			"tailscale_device_tags":          resourceDeviceTags(),
			"tailscale_device_key":           resourceDeviceKey(),
		},
		DataSourcesMap: map[string]*schema.Resource{
			"tailscale_device":  dataSourceDevice(),
			"tailscale_devices": dataSourceDevices(),
		},
	}

	for _, option := range options {
		option(provider)
	}

	return provider
}

func providerConfigure(_ context.Context, d *schema.ResourceData) (interface{}, diag.Diagnostics) {
	apiKey := d.Get("api_key").(string)
	tailnet := d.Get("tailnet").(string)
	baseURL := d.Get("base_url").(string)

	client, err := tailscale.NewClient(apiKey, tailnet, tailscale.WithBaseURL(baseURL))
	if err != nil {
		return nil, diagnosticsError(err, "failed to initialise client")
	}

	return client, nil
}

func diagnosticsError(err error, message string, args ...interface{}) diag.Diagnostics {
	var detail string
	if err != nil {
		detail = err.Error()
	}

	diags := []diag.Diagnostic{
		{
			Severity: diag.Error,
			Summary:  fmt.Sprintf(message, args...),
			Detail:   detail,
		},
	}

	if details := tailscale.ErrorData(err); len(details) > 0 {
		for _, dt := range details {
			for _, e := range dt.Errors {
				diags = append(diags, diag.Diagnostic{
					Severity: diag.Error,
					Summary:  fmt.Sprintf("user: %s\nerror: %s", dt.User, e),
				})
			}
		}
	}

	return diags
}

func diagnosticsErrorWithPath(err error, message string, path cty.Path, args ...interface{}) diag.Diagnostics {
	d := diagnosticsError(err, message, args...)
	for i := range d {
		d[i].AttributePath = path
	}

	return d
}

func createUUID() string {
	val, err := uuid.GenerateUUID()
	if err != nil {
		panic(err)
	}
	return val
}

func readWithWaitFor(fn schema.ReadContextFunc) schema.ReadContextFunc {
	return func(ctx context.Context, data *schema.ResourceData, i interface{}) diag.Diagnostics {
		var d diag.Diagnostics

		// Do an initial check in case we don't need to wait at all.
		d = fn(ctx, data, i)
		if !d.HasError() {
			return d
		}

		waitFor := data.Get("wait_for").(string)
		if waitFor == "" {
			return fn(ctx, data, i)
		}

		dur, err := time.ParseDuration(waitFor)
		if err != nil {
			return diagnosticsError(err, "failed to parse wait_for")
		}

		maxTicker := time.NewTicker(dur)
		defer maxTicker.Stop()

		intervalTicker := time.NewTicker(time.Second)
		defer intervalTicker.Stop()

		// Check every second for the data, until we reach the maximum specified duration.
		for {
			select {
			case <-ctx.Done():
				return diag.FromErr(ctx.Err())
			case <-maxTicker.C:
				return d
			case <-intervalTicker.C:
				d = fn(ctx, data, i)
				if d.HasError() {
					continue
				}

				return d
			}
		}
	}
}

func importWithDeviceIDFromName(ctx context.Context, d *schema.ResourceData, m interface{}) ([]*schema.ResourceData, error) {
	client := m.(*tailscale.Client)

	devices, err := client.Devices(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch devices: %w", err)
	}

	var selected *tailscale.Device
	for _, device := range devices {
		if device.Name != d.Id() {
			continue
		}

		selected = &device
		break
	}

	if selected == nil {
		return nil, fmt.Errorf("could not find device with name %s", d.Id())
	}

	if err = d.Set("device_id", selected.ID); err != nil {
		return nil, err
	}

	return []*schema.ResourceData{d}, nil

}
