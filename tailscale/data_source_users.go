// Copyright (c) David Bond, Tailscale Inc, & Contributors
// SPDX-License-Identifier: MIT

package tailscale

import (
	"context"
	"maps"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"tailscale.com/client/tailscale/v2"
)

// NewMultipleUsersDataSource returns a new multiple-users data data source.
func NewMultipleUsersDataSource() datasource.DataSource {
	return &multipleUsersDataSource{}
}

type multipleUsersDataSource struct {
	DataSourceBase
}

type multipleUsersDataSourceModel struct {
	ID    types.String          `tfsdk:"id"`
	Type  types.String          `tfsdk:"type"`
	Role  types.String          `tfsdk:"role"`
	Users []userDataSourceModel `tfsdk:"users"`
}

// Metadata defines the data source name as it appears in Terraform configurations.
func (d multipleUsersDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_users"
}

// Schema defines a schema describing what data is available in the data source response.
func (d multipleUsersDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	nestedUserAttributes := map[string]schema.Attribute{
		"id": schema.StringAttribute{
			Description: "The unique identifier for the user.",
			Computed:    true,
		},
		"login_name": schema.StringAttribute{
			Description: "The emailish login name of the user.",
			Computed:    true,
		},
	}
	maps.Copy(nestedUserAttributes, userSchema)

	resp.Schema = schema.Schema{
		Description: "The users data source describes a list of users in a tailnet",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed: true,
			},
			"type": schema.StringAttribute{
				Optional:    true,
				Description: "Filter the results to only include users of a specific type. Valid values are `member` or `shared`.",
				Validators: []validator.String{
					stringvalidator.OneOf(
						string(tailscale.UserTypeMember),
						string(tailscale.UserTypeShared),
					),
				},
			},
			"role": schema.StringAttribute{
				Optional:    true,
				Description: "Filter the results to only include users with a specific role. Valid values are `owner`, `member`, `admin`, `it-admin`, `network-admin`, `billing-admin`, and `auditor`.",
				Validators: []validator.String{
					stringvalidator.OneOf(
						string(tailscale.UserRoleOwner),
						string(tailscale.UserRoleMember),
						string(tailscale.UserRoleAdmin),
						string(tailscale.UserRoleITAdmin),
						string(tailscale.UserRoleNetworkAdmin),
						string(tailscale.UserRoleBillingAdmin),
						string(tailscale.UserRoleAuditor),
					),
				},
			},
		},
		Blocks: map[string]schema.Block{
			"users": schema.ListNestedBlock{
				Description: "The list of users in the tailnet",
				NestedObject: schema.NestedBlockObject{
					Attributes: nestedUserAttributes,
				},
			},
		},
	}
}

// Read fetches the data from the Tailscale API.
func (d multipleUsersDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data multipleUsersDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var userType *tailscale.UserType
	if !data.Type.IsNull() {
		userType = new(tailscale.UserType(data.Type.ValueString()))
	}

	var userRole *tailscale.UserRole
	if !data.Role.IsNull() {
		userRole = new(tailscale.UserRole(data.Role.ValueString()))
	}

	users, err := d.Client.Users().List(ctx, userType, userRole)
	if err != nil {
		resp.Diagnostics.AddError("Failed to fetch users", err.Error())
		return
	}

	data.Users = make([]userDataSourceModel, 0, len(users))
	for _, u := range users {
		userData := toUserDataSourceModel(&u)
		data.Users = append(data.Users, userData)
	}

	data.ID = types.StringValue(createUUID())
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
