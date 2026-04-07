// Copyright (c) David Bond, Tailscale Inc, & Contributors
// SPDX-License-Identifier: MIT

package tailscale

import (
	"context"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"

	"tailscale.com/client/tailscale/v2"
)

func dataSourceService() *schema.Resource {
	return &schema.Resource{
		Description: "The Service data source describes a single Service in a tailnet. See https://tailscale.com/docs/features/tailscale-services for more information.",
		ReadContext: dataSourceServiceRead,
		Schema: map[string]*schema.Schema{
			"name": {
				Type:        schema.TypeString,
				Description: "The name of the Service (e.g. `svc:my-service`).",
				Required:    true,
			},
			"id": {
				Type:        schema.TypeString,
				Description: "The Service name, e.g. 'svc:my-service'.",
				Computed:    true,
			},
			"addrs": {
				Type:        schema.TypeList,
				Description: "The IP addresses assigned to the Service.",
				Computed:    true,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
			},
			"comment": {
				Type:        schema.TypeString,
				Description: "A comment describing the Service.",
				Computed:    true,
			},
			"ports": {
				Type:        schema.TypeList,
				Description: "The ports that the Service listens on.",
				Computed:    true,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
			},
			"tags": {
				Type:        schema.TypeSet,
				Description: "The ACL tags applied to the Service.",
				Computed:    true,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
			},
		},
	}
}

func dataSourceServiceRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	client := m.(*tailscale.Client)

	name, ok := d.GetOk("name")
	if !ok {
		return diag.Errorf("A Service `name` is required")
	}
	svc, err := client.VIPServices().Get(ctx, name.(string))
	if err != nil {
		return diagnosticsError(err, "Failed to fetch Service %q", name)
	}

	d.SetId(svc.Name)
	return setProperties(d, map[string]any{
		"name":    svc.Name,
		"addrs":   svc.Addrs,
		"comment": svc.Comment,
		"ports":   svc.Ports,
		"tags":    svc.Tags,
	})
}
