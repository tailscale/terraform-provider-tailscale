package tailscale

import (
	"context"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"

	"github.com/tailscale/tailscale-client-go/tailscale"
)

func resourceDNSSplitNameservers() *schema.Resource {
	return &schema.Resource{
		Description:   "The dns_split_nameservers resource allows you to configure split DNS nameservers for your Tailscale network. See https://tailscale.com/kb/1054/dns for more information.",
		ReadContext:   resourceSplitDNSNameserversRead,
		CreateContext: resourceSplitDNSNameserversCreate,
		UpdateContext: resourceSplitDNSNameserversUpdate,
		DeleteContext: resourceSplitDNSNameserversDelete,
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},
		Schema: map[string]*schema.Schema{
			"domain": {
				Type:        schema.TypeString,
				Description: "Domain to configure split DNS for. Requests for this domain will be resolved using the provided nameservers.",
				Required:    true,
			},
			"nameservers": {
				Type:        schema.TypeSet,
				Description: "Devices on your network will use these nameservers to resolve DNS names. IPv4 or IPv6 addresses are accepted.",
				Required:    true,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
			},
		},
	}
}

func resourceSplitDNSNameserversRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	client := m.(*tailscale.Client)
	splitDNS, err := client.SplitDNS(ctx)
	if err != nil {
		return diagnosticsError(err, "Failed to fetch split DNS configs")
	}

	nameservers := splitDNS[d.Id()]

	if err = d.Set("nameservers", nameservers); err != nil {
		return diag.FromErr(err)
	}

	return nil
}

func resourceSplitDNSNameserversCreate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	client := m.(*tailscale.Client)
	nameserversSet := d.Get("nameservers").(*schema.Set)
	domain := d.Get("domain").(string)

	nameserversList := nameserversSet.List()

	req := make(tailscale.SplitDnsRequest)
	var nameservers []string
	for _, nameserver := range nameserversList {
		nameservers = append(nameservers, nameserver.(string))
	}
	req[domain] = nameservers

	// Return value is not useful to us here, ignore.
	if _, err := client.UpdateSplitDNS(ctx, req); err != nil {
		return diagnosticsError(err, "Failed to set dns split nameservers")
	}

	d.SetId(domain)
	return nil
}

func resourceSplitDNSNameserversUpdate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	if !d.HasChange("nameservers") {
		return resourceSplitDNSNameserversRead(ctx, d, m)
	}

	return resourceSplitDNSNameserversCreate(ctx, d, m)
}

func resourceSplitDNSNameserversDelete(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	client := m.(*tailscale.Client)
	domain := d.Get("domain").(string)

	req := make(tailscale.SplitDnsRequest)
	req[domain] = []string{}

	// Return value is not useful to us here, ignore.
	if _, err := client.UpdateSplitDNS(ctx, req); err != nil {
		return diagnosticsError(err, "Failed to set dns split nameservers")
	}

	return nil
}
