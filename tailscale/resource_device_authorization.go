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
)

var (
	_ resource.Resource                = &deviceAuthorizationResource{}
	_ resource.ResourceWithImportState = &deviceAuthorizationResource{}
)

type deviceAuthorizationResourceModel struct {
	ID         types.String `tfsdk:"id"`
	DeviceID   types.String `tfsdk:"device_id"`
	Authorized types.Bool   `tfsdk:"authorized"`
}

// NewDeviceAuthorizationResource returns a new device authorization resource.
func NewDeviceAuthorizationResource() resource.Resource {
	return &deviceAuthorizationResource{}
}

type deviceAuthorizationResource struct {
	ResourceBase
}

func (d deviceAuthorizationResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
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
		resp.Diagnostics.AddError(
			"Failed to fetch device",
			"Could not read device authorization for device with ID "+deviceID+": "+err.Error(),
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

	if authorized {
		if err := d.Client.Devices().SetAuthorized(ctx, deviceID, true); err != nil {
			resp.Diagnostics.AddError(
				"Failed to authorize device",
				"Could not authorize device with ID "+deviceID+": "+err.Error(),
			)
		}
	}

	plan.ID = types.StringValue(deviceID)
	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
}

func (d deviceAuthorizationResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan deviceAuthorizationResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	deviceID := plan.DeviceID.ValueString()

	device, err := d.Client.Devices().Get(ctx, deviceID)
	if err != nil {
		resp.Diagnostics.AddError(
			"Failed to fetch device",
			"Could not fetch device with device with ID"+deviceID+": "+err.Error(),
		)
		return
	}

	// Currently, the Tailscale API only supports authorizing a device, but not un-authorizing one. So if the device
	// data from the API states it is authorized then we can't do anything else here.
	if device.Authorized {
		plan.Authorized = types.BoolValue(true)
		return
	}

	if err = d.Client.Devices().SetAuthorized(ctx, deviceID, true); err != nil {
		resp.Diagnostics.AddError(
			"Failed to authorize device",
			"Could not authorize device with ID"+deviceID+": "+err.Error(),
		)
		return
	}
	plan.Authorized = types.BoolValue(true)

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
}

func (d deviceAuthorizationResource) Delete(_ context.Context, _ resource.DeleteRequest, _ *resource.DeleteResponse) {
	// Since authorization cannot be removed at this point, deleting the resource will do nothing.
	return
}
