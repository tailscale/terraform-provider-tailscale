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

func resourceLogstreamConfiguration() *schema.Resource {
	return &schema.Resource{
		Description:   "The logstream_configuration resource allows you to configure streaming configuration or network flow logs to a supported security information and event management (SIEM) system. See https://tailscale.com/kb/1255/log-streaming for more information.",
		ReadContext:   resourceLogstreamConfigurationRead,
		CreateContext: resourceLogstreamConfigurationCreate,
		UpdateContext: resourceLogstreamUpdate,
		DeleteContext: resourceLogstreamDelete,
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},
		Schema: map[string]*schema.Schema{
			"log_type": {
				Type:        schema.TypeString,
				Description: "The type of log that is streamed to this endpoint.",
				Required:    true,
				ForceNew:    true,
				ValidateFunc: validation.StringInSlice(
					[]string{
						string(tsclient.LogTypeConfig),
						string(tsclient.LogTypeNetwork),
					},
					false,
				),
			},
			"destination_type": {
				Type:        schema.TypeString,
				Description: "The type of system to which logs are being streamed.",
				Required:    true,
				ValidateFunc: validation.StringInSlice(
					[]string{
						string(tsclient.LogstreamSplunkEndpoint),
						string(tsclient.LogstreamElasticEndpoint),
						string(tsclient.LogstreamPantherEndpoint),
						string(tsclient.LogstreamCriblEndpoint),
						string(tsclient.LogstreamDatadogEndpoint),
						string(tsclient.LogstreamAxiomEndpoint),
					},
					false),
			},
			"url": {
				Type:        schema.TypeString,
				Description: "The URL to which log streams are being posted.",
				Required:    true,
			},
			"user": {
				Type:        schema.TypeString,
				Description: "The username with which log streams to this endpoint are authenticated. Only required if destination_type is 'elastic', defaults to 'user' if not set.",
				Optional:    true,
				Default:     "user",
			},
			"token": {
				Type:        schema.TypeString,
				Description: "The token/password with which log streams to this endpoint should be authenticated.",
				Required:    true,
				Sensitive:   true,
			},
		},
		EnableLegacyTypeSystemApplyErrors: true,
		EnableLegacyTypeSystemPlanErrors:  true,
	}
}

func resourceLogstreamConfigurationCreate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	client := m.(*tsclient.Client)

	logType := d.Get("log_type").(string)
	destinationType := d.Get("destination_type").(string)
	endpointURL := d.Get("url").(string)
	user := d.Get("user").(string)
	token := d.Get("token").(string)

	err := client.Logging().SetLogstreamConfiguration(ctx, tsclient.LogType(logType), tsclient.SetLogstreamConfigurationRequest{
		DestinationType: tsclient.LogstreamEndpointType(destinationType),
		URL:             endpointURL,
		User:            user,
		Token:           token,
	})

	if err != nil {
		return diagnosticsError(err, "Failed to create logstream configuration")
	}

	d.SetId(logType)
	return resourceLogstreamConfigurationRead(ctx, d, m)
}

func resourceLogstreamConfigurationRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	client := m.(*tsclient.Client)

	logstream, err := client.Logging().LogstreamConfiguration(ctx, tsclient.LogType(d.Id()))
	if err != nil {
		return diagnosticsError(err, "Failed to fetch logstream configuration")
	}

	if err = d.Set("log_type", string(logstream.LogType)); err != nil {
		return diagnosticsError(err, "Failed to set log_type field")
	}

	if err = d.Set("destination_type", string(logstream.DestinationType)); err != nil {
		return diagnosticsError(err, "Failed to set destination_type field")
	}

	if err = d.Set("url", logstream.URL); err != nil {
		return diagnosticsError(err, "Failed to set url field")
	}

	if err = d.Set("user", logstream.User); err != nil {
		return diagnosticsError(err, "Failed to set user field")
	}

	return nil
}

func resourceLogstreamUpdate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	// Update operation is the same as a create as we set / PUT the config.
	return resourceLogstreamConfigurationCreate(ctx, d, m)
}

func resourceLogstreamDelete(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	client := m.(*tsclient.Client)

	err := client.Logging().DeleteLogstreamConfiguration(ctx, tsclient.LogType(d.Id()))
	if err != nil {
		return diagnosticsError(err, "Failed to delete logstream configuration")
	}

	return nil
}
