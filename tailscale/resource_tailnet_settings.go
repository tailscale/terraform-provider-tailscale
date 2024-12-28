// Copyright (c) David Bond, Tailscale Inc, & Contributors
// SPDX-License-Identifier: MIT

package tailscale

import (
	"context"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"

	tsclient "github.com/tailscale/tailscale-client-go/v2"
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
			"devices_approval_on": {
				Type:        schema.TypeBool,
				Description: "Whether device approval is enabled for the tailnet",
				Optional:    true,
			},
			"devices_auto_updates_on": {
				Type:        schema.TypeBool,
				Description: "Whether auto updates are enabled for devices that belong to this tailnet",
				Optional:    true,
			},
			"devices_key_duration_days": {
				Type:        schema.TypeInt,
				Description: "The key expiry duration for devices on this tailnet",
				Optional:    true,
			},
			"users_approval_on": {
				Type:        schema.TypeBool,
				Description: "Whether user approval is enabled for this tailnet",
				Optional:    true,
			},
			"users_role_allowed_to_join_external_tailnet": {
				Type:        schema.TypeString,
				Description: "Which user roles are allowed to join external tailnets",
				Optional:    true,
				ValidateFunc: validation.StringInSlice(
					[]string{
						string(tsclient.RoleAllowedToJoinExternalTailnetsNone),
						string(tsclient.RoleAllowedToJoinExternalTailnetsMember),
						string(tsclient.RoleAllowedToJoinExternalTailnetsAdmin),
					},
					false,
				),
			},
			"network_flow_logging_on": {
				Type:        schema.TypeBool,
				Description: "Whether network flog logs are enabled for the tailnet",
				Optional:    true,
			},
			"regional_routing_on": {
				Type:        schema.TypeBool,
				Description: "Whether regional routing is enabled for the tailnet",
				Optional:    true,
			},
			"posture_identity_collection_on": {
				Type:        schema.TypeBool,
				Description: "Whether identity collection is enabled for device posture integrations for the tailnet",
				Optional:    true,
			},
		},
		EnableLegacyTypeSystemApplyErrors: true,
		EnableLegacyTypeSystemPlanErrors:  true,
	}
}

func resourceTailnetSettingsRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	client := m.(*tsclient.Client)

	settings, err := client.TailnetSettings().Get(ctx)
	if err != nil {
		return diagnosticsError(err, "Failed to fetch tailnet settings")
	}

	settingsMap := map[string]any{
		"devices_approval_on":                         settings.DevicesApprovalOn,
		"devices_auto_updates_on":                     settings.DevicesAutoUpdatesOn,
		"devices_key_duration_days":                   settings.DevicesKeyDurationDays,
		"users_approval_on":                           settings.UsersApprovalOn,
		"users_role_allowed_to_join_external_tailnet": string(settings.UsersRoleAllowedToJoinExternalTailnets),
		"network_flow_logging_on":                     settings.NetworkFlowLoggingOn,
		"regional_routing_on":                         settings.RegionalRoutingOn,
		"posture_identity_collection_on":              settings.PostureIdentityCollectionOn,
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
	var role *tsclient.RoleAllowedToJoinExternalTailnets
	_role, ok := d.GetOk("users_role_allowed_to_join_external_tailnet")
	if ok {
		role = tsclient.PointerTo(tsclient.RoleAllowedToJoinExternalTailnets(_role.(string)))
	}
	settings := tsclient.UpdateTailnetSettingsRequest{
		DevicesApprovalOn:                      optional[bool](d, "devices_approval_on"),
		DevicesAutoUpdatesOn:                   optional[bool](d, "devices_auto_updates_on"),
		DevicesKeyDurationDays:                 optional[int](d, "devices_key_duration_days"),
		UsersApprovalOn:                        optional[bool](d, "users_approval_on"),
		UsersRoleAllowedToJoinExternalTailnets: role,
		NetworkFlowLoggingOn:                   optional[bool](d, "network_flow_logging_on"),
		RegionalRoutingOn:                      optional[bool](d, "regional_routing_on"),
		PostureIdentityCollectionOn:            optional[bool](d, "posture_identity_collection_on"),
	}

	client := m.(*tsclient.Client)
	if err := client.TailnetSettings().Update(ctx, settings); err != nil {
		return diagnosticsError(err, "Failed to update tailnet settings")
	}

	return nil
}

func resourceTailnetSettingsDelete(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	// We don't know what the default values for Tailnet settings should be, so deleting is a noop.
	return nil
}
