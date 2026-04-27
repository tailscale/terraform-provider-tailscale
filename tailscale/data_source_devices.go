// Copyright (c) David Bond, Tailscale Inc, & Contributors
// SPDX-License-Identifier: MIT

package tailscale

import (
	"context"
	"maps"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"

	"github.com/hashicorp/terraform-plugin-framework/types"

	"tailscale.com/client/tailscale/v2"
)

// NewMultipleDevicesDataSource returns a new multiple-users data data source.
func NewMultipleDevicesDataSource() datasource.DataSource {
	return &multipleDevicesDataSource{}
}

type multipleDevicesDataSource struct {
	DataSourceBase
}

type multipleDevicesDataSourceModel struct {
	ID         types.String            `tfsdk:"id"`
	NamePrefix types.String            `tfsdk:"name_prefix"`
	Filters    []filterModel           `tfsdk:"filter"`
	Devices    []deviceDataSourceModel `tfsdk:"devices"`
}

type filterModel struct {
	Name   types.String `tfsdk:"name"`
	Values types.Set    `tfsdk:"values"`
}

// Metadata defines the data source name as it appears in Terraform configurations.
func (d multipleDevicesDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_devices"
}

// Schema defines a schema describing what data is available in the data source response.
func (d multipleDevicesDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	nestedDeviceAttributes := map[string]schema.Attribute{
		"name": schema.StringAttribute{

			Description: "The full name of the device (e.g. `hostname.domain.ts.net`)",
			Computed:    true,
		},
		"hostname": schema.StringAttribute{
			Description: "The short hostname of the device",
			Computed:    true,
		},
	}
	maps.Copy(nestedDeviceAttributes, deviceSchema)

	resp.Schema = schema.Schema{
		Description: "The devices data source describes a list of devices in a tailnet",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed: true,
			},
			"name_prefix": schema.StringAttribute{
				Optional:    true,
				Description: "Filters the device list to elements whose name has the provided prefix",
			},
		},
		Blocks: map[string]schema.Block{
			"filter": schema.SetNestedBlock{
				Description: "Filters the device list to elements devices whose fields match the provided values.",
				NestedObject: schema.NestedBlockObject{
					Attributes: map[string]schema.Attribute{
						"name": schema.StringAttribute{
							Description: "The name must be a top-level device property, e.g. isEphemeral, tags, hostname, etc.",
							Required:    true,
						},
						"values": schema.SetAttribute{
							Description: "The list of values to filter for. Values are matched as exact matches.",
							ElementType: types.StringType,
							Required:    true,
						},
					},
				},
			},
			"devices": schema.ListNestedBlock{
				Description: "The list of devices in the tailnet",
				NestedObject: schema.NestedBlockObject{
					Attributes: nestedDeviceAttributes,
				},
			},
		},
	}
}

// Read fetches the data from the Tailscale API.
func (d multipleDevicesDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data multipleDevicesDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	opts := make([]tailscale.ListDevicesOptions, 0, len(data.Filters))
	for _, f := range data.Filters {
		var values []string

		diags := f.Values.ElementsAs(ctx, &values, false)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}

		opts = append(opts, tailscale.WithFilter(f.Name.ValueString(), values))
	}

	devices, err := d.Client.Devices().List(ctx, opts...)
	if err != nil {
		resp.Diagnostics.AddError("Failed to fetch devices", err.Error())
		return
	}

	prefix := data.NamePrefix.ValueString()
	data.Devices = make([]deviceDataSourceModel, 0)

	for _, dev := range devices {
		if prefix != "" && !strings.HasPrefix(dev.Name, prefix) {
			continue
		}

		deviceModel, diagnostics := toDeviceDataSourceModel(ctx, &dev)
		if diagnostics.HasError() {
			resp.Diagnostics.Append(diagnostics...)
			return
		}

		data.Devices = append(data.Devices, deviceModel)
	}

	data.ID = types.StringValue(createUUID())
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
