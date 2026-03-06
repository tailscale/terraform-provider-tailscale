// Copyright (c) David Bond, Tailscale Inc, & Contributors
// SPDX-License-Identifier: MIT

package tailscale

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"tailscale.com/client/tailscale/v2"
)

// DataSourceBase is a base struct for all Tailscale data sources.
//
// All data sources should extend this struct, then the authenticated [Client]
// will be available in their [datasource.DataSource.Read] method.
type DataSourceBase struct {
	Client *tailscale.Client
}

// Configure attaches the client to the data source, so it can be used in the
// [datasource.DataSource.Read] method.
func (d *DataSourceBase) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	client, ok := req.ProviderData.(*tailscale.Client)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Data Source Configure Type",
			fmt.Sprintf(
				"Expected *tailscale.Client, got: %T. Please report this error at https://github.com/tailscale/tailscale.",
				req.ProviderData),
		)
		return
	}

	d.Client = client
}
