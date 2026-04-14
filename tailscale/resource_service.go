// Copyright (c) David Bond, Tailscale Inc, & Contributors
// SPDX-License-Identifier: MIT

package tailscale

import (
	"context"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"

	"tailscale.com/client/tailscale/v2"
)

func resourceService() *schema.Resource {
	return &schema.Resource{
		Description:   "The Service resource allows you to manage Tailscale Services in your Tailscale network. Services let you publish internal resources (like databases or web servers) as named resources in your tailnet. Services provide a stable MagicDNS name, a Tailscale virtual IP address pair, can be served by multiple nodes, and are valid access control destinations. See https://tailscale.com/docs/features/tailscale-services) for more information.",
		ReadContext:   resourceServiceRead,
		CreateContext: resourceServiceCreate,
		UpdateContext: resourceServiceUpdate,
		DeleteContext: resourceServiceDelete,
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},
		Schema: map[string]*schema.Schema{
			"name": {
				Type:        schema.TypeString,
				Description: "The name of the Service. Must begin with `svc:`.",
				Required:    true,
				ForceNew:    true,
			},
			"id": {
				Type: schema.TypeString,
				// The ID must be predictable to support importing existing
				// Services, e.g. 'terraform import tailscale_service.my_service
				// svc:my-service'. The Service name will be a known value and
				// is the ID used by the API anyway.
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
				Description: "An optional comment describing the Service.",
				Optional:    true,
			},
			"ports": {
				Type:        schema.TypeSet,
				Description: "A list of protocol:port pairs to be exposed by the Service. The only supported protocol is \"tcp\" at this time. \"do-not-validate\" can be used to skip validation.",
				Required:    true,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
			},
			"tags": {
				Type:        schema.TypeSet,
				Description: "The ACL tags applied to the Service.",
				Optional:    true,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
			},
		},
	}
}

func resourceServiceCreate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	client := m.(*tailscale.Client)

	svc := buildServiceFromResource(d)
	if err := client.VIPServices().CreateOrUpdate(ctx, svc); err != nil {
		return diagnosticsError(err, "Failed to create Service")
	}

	d.SetId(svc.Name)
	return resourceServiceRead(ctx, d, m)
}

func resourceServiceRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	client := m.(*tailscale.Client)

	svc, err := client.VIPServices().Get(ctx, d.Id())
	if err != nil {
		if tailscale.IsNotFound(err) {
			d.SetId("")
			return nil
		}
		return diagnosticsError(err, "Failed to fetch Service %q", d.Id())
	}

	return setProperties(d, map[string]any{
		"name":    svc.Name,
		"addrs":   svc.Addrs,
		"comment": svc.Comment,
		"ports":   svc.Ports,
		"tags":    svc.Tags,
	})
}

func resourceServiceUpdate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	client := m.(*tailscale.Client)

	svc := buildServiceFromResource(d)
	if err := client.VIPServices().CreateOrUpdate(ctx, svc); err != nil {
		return diagnosticsError(err, "Failed to update Service %q", d.Id())
	}

	return resourceServiceRead(ctx, d, m)
}

func resourceServiceDelete(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	client := m.(*tailscale.Client)

	if err := client.VIPServices().Delete(ctx, d.Id()); err != nil && !tailscale.IsNotFound(err) {
		return diagnosticsError(err, "Failed to delete Service %q", d.Id())
	}

	return nil
}

func buildServiceFromResource(d *schema.ResourceData) tailscale.VIPService {
	svc := tailscale.VIPService{
		Name:    d.Get("name").(string),
		Comment: d.Get("comment").(string),
	}

	if v, ok := d.GetOk("addrs"); ok {
		addrs := v.([]interface{})
		svc.Addrs = make([]string, len(addrs))
		for i, a := range addrs {
			svc.Addrs[i] = a.(string)
		}
	}

	if v, ok := d.GetOk("ports"); ok {
		ports := v.(*schema.Set).List()
		svc.Ports = make([]string, len(ports))
		for i, p := range ports {
			svc.Ports[i] = p.(string)
		}
	}

	if v, ok := d.GetOk("tags"); ok {
		tags := v.(*schema.Set).List()
		svc.Tags = make([]string, len(tags))
		for i, t := range tags {
			svc.Tags[i] = t.(string)
		}
	}

	return svc
}
