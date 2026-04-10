// Copyright (c) David Bond, Tailscale Inc, & Contributors
// SPDX-License-Identifier: MIT

package tailscale

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"tailscale.com/client/tailscale/v2"
)

var _ resource.ResourceWithImportState = &dnsPreferencesResource{}

// NewDNSPreferencesResource returns a new DNS preferences resources.
func NewDNSPreferencesResource() resource.Resource {
	return &dnsPreferencesResource{}
}

type dnsPreferencesResource struct {
	ResourceBase
}

// Metadata defines the resource name as it appears in Terraform configurations.
func (r *dnsPreferencesResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_dns_preferences"
}

// Schema defines a schema describing what fields can be defined in the resource.
func (r *dnsPreferencesResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "The dns_preferences resource allows you to configure DNS preferences for your Tailscale network. See https://tailscale.com/kb/1054/dns for more information.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed: true,
			},
			"magic_dns": schema.BoolAttribute{
				Description: "Whether or not to enable magic DNS",
				Required:    true,
			},
		},
	}
}

type dnsPreferencesResourceData struct {
	ID       types.String `tfsdk:"id"`
	MagicDNS types.Bool   `tfsdk:"magic_dns"`
}

func (r *dnsPreferencesResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state dnsPreferencesResourceData
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	preferences, err := r.Client.DNS().Preferences(ctx)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error fetching DNS preferences",
			"Failed to fetch DNS preferences: "+err.Error(),
		)
		return
	}

	state.MagicDNS = types.BoolValue(preferences.MagicDNS)
	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
}

func (r *dnsPreferencesResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan dnsPreferencesResourceData
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	plan.ID = types.StringValue(createUUID())

	r.updateDNSPreferences(ctx, &plan, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	diags = resp.State.Set(ctx, &plan)
	resp.Diagnostics.Append(diags...)
}

func (r *dnsPreferencesResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state dnsPreferencesResourceData
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)

	if resp.Diagnostics.HasError() {
		return
	}

	plan.ID = state.ID

	if !plan.MagicDNS.Equal(state.MagicDNS) {
		r.updateDNSPreferences(ctx, &plan, &resp.Diagnostics)
		if resp.Diagnostics.HasError() {
			return
		}
	}

	diags := resp.State.Set(ctx, &plan)
	resp.Diagnostics.Append(diags...)
}

func (r *dnsPreferencesResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	if err := r.Client.DNS().SetPreferences(ctx, tailscale.DNSPreferences{}); err != nil {
		resp.Diagnostics.AddError("Failed to set DNS preferences", "Failed to set DNS preferences: "+err.Error())
	}
}

func (r *dnsPreferencesResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

// updateDNSPreferences calls the Tailscale API to update the DNS preferences based
// on the given input.
func (r *dnsPreferencesResource) updateDNSPreferences(ctx context.Context, data *dnsPreferencesResourceData, diags *diag.Diagnostics) {
	prefs := tailscale.DNSPreferences{
		MagicDNS: data.MagicDNS.ValueBool(),
	}

	if err := r.Client.DNS().SetPreferences(ctx, prefs); err != nil {
		diags.AddError("Failed to set DNS preferences", "Failed to set DNS preferences: "+err.Error())
	}
}
