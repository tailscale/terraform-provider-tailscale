package tailscale

import (
	"context"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"

	tsclient "github.com/tailscale/tailscale-client-go/v2"
)

func resourceDNSSplitNameservers() *schema.Resource {
	return &schema.Resource{
		Description:   "The dns_split_nameservers resource allows you to configure split DNS nameservers for your Tailscale network. See https://tailscale.com/kb/1054/dns for more information.",
		ReadContext:   resourceSplitDNSNameserversRead,
		CreateContext: resourceSplitDNSNameserversCreateOrUpdate,
		UpdateContext: resourceSplitDNSNameserversCreateOrUpdate,
		DeleteContext: resourceSplitDNSNameserversDelete,
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},
		Schema: map[string]*schema.Schema{
			"domain": {
				Type:        schema.TypeString,
				Description: "Domain to configure split DNS for. Requests for this domain will be resolved using the provided nameservers. Changing this will force the resource to be recreated.",
				Required:    true,
				ForceNew:    true,
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
	client := m.(*Clients).V2
	splitDNS, err := client.DNS().SplitDNS(ctx)
	if err != nil {
		return diagnosticsError(err, "Failed to fetch split DNS configs")
	}

	domain := d.Id()

	if err = d.Set("domain", domain); err != nil {
		return diag.FromErr(err)
	}

	nameservers := splitDNS[d.Id()]

	if err = d.Set("nameservers", nameservers); err != nil {
		return diag.FromErr(err)
	}

	return nil
}

func resourceSplitDNSNameserversCreateOrUpdate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	client := m.(*Clients).V2
	nameserversSet := d.Get("nameservers").(*schema.Set)
	domain := d.Get("domain").(string)

	nameserversList := nameserversSet.List()

	req := make(tsclient.SplitDNSRequest)
	var nameservers []string
	for _, nameserver := range nameserversList {
		nameservers = append(nameservers, nameserver.(string))
	}
	req[domain] = nameservers

	// Return value is not useful to us here, ignore.
	if _, err := client.DNS().UpdateSplitDNS(ctx, req); err != nil {
		return diagnosticsError(err, "Failed to set dns split nameservers")
	}

	d.SetId(domain)
	return resourceSplitDNSNameserversRead(ctx, d, m)
}

func resourceSplitDNSNameserversDelete(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	client := m.(*Clients).V2
	domain := d.Get("domain").(string)

	req := make(tsclient.SplitDNSRequest)
	req[domain] = []string{}

	// Return value is not useful to us here, ignore.
	if _, err := client.DNS().UpdateSplitDNS(ctx, req); err != nil {
		return diagnosticsError(err, "Failed to delete dns split nameservers")
	}

	return nil
}
