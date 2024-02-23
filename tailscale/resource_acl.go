package tailscale

import (
	"context"
	"fmt"
	"strings"

	"github.com/hashicorp/go-cty/cty"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"

	"github.com/tailscale/hujson"

	"github.com/tailscale/tailscale-client-go/tailscale"
)

const resourceACLDescription = `The acl resource allows you to configure a Tailscale ACL. See https://tailscale.com/kb/1018/acls for more information. Note that this resource will completely overwrite existing ACL contents for a given tailnet.

If tests are defined in the ACL (the top-level "tests" section), ACL validation will occur before creation and update operations are applied.`

// from https://github.com/hashicorp/terraform-plugin-sdk/blob/34d8a9ebca6bed68fddb983123d6fda72481752c/internal/configs/hcl2shim/values.go#L19
const UnknownVariableValue = "74D93920-ED26-11E3-AC10-0800200C9A66"

func resourceACL() *schema.Resource {
	return &schema.Resource{
		Description:   resourceACLDescription,
		ReadContext:   resourceACLRead,
		CreateContext: resourceACLCreate,
		UpdateContext: resourceACLUpdate,
		// Each tailnet always has an associated ACL file, so deleting a resource will
		// only remove it from Terraform state, leaving ACL contents intact.
		Delete: schema.Noop,
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},
		CustomizeDiff: func(ctx context.Context, rd *schema.ResourceDiff, m interface{}) error {
			client := m.(*tailscale.Client)

			//if the acl is only known after apply, then acl will be an empty string and validation will fail
			if rd.Get("acl").(string) == "" {
				return nil
			}
			return client.ValidateACL(ctx, rd.Get("acl").(string))
		},
		Schema: map[string]*schema.Schema{
			"acl": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "The policy that defines which devices and users are allowed to connect in your network. Can be either a JSON or a HuJSON string.",

				// Field-level validation just checks that it's valid JSON or HuJSON.
				// Actual contents of the policy is validated by calling the API when
				// the whole resource is validated in CustomizeDiff.
				ValidateDiagFunc: func(i interface{}, p cty.Path) diag.Diagnostics {
					_, err := hujson.Parse([]byte(i.(string)))
					if err != nil {
						return diagnosticsErrorWithPath(err, "ACL is not a valid HuJSON string", p)
					}
					return nil
				},

				// Do not show a diff if canonical HuJSON representation of the policy did not
				// change. Note that a policy that is valid JSON will not be formatted as HuJSON
				// (see hujson.Format docs), so a diff is expected when switching from JSON to
				// HuJSON (or back), even if there are no semantic changes.
				DiffSuppressFunc: func(k, oldValue, newValue string, d *schema.ResourceData) bool {
					old, oldErr := hujson.Format([]byte(oldValue))
					new, newErr := hujson.Format([]byte(newValue))
					if oldErr != nil || newErr != nil {
						return false
					}
					return string(old) == string(new)
				},
				DiffSuppressOnRefresh: true,

				// Use the canonical HuJSON representation of the policy in Terraform state.
				StateFunc: func(i interface{}) string {
					//if the acl is only known after apply, then it will be the magic UUID `UnknownVariableValue` and not valid json, and formatting will fail
					if i.(string) == UnknownVariableValue {
						return i.(string)
					}
					value, err := hujson.Format([]byte(i.(string)))
					if err != nil {
						panic(fmt.Errorf("could not parse ACL as HuJSON: %s", err))
					}
					return string(value)
				},
			},
			"overwrite_existing_content": {
				Type:        schema.TypeBool,
				Optional:    true,
				Description: "If true, will skip requirement to import acl before allowing changes. Be careful, can cause ACL to be overwritten",
			},
		},
	}
}

func resourceACLRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	client := m.(*tailscale.Client)
	acl, err := client.RawACL(ctx)
	if err != nil {
		return diagnosticsError(err, "Failed to fetch ACL")
	}

	if err := d.Set("acl", acl); err != nil {
		return diag.FromErr(err)
	}
	return nil
}

func resourceACLCreate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	client := m.(*tailscale.Client)
	acl := d.Get("acl").(string)

	// Setting the `ts-default` ETag will make this operation succeed only if
	// ACL contents has never been changed from its default value.
	var opts []tailscale.SetACLOption
	if !d.Get("overwrite_existing_content").(bool) {
		opts = append(opts, tailscale.WithETag("ts-default"))
	}

	if err := client.SetACL(ctx, acl, opts...); err != nil {
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

	if !d.HasChange("acl") {
		return nil
	}

	if err := client.SetACL(ctx, d.Get("acl").(string)); err != nil {
		return diagnosticsError(err, "Failed to set ACL")
	}

	return nil
}
