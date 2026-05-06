// Copyright (c) David Bond, Tailscale Inc, & Contributors
// SPDX-License-Identifier: MIT

package tailscale

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// NewServiceDataSource() returns a new Services data source.
func NewServiceDataSource() datasource.DataSource {
	return &dataSourceService{}
}

type dataSourceService struct {
	DataSourceBase
}

// Metadata defines the data source name as it appears in Terraform configurations.
func (d dataSourceService) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_service"
}

// Schema defines a schema describing what data is available in the data source response.
func (d dataSourceService) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "The Service data source describes a single Service in a tailnet. See https://tailscale.com/docs/features/tailscale-services for more information.",
		Attributes: map[string]schema.Attribute{
			"name": schema.StringAttribute{
				Description: "The name of the Service (e.g. `svc:my-service`).",
				Required:    true,
			},
			"id": schema.StringAttribute{
				Description: "The Service name, e.g. 'svc:my-service'.",
				Computed:    true,
			},
			"addrs": schema.ListAttribute{
				Description: "The IP addresses assigned to the Service.",
				Computed:    true,
				ElementType: types.StringType,
			},
			"comment": schema.StringAttribute{
				Description: "A comment describing the Service.",
				Computed:    true,
			},
			"ports": schema.ListAttribute{
				Description: "The ports that the Service listens on.",
				Computed:    true,
				ElementType: types.StringType,
			},
			"tags": schema.SetAttribute{
				Description: "The ACL tags applied to the Service.",
				Computed:    true,
				ElementType: types.StringType,
			},
		},
	}
}

type dataSourceServiceModel struct {
	ID      types.String `tfsdk:"id"`
	Name    types.String `tfsdk:"name"`
	Addrs   types.List   `tfsdk:"addrs"`
	Comment types.String `tfsdk:"comment"`
	Ports   types.List   `tfsdk:"ports"`
	Tags    types.Set    `tfsdk:"tags"`
}

// Read fetches the data from the Tailscale API.
func (d dataSourceService) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data dataSourceServiceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	name := data.Name.ValueString()
	svc, err := d.Client.VIPServices().Get(ctx, name)
	if err != nil {
		resp.Diagnostics.AddError("Failed to fetch service", err.Error())
		return
	}

	data.ID = types.StringValue(svc.Name)
	data.Name = types.StringValue(svc.Name)
	data.Addrs = ListOfStringValue(ctx, svc.Addrs, &resp.Diagnostics)
	data.Comment = types.StringValue(svc.Comment)
	data.Ports = ListOfStringValue(ctx, svc.Ports, &resp.Diagnostics)
	data.Tags = SetOfStringValue(ctx, svc.Tags, &resp.Diagnostics)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
