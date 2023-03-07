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

	"github.com/tailscale/tailscale-client-go/tailscale"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/clientcredentials"
)

type ProviderOption func(p *schema.Provider)

// Provider returns the *schema.Provider instance that implements the terraform provider.
func Provider(options ...ProviderOption) *schema.Provider {
	provider := &schema.Provider{
		ConfigureContextFunc: providerConfigure,
		Schema: map[string]*schema.Schema{
			"api_key": {
				Type:        schema.TypeString,
				DefaultFunc: schema.EnvDefaultFunc("TAILSCALE_API_KEY", ""),
				Optional:    true,
				Description: "The API key to use for authenticating requests to the API. Can be set via the TAILSCALE_API_KEY environment variable.",
				Sensitive:   true,
			},
			"oauth_client_id": {
				Type:        schema.TypeString,
				DefaultFunc: schema.EnvDefaultFunc("OAUTH_CLIENT_ID", ""),
				Optional:    true,
				Description: "The OAuth application's ID when using OAuth client credentials. Can be set via the OAUTH_CLIENT_ID environment variable.",
			},
			"oauth_client_secret": {
				Type:        schema.TypeString,
				DefaultFunc: schema.EnvDefaultFunc("OAUTH_CLIENT_SECRET", ""),
				Optional:    true,
				Description: "The OAuth application's secret when using OAuth client credentials. Can be set via the OAUTH_CLIENT_SECRET environment variable.",
				Sensitive:   true,
			},
			"oauth_token_url": {
				Type:        schema.TypeString,
				DefaultFunc: schema.EnvDefaultFunc("OAUTH_TOKEN_URL", "https://api.tailscale.com/api/v2/oauth/token"),
				Optional:    true,
				Description: "TokenURL is the resource server's token endpoint URL. Can be set via the OAUTH_TOKEN_URL environment variable.",
			},
			"tailnet": {
				Type:        schema.TypeString,
				DefaultFunc: schema.EnvDefaultFunc("TAILSCALE_TAILNET", ""),
				Optional:    true,
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
			"tailscale_4via6":   dataSource4Via6(),
		},
	}

	for _, option := range options {
		option(provider)
	}

	return provider
}

func providerConfigure(ctx context.Context, d *schema.ResourceData) (interface{}, diag.Diagnostics) {
	apiKey := d.Get("api_key").(string)
	oauthClientID := d.Get("oauth_client_id").(string)
	oauthClientSecret := d.Get("oauth_client_secret").(string)

	if apiKey == "" && oauthClientID == "" && oauthClientSecret == "" {
		return nil, diag.Errorf("tailscale provider credentials are empty - set `api_key` or 'oauth_client_id' and 'oauth_client_secret'")
	} else if apiKey != "" && (oauthClientID != "" || oauthClientSecret != "") {
		return nil, diag.Errorf("tailscale provider credentials are conflicting - set `api_key` or 'oauth_client_id' and 'oauth_client_secret'")
	} else if oauthClientID == "" {
		return nil, diag.Errorf("tailscale provider argument 'oauth_client_id' is empty")
	} else if oauthClientSecret == "" {
		return nil, diag.Errorf("tailscale provider argument 'oauth_client_secret' is empty")
	}

	tailnet := d.Get("tailnet").(string)
	if tailnet == "" {
		return nil, diag.Errorf("tailscale provider argument 'tailnet' is empty")
	}

	if apiKey == "" {
		oauthTokenURL := d.Get("oauth_token_url").(string)
		oauthToken, err := retrieveOAuthToken(ctx, oauthClientID, oauthClientSecret, oauthTokenURL, tailnet)
		if err != nil {
			return nil, diagnosticsError(err, "failed to retrieve api key using OAuth credentials")
		}
		apiKey = oauthToken.AccessToken
	}

	baseURL := d.Get("base_url").(string)

	client, err := tailscale.NewClient(apiKey, tailnet, tailscale.WithBaseURL(baseURL))
	if err != nil {
		return nil, diagnosticsError(err, "failed to initialise client")
	}

	return client, nil
}

func retrieveOAuthToken(ctx context.Context, oauthClientID string, oauthClientSecret string, oauthTokenURL string, tailnet string) (*oauth2.Token, error) {
	var oauthConfig = &clientcredentials.Config{
		ClientID:     oauthClientID,
		ClientSecret: oauthClientSecret,
		TokenURL:     oauthTokenURL,
	}

	tokenResponse, err := oauthConfig.Token(context.Background())
	if err != nil {
		return nil, fmt.Errorf("error getting token: %w", err)
	}

	return tokenResponse, nil
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
