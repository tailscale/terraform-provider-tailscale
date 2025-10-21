// Copyright (c) David Bond, Tailscale Inc, & Contributors
// SPDX-License-Identifier: MIT

package tailscale

import (
	"context"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"

	"tailscale.com/client/tailscale/v2"
)

func resourceTailnetSettings() *schema.Resource {
	return &schema.Resource{
		Description:   "The tailnet_settings resource allows you to configure settings for your tailnet. See https://tailscale.com/api#tag/tailnetsettings for more information.",
		ReadContext:   resourceTailnetSettingsRead,
		CreateContext: resourceTailnetSettingsCreate,
		UpdateContext: resourceTailnetSettingsUpdate,
		DeleteContext: resourceTailnetSettingsDelete,
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},
		Schema: map[string]*schema.Schema{
			"acls_externally_managed_on": {
				Type:        schema.TypeBool,
				Description: "Prevent users from editing policies in the admin console to avoid conflicts with external management workflows like GitOps or Terraform.",
				Optional:    true,
				Computed:    true,
			},
			"acls_external_link": {
				Type:         schema.TypeString,
				Description:  "Link to your external ACL definition or management system. Must be a valid URL.",
				ValidateFunc: validation.IsURLWithHTTPorHTTPS,
				Optional:     true,
				Computed:     true,
			},
			"devices_approval_on": {
				Type:        schema.TypeBool,
				Description: "Whether device approval is enabled for the tailnet",
				Optional:    true,
				Computed:    true,
			},
			"devices_auto_updates_on": {
				Type:        schema.TypeBool,
				Description: "Whether auto updates are enabled for devices that belong to this tailnet",
				Optional:    true,
				Computed:    true,
			},
			"devices_key_duration_days": {
				Type:        schema.TypeInt,
				Description: "The key expiry duration for devices on this tailnet",
				Optional:    true,
				Computed:    true,
			},
			"users_approval_on": {
				Type:        schema.TypeBool,
				Description: "Whether user approval is enabled for this tailnet",
				Optional:    true,
				Computed:    true,
			},
			"users_role_allowed_to_join_external_tailnet": {
				Type:        schema.TypeString,
				Description: "Which user roles are allowed to join external tailnets",
				Optional:    true,
				Computed:    true,
				ValidateFunc: validation.StringInSlice(
					[]string{
						string(tailscale.RoleAllowedToJoinExternalTailnetsNone),
						string(tailscale.RoleAllowedToJoinExternalTailnetsMember),
						string(tailscale.RoleAllowedToJoinExternalTailnetsAdmin),
					},
					false,
				),
			},
			"network_flow_logging_on": {
				Type:        schema.TypeBool,
				Description: "Whether network flow logs are enabled for the tailnet",
				Optional:    true,
				Computed:    true,
			},
			"regional_routing_on": {
				Type:        schema.TypeBool,
				Description: "Whether regional routing is enabled for the tailnet",
				Optional:    true,
				Computed:    true,
			},
			"posture_identity_collection_on": {
				Type:        schema.TypeBool,
				Description: "Whether identity collection is enabled for device posture integrations for the tailnet",
				Optional:    true,
				Computed:    true,
			},
			"https_enabled": {
				Type:        schema.TypeBool,
				Description: "Whether provisioning of HTTPS certificates is enabled for the tailnet",
				Optional:    true,
				Computed:    true,
			},
		},
	}
}

func resourceTailnetSettingsRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	client := m.(*tailscale.Client)

	settings, err := client.TailnetSettings().Get(ctx)
	if err != nil {
		return diagnosticsError(err, "Failed to fetch tailnet settings")
	}

	settingsMap := map[string]any{
		"acls_externally_managed_on":                  settings.ACLsExternallyManagedOn,
		"acls_external_link":                          settings.ACLsExternalLink,
		"devices_approval_on":                         settings.DevicesApprovalOn,
		"devices_auto_updates_on":                     settings.DevicesAutoUpdatesOn,
		"devices_key_duration_days":                   settings.DevicesKeyDurationDays,
		"users_approval_on":                           settings.UsersApprovalOn,
		"users_role_allowed_to_join_external_tailnet": string(settings.UsersRoleAllowedToJoinExternalTailnets),
		"network_flow_logging_on":                     settings.NetworkFlowLoggingOn,
		"regional_routing_on":                         settings.RegionalRoutingOn,
		"posture_identity_collection_on":              settings.PostureIdentityCollectionOn,
		"https_enabled":                               settings.HTTPSEnabled,
	}
	return setProperties(d, settingsMap)
}

func resourceTailnetSettingsCreate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	if err := resourceTailnetSettingsDoUpdate(ctx, d, m); err != nil {
		return err
	}
	d.SetId(createUUID())
	return resourceTailnetSettingsRead(ctx, d, m)
}

func resourceTailnetSettingsUpdate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	if err := resourceTailnetSettingsDoUpdate(ctx, d, m); err != nil {
		return err
	}
	return resourceTailnetSettingsRead(ctx, d, m)
}

func resourceTailnetSettingsDoUpdate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	var role *tailscale.RoleAllowedToJoinExternalTailnets
	_role, ok := d.GetOk("users_role_allowed_to_join_external_tailnet")
	if ok {
		role = tailscale.PointerTo(tailscale.RoleAllowedToJoinExternalTailnets(_role.(string)))
	}
	settings := tailscale.UpdateTailnetSettingsRequest{
		ACLsExternallyManagedOn:                optional[bool](d, "acls_externally_managed_on"),
		ACLsExternalLink:                       optional[string](d, "acls_external_link"),
		DevicesApprovalOn:                      optional[bool](d, "devices_approval_on"),
		DevicesAutoUpdatesOn:                   optional[bool](d, "devices_auto_updates_on"),
		DevicesKeyDurationDays:                 optional[int](d, "devices_key_duration_days"),
		UsersApprovalOn:                        optional[bool](d, "users_approval_on"),
		UsersRoleAllowedToJoinExternalTailnets: role,
		NetworkFlowLoggingOn:                   optional[bool](d, "network_flow_logging_on"),
		RegionalRoutingOn:                      optional[bool](d, "regional_routing_on"),
		PostureIdentityCollectionOn:            optional[bool](d, "posture_identity_collection_on"),
		HTTPSEnabled:                           optional[bool](d, "https_enabled"),
	}

	client := m.(*tailscale.Client)
	if err := client.TailnetSettings().Update(ctx, settings); err != nil {
		return diagnosticsError(err, "Failed to update tailnet settings")
	}

	return nil
}

func resourceTailnetSettingsDelete(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	// We don't know what the default values for Tailnet settings should be, so deleting is a noop.
	return nil
}
