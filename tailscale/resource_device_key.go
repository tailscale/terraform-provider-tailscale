// Copyright (c) David Bond, Tailscale Inc, & Contributors
// SPDX-License-Identifier: MIT

package tailscale

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"tailscale.com/client/tailscale/v2"
)

type deviceKeyResourceModel struct {
	ID                types.String `tfsdk:"id"`
	DeviceID          types.String `tfsdk:"device_id"`
	KeyExpiryDisabled types.Bool   `tfsdk:"key_expiry_disabled"`
}

// NewDeviceKeyResource returns a new device key resource.
func NewDeviceKeyResource() resource.Resource {
	return &deviceKeyResource{}
}

type deviceKeyResource struct {
	ResourceBase
}

func (d deviceKeyResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_device_key"
}

func (d deviceKeyResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "The device_key resource allows you to update the properties of a device's key",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed: true,
			},
			"device_id": schema.StringAttribute{
				Required:    true,
				Description: "The device to update the key properties of",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"key_expiry_disabled": schema.BoolAttribute{
				Optional:    true,
				Description: "Determines whether or not the device's key will expire. Defaults to `false`.",
			},
		},
	}
}

func (d deviceKeyResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan deviceKeyResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	deviceID := plan.DeviceID.ValueString()
	keyExpiryDisabled := plan.KeyExpiryDisabled.ValueBool()

	key := tailscale.DeviceKey{
		KeyExpiryDisabled: keyExpiryDisabled,
	}

	if err := d.Client.Devices().SetKey(ctx, deviceID, key); err != nil {
		resp.Diagnostics.AddError(
			"Failed to update device key",
			"Failed to update key for device with ID "+deviceID+": "+err.Error(),
		)
		return
	}

	plan.ID = types.StringValue(deviceID)
	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
}

func (d deviceKeyResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state deviceKeyResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	deviceID := state.DeviceID.ValueString()
	key := tailscale.DeviceKey{}

	if err := d.Client.Devices().SetKey(ctx, deviceID, key); err != nil {
		resp.Diagnostics.AddError(
			"Failed to update device key",
			"Failed to update key for device with ID "+deviceID+": "+err.Error(),
		)
		return
	}
}

func (d deviceKeyResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state deviceKeyResourceModel
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
			"Failed to fetch device key",
			"Failed to fetch key for device with ID "+deviceID+": "+err.Error(),
		)
		return
	}

	// If the device lookup succeeds and the state ID is not the same as the legacy ID, we can assume the ID is the node ID.
	canonicalDeviceID := device.ID
	if device.ID != deviceID {
		canonicalDeviceID = device.NodeID
	}

	state.DeviceID = types.StringValue(canonicalDeviceID)
	state.KeyExpiryDisabled = types.BoolValue(device.KeyExpiryDisabled)

	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
}

func (d deviceKeyResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan deviceKeyResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	deviceID := plan.DeviceID.ValueString()
	keyExpiryDisabled := plan.KeyExpiryDisabled.ValueBool()

	key := tailscale.DeviceKey{
		KeyExpiryDisabled: keyExpiryDisabled,
	}

	if err := d.Client.Devices().SetKey(ctx, deviceID, key); err != nil {
		resp.Diagnostics.AddError(
			"Failed to update device key",
			"Failed to update key for device with ID "+deviceID+": "+err.Error(),
		)
		return
	}

	plan.ID = types.StringValue(deviceID)
	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
}

func (d deviceKeyResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
