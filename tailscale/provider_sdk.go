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

	"github.com/hashicorp/go-cty/cty"
	"github.com/hashicorp/go-uuid"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"

	"tailscale.com/client/tailscale/v2"
)

// providerVersion is filled by goreleaser at build time.
var providerVersion = "dev"

type ProviderOption func(p *schema.Provider)

// Provider returns the [schema.Provider] instance that implements the terraform provider.
//
// This implements the SDKv2 version of the Terraform provider, and will gradually be
// removed and eventually deleted as we migrate to the plugin framework.
//
// Remove this when we close https://github.com/tailscale/corp/issues/37032
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
				Description: "The jwt identity token to exchange for a Tailscale API token when using a federated identity. Can be set via the TAILSCALE_IDENTITY_TOKEN environment variable. Conflicts with 'api_key', 'oauth_client_secret', and 'identity_token_environment_variable_name'.",
				Sensitive:   true,
			},
			"identity_token_environment_variable_name": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "The name of an environment variable to read the identity token from. This is useful when the identity token is provided by an external system (such as Terraform Cloud workload identity) in an environment variable you do not control. Conflicts with 'identity_token'.",
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
			"tailscale_tailnet_key":             resourceTailnetKey(),
			"tailscale_oauth_client":            resourceOAuthClient(),
			"tailscale_webhook":                 resourceWebhook(),
			"tailscale_contacts":                resourceContacts(),
			"tailscale_posture_integration":     resourcePostureIntegration(),
			"tailscale_logstream_configuration": resourceLogstreamConfiguration(),
			"tailscale_tailnet_settings":        resourceTailnetSettings(),
			"tailscale_federated_identity":      resourceFederatedIdentity(),
			"tailscale_service":                 resourceService(),
		},
		DataSourcesMap: map[string]*schema.Resource{
			"tailscale_devices": dataSourceDevices(),
			"tailscale_user":    dataSourceUser(),
			"tailscale_users":   dataSourceUsers(),
			"tailscale_service": dataSourceService(),
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
	if idToken == "" {
		if envVarName := d.Get("identity_token_environment_variable_name").(string); envVarName != "" {
			idToken = os.Getenv(envVarName)
		}
	}

	if diags := validateProviderCreds(apiKey, oauthClientID, oauthClientSecret, idToken); diags != nil && diags.HasError() {
		return nil, diags
	}

	userAgent := d.Get("user_agent").(string)
	if userAgent == "" {
		userAgent = provider.UserAgent("terraform-provider-tailscale", providerVersion)
	}

	var scopes []string
	if oauthClientID != "" && oauthClientSecret != "" {
		oauthScopesFromConfig := d.Get("scopes").([]interface{})
		if len(oauthScopesFromConfig) > 0 {
			scopes = make([]string, len(oauthScopesFromConfig))
		}
		for i, scope := range oauthScopesFromConfig {
			scopes[i] = scope.(string)
		}
	}

	client := createTailscaleClient(parsedBaseURL, userAgent, tailnet, apiKey, oauthClientID, oauthClientSecret, idToken, scopes)
	return &client, nil
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
