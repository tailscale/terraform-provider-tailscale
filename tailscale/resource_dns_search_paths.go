package tailscale

import (
	"context"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"

	"github.com/davidsbond/tailscale-client-go/tailscale"
)

func resourceDNSSearchPaths() *schema.Resource {
	return &schema.Resource{
		Description:   "The dns_nameservers resource allows you to configure DNS nameservers for your Tailscale network. See https://tailscale.com/kb/1054/dns for more information.",
		ReadContext:   resourceDNSSearchPathsRead,
		UpdateContext: resourceDNSSearchPathsUpdate,
		DeleteContext: resourceDNSSearchPathsDelete,
		CreateContext: resourceDNSSearchPathsCreate,
		Schema: map[string]*schema.Schema{
			"search_paths": {
				Type: schema.TypeList,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
				Required:    true,
				Description: "Devices on your network will use these domain suffixes to resolve DNS names.",
			},
		},
	}
}

func resourceDNSSearchPathsRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	client := m.(*tailscale.Client)
	paths, err := client.DNSSearchPaths(ctx)
	if err != nil {
		return diagnosticsError(err, "Failed to fetch dns search paths")
	}

	if err = d.Set("search_paths", paths); err != nil {
		return diag.FromErr(err)
	}

	return nil
}

func resourceDNSSearchPathsCreate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	client := m.(*tailscale.Client)
	paths := d.Get("search_paths").([]interface{})

	searchPaths := make([]string, len(paths))
	for i, path := range paths {
		searchPaths[i] = path.(string)
	}

	if err := client.SetDNSSearchPaths(ctx, searchPaths); err != nil {
		return diagnosticsError(err, "Failed to fetch set search paths")
	}

	d.SetId(createUUID())
	return nil
}

func resourceDNSSearchPathsUpdate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	if !d.HasChange("search_paths") {
		return resourceDNSSearchPathsRead(ctx, d, m)
	}

	client := m.(*tailscale.Client)
	paths := d.Get("search_paths").([]interface{})

	searchPaths := make([]string, len(paths))
	for i, path := range paths {
		searchPaths[i] = path.(string)
	}

	if err := client.SetDNSSearchPaths(ctx, searchPaths); err != nil {
		return diagnosticsError(err, "Failed to fetch set search paths")
	}

	return nil
}

func resourceDNSSearchPathsDelete(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	client := m.(*tailscale.Client)

	if err := client.SetDNSSearchPaths(ctx, []string{}); err != nil {
		return diagnosticsError(err, "Failed to fetch set search paths")
	}

	return nil
}
