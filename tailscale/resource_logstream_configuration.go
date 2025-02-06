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
						string(tailscale.LogTypeConfig),
						string(tailscale.LogTypeNetwork),
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
						string(tailscale.LogstreamSplunkEndpoint),
						string(tailscale.LogstreamElasticEndpoint),
						string(tailscale.LogstreamPantherEndpoint),
						string(tailscale.LogstreamCriblEndpoint),
						string(tailscale.LogstreamDatadogEndpoint),
						string(tailscale.LogstreamAxiomEndpoint),
						string(tailscale.LogstreamS3Endpoint),
					},
					false),
			},
			"url": {
				Type:        schema.TypeString,
				Description: "The URL to which log streams are being posted. If destination_type is 's3' and you want to use the official Amazon S3 endpoint, leave this empty.",
				Optional:    true,
			},
			"user": {
				Type:        schema.TypeString,
				Description: "The username with which log streams to this endpoint are authenticated. Only required if destination_type is 'elastic', defaults to 'user' if not set.",
				Optional:    true,
				Default:     "user",
			},
			"token": {
				Type:        schema.TypeString,
				Description: "The token/password with which log streams to this endpoint should be authenticated, required unless destination_type is 's3'.",
				Optional:    true,
				Sensitive:   true,
			},
			"s3_bucket": {
				Type:        schema.TypeString,
				Description: "The S3 bucket name. Required if destination_type is 's3'.",
				Optional:    true,
			},
			"s3_region": {
				Type:        schema.TypeString,
				Description: "The region in which the S3 bucket is located. Required if destination_type is 's3'.",
				Optional:    true,
			},
			"s3_key_prefix": {
				Type:        schema.TypeString,
				Description: "An optional S3 key prefix to prepend to the auto-generated S3 key name.",
				Optional:    true,
			},
			"s3_authentication_type": {
				Type:        schema.TypeString,
				Description: "What type of authentication to use for S3. Required if destination_type is 's3'. Tailscale recommends using 'rolearn'.",
				Optional:    true,
				ValidateFunc: validation.StringInSlice(
					[]string{
						string(tailscale.S3AccessKeyAuthentication),
						string(tailscale.S3RoleARNAuthentication),
					},
					false,
				),
			},
			"s3_access_key_id": {
				Type:        schema.TypeString,
				Description: "The S3 access key ID. Required if destination_type is s3 and s3_authentication_type is 'accesskey'.",
				Optional:    true,
			},
			"s3_secret_access_key": {
				Type:        schema.TypeString,
				Description: "The S3 secret access key. Required if destination_type is 's3' and s3_authentication_type is 'accesskey'.",
				Optional:    true,
				Sensitive:   true,
			},
			"s3_role_arn": {
				Type:        schema.TypeString,
				Description: "ARN of the AWS IAM role that Tailscale should assume when using role-based authentication. Required if destination_type is 's3' and s3_authentication_type is 'rolearn'.",
				Optional:    true,
			},
			"s3_external_id": {
				Type:        schema.TypeString,
				Description: "The AWS External ID that Tailscale supplies when authenticating using role-based authentication. Required if destination_type is 's3' and s3_authentication_type is 'rolearn'. This can be obtained via the tailscale_aws_external_id resource.",
				Optional:    true,
			},
		},
	}
}

func resourceLogstreamConfigurationCreate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	client := m.(*tailscale.Client)

	logType := d.Get("log_type").(string)
	destinationType := d.Get("destination_type").(string)
	endpointURL := d.Get("url").(string)
	user := d.Get("user").(string)
	token := d.Get("token").(string)
	s3Bucket := d.Get("s3_bucket").(string)
	s3Region := d.Get("s3_region").(string)
	s3KeyPrefix := d.Get("s3_key_prefix").(string)
	s3AuthenticationType := tailscale.S3AuthenticationType(d.Get("s3_authentication_type").(string))
	s3AccessKeyID := d.Get("s3_access_key_id").(string)
	s3SecretAccessKey := d.Get("s3_secret_access_key").(string)
	s3RoleARN := d.Get("s3_role_arn").(string)
	s3ExternalID := d.Get("s3_external_id").(string)

	err := client.Logging().SetLogstreamConfiguration(ctx, tailscale.LogType(logType), tailscale.SetLogstreamConfigurationRequest{
		DestinationType:      tailscale.LogstreamEndpointType(destinationType),
		URL:                  endpointURL,
		User:                 user,
		Token:                token,
		S3Bucket:             s3Bucket,
		S3Region:             s3Region,
		S3KeyPrefix:          s3KeyPrefix,
		S3AuthenticationType: s3AuthenticationType,
		S3AccessKeyID:        s3AccessKeyID,
		S3SecretAccessKey:    s3SecretAccessKey,
		S3RoleARN:            s3RoleARN,
		S3ExternalID:         s3ExternalID,
	})

	if err != nil {
		return diagnosticsError(err, "Failed to create logstream configuration")
	}

	d.SetId(logType)
	return resourceLogstreamConfigurationRead(ctx, d, m)
}

func resourceLogstreamConfigurationRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	client := m.(*tailscale.Client)

	logstream, err := client.Logging().LogstreamConfiguration(ctx, tailscale.LogType(d.Id()))
	if err != nil && tailscale.IsNotFound(err) {
		d.SetId("")
		return nil
	}

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

	if err := d.Set("s3_bucket", logstream.S3Bucket); err != nil {
		return diagnosticsError(err, "Failed to set s3_bucket field")
	}

	if err := d.Set("s3_region", logstream.S3Region); err != nil {
		return diagnosticsError(err, "Failed to set s3_region field")
	}

	if err := d.Set("s3_key_prefix", logstream.S3KeyPrefix); err != nil {
		return diagnosticsError(err, "Failed to set s3_key_prefix field")
	}

	if err := d.Set("s3_authentication_type", logstream.S3AuthenticationType); err != nil {
		return diagnosticsError(err, "Failed to set s3_authentication_type field")
	}

	if err := d.Set("s3_access_key_id", logstream.S3AccessKeyID); err != nil {
		return diagnosticsError(err, "Failed to set s3_access_key_id field")
	}

	if err := d.Set("s3_role_arn", logstream.S3RoleARN); err != nil {
		return diagnosticsError(err, "Failed to set s3_role_arn field")
	}

	if err := d.Set("s3_external_id", logstream.S3ExternalID); err != nil {
		return diagnosticsError(err, "Failed to set s3_external_id field")
	}

	return nil
}

func resourceLogstreamUpdate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	// Update operation is the same as a create as we set / PUT the config.
	return resourceLogstreamConfigurationCreate(ctx, d, m)
}

func resourceLogstreamDelete(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	client := m.(*tailscale.Client)

	err := client.Logging().DeleteLogstreamConfiguration(ctx, tailscale.LogType(d.Id()))
	if err != nil {
		return diagnosticsError(err, "Failed to delete logstream configuration")
	}

	return nil
}
