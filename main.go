// Copyright (c) David Bond, Tailscale Inc, & Contributors
// SPDX-License-Identifier: MIT

//go:generate go run github.com/hashicorp/terraform-plugin-docs/cmd/tfplugindocs
package main

import (
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/plugin"

	"github.com/tailscale/terraform-provider-tailscale/tailscale"
)

func main() {
	plugin.Serve(&plugin.ServeOpts{
		ProviderFunc: func() *schema.Provider {
			return tailscale.Provider()
		},
	})
}
