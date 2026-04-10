// Copyright (c) David Bond, Tailscale Inc, & Contributors
// SPDX-License-Identifier: MIT

package tailscale

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"tailscale.com/client/tailscale/v2"
)

var (
	_ resource.Resource                = &dnsSplitNameserversResource{}
	_ resource.ResourceWithImportState = &dnsSplitNameserversResource{}
)

// NewDNSSplitNameserversResource returns a new DNS preferences resources.
func NewDNSSplitNameserversResource() resource.Resource {
	return &dnsSplitNameserversResource{}
}

type dnsSplitNameserversResource struct {
	ResourceBase
}

// Metadata defines the resource name as it appears in Terraform configurations.
func (r *dnsSplitNameserversResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_dns_split_nameservers"
}

// Schema defines a schema describing what fields can be defined in the resource.
func (r *dnsSplitNameserversResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "The dns_split_nameservers resource allows you to configure split DNS nameservers for your Tailscale network. See https://tailscale.com/kb/1054/dns for more information.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed: true,
			},
			"domain": schema.StringAttribute{
				Description: "Domain to configure split DNS for. Requests for this domain will be resolved using the provided nameservers. Changing this will force the resource to be recreated.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"nameservers": schema.SetAttribute{
				ElementType: types.StringType,
				Description: "Devices on your network will use these nameservers to resolve DNS names. IPv4 or IPv6 addresses are accepted.",
				Required:    true,
			},
		},
	}
}

type dnsSplitNameserversResourceData struct {
	ID          types.String `tfsdk:"id"`
	Domain      types.String `tfsdk:"domain"`
	Nameservers types.Set    `tfsdk:"nameservers"`
}

func (r *dnsSplitNameserversResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state dnsSplitNameserversResourceData
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	splitDNS, err := r.Client.DNS().SplitDNS(ctx)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error fetching split DNS config",
			"Failed to fetch split DNS config: "+err.Error(),
		)
		return
	}

	domain := state.Domain.ValueString()
	nameservers := splitDNS[domain]

	nsSet, diags := types.SetValueFrom(ctx, types.StringType, nameservers)
	resp.Diagnostics.Append(diags...)
	state.Nameservers = nsSet
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *dnsSplitNameserversResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan dnsSplitNameserversResourceData
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	r.updateSplitDNSConfig(ctx, &plan, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	plan.ID = plan.Domain
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *dnsSplitNameserversResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state dnsSplitNameserversResourceData
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	plan.ID = state.ID

	r.updateSplitDNSConfig(ctx, &plan, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	diags := resp.State.Set(ctx, &plan)
	resp.Diagnostics.Append(diags...)
}

// updateSplitDNSConfig calls the Tailscale API to update the split DNS config based
// on the given input.
func (r *dnsSplitNameserversResource) updateSplitDNSConfig(ctx context.Context, data *dnsSplitNameserversResourceData, diags *diag.Diagnostics) {
	domain := data.Domain.ValueString()

	var nameservers []string
	diags.Append(data.Nameservers.ElementsAs(ctx, &nameservers, false)...)
	if diags.HasError() {
		return
	}

	updateReq := tailscale.SplitDNSRequest{
		domain: nameservers,
	}

	if _, err := r.Client.DNS().UpdateSplitDNS(ctx, updateReq); err != nil {
		diags.AddError("Failed to update DNS split nameservers", err.Error())
		return
	}
}

func (r *dnsSplitNameserversResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state dnsSplitNameserversResourceData
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	domain := state.Domain.ValueString()
	updateReq := tailscale.SplitDNSRequest{domain: {}}

	if _, err := r.Client.DNS().UpdateSplitDNS(ctx, updateReq); err != nil {
		resp.Diagnostics.AddError("Failed to delete DNS split nameservers", err.Error())
		return
	}
}

func (r *dnsSplitNameserversResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("domain"), req.ID)...)
}
