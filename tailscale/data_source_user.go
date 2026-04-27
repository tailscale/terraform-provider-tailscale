// Copyright (c) David Bond, Tailscale Inc, & Contributors
// SPDX-License-Identifier: MIT

package tailscale

import (
	"context"
	"maps"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"

	"tailscale.com/client/tailscale/v2"
)

// NewUserDataSource returns a new single-user data source.
func NewSingleUserDataSource() datasource.DataSource {
	return &singleUserDataSource{}
}

type singleUserDataSource struct {
	DataSourceBase
}

// Metadata defines the data source name as it appears in Terraform configurations.
func (d singleUserDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_user"
}

// Schema defines a schema describing what data is available in the data source response.
func (d singleUserDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	attributes := map[string]schema.Attribute{
		"id": schema.StringAttribute{
			Description: "The unique identifier for the user.",
			Optional:    true,
			Validators: []validator.String{
				stringvalidator.ExactlyOneOf(path.MatchRoot("id"), path.MatchRoot("login_name")),
			},
		},
		"login_name": schema.StringAttribute{
			Description: "The emailish login name of the user.",
			Optional:    true,
		},
	}

	maps.Copy(attributes, userSchema)

	resp.Schema = schema.Schema{
		Description: "The user data source describes a single user in a tailnet",
		Attributes:  attributes,
	}
}

// Read fetches the data from the Tailscale API.
func (d singleUserDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data userDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var user *tailscale.User
	var err error

	if !data.ID.IsNull() {
		user, err = d.Client.Users().Get(ctx, data.ID.ValueString())
		if err != nil {
			resp.Diagnostics.AddError("Failed to fetch user by ID", err.Error())
			return
		}
	} else if !data.LoginName.IsNull() {
		users, err := d.Client.Users().List(ctx, nil, nil)
		if err != nil {
			resp.Diagnostics.AddError("Failed to fetch users", err.Error())
		}

		for _, u := range users {
			if u.LoginName == data.LoginName.ValueString() {
				user = &u
				break
			}
		}

		if user == nil {
			resp.Diagnostics.AddError("User not found", "Could not find user with login name: "+data.LoginName.ValueString())
			return
		}
	} else {
		// The `ExactlyOneOf` validator should ensure we never reach this point,
		// because if the user doesn't pass an ID or login_name, the plan will
		// be rejected.
		panic("unreachable!")
	}

	data = toUserDataSourceModel(user)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
