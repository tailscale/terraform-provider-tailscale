// Copyright (c) David Bond, Tailscale Inc, & Contributors
// SPDX-License-Identifier: MIT

// Package tailscale describes the resources and data sources provided by the terraform provider. Each resource
// or data source is described within its own file.
package tailscale

import (
	"context"
	"errors"
	"fmt"
	"maps"
	"net/url"
	"os"
	"time"

	"github.com/hashicorp/go-cty/cty"
	"github.com/hashicorp/go-uuid"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	schemav2 "github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"

	"tailscale.com/client/tailscale/v2"
)

// providerVersion is filled by goreleaser at build time.
var providerVersion = "dev"

type ProviderOption func(p *schema.Provider)

type tailscaleProvider struct {
	Client tailscale.Client
}

func New() provider.Provider {
	return &tailscaleProvider{}
}

// Metadata defines information about the provider itself.
func (p *tailscaleProvider) Metadata(ctx context.Context, req provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "tailscale"
	resp.Version = providerVersion
}

// Schema defines a [schemav2.Schema] describing what data is available in the provider's
// configuration.
func (p *tailscaleProvider) Schema(ctx context.Context, req provider.SchemaRequest, resp *provider.SchemaResponse) {
	resp.Schema = schemav2.Schema{
		Attributes: map[string]schemav2.Attribute{
			"api_key": schemav2.StringAttribute{
				Optional:    true,
				Description: "The API key to use for authenticating requests to the API. Can be set via the TAILSCALE_API_KEY environment variable. Conflicts with 'oauth_client_id' and 'oauth_client_secret'.",
				Sensitive:   true,
			},
			"identity_token": schemav2.StringAttribute{
				Optional:    true,
				Description: "The jwt identity token to exchange for a Tailscale API token when using a federated identity. Can be set via the TAILSCALE_IDENTITY_TOKEN environment variable. Conflicts with 'api_key' and 'oauth_client_secret'.",
				Sensitive:   true,
			},
			"oauth_client_id": schemav2.StringAttribute{
				Optional:    true,
				Description: "The OAuth application or federated identity's ID when using OAuth client credentials or workload identity federation. Can be set via the TAILSCALE_OAUTH_CLIENT_ID environment variable. Either 'oauth_client_secret' or 'identity_token' must be set alongside 'oauth_client_id'. Conflicts with 'api_key'.",
			},
			"oauth_client_secret": schemav2.StringAttribute{
				Optional:    true,
				Description: "The OAuth application's secret when using OAuth client credentials. Can be set via the TAILSCALE_OAUTH_CLIENT_SECRET environment variable. Conflicts with 'api_key' and 'identity_token'.",
				Sensitive:   true,
			},
			"scopes": schemav2.ListAttribute{
				ElementType: types.StringType,
				Optional:    true,
				Description: "The OAuth 2.0 scopes to request when generating the access token using the supplied OAuth client credentials. See https://tailscale.com/kb/1623/trust-credentials#scopes for available scopes. Only valid when both 'oauth_client_id' and 'oauth_client_secret', or both are set.",
			},
			"tailnet": schemav2.StringAttribute{
				Optional:    true,
				Description: "The tailnet ID. Tailnets created before Oct 2025 can still use the legacy ID, but the Tailnet ID is the preferred identifier. Can be set via the TAILSCALE_TAILNET environment variable. Default is the tailnet that owns API credentials passed to the provider.",
			},
			"base_url": schemav2.StringAttribute{
				Optional:    true,
				Description: "The base URL of the Tailscale API. Defaults to https://api.tailscale.com. Can be set via the TAILSCALE_BASE_URL environment variable.",
			},
			"user_agent": schemav2.StringAttribute{
				Optional:    true,
				Description: "User-Agent header for API requests.",
			},
		},
	}
}

type tailscaleProviderModel struct {
	APIKey            types.String `tfsdk:"api_key"`
	IdentityToken     types.String `tfsdk:"identity_token"`
	OAuthClientID     types.String `tfsdk:"oauth_client_id"`
	OAuthClientSecret types.String `tfsdk:"oauth_client_secret"`
	Tailnet           types.String `tfsdk:"tailnet"`
	BaseURL           types.String `tfsdk:"base_url"`
	UserAgent         types.String `tfsdk:"user_agent"`
	Scopes            types.List   `tfsdk:"scopes"`
}

// Configure sets up the Tailscale client based on the provider-level data.
func (p *tailscaleProvider) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {
	// Check environment variables
	apiKey := os.Getenv("TAILSCALE_API_KEY")

	// Support both sets of OAuth Env vars for backwards compatibility
	identityToken := getMultiEnv("TAILSCALE_IDENTITY_TOKEN", "IDENTITY_TOKEN")
	oauthClientID := getMultiEnv("TAILSCALE_OAUTH_CLIENT_ID", "OAUTH_CLIENT_ID")
	oauthClientSecret := getMultiEnv("TAILSCALE_OAUTH_CLIENT_SECRET", "OAUTH_CLIENT_SECRET")

	tailnet := "-"
	if value, ok := os.LookupEnv("TAILSCALE_TAILNET"); ok {
		tailnet = value
	}

	baseURL := "https://api.tailscale.com"
	if value, ok := os.LookupEnv("TAILSCALE_BASE_URL"); ok {
		baseURL = value
	}

	userAgent := fmt.Sprintf(
		"Terraform/%s (+https://www.terraform.io) terraform-provider-tailscale/%s",
		req.TerraformVersion,
		providerVersion)

	var data tailscaleProviderModel

	// Read configuration data into model
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)

	// Check configuration data, which should take precedence over
	// environment variable data, if found.
	if data.APIKey.ValueString() != "" {
		apiKey = data.APIKey.ValueString()
	}
	if data.IdentityToken.ValueString() != "" {
		identityToken = data.IdentityToken.ValueString()
	}
	if data.OAuthClientID.ValueString() != "" {
		oauthClientID = data.OAuthClientID.ValueString()
	}
	if data.OAuthClientSecret.ValueString() != "" {
		oauthClientSecret = data.OAuthClientSecret.ValueString()
	}
	if data.Tailnet.ValueString() != "" {
		tailnet = data.Tailnet.ValueString()
	}
	if data.BaseURL.ValueString() != "" {
		baseURL = data.BaseURL.ValueString()
	}
	if data.UserAgent.ValueString() != "" {
		userAgent = data.UserAgent.ValueString()
	}

	var scopes []string
	resp.Diagnostics.Append(data.Scopes.ElementsAs(ctx, &scopes, false)...)

	parsedBaseURL, err := url.Parse(baseURL)
	if err != nil {
		resp.Diagnostics.AddError(
			"Could not parse base URL",
			fmt.Sprintf("While configuring the provider, "+
				"the base URL %q could not be parsed: %v", baseURL, err),
		)
	}

	// TODO(alexc): I copied this check from the old provider, but if I've read the code
	// correctly it will never be triggered: the default value for tailnet is a hyphen,
	// not an empty string.
	if tailnet == "" {
		resp.Diagnostics.AddError(
			"Missing Tailnet ID",
			"While configuring the provider, a Tailnet ID was not found in the "+
				"TAILSCALE_TAILNET environment variable or provider configuration block "+
				"tailnet attribute.",
		)
	}

	if apiKey == "" && oauthClientID == "" && oauthClientSecret == "" && identityToken == "" {
		resp.Diagnostics.AddError(
			"Provider credentials are missing",
			"While configuring the provider, no provider credentials were found. Either set "+
				"an API key, or an OAuth client ID and OAuth client secret, or an OAuth client ID "+
				"and identity token.",
		)
	} else if apiKey != "" && (oauthClientID != "" || oauthClientSecret != "" || identityToken != "") {
		resp.Diagnostics.AddError(
			"Provider credentials are conflicting",
			"While configuring the provider, both API key and OAuth client credentials were found. "+
				"Only one can be used. Remove either your API key or OAuth client configuration.",
		)
	} else if apiKey == "" && oauthClientID == "" && !(oauthClientSecret == "" && identityToken == "") {
		resp.Diagnostics.AddError(
			"OAuth client ID is missing",
			"While configuring the provider, no provider credentials were found. Set an OAuth "+
				"client ID in the TAILSCALE_OAUTH_CLIENT_ID environment variable or provider "+
				"configuration block oauth_client_id attribute.",
		)
	} else if apiKey == "" && oauthClientID != "" && (oauthClientSecret == "" && identityToken == "") {
		resp.Diagnostics.AddError(
			"OAuth client secret or identity token is missing",
			"While configuring the provider, no provider credentials were found. Set either "+
				"(1) an OAuth client secret in the TAILSCALE_OAUTH_CLIENT_SECRET environment variable "+
				"or provider configuration block oauth_client_secret attribute, or (2) an identity token "+
				"in the TAILSCALE_IDENTITY_TOKEN environment variable or provider configuration block "+
				"identity_token attribute.",
		)
	}

	if resp.Diagnostics.HasError() {
		return
	}

	if oauthClientID != "" && oauthClientSecret != "" {
		p.Client = tailscale.Client{
			BaseURL:   parsedBaseURL,
			UserAgent: userAgent,
			Tailnet:   tailnet,
			Auth: &tailscale.OAuth{
				ClientID:     oauthClientID,
				ClientSecret: oauthClientSecret,
				Scopes:       scopes,
			},
		}
	} else if oauthClientID != "" && identityToken != "" {
		p.Client = tailscale.Client{
			BaseURL:   parsedBaseURL,
			UserAgent: userAgent,
			Tailnet:   tailnet,
			Auth: &tailscale.IdentityFederation{
				ClientID: oauthClientID,
				IDTokenFunc: func() (string, error) {
					return identityToken, nil
				},
			},
		}
	} else {
		p.Client = tailscale.Client{
			BaseURL:   parsedBaseURL,
			UserAgent: userAgent,
			APIKey:    apiKey,
			Tailnet:   tailnet,
		}
	}
}

// Resources returns a slice of resources.
func (p *tailscaleProvider) Resources(_ context.Context) []func() resource.Resource {
	return []func() resource.Resource{}
}

// DataSources returns a slice of data sources.
func (p *tailscaleProvider) DataSources(_ context.Context) []func() datasource.DataSource {
	return []func() datasource.DataSource{}
}

// getMultiEnv is a helper function that returns the value of the first environment
// variable in the given list that returns a non-empty value.
//
// It's the multi-variate version of [os.GetEnv].
//
// If none of the environment variables returns a value, it returns an empty string.
func getMultiEnv(ks ...string) string {
	for _, key := range ks {
		if s := os.Getenv(key); s != "" {
			return s
		}
	}
	return ""
}

// Provider returns the [schema.Provider] instance that implements the terraform provider.
//
// This implements the SDKv2 version of the Terraform provider, and will gradually be
// removed and eventually deleted as we migrate to the plugin framework.
func Provider(options ...ProviderOption) *schema.Provider {
	// Support both sets of OAuth Env vars for backwards compatibility
	oauthClientIDEnvVars := []string{"TAILSCALE_OAUTH_CLIENT_ID", "OAUTH_CLIENT_ID"}
	oauthClientSecretEnvVars := []string{"TAILSCALE_OAUTH_CLIENT_SECRET", "OAUTH_CLIENT_SECRET"}
	identityTokenEnvVars := []string{"TAILSCALE_IDENTITY_TOKEN", "IDENTITY_TOKEN"}

	provider := &schema.Provider{
		Schema: map[string]*schema.Schema{
			"api_key": {
				Type:        schema.TypeString,
				DefaultFunc: schema.EnvDefaultFunc("TAILSCALE_API_KEY", ""),
				Optional:    true,
				Description: "The API key to use for authenticating requests to the API. Can be set via the TAILSCALE_API_KEY environment variable. Conflicts with 'oauth_client_id' and 'oauth_client_secret'.",
				Sensitive:   true,
			},
			"identity_token": {
				Type:        schema.TypeString,
				DefaultFunc: schema.MultiEnvDefaultFunc(identityTokenEnvVars, ""),
				Optional:    true,
				Description: "The jwt identity token to exchange for a Tailscale API token when using a federated identity. Can be set via the TAILSCALE_IDENTITY_TOKEN environment variable. Conflicts with 'api_key' and 'oauth_client_secret'.",
				Sensitive:   true,
			},
			"oauth_client_id": {
				Type:        schema.TypeString,
				DefaultFunc: schema.MultiEnvDefaultFunc(oauthClientIDEnvVars, ""),
				Optional:    true,
				Description: "The OAuth application or federated identity's ID when using OAuth client credentials or workload identity federation. Can be set via the TAILSCALE_OAUTH_CLIENT_ID environment variable. Either 'oauth_client_secret' or 'identity_token' must be set alongside 'oauth_client_id'. Conflicts with 'api_key'.",
			},
			"oauth_client_secret": {
				Type:        schema.TypeString,
				DefaultFunc: schema.MultiEnvDefaultFunc(oauthClientSecretEnvVars, ""),
				Optional:    true,
				Description: "The OAuth application's secret when using OAuth client credentials. Can be set via the TAILSCALE_OAUTH_CLIENT_SECRET environment variable. Conflicts with 'api_key' and 'identity_token'.",
				Sensitive:   true,
			},
			"scopes": {
				Type:        schema.TypeList,
				Optional:    true,
				Elem:        &schema.Schema{Type: schema.TypeString},
				Description: "The OAuth 2.0 scopes to request when generating the access token using the supplied OAuth client credentials. See https://tailscale.com/kb/1623/trust-credentials#scopes for available scopes. Only valid when both 'oauth_client_id' and 'oauth_client_secret', or both are set.",
			},
			"tailnet": {
				Type:        schema.TypeString,
				DefaultFunc: schema.EnvDefaultFunc("TAILSCALE_TAILNET", "-"),
				Optional:    true,
				Description: "The tailnet ID. Tailnets created before Oct 2025 can still use the legacy ID, but the Tailnet ID is the preferred identifier. Can be set via the TAILSCALE_TAILNET environment variable. Default is the tailnet that owns API credentials passed to the provider.",
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
			"tailscale_acl":                     resourceACL(),
			"tailscale_dns_configuration":       resourceDNSConfiguration(),
			"tailscale_dns_nameservers":         resourceDNSNameservers(),
			"tailscale_dns_preferences":         resourceDNSPreferences(),
			"tailscale_dns_search_paths":        resourceDNSSearchPaths(),
			"tailscale_dns_split_nameservers":   resourceDNSSplitNameservers(),
			"tailscale_device_subnet_routes":    resourceDeviceSubnetRoutes(),
			"tailscale_device_authorization":    resourceDeviceAuthorization(),
			"tailscale_tailnet_key":             resourceTailnetKey(),
			"tailscale_device_tags":             resourceDeviceTags(),
			"tailscale_device_key":              resourceDeviceKey(),
			"tailscale_oauth_client":            resourceOAuthClient(),
			"tailscale_webhook":                 resourceWebhook(),
			"tailscale_contacts":                resourceContacts(),
			"tailscale_posture_integration":     resourcePostureIntegration(),
			"tailscale_logstream_configuration": resourceLogstreamConfiguration(),
			"tailscale_aws_external_id":         resourceAWSExternalID(),
			"tailscale_tailnet_settings":        resourceTailnetSettings(),
			"tailscale_federated_identity":      resourceFederatedIdentity(),
		},
		DataSourcesMap: map[string]*schema.Resource{
			"tailscale_device":  dataSourceDevice(),
			"tailscale_devices": dataSourceDevices(),
			"tailscale_4via6":   dataSource4Via6(),
			"tailscale_acl":     dataSourceACL(),
			"tailscale_user":    dataSourceUser(),
			"tailscale_users":   dataSourceUsers(),
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
	idToken := d.Get("identity_token").(string)

	if diags := validateProviderCreds(apiKey, oauthClientID, oauthClientSecret, idToken); diags != nil && diags.HasError() {
		return nil, diags
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

		client := &tailscale.Client{
			BaseURL:   parsedBaseURL,
			UserAgent: userAgent,
			Tailnet:   tailnet,
			Auth: &tailscale.OAuth{
				ClientID:     oauthClientID,
				ClientSecret: oauthClientSecret,
				Scopes:       oauthScopes,
			},
		}

		return client, nil
	}

	if oauthClientID != "" && idToken != "" {
		return &tailscale.Client{
			BaseURL:   parsedBaseURL,
			UserAgent: userAgent,
			Tailnet:   tailnet,
			Auth: &tailscale.IdentityFederation{
				ClientID: oauthClientID,
				IDTokenFunc: func() (string, error) {
					return idToken, nil
				},
			},
		}, nil
	}

	client := &tailscale.Client{
		BaseURL:   parsedBaseURL,
		UserAgent: userAgent,
		APIKey:    apiKey,
		Tailnet:   tailnet,
	}

	return client, nil
}

func validateProviderCreds(apiKey string, oauthClientID string, oauthClientSecret string, idToken string) diag.Diagnostics {
	if apiKey == "" && oauthClientID == "" && oauthClientSecret == "" && idToken == "" {
		return diag.Errorf("tailscale provider credentials are empty - set `api_key` or 'oauth_client_id' and either 'oauth_client_secret' or 'identity_token'")
	} else if apiKey != "" && (oauthClientID != "" || oauthClientSecret != "" || idToken != "") {
		return diag.Errorf("tailscale provider credentials are conflicting - `api_key` conflicts with 'oauth_client_id', 'oauth_client_secret' and 'identity_token'")
	} else if apiKey == "" && oauthClientID == "" {
		return diag.Errorf("tailscale provider argument 'oauth_client_id' is empty")
	} else if oauthClientID != "" && (oauthClientSecret == "" && idToken == "") {
		return diag.Errorf("one of tailscale provider arguments 'oauth_client_secret' or 'identity_token' are mandatory with 'oauth_client_id'")
	}

	return nil
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

func diagnosticsAsError(diags diag.Diagnostics) error {
	var combined string
	for _, d := range diags {
		if d.Severity == diag.Error {
			combined += fmt.Sprintf("%s: %s\n", d.Summary, d.Detail)
		}
	}

	if combined == "" {
		return nil
	}

	return errors.New(combined)
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

// setProperties sets the properties of a ResourceData from the values in the
// given map. Existing ResourceData properties that don't appear in the map are
// left as-is.
func setProperties(d *schema.ResourceData, props map[string]any) diag.Diagnostics {
	for name, value := range props {
		if err := d.Set(name, value); err != nil {
			return diagnosticsError(err, "failed to set %s", name)
		}
	}
	return nil
}

// optional returns a pointer to the value at key in the given resource if,
// and only if, the value has changed. If the value is unchanged, it returns nil.
func optional[T any](d *schema.ResourceData, key string) *T {
	if !d.HasChange(key) {
		return nil
	}
	return tailscale.PointerTo(d.Get(key).(T))
}

// isAcceptanceTesting returns true if we're running acceptance tests.
func isAcceptanceTesting() bool {
	return os.Getenv("TF_ACC") != ""
}

// combinedSchemas creates a schema that combines two supplied schemas.
// Properties in schema b overwrite the same properties in schema b.
func combinedSchemas(a, b map[string]*schema.Schema) map[string]*schema.Schema {
	out := make(map[string]*schema.Schema, len(a)+len(b))
	maps.Copy(out, a)
	maps.Copy(out, b)
	return out
}
