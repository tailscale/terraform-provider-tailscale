// Copyright (c) David Bond, Tailscale Inc, & Contributors
// SPDX-License-Identifier: MIT

package tailscale

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework-validators/listvalidator"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ resource.Resource                = &dnsNameserversResource{}
	_ resource.ResourceWithImportState = &dnsNameserversResource{}
)

// NewDNSNameserversResource returns a new DNS preferences resources.
func NewDNSNameserversResource() resource.Resource {
	return &dnsNameserversResource{}
}

type dnsNameserversResource struct {
	ResourceBase
}

// Metadata defines the resource name as it appears in Terraform configurations.
func (r *dnsNameserversResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_dns_nameservers"
}

// Schema defines a schema describing what fields can be defined in the resource.
func (r *dnsNameserversResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "The dns_nameservers resource allows you to configure DNS nameservers for your Tailscale network. See https://tailscale.com/kb/1054/dns for more information.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"nameservers": schema.ListAttribute{
				ElementType: types.StringType,
				Description: "Devices on your network will use these nameservers to resolve DNS names. IPv4 or IPv6 addresses are accepted.",
				Required:    true,
				Validators: []validator.List{
					listvalidator.SizeAtLeast(1),
				},
			},
		},
	}
}

type dnsNameserversResourceData struct {
	ID          types.String `tfsdk:"id"`
	Nameservers types.List   `tfsdk:"nameservers"`
}

func (r *dnsNameserversResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state dnsNameserversResourceData
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	servers, err := r.Client.DNS().Nameservers(ctx)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error fetching DNS name servers",
			"Failed to fetch DNS name servers: "+err.Error(),
		)
		return
	}

	nsSet, diags := types.ListValueFrom(ctx, types.StringType, servers)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	state.Nameservers = nsSet
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *dnsNameserversResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan dnsNameserversResourceData
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	r.updateDNSNameservers(ctx, &plan, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	plan.ID = types.StringValue(createUUID())
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *dnsNameserversResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan dnsNameserversResourceData
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	r.updateDNSNameservers(ctx, &plan, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	diags := resp.State.Set(ctx, &plan)
	resp.Diagnostics.Append(diags...)
}

func (r *dnsNameserversResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	if err := r.Client.DNS().SetNameservers(ctx, []string{}); err != nil {
		resp.Diagnostics.AddError("Failed to delete DNS nameservers", err.Error())
	}
}

func (r *dnsNameserversResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

// updateDNSNameservers calls the Tailscale API to update the DNS nameservers based
// on the given input.
func (r *dnsNameserversResource) updateDNSNameservers(ctx context.Context, data *dnsNameserversResourceData, diags *diag.Diagnostics) {
	var nameservers []string

	if !data.Nameservers.IsNull() {
		diags.Append(data.Nameservers.ElementsAs(ctx, &nameservers, false)...)
		if diags.HasError() {
			return
		}
	}

	if err := r.Client.DNS().SetNameservers(ctx, nameservers); err != nil {
		diags.AddError("Failed to update DNS nameservers", err.Error())
		return
	}
}
