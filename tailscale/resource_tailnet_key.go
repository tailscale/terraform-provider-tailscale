// Copyright (c) David Bond, Tailscale Inc, & Contributors
// SPDX-License-Identifier: MIT

package tailscale

import (
	"context"
	"fmt"
	"time"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/setplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"tailscale.com/client/tailscale/v2"
)

var (
	_ resource.Resource                = &tailnetKeyResource{}
	_ resource.ResourceWithConfigure   = &tailnetKeyResource{}
	_ resource.ResourceWithModifyPlan  = &tailnetKeyResource{}
	_ resource.ResourceWithImportState = &tailnetKeyResource{}
)

type tailnetKeyResourceModel struct {
	ID                types.String `tfsdk:"id"`
	Reusable          types.Bool   `tfsdk:"reusable"`
	Ephemeral         types.Bool   `tfsdk:"ephemeral"`
	Tags              types.Set    `tfsdk:"tags"`
	Preauthorized     types.Bool   `tfsdk:"preauthorized"`
	Key               types.String `tfsdk:"key"`
	Expiry            types.Int64  `tfsdk:"expiry"`
	CreatedAt         types.String `tfsdk:"created_at"`
	ExpiresAt         types.String `tfsdk:"expires_at"`
	Description       types.String `tfsdk:"description"`
	Invalid           types.Bool   `tfsdk:"invalid"`
	RecreateIfInvalid types.String `tfsdk:"recreate_if_invalid"`
	UserID            types.String `tfsdk:"user_id"`
}

func NewTailnetKeyResource() resource.Resource {
	return &tailnetKeyResource{}
}

type tailnetKeyResource struct {
	ResourceBase
}

func (t *tailnetKeyResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_tailnet_key"
}

func (t *tailnetKeyResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "The tailnet_key resource allows you to create pre-authentication keys that can register new nodes without needing to sign in via a web browser. See https://tailscale.com/kb/1085/auth-keys for more information",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"reusable": schema.BoolAttribute{
				Optional:      true,
				Description:   "Indicates if the key is reusable or single-use. Defaults to `false`.",
				PlanModifiers: []planmodifier.Bool{boolplanmodifier.RequiresReplace()},
			},
			"ephemeral": schema.BoolAttribute{
				Optional:      true,
				Description:   "Indicates if the key is ephemeral. Defaults to `false`.",
				PlanModifiers: []planmodifier.Bool{boolplanmodifier.RequiresReplace()},
			},
			"tags": schema.SetAttribute{
				ElementType:   types.StringType,
				Optional:      true,
				Description:   "List of tags to apply to the machines authenticated by the key.",
				PlanModifiers: []planmodifier.Set{setplanmodifier.RequiresReplace()},
			},
			"preauthorized": schema.BoolAttribute{
				Optional:      true,
				Description:   "Determines whether or not the machines authenticated by the key will be authorized for the tailnet by default. Defaults to `false`.",
				PlanModifiers: []planmodifier.Bool{boolplanmodifier.RequiresReplace()},
			},
			"key": schema.StringAttribute{
				Description:   "The authentication key",
				Computed:      true,
				Sensitive:     true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"expiry": schema.Int64Attribute{
				Optional:      true,
				Computed:      true,
				Description:   "The expiry of the key in seconds. Defaults to `7776000` (90 days).",
				PlanModifiers: []planmodifier.Int64{int64planmodifier.RequiresReplace()},
			},
			"created_at": schema.StringAttribute{
				Description:   "The creation timestamp of the key in RFC3339 format",
				Computed:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"expires_at": schema.StringAttribute{
				Description:   "The expiry timestamp of the key in RFC3339 format",
				Computed:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"description": schema.StringAttribute{
				Optional:      true,
				Description:   "A description of the key consisting of alphanumeric characters. Defaults to `\"\"`.",
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
				Validators: []validator.String{
					stringvalidator.LengthAtMost(50),
				},
			},
			"invalid": schema.BoolAttribute{
				Description:   "Indicates whether the key is invalid (e.g. expired, revoked or has been deleted).",
				Computed:      true,
				PlanModifiers: []planmodifier.Bool{boolplanmodifier.UseStateForUnknown()},
			},
			"recreate_if_invalid": schema.StringAttribute{
				Optional:    true,
				Description: "Determines whether the key should be created again if it becomes invalid. By default, reusable keys will be recreated, but single-use keys will not. Possible values: 'always', 'never'.",
				Validators: []validator.String{
					stringvalidator.OneOf(
						"",
						"always",
						"never",
					),
				},
			},
			"user_id": schema.StringAttribute{
				Optional:      true,
				Computed:      true,
				Description:   "ID of the user who created this key, empty for keys created by OAuth clients.",
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
		},
	}
}

func (t *tailnetKeyResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan tailnetKeyResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	var createKeyRequest tailscale.CreateKeyRequest
	createKeyRequest.Capabilities.Devices.Create.Reusable = plan.Reusable.ValueBool()
	createKeyRequest.Capabilities.Devices.Create.Ephemeral = plan.Ephemeral.ValueBool()

	var tags []string
	diags = plan.Tags.ElementsAs(ctx, &tags, false)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	createKeyRequest.Capabilities.Devices.Create.Tags = tags
	createKeyRequest.Capabilities.Devices.Create.Preauthorized = plan.Preauthorized.ValueBool()
	createKeyRequest.ExpirySeconds = plan.Expiry.ValueInt64()
	createKeyRequest.Description = plan.Description.ValueString()

	key, err := t.Client.Keys().CreateAuthKey(ctx, createKeyRequest)
	if err != nil {
		resp.Diagnostics.AddError("Failed to create key", fmt.Sprintf("Error creating tailnet key: %s", err.Error()))
		return
	}

	plan.ID = types.StringValue(key.ID)
	plan.Key = types.StringValue(key.Key)
	plan.CreatedAt = types.StringValue(key.Created.Format(time.RFC3339))
	plan.ExpiresAt = types.StringValue(key.Expires.Format(time.RFC3339))
	plan.Expiry = types.Int64Value(int64(*key.ExpirySeconds))
	plan.Invalid = types.BoolValue(key.Invalid)
	plan.UserID = types.StringValue(key.UserID)

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
}

func (t *tailnetKeyResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state tailnetKeyResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := t.Client.Keys().Delete(ctx, state.ID.ValueString())
	// Single-use keys may no longer be here, so we can ignore deletions that fail due to not-found errors.
	if err != nil && !tailscale.IsNotFound(err) {
		resp.Diagnostics.AddError("Failed to delete key", fmt.Sprintf("Error deleting tailnet key with id %q: %s", state.ID, err.Error()))
	}
}

// shouldRecreateIfInvalid determines if a resource should be recreated when
// it's invalid, based on the values of `reusable` and `recreate_if_invalid` fields.
// By default, we automatically recreate reusable keys, but ignore invalid single-use
// keys, assuming they have successfully been used, and recreating them might trigger
// unnecessary updates of other Terraform resources that depend on the key.
func shouldRecreateIfInvalid(reusable bool, recreateIfInvalid string) bool {
	if recreateIfInvalid == "always" {
		return true
	}
	if recreateIfInvalid == "never" {
		return false
	}
	return reusable
}

func (t *tailnetKeyResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state tailnetKeyResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	key, err := t.Client.Keys().Get(ctx, state.ID.ValueString())
	if tailscale.IsNotFound(err) {
		state.Invalid = types.BoolValue(true)
	} else if err != nil {
		resp.Diagnostics.AddError("Failed to fetch key", fmt.Sprintf("Error reading tailnet key with id %q: %s", state.ID, err.Error()))
		return
	} else {
		state.Invalid = types.BoolValue(false)
	}

	if key.KeyType != "auth" {
		resp.Diagnostics.AddError(fmt.Sprintf("Invalid key type '%s'", key.KeyType), "Only 'auth' keys are supported by this resource")
		return
	}

	if key.Invalid {
		state.Invalid = types.BoolValue(true)
	}

	state.ID = types.StringValue(key.ID)
	state.Reusable = types.BoolValue(key.Capabilities.Devices.Create.Reusable)
	state.Ephemeral = types.BoolValue(key.Capabilities.Devices.Create.Ephemeral)
	tags, diags := types.SetValueFrom(ctx, types.StringType, key.Capabilities.Devices.Create.Tags)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	state.Tags = tags
	state.Preauthorized = types.BoolValue(key.Capabilities.Devices.Create.Preauthorized)
	state.Key = types.StringValue(key.Key)
	state.Expiry = types.Int64Value(int64(*key.ExpirySeconds))
	state.CreatedAt = types.StringValue(key.Created.Format(time.RFC3339))
	state.ExpiresAt = types.StringValue(key.Expires.Format(time.RFC3339))
	state.Description = types.StringValue(key.Description)
	state.UserID = types.StringValue(key.UserID)

	diags = resp.State.Set(ctx, state)
	resp.Diagnostics.Append(diags...)
}

func (t *tailnetKeyResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var state tailnetKeyResourceModel
	diags := req.Plan.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	key, err := t.Client.Keys().Get(ctx, state.ID.ValueString())
	if tailscale.IsNotFound(err) {
		state.Invalid = types.BoolValue(true)
	} else if err != nil {
		resp.Diagnostics.AddError("Failed to fetch key", fmt.Sprintf("Error reading tailnet key with id %q: %s", state.ID, err.Error()))
		return
	} else {
		state.Invalid = types.BoolValue(false)
	}

	if key.KeyType != "auth" {
		resp.Diagnostics.AddError(fmt.Sprintf("Invalid key type '%s'", key.KeyType), "Only 'auth' keys are supported by this resource")
		return
	}

	if key.Invalid {
		state.Invalid = types.BoolValue(true)
	}

	state.Key = types.StringValue(key.Key)
	state.CreatedAt = types.StringValue(key.Created.Format(time.RFC3339))
	state.ExpiresAt = types.StringValue(key.Expires.Format(time.RFC3339))
	state.Description = types.StringValue(key.Description)
	state.UserID = types.StringValue(key.UserID)

	diags = resp.State.Set(ctx, state)
	resp.Diagnostics.Append(diags...)
}

func (t *tailnetKeyResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func (t *tailnetKeyResource) ModifyPlan(ctx context.Context, req resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse) {
	// Do not replace on resource creation.
	if req.State.Raw.IsNull() {
		return
	}

	// Do not replace on resource destroy.
	if req.Plan.Raw.IsNull() {
		return
	}

	var plan tailnetKeyResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Don't ever need to replace if the key is still valid.
	if !plan.Invalid.ValueBool() {
		return
	}

	if shouldRecreateIfInvalid(plan.Reusable.ValueBool(), plan.RecreateIfInvalid.ValueString()) {
		resp.Plan.SetAttribute(ctx, path.Root("id"), types.StringUnknown())
		resp.RequiresReplace = append(resp.RequiresReplace, path.Root("id"))
	}
}
