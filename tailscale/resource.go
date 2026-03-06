// Copyright (c) David Bond, Tailscale Inc, & Contributors
// SPDX-License-Identifier: MIT

package tailscale

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"tailscale.com/client/tailscale/v2"
)

// ResourceBase is a base struct for all Tailscale resources.
//
// All resources should extend this struct, then the authenticated [Client] will
// be available in their CRUD methods.
type ResourceBase struct {
	Client *tailscale.Client
}

// Configure attaches the client to the resource, so it can be used in the
// CRUD methods.
func (d *ResourceBase) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	client, ok := req.ProviderData.(*tailscale.Client)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Resource Configure Type",
			fmt.Sprintf(
				"Expected *tailscale.Client, got: %T. Please report this error at https://github.com/tailscale/tailscale.",
				req.ProviderData),
		)
		return
	}

	d.Client = client
}
