package tailscale

import (
	"context"

	"github.com/davidsbond/terraform-provider-tailscale/internal/tailscale"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func resourceDNSPreferences() *schema.Resource {
	return &schema.Resource{
		ReadContext:   resourceDNSPreferencesRead,
		CreateContext: resourceDNSPreferencesCreate,
		UpdateContext: resourceDNSPreferencesUpdate,
		DeleteContext: resourceDNSPreferencesDelete,
		Schema: map[string]*schema.Schema{
			"magic_dns": {
				Type:        schema.TypeBool,
				Description: "Whether or not to enable magic DNS",
				Required:    true,
			},
		},
	}
}

func resourceDNSPreferencesRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	client := m.(*tailscale.Client)

	preferences, err := client.DNSPreferences(ctx)
	if err != nil {
		return diagnosticsError(err, "Failed to fetch dns preferences")
	}

	if err = d.Set("magic_dns", preferences.MagicDNS); err != nil {
		return diag.FromErr(err)
	}

	return nil
}

func resourceDNSPreferencesCreate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	client := m.(*tailscale.Client)
	magicDNS := d.Get("magic_dns").(bool)
	preferences := tailscale.DNSPreferences{
		MagicDNS: magicDNS,
	}

	if err := client.SetDNSPreferences(ctx, preferences); err != nil {
		return diagnosticsError(err, "Failed to set dns preferences")
	}

	d.SetId(createUUID())
	return nil
}

func resourceDNSPreferencesUpdate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	if !d.HasChange("magic_dns") {
		return resourceDNSPreferencesRead(ctx, d, m)
	}

	client := m.(*tailscale.Client)
	magicDNS := d.Get("magic_dns").(bool)

	preferences := tailscale.DNSPreferences{
		MagicDNS: magicDNS,
	}

	if err := client.SetDNSPreferences(ctx, preferences); err != nil {
		return diagnosticsError(err, "Failed to set dns preferences")
	}

	return nil
}

func resourceDNSPreferencesDelete(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	client := m.(*tailscale.Client)

	if err := client.SetDNSPreferences(ctx, tailscale.DNSPreferences{}); err != nil {
		return diagnosticsError(err, "Failed to set dns preferences")
	}

	return nil
}
