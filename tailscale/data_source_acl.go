// Copyright (c) David Bond, Tailscale Inc, & Contributors
// SPDX-License-Identifier: MIT

package tailscale

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"tailscale.com/client/tailscale/v2"

	"github.com/tailscale/hujson"
)

// NewACLDataSource returns a new ACL data source.
func NewACLDataSource() datasource.DataSource {
	return &aclDataSource{}
}

type aclDataSource struct {
	DataSourceBase
}

type aclDataSourceModel struct {
	ID     types.String `tfsdk:"id"`
	JSON   types.String `tfsdk:"json"`
	HuJSON types.String `tfsdk:"hujson"`
}

// Metadata defines the data source name as it appears in Terraform configurations.
func (d *aclDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_acl"
}

// Schema defines a schema describing what data is available in the data source response.
func (d *aclDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Returns the Tailscale policy file for a tailnet.",
		Attributes: map[string]schema.Attribute{
			"json": schema.StringAttribute{
				Computed:    true,
				Description: "The contents of the policy file as a JSON string.",
			},
			"hujson": schema.StringAttribute{
				Computed:    true,
				Description: "The contents of the policy file as a HuJSON string.",
			},
			"id": schema.StringAttribute{
				Computed: true,
			},
		},
	}
}

// toAclDataSourceModel converts a [tailscale.RawACL] response from the Tailscale API
// to an instance of [aclDataSourceModel], or a diagnostic if the conversion fails.
func toAclDataSourceModel(acl *tailscale.RawACL) (*aclDataSourceModel, *diag.ErrorDiagnostic) {
	huj, err := hujson.Parse([]byte(acl.HuJSON))
	if err != nil {
		diagnostic := diag.NewErrorDiagnostic("Failed to parse ACL as HuJSON", err.Error())
		return nil, &diagnostic
	}

	hujsonString := huj.String()

	// Minimize transforms the underlying HuJSON representation into valid JSON.
	// This is an in-place change of the [hujson.Value] and must be done after we have
	// stored the value of the HuJSON representation.
	huj.Minimize()
	jsonString := huj.String()

	data := aclDataSourceModel{
		ID:     types.StringValue(createUUID()),
		HuJSON: types.StringValue(hujsonString),
		JSON:   types.StringValue(jsonString),
	}

	return &data, nil
}

// Read fetches the data from the Tailscale API.
func (d *aclDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	acl, err := d.Client.PolicyFile().Raw(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Failed to fetch ACL", err.Error())
		return
	}
	data, diag := toAclDataSourceModel(acl)
	if diag != nil {
		resp.Diagnostics.Append(diag)
		return
	} else {
		resp.Diagnostics.Append(resp.State.Set(ctx, data)...)
	}
}
