package tailscale

import (
	"context"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"

	"github.com/tailscale/hujson"
)

func dataSourceACL() *schema.Resource {
	return &schema.Resource{
		Description: "The acl data source gets the Tailscale ACL for a tailnet",
		ReadContext: dataSourceACLRead,
		Schema: map[string]*schema.Schema{
			"json": {
				Computed:    true,
				Type:        schema.TypeString,
				Description: "The contents of Tailscale ACL as a JSON string",
			},
			"hujson": {
				Computed:    true,
				Type:        schema.TypeString,
				Description: "The contents of Tailscale ACL as a HuJSON string",
			},
		},
	}
}

func dataSourceACLRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	client := m.(*Clients).V1

	acl, err := client.RawACL(ctx)
	if err != nil {
		return diagnosticsError(err, "Failed to fetch ACL")
	}
	huj, err := hujson.Parse([]byte(acl))
	if err != nil {
		return diagnosticsError(err, "Failed to parse ACL as HuJSON")
	}
	if err := d.Set("hujson", huj.String()); err != nil {
		return diagnosticsError(err, "Failed to set 'hujson'")
	}

	huj.Minimize()
	if err := d.Set("json", huj.String()); err != nil {
		return diagnosticsError(err, "Failed to set 'json'")
	}

	d.SetId(createUUID())
	return nil
}
