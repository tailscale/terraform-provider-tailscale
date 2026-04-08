// Copyright (c) David Bond, Tailscale Inc, & Contributors
// SPDX-License-Identifier: MIT

package tailscale

import (
	"context"
	"net"
	"net/netip"

	"github.com/hashicorp/terraform-plugin-framework-validators/helpers/validatordiag"
	"github.com/hashicorp/terraform-plugin-framework-validators/int32validator"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"tailscale.com/net/tsaddr"
)

type dataSource4Via6Model struct {
	ID   types.String `tfsdk:"id"`
	Site types.Int32  `tfsdk:"site"`
	CIDR types.String `tfsdk:"cidr"`
	IPv6 types.String `tfsdk:"ipv6"`
}

// New4Via6DataSource() returns a new 4via6 data source.
func New4Via6DataSource() datasource.DataSource {
	return &dataSource4via6{}
}

type dataSource4via6 struct {
	DataSourceBase
}

// Metadata defines the data source name as it appears in Terraform configurations.
func (d dataSource4via6) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_4via6"
}

// Schema defines a schema describing what data is available in the data source response.
func (d dataSource4via6) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "The 4via6 data source is calculates an IPv6 prefix for a given site ID and IPv4 CIDR. See Tailscale documentation for [4via6 subnets](https://tailscale.com/kb/1201/4via6-subnets/) for more details.",
		Attributes: map[string]schema.Attribute{
			"site": schema.Int32Attribute{
				Required:    true,
				Description: "Site ID (between 0 and 65535)",
				Validators: []validator.Int32{
					int32validator.Between(0, 65535),
				},
			},
			"cidr": schema.StringAttribute{
				Description: "The IPv4 CIDR to map",
				Required:    true,
				Validators: []validator.String{
					cidrValidator{},
				},
			},
			"ipv6": schema.StringAttribute{
				Description: "The 4via6 mapped address",
				Computed:    true,
			},
			"id": schema.StringAttribute{
				Computed: true,
			},
		},
	}
}

// Read fetches the data from the Tailscale API.
func (d dataSource4via6) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data dataSource4Via6Model
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	site := uint32(data.Site.ValueInt32())

	cidr, err := netip.ParsePrefix(data.CIDR.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Invalid CIDR",
			"The provided CIDR "+data.CIDR.ValueString()+" is invalid: "+err.Error(),
		)
		return
	}

	via, err := tsaddr.MapVia(site, cidr)
	if err != nil {
		resp.Diagnostics.AddError(
			"Calculation Error",
			"Failed to map 4via6 address: "+err.Error(),
		)
		return
	}

	mapped := via.String()
	data.ID = types.StringValue(mapped)
	data.IPv6 = types.StringValue(mapped)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// cidrValidator is a [validator.String] for CIDR addresses.
type cidrValidator struct{}

func (v cidrValidator) Description(_ context.Context) string {
	return "value must be a CIDR address."
}

func (v cidrValidator) MarkdownDescription(ctx context.Context) string {
	return v.Description(ctx)
}

func (v cidrValidator) ValidateString(ctx context.Context, req validator.StringRequest, resp *validator.StringResponse) {
	if req.ConfigValue.IsUnknown() || req.ConfigValue.IsNull() {
		return
	}

	value := req.ConfigValue.ValueString()
	_, _, err := net.ParseCIDR(value)
	if err != nil {
		resp.Diagnostics.Append(validatordiag.InvalidAttributeValueDiagnostic(
			req.Path,
			v.Description(ctx),
			req.ConfigValue.ValueString(),
		))
	}
}
