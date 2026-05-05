// Copyright (c) David Bond, Tailscale Inc, & Contributors
// SPDX-License-Identifier: MIT

package tailscale

import (
	"context"
	"time"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"tailscale.com/client/tailscale/v2"
)

var (
	_ resource.Resource                = &oauthClientResource{}
	_ resource.ResourceWithImportState = &oauthClientResource{}
)

type oauthClientResourceModel struct {
	ID          types.String `tfsdk:"id"`
	Description types.String `tfsdk:"description"`
	Scopes      types.Set    `tfsdk:"scopes"`
	Tags        types.Set    `tfsdk:"tags"`
	Key         types.String `tfsdk:"key"`
	CreatedAt   types.String `tfsdk:"created_at"`
	UpdatedAt   types.String `tfsdk:"updated_at"`
	UserID      types.String `tfsdk:"user_id"`
}

func NewOAuthClientResource() resource.Resource {
	return &oauthClientResource{}
}

type oauthClientResource struct {
	ResourceBase
}

func (r *oauthClientResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_oauth_client"
}

func (r *oauthClientResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "The oauth_client resource allows you to create OAuth clients to programmatically interact with the Tailscale API.",
		Attributes: map[string]schema.Attribute{
			"description": schema.StringAttribute{
				Optional:    true,
				Description: "A description of the OAuth client consisting of alphanumeric characters. Defaults to `\"\"`.",
				Validators: []validator.String{
					stringvalidator.LengthAtMost(50),
				},
			},
			"scopes": schema.SetAttribute{
				ElementType: types.StringType,
				Required:    true,
				Description: "Scopes to grant to the client. See https://tailscale.com/kb/1623/ for a list of available scopes.",
			},
			"tags": schema.SetAttribute{
				ElementType: types.StringType,
				Optional:    true,
				Description: "A list of tags that access tokens generated for the OAuth client will be able to assign to devices. Mandatory if the scopes include \"devices:core\" or \"auth_keys\".",
			},
			"id": schema.StringAttribute{
				Description: "The client ID, also known as the key id. Used with the client secret to generate access tokens.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"key": schema.StringAttribute{
				Description: "The client secret, also known as the key. Used with the client ID to generate access tokens.",
				Computed:    true,
				Sensitive:   true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
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
				Description: "ID of the user who created this key, empty for OAuth clients created by other trust credentials.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
		},
	}
}

func (r *oauthClientResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state oauthClientResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	key, err := r.Client.Keys().Get(ctx, state.ID.ValueString())
	if err != nil {
		if tailscale.IsNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Failed to fetch oauth client", err.Error())
		return
	}

	state.Description = types.StringValue(key.Description)
	state.CreatedAt = types.StringValue(key.Created.Format(time.RFC3339))
	state.UpdatedAt = types.StringValue(key.Updated.Format(time.RFC3339))
	state.UserID = types.StringValue(key.UserID)
	state.Scopes = SetOfStringValue(ctx, key.Scopes, &resp.Diagnostics)
	state.Tags = SetOfStringValue(ctx, key.Tags, &resp.Diagnostics)

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *oauthClientResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan oauthClientResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var scopes, tags []string
	resp.Diagnostics.Append(plan.Scopes.ElementsAs(ctx, &scopes, false)...)
	resp.Diagnostics.Append(plan.Tags.ElementsAs(ctx, &tags, false)...)

	key, err := r.Client.Keys().CreateOAuthClient(ctx, tailscale.CreateOAuthClientRequest{
		Description: plan.Description.ValueString(),
		Scopes:      scopes,
		Tags:        tags,
	})
	if err != nil {
		resp.Diagnostics.AddError("Failed to create oauth client", err.Error())
		return
	}

	plan.ID = types.StringValue(key.ID)
	plan.Key = types.StringValue(key.Key)
	plan.CreatedAt = types.StringValue(key.Created.Format(time.RFC3339))
	plan.UpdatedAt = types.StringValue(key.Updated.Format(time.RFC3339))
	plan.UserID = types.StringValue(key.UserID)

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *oauthClientResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan oauthClientResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var scopes, tags []string
	resp.Diagnostics.Append(plan.Scopes.ElementsAs(ctx, &scopes, false)...)
	resp.Diagnostics.Append(plan.Tags.ElementsAs(ctx, &tags, false)...)

	key, err := r.Client.Keys().SetOAuthClient(ctx, plan.ID.ValueString(), tailscale.SetOAuthClientRequest{
		Description: plan.Description.ValueString(),
		Scopes:      scopes,
		Tags:        tags,
	})
	if err != nil {
		resp.Diagnostics.AddError("Failed to update oauth client", err.Error())
		return
	}

	plan.UpdatedAt = types.StringValue(key.Updated.Format(time.RFC3339))

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *oauthClientResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state oauthClientResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := r.Client.Keys().Delete(ctx, state.ID.ValueString())
	if err != nil && !tailscale.IsNotFound(err) {
		resp.Diagnostics.AddError("Failed to delete oauth client", err.Error())
	}
}

func (r *oauthClientResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
