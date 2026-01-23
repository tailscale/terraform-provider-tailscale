// Copyright (c) David Bond, Tailscale Inc, & Contributors
// SPDX-License-Identifier: MIT

package tailscale

import (
	"context"
	"encoding/json"

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
				Description: "The type of logs to stream. Valid values are `configuration` (configuration audit logs) and `network` (network flow logs).",
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
				Description: "The type of SIEM platform to stream to. Valid values are `axiom`, `cribl`, `datadog`, `elastic`, `gcs`, `panther`, `splunk`, and `s3`.",
				Required:    true,
				ValidateFunc: validation.StringInSlice(
					[]string{
						string(tailscale.LogstreamAxiomEndpoint),
						string(tailscale.LogstreamDatadogEndpoint),
						string(tailscale.LogstreamCriblEndpoint),
						string(tailscale.LogstreamElasticEndpoint),
						string(tailscale.LogstreamPantherEndpoint),
						string(tailscale.LogstreamSplunkEndpoint),
						string(tailscale.LogstreamS3Endpoint),
						string(tailscale.LogstreamGCSEndpoint),
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
			"upload_period_minutes": {
				Type:        schema.TypeInt,
				Description: "An optional number of minutes to wait in between uploading new logs. If the quantity of logs does not fit within a single upload, multiple uploads will be made.",
				Optional:    true,
			},
			"compression_format": {
				Type:        schema.TypeString,
				Description: "The compression algorithm used for logs. Valid values are `none`, `zstd` or `gzip`. Defaults to `none`.",
				Optional:    true,
				Default:     string(tailscale.CompressionFormatNone),
				ValidateFunc: validation.StringInSlice(
					[]string{
						string(tailscale.CompressionFormatNone),
						string(tailscale.CompressionFormatZstd),
						string(tailscale.CompressionFormatGzip),
					},
					false,
				),
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
				Description: "The type of authentication to use for S3. Required if destination_type is `s3`. Valid values are `accesskey` and `rolearn`. Tailscale recommends using `rolearn`.",
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
			"gcs_credentials": {
				Type:        schema.TypeString,
				Description: "The encoded string of JSON that is used to authenticate for workload identity in GCS",
				Optional:    true,
				// Suppress the diff if the JSON value is semantically identical even if the content is different (e.g.,
				// due to whitespace differences / sorting of keys in the returned JSON encoded string).
				DiffSuppressFunc: func(k, oldValue, newValue string, d *schema.ResourceData) bool {
					container := make(map[string]any)
					if oldErr := json.Unmarshal([]byte(oldValue), &container); oldErr != nil {
						return false
					}
					oldValueBytes, oldErr := json.Marshal(container)

					if newErr := json.Unmarshal([]byte(newValue), &container); newErr != nil {
						return false
					}
					newValueBytes, newErr := json.Marshal(container)

					if oldErr != nil || newErr != nil {
						return false
					}
					return string(oldValueBytes) == string(newValueBytes)
				},
				DiffSuppressOnRefresh: true,
			},
			"gcs_bucket": {
				Type:        schema.TypeString,
				Description: "The name of the GCS bucket",
				Optional:    true,
			},
			"gcs_scopes": {
				Type:        schema.TypeSet,
				Description: "The GCS scopes needed to be able to write in the bucket",
				Optional:    true,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
			},
			"gcs_key_prefix": {
				Type:        schema.TypeString,
				Description: "The GCS key prefix for the bucket",
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
	uploadPeriodMinutes := d.Get("upload_period_minutes").(int)
	compressionFormat := d.Get("compression_format").(string)
	s3Bucket := d.Get("s3_bucket").(string)
	s3Region := d.Get("s3_region").(string)
	s3KeyPrefix := d.Get("s3_key_prefix").(string)
	s3AuthenticationType := tailscale.S3AuthenticationType(d.Get("s3_authentication_type").(string))
	s3AccessKeyID := d.Get("s3_access_key_id").(string)
	s3SecretAccessKey := d.Get("s3_secret_access_key").(string)
	s3RoleARN := d.Get("s3_role_arn").(string)
	s3ExternalID := d.Get("s3_external_id").(string)
	gcsCredentials := d.Get("gcs_credentials").(string)
	gcsKeyPrefix := d.Get("gcs_key_prefix").(string)
	gcsBucket := d.Get("gcs_bucket").(string)

	var gcsScopes []string
	for _, scope := range d.Get("gcs_scopes").(*schema.Set).List() {
		gcsScopes = append(gcsScopes, scope.(string))
	}

	err := client.Logging().SetLogstreamConfiguration(ctx, tailscale.LogType(logType), tailscale.SetLogstreamConfigurationRequest{
		DestinationType:      tailscale.LogstreamEndpointType(destinationType),
		URL:                  endpointURL,
		User:                 user,
		Token:                token,
		UploadPeriodMinutes:  uploadPeriodMinutes,
		CompressionFormat:    tailscale.CompressionFormat(compressionFormat),
		S3Bucket:             s3Bucket,
		S3Region:             s3Region,
		S3KeyPrefix:          s3KeyPrefix,
		S3AuthenticationType: s3AuthenticationType,
		S3AccessKeyID:        s3AccessKeyID,
		S3SecretAccessKey:    s3SecretAccessKey,
		S3RoleARN:            s3RoleARN,
		S3ExternalID:         s3ExternalID,
		GCSCredentials:       gcsCredentials,
		GCSScopes:            gcsScopes,
		GCSKeyPrefix:         gcsKeyPrefix,
		GCSBucket:            gcsBucket,
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

	if err = d.Set("upload_period_minutes", logstream.UploadPeriodMinutes); err != nil {
		return diagnosticsError(err, "Failed to set upload_period_minutes field")
	}

	if err = d.Set("compression_format", logstream.CompressionFormat); err != nil {
		return diagnosticsError(err, "Failed to set compression_format field")
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

	if err := d.Set("gcs_credentials", logstream.GCSCredentials); err != nil {
		return diagnosticsError(err, "Failed to set gcs_credentials field")
	}

	if err := d.Set("gcs_scopes", logstream.GCSScopes); err != nil {
		return diagnosticsError(err, "Failed to set gcs_scopes field")
	}

	if err := d.Set("gcs_key_prefix", logstream.GCSKeyPrefix); err != nil {
		return diagnosticsError(err, "Failed to set gcs_key_prefix field")
	}

	if err := d.Set("gcs_bucket", logstream.GCSBucket); err != nil {
		return diagnosticsError(err, "Failed to set gcs_bucket field")
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
