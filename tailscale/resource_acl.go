package tailscale

import (
	"context"

	"github.com/davidsbond/tailscale-client-go/tailscale"
	"github.com/google/go-cmp/cmp"
	"github.com/hashicorp/go-cty/cty"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/tailscale/hujson"
)

func resourceACL() *schema.Resource {
	return &schema.Resource{
		Description:   "The acl resource allows you to configure a Tailscale ACL. See https://tailscale.com/kb/1018/acls for more information.",
		ReadContext:   resourceACLRead,
		CreateContext: resourceACLCreate,
		UpdateContext: resourceACLUpdate,
		DeleteContext: resourceACLDelete,
		Schema: map[string]*schema.Schema{
			"acl": {
				Type:             schema.TypeString,
				Required:         true,
				ValidateDiagFunc: validateACL,
				DiffSuppressFunc: suppressACLDiff,
				Description:      "The JSON-based policy that defines which devices and users are allowed to connect in your network",
			},
		},
	}
}

func validateACL(i interface{}, p cty.Path) diag.Diagnostics {
	var acl tailscale.ACL
	if err := hujson.Unmarshal([]byte(i.(string)), &acl); err != nil {
		return diagnosticsErrorWithPath(err, "Invalid ACL", p)
	}
	return nil
}

func suppressACLDiff(_, old, new string, _ *schema.ResourceData) bool {
	var oldACL tailscale.ACL
	var newACL tailscale.ACL

	if err := hujson.Unmarshal([]byte(old), &oldACL); err != nil {
		return false
	}

	if err := hujson.Unmarshal([]byte(new), &newACL); err != nil {
		return false
	}

	return cmp.Equal(oldACL, newACL)
}

func resourceACLRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	client := m.(*tailscale.Client)
	acl, err := client.ACL(ctx)
	if err != nil {
		return diagnosticsError(err, "Failed to fetch ACL")
	}

	aclStr, err := hujson.MarshalIndent(acl, "", "  ")
	if err != nil {
		return diagnosticsError(err, "Failed to marshal ACL for")
	}

	values := map[string]interface{}{
		"acl": string(aclStr),
	}

	for k, v := range values {
		if err = d.Set(k, v); err != nil {
			return diag.FromErr(err)
		}
	}

	return nil
}

func resourceACLCreate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	client := m.(*tailscale.Client)
	aclStr := d.Get("acl").(string)

	var acl tailscale.ACL
	if err := hujson.Unmarshal([]byte(aclStr), &acl); err != nil {
		return diagnosticsError(err, "Failed to unmarshal ACL")
	}

	if err := client.SetACL(ctx, acl); err != nil {
		return diagnosticsError(err, "Failed to set ACL")
	}

	d.SetId(createUUID())
	return nil
}

func resourceACLUpdate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	client := m.(*tailscale.Client)
	aclStr := d.Get("acl").(string)

	if !d.HasChange("acl") {
		return nil
	}

	var acl tailscale.ACL
	if err := hujson.Unmarshal([]byte(aclStr), &acl); err != nil {
		return diagnosticsError(err, "Failed to unmarshal ACL")
	}

	if err := client.SetACL(ctx, acl); err != nil {
		return diagnosticsError(err, "Failed to set ACL")
	}

	return nil
}

func resourceACLDelete(ctx context.Context, _ *schema.ResourceData, m interface{}) diag.Diagnostics {
	client := m.(*tailscale.Client)

	acl := tailscale.ACL{
		ACLs: []tailscale.ACLEntry{
			{
				Action: "accept",
				Users:  []string{"*"},
				Ports:  []string{"*:*"},
			},
		},
	}

	if err := client.SetACL(ctx, acl); err != nil {
		return diagnosticsError(err, "Failed to set ACL")
	}

	return nil
}
