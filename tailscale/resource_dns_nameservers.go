// Copyright (c) David Bond, Tailscale Inc, & Contributors
// SPDX-License-Identifier: MIT

package tailscale

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework-validators/listvalidator"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"tailscale.com/client/tailscale/v2"
)

var (
	_ resource.Resource                = &dnsNameserversResource{}
	_ resource.ResourceWithConfigure   = &dnsNameserversResource{}
	_ resource.ResourceWithImportState = &dnsNameserversResource{}
)

// NewDNSNameserversResource returns a new DNS preferences resources.
func NewDNSNameserversResource() resource.Resource {
	return &dnsNameserversResource{}
}

type dnsNameserversResource struct {
	ResourceBase
	ResourceImportedByID
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
			"use_with_exit_node": schema.BoolAttribute{
				Description: "Whether all of these nameservers will continue to be used when an exit node is selected (requires Tailscale v1.88.1 or later). Leave unset to preserve each nameserver's current setting.",
				Optional:    true,
				Computed:    true,
			},
		},
	}
}

type dnsNameserversResourceData struct {
	ID              types.String `tfsdk:"id"`
	Nameservers     types.List   `tfsdk:"nameservers"`
	UseWithExitNode types.Bool   `tfsdk:"use_with_exit_node"`
}

func (r *dnsNameserversResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state dnsNameserversResourceData
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	configuration, err := r.Client.DNS().Configuration(ctx)
	if err != nil {
		resp.Diagnostics.AddError(
			"Failed to fetch DNS configuration",
			"Failed to fetch DNS configuration: "+err.Error(),
		)
		return
	}

	servers := make([]string, 0, len(configuration.Nameservers))
	state.Nameservers = ListOfStringValue(ctx, appendNameserverAddresses(servers, configuration.Nameservers), &resp.Diagnostics)
	state.UseWithExitNode = types.BoolValue(allUseWithExitNode(configuration.Nameservers))
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *dnsNameserversResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan, config dnsNameserversResourceData
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	r.updateDNSNameservers(ctx, &plan, !config.UseWithExitNode.IsNull(), &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	plan.ID = types.StringValue(createUUID())
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *dnsNameserversResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, config dnsNameserversResourceData
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	r.updateDNSNameservers(ctx, &plan, !config.UseWithExitNode.IsNull(), &resp.Diagnostics)
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

// updateDNSNameservers calls the Tailscale API to update the DNS nameservers
// based on the given input. When use_with_exit_node is unset in the
// configuration, the nameservers are sent as plain addresses so the server
// preserves each matching nameserver's current options.
func (r *dnsNameserversResource) updateDNSNameservers(ctx context.Context, data *dnsNameserversResourceData, manageUseWithExitNode bool, diags *diag.Diagnostics) {
	var addresses []string

	if !data.Nameservers.IsNull() {
		diags.Append(data.Nameservers.ElementsAs(ctx, &addresses, false)...)
		if diags.HasError() {
			return
		}
	}

	if !manageUseWithExitNode {
		if err := r.Client.DNS().SetNameservers(ctx, addresses); err != nil {
			diags.AddError("Failed to update DNS nameservers", err.Error())
			return
		}

		// The preserved options are not known until after the update, so
		// read them back for the state value.
		configuration, err := r.Client.DNS().Configuration(ctx)
		if err != nil {
			diags.AddError("Failed to fetch DNS configuration", err.Error())
			return
		}
		data.UseWithExitNode = types.BoolValue(allUseWithExitNode(configuration.Nameservers))
		return
	}

	nameservers := make([]tailscale.DNSConfigurationResolver, 0, len(addresses))
	for _, address := range addresses {
		nameservers = append(nameservers, tailscale.DNSConfigurationResolver{
			Address:         address,
			UseWithExitNode: data.UseWithExitNode.ValueBool(),
		})
	}

	if err := r.Client.DNS().SetNameserverResolvers(ctx, nameservers); err != nil {
		diags.AddError("Failed to update DNS nameservers", err.Error())
		return
	}
}

func appendNameserverAddresses(addresses []string, nameservers []tailscale.DNSConfigurationResolver) []string {
	for _, nameserver := range nameservers {
		addresses = append(addresses, nameserver.Address)
	}
	return addresses
}

// allUseWithExitNode reports whether every nameserver has the
// use-with-exit-node option. Mixed values collapse to false because the
// resource manages one value for all of its nameservers.
func allUseWithExitNode(nameservers []tailscale.DNSConfigurationResolver) bool {
	for _, nameserver := range nameservers {
		if !nameserver.UseWithExitNode {
			return false
		}
	}
	return len(nameservers) > 0
}
