// Copyright (c) David Bond, Tailscale Inc, & Contributors
// SPDX-License-Identifier: MIT

package tailscale

import (
	"context"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"tailscale.com/client/tailscale/v2"
	"tailscale.com/types/bools"
)

type deviceDataSourceModel struct {
	Hostname types.String `tfsdk:"hostname"`
	Name     types.String `tfsdk:"name"`

	Addresses                 types.List   `tfsdk:"addresses"`
	Authorized                types.Bool   `tfsdk:"authorized"`
	BlocksIncomingConnections types.Bool   `tfsdk:"blocks_incoming_connections"`
	ClientVersion             types.String `tfsdk:"client_version"`
	Created                   types.String `tfsdk:"created"`
	Expires                   types.String `tfsdk:"expires"`
	ID                        types.String `tfsdk:"id"`
	IsExternal                types.Bool   `tfsdk:"is_external"`
	KeyExpiryDisabled         types.Bool   `tfsdk:"key_expiry_disabled"`
	LastSeen                  types.String `tfsdk:"last_seen"` // Will be nil if connected_to_control is true.
	MachineKey                types.String `tfsdk:"machine_key"`
	NodeID                    types.String `tfsdk:"node_id"` // The preferred identifier for a device.
	NodeKey                   types.String `tfsdk:"node_key"`
	OS                        types.String `tfsdk:"os"`
	Tags                      types.Set    `tfsdk:"tags"`
	TailnetLockError          types.String `tfsdk:"tailnet_lock_error"`
	TailnetLockKey            types.String `tfsdk:"tailnet_lock_key"`
	UpdateAvailable           types.Bool   `tfsdk:"update_available"`
	User                      types.String `tfsdk:"user"`
}

func toDeviceDataSourceModel(ctx context.Context, device *tailscale.Device) (deviceDataSourceModel, diag.Diagnostics) {
	var lastSeen string
	if device.LastSeen == nil {
		lastSeen = ""
	} else {
		lastSeen = device.LastSeen.Format(time.RFC3339)
	}

	data := deviceDataSourceModel{
		Hostname: types.StringValue(device.Hostname),
		Name:     types.StringValue(device.Name),

		Authorized:                types.BoolValue(device.Authorized),
		BlocksIncomingConnections: types.BoolValue(device.BlocksIncomingConnections),
		ClientVersion:             types.StringValue(device.ClientVersion),
		Created:                   types.StringValue(device.Created.Format(time.RFC3339)),
		Expires:                   types.StringValue(device.Expires.Format(time.RFC3339)),
		ID:                        types.StringValue(device.ID),
		IsExternal:                types.BoolValue(device.IsExternal),
		KeyExpiryDisabled:         types.BoolValue(device.KeyExpiryDisabled),
		LastSeen:                  types.StringValue(lastSeen),
		MachineKey:                types.StringValue(device.MachineKey),
		NodeID:                    types.StringValue(device.NodeID),
		NodeKey:                   types.StringValue(device.NodeKey),
		OS:                        types.StringValue(device.OS),
		TailnetLockError:          types.StringValue(device.TailnetLockError),
		TailnetLockKey:            types.StringValue(device.TailnetLockKey),
		UpdateAvailable:           types.BoolValue(device.UpdateAvailable),
		User:                      types.StringValue(device.User),
	}

	addresses, diagnostics := types.ListValueFrom(ctx, types.StringType, device.Addresses)
	if diagnostics.HasError() {
		return deviceDataSourceModel{}, diagnostics
	}
	data.Addresses = addresses

	// Normalize nil to empty slice before converting it to plugin framework
	// types.
	// This is to avoid drift when upgrading from a provider version that
	// is still backed by SDKv2.
	deviceTags := bools.IfElse(device.Tags != nil, device.Tags, []string{})

	tags, diagnostics := types.SetValueFrom(ctx, types.StringType, deviceTags)
	if diagnostics.HasError() {
		return deviceDataSourceModel{}, diagnostics
	}
	data.Tags = tags

	return data, diag.Diagnostics{}
}

var deviceSchema = map[string]schema.Attribute{
	"user": schema.StringAttribute{
		Description: "The user associated with the device",
		Computed:    true,
	},
	"node_id": schema.StringAttribute{
		Description: "The preferred indentifier for a device.",
		Computed:    true,
	},
	"addresses": schema.ListAttribute{
		Description: "The list of device's IPs",
		Computed:    true,
		ElementType: types.StringType,
	},
	"tags": schema.SetAttribute{
		Description: "The tags applied to the device",
		Computed:    true,
		ElementType: types.StringType,
	},
	"authorized": schema.BoolAttribute{
		Description: "Whether the device is authorized to access the tailnet",
		Computed:    true,
	},
	"key_expiry_disabled": schema.BoolAttribute{
		Description: "Whether the device's key expiry is disabled",
		Computed:    true,
	},
	"blocks_incoming_connections": schema.BoolAttribute{
		Description: "Whether the device blocks incoming connections",
		Computed:    true,
	},
	"client_version": schema.StringAttribute{
		Description: "The Tailscale client version running on the device",
		Computed:    true,
	},
	"created": schema.StringAttribute{
		Description: "The creation time of the device",
		Computed:    true,
	},
	"expires": schema.StringAttribute{
		Description: "The expiry time of the device's key",
		Computed:    true,
	},
	"id": schema.StringAttribute{
		Description: "The ID of this resource.",
		Computed:    true,
	},
	"is_external": schema.BoolAttribute{
		Description: "Whether the device is marked as external",
		Computed:    true,
	},
	"last_seen": schema.StringAttribute{
		Description: "The last seen time of the device",
		Computed:    true,
	},
	"machine_key": schema.StringAttribute{
		Description: "The machine key of the device",
		Computed:    true,
	},
	"node_key": schema.StringAttribute{
		Description: "The node key of the device",
		Computed:    true,
	},
	"os": schema.StringAttribute{
		Description: "The operating system of the device",
		Computed:    true,
	},
	"update_available": schema.BoolAttribute{
		Description: "Whether an update is available for the device",
		Computed:    true,
	},
	"tailnet_lock_error": schema.StringAttribute{
		Description: "The tailnet lock error for the device, if any",
		Computed:    true,
	},
	"tailnet_lock_key": schema.StringAttribute{
		Description: "The tailnet lock key for the device, if any",
		Computed:    true,
	},
}
