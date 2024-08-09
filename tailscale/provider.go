// Package tailscale describes the resources and data sources provided by the terraform provider. Each resource
// or data source is described within its own file.
package tailscale

import (
	"context"
	"fmt"
	"net/url"
	"time"

	"github.com/hashicorp/go-cty/cty"
	"github.com/hashicorp/go-uuid"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"

	tsclientv1 "github.com/tailscale/tailscale-client-go/tailscale"
	tsclient "github.com/tailscale/tailscale-client-go/v2"
)

// providerVersion is filled by goreleaser at build time.
var providerVersion = "dev"

type ProviderOption func(p *schema.Provider)

// Clients contains both v1 and v2 Tailscale Clients
type Clients struct {
	V1 *tsclientv1.Client
	V2 *tsclient.Client
}

// Provider returns the *schema.Provider instance that implements the terraform provider.
func Provider(options ...ProviderOption) *schema.Provider {
	// Support both sets of OAuth Env vars for backwards compatibility
	oauthClientIDEnvVars := []string{"TAILSCALE_OAUTH_CLIENT_ID", "OAUTH_CLIENT_ID"}
	oauthClientSecretEnvVars := []string{"TAILSCALE_OAUTH_CLIENT_SECRET", "OAUTH_CLIENT_SECRET"}

	provider := &schema.Provider{
		Schema: map[string]*schema.Schema{
			"api_key": {
				Type:        schema.TypeString,
				DefaultFunc: schema.EnvDefaultFunc("TAILSCALE_API_KEY", ""),
				Optional:    true,
				Description: "The API key to use for authenticating requests to the API. Can be set via the TAILSCALE_API_KEY environment variable. Conflicts with 'oauth_client_id' and 'oauth_client_secret'.",
				Sensitive:   true,
			},
			"oauth_client_id": {
				Type:        schema.TypeString,
				DefaultFunc: schema.MultiEnvDefaultFunc(oauthClientIDEnvVars, ""),
				Optional:    true,
				Description: "The OAuth application's ID when using OAuth client credentials. Can be set via the TAILSCALE_OAUTH_CLIENT_ID environment variable. Both 'oauth_client_id' and 'oauth_client_secret' must be set. Conflicts with 'api_key'.",
			},
			"oauth_client_secret": {
				Type:        schema.TypeString,
				DefaultFunc: schema.MultiEnvDefaultFunc(oauthClientSecretEnvVars, ""),
				Optional:    true,
				Description: "The OAuth application's secret when using OAuth client credentials. Can be set via the TAILSCALE_OAUTH_CLIENT_SECRET environment variable. Both 'oauth_client_id' and 'oauth_client_secret' must be set. Conflicts with 'api_key'.",
				Sensitive:   true,
			},
			"scopes": {
				Type:        schema.TypeList,
				Optional:    true,
				Elem:        &schema.Schema{Type: schema.TypeString},
				Description: "The OAuth 2.0 scopes to request when for the access token generated using the supplied OAuth client credentials. See https://tailscale.com/kb/1215/oauth-clients/#scopes for available scopes. Only valid when both 'oauth_client_id' and 'oauth_client_secret' are set.",
			},
			"tailnet": {
				Type:        schema.TypeString,
				DefaultFunc: schema.EnvDefaultFunc("TAILSCALE_TAILNET", "-"),
				Optional:    true,
				Description: "The organization name of the Tailnet in which to perform actions. Can be set via the TAILSCALE_TAILNET environment variable. Default is the tailnet that owns API credentials passed to the provider.",
			},
			"base_url": {
				Type:        schema.TypeString,
				DefaultFunc: schema.EnvDefaultFunc("TAILSCALE_BASE_URL", "https://api.tailscale.com"),
				Optional:    true,
				Description: "The base URL of the Tailscale API. Defaults to https://api.tailscale.com. Can be set via the TAILSCALE_BASE_URL environment variable.",
			},
			"user_agent": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "User-Agent header for API requests.",
			},
		},
		ResourcesMap: map[string]*schema.Resource{
			"tailscale_acl":                   resourceACL(),
			"tailscale_dns_nameservers":       resourceDNSNameservers(),
			"tailscale_dns_preferences":       resourceDNSPreferences(),
			"tailscale_dns_search_paths":      resourceDNSSearchPaths(),
			"tailscale_dns_split_nameservers": resourceDNSSplitNameservers(),
			"tailscale_device_subnet_routes":  resourceDeviceSubnetRoutes(),
			"tailscale_device_authorization":  resourceDeviceAuthorization(),
			"tailscale_tailnet_key":           resourceTailnetKey(),
			"tailscale_device_tags":           resourceDeviceTags(),
			"tailscale_device_key":            resourceDeviceKey(),
			"tailscale_webhook":               resourceWebhook(),
			"tailscale_contacts":              resourceContacts(),
			"tailscale_posture_integration":   resourcePostureIntegration(),
		},
		DataSourcesMap: map[string]*schema.Resource{
			"tailscale_device":  dataSourceDevice(),
			"tailscale_devices": dataSourceDevices(),
			"tailscale_4via6":   dataSource4Via6(),
			"tailscale_acl":     dataSourceACL(),
		},
	}

	provider.ConfigureContextFunc = func(ctx context.Context, d *schema.ResourceData) (interface{}, diag.Diagnostics) {
		return providerConfigure(ctx, provider, d)
	}

	for _, option := range options {
		option(provider)
	}

	return provider
}

func providerConfigure(_ context.Context, provider *schema.Provider, d *schema.ResourceData) (interface{}, diag.Diagnostics) {
	baseURL := d.Get("base_url").(string)
	parsedBaseURL, err := url.Parse(baseURL)
	if err != nil {
		return nil, diag.Errorf("could not parse baseURL %q: %s", baseURL, err)
	}

	tailnet := d.Get("tailnet").(string)
	if tailnet == "" {
		return nil, diag.Errorf("tailscale provider argument 'tailnet' is empty")
	}

	apiKey := d.Get("api_key").(string)
	oauthClientID := d.Get("oauth_client_id").(string)
	oauthClientSecret := d.Get("oauth_client_secret").(string)

	if apiKey == "" && oauthClientID == "" && oauthClientSecret == "" {
		return nil, diag.Errorf("tailscale provider credentials are empty - set `api_key` or 'oauth_client_id' and 'oauth_client_secret'")
	} else if apiKey != "" && (oauthClientID != "" || oauthClientSecret != "") {
		return nil, diag.Errorf("tailscale provider credentials are conflicting - `api_key` conflicts with 'oauth_client_id' and 'oauth_client_secret'")
	} else if apiKey == "" && oauthClientID == "" && oauthClientSecret != "" {
		return nil, diag.Errorf("tailscale provider argument 'oauth_client_id' is empty")
	} else if apiKey == "" && oauthClientID != "" && oauthClientSecret == "" {
		return nil, diag.Errorf("tailscale provider argument 'oauth_client_secret' is empty")
	}

	userAgent := d.Get("user_agent").(string)
	if userAgent == "" {
		userAgent = provider.UserAgent("terraform-provider-tailscale", providerVersion)
	}

	if oauthClientID != "" && oauthClientSecret != "" {
		var oauthScopes []string
		oauthScopesFromConfig := d.Get("scopes").([]interface{})
		if len(oauthScopesFromConfig) > 0 {
			oauthScopes = make([]string, len(oauthScopesFromConfig))
		}
		for i, scope := range oauthScopesFromConfig {
			oauthScopes[i] = scope.(string)
		}

		client, err := tsclientv1.NewClient(
			"",
			tailnet,
			tsclientv1.WithBaseURL(baseURL),
			tsclientv1.WithUserAgent(userAgent),
			tsclientv1.WithOAuthClientCredentials(oauthClientID, oauthClientSecret, oauthScopes),
		)
		if err != nil {
			return nil, diagnosticsError(err, "failed to initialise client")
		}

		clientV2 := &tsclient.Client{
			BaseURL:   parsedBaseURL,
			UserAgent: userAgent,
			Tailnet:   tailnet,
		}
		clientV2.UseOAuth(oauthClientID, oauthClientSecret, oauthScopes)

		return &Clients{client, clientV2}, nil
	}

	client, err := tsclientv1.NewClient(
		apiKey,
		tailnet,
		tsclientv1.WithBaseURL(baseURL),
		tsclientv1.WithUserAgent(userAgent),
	)
	if err != nil {
		return nil, diagnosticsError(err, "failed to initialise client")
	}

	clientV2 := &tsclient.Client{
		BaseURL:   parsedBaseURL,
		UserAgent: userAgent,
		APIKey:    apiKey,
		Tailnet:   tailnet,
	}

	return &Clients{client, clientV2}, nil
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

	if details := tsclientv1.ErrorData(err); len(details) > 0 {
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
