package tailscale

import (
	"context"
	"time"

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
		Schema: map[string]*schema.Schema{
			"reusable": {
				Type:        schema.TypeBool,
				Optional:    true,
				Description: "Indicates if the key is reusable or single-use.",
				ForceNew:    true,
			},
			"ephemeral": {
				Type:        schema.TypeBool,
				Optional:    true,
				Description: "Indicates if the key is ephemeral.",
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
				Description: "Determines whether or not the machines authenticated by the key will be authorized for the tailnet by default.",
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
				Description: "The expiry of the key in seconds",
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
				Description: "A description of the key consisting of alphanumeric characters.",
				ForceNew:    true,
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
	description := d.Get("description").(string)
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

	if len(description) > 0 {
		opts = append(opts, tailscale.WithKeyDescription(description))
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

func resourceTailnetKeyRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	client := m.(*tailscale.Client)
	key, err := client.GetKey(ctx, d.Id())

	reusable := d.Get("reusable").(bool)

	switch {
	case tailscale.IsNotFound(err) && !reusable:
		// If we get a 404 on a one-off key, don't return an error here.
		return nil
	case err != nil:
		return diagnosticsError(err, "Failed to fetch key")
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

	return nil
}
