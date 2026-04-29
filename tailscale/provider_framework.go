// Copyright (c) David Bond, Tailscale Inc, & Contributors
// SPDX-License-Identifier: MIT

// Package tailscale describes the resources and data sources provided by the terraform provider. Each resource
// or data source is described within its own file.
package tailscale

import (
	"context"
	"fmt"
	"net/url"
	"os"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"tailscale.com/client/tailscale/v2"
)

var (
	_ planmodifier.String = PreserveEmptyStringAsNull{}
)

type tailscaleProvider struct {
	Client tailscale.Client
}

// NewFrameworkProvider creates a new instance of the Terraform provider.
//
// TODO(alexc): This name is to distinguish it from the old provider written using
// the plugin SDK. When we delete the plugin SDK code, we can rename this to NewProvider.
func NewFrameworkProvider() provider.Provider {
	return &tailscaleProvider{}
}

// Metadata defines information about the provider itself.
func (p *tailscaleProvider) Metadata(_ context.Context, _ provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "tailscale"
	resp.Version = providerVersion
}

// Schema defines a [schema.Schema] describing what data is available in the provider's
// configuration.
func (p *tailscaleProvider) Schema(_ context.Context, _ provider.SchemaRequest, resp *provider.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"api_key": schema.StringAttribute{
				Optional:    true,
				Description: "The API key to use for authenticating requests to the API. Can be set via the TAILSCALE_API_KEY environment variable. Conflicts with 'oauth_client_id' and 'oauth_client_secret'.",
				Sensitive:   true,
			},
			"identity_token": schema.StringAttribute{
				Optional:    true,
				Description: "The jwt identity token to exchange for a Tailscale API token when using a federated identity. Can be set via the TAILSCALE_IDENTITY_TOKEN environment variable. Conflicts with 'api_key', 'oauth_client_secret', and 'identity_token_environment_variable_name'.",
				Sensitive:   true,
			},
			"identity_token_environment_variable_name": schema.StringAttribute{
				Optional:    true,
				Description: "The name of an environment variable to read the identity token from. This is useful when the identity token is provided by an external system (such as Terraform Cloud workload identity) in an environment variable you do not control. Conflicts with 'identity_token'.",
			},
			"oauth_client_id": schema.StringAttribute{
				Optional:    true,
				Description: "The OAuth application or federated identity's ID when using OAuth client credentials or workload identity federation. Can be set via the TAILSCALE_OAUTH_CLIENT_ID environment variable. Either 'oauth_client_secret' or 'identity_token' must be set alongside 'oauth_client_id'. Conflicts with 'api_key'.",
			},
			"oauth_client_secret": schema.StringAttribute{
				Optional:    true,
				Description: "The OAuth application's secret when using OAuth client credentials. Can be set via the TAILSCALE_OAUTH_CLIENT_SECRET environment variable. Conflicts with 'api_key' and 'identity_token'.",
				Sensitive:   true,
			},
			"scopes": schema.ListAttribute{
				ElementType: types.StringType,
				Optional:    true,
				Description: "The OAuth 2.0 scopes to request when generating the access token using the supplied OAuth client credentials. See https://tailscale.com/kb/1623/trust-credentials#scopes for available scopes. Only valid when both 'oauth_client_id' and 'oauth_client_secret', or both are set.",
			},
			"tailnet": schema.StringAttribute{
				Optional:    true,
				Description: "The tailnet ID. Tailnets created before Oct 2025 can still use the legacy ID, but the Tailnet ID is the preferred identifier. Can be set via the TAILSCALE_TAILNET environment variable. Default is the tailnet that owns API credentials passed to the provider.",
			},
			"base_url": schema.StringAttribute{
				Optional:    true,
				Description: "The base URL of the Tailscale API. Defaults to https://api.tailscale.com. Can be set via the TAILSCALE_BASE_URL environment variable.",
			},
			"user_agent": schema.StringAttribute{
				Optional:    true,
				Description: "User-Agent header for API requests.",
			},
		},
	}
}

type tailscaleProviderModel struct {
	APIKey                               types.String `tfsdk:"api_key"`
	IdentityToken                        types.String `tfsdk:"identity_token"`
	IdentityTokenEnvironmentVariableName types.String `tfsdk:"identity_token_environment_variable_name"`
	OAuthClientID                        types.String `tfsdk:"oauth_client_id"`
	OAuthClientSecret                    types.String `tfsdk:"oauth_client_secret"`
	Tailnet                              types.String `tfsdk:"tailnet"`
	BaseURL                              types.String `tfsdk:"base_url"`
	UserAgent                            types.String `tfsdk:"user_agent"`
	Scopes                               types.List   `tfsdk:"scopes"`
}

// Configure sets up the Tailscale client based on the provider-level data.
func (p *tailscaleProvider) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {
	var data tailscaleProviderModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)

	apiKey := coalesce(data.APIKey, os.Getenv("TAILSCALE_API_KEY"))
	tailnet := coalesce(data.Tailnet, os.Getenv("TAILSCALE_TAILNET"), "-")
	baseURL := coalesce(data.BaseURL, os.Getenv("TAILSCALE_BASE_URL"), "https://api.tailscale.com")

	// Support both sets of OAuth Env vars for backwards compatibility
	identityTokenFallbacks := []string{os.Getenv("TAILSCALE_IDENTITY_TOKEN"), os.Getenv("IDENTITY_TOKEN")}
	if envVarName := data.IdentityTokenEnvironmentVariableName.ValueString(); envVarName != "" {
		identityTokenFallbacks = append(identityTokenFallbacks, os.Getenv(envVarName))
	}
	identityToken := coalesce(data.IdentityToken, identityTokenFallbacks...)
	oauthClientID := coalesce(data.OAuthClientID, os.Getenv("TAILSCALE_OAUTH_CLIENT_ID"), os.Getenv("OAUTH_CLIENT_ID"))
	oauthClientSecret := coalesce(data.OAuthClientSecret, os.Getenv("TAILSCALE_OAUTH_CLIENT_SECRET"), os.Getenv("OAUTH_CLIENT_SECRET"))

	var userAgent string
	if data.UserAgent.ValueString() != "" {
		userAgent = data.UserAgent.ValueString()
	} else {
		userAgent = fmt.Sprintf(
			"Terraform/%s (+https://www.terraform.io) terraform-provider-tailscale/%s",
			req.TerraformVersion,
			providerVersion)
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

	if tailnet == "" {
		resp.Diagnostics.AddError(
			"Missing Tailnet ID",
			"While configuring the provider, a Tailnet ID was not found in the "+
				"TAILSCALE_TAILNET environment variable or provider configuration block "+
				"tailnet attribute.",
		)
	}

	if err := validateProviderCreds(apiKey, oauthClientID, oauthClientID, identityToken); err != nil {
		resp.Diagnostics.AddError("Provider credentials error", err[0].Summary)
	}

	if resp.Diagnostics.HasError() {
		return
	}

	p.Client = createTailscaleClient(parsedBaseURL, userAgent, tailnet, apiKey, oauthClientID, oauthClientSecret, identityToken, scopes)

	// Make the Tailscale client available during DataSource and Resource
	// type Configure methods.
	resp.ResourceData = &p.Client
	resp.DataSourceData = &p.Client
}

// Resources returns a slice of resources.
func (p *tailscaleProvider) Resources(_ context.Context) []func() resource.Resource {
	return []func() resource.Resource{
		NewAWSExternalIDResource,
		NewDeviceAuthorizationResource,
		NewDeviceKeyResource,
		NewDeviceSubnetRoutesResource,
		NewDeviceTagsResource,
		NewDNSConfigurationResource,
		NewDNSNameserversResource,
		NewDNSPreferencesResource,
		NewDNSSearchPathsResource,
		NewDNSSplitNameserversResource,
		NewPostureIntegrationResource,
		NewWebhookResource,
	}
}

// DataSources returns a slice of data sources.
func (p *tailscaleProvider) DataSources(_ context.Context) []func() datasource.DataSource {
	return []func() datasource.DataSource{
		New4Via6DataSource,
		NewACLDataSource,
		NewDeviceDataSource,
	}
}

// coalesce chooses a string value in order of decreasing priority.
//
// It returns the first value which is non-empty -- either configuration data, or
// the first non-empty fallback value.
func coalesce(val types.String, fallbacks ...string) string {
	if !val.IsNull() && !val.IsUnknown() {
		return val.ValueString()
	}
	for _, f := range fallbacks {
		if f != "" {
			return f
		}
	}
	return ""
}

// createTailscaleClient creates a new Tailscale API client based on the credentials
// provided to the Terraform provider.
func createTailscaleClient(baseURL *url.URL, userAgent string, tailnet string, apiKey string, oauthClientID string, oauthClientSecret string, identityToken string, scopes []string) tailscale.Client {
	if oauthClientID != "" && oauthClientSecret != "" {
		return tailscale.Client{
			BaseURL:   baseURL,
			UserAgent: userAgent,
			Tailnet:   tailnet,
			Auth: &tailscale.OAuth{
				ClientID:     oauthClientID,
				ClientSecret: oauthClientSecret,
				Scopes:       scopes,
			},
		}
	} else if oauthClientID != "" && identityToken != "" {
		return tailscale.Client{
			BaseURL:   baseURL,
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
		return tailscale.Client{
			BaseURL:   baseURL,
			UserAgent: userAgent,
			APIKey:    apiKey,
			Tailnet:   tailnet,
		}
	}
}

// StringValueNullIfEmpty returns a StringValue of the given input string, or a
// null StringValue if the input string is empty. Useful for cases where ""
// being returned from the API is equivalent to an unset / null value in the
// Terraform state.
func StringValueNullIfEmpty(s string) types.String {
	if s == "" {
		return types.StringNull()
	} else {
		return types.StringValue(s)
	}
}

// CoalesceStringEmptyOrNull returns a StringValue based on both the new string
// value we have received, and the existing value in state, ensuring that we
// won't set a null string to an empty string, or vice-versa. This is useful in
// cases where null and empty strings are equivalent as far as the client or API
// are concerned, and we want to avoid Terraform thinking there is a diff if
// e.g. the API returns "" but the state has null.
//
// If newValue is a non-empty string, this will always return a StringValue of
// that string.
//
// If newValue is empty, and the current value in state is null or empty, it
// will return the existing stateValue, ensuring the update is no-op.
//
// Otherwise, this follows the rules of [StringValueNullIfEmpty], and will
// return a null StringValue if newValue is empty, and a StringValue of newValue
// otherwise.
func CoalesceStringEmptyOrNull(stateValue types.String, newValue string) types.String {
	if newValue == "" && stateValue.ValueString() == "" && !stateValue.IsUnknown() {
		return stateValue
	}
	return StringValueNullIfEmpty(newValue)
}

// PreserveEmptyStringAsNull is a plan modifier that will treat empty strings in
// the state as equivalent to null values, and not change them. This is needed
// because the plugin SDK provider may have saved empty strings in the state for
// certain attributes when set to null, but the plugin framework-based provider
// will always save null strings as null. In cases where the empty string and
// null are equivalent as far as the client or API are concerned, we therefore
// need to change the plan to avoid changing an empty string to a null, and a
// confusing no-op diff from Terraform. For more details, see:
//   - https://github.com/hashicorp/terraform-plugin-framework/issues/510
//   - https://discuss.hashicorp.com/t/framework-migration-test-produces-non-empty-plan/54523/12
type PreserveEmptyStringAsNull struct{}

func (pm PreserveEmptyStringAsNull) Description(_ context.Context) string {
	return `If the existing value of this attribute in state is "" and the new value is null, the value of this attribute in state will remain as the empty string.`
}

func (pm PreserveEmptyStringAsNull) MarkdownDescription(_ context.Context) string {
	return `If the existing value of this attribute in state is "" and the new value is null, the value of this attribute in state will remain as the empty string.`
}

func (pm PreserveEmptyStringAsNull) PlanModifyString(ctx context.Context, req planmodifier.StringRequest, resp *planmodifier.StringResponse) {
	if req.StateValue.ValueString() == "" && !req.StateValue.IsUnknown() && req.ConfigValue.IsNull() {
		resp.PlanValue = types.StringValue("")
	}
}
