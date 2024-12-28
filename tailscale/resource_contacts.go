// Copyright (c) David Bond, Tailscale Inc, & Contributors
// SPDX-License-Identifier: MIT

package tailscale

import (
	"context"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"

	tsclient "github.com/tailscale/tailscale-client-go/v2"
)

const resourceContactsDescription = `The contacts resource allows you to configure contact details for your Tailscale network. See https://tailscale.com/kb/1224/contact-preferences for more information.

Destroying this resource does not unset or modify values in the tailscale control plane, and simply removes the resource from Terraform state.
`

func resourceContacts() *schema.Resource {
	return &schema.Resource{
		Description:   resourceContactsDescription,
		ReadContext:   resourceContactsRead,
		CreateContext: resourceContactsCreate,
		UpdateContext: resourceContactsUpdate,
		DeleteContext: resourceContactsDelete,
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},
		Schema: map[string]*schema.Schema{
			"account": {
				Type:        schema.TypeSet,
				Description: "Configuration for communications about important changes to your tailnet",
				Required:    true,
				MaxItems:    1,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"email": {
							Type:        schema.TypeString,
							Description: "Email address to send communications to",
							Required:    true,
						},
					},
				},
			},
			"support": {
				Type:        schema.TypeSet,
				Description: "Configuration for communications about misconfigurations in your tailnet",
				Required:    true,
				MaxItems:    1,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"email": {
							Type:        schema.TypeString,
							Description: "Email address to send communications to",
							Required:    true,
						},
					},
				},
			},
			"security": {
				Type:        schema.TypeSet,
				Description: "Configuration for communications about security issues affecting your tailnet",
				Required:    true,
				MaxItems:    1,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"email": {
							Type:        schema.TypeString,
							Description: "Email address to send communications to",
							Required:    true,
						},
					},
				},
			},
		},
		EnableLegacyTypeSystemApplyErrors: true,
		EnableLegacyTypeSystemPlanErrors:  true,
	}
}

func resourceContactsCreate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	client := m.(*tsclient.Client)

	if diagErr := updateContact(ctx, client, d, tsclient.ContactAccount); diagErr != nil {
		return diagErr
	}

	if diagErr := updateContact(ctx, client, d, tsclient.ContactSupport); diagErr != nil {
		return diagErr
	}

	if diagErr := updateContact(ctx, client, d, tsclient.ContactSecurity); diagErr != nil {
		return diagErr
	}

	d.SetId(createUUID())
	return resourceContactsRead(ctx, d, m)
}

func resourceContactsRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	client := m.(*tsclient.Client)

	contacts, err := client.Contacts().Get(ctx)
	if err != nil {
		return diagnosticsError(err, "Failed to fetch contacts")
	}

	if err = d.Set("account", buildContactMap(contacts.Account)); err != nil {
		return diagnosticsError(err, "Failed to set account field")
	}

	if err = d.Set("support", buildContactMap(contacts.Support)); err != nil {
		return diagnosticsError(err, "Failed to set support field")
	}

	if err = d.Set("security", buildContactMap(contacts.Security)); err != nil {
		return diagnosticsError(err, "Failed to set security field")
	}

	return nil
}

func resourceContactsUpdate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	client := m.(*tsclient.Client)

	if d.HasChange("account") {
		if diagErr := updateContact(ctx, client, d, tsclient.ContactAccount); diagErr != nil {
			return diagErr
		}
	}

	if d.HasChange("support") {
		if diagErr := updateContact(ctx, client, d, tsclient.ContactSupport); diagErr != nil {
			return diagErr
		}
	}

	if d.HasChange("security") {
		if diagErr := updateContact(ctx, client, d, tsclient.ContactSecurity); diagErr != nil {
			return diagErr
		}
	}

	return resourceContactsRead(ctx, d, m)
}

func resourceContactsDelete(_ context.Context, _ *schema.ResourceData, _ interface{}) diag.Diagnostics {
	// Deleting is a no-op since we cannot have unset contact information.
	// Deletion in this context is simply removing from terraform state.
	const diagDetail = `This resource has been successfully destroyed, but values in tailscale will remain set.
See https://tailscale.com/kb/1224/contact-preferences to learn more.`

	return diag.Diagnostics{
		diag.Diagnostic{
			Severity: diag.Warning,
			Summary:  "Destroying tailscale_contacts does not unset contact values on tailscale",
			Detail:   diagDetail,
		},
	}
}

// buildContactMap transforms a tailscale.Contact into an equivalnet single element
// slice of map[string]interface{} so that it can be set on a schema.TypeSet property
// in the resource.
func buildContactMap(contact tsclient.Contact) []map[string]interface{} {
	return []map[string]interface{}{
		{
			"email": contact.Email,
		},
	}
}

// updateContact updates the contact specified by the tailscale.ContactType by
// reading the resource property with the correct name and using it to build a
// request to the underlying client.
func updateContact(ctx context.Context, client *tsclient.Client, d *schema.ResourceData, contactType tsclient.ContactType) diag.Diagnostics {
	contact := d.Get(string(contactType)).(*schema.Set).List()
	contactEmail := contact[0].(map[string]interface{})["email"].(string)

	if err := client.Contacts().Update(ctx, contactType, tsclient.UpdateContactRequest{Email: &contactEmail}); err != nil {
		return diagnosticsError(err, "Failed to create contacts")
	}

	return nil
}
