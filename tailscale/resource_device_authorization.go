// Copyright (c) David Bond, Tailscale Inc, & Contributors
// SPDX-License-Identifier: MIT

package tailscale

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"tailscale.com/client/tailscale/v2"
)

var (
	_ resource.Resource                = &deviceAuthorizationResource{}
	_ resource.ResourceWithConfigure   = &deviceAuthorizationResource{}
	_ resource.ResourceWithImportState = &deviceAuthorizationResource{}
	_ resource.ResourceWithModifyPlan  = &deviceAuthorizationResource{}
)

type deviceAuthorizationResourceModel struct {
	ID              types.String `tfsdk:"id"`
	DeviceID        types.String `tfsdk:"device_id"`
	Authorized      types.Bool   `tfsdk:"authorized"`
	DeleteOnDestroy types.Bool   `tfsdk:"delete_on_destroy"`
}

// NewDeviceAuthorizationResource returns a new device authorization resource.
func NewDeviceAuthorizationResource() resource.Resource {
	return &deviceAuthorizationResource{}
}

type deviceAuthorizationResource struct {
	ResourceBase
	ResourceImportedByID
}

func (d deviceAuthorizationResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_device_authorization"
}

func (d deviceAuthorizationResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "The device_authorization resource is used to approve new devices before they can join the tailnet. See https://tailscale.com/kb/1099/device-authorization/ for more details.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"device_id": schema.StringAttribute{
				Required:    true,
				Description: "The device to set as authorized",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				}},
			"authorized": schema.BoolAttribute{
				Required:    true,
				Description: "Whether or not the device is authorized",
			},
			"delete_on_destroy": schema.BoolAttribute{
				Optional: true,
				Computed: true,
				Default:  booldefault.StaticBool(false),
				Description: "Whether to delete the device from the tailnet when this resource is destroyed. " +
					"Defaults to `false`, in which case destroying this resource only removes it from the Terraform state. " +
					"Terraform destroys a resource using the values recorded in its state, so this must be set to `true` " +
					"and applied before the resource is removed from the configuration or destroyed. " +
					"Changing `device_id` forces replacement, which destroys the resource, so with this set to `true` " +
					"the previous device is deleted.",
			},
		},
	}
}

func (d deviceAuthorizationResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state deviceAuthorizationResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	deviceID := state.ID.ValueString()

	device, err := d.Client.Devices().Get(ctx, deviceID)
	if err != nil {
		// If the device is not found, remove from the state so we can create it again.
		if tailscale.IsNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}

		resp.Diagnostics.AddError(
			"Failed to fetch device authorization",
			"Failed to fetch authorization for device with with ID "+deviceID+": "+err.Error(),
		)
		return
	}

	// If the device lookup succeeds and the state ID is not the same as the legacy ID, we can assume the ID is the node ID.
	canonicalDeviceID := device.ID
	if device.ID != deviceID {
		canonicalDeviceID = device.NodeID
	}
	state.ID = types.StringValue(canonicalDeviceID)
	state.DeviceID = types.StringValue(canonicalDeviceID)
	state.Authorized = types.BoolValue(device.Authorized)

	// delete_on_destroy only exists in Terraform, so the API cannot tell us its value.
	// After an import or a provider upgrade it is null; set the default so the plan stays empty.
	if state.DeleteOnDestroy.IsNull() || state.DeleteOnDestroy.IsUnknown() {
		state.DeleteOnDestroy = types.BoolValue(false)
	}

	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
}

func (d deviceAuthorizationResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan deviceAuthorizationResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	deviceID := plan.DeviceID.ValueString()
	authorized := plan.Authorized.ValueBool()

	if err := d.Client.Devices().SetAuthorized(ctx, deviceID, authorized); err != nil {
		resp.Diagnostics.AddError(
			"Failed to update device authorization",
			"Failed to update authorization for device with ID "+deviceID+": "+err.Error(),
		)
		return
	}

	plan.ID = types.StringValue(deviceID)
	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
}

func (d deviceAuthorizationResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state deviceAuthorizationResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	deviceID := plan.DeviceID.ValueString()

	authorized := plan.Authorized.ValueBool()

	// A change to delete_on_destroy alone only needs to be stored in state.
	if !plan.Authorized.Equal(state.Authorized) {
		if err := d.Client.Devices().SetAuthorized(ctx, deviceID, authorized); err != nil {
			resp.Diagnostics.AddError(
				"Failed to update device authorization",
				"Failed to update authorization for device with ID "+deviceID+": "+err.Error(),
			)
			return
		}
	}

	diags := resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
}

func (d deviceAuthorizationResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	// Terraform destroys a resource using its prior state, because the configuration
	// may no longer exist, so delete_on_destroy is read from state.
	var state deviceAuthorizationResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if !state.DeleteOnDestroy.ValueBool() {
		// Historically it used to not be possible to de-authorize (un-approve) a device via the API and
		// the destroy operation therefore used to be a no-op as far as the remote state was concerned.
		// For backwards-compatibility, this remains the default unless delete_on_destroy is set.
		return
	}

	deviceID := state.DeviceID.ValueString()

	// A device that no longer exists has already been deleted.
	if err := d.Client.Devices().Delete(ctx, deviceID); err != nil && !tailscale.IsNotFound(err) {
		resp.Diagnostics.AddError(
			"Failed to delete device",
			"Failed to delete device with ID "+deviceID+": "+err.Error(),
		)
	}
}

func (d deviceAuthorizationResource) ModifyPlan(ctx context.Context, req resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse) {
	// Nothing to warn about when the resource is being created.
	if req.State.Raw.IsNull() {
		return
	}

	var state deviceAuthorizationResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	deleteOnDestroy := state.DeleteOnDestroy.ValueBool()

	// If the entire plan is null, the resource is planned for destruction.
	if req.Plan.Raw.IsNull() {
		if deleteOnDestroy {
			resp.Diagnostics.AddWarning(
				"Resource Destruction Considerations",
				"Applying this resource destruction will delete the device "+state.DeviceID.ValueString()+
					" from the tailnet, because delete_on_destroy is true.",
			)
			return
		}

		resp.Diagnostics.AddWarning(
			"Resource Destruction Considerations",
			"Applying this resource destruction will only remove the resource from the Terraform state and "+
				"will not modify the device's authorization. ",
		)
		return
	}

	if !deleteOnDestroy {
		return
	}

	// Changing device_id forces replacement, and the destroy half of a replacement
	// deletes the previous device when delete_on_destroy is true in state.
	var planDeviceID types.String
	resp.Diagnostics.Append(req.Plan.GetAttribute(ctx, path.Root("device_id"), &planDeviceID)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if planDeviceID.IsUnknown() || !planDeviceID.Equal(state.DeviceID) {
		resp.Diagnostics.AddWarning(
			"Resource Replacement Considerations",
			"Changing device_id replaces this resource. Because delete_on_destroy is true, the replacement will "+
				"delete the device "+state.DeviceID.ValueString()+" from the tailnet. To keep the device, set "+
				"delete_on_destroy to false and apply that before changing device_id.",
		)
	}
}
