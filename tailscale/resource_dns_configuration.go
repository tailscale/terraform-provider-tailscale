// Copyright (c) David Bond, Tailscale Inc, & Contributors
// SPDX-License-Identifier: MIT

package tailscale

import (
	"context"
	"maps"
	"slices"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/listdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"tailscale.com/client/tailscale/v2"
)

var (
	_ resource.Resource                = &dnsConfigurationResource{}
	_ resource.ResourceWithConfigure   = &dnsConfigurationResource{}
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
				Computed:    true,
				Default:     listdefault.StaticValue(types.ListValueMust(types.StringType, []attr.Value{})),
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
			"split_dns": schema.SetNestedBlock{
				Description: "Set the nameservers used by devices on your network to resolve DNS queries on specific domains (requires Tailscale v1.8 or later). Configuration does not depend on `override_local_dns`.",
				// Custom plan modifier used to enforce default of `false` for
				// `use_with_exit_node` for split DNS nameservers.
				// This is a workaround for the framework's broken defaults in set elements:
				// https://github.com/hashicorp/terraform-plugin-framework/issues/783
				PlanModifiers: []planmodifier.Set{splitDNSDefaultsModifier{}},
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
								AtLeastOneBlockRequiredList(),
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

var _ planmodifier.Set = splitDNSDefaultsModifier{}

// splitDNSDefaultsModifier sets use_with_exit_node to false for split DNS
// nameservers where it is not configured.
//
// This replaces a schema default on use_with_exit_node, which is broken:
// See https://github.com/hashicorp/terraform-plugin-framework/issues/783
type splitDNSDefaultsModifier struct{}

func (m splitDNSDefaultsModifier) Description(_ context.Context) string {
	return "Sets use_with_exit_node to false for split DNS nameservers where it is not configured."
}

func (m splitDNSDefaultsModifier) MarkdownDescription(ctx context.Context) string {
	return m.Description(ctx)
}

func (m splitDNSDefaultsModifier) PlanModifySet(ctx context.Context, req planmodifier.SetRequest, resp *planmodifier.SetResponse) {
	if req.PlanValue.IsNull() || req.PlanValue.IsUnknown() ||
		req.ConfigValue.IsNull() || req.ConfigValue.IsUnknown() {
		return
	}

	elems := make([]attr.Value, 0, len(req.ConfigValue.Elements()))
	for _, elem := range req.ConfigValue.Elements() {
		updated, diags := defaultSplitDNS(ctx, elem)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}
		elems = append(elems, updated)
	}

	plan, diags := types.SetValue(req.ConfigValue.ElementType(ctx), elems)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.PlanValue = plan
}

// defaultSplitDNS applies nameserver defaults to one split DNS entry.
func defaultSplitDNS(ctx context.Context, elem attr.Value) (attr.Value, diag.Diagnostics) {
	obj, ok := elem.(types.Object)
	if !ok || obj.IsNull() || obj.IsUnknown() {
		return elem, nil
	}

	attrs := maps.Clone(obj.Attributes())
	nameservers, ok := attrs["nameservers"].(types.List)
	if !ok || nameservers.IsNull() || nameservers.IsUnknown() {
		return elem, nil
	}

	nsElems := make([]attr.Value, 0, len(nameservers.Elements()))
	for _, nsElem := range nameservers.Elements() {
		updated, diags := defaultNameserver(ctx, nsElem)
		if diags.HasError() {
			return nil, diags
		}

		nsElems = append(nsElems, updated)
	}

	updatedNameservers, diags := types.ListValue(
		nameservers.ElementType(ctx),
		nsElems,
	)
	if diags.HasError() {
		return nil, diags
	}

	attrs["nameservers"] = updatedNameservers
	return types.ObjectValue(obj.AttributeTypes(ctx), attrs)
}

// defaultNameserver defaults use_with_exit_node to false only when null.
// Explicit true/false and unknown values are preserved.
func defaultNameserver(ctx context.Context, elem attr.Value) (attr.Value, diag.Diagnostics) {
	obj, ok := elem.(types.Object)
	if !ok || obj.IsNull() || obj.IsUnknown() {
		return elem, nil
	}

	attrs := obj.Attributes()
	useWithExitNode, ok := attrs["use_with_exit_node"]
	if !ok || !useWithExitNode.IsNull() {
		return elem, nil
	}

	attrs = maps.Clone(attrs)
	attrs["use_with_exit_node"] = types.BoolValue(false)

	return types.ObjectValue(obj.AttributeTypes(ctx), attrs)
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
	state.SplitDNS = reconcileSplitDNS(state.SplitDNS, remote.SplitDNS)
	if remote.SearchPaths == nil {
		remote.SearchPaths = []string{}
	}
	state.SearchPaths = ListOfStringValue(ctx, remote.SearchPaths, &resp.Diagnostics)
	state.OverrideLocalDNS = types.BoolValue(remote.Preferences.OverrideLocalDNS)
	state.MagicDNS = types.BoolValue(remote.Preferences.MagicDNS)

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

// reconcileNameservers updates an existing list of nameservers with an updated list of nameservers,
// preserving the original ordering of any retained existing nameservers.
func reconcileNameservers(existing []nameserverModel, updates []tailscale.DNSConfigurationResolver) []nameserverModel {
	updates = slices.Clone(updates)
	nameservers := make([]nameserverModel, 0, len(updates))

	for _, nameserver := range existing {
		idx := slices.IndexFunc(updates, func(u tailscale.DNSConfigurationResolver) bool {
			return u.Address == nameserver.Address.ValueString()
		})
		if idx >= 0 {
			nameservers = append(nameservers, nameserverToMap(updates[idx]))
			updates = slices.Delete(updates, idx, idx+1)
		}
	}

	for _, n := range updates {
		nameservers = append(nameservers, nameserverToMap(n))
	}

	return nameservers
}

// reconcileSplitDNS updates an existing list of split DNS entries with an updated list
// of domains and nameservers, preserving the original ordering of any retained
// existing nameservers.
func reconcileSplitDNS(existing []splitDNSModel, updates map[string][]tailscale.DNSConfigurationResolver) []splitDNSModel {
	splitDNS := make([]splitDNSModel, 0, len(existing))
	for _, s := range existing {
		domain := s.Domain.ValueString()
		nameservers, found := updates[domain]
		if found {
			splitDNS = append(splitDNS, splitDNSModel{
				Domain:      s.Domain,
				Nameservers: reconcileNameservers(s.Nameservers, nameservers),
			})
			delete(updates, domain)
		}
	}
	// Add new SplitDNS
	for domain, nameservers := range updates {
		splitDNS = append(splitDNS, splitDNSModel{
			Domain:      types.StringValue(domain),
			Nameservers: reconcileNameservers(nil, nameservers),
		})
	}

	return splitDNS
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
