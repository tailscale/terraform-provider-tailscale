// Copyright (c) David Bond, Tailscale Inc, & Contributors
// SPDX-License-Identifier: MIT

package tailscale

import (
	"context"
	"net/url"
	"time"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"tailscale.com/client/tailscale/v2"
)

// NewFederatedIdentityResource returns a new federated identity resource.
func NewFederatedIdentityResource() resource.Resource {
	return &federatedIdentityResource{}
}

type federatedIdentityResource struct {
	ResourceBase
}

// Metadata defines the resource name as it appears in Terraform configurations.
func (r *federatedIdentityResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_federated_identity"
}

// Schema defines a schema describing what fields can be defined in the resource.
func (r *federatedIdentityResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "The federated_identity resource allows you to create federated identities to programmatically interact with the Tailscale API using workload identity federation.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "The client ID, also known as the key id. Used with an OIDC identity token to generate access tokens.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"description": schema.StringAttribute{
				Description: "A description of the federated identity consisting of alphanumeric characters. Defaults to `\"\"`.",
				Optional:    true,
				Validators: []validator.String{
					stringvalidator.LengthAtMost(50),
				},
			},
			"scopes": schema.SetAttribute{
				Description: "Scopes to grant to the federated identity. See https://tailscale.com/kb/1623/ for a list of available scopes.",
				Required:    true,
				ElementType: types.StringType,
			},
			"tags": schema.SetAttribute{
				Description: "A list of tags that access tokens generated for the federated identity will be able to assign to devices. Mandatory if the scopes include \"devices:core\" or \"auth_keys\".",
				Optional:    true,
				ElementType: types.StringType,
			},
			"audience": schema.StringAttribute{
				Description: "The value used when matching against the `aud` claim from an OIDC identity token. Specifying the audience is optional as Tailscale will generate a secure audience at creation time by default.   It is recommended to let Tailscale generate the audience unless the identity provider you are integrating with requires a specific audience format.",
				Optional:    true,
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"subject": schema.StringAttribute{
				Description: "The pattern used when matching against the `sub` claim from an OIDC identity token. Patterns can include `*` characters to match against any character.",
				Required:    true,
			},
			"issuer": schema.StringAttribute{
				Description: "The issuer of the OIDC identity token used in the token exchange. Must be a valid and publicly reachable https:// URL.",
				Required:    true,
				Validators: []validator.String{
					httpsURLValidator{},
				},
			},
			"custom_claim_rules": schema.MapAttribute{
				Description: "A map of claim names to pattern strings used to match against arbitrary claims in the OIDC identity token. Patterns can include `*` characters to match against any character.",
				Optional:    true,
				ElementType: types.StringType,
				Validators: []validator.Map{
					reservedClaimKeysValidator{},
				},
			},
			"created_at": schema.StringAttribute{
				Description: "The creation timestamp of the key in RFC3339 format",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"updated_at": schema.StringAttribute{
				Description: "The updated timestamp of the key in RFC3339 format",
				Computed:    true,
			},
			"user_id": schema.StringAttribute{
				Description: "ID of the user who created this federated identity, empty for federated identities created by other trust credentials.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
		},
	}
}

type federatedIdentityResourceModel struct {
	ID               types.String `tfsdk:"id"`
	Description      types.String `tfsdk:"description"`
	Scopes           types.Set    `tfsdk:"scopes"`
	Tags             types.Set    `tfsdk:"tags"`
	Audience         types.String `tfsdk:"audience"`
	Subject          types.String `tfsdk:"subject"`
	Issuer           types.String `tfsdk:"issuer"`
	CustomClaimRules types.Map    `tfsdk:"custom_claim_rules"`
	CreatedAt        types.String `tfsdk:"created_at"`
	UpdatedAt        types.String `tfsdk:"updated_at"`
	UserID           types.String `tfsdk:"user_id"`
}

// Create creates a new federated identity.
func (r *federatedIdentityResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data federatedIdentityResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	createReq := tailscale.CreateFederatedIdentityRequest{
		Description: data.Description.ValueString(),
		Subject:     data.Subject.ValueString(),
		Issuer:      data.Issuer.ValueString(),
		Audience:    data.Audience.ValueString(),
	}

	if !data.Scopes.IsNull() {
		resp.Diagnostics.Append(data.Scopes.ElementsAs(ctx, &createReq.Scopes, false)...)
	}
	if !data.Tags.IsNull() {
		resp.Diagnostics.Append(data.Tags.ElementsAs(ctx, &createReq.Tags, false)...)
	}

	createReq.CustomClaimRules = map[string]string{}
	if !data.CustomClaimRules.IsNull() {
		resp.Diagnostics.Append(data.CustomClaimRules.ElementsAs(ctx, &createReq.CustomClaimRules, false)...)
	}

	if resp.Diagnostics.HasError() {
		return
	}

	key, err := r.Client.Keys().CreateFederatedIdentity(ctx, createReq)
	if err != nil {
		resp.Diagnostics.AddError("Failed to create federated identity", err.Error())
		return
	}

	resp.Diagnostics.Append(r.populateFromKey(ctx, &data, key)...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// Read fetches the current state of the federated identity.
func (r *federatedIdentityResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data federatedIdentityResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	key, err := r.Client.Keys().Get(ctx, data.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Failed to fetch federated identity", err.Error())
		return
	}

	resp.Diagnostics.Append(r.populateFromKey(ctx, &data, key)...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// Update updates an existing federated identity.
func (r *federatedIdentityResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data federatedIdentityResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	updateReq := tailscale.SetFederatedIdentityRequest{
		Description: data.Description.ValueString(),
		Subject:     data.Subject.ValueString(),
		Issuer:      data.Issuer.ValueString(),
		Audience:    data.Audience.ValueString(),
	}

	if !data.Scopes.IsNull() && !data.Scopes.IsUnknown() {
		resp.Diagnostics.Append(data.Scopes.ElementsAs(ctx, &updateReq.Scopes, false)...)
	}
	if !data.Tags.IsNull() {
		resp.Diagnostics.Append(data.Tags.ElementsAs(ctx, &updateReq.Tags, false)...)
	}

	updateReq.CustomClaimRules = map[string]string{}
	if !data.CustomClaimRules.IsNull() {
		resp.Diagnostics.Append(data.CustomClaimRules.ElementsAs(ctx, &updateReq.CustomClaimRules, false)...)
	}

	if resp.Diagnostics.HasError() {
		return
	}

	key, err := r.Client.Keys().SetFederatedIdentity(ctx, data.ID.ValueString(), updateReq)
	if err != nil {
		resp.Diagnostics.AddError("Failed to update federated identity", err.Error())
		return
	}

	resp.Diagnostics.Append(r.populateFromKey(ctx, &data, key)...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// Delete deletes a federated identity.
func (r *federatedIdentityResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data federatedIdentityResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := r.Client.Keys().Delete(ctx, data.ID.ValueString())
	if err != nil && !tailscale.IsNotFound(err) {
		resp.Diagnostics.AddError("Failed to delete federated identity", err.Error())
	}
}

// ImportState implements state passthrough for import.
func (r *federatedIdentityResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

// populateFromKey updates the model with data from the API key response.
func (r *federatedIdentityResource) populateFromKey(ctx context.Context, data *federatedIdentityResourceModel, key *tailscale.Key) diag.Diagnostics {
	var diags diag.Diagnostics

	data.ID = types.StringValue(key.ID)
	// description is Optional (not Computed): preserve null when the API returns ""
	// and the prior value was already null, to avoid unnecessary drift.
	if key.Description != "" || !data.Description.IsNull() {
		data.Description = types.StringValue(key.Description)
	}
	data.Audience = types.StringValue(key.Audience)
	data.Subject = types.StringValue(key.Subject)
	data.Issuer = types.StringValue(key.Issuer)
	data.CreatedAt = types.StringValue(key.Created.Format(time.RFC3339))
	data.UpdatedAt = types.StringValue(key.Updated.Format(time.RFC3339))
	data.UserID = types.StringValue(key.UserID)

	scopes := key.Scopes
	if scopes == nil {
		scopes = []string{}
	}
	scopesVal, d := types.SetValueFrom(ctx, types.StringType, scopes)
	diags.Append(d...)
	data.Scopes = scopesVal

	tags := key.Tags
	if tags == nil {
		tags = []string{}
	}
	tagsVal, d := types.SetValueFrom(ctx, types.StringType, tags)
	diags.Append(d...)
	data.Tags = tagsVal

	if len(key.CustomClaimRules) > 0 || !data.CustomClaimRules.IsNull() {
		claimRules := key.CustomClaimRules
		if claimRules == nil {
			claimRules = map[string]string{}
		}
		claimRulesVal, d := types.MapValueFrom(ctx, types.StringType, claimRules)
		diags.Append(d...)
		data.CustomClaimRules = claimRulesVal
	}

	return diags
}

// reservedClaimKeysValidator rejects "sub" and "iss" as custom_claim_rules keys,
// since those claims are matched via the dedicated subject and issuer fields.
type reservedClaimKeysValidator struct{}

func (v reservedClaimKeysValidator) Description(_ context.Context) string {
	return `Keys "sub","iss" and "aud" are reserved and must not appear in custom_claim_rules.`
}

func (v reservedClaimKeysValidator) MarkdownDescription(ctx context.Context) string {
	return v.Description(ctx)
}

func (v reservedClaimKeysValidator) ValidateMap(_ context.Context, req validator.MapRequest, resp *validator.MapResponse) {
	if req.ConfigValue.IsNull() || req.ConfigValue.IsUnknown() {
		return
	}
	for key := range req.ConfigValue.Elements() {
		if key == "sub" || key == "iss" || key == "aud" {
			resp.Diagnostics.AddAttributeError(
				req.Path,
				"Reserved claim key",
				`Keys "sub", "iss" and "aud" are reserved and must not appear in custom_claim_rules. Use the "subject" and "issuer" fields instead.`,
			)
			return
		}
	}
}

// httpsURLValidator validates that a string value is a valid HTTPS URL.
type httpsURLValidator struct{}

func (v httpsURLValidator) Description(_ context.Context) string {
	return "Value must be a valid https:// URL."
}

func (v httpsURLValidator) MarkdownDescription(ctx context.Context) string {
	return v.Description(ctx)
}

func (v httpsURLValidator) ValidateString(_ context.Context, req validator.StringRequest, resp *validator.StringResponse) {
	if req.ConfigValue.IsNull() || req.ConfigValue.IsUnknown() {
		return
	}
	u, err := url.ParseRequestURI(req.ConfigValue.ValueString())
	if err != nil || u.Scheme != "https" || u.Host == "" {
		resp.Diagnostics.AddAttributeError(req.Path, "Invalid URL", "Must be a valid https:// URL.")
	}
}
