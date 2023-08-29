package tailscale

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/google/go-cmp/cmp"
	"github.com/hashicorp/go-cty/cty"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"

	"github.com/tailscale/hujson"

	"github.com/tailscale/tailscale-client-go/tailscale"
)

func resourceACL() *schema.Resource {
	return &schema.Resource{
		Description:   "The acl resource allows you to configure a Tailscale ACL. See https://tailscale.com/kb/1018/acls for more information. Note that this resource will completely overwrite existing ACL contents for a given tailnet.",
		ReadContext:   resourceACLRead,
		CreateContext: resourceACLCreate,
		UpdateContext: resourceACLUpdate,
		// Each tailnet always has an associated ACL file, so deleting a resource will
		// only remove it from Terraform state, leaving ACL contents intact.
		Delete: schema.Noop,
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},
		Schema: map[string]*schema.Schema{
			"acl": {
				Type:             schema.TypeString,
				Required:         true,
				ValidateDiagFunc: validateACL,
				Description:      "The JSON-based policy that defines which devices and users are allowed to connect in your network",
			},
		},
	}
}

func validateACL(i interface{}, p cty.Path) diag.Diagnostics {
	if _, err := unmarshalACL(i.(string)); err != nil {
		return diagnosticsErrorWithPath(err, "Invalid ACL", p)
	}
	return nil
}

func sameACL(a, b tailscale.ACL) bool {
	return cmp.Equal(a, b)
}

func resourceACLRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	client := m.(*tailscale.Client)
	acl, err := client.ACL(ctx)
	if err != nil {
		return diagnosticsError(err, "Failed to fetch ACL")
	}

	// If we have ACL content in Terraform state that is equivalent to the ACL
	// fetched via API, keep the local version. This is to have further changes
	// diffed against previous version specified by the user, avoiding spurious
	// diffs caused by potentially different spelling of ACL field names.
	current := d.Get("acl").(string)
	if current != "" {
		cur, err := unmarshalACL(current)
		if err != nil {
			return diagnosticsError(err, "Failed to unmarshal current ACL")
		}
		if sameACL(cur, *acl) {
			return nil
		}
	}

	aclStr, err := json.MarshalIndent(acl, "", "  ")
	if err != nil {
		return diagnosticsError(err, "Failed to marshal ACL for")
	}

	if err := d.Set("acl", string(aclStr)); err != nil {
		return diag.FromErr(err)
	}
	return nil
}

func resourceACLCreate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	client := m.(*tailscale.Client)
	aclStr := d.Get("acl").(string)

	acl, err := unmarshalACL(aclStr)
	if err != nil {
		return diagnosticsError(err, "Failed to unmarshal ACL")
	}

	// Setting the `ts-default` ETag will make this operation succeed only if
	// ACL contents has never been changed from its default value.
	if err := client.SetACL(ctx, acl, tailscale.WithETag("ts-default")); err != nil {
		if strings.HasSuffix(err.Error(), "(412)") {
			err = fmt.Errorf(
				"! You seem to be trying to overwrite a non-default ACL with a tailscale_acl resource.\n"+
					"Before doing this, please import your existing ACL into Terraform state using:\n"+
					" terraform import $(this_resource) acl\n"+
					"(got error %q)", err)
		}
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

	acl, err := unmarshalACL(aclStr)
	if err != nil {
		return diagnosticsError(err, "Failed to unmarshal ACL")
	}

	if err := client.SetACL(ctx, acl); err != nil {
		return diagnosticsError(err, "Failed to set ACL")
	}

	return nil
}

func unmarshalACL(s string) (tailscale.ACL, error) {
	b, err := hujson.Standardize([]byte(s))
	if err != nil {
		return tailscale.ACL{}, err
	}

	decoder := json.NewDecoder(bytes.NewBuffer(b))
	decoder.DisallowUnknownFields()

	var acl tailscale.ACL
	if err = decoder.Decode(&acl); err != nil {
		return acl, fmt.Errorf("%w. (This error may be caused by a new ACL feature that is not yet supported by "+
			"this terraform provider. If you're using a valid ACL field, please raise an issue at "+
			"https://github.com/tailscale/terraform-provider-tailscale/issues/new/choose)", err)
	}

	return acl, nil
}
