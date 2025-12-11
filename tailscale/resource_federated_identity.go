// Copyright (c) David Bond, Tailscale Inc, & Contributors
// SPDX-License-Identifier: MIT

package tailscale

import (
	"context"
	"time"

	"github.com/hashicorp/go-cty/cty"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"

	"tailscale.com/client/tailscale/v2"
)

func resourceFederatedIdentity() *schema.Resource {
	return &schema.Resource{
		Description:   "The federated_identity resource allows you to create federated identities to programmatically interact with the Tailscale API using workload identity federation.",
		ReadContext:   resourceFederatedIdentityRead,
		CreateContext: resourceFederatedIdentityCreate,
		DeleteContext: resourceFederatedIdentityDelete,
		UpdateContext: resourceFederatedIdentityUpdate,
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},
		Schema: map[string]*schema.Schema{
			"description": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "A description of the federated identity consisting of alphanumeric characters. Defaults to `\"\"`.",
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
				Description: "Scopes to grant to the federated identity. See https://tailscale.com/kb/1623/ for a list of available scopes.",
			},
			"tags": {
				Type: schema.TypeSet,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
				Optional:    true,
				Description: "A list of tags that access tokens generated for the federated identity will be able to assign to devices. Mandatory if the scopes include \"devices:core\" or \"auth_keys\".",
			},
			"id": {
				Type:        schema.TypeString,
				Description: "The client ID, also known as the key id. Used with an OIDC identity token to generate access tokens.",
				Computed:    true,
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
				Description: "ID of the user who created this federated identity, empty for federated identities created by other trust credentials.",
				Computed:    true,
			},
			"audience": {
				Type:        schema.TypeString,
				Description: "The value used when matching against the `aud` claim from an OIDC identity token. Specifying the audience is optional as Tailscale will generate a secure audience at creation time by default.   It is recommended to let Tailscale generate the audience unless the identity provider you are integrating with requires a specific audience format.",
				Optional:    true,
				Computed:    true,
			},
			"subject": {
				Type:        schema.TypeString,
				Description: "The pattern used when matching against the `sub` claim from an OIDC identity token. Patterns can include `*` characters to match against any character.",
				Required:    true,
			},
			"issuer": {
				Type:         schema.TypeString,
				Description:  "The issuer of the OIDC identity token used in the token exchange. Must be a valid and publicly reachable https:// URL.",
				ValidateFunc: validation.IsURLWithHTTPS,
				Required:     true,
			},
			"custom_claim_rules": {
				Type:        schema.TypeMap,
				Description: "A map of claim names to pattern strings used to match against arbitrary claims in the OIDC identity token. Patterns can include `*` characters to match against any character.",
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
				Optional: true,
			},
		},
	}
}

func resourceFederatedIdentityRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
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
		return diagnosticsError(err, "Failed to set created_at")
	}

	if err = d.Set("user_id", key.UserID); err != nil {
		return diagnosticsError(err, "Failed to set 'user_id'")
	}

	if err = d.Set("audience", key.Audience); err != nil {
		return diagnosticsError(err, "Failed to set 'audience'")
	}

	if err = d.Set("subject", key.Subject); err != nil {
		return diagnosticsError(err, "Failed to set 'subject'")
	}

	if err = d.Set("issuer", key.Issuer); err != nil {
		return diagnosticsError(err, "Failed to set 'issuer'")
	}

	if err = d.Set("custom_claim_rules", key.CustomClaimRules); err != nil {
		return diagnosticsError(err, "Failed to set 'custom_claim_rules'")
	}

	return nil
}

func resourceFederatedIdentityCreate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
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

	audience, ok := d.GetOk("audience")
	if !ok {
		audience = ""
	}

	customClaimRules := map[string]string{}
	maybeRules, ok := d.GetOk("custom_claim_rules")
	if ok {
		for k, v := range maybeRules.(map[string]interface{}) {
			customClaimRules[k] = v.(string)
		}
	}

	key, err := client.Keys().CreateFederatedIdentity(ctx, tailscale.CreateFederatedIdentityRequest{
		Description:      description.(string),
		Scopes:           scopes,
		Tags:             tags,
		Audience:         audience.(string),
		Subject:          d.Get("subject").(string),
		CustomClaimRules: customClaimRules,
		Issuer:           d.Get("issuer").(string),
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

	if err = d.Set("audience", key.Audience); err != nil {
		return diagnosticsError(err, "Failed to set audience")
	}

	return resourceFederatedIdentityRead(ctx, d, m)
}

func resourceFederatedIdentityUpdate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
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

	customClaimRules := map[string]string{}
	maybeRules, ok := d.GetOk("custom_claim_rules")
	if ok {
		for k, v := range maybeRules.(map[string]interface{}) {
			customClaimRules[k] = v.(string)
		}
	}

	key, err := client.Keys().SetFederatedIdentity(ctx, d.Id(),
		tailscale.SetFederatedIdentityRequest{
			Description:      description.(string),
			Scopes:           scopes,
			Tags:             tags,
			Audience:         d.Get("audience").(string),
			Subject:          d.Get("subject").(string),
			CustomClaimRules: customClaimRules,
			Issuer:           d.Get("issuer").(string),
		})
	if err != nil {
		return diagnosticsError(err, "Failed to update federated identity")
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

	if err = d.Set("audience", key.Audience); err != nil {
		return diagnosticsError(err, "Failed to set audience")
	}

	return resourceFederatedIdentityRead(ctx, d, m)
}

func resourceFederatedIdentityDelete(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	client := m.(*tailscale.Client)

	err := client.Keys().Delete(ctx, d.Id())
	if err != nil && !tailscale.IsNotFound(err) {
		return diagnosticsError(err, "Failed to delete federated identity")
	}

	return nil
}
