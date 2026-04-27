// Copyright (c) David Bond, Tailscale Inc, & Contributors
// SPDX-License-Identifier: MIT

package tailscale

import (
	"time"

	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"tailscale.com/client/tailscale/v2"
)

type userDataSourceModel struct {
	ID                 types.String `tfsdk:"id"`
	LoginName          types.String `tfsdk:"login_name"`
	DisplayName        types.String `tfsdk:"display_name"`
	ProfilePicURL      types.String `tfsdk:"profile_pic_url"`
	TailnetID          types.String `tfsdk:"tailnet_id"`
	Created            types.String `tfsdk:"created"`
	Type               types.String `tfsdk:"type"`
	Role               types.String `tfsdk:"role"`
	Status             types.String `tfsdk:"status"`
	DeviceCount        types.Int32  `tfsdk:"device_count"`
	LastSeen           types.String `tfsdk:"last_seen"`
	CurrentlyConnected types.Bool   `tfsdk:"currently_connected"`
}

func (d *userDataSourceModel) setUserProperties(user *tailscale.User) {
	d.ID = types.StringValue(user.ID)
	d.LoginName = types.StringValue(user.LoginName)
	d.DisplayName = types.StringValue(user.DisplayName)
	d.ProfilePicURL = types.StringValue(user.ProfilePicURL)
	d.TailnetID = types.StringValue(user.TailnetID)
	d.Created = types.StringValue(user.Created.Format(time.RFC3339))
	d.Type = types.StringValue(string(user.Type))
	d.Role = types.StringValue(string(user.Role))
	d.Status = types.StringValue(string(user.Status))
	d.DeviceCount = types.Int32Value(int32(user.DeviceCount))
	d.LastSeen = types.StringValue(user.LastSeen.Format(time.RFC3339))
	d.CurrentlyConnected = types.BoolValue(user.CurrentlyConnected)
}

var userSchema = map[string]schema.Attribute{
	"display_name": schema.StringAttribute{
		Description: "The name of the user.",
		Computed:    true,
	},
	"profile_pic_url": schema.StringAttribute{
		Description: "The profile pic URL for the user.",
		Computed:    true,
	},
	"tailnet_id": schema.StringAttribute{
		Description: "The tailnet that owns the user.",
		Computed:    true,
	},
	"created": schema.StringAttribute{
		Description: "The time the user joined their tailnet.",
		Computed:    true,
	},
	"type": schema.StringAttribute{
		Description: "The type of relation this user has to the tailnet associated with the request.",
		Computed:    true,
	},
	"role": schema.StringAttribute{
		Description: "The role of the user.",
		Computed:    true,
	},
	"status": schema.StringAttribute{
		Description: "The status of the user.",
		Computed:    true,
	},
	"device_count": schema.Int32Attribute{
		Description: "Number of devices the user owns.",
		Computed:    true,
	},
	"last_seen": schema.StringAttribute{
		Description: "The later of either: a) The last time any of the user's nodes were connected to the network or b) The last time the user authenticated to any tailscale service, including the admin panel.",
		Computed:    true,
	},
	"currently_connected": schema.BoolAttribute{
		Description: "true when the user has a node currently connected to the control server.",
		Computed:    true,
	},
}
