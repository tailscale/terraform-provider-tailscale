// Copyright (c) David Bond, Tailscale Inc, & Contributors
// SPDX-License-Identifier: MIT

package tailscale

import (
	"context"
	"fmt"

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
	_ resource.Resource                = &postureIntegrationResource{}
	_ resource.ResourceWithConfigure   = &postureIntegrationResource{}
	_ resource.ResourceWithImportState = &postureIntegrationResource{}
)

type postureIntegrationResourceModel struct {
	ID              types.String `tfsdk:"id"`
	PostureProvider types.String `tfsdk:"posture_provider"`
	CloudID         types.String `tfsdk:"cloud_id"`
	ClientID        types.String `tfsdk:"client_id"`
	TenantID        types.String `tfsdk:"tenant_id"`
	ClientSecret    types.String `tfsdk:"client_secret"`
}

func NewPostureIntegrationResource() resource.Resource {
	return &postureIntegrationResource{}
}

type postureIntegrationResource struct {
	ResourceBase
}

func (p *postureIntegrationResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func (p *postureIntegrationResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_posture_integration"
}

func (p *postureIntegrationResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "The posture_integration resource allows you to manage integrations with device posture data providers. See https://tailscale.com/kb/1288/device-posture for more information.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"posture_provider": schema.StringAttribute{
				Description:   "The third-party provider for posture data. Valid values are `falcon`, `fleet`, `huntress`, `intune`, `jamfpro`, `kandji`, `kolide`, and `sentinelone`.",
				Required:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
				Validators: []validator.String{
					stringvalidator.OneOf(
						string(tailscale.PostureIntegrationProviderFalcon),
						string(tailscale.PostureIntegrationProviderFleet),
						string(tailscale.PostureIntegrationProviderHuntress),
						string(tailscale.PostureIntegrationProviderIntune),
						string(tailscale.PostureIntegrationProviderJamfPro),
						string(tailscale.PostureIntegrationProviderKandji),
						string(tailscale.PostureIntegrationProviderKolide),
						string(tailscale.PostureIntegrationProviderSentinelOne),
					),
				},
			},
			"cloud_id": schema.StringAttribute{
				Description: "Identifies which of the provider's clouds to integrate with.",
				Optional:    true,
			},
			"client_id": schema.StringAttribute{
				Description: "Unique identifier for your client.",
				Optional:    true,
			},
			"tenant_id": schema.StringAttribute{
				Description: "The Microsoft Intune directory (tenant) ID. For other providers, this is left blank.",
				Optional:    true,
			},
			"client_secret": schema.StringAttribute{
				Description: "The secret (auth key, token, etc.) used to authenticate with the provider.",
				Required:    true,
				Sensitive:   true,
			},
		},
	}
}

func (p *postureIntegrationResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state postureIntegrationResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	integration, err := p.Client.DevicePosture().GetIntegration(ctx, state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Failed to fetch posture integration",
			fmt.Sprintf("Error reading posture integration with id %q: %s", state.ID.ValueString(), err.Error()))
		return
	}

	state.ID = types.StringValue(integration.ID)
	state.PostureProvider = types.StringValue(string(integration.Provider))
	state.CloudID = StringValueNullIfEmpty(integration.CloudID)
	state.ClientID = StringValueNullIfEmpty(integration.ClientID)
	state.TenantID = StringValueNullIfEmpty(integration.TenantID)

	diags = resp.State.Set(ctx, state)
	resp.Diagnostics.Append(diags...)
}

func (p *postureIntegrationResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan postureIntegrationResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	integration, err := p.Client.DevicePosture().CreateIntegration(
		ctx,
		tailscale.CreatePostureIntegrationRequest{
			Provider:     tailscale.PostureIntegrationProvider(plan.PostureProvider.ValueString()),
			CloudID:      plan.CloudID.ValueString(),
			ClientID:     plan.ClientID.ValueString(),
			TenantID:     plan.TenantID.ValueString(),
			ClientSecret: plan.ClientSecret.ValueString(),
		},
	)

	if err != nil {
		resp.Diagnostics.AddError("Failed to create posture integration",
			fmt.Sprintf("Error creating posture integration with provider %q: %s", plan.PostureProvider.ValueString(), err.Error()))
		return
	}

	plan.ID = types.StringValue(integration.ID)
	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
}

func (p *postureIntegrationResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan postureIntegrationResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	_, err := p.Client.DevicePosture().UpdateIntegration(
		ctx,
		plan.ID.ValueString(),
		tailscale.UpdatePostureIntegrationRequest{
			CloudID:      plan.CloudID.ValueString(),
			ClientID:     plan.ClientID.ValueString(),
			TenantID:     plan.TenantID.ValueString(),
			ClientSecret: plan.ClientSecret.ValueStringPointer(),
		},
	)
	if err != nil {
		resp.Diagnostics.AddError("Failed to update posture integration",
			fmt.Sprintf("Error updating posture integration with id %q: %s", plan.ID.ValueString(), err.Error()))
		return
	}

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
}

func (p *postureIntegrationResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var plan postureIntegrationResourceModel
	diags := req.State.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := p.Client.DevicePosture().DeleteIntegration(ctx, plan.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Failed to delete posture integration",
			fmt.Sprintf("Error deleting posture integration with id %q: %s", plan.ID.ValueString(), err.Error()))
		return
	}
}
