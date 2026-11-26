// Copyright (c) David Bond, Tailscale Inc, & Contributors
// SPDX-License-Identifier: MIT

package tailscale

import (
	"context"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"tailscale.com/client/tailscale/v2"
)

var (
	_ resource.Resource              = &tailnetResource{}
	_ resource.ResourceWithConfigure = &tailnetResource{}
)

type tailnetResourceModel struct {
	ID                types.String `tfsdk:"id"`
	DisplayName       types.String `tfsdk:"display_name"`
	OrgID             types.String `tfsdk:"org_id"`
	DNSName           types.String `tfsdk:"dns_name"`
	CreatedAt         types.String `tfsdk:"created_at"`
	OAuthClientID     types.String `tfsdk:"oauth_client_id"`
	OAuthClientSecret types.String `tfsdk:"oauth_client_secret"`
}

func NewTailnetResource() resource.Resource {
	return &tailnetResource{}
}

type tailnetResource struct {
	ResourceBase
}

func (r *tailnetResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_tailnet"
}

func (r *tailnetResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "The tailnet resource allows you to create and delete API-only tailnets in an organization using the alpha tailnet creation API. Provider OAuth credentials require the tailnets scope. Import is not supported because deletion requires the OAuth credentials returned at creation.",
		Attributes: map[string]schema.Attribute{
			"display_name": schema.StringAttribute{
				Description: "A custom display name for the API-only tailnet. The name must be unique within your organization. Changing this value replaces the tailnet.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"id": schema.StringAttribute{
				Description: "The stable, globally unique identifier for the tailnet.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"org_id": schema.StringAttribute{
				Description: "The stable, globally unique identifier for the organization.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"dns_name": schema.StringAttribute{
				Description: "The DNS suffix used to construct fully qualified device names in this tailnet.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"created_at": schema.StringAttribute{
				Description: "The creation timestamp of the tailnet in RFC3339 format.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"oauth_client_id": schema.StringAttribute{
				Description: "The OAuth client ID returned when the API-only tailnet was created.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"oauth_client_secret": schema.StringAttribute{
				Description: "The OAuth client secret returned when the API-only tailnet was created. This is stored in Terraform state because the delete API requires tailnet-specific OAuth credentials.",
				Computed:    true,
				Sensitive:   true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
		},
	}
}

func (r *tailnetResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan tailnetResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tailnet, err := r.Client.Tailnets().Create(ctx, tailscale.CreateTailnetRequest{
		DisplayName: plan.DisplayName.ValueString(),
	})
	if err != nil {
		resp.Diagnostics.AddError("Failed to create tailnet", err.Error())
		return
	}
	r.updateStateFromTailnet(&plan, tailnet)
	plan.DNSName = StringValueNullIfEmpty(tailnet.DNSName)
	plan.OAuthClientID = types.StringNull()
	plan.OAuthClientSecret = types.StringNull()
	if tailnet.OAuthClient != nil {
		plan.OAuthClientID = StringValueNullIfEmpty(tailnet.OAuthClient.ID)
		plan.OAuthClientSecret = StringValueNullIfEmpty(tailnet.OAuthClient.Secret)
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
	if plan.OAuthClientID.IsNull() || plan.OAuthClientSecret.IsNull() {
		resp.Diagnostics.AddError("Missing tailnet OAuth credentials", "The tailnet was created, but Tailscale did not return complete OAuth credentials. Its identity has been saved in state. Terraform cannot delete it without those credentials; recover access through the creating tailnet or contact Tailscale support to delete it before retrying.")
	}
}

func (r *tailnetResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state tailnetResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tailnet, err := r.findTailnet(ctx, state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Failed to fetch tailnet", err.Error())
		return
	}
	if tailnet == nil {
		resp.State.RemoveResource(ctx)
		return
	}

	r.updateStateFromTailnet(&state, tailnet)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *tailnetResource) Update(ctx context.Context, _ resource.UpdateRequest, resp *resource.UpdateResponse) {
	resp.Diagnostics.AddError("Cannot update tailnet", "Tailnets do not support in-place updates. Terraform should have planned a replacement.")
}

func (r *tailnetResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state tailnetResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if state.OAuthClientID.ValueString() == "" || state.OAuthClientSecret.ValueString() == "" {
		resp.Diagnostics.AddError("Missing tailnet OAuth credentials", "Terraform cannot delete this tailnet because its creation OAuth credentials are missing from state. Recover access through the creating tailnet or contact Tailscale support to delete it, then refresh Terraform state.")
		return
	}

	deleteClient := tailscale.Client{
		BaseURL:   r.Client.BaseURL,
		UserAgent: r.Client.UserAgent,
		Tailnet:   state.ID.ValueString(),
		Auth: &tailscale.OAuth{
			ClientID:     state.OAuthClientID.ValueString(),
			ClientSecret: state.OAuthClientSecret.ValueString(),
			Scopes:       []string{"all"},
		},
	}

	err := deleteClient.Tailnets().Delete(ctx)
	if err != nil && !tailscale.IsNotFound(err) {
		resp.Diagnostics.AddError("Failed to delete tailnet", err.Error())
	}
}

func (r *tailnetResource) findTailnet(ctx context.Context, id string) (*tailscale.Tailnet, error) {
	tailnets, err := listTailnets(ctx, r.Client)
	if err != nil {
		return nil, err
	}
	for _, tailnet := range tailnets {
		if tailnet.ID == id {
			return &tailnet, nil
		}
	}
	return nil, nil
}

func (r *tailnetResource) updateStateFromTailnet(state *tailnetResourceModel, tailnet *tailscale.Tailnet) {
	state.ID = types.StringValue(tailnet.ID)
	state.DisplayName = types.StringValue(tailnet.DisplayName)
	state.OrgID = types.StringValue(tailnet.OrgID)
	state.CreatedAt = types.StringValue(tailnet.CreatedAt.Format(time.RFC3339))
}
