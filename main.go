//go:generate tfplugindocs
package main

import (
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/plugin"

	"github.com/tailscale/terraform-provider-tailscale/tailscale"
)

// version is filled by goreleaser at build time.
var version = "dev"

func main() {
	plugin.Serve(&plugin.ServeOpts{
		ProviderFunc: func() *schema.Provider {
			return tailscale.Provider(addUserAgent)
		},
	})
}

// addUserAgent adds a `user_agent` configuration key to the provider with a
// default value based on provider version.
func addUserAgent(p *schema.Provider) {
	p.Schema["user_agent"] = &schema.Schema{
		Type:        schema.TypeString,
		Default:     p.UserAgent("terraform-provider-tailscale", version),
		Optional:    true,
		Description: "User-Agent header for API requests.",
	}
}
