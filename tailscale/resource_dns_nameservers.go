// Copyright (c) David Bond, Tailscale Inc, & Contributors
// SPDX-License-Identifier: MIT

package tailscale

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework-validators/listvalidator"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
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
				Description: "All nameservers will continue to be used when an exit node is selected (requires Tailscale v1.88.1 or later). Defaults to false.",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(false),
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
			"Error fetching DNS name servers",
			"Failed to fetch DNS name servers: "+err.Error(),
		)
		return
	}

	servers := make([]string, 0, len(configuration.Nameservers))
	useWithExitNode := len(configuration.Nameservers) > 0
	for _, nameserver := range configuration.Nameservers {
		servers = append(servers, nameserver.Address)
		useWithExitNode = useWithExitNode && nameserver.UseWithExitNode
	}
	state.Nameservers = ListOfStringValue(ctx, servers, &resp.Diagnostics)
	state.UseWithExitNode = types.BoolValue(useWithExitNode)
	if resp.Diagnostics.HasError() {
		return
	}

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
	if err := r.Client.DNS().SetNameserversWithOptions(ctx, nil); err != nil {
		resp.Diagnostics.AddError("Failed to delete DNS nameservers", err.Error())
	}
}

// updateDNSNameservers calls the Tailscale API to update the DNS nameservers based
// on the given input.
func (r *dnsNameserversResource) updateDNSNameservers(ctx context.Context, data *dnsNameserversResourceData, diags *diag.Diagnostics) {
	var addresses []string

	if !data.Nameservers.IsNull() {
		diags.Append(data.Nameservers.ElementsAs(ctx, &addresses, false)...)
		if diags.HasError() {
			return
		}
	}

	nameservers := make([]tailscale.DNSConfigurationResolver, 0, len(addresses))
	for _, address := range addresses {
		nameservers = append(nameservers, tailscale.DNSConfigurationResolver{
			Address:         address,
			UseWithExitNode: data.UseWithExitNode.ValueBool(),
		})
	}

	if err := r.Client.DNS().SetNameserversWithOptions(ctx, nameservers); err != nil {
		diags.AddError("Failed to update DNS nameservers", err.Error())
		return
	}
}
