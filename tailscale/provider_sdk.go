// Copyright (c) David Bond, Tailscale Inc, & Contributors
// SPDX-License-Identifier: MIT

// Package tailscale describes the resources and data sources provided by the terraform provider. Each resource
// or data source is described within its own file.
package tailscale

import (
	"context"
	"errors"
	"net/url"
	"os"

	"github.com/hashicorp/go-uuid"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
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
			"audience": {
				Type:        schema.TypeString,
				DefaultFunc: schema.EnvDefaultFunc("TAILSCALE_AUDIENCE", ""),
				Optional:    true,
				Description: "The OIDC audience to request when discovering an identity token from the runtime (GitHub Actions, AWS, or GCP) for workload identity federation. Can be set via the TAILSCALE_AUDIENCE environment variable. Requires 'oauth_client_id'. Conflicts with 'api_key', 'oauth_client_secret', 'identity_token', and 'identity_token_environment_variable_name'.",
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
		ResourcesMap: map[string]*schema.Resource{},
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
	audience := d.Get("audience").(string)

	if err := validateProviderCreds(apiKey, oauthClientID, oauthClientSecret, idToken, audience); err != nil {
		return nil, diag.Errorf("%s", err.Error())
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

	client := createTailscaleClient(parsedBaseURL, userAgent, tailnet, apiKey, oauthClientID, oauthClientSecret, idToken, audience, scopes)
	return &client, nil
}

func validateProviderCreds(apiKey, oauthClientID, oauthClientSecret, idToken, audience string) error {
	if apiKey == "" && oauthClientID == "" && oauthClientSecret == "" && idToken == "" && audience == "" {
		return errors.New("tailscale provider credentials are empty - set `api_key` or 'oauth_client_id' and one of 'oauth_client_secret', 'identity_token', or 'audience'")
	} else if apiKey != "" && (oauthClientID != "" || oauthClientSecret != "" || idToken != "" || audience != "") {
		return errors.New("tailscale provider credentials are conflicting - `api_key` conflicts with 'oauth_client_id', 'oauth_client_secret', 'identity_token', and 'audience'")
	} else if audience != "" && (oauthClientSecret != "" || idToken != "") {
		return errors.New("tailscale provider argument 'audience' conflicts with 'oauth_client_secret' and 'identity_token'")
	} else if apiKey == "" && oauthClientID == "" {
		return errors.New("tailscale provider argument 'oauth_client_id' is empty")
	} else if oauthClientID != "" && oauthClientSecret == "" && idToken == "" && audience == "" {
		return errors.New("one of tailscale provider arguments 'oauth_client_secret', 'identity_token', or 'audience' are mandatory with 'oauth_client_id'")
	}

	return nil
}

func createUUID() string {
	val, err := uuid.GenerateUUID()
	if err != nil {
		panic(err)
	}
	return val
}

// isAcceptanceTesting returns true if we're running acceptance tests.
func isAcceptanceTesting() bool {
	return os.Getenv("TF_ACC") != ""
}
