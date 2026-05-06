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

const resourceDeviceSubnetRoutesDescription = `The device_subnet_routes resource allows you to configure enabled subnet routes for your Tailscale devices. See https://tailscale.com/kb/1019/subnets for more information.

Routes must be both advertised and enabled for a device to act as a subnet router or exit node. Routes must be advertised directly from the device: advertised routes cannot be managed through Terraform. If a device is advertising routes, they are not exposed to traffic until they are enabled. Conversely, if routes are enabled before they are advertised, they are not available for routing until the device in question is advertising them.

Note: all routes enabled for the device through the admin console or autoApprovers in the ACL must be explicitly added to the routes attribute of this resource to avoid configuration drift.
`

var (
	_ resource.Resource                = &deviceSubnetRoutesResource{}
	_ resource.ResourceWithImportState = &deviceSubnetRoutesResource{}
)

type deviceSubnetRoutesModel struct {
	ID       types.String `tfsdk:"id"`
	DeviceID types.String `tfsdk:"device_id"`
	Routes   types.Set    `tfsdk:"routes"`
}

func NewDeviceSubnetRoutesResource() resource.Resource {
	return &deviceSubnetRoutesResource{}
}

type deviceSubnetRoutesResource struct {
	ResourceBase
}

func (d deviceSubnetRoutesResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	// We can't do a simple passthrough here as the ID used for this resource is a
	// randomly generated UUID and we need to instead fetch based on the device_id.
	//
	// TODO(mpminardi): investigate changing the ID in state to be the device_id instead
	// in an eventual major version bump.
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), createUUID())...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("device_id"), req.ID)...)
}

func (d deviceSubnetRoutesResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_device_subnet_routes"
}

func (d deviceSubnetRoutesResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: resourceDeviceSubnetRoutesDescription,
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"device_id": schema.StringAttribute{
				Required:    true,
				Description: "The device to set subnet routes for",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"routes": schema.SetAttribute{
				Required:    true,
				Description: "The subnet routes that are enabled to be routed by a device",
				ElementType: types.StringType,
			},
		},
	}
}

func (d deviceSubnetRoutesResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state deviceSubnetRoutesModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	deviceID := state.DeviceID.ValueString()

	deviceRoutes, err := d.Client.Devices().SubnetRoutes(ctx, deviceID)

	if err != nil {
		// If the device is not found, remove from the state so we can create it again.
		if tailscale.IsNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}

		resp.Diagnostics.AddError(
			"Failed to fetch device subnet routes",
			"Failed to fetch subnet routes for device with ID "+deviceID+": "+err.Error(),
		)
		return
	}

	state.Routes = SetOfStringValue(ctx, deviceRoutes.Enabled, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
}

func (d deviceSubnetRoutesResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan deviceSubnetRoutesModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	deviceID := plan.DeviceID.ValueString()
	routes := plan.Routes

	subnetRoutes := make([]string, len(routes.Elements()))
	diags = routes.ElementsAs(ctx, &subnetRoutes, false)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := d.Client.Devices().SetSubnetRoutes(ctx, deviceID, subnetRoutes); err != nil {
		resp.Diagnostics.AddError(
			"Failed to update device subnet routes",
			"Failed to update subnet routes for device with ID "+deviceID+": "+err.Error(),
		)
		return
	}

	plan.ID = types.StringValue(createUUID())
	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
}

func (d deviceSubnetRoutesResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan deviceSubnetRoutesModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	deviceID := plan.DeviceID.ValueString()
	routes := plan.Routes

	subnetRoutes := make([]string, len(routes.Elements()))
	diags = routes.ElementsAs(ctx, &subnetRoutes, false)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := d.Client.Devices().SetSubnetRoutes(ctx, deviceID, subnetRoutes); err != nil {
		resp.Diagnostics.AddError(
			"Failed to update device subnet routes",
			"Failed to update subnet routes for device with ID "+deviceID+": "+err.Error(),
		)
		return
	}

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
}

func (d deviceSubnetRoutesResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state deviceSubnetRoutesModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	deviceID := state.DeviceID.ValueString()

	if err := d.Client.Devices().SetSubnetRoutes(ctx, deviceID, []string{}); err != nil {
		resp.Diagnostics.AddError(
			"Failed to delete device subnet routes",
			"Failed to delete subnet routes for device with ID "+deviceID+": "+err.Error(),
		)
		return
	}
}
