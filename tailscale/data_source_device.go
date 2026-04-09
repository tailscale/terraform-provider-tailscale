// Copyright (c) David Bond, Tailscale Inc, & Contributors
// SPDX-License-Identifier: MIT

package tailscale

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"tailscale.com/client/tailscale/v2"
	"tailscale.com/types/bools"
)

type deviceDataSourceModel struct {
	Hostname types.String `tfsdk:"hostname"`
	Name     types.String `tfsdk:"name"`
	WaitFor  types.String `tfsdk:"wait_for"`

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

// NewDeviceDataSource returns a new Device data source.
func NewDeviceDataSource() datasource.DataSource {
	return &deviceDataSource{}
}

type deviceDataSource struct {
	DataSourceBase
}

// Metadata defines the data source name as it appears in Terraform configurations.
func (d deviceDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_device"
}

// Schema defines a schema describing what data is available in the data source response.
func (d deviceDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "The device data source describes a single device in a tailnet",
		Attributes: map[string]schema.Attribute{
			"name": schema.StringAttribute{
				Description: "The full name of the device (e.g. `hostname.domain.ts.net`)",
				Optional:    true,
				Validators: []validator.String{
					stringvalidator.ExactlyOneOf(path.Expressions{
						path.MatchRoot("name"),
						path.MatchRoot("hostname"),
					}...),
				},
			},
			"hostname": schema.StringAttribute{
				Description: "The short hostname of the device",
				Optional:    true,
				Validators: []validator.String{
					stringvalidator.ExactlyOneOf(path.Expressions{
						path.MatchRoot("name"),
						path.MatchRoot("hostname"),
					}...),
				},
			},
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
			"wait_for": schema.StringAttribute{
				Description: "If specified, the provider will make multiple attempts to obtain the data source until the wait_for duration is reached. Retries are made every second so this value should be greater than 1s",
				Optional:    true,
				Validators: []validator.String{
					retryDeadlineValidator{},
				},
			},
		},
	}
}

type retryDeadlineValidator struct {
}

func (r retryDeadlineValidator) Description(_ context.Context) string {
	return "Validates that the value is a duration greater than 1s."
}

func (r retryDeadlineValidator) MarkdownDescription(ctx context.Context) string {
	return r.Description(ctx)
}

func (r retryDeadlineValidator) ValidateString(_ context.Context, req validator.StringRequest, resp *validator.StringResponse) {
	if req.ConfigValue.IsNull() {
		return
	}
	waitFor, err := time.ParseDuration(req.ConfigValue.ValueString())
	switch {
	case err != nil:
		resp.Diagnostics.AddAttributeError(req.Path, "failed to parse wait_for", "not a duration")
	case waitFor <= time.Second:
		resp.Diagnostics.AddAttributeError(req.Path, "failed to parse wait_for", "wait_for must be greater than 1 second")
	default:
	}
}

// Read fetches the data from the Tailscale API.
func (d deviceDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var device deviceDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &device)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var filter tailscale.ListDevicesOptions
	var filterDesc string

	if !device.Name.IsNull() {
		filter = tailscale.WithFilter("name", []string{device.Name.ValueString()})
		filterDesc = fmt.Sprintf("name=%q", device.Name.ValueString())
	}

	if !device.Hostname.IsNull() {
		filter = tailscale.WithFilter("hostname", []string{device.Hostname.ValueString()})
		filterDesc = fmt.Sprintf("hostname=%q", device.Hostname.ValueString())
	}

	var deadline time.Duration
	if !device.WaitFor.IsNull() {
		parsed, err := time.ParseDuration(device.WaitFor.ValueString())
		if err != nil {
			resp.Diagnostics.AddError("Failed to parse wait_for", err.Error())
			return
		}
		deadline = parsed
	}

	var selected *tailscale.Device
	poll := func(ctx context.Context) error {
		devices, err := d.Client.Devices().List(ctx, filter)
		if err != nil {
			return err
		}

		if len(devices) == 0 {
			return errors.New("could not find device with" + filterDesc)
		}

		selected = &devices[0]
		return nil
	}

	err := retryWithDeadline(ctx, deadline, 1*time.Second, poll)
	if err != nil {
		resp.Diagnostics.AddError("Failed to fetch devices", err.Error())
		return
	}

	data, diagnostics := toDeviceDataSourceModel(ctx, selected)
	if diagnostics.HasError() {
		resp.Diagnostics.Append(diagnostics...)
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, data)...)
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

// deviceToMap converts the given device into a map representing the device as a
// resource in Terraform. This omits the "id" which is expected to be set
// using [schema.ResourceData.SetId].
func deviceToMap(device *tailscale.Device) map[string]any {
	var lastSeen string
	if device.LastSeen == nil {
		lastSeen = ""
	} else {
		lastSeen = device.LastSeen.Format(time.RFC3339)
	}

	return map[string]any{
		"name":                        device.Name,
		"hostname":                    device.Hostname,
		"user":                        device.User,
		"node_id":                     device.NodeID,
		"addresses":                   device.Addresses,
		"tags":                        device.Tags,
		"authorized":                  device.Authorized,
		"key_expiry_disabled":         device.KeyExpiryDisabled,
		"blocks_incoming_connections": device.BlocksIncomingConnections,
		"client_version":              device.ClientVersion,
		"created":                     device.Created.Format(time.RFC3339),
		"expires":                     device.Expires.Format(time.RFC3339),
		"is_external":                 device.IsExternal,
		"last_seen":                   lastSeen,
		"machine_key":                 device.MachineKey,
		"node_key":                    device.NodeKey,
		"os":                          device.OS,
		"update_available":            device.UpdateAvailable,
		"tailnet_lock_error":          device.TailnetLockError,
		"tailnet_lock_key":            device.TailnetLockKey,
	}
}
