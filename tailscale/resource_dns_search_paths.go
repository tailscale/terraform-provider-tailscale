// Copyright (c) David Bond, Tailscale Inc, & Contributors
// SPDX-License-Identifier: MIT

package tailscale

import (
	"context"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"

	tsclient "github.com/tailscale/tailscale-client-go/v2"
)

func resourceDNSSearchPaths() *schema.Resource {
	return &schema.Resource{
		Description:   "The dns_nameservers resource allows you to configure DNS nameservers for your Tailscale network. See https://tailscale.com/kb/1054/dns for more information.",
		ReadContext:   resourceDNSSearchPathsRead,
		UpdateContext: resourceDNSSearchPathsUpdate,
		DeleteContext: resourceDNSSearchPathsDelete,
		CreateContext: resourceDNSSearchPathsCreate,
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},
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
	client := m.(*tsclient.Client)
	paths, err := client.DNS().SearchPaths(ctx)
	if err != nil {
		return diagnosticsError(err, "Failed to fetch dns search paths")
	}

	if err = d.Set("search_paths", paths); err != nil {
		return diag.FromErr(err)
	}

	return nil
}

func resourceDNSSearchPathsCreate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	client := m.(*tsclient.Client)
	paths := d.Get("search_paths").([]interface{})

	searchPaths := make([]string, len(paths))
	for i, path := range paths {
		searchPaths[i] = path.(string)
	}

	if err := client.DNS().SetSearchPaths(ctx, searchPaths); err != nil {
		return diagnosticsError(err, "Failed to fetch set search paths")
	}

	d.SetId(createUUID())
	return resourceDNSSearchPathsRead(ctx, d, m)
}

func resourceDNSSearchPathsUpdate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	if !d.HasChange("search_paths") {
		return resourceDNSSearchPathsRead(ctx, d, m)
	}

	client := m.(*tsclient.Client)
	paths := d.Get("search_paths").([]interface{})

	searchPaths := make([]string, len(paths))
	for i, path := range paths {
		searchPaths[i] = path.(string)
	}

	if err := client.DNS().SetSearchPaths(ctx, searchPaths); err != nil {
		return diagnosticsError(err, "Failed to fetch set search paths")
	}

	return resourceDNSSearchPathsRead(ctx, d, m)
}

func resourceDNSSearchPathsDelete(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	client := m.(*tsclient.Client)

	if err := client.DNS().SetSearchPaths(ctx, []string{}); err != nil {
		return diagnosticsError(err, "Failed to fetch set search paths")
	}

	return nil
}
