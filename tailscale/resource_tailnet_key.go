package tailscale

import (
	"context"
	"time"

	"github.com/hashicorp/go-cty/cty"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"

	"github.com/tailscale/tailscale-client-go/tailscale"
)

func resourceTailnetKey() *schema.Resource {
	return &schema.Resource{
		Description:   "The tailnet_key resource allows you to create pre-authentication keys that can register new nodes without needing to sign in via a web browser. See https://tailscale.com/kb/1085/auth-keys for more information",
		ReadContext:   resourceTailnetKeyRead,
		CreateContext: resourceTailnetKeyCreate,
		DeleteContext: resourceTailnetKeyDelete,
		UpdateContext: schema.NoopContext,
		CustomizeDiff: resourceTailnetKeyDiff,
		Schema: map[string]*schema.Schema{
			"reusable": {
				Type:        schema.TypeBool,
				Optional:    true,
				Description: "Indicates if the key is reusable or single-use. Defaults to `false`.",
				ForceNew:    true,
			},
			"ephemeral": {
				Type:        schema.TypeBool,
				Optional:    true,
				Description: "Indicates if the key is ephemeral. Defaults to `false`.",
				ForceNew:    true,
			},
			"tags": {
				Type: schema.TypeSet,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
				Optional:    true,
				Description: "List of tags to apply to the machines authenticated by the key.",
				ForceNew:    true,
			},
			"preauthorized": {
				Type:        schema.TypeBool,
				Optional:    true,
				Description: "Determines whether or not the machines authenticated by the key will be authorized for the tailnet by default. Defaults to `false`.",
				ForceNew:    true,
			},
			"key": {
				Type:        schema.TypeString,
				Description: "The authentication key",
				Computed:    true,
				Sensitive:   true,
			},
			"expiry": {
				Type:        schema.TypeInt,
				Optional:    true,
				Description: "The expiry of the key in seconds. Defaults to `7776000` (90 days).",
				ForceNew:    true,
			},
			"created_at": {
				Type:        schema.TypeString,
				Description: "The creation timestamp of the key in RFC3339 format",
				Computed:    true,
			},
			"expires_at": {
				Type:        schema.TypeString,
				Description: "The expiry timestamp of the key in RFC3339 format",
				Computed:    true,
			},
			"description": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "A description of the key consisting of alphanumeric characters. Defaults to `\"\"`.",
				ForceNew:    true,
			},
			"invalid": {
				Type:        schema.TypeBool,
				Description: "Indicates whether the key is invalid (e.g. expired, revoked or has been deleted).",
				Computed:    true,
			},
			"recreate_if_invalid": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "Determines whether the key should be created again if it becomes invalid. By default, reusable keys will be recreated, but single-use keys will not. Possible values: 'always', 'never'.",
				ValidateDiagFunc: func(i interface{}, p cty.Path) diag.Diagnostics {
					switch i.(string) {
					case "", "always", "never":
						return nil
					default:
						return diagnosticsError(nil, "unexpected value of recreate_if_invalid: %s", i)
					}
				},
			},
		},
	}
}

func resourceTailnetKeyCreate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	client := m.(*tailscale.Client)
	reusable := d.Get("reusable").(bool)
	ephemeral := d.Get("ephemeral").(bool)
	preauthorized := d.Get("preauthorized").(bool)
	expiry, hasExpiry := d.GetOk("expiry")
	description, hasDescription := d.GetOk("description")
	var tags []string
	for _, tag := range d.Get("tags").(*schema.Set).List() {
		tags = append(tags, tag.(string))
	}

	var capabilities tailscale.KeyCapabilities
	capabilities.Devices.Create.Reusable = reusable
	capabilities.Devices.Create.Ephemeral = ephemeral
	capabilities.Devices.Create.Tags = tags
	capabilities.Devices.Create.Preauthorized = preauthorized

	var opts []tailscale.CreateKeyOption
	if hasExpiry {
		opts = append(opts, tailscale.WithKeyExpiry(time.Duration(expiry.(int))*time.Second))
	}

	if hasDescription {
		opts = append(opts, tailscale.WithKeyDescription(description.(string)))
	}

	key, err := client.CreateKey(ctx, capabilities, opts...)
	if err != nil {
		return diagnosticsError(err, "Failed to create key")
	}

	d.SetId(key.ID)

	if err = d.Set("key", key.Key); err != nil {
		return diagnosticsError(err, "Failed to set key")
	}

	if err = d.Set("created_at", key.Created.Format(time.RFC3339)); err != nil {
		return diagnosticsError(err, "Failed to set created_at")
	}

	if err = d.Set("expires_at", key.Expires.Format(time.RFC3339)); err != nil {
		return diagnosticsError(err, "Failed to set expires_at")
	}

	if err = d.Set("invalid", key.Invalid); err != nil {
		return diagnosticsError(err, "Failed to set 'invalid'")
	}

	return nil
}

func resourceTailnetKeyDelete(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	client := m.(*tailscale.Client)

	err := client.DeleteKey(ctx, d.Id())
	switch {
	case tailscale.IsNotFound(err):
		// Single-use keys may no longer be here, so we can ignore deletions that fail due to not-found errors.
		return nil
	case err != nil:
		return diagnosticsError(err, "Failed to delete key")
	default:
		return nil
	}
}

// shouldRecreateIfInvalid determines if a resource should be recreated when
// it's invalid, based on the values of `reusable` and `recreate_if_invalid` fields.
// By default, we automatically recreate reusable keys, but ignore invalid single-use
// keys, assuming they have successfully been used, and recreating them might trigger
// unnecessary updates of other Terraform resources that depend on the key.
func shouldRecreateIfInvalid(reusable bool, recreateIfInvalid string) bool {
	if recreateIfInvalid == "always" {
		return true
	}
	if recreateIfInvalid == "never" {
		return false
	}
	return reusable
}

// resourceTailnetKeyDiff makes sure a resource is recreated when a `recreate_if_invalid`
// field changes in a way that requires it.
func resourceTailnetKeyDiff(ctx context.Context, d *schema.ResourceDiff, m interface{}) error {
	old, new := d.GetChange("recreate_if_invalid")
	if old == new {
		return nil
	}

	recreateIfInvalid := shouldRecreateIfInvalid(d.Get("reusable").(bool), d.Get("recreate_if_invalid").(string))
	if !recreateIfInvalid {
		return nil
	}

	client := m.(*tailscale.Client)
	key, err := client.GetKey(ctx, d.Id())
	if tailscale.IsNotFound(err) || (err == nil && key.Invalid) {
		d.ForceNew("recreate_if_invalid")
	}
	return nil
}

func resourceTailnetKeyRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	recreateIfInvalid := shouldRecreateIfInvalid(d.Get("reusable").(bool), d.Get("recreate_if_invalid").(string))

	client := m.(*tailscale.Client)
	key, err := client.GetKey(ctx, d.Id())

	switch {
	case tailscale.IsNotFound(err):
		if recreateIfInvalid {
			d.SetId("")
		}
		return nil
	case err != nil:
		return diagnosticsError(err, "Failed to fetch key")
	}

	// The Tailscale API continues to return keys for some time after they've expired.
	// Use `invalid` key property to determine if key should be recreated.
	if key.Invalid && recreateIfInvalid {
		d.SetId("")
		return nil
	}

	d.SetId(key.ID)
	if err = d.Set("reusable", key.Capabilities.Devices.Create.Reusable); err != nil {
		return diagnosticsError(err, "Failed to set reusable")
	}

	if err = d.Set("ephemeral", key.Capabilities.Devices.Create.Ephemeral); err != nil {
		return diagnosticsError(err, "failed to set ephemeral")
	}

	if err = d.Set("created_at", key.Created.Format(time.RFC3339)); err != nil {
		return diagnosticsError(err, "Failed to set created_at")
	}

	if err = d.Set("expires_at", key.Expires.Format(time.RFC3339)); err != nil {
		return diagnosticsError(err, "Failed to set expires_at")
	}

	if err = d.Set("description", key.Description); err != nil {
		return diagnosticsError(err, "Failed to set description")
	}

	if err = d.Set("invalid", key.Invalid); err != nil {
		return diagnosticsError(err, "Failed to set 'invalid'")
	}

	return nil
}
