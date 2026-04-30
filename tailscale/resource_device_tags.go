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
	_ resource.Resource                = &deviceTagsResource{}
	_ resource.ResourceWithImportState = &deviceTagsResource{}
)

type deviceTagsResourceModel struct {
	ID       types.String `tfsdk:"id"`
	DeviceID types.String `tfsdk:"device_id"`
	Tags     types.Set    `tfsdk:"tags"`
}

// NewDeviceTagsResource returns a new device tags resource.
func NewDeviceTagsResource() resource.Resource {
	return &deviceTagsResource{}
}

type deviceTagsResource struct {
	ResourceBase
}

func (d deviceTagsResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func (d deviceTagsResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_device_tags"
}

func (d deviceTagsResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "The device_tags resource is used to apply tags to Tailscale devices. See https://tailscale.com/kb/1068/acl-tags/ for more details.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed: true,
			},
			"device_id": schema.StringAttribute{
				Required:    true,
				Description: "The device to set tags for",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"tags": schema.SetAttribute{
				Required:    true,
				Description: "The tags to apply to the device",
				ElementType: types.StringType,
			},
		},
	}
}

func (d deviceTagsResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state deviceTagsResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	deviceID := state.ID.ValueString()

	device, err := d.Client.Devices().Get(ctx, deviceID)
	if err != nil {
		resp.Diagnostics.AddError(
			"Failed to fetch device tags",
			"Failed to fetch device with ID "+deviceID+": "+err.Error(),
		)
		return
	}

	// If the device lookup succeeds and the state ID is not the same as the legacy ID, we can assume the ID is the node ID.
	canonicalDeviceID := device.ID
	if device.ID != deviceID {
		canonicalDeviceID = device.NodeID
	}

	state.DeviceID = types.StringValue(canonicalDeviceID)
	tags, diags := types.SetValueFrom(ctx, types.StringType, device.Tags)
	resp.Diagnostics.Append(diags...)
	if diags.HasError() {
		return
	}
	state.Tags = tags

	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
}

func (d deviceTagsResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan deviceTagsResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	deviceID := plan.DeviceID.ValueString()
	tags := make([]string, len(plan.Tags.Elements()))
	diags = plan.Tags.ElementsAs(ctx, &tags, false)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := d.Client.Devices().SetTags(ctx, deviceID, tags); err != nil {
		resp.Diagnostics.AddError(
			"Failed to update device tags",
			"Failed to update tags for device with ID "+deviceID+": "+err.Error(),
		)
		return
	}

	plan.ID = types.StringValue(deviceID)
	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
}

func (d deviceTagsResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan deviceTagsResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	deviceID := plan.DeviceID.ValueString()
	tags := make([]string, len(plan.Tags.Elements()))
	diags = plan.Tags.ElementsAs(ctx, &tags, false)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := d.Client.Devices().SetTags(ctx, deviceID, tags); err != nil {
		resp.Diagnostics.AddError(
			"Failed to update device tags",
			"Failed to update tags for device with ID "+deviceID+": "+err.Error(),
		)
		return
	}

	plan.ID = types.StringValue(deviceID)
	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
}

func (d deviceTagsResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	if isAcceptanceTesting() {
		// Tags cannot be removed without reauthorizing the device as a user.
		// We have no way of doing this during testing.
		// Because of https://github.com/hashicorp/terraform-plugin-sdk/issues/609,
		// we also have no way of telling the Terraform acceptance test to not test
		// resource deletion.
		// So, as a workaround, we don't actually delete during acceptance tests.
		return
	}

	var state deviceTagsResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	deviceID := state.DeviceID.ValueString()

	if err := d.Client.Devices().SetTags(ctx, deviceID, []string{}); err != nil {
		resp.Diagnostics.AddError(
			"Failed to delete device tags",
			"Failed to delete tags for device with ID "+deviceID+": "+err.Error(),
		)
		return
	}
}
