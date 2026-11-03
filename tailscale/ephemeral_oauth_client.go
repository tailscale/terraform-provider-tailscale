// Copyright (c) David Bond, Tailscale Inc, & Contributors
// SPDX-License-Identifier: MIT

package tailscale

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/ephemeral"
	"github.com/hashicorp/terraform-plugin-framework/ephemeral/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"tailscale.com/client/tailscale/v2"
)

var (
	_ ephemeral.EphemeralResource              = &oauthClientEphemeralResource{}
	_ ephemeral.EphemeralResourceWithConfigure = &oauthClientEphemeralResource{}
	_ ephemeral.EphemeralResourceWithClose     = &oauthClientEphemeralResource{}
)

type oauthClientEphemeralModel struct {
	Description types.String `tfsdk:"description"`
	Scopes      types.Set    `tfsdk:"scopes"`
	Tags        types.Set    `tfsdk:"tags"`
	ID          types.String `tfsdk:"id"`
	Key         types.String `tfsdk:"key"`
	CreatedAt   types.String `tfsdk:"created_at"`
	UpdatedAt   types.String `tfsdk:"updated_at"`
	UserID      types.String `tfsdk:"user_id"`
}

func NewOAuthClientEphemeralResource() ephemeral.EphemeralResource {
	return &oauthClientEphemeralResource{}
}

type oauthClientEphemeralResource struct {
	EphemeralResourceBase
}

func (r *oauthClientEphemeralResource) Metadata(_ context.Context, req ephemeral.MetadataRequest, resp *ephemeral.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_oauth_client"
}

func (r *oauthClientEphemeralResource) Schema(_ context.Context, _ ephemeral.SchemaRequest, resp *ephemeral.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "The oauth_client ephemeral resource creates an OAuth client for temporary use. The client and its secret are deleted when Terraform closes the ephemeral resource. Requires Terraform 1.10 or later.",
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
				Description: "Tags that access tokens generated for the OAuth client will be able to assign to devices. Mandatory if scopes include `devices:core` or `auth_keys`.",
			},
			"id": schema.StringAttribute{
				Computed:    true,
				Description: "The OAuth client ID.",
			},
			"key": schema.StringAttribute{
				Computed:    true,
				Sensitive:   true,
				Description: "The OAuth client secret. This value is ephemeral and is not stored in Terraform plan or state.",
			},
			"created_at": schema.StringAttribute{
				Computed:    true,
				Description: "The creation timestamp in RFC3339 format.",
			},
			"updated_at": schema.StringAttribute{
				Computed:    true,
				Description: "The updated timestamp in RFC3339 format.",
			},
			"user_id": schema.StringAttribute{
				Computed:    true,
				Description: "ID of the user who created this OAuth client.",
			},
		},
	}
}

func (r *oauthClientEphemeralResource) Open(ctx context.Context, req ephemeral.OpenRequest, resp *ephemeral.OpenResponse) {
	var plan oauthClientEphemeralModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if plan.Description.IsUnknown() || plan.Scopes.IsUnknown() || plan.Tags.IsUnknown() {
		resp.Diagnostics.AddError("Unknown OAuth client configuration", "All OAuth client configuration values must be known when the ephemeral resource is opened.")
		return
	}
	var scopes, tags []string
	resp.Diagnostics.Append(plan.Scopes.ElementsAs(ctx, &scopes, false)...)
	if !plan.Tags.IsNull() {
		resp.Diagnostics.Append(plan.Tags.ElementsAs(ctx, &tags, false)...)
	}
	if resp.Diagnostics.HasError() {
		return
	}

	key, err := r.Client.Keys().CreateOAuthClient(ctx, tailscale.CreateOAuthClientRequest{
		Description: plan.Description.ValueString(),
		Scopes:      scopes,
		Tags:        tags,
	})
	if err != nil {
		resp.Diagnostics.AddError("Failed to create OAuth client", fmt.Sprintf("Error creating ephemeral OAuth client: %s", err))
		return
	}

	if key.ID == "" || key.Key == "" {
		if key.ID != "" {
			if deleteErr := r.Client.Keys().Delete(ctx, key.ID); deleteErr != nil && !tailscale.IsNotFound(deleteErr) {
				resp.Diagnostics.AddError("Failed to delete incomplete OAuth client", fmt.Sprintf("Error deleting incomplete ephemeral OAuth client with id %q: %s", key.ID, deleteErr))
			}
		}
		resp.Diagnostics.AddError("Incomplete OAuth client response", "The Tailscale API did not return both an ID and a client secret.")
		return
	}

	plan.ID = types.StringValue(key.ID)
	plan.Key = types.StringValue(key.Key)
	plan.CreatedAt = types.StringValue(key.Created.Format(time.RFC3339))
	plan.UpdatedAt = types.StringValue(key.Updated.Format(time.RFC3339))
	plan.UserID = types.StringValue(key.UserID)

	privateClientID, err := json.Marshal(key.ID)
	if err != nil {
		resp.Diagnostics.AddError("Failed to encode OAuth client ID", fmt.Sprintf("Error encoding ephemeral OAuth client ID: %s", err))
		if deleteErr := r.Client.Keys().Delete(ctx, key.ID); deleteErr != nil && !tailscale.IsNotFound(deleteErr) {
			resp.Diagnostics.AddError("Failed to delete OAuth client after private state error", fmt.Sprintf("Error deleting ephemeral OAuth client with id %q: %s", key.ID, deleteErr))
		}
		return
	}

	resp.Diagnostics.Append(resp.Private.SetKey(ctx, "client_id", privateClientID)...)
	if resp.Diagnostics.HasError() {
		if deleteErr := r.Client.Keys().Delete(ctx, key.ID); deleteErr != nil && !tailscale.IsNotFound(deleteErr) {
			resp.Diagnostics.AddError("Failed to delete OAuth client after private state error", fmt.Sprintf("Error deleting ephemeral OAuth client with id %q: %s", key.ID, deleteErr))
		}
		return
	}

	resp.Diagnostics.Append(resp.Result.Set(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		if deleteErr := r.Client.Keys().Delete(ctx, key.ID); deleteErr != nil && !tailscale.IsNotFound(deleteErr) {
			resp.Diagnostics.AddError("Failed to delete OAuth client after result error", fmt.Sprintf("Error deleting ephemeral OAuth client with id %q: %s", key.ID, deleteErr))
		}
	}
}

func (r *oauthClientEphemeralResource) Close(ctx context.Context, req ephemeral.CloseRequest, resp *ephemeral.CloseResponse) {
	if req.Private == nil {
		resp.Diagnostics.AddError("Missing ephemeral resource private state", "Cannot delete the OAuth client because its private ID was not provided.")
		return
	}

	privateClientID, diags := req.Private.GetKey(ctx, "client_id")
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	var clientID string
	if err := json.Unmarshal(privateClientID, &clientID); err != nil {
		resp.Diagnostics.AddError("Invalid OAuth client ID", fmt.Sprintf("Unable to decode the private OAuth client ID: %s", err))
		return
	}
	if clientID == "" {
		resp.Diagnostics.AddError("Missing OAuth client ID", "Cannot delete the OAuth client because its private ID is empty.")
		return
	}

	err := r.Client.Keys().Delete(ctx, clientID)
	if err != nil && !tailscale.IsNotFound(err) {
		resp.Diagnostics.AddError("Failed to delete OAuth client", fmt.Sprintf("Error deleting ephemeral OAuth client with id %q: %s", clientID, err))
	}
}
