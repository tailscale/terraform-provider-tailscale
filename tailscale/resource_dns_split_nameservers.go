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
	_ resource.ResourceWithConfigure   = &dnsSplitNameserversResource{}
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
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
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
			"use_with_exit_node": schema.BoolAttribute{
				Description: "Whether all of these nameservers will continue to be used when an exit node is selected (requires Tailscale v1.88.1 or later). Leave unset to preserve each nameserver's current setting.",
				Optional:    true,
				Computed:    true,
			},
		},
	}
}

type dnsSplitNameserversResourceData struct {
	ID              types.String `tfsdk:"id"`
	Domain          types.String `tfsdk:"domain"`
	Nameservers     types.Set    `tfsdk:"nameservers"`
	UseWithExitNode types.Bool   `tfsdk:"use_with_exit_node"`
}

func (r *dnsSplitNameserversResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state dnsSplitNameserversResourceData
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	configuration, err := r.Client.DNS().Configuration(ctx)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error fetching split DNS config",
			"Failed to fetch split DNS config: "+err.Error(),
		)
		return
	}

	domain := state.Domain.ValueString()
	resolvers := configuration.SplitDNS[domain]
	nameservers := make([]string, 0, len(resolvers))
	state.Nameservers = SetOfStringValue(ctx, appendNameserverAddresses(nameservers, resolvers), &resp.Diagnostics)
	if len(resolvers) > 0 {
		state.UseWithExitNode = types.BoolValue(allUseWithExitNode(resolvers))
	} else if state.UseWithExitNode.IsNull() {
		// There are no nameservers to derive the value from, so keep the
		// prior state value when one exists.
		state.UseWithExitNode = types.BoolValue(false)
	}

	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *dnsSplitNameserversResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan, config dnsSplitNameserversResourceData
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	r.updateSplitDNSConfig(ctx, &plan, !config.UseWithExitNode.IsNull(), &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	plan.ID = plan.Domain
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *dnsSplitNameserversResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, config dnsSplitNameserversResourceData
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	r.updateSplitDNSConfig(ctx, &plan, !config.UseWithExitNode.IsNull(), &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	diags := resp.State.Set(ctx, &plan)
	resp.Diagnostics.Append(diags...)
}

// updateSplitDNSConfig calls the Tailscale API to update the split DNS config
// based on the given input. When use_with_exit_node is unset in the
// configuration, the nameservers are sent as plain addresses so the server
// preserves each matching nameserver's current options.
func (r *dnsSplitNameserversResource) updateSplitDNSConfig(ctx context.Context, data *dnsSplitNameserversResourceData, manageUseWithExitNode bool, diags *diag.Diagnostics) {
	domain := data.Domain.ValueString()

	var nameservers []string
	diags.Append(data.Nameservers.ElementsAs(ctx, &nameservers, false)...)
	if diags.HasError() {
		return
	}

	if !manageUseWithExitNode {
		if _, err := r.Client.DNS().UpdateSplitDNS(ctx, tailscale.SplitDNSRequest{domain: nameservers}); err != nil {
			diags.AddError("Failed to update DNS split nameservers", err.Error())
			return
		}

		// The preserved options are not known until after the update, so
		// read them back for the state value.
		configuration, err := r.Client.DNS().Configuration(ctx)
		if err != nil {
			diags.AddError("Failed to fetch DNS configuration", err.Error())
			return
		}
		data.UseWithExitNode = types.BoolValue(allUseWithExitNode(configuration.SplitDNS[domain]))
		return
	}

	resolvers := make([]tailscale.DNSConfigurationResolver, 0, len(nameservers))
	for _, address := range nameservers {
		resolvers = append(resolvers, tailscale.DNSConfigurationResolver{
			Address:         address,
			UseWithExitNode: data.UseWithExitNode.ValueBool(),
		})
	}

	if _, err := r.Client.DNS().UpdateSplitDNSResolvers(ctx, tailscale.SplitDNSResolverRequest{domain: resolvers}); err != nil {
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

	if _, err := r.Client.DNS().UpdateSplitDNS(ctx, tailscale.SplitDNSRequest{domain: nil}); err != nil {
		resp.Diagnostics.AddError("Failed to delete DNS split nameservers", err.Error())
		return
	}
}

func (r *dnsSplitNameserversResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("domain"), req.ID)...)
}
