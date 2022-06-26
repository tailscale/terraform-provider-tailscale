package tailscale

import (
	"context"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"

	"github.com/davidsbond/tailscale-client-go/tailscale"
)

func resourceDNSNameservers() *schema.Resource {
	return &schema.Resource{
		Description:   "The dns_nameservers resource allows you to configure DNS nameservers for your Tailscale network. See https://tailscale.com/kb/1054/dns for more information.",
		ReadContext:   resourceDNSNameserversRead,
		CreateContext: resourceDNSNameserversCreate,
		UpdateContext: resourceDNSNameserversUpdate,
		DeleteContext: resourceDNSNameserversDelete,
		Schema: map[string]*schema.Schema{
			"nameservers": {
				Type: schema.TypeList,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
				Description: "Devices on your network will use these nameservers to resolve DNS names. IPv4 or IPv6 addresses are accepted.",
				Required:    true,
				MinItems:    1,
			},
		},
	}
}

func resourceDNSNameserversRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	client := m.(*tailscale.Client)
	servers, err := client.DNSNameservers(ctx)
	if err != nil {
		return diagnosticsError(err, "Failed to fetch dns nameservers")
	}

	if err = d.Set("nameservers", servers); err != nil {
		return diag.FromErr(err)
	}

	return nil
}

func resourceDNSNameserversCreate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	client := m.(*tailscale.Client)
	nameservers := d.Get("nameservers").([]interface{})

	servers := make([]string, len(nameservers))
	for i, server := range nameservers {
		servers[i] = server.(string)
	}

	if err := client.SetDNSNameservers(ctx, servers); err != nil {
		return diagnosticsError(err, "Failed to set dns nameservers")
	}

	d.SetId(createUUID())
	return nil
}

func resourceDNSNameserversUpdate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	if !d.HasChange("nameservers") {
		return resourceDNSNameserversRead(ctx, d, m)
	}

	client := m.(*tailscale.Client)
	nameservers := d.Get("nameservers").([]interface{})

	servers := make([]string, len(nameservers))
	for i, server := range nameservers {
		servers[i] = server.(string)
	}

	if err := client.SetDNSNameservers(ctx, servers); err != nil {
		return diagnosticsError(err, "Failed to set dns nameservers")
	}

	return nil
}

func resourceDNSNameserversDelete(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	client := m.(*tailscale.Client)

	if err := client.SetDNSNameservers(ctx, []string{}); err != nil {
		return diagnosticsError(err, "Failed to set dns nameservers")
	}

	return nil
}
