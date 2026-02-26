// Copyright (c) David Bond, Tailscale Inc, & Contributors
// SPDX-License-Identifier: MIT

package tailscale

import (
	"context"
	"slices"
	"strings"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"

	"tailscale.com/client/tailscale/v2"
)

func resourceDNSConfiguration() *schema.Resource {
	return &schema.Resource{
		Description:   "The dns_configuration resource allows you to manage the complete DNS configuration for your Tailscale network. See https://tailscale.com/kb/1054/dns for more information.",
		ReadContext:   resourceDNSConfigurationRead,
		CreateContext: resourceDNSConfigurationCreate,
		UpdateContext: resourceDNSConfigurationUpdate,
		DeleteContext: resourceDNSConfigurationDelete,
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},
		Schema: map[string]*schema.Schema{
			"nameservers": {
				Description: "Set the nameservers used by devices on your network to resolve DNS queries. `override_local_dns` must also be true to prefer these nameservers over local DNS configuration.",
				Type:        schema.TypeList,
				Optional:    true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"address": {
							Description: "The nameserver's IPv4 or IPv6 address",
							Type:        schema.TypeString,
							Required:    true,
						},
						"use_with_exit_node": {
							Description: "This nameserver will continue to be used when an exit node is selected (requires Tailscale v1.88.1 or later). Defaults to false.",
							Type:        schema.TypeBool,
							Optional:    true,
							Default:     false,
						},
					},
				},
			},
			"split_dns": {
				Description: "Set the nameservers used by devices on your network to resolve DNS queries on specific domains (requires Tailscale v1.8 or later). Configuration does not depend on `override_local_dns`.",
				Type:        schema.TypeList,
				Optional:    true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"domain": {
							Description: "The nameservers will be used only for this domain.",
							Type:        schema.TypeString,
							Required:    true,
						},
						"nameservers": {
							Description: "Set the nameservers used by devices on your network to resolve DNS queries.",
							Type:        schema.TypeList,
							Required:    true,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"address": {
										Description: "The nameserver's IPv4 or IPv6 address.",
										Type:        schema.TypeString,
										Required:    true,
									},
									"use_with_exit_node": {
										Description: "This nameserver will continue to be used when an exit node is selected (requires Tailscale v1.88.1 or later). Defaults to false.",
										Type:        schema.TypeBool,
										Optional:    true,
										Default:     false,
									},
								},
							},
						},
					},
				},
			},
			"search_paths": {
				Description: "Additional search domains. When MagicDNS is on, the tailnet domain is automatically included as the first search domain.",
				Type:        schema.TypeList,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
				Optional: true,
			},
			"override_local_dns": {
				Description: "When enabled, use the configured DNS servers in `nameservers` to resolve names outside the tailnet. When disabled, devices will prefer their local DNS configuration. Defaults to false.",
				Type:        schema.TypeBool,
				Optional:    true,
				Default:     false,
			},
			"magic_dns": {
				Description: "Whether or not to enable MagicDNS. Defaults to true.",
				Type:        schema.TypeBool,
				Optional:    true,
				Default:     true,
			},
			"magic_dns_name": {
				Description: "The tailnet/MagicDNS domain name. Null if disabled or undeterminable.",
				Type:        schema.TypeString,
				Computed:    true,
			},
		},
	}
}

func resourceDNSConfigurationRead(ctx context.Context, d *schema.ResourceData, m any) diag.Diagnostics {
	client := m.(*tailscale.Client)

	configuration, err := client.DNS().Configuration(ctx)
	if err != nil {
		return diagnosticsError(err, "Failed to fetch dns configuration")
	}

	nameservers := updateNameservers(d.Get("nameservers").([]any), configuration.Nameservers)
	// Read existing SplitDNS to preserve order in TF resource
	splitDNS := make([]map[string]any, 0, len(configuration.SplitDNS))
	for _, _nameserversWithDomain := range d.Get("split_dns").([]any) {
		nameserversWithDomain := _nameserversWithDomain.(map[string]any)
		domain := nameserversWithDomain["domain"].(string)
		nameservers, found := configuration.SplitDNS[domain]
		if found {
			splitDNS = append(splitDNS, map[string]any{
				"domain":      domain,
				"nameservers": updateNameservers(nameserversWithDomain["nameservers"].([]any), nameservers),
			})
			delete(configuration.SplitDNS, domain)
		}
	}

	// Add new SplitDNS
	for domain, nameserversForDomain := range configuration.SplitDNS {
		splitDNS = append(splitDNS, map[string]any{
			"domain":      domain,
			"nameservers": updateNameservers(nil, nameserversForDomain),
		})
	}

	var magicDNSName string
	var diags diag.Diagnostics
	if configuration.Preferences.MagicDNS {
		devices, err := client.Devices().List(ctx)
		if err != nil {
			diags = append(diags, diagnosticsError(err, "There is a MagicDNS name, but we failed to get devices")...)
		} else if len(devices) == 0 {
			diags = append(diags, diag.Diagnostic{
				Severity: diag.Warning,
				Summary: "There is a MagicDNS name, but we can't determine it with 0 devices",
			})
		} else {
			parts := strings.Split(devices[0].Name, ".")
			if len(parts) != 4 {
				diags = append(diags, diag.Diagnostic{
					Severity: diag.Error,
					Summary: "There is a MagicDNS name, but unexpected device name format",
				})
			} else {
				magicDNSName = strings.Join(parts[1:], ".")
			}
		}
	}

	diags = append(diags, setProperties(d, map[string]any{
		"nameservers":        nameservers,
		"split_dns":          splitDNS,
		"search_paths":       configuration.SearchPaths,
		"override_local_dns": configuration.Preferences.OverrideLocalDNS,
		"magic_dns":          configuration.Preferences.MagicDNS,
		"magic_dns_name":     magicDNSName,
	})...)

	if diags != nil {
		return diags
	}

	return []diag.Diagnostic{
		{
			Severity: diag.Warning,
			Summary:  "The tailscale_dns_configuration resource is currently in alpha and subject to change, proceed with caution.",
		},
	}
}

// updateNameservers updates an existing list of nameservers with an updated list of nameservers,
// preserving the original ordering of any retained existing nameservers.
func updateNameservers(existing []any, updates []tailscale.DNSConfigurationResolver) []map[string]any {
	nameservers := make([]map[string]any, 0, len(updates))

	// Update existing in place
	for _, _nameserver := range existing {
		nameserver := _nameserver.(map[string]any)
		idx, found := slices.BinarySearchFunc(updates, nameserver["address"].(string), func(a tailscale.DNSConfigurationResolver, b string) int {
			return strings.Compare(a.Address, b)
		})
		if found {
			nameservers = append(nameservers, nameserverToMap(updates[idx]))
			updates = slices.Delete(updates, idx, idx+1)
		}
	}

	// Append new
	for _, nameserver := range updates {
		nameservers = append(nameservers, map[string]any{
			"address":            nameserver.Address,
			"use_with_exit_node": nameserver.UseWithExitNode,
		})
	}

	return nameservers
}

func nameserverToMap(nameserver tailscale.DNSConfigurationResolver) map[string]any {
	return map[string]any{
		"address":            nameserver.Address,
		"use_with_exit_node": nameserver.UseWithExitNode,
	}
}

func resourceDNSConfigurationCreate(ctx context.Context, d *schema.ResourceData, m any) diag.Diagnostics {
	if d := resourceDNSConfigurationSet(ctx, d, m); d != nil {
		return d
	}
	d.SetId(createUUID())
	return resourceDNSConfigurationRead(ctx, d, m)
}

func resourceDNSConfigurationUpdate(ctx context.Context, d *schema.ResourceData, m any) diag.Diagnostics {
	if d := resourceDNSConfigurationSet(ctx, d, m); d != nil {
		return d
	}
	return resourceDNSConfigurationRead(ctx, d, m)
}

func resourceDNSConfigurationSet(ctx context.Context, d *schema.ResourceData, m any) diag.Diagnostics {
	client := m.(*tailscale.Client)
	configuration := tailscale.DNSConfiguration{
		SplitDNS: make(map[string][]tailscale.DNSConfigurationResolver),
		Preferences: tailscale.DNSConfigurationPreferences{
			OverrideLocalDNS: d.Get("override_local_dns").(bool),
			MagicDNS:         d.Get("magic_dns").(bool),
		},
	}

	for _, _nameserver := range d.Get("nameservers").([]any) {
		nameserver := _nameserver.(map[string]any)
		configuration.Nameservers = append(configuration.Nameservers, tailscale.DNSConfigurationResolver{
			Address:         nameserver["address"].(string),
			UseWithExitNode: nameserver["use_with_exit_node"].(bool),
		})
	}

	for _, _splitDNS := range d.Get("split_dns").([]any) {
		splitDNS := _splitDNS.(map[string]any)
		domain := splitDNS["domain"].(string)
		var nameservers []tailscale.DNSConfigurationResolver
		for _, _nameserver := range splitDNS["nameservers"].([]any) {
			nameserver := _nameserver.(map[string]any)
			nameservers = append(nameservers, tailscale.DNSConfigurationResolver{
				Address:         nameserver["address"].(string),
				UseWithExitNode: nameserver["use_with_exit_node"].(bool),
			})
		}
		configuration.SplitDNS[domain] = nameservers
	}

	for _, searchPath := range d.Get("search_paths").([]any) {
		configuration.SearchPaths = append(configuration.SearchPaths, searchPath.(string))
	}

	if err := client.DNS().SetConfiguration(ctx, configuration); err != nil {
		return diagnosticsError(err, "Failed to set dns configuration")
	}

	return nil
}

func resourceDNSConfigurationDelete(ctx context.Context, d *schema.ResourceData, m any) diag.Diagnostics {
	client := m.(*tailscale.Client)

	if err := client.DNS().SetConfiguration(ctx, tailscale.DNSConfiguration{}); err != nil {
		return diagnosticsError(err, "Failed to delete dns configuration")
	}

	return nil
}
