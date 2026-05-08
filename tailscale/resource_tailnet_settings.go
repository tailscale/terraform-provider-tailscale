// Copyright (c) David Bond, Tailscale Inc, & Contributors
// SPDX-License-Identifier: MIT

package tailscale

import (
	"context"
	"fmt"
	"regexp"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"tailscale.com/client/tailscale/v2"
)

var (
	_ resource.Resource                = &tailnetSettingsResource{}
	_ resource.ResourceWithConfigure   = &tailnetSettingsResource{}
	_ resource.ResourceWithImportState = &tailnetSettingsResource{}
)

type tailnetSettingsResourceModel struct {
	ID                                    types.String `tfsdk:"id"`
	ACLsExternallyManagedOn               types.Bool   `tfsdk:"acls_externally_managed_on"`
	ACLsExternalLink                      types.String `tfsdk:"acls_external_link"`
	DevicesApprovalOn                     types.Bool   `tfsdk:"devices_approval_on"`
	DevicesAutoUpdatesOn                  types.Bool   `tfsdk:"devices_auto_updates_on"`
	DevicesKeyDurationDays                types.Int64  `tfsdk:"devices_key_duration_days"`
	UsersApprovalOn                       types.Bool   `tfsdk:"users_approval_on"`
	UsersRoleAllowedToJoinExternalTailnet types.String `tfsdk:"users_role_allowed_to_join_external_tailnet"`
	NetworkFlowLoggingOn                  types.Bool   `tfsdk:"network_flow_logging_on"`
	RegionalRoutingOn                     types.Bool   `tfsdk:"regional_routing_on"`
	PostureIdentityCollectionOn           types.Bool   `tfsdk:"posture_identity_collection_on"`
	HTTPSEnabled                          types.Bool   `tfsdk:"https_enabled"`
}

func NewTailnetSettingsResource() resource.Resource {
	return &tailnetSettingsResource{}
}

type tailnetSettingsResource struct {
	ResourceBase
	ResourceImportedByID
}

func (s *tailnetSettingsResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_tailnet_settings"
}

func (s *tailnetSettingsResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "The tailnet_settings resource allows you to configure settings for your tailnet. See https://tailscale.com/api#tag/tailnetsettings for more information.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"acls_externally_managed_on": schema.BoolAttribute{
				Description: "Prevent users from editing policies in the admin console to avoid conflicts with external management workflows like GitOps or Terraform.",
				Optional:    true,
				Computed:    true,
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.UseStateForUnknown(),
				},
			},
			"acls_external_link": schema.StringAttribute{
				Description: "Link to your external ACL definition or management system. Must be a valid URL.",
				Optional:    true,
				Computed:    true,
				Validators: []validator.String{
					stringvalidator.RegexMatches(
						regexp.MustCompile(`^https?://`),
						"must be a valid URL with http or https scheme",
					),
				},
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"devices_approval_on": schema.BoolAttribute{
				Description: "Whether device approval is enabled for the tailnet",
				Optional:    true,
				Computed:    true,
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.UseStateForUnknown(),
				},
			},
			"devices_auto_updates_on": schema.BoolAttribute{
				Description: "Whether auto updates are enabled for devices that belong to this tailnet",
				Optional:    true,
				Computed:    true,
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.UseStateForUnknown(),
				},
			},
			"devices_key_duration_days": schema.Int64Attribute{
				Description: "The key expiry duration for devices on this tailnet",
				Optional:    true,
				Computed:    true,
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.UseStateForUnknown(),
				},
			},
			"users_approval_on": schema.BoolAttribute{
				Description: "Whether user approval is enabled for this tailnet",
				Optional:    true,
				Computed:    true,
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.UseStateForUnknown(),
				},
			},
			"users_role_allowed_to_join_external_tailnet": schema.StringAttribute{
				Description: "Which user roles are allowed to join external tailnets",
				Optional:    true,
				Computed:    true,
				Validators: []validator.String{
					stringvalidator.OneOf(
						string(tailscale.RoleAllowedToJoinExternalTailnetsNone),
						string(tailscale.RoleAllowedToJoinExternalTailnetsMember),
						string(tailscale.RoleAllowedToJoinExternalTailnetsAdmin),
					),
				},
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"network_flow_logging_on": schema.BoolAttribute{
				Description: "Whether network flow logs are enabled for the tailnet",
				Optional:    true,
				Computed:    true,
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.UseStateForUnknown(),
				},
			},
			"regional_routing_on": schema.BoolAttribute{
				Description: "Whether regional routing is enabled for the tailnet",
				Optional:    true,
				Computed:    true,
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.UseStateForUnknown(),
				},
			},
			"posture_identity_collection_on": schema.BoolAttribute{
				Description: "Whether identity collection is enabled for device posture integrations for the tailnet",
				Optional:    true,
				Computed:    true,
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.UseStateForUnknown(),
				},
			},
			"https_enabled": schema.BoolAttribute{
				Description: "Whether provisioning of HTTPS certificates is enabled for the tailnet",
				Optional:    true,
				Computed:    true,
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.UseStateForUnknown(),
				},
			},
		},
	}
}

func (s *tailnetSettingsResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state tailnetSettingsResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := s.readSettings(ctx, &state)
	if err != nil {
		resp.Diagnostics.AddError("Failed to fetch tailnet settings", fmt.Sprintf("Error reading tailnet settings: %s", err))
		return
	}

	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
}

func (s *tailnetSettingsResource) readSettings(ctx context.Context, state *tailnetSettingsResourceModel) error {
	settings, err := s.Client.TailnetSettings().Get(ctx)
	if err != nil {
		return err
	}

	state.ID = types.StringValue("singleton")
	state.ACLsExternallyManagedOn = types.BoolValue(settings.ACLsExternallyManagedOn)
	state.ACLsExternalLink = types.StringValue(settings.ACLsExternalLink)
	state.DevicesApprovalOn = types.BoolValue(settings.DevicesApprovalOn)
	state.DevicesAutoUpdatesOn = types.BoolValue(settings.DevicesAutoUpdatesOn)
	state.DevicesKeyDurationDays = types.Int64Value(int64(settings.DevicesKeyDurationDays))
	state.UsersApprovalOn = types.BoolValue(settings.UsersApprovalOn)
	state.UsersRoleAllowedToJoinExternalTailnet = types.StringValue(string(settings.UsersRoleAllowedToJoinExternalTailnets))
	state.NetworkFlowLoggingOn = types.BoolValue(settings.NetworkFlowLoggingOn)
	state.RegionalRoutingOn = types.BoolValue(settings.RegionalRoutingOn)
	state.PostureIdentityCollectionOn = types.BoolValue(settings.PostureIdentityCollectionOn)
	state.HTTPSEnabled = types.BoolValue(settings.HTTPSEnabled)

	return nil
}

func (s *tailnetSettingsResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan tailnetSettingsResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := s.resourceTailnetSettingsDoCreate(ctx, plan); err != nil {
		resp.Diagnostics.AddError("Failed to update tailnet settings", fmt.Sprintf("Error updating tailnet settings: %s", err))
		return
	}

	err := s.readSettings(ctx, &plan)
	if err != nil {
		resp.Diagnostics.AddError("Failed to fetch tailnet settings", fmt.Sprintf("Error reading tailnet settings: %s", err))
		return
	}

	diags = resp.State.Set(ctx, &plan)
	resp.Diagnostics.Append(diags...)
}

func BoolNullPointerIfUnknown(b types.Bool) *bool {
	if b.IsUnknown() {
		return nil
	}
	return b.ValueBoolPointer()
}

func BoolNullPointerIfSame(plan types.Bool, state types.Bool) *bool {
	if plan.Equal(state) {
		return nil
	}
	return plan.ValueBoolPointer()
}

func StringNullPointerIfUnknown(s types.String) *string {
	if s.IsUnknown() {
		return nil
	}
	return s.ValueStringPointer()
}

func StringNullPointerIfSame(plan types.String, state types.String) *string {
	if plan.Equal(state) {
		return nil
	}
	return plan.ValueStringPointer()
}

func Int64ToIntNullPointerIfUnknown(i types.Int64) *int {
	if i.IsUnknown() || i.IsNull() {
		return nil
	}
	v := int(i.ValueInt64())
	return &v
}

func Int64ToIntNullPointerIfSame(plan types.Int64, state types.Int64) *int {
	if plan.Equal(state) {
		return nil
	}
	v := int(plan.ValueInt64())
	return &v
}

func (s *tailnetSettingsResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan tailnetSettingsResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	var state tailnetSettingsResourceModel
	diags = req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := s.resourceTailnetSettingsDoUpdate(ctx, plan, state); err != nil {
		resp.Diagnostics.AddError("Failed to update tailnet settings", fmt.Sprintf("Error updating tailnet settings: %s", err))
		return
	}

	if err := s.readSettings(ctx, &plan); err != nil {
		resp.Diagnostics.AddError("Failed to fetch tailnet settings", fmt.Sprintf("Error reading tailnet settings: %s", err))
		return
	}

	diags = resp.State.Set(ctx, &plan)
	resp.Diagnostics.Append(diags...)
}

func (s *tailnetSettingsResource) resourceTailnetSettingsDoCreate(ctx context.Context, plan tailnetSettingsResourceModel) error {
	settingsRequest := tailscale.UpdateTailnetSettingsRequest{}

	settingsRequest.ACLsExternallyManagedOn = BoolNullPointerIfUnknown(plan.ACLsExternallyManagedOn)
	settingsRequest.ACLsExternalLink = StringNullPointerIfUnknown(plan.ACLsExternalLink)
	settingsRequest.DevicesApprovalOn = BoolNullPointerIfUnknown(plan.DevicesApprovalOn)
	settingsRequest.DevicesAutoUpdatesOn = BoolNullPointerIfUnknown(plan.DevicesAutoUpdatesOn)
	settingsRequest.DevicesKeyDurationDays = Int64ToIntNullPointerIfUnknown(plan.DevicesKeyDurationDays)
	settingsRequest.UsersApprovalOn = BoolNullPointerIfUnknown(plan.UsersApprovalOn)
	settingsRequest.UsersRoleAllowedToJoinExternalTailnets = (*tailscale.RoleAllowedToJoinExternalTailnets)(StringNullPointerIfUnknown(plan.UsersRoleAllowedToJoinExternalTailnet))
	settingsRequest.NetworkFlowLoggingOn = BoolNullPointerIfUnknown(plan.NetworkFlowLoggingOn)
	settingsRequest.RegionalRoutingOn = BoolNullPointerIfUnknown(plan.RegionalRoutingOn)
	settingsRequest.PostureIdentityCollectionOn = BoolNullPointerIfUnknown(plan.PostureIdentityCollectionOn)
	settingsRequest.HTTPSEnabled = BoolNullPointerIfUnknown(plan.HTTPSEnabled)

	// panic(fmt.Sprintf("settingsRequest: %+v", settingsRequest)) // easy way to see what actual request will be made

	return s.Client.TailnetSettings().Update(ctx, settingsRequest)
}

func (s *tailnetSettingsResource) resourceTailnetSettingsDoUpdate(ctx context.Context, plan tailnetSettingsResourceModel, state tailnetSettingsResourceModel) error {
	settingsRequest := tailscale.UpdateTailnetSettingsRequest{}

	settingsRequest.ACLsExternallyManagedOn = BoolNullPointerIfSame(plan.ACLsExternallyManagedOn, state.ACLsExternallyManagedOn)
	settingsRequest.ACLsExternalLink = StringNullPointerIfSame(plan.ACLsExternalLink, state.ACLsExternalLink)
	settingsRequest.DevicesApprovalOn = BoolNullPointerIfSame(plan.DevicesApprovalOn, state.DevicesApprovalOn)
	settingsRequest.DevicesAutoUpdatesOn = BoolNullPointerIfSame(plan.DevicesAutoUpdatesOn, state.DevicesAutoUpdatesOn)
	settingsRequest.DevicesKeyDurationDays = Int64ToIntNullPointerIfSame(plan.DevicesKeyDurationDays, state.DevicesKeyDurationDays)
	settingsRequest.UsersApprovalOn = BoolNullPointerIfSame(plan.UsersApprovalOn, state.UsersApprovalOn)
	settingsRequest.UsersRoleAllowedToJoinExternalTailnets = (*tailscale.RoleAllowedToJoinExternalTailnets)(StringNullPointerIfSame(plan.UsersRoleAllowedToJoinExternalTailnet, state.UsersRoleAllowedToJoinExternalTailnet))
	settingsRequest.NetworkFlowLoggingOn = BoolNullPointerIfSame(plan.NetworkFlowLoggingOn, state.NetworkFlowLoggingOn)
	settingsRequest.RegionalRoutingOn = BoolNullPointerIfSame(plan.RegionalRoutingOn, state.RegionalRoutingOn)
	settingsRequest.PostureIdentityCollectionOn = BoolNullPointerIfSame(plan.PostureIdentityCollectionOn, state.PostureIdentityCollectionOn)
	settingsRequest.HTTPSEnabled = BoolNullPointerIfSame(plan.HTTPSEnabled, state.HTTPSEnabled)

	// panic(fmt.Sprintf("settingsRequest: %+v", settingsRequest)) // easy way to see what actual request will be made

	return s.Client.TailnetSettings().Update(ctx, settingsRequest)
}

func (s *tailnetSettingsResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	// We don't know what the default values for Tailnet settings should be, so deleting is a noop.

	// TODO: Add a plan modifier to tell users that a delete is doing nothing
}
