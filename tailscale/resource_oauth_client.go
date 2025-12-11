// Copyright (c) David Bond, Tailscale Inc, & Contributors
// SPDX-License-Identifier: MIT

package tailscale

import (
	"context"
	"time"

	"github.com/hashicorp/go-cty/cty"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"

	"tailscale.com/client/tailscale/v2"
)

func resourceOAuthClient() *schema.Resource {
	return &schema.Resource{
		Description:   "The oauth_client resource allows you to create OAuth clients to programmatically interact with the Tailscale API.",
		ReadContext:   resourceOAuthClientRead,
		CreateContext: resourceOAuthClientCreate,
		DeleteContext: resourceOAuthClientDelete,
		UpdateContext: resourceOAuthClientUpdate,
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},
		Schema: map[string]*schema.Schema{
			"description": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "A description of the OAuth client consisting of alphanumeric characters. Defaults to `\"\"`.",
				ValidateDiagFunc: func(i interface{}, p cty.Path) diag.Diagnostics {
					if len(i.(string)) > 50 {
						return diagnosticsError(nil, "description must be 50 characters or less")
					}
					return nil
				},
			},
			"scopes": {
				Type: schema.TypeSet,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
				Required:    true,
				Description: "Scopes to grant to the client. See https://tailscale.com/kb/1623/ for a list of available scopes.",
			},
			"tags": {
				Type: schema.TypeSet,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
				Optional:    true,
				Description: "A list of tags that access tokens generated for the OAuth client will be able to assign to devices. Mandatory if the scopes include \"devices:core\" or \"auth_keys\".",
			},
			"id": {
				Type:        schema.TypeString,
				Description: "The client ID, also known as the key id. Used with the client secret to generate access tokens.",
				Computed:    true,
			},
			"key": {
				Type:        schema.TypeString,
				Description: "The client secret, also known as the key. Used with the client ID to generate access tokens.",
				Computed:    true,
				Sensitive:   true,
			},
			"created_at": {
				Type:        schema.TypeString,
				Description: "The creation timestamp of the key in RFC3339 format",
				Computed:    true,
			},
			"updated_at": {
				Type:        schema.TypeString,
				Description: "The updated timestamp of the key in RFC3339 format",
				Computed:    true,
			},
			"user_id": {
				Type:        schema.TypeString,
				Description: "ID of the user who created this key, empty for OAuth clients created by other trust credentials.",
				Computed:    true,
			},
		},
	}
}

func resourceOAuthClientRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	client := m.(*tailscale.Client)
	key, err := client.Keys().Get(ctx, d.Id())
	if err != nil {
		return diagnosticsError(err, "Failed to fetch key")
	}

	d.SetId(key.ID)
	if err = d.Set("description", key.Description); err != nil {
		return diagnosticsError(err, "Failed to set description")
	}

	if err = d.Set("scopes", key.Scopes); err != nil {
		return diagnosticsError(err, "Failed to set 'scopes'")
	}

	if err = d.Set("tags", key.Tags); err != nil {
		return diagnosticsError(err, "Failed to set 'tags'")
	}

	if err = d.Set("created_at", key.Created.Format(time.RFC3339)); err != nil {
		return diagnosticsError(err, "Failed to set created_at")
	}

	if err = d.Set("updated_at", key.Updated.Format(time.RFC3339)); err != nil {
		return diagnosticsError(err, "Failed to set updated_at")
	}

	if err = d.Set("user_id", key.UserID); err != nil {
		return diagnosticsError(err, "Failed to set 'user_id'")
	}

	return nil
}

func resourceOAuthClientCreate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	client := m.(*tailscale.Client)

	description, ok := d.GetOk("description")
	if !ok {
		description = ""
	}
	var scopes []string
	for _, scope := range d.Get("scopes").(*schema.Set).List() {
		scopes = append(scopes, scope.(string))
	}
	var tags []string
	for _, tag := range d.Get("tags").(*schema.Set).List() {
		tags = append(tags, tag.(string))
	}

	key, err := client.Keys().CreateOAuthClient(ctx, tailscale.CreateOAuthClientRequest{
		Description: description.(string),
		Scopes:      scopes,
		Tags:        tags,
	})
	if err != nil {
		return diagnosticsError(err, "Failed to create oauth client")
	}

	d.SetId(key.ID)
	if err = d.Set("key", key.Key); err != nil {
		return diagnosticsError(err, "Failed to set key")
	}
	if err = d.Set("created_at", key.Created.Format(time.RFC3339)); err != nil {
		return diagnosticsError(err, "Failed to set created_at")
	}
	if err = d.Set("updated_at", key.Updated.Format(time.RFC3339)); err != nil {
		return diagnosticsError(err, "Failed to set updated_at")
	}
	if err = d.Set("user_id", key.UserID); err != nil {
		return diagnosticsError(err, "Failed to set user_id")
	}

	return resourceOAuthClientRead(ctx, d, m)
}

func resourceOAuthClientUpdate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	client := m.(*tailscale.Client)

	description, ok := d.GetOk("description")
	if !ok {
		description = ""
	}
	var scopes []string
	for _, scope := range d.Get("scopes").(*schema.Set).List() {
		scopes = append(scopes, scope.(string))
	}
	var tags []string
	for _, tag := range d.Get("tags").(*schema.Set).List() {
		tags = append(tags, tag.(string))
	}

	key, err := client.Keys().SetOAuthClient(ctx, d.Id(),
		tailscale.SetOAuthClientRequest{
			Description: description.(string),
			Scopes:      scopes,
			Tags:        tags,
		})
	if err != nil {
		return diagnosticsError(err, "Failed to create oauth client")
	}

	d.SetId(key.ID)
	if err = d.Set("created_at", key.Created.Format(time.RFC3339)); err != nil {
		return diagnosticsError(err, "Failed to set created_at")
	}
	if err = d.Set("updated_at", key.Updated.Format(time.RFC3339)); err != nil {
		return diagnosticsError(err, "Failed to set updated_at")
	}
	if err = d.Set("user_id", key.UserID); err != nil {
		return diagnosticsError(err, "Failed to set user_id")
	}

	return resourceOAuthClientRead(ctx, d, m)
}

func resourceOAuthClientDelete(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	client := m.(*tailscale.Client)

	err := client.Keys().Delete(ctx, d.Id())
	if err != nil && !tailscale.IsNotFound(err) {
		return diagnosticsError(err, "Failed to delete oauth client")
	}

	return nil
}
