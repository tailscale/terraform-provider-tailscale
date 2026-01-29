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

func resourcePostureIntegration() *schema.Resource {
	return &schema.Resource{
		Description:   "The posture_integration resource allows you to manage integrations with device posture data providers. See https://tailscale.com/kb/1288/device-posture for more information.",
		ReadContext:   resourcePostureIntegrationRead,
		CreateContext: resourcePostureIntegrationCreate,
		UpdateContext: resourcePostureIntegrationUpdate,
		DeleteContext: resourcePostureIntegrationDelete,
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},
		Schema: map[string]*schema.Schema{
			"posture_provider": {
				Type:        schema.TypeString,
				Description: "The third-party provider for posture data. Valid values are `falcon`, `intune`, `jamfpro`, `kandji`, `kolide`, and `sentinelone`.",
				Required:    true,
				ForceNew:    true,
				ValidateFunc: validation.StringInSlice(
					[]string{
						string(tailscale.PostureIntegrationProviderFalcon),
						string(tailscale.PostureIntegrationProviderIntune),
						string(tailscale.PostureIntegrationProviderJamfPro),
						string(tailscale.PostureIntegrationProviderKandji),
						string(tailscale.PostureIntegrationProviderKolide),
						string(tailscale.PostureIntegrationProviderSentinelOne),
					},
					false,
				),
			},
			"cloud_id": {
				Type:        schema.TypeString,
				Description: "Identifies which of the provider's clouds to integrate with.",
				Optional:    true,
			},
			"client_id": {
				Type:        schema.TypeString,
				Description: "Unique identifier for your client.",
				Optional:    true,
			},
			"tenant_id": {
				Type:        schema.TypeString,
				Description: "The Microsoft Intune directory (tenant) ID. For other providers, this is left blank.",
				Optional:    true,
			},
			"client_secret": {
				Type:        schema.TypeString,
				Description: "The secret (auth key, token, etc.) used to authenticate with the provider.",
				Required:    true,
			},
		},
	}
}

func resourcePostureIntegrationRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	client := m.(*tailscale.Client)

	integration, err := client.DevicePosture().GetIntegration(ctx, d.Id())
	if err != nil {
		return diagnosticsError(err, "Failed to find posture integration with id %q", d.Id())
	}
	return resourcePostureIntegrationUpdateFromRemote(d, integration)
}

func resourcePostureIntegrationUpdateFromRemote(d *schema.ResourceData, integration *tailscale.PostureIntegration) diag.Diagnostics {
	if err := d.Set("posture_provider", string(integration.Provider)); err != nil {
		return diagnosticsError(err, "Failed to set posture_provider field")
	}
	if err := d.Set("cloud_id", string(integration.CloudID)); err != nil {
		return diagnosticsError(err, "Failed to set cloud_id field")
	}
	if err := d.Set("client_id", string(integration.ClientID)); err != nil {
		return diagnosticsError(err, "Failed to set client_id field")
	}
	if err := d.Set("tenant_id", string(integration.TenantID)); err != nil {
		return diagnosticsError(err, "Failed to set tenant_id field")
	}

	return nil
}

func resourcePostureIntegrationCreate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	client := m.(*tailscale.Client)

	integration, err := client.DevicePosture().CreateIntegration(
		ctx,
		tailscale.CreatePostureIntegrationRequest{
			Provider:     tailscale.PostureIntegrationProvider(d.Get("posture_provider").(string)),
			CloudID:      d.Get("cloud_id").(string),
			ClientID:     d.Get("client_id").(string),
			TenantID:     d.Get("tenant_id").(string),
			ClientSecret: d.Get("client_secret").(string),
		},
	)
	if err != nil {
		return diagnosticsError(err, "Failed to create posture integration")
	}

	d.SetId(integration.ID)
	return resourcePostureIntegrationUpdateFromRemote(d, integration)
}

func resourcePostureIntegrationUpdate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	client := m.(*tailscale.Client)

	integration, err := client.DevicePosture().UpdateIntegration(
		ctx,
		d.Id(),
		tailscale.UpdatePostureIntegrationRequest{
			CloudID:      d.Get("cloud_id").(string),
			ClientID:     d.Get("client_id").(string),
			TenantID:     d.Get("tenant_id").(string),
			ClientSecret: tailscale.PointerTo(d.Get("client_secret").(string)),
		})
	if err != nil {
		return diagnosticsError(err, "Failed to update posture integration with id %q", d.Id())
	}

	return resourcePostureIntegrationUpdateFromRemote(d, integration)
}

func resourcePostureIntegrationDelete(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	client := m.(*tailscale.Client)

	err := client.DevicePosture().DeleteIntegration(ctx, d.Id())
	if err != nil {
		return diagnosticsError(err, "Failed to delete posture integration with id %q", d.Id())
	}

	return nil
}
