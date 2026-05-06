// Copyright (c) David Bond, Tailscale Inc, & Contributors
// SPDX-License-Identifier: MIT

package tailscale

import (
	"context"
	"slices"
	"strings"

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
	_ resource.Resource                = &dnsConfigurationResource{}
	_ resource.ResourceWithImportState = &dnsConfigurationResource{}
)

// NewDNSConfigurationResource returns a new DNS configuration resource.
func NewDNSConfigurationResource() resource.Resource {
	return &dnsConfigurationResource{}
}

type dnsConfigurationResource struct {
	ResourceBase
	ResourceImportedByID
}

// Metadata defines the resource name as it appears in Terraform configurations.
func (r *dnsConfigurationResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_dns_configuration"
}

// Schema defines a schema describing what fields can be defined in the resource.
func (r *dnsConfigurationResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "The dns_configuration resource allows you to manage the complete DNS configuration for your Tailscale network. See https://tailscale.com/kb/1054/dns for more information.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"search_paths": schema.ListAttribute{
				Description: "Additional search domains. When MagicDNS is on, the tailnet domain is automatically included as the first search domain.",
				ElementType: types.StringType,
				Optional:    true,
			},
			"override_local_dns": schema.BoolAttribute{
				Description: "When enabled, use the configured DNS servers in `nameservers` to resolve names outside the tailnet. When disabled, devices will prefer their local DNS configuration. Defaults to false.",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(false),
			},
			"magic_dns": schema.BoolAttribute{
				Description: "Whether or not to enable MagicDNS. Defaults to true.",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(true),
			},
		},
		// TODO: When we migrate to v6 of the Terraform plugin framework,
		// these should be converted to use [schema.NestedAttribute].
		Blocks: map[string]schema.Block{
			"nameservers": schema.ListNestedBlock{
				Description: "Set the nameservers used by devices on your network to resolve DNS queries. `override_local_dns` must also be true to prefer these nameservers over local DNS configuration.",
				NestedObject: schema.NestedBlockObject{
					Attributes: map[string]schema.Attribute{
						"address": schema.StringAttribute{
							Description: "The nameserver's IPv4 or IPv6 address",
							Required:    true,
						},
						"use_with_exit_node": schema.BoolAttribute{
							Description: "This nameserver will continue to be used when an exit node is selected (requires Tailscale v1.88.1 or later). Defaults to false.",
							Optional:    true,
							Computed:    true,
							Default:     booldefault.StaticBool(false),
						},
					},
				},
			},
			"split_dns": schema.ListNestedBlock{
				Description: "Set the nameservers used by devices on your network to resolve DNS queries on specific domains (requires Tailscale v1.8 or later). Configuration does not depend on `override_local_dns`.",
				NestedObject: schema.NestedBlockObject{
					Attributes: map[string]schema.Attribute{
						"domain": schema.StringAttribute{
							Description: "The nameservers will be used only for this domain.",
							Required:    true,
						},
					},
					Blocks: map[string]schema.Block{
						"nameservers": schema.ListNestedBlock{
							Description: "Set the nameservers used by devices on your network to resolve DNS queries.",
							Validators: []validator.List{
								listvalidator.SizeAtLeast(1),
							},
							NestedObject: schema.NestedBlockObject{
								Attributes: map[string]schema.Attribute{
									"address": schema.StringAttribute{
										Description: "The nameserver's IPv4 or IPv6 address.",
										Required:    true,
									},
									"use_with_exit_node": schema.BoolAttribute{
										Description: "This nameserver will continue to be used when an exit node is selected (requires Tailscale v1.88.1 or later). Defaults to false.",
										Optional:    true,
										Computed:    true,
										Default:     booldefault.StaticBool(false),
									},
								},
							},
						},
					},
				},
			},
		},
	}
}

type dnsConfigurationResourceData struct {
	ID               types.String      `tfsdk:"id"`
	MagicDNS         types.Bool        `tfsdk:"magic_dns"`
	OverrideLocalDNS types.Bool        `tfsdk:"override_local_dns"`
	SearchPaths      types.List        `tfsdk:"search_paths"`
	Nameservers      []nameserverModel `tfsdk:"nameservers"`
	SplitDNS         []splitDNSModel   `tfsdk:"split_dns"`
}

type nameserverModel struct {
	Address         types.String `tfsdk:"address"`
	UseWithExitNode types.Bool   `tfsdk:"use_with_exit_node"`
}

type splitDNSModel struct {
	Domain      types.String      `tfsdk:"domain"`
	Nameservers []nameserverModel `tfsdk:"nameservers"`
}

func (r *dnsConfigurationResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state dnsConfigurationResourceData
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	remote, err := r.Client.DNS().Configuration(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Failed to fetch DNS configuration", err.Error())
		return
	}

	state.Nameservers = reconcileNameservers(state.Nameservers, remote.Nameservers)

	// Read existing SplitDNS to preserve order in TF resource
	// TODO(alexc): Extract this into a reconcileSplitDNS function to match nameservers.
	splitDNS := make([]splitDNSModel, 0, len(state.SplitDNS))
	for _, s := range state.SplitDNS {
		domain := s.Domain.ValueString()
		nameservers, found := remote.SplitDNS[domain]
		if found {
			splitDNS = append(splitDNS, splitDNSModel{
				Domain:      s.Domain,
				Nameservers: reconcileNameservers(s.Nameservers, nameservers),
			})
			delete(remote.SplitDNS, domain)
		}
	}
	// Add new SplitDNS
	for domain, nameservers := range remote.SplitDNS {
		splitDNS = append(splitDNS, splitDNSModel{
			Domain:      types.StringValue(domain),
			Nameservers: reconcileNameservers(nil, nameservers),
		})
	}
	state.SplitDNS = splitDNS
	state.SearchPaths = ListOfStringValue(ctx, remote.SearchPaths, &resp.Diagnostics)
	state.OverrideLocalDNS = types.BoolValue(remote.Preferences.OverrideLocalDNS)
	state.MagicDNS = types.BoolValue(remote.Preferences.MagicDNS)

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)

	resp.Diagnostics.AddWarning(
		"Alpha Resource",
		"The tailscale_dns_configuration resource is currently in alpha and subject to change, proceed with caution.",
	)
}

// reconcileNameservers updates an existing list of nameservers with an updated list of nameservers,
// preserving the original ordering of any retained existing nameservers.
func reconcileNameservers(existing []nameserverModel, updates []tailscale.DNSConfigurationResolver) []nameserverModel {
	nameservers := make([]nameserverModel, 0, len(updates))

	for _, nameserver := range existing {
		idx, found := slices.BinarySearchFunc(updates, nameserver.Address.ValueString(), func(a tailscale.DNSConfigurationResolver, b string) int {
			return strings.Compare(a.Address, b)
		})
		if found {
			nameservers = append(nameservers, nameserverToMap(updates[idx]))
			updates = slices.Delete(updates, idx, idx+1)
		}
	}

	for _, n := range updates {
		nameservers = append(nameservers, nameserverToMap(n))
	}

	return nameservers
}

func nameserverToMap(nameserver tailscale.DNSConfigurationResolver) nameserverModel {
	return nameserverModel{
		Address:         types.StringValue(nameserver.Address),
		UseWithExitNode: types.BoolValue(nameserver.UseWithExitNode),
	}
}

func (r *dnsConfigurationResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan dnsConfigurationResourceData
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	r.updateDNSConfiguration(ctx, &plan, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	plan.ID = types.StringValue(createUUID())
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *dnsConfigurationResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan dnsConfigurationResourceData
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	r.updateDNSConfiguration(ctx, &plan, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	diags := resp.State.Set(ctx, &plan)
	resp.Diagnostics.Append(diags...)
}

func (r *dnsConfigurationResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	if err := r.Client.DNS().SetConfiguration(ctx, tailscale.DNSConfiguration{}); err != nil {
		resp.Diagnostics.AddError("Failed to delete DNS configuration", err.Error())
	}
}

// updateDNSConfiguration calls the Tailscale API to update the DNS configuration based
// on the given input.
func (r *dnsConfigurationResource) updateDNSConfiguration(ctx context.Context, data *dnsConfigurationResourceData, diags *diag.Diagnostics) {
	configuration := tailscale.DNSConfiguration{
		SplitDNS: make(map[string][]tailscale.DNSConfigurationResolver),
		Preferences: tailscale.DNSConfigurationPreferences{
			OverrideLocalDNS: data.OverrideLocalDNS.ValueBool(),
			MagicDNS:         data.MagicDNS.ValueBool(),
		},
	}

	for _, nameserver := range data.Nameservers {
		configuration.Nameservers = append(configuration.Nameservers, tailscale.DNSConfigurationResolver{
			Address:         nameserver.Address.ValueString(),
			UseWithExitNode: nameserver.UseWithExitNode.ValueBool(),
		})
	}

	for _, splitDNS := range data.SplitDNS {
		domain := splitDNS.Domain.ValueString()
		var nameservers []tailscale.DNSConfigurationResolver
		for _, nameserver := range splitDNS.Nameservers {
			nameservers = append(nameservers, tailscale.DNSConfigurationResolver{
				Address:         nameserver.Address.ValueString(),
				UseWithExitNode: nameserver.UseWithExitNode.ValueBool(),
			})
		}
		configuration.SplitDNS[domain] = nameservers
	}

	var searchPaths []string
	diags.Append(data.SearchPaths.ElementsAs(ctx, &searchPaths, false)...)
	for _, path := range searchPaths {
		configuration.SearchPaths = append(configuration.SearchPaths, path)
	}

	if err := r.Client.DNS().SetConfiguration(ctx, configuration); err != nil {
		diags.AddError("Failed to set DNS configuration", err.Error())
	}
}
