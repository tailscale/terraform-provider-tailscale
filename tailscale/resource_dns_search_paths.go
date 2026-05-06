// Copyright (c) David Bond, Tailscale Inc, & Contributors
// SPDX-License-Identifier: MIT

package tailscale

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ resource.Resource                = &dnsSearchPathsResource{}
	_ resource.ResourceWithImportState = &dnsSearchPathsResource{}
)

// NewDNSPreferencesResource returns a new DNS search paths resources.
func NewDNSSearchPathsResource() resource.Resource {
	return &dnsSearchPathsResource{}
}

type dnsSearchPathsResource struct {
	ResourceBase
	ResourceImportedByID
}

// Metadata defines the resource name as it appears in Terraform configurations.
func (r *dnsSearchPathsResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_dns_search_paths"
}

// Schema defines a schema describing what fields can be defined in the resource.
func (r *dnsSearchPathsResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "The dns_search_paths resource allows you to configure DNS search paths for your Tailscale network. See https://tailscale.com/kb/1054/dns for more information.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"search_paths": schema.ListAttribute{
				ElementType: types.StringType,
				Required:    true,
				Description: "Devices on your network will use these domain suffixes to resolve DNS names.",
			},
		},
	}
}

type dnsSearchPathsResourceData struct {
	ID          types.String `tfsdk:"id"`
	SearchPaths types.List   `tfsdk:"search_paths"`
}

func (r *dnsSearchPathsResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state dnsSearchPathsResourceData
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	paths, err := r.Client.DNS().SearchPaths(ctx)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error fetching DNS search paths",
			"Failed to fetch DNS search paths: "+err.Error(),
		)
		return
	}

	state.SearchPaths = ListOfStringValue(ctx, paths, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
}

func (r *dnsSearchPathsResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan dnsSearchPathsResourceData
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	plan.ID = types.StringValue(createUUID())

	r.updateDNSSearchPaths(ctx, &plan, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	diags = resp.State.Set(ctx, &plan)
	resp.Diagnostics.Append(diags...)
}

func (r *dnsSearchPathsResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state dnsSearchPathsResourceData
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)

	if resp.Diagnostics.HasError() {
		return
	}

	if !plan.SearchPaths.Equal(state.SearchPaths) {
		r.updateDNSSearchPaths(ctx, &plan, &resp.Diagnostics)
		if resp.Diagnostics.HasError() {
			return
		}
	}

	diags := resp.State.Set(ctx, &plan)
	resp.Diagnostics.Append(diags...)
}

func (r *dnsSearchPathsResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	if err := r.Client.DNS().SetSearchPaths(ctx, []string{}); err != nil {
		resp.Diagnostics.AddError("Failed to delete DNS search paths", err.Error())
	}
}

// updateDNSSearchPaths calls the Tailscale API to update the DNS search paths based
// on the given input.
func (r *dnsSearchPathsResource) updateDNSSearchPaths(ctx context.Context, data *dnsSearchPathsResourceData, diags *diag.Diagnostics) {
	var searchPaths []string

	diags.Append(data.SearchPaths.ElementsAs(ctx, &searchPaths, false)...)
	if diags.HasError() {
		return
	}

	if err := r.Client.DNS().SetSearchPaths(ctx, searchPaths); err != nil {
		diags.AddError("Failed to set DNS search paths", "Failed to set DNS search paths: "+err.Error())
	}
}
