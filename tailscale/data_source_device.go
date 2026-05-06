// Copyright (c) David Bond, Tailscale Inc, & Contributors
// SPDX-License-Identifier: MIT

package tailscale

import (
	"context"
	"errors"
	"fmt"
	"maps"
	"time"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"tailscale.com/client/tailscale/v2"
)

// NewSingleDeviceDataSource returns a new single-device data source.
func NewSingleDeviceDataSource() datasource.DataSource {
	return &singleDeviceDataSource{}
}

type singleDeviceDataSource struct {
	DataSourceBase
}

type singleDeviceDataSourceModel struct {
	deviceDataSourceModel

	WaitFor types.String `tfsdk:"wait_for"`
}

// Metadata defines the data source name as it appears in Terraform configurations.
func (d singleDeviceDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_device"
}

// Schema defines a schema describing what data is available in the data source response.
func (d singleDeviceDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	attributes := map[string]schema.Attribute{
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
		"wait_for": schema.StringAttribute{
			Description: "If specified, the provider will make multiple attempts to obtain the data source until the wait_for duration is reached. Retries are made every second so this value should be greater than 1s",
			Optional:    true,
			Validators: []validator.String{
				retryDeadlineValidator{},
			},
		},
	}
	maps.Copy(attributes, deviceSchema)

	resp.Schema = schema.Schema{
		Description: "The device data source describes a single device in a tailnet",
		Attributes:  attributes,
	}
}

// Read fetches the data from the Tailscale API.
func (d singleDeviceDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var device singleDeviceDataSourceModel
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

	err := retryWithDeadline(ctx, poll, deadline, 1*time.Second)
	if err != nil {
		resp.Diagnostics.AddError("Failed to fetch devices", err.Error())
		return
	}

	apiData, diagnostics := toDeviceDataSourceModel(ctx, selected)
	if diagnostics.HasError() {
		resp.Diagnostics.Append(diagnostics...)
		return
	}

	device.deviceDataSourceModel = apiData
	resp.Diagnostics.Append(resp.State.Set(ctx, device)...)
}

// retryWithDeadline calls fn once. If fn errors and maxWait and retryInterval are positive, it retries fn until fn
// succeeds or maxWait elapses, waiting for the duration of retryInterval between attempts.
func retryWithDeadline(ctx context.Context, fn func(context.Context) error, maxWait time.Duration, retryInterval time.Duration) error {
	// Do an initial check in case we don't need to wait at all.
	err := fn(ctx)
	if err == nil {
		return nil
	}
	if maxWait <= 0 || retryInterval <= 0 {
		return err
	}

	maxTicker := time.NewTicker(maxWait)
	defer maxTicker.Stop()
	intervalTicker := time.NewTicker(retryInterval)
	defer intervalTicker.Stop()

	// Check for the data at intervals, until we reach the maximum specified duration.
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-maxTicker.C:
			return fmt.Errorf("failed after maximum of retries within %v: %w", maxWait, err)
		case <-intervalTicker.C:
			err = fn(ctx)
			if err != nil {
				continue
			}
			return nil
		}
	}
}
