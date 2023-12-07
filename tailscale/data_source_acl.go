package tailscale

import (
	"context"
	"encoding/json"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"

	"github.com/tailscale/tailscale-client-go/tailscale"
)

func dataSourceACL() *schema.Resource {
	return &schema.Resource{
		Description: "The acl data source gets the Tailscale ACL for a tailnet",
		ReadContext: dataSourceACLRead,
		Schema: map[string]*schema.Schema{
			"json": {
				Computed:    true,
				Type:        schema.TypeString,
				Description: "The contents of Tailscale ACL as JSON",
			},
		},
	}
}

func dataSourceACLRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	client := m.(*tailscale.Client)

	acl, err := client.ACL(ctx)
	if err != nil {
		return diagnosticsError(err, "Failed to fetch ACL")
	}

	aclJson, err := json.Marshal(acl)
	if err != nil {
		return diag.FromErr(err)
	}
	if err := d.Set("json", string(aclJson)); err != nil {
		return diag.Errorf("setting json: %s", err)
	}

	d.SetId(createUUID())
	return nil
}
