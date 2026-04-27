// Copyright (c) David Bond, Tailscale Inc, & Contributors
// SPDX-License-Identifier: MIT

package tailscale

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"tailscale.com/client/tailscale/v2"
)

var (
	_ resource.Resource                = &logstreamConfigurationResource{}
	_ resource.ResourceWithImportState = &logstreamConfigurationResource{}
)

// NewLogstreamConfigurationResource returns a new logtsream configuration resource.
func NewLogstreamConfigurationResource() resource.Resource {
	return &logstreamConfigurationResource{}
}

type logstreamConfigurationResource struct {
	ResourceBase
}

// Metadata defines the resource name as it appears in Terraform configurations.
func (r *logstreamConfigurationResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_logstream_configuration"
}

// Schema defines a schema describing what fields can be defined in the resource.
func (r *logstreamConfigurationResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "The logstream_configuration resource allows you to configure streaming configuration or network flow logs to a supported security information and event management (SIEM) system. See https://tailscale.com/kb/1255/log-streaming for more information.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"log_type": schema.StringAttribute{
				Description: "The type of logs to stream. Valid values are `configuration` (configuration audit logs) and `network` (network flow logs).",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Validators: []validator.String{
					stringvalidator.OneOf(
						string(tailscale.LogTypeConfig),
						string(tailscale.LogTypeNetwork),
					),
				},
			},
			"destination_type": schema.StringAttribute{
				Description: "The type of SIEM platform to stream to. Valid values are `axiom`, `cribl`, `datadog`, `elastic`, `gcs`, `panther`, `splunk`, and `s3`.",
				Required:    true,
				Validators: []validator.String{
					stringvalidator.OneOf(
						string(tailscale.LogstreamAxiomEndpoint),
						string(tailscale.LogstreamDatadogEndpoint),
						string(tailscale.LogstreamCriblEndpoint),
						string(tailscale.LogstreamElasticEndpoint),
						string(tailscale.LogstreamPantherEndpoint),
						string(tailscale.LogstreamSplunkEndpoint),
						string(tailscale.LogstreamS3Endpoint),
						string(tailscale.LogstreamGCSEndpoint),
					),
				},
			},
			"url": schema.StringAttribute{
				Description: "The URL to which log streams are being posted. If destination_type is 's3' and you want to use the official Amazon S3 endpoint, leave this empty.",
				Optional:    true,
			},
			"user": schema.StringAttribute{
				Description: "The username with which log streams to this endpoint are authenticated. Only required if destination_type is 'elastic', defaults to 'user' if not set.",
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString("user"),
			},
			"token": schema.StringAttribute{
				Description: "The token/password with which log streams to this endpoint should be authenticated, required unless destination_type is 's3'.",
				Optional:    true,
				Sensitive:   true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"upload_period_minutes": schema.Int32Attribute{
				Description: "An optional number of minutes to wait in between uploading new logs. If the quantity of logs does not fit within a single upload, multiple uploads will be made.",
				Optional:    true,
			},
			"compression_format": schema.StringAttribute{
				Description: "The compression algorithm used for logs. Valid values are `none`, `zstd` or `gzip`. Defaults to `none`.",
				Computed:    true,
				Optional:    true,
				Default:     stringdefault.StaticString(string(tailscale.CompressionFormatNone)),
				Validators: []validator.String{
					stringvalidator.OneOf(
						string(tailscale.CompressionFormatNone),
						string(tailscale.CompressionFormatZstd),
						string(tailscale.CompressionFormatGzip),
					),
				},
			},
			"s3_bucket": schema.StringAttribute{
				Description: "The S3 bucket name. Required if destination_type is 's3'.",
				Optional:    true,
			},
			"s3_region": schema.StringAttribute{
				Description: "The region in which the S3 bucket is located. Required if destination_type is 's3'.",
				Optional:    true,
			},
			"s3_key_prefix": schema.StringAttribute{
				Description: "An optional S3 key prefix to prepend to the auto-generated S3 key name.",
				Optional:    true,
			},
			"s3_authentication_type": schema.StringAttribute{
				Description: "The type of authentication to use for S3. Required if destination_type is `s3`. Valid values are `accesskey` and `rolearn`. Tailscale recommends using `rolearn`.",
				Optional:    true,
				Validators: []validator.String{
					stringvalidator.OneOf(
						string(tailscale.S3AccessKeyAuthentication),
						string(tailscale.S3RoleARNAuthentication),
					),
				},
			},
			"s3_access_key_id": schema.StringAttribute{
				Description: "The S3 access key ID. Required if destination_type is s3 and s3_authentication_type is 'accesskey'.",
				Optional:    true,
			},
			"s3_secret_access_key": schema.StringAttribute{
				Description: "The S3 secret access key. Required if destination_type is 's3' and s3_authentication_type is 'accesskey'.",
				Optional:    true,
				Sensitive:   true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"s3_role_arn": schema.StringAttribute{
				Description: "ARN of the AWS IAM role that Tailscale should assume when using role-based authentication. Required if destination_type is 's3' and s3_authentication_type is 'rolearn'.",
				Optional:    true,
			},
			"s3_external_id": schema.StringAttribute{
				Description: "The AWS External ID that Tailscale supplies when authenticating using role-based authentication. Required if destination_type is 's3' and s3_authentication_type is 'rolearn'. This can be obtained via the tailscale_aws_external_id resource.",
				Optional:    true,
			},
			"gcs_credentials": schema.StringAttribute{
				Description: "The encoded string of JSON that is used to authenticate for workload identity in GCS",
				Optional:    true,
				PlanModifiers: []planmodifier.String{
					jsonSemanticDiffModifier{},
				},
			},
			"gcs_bucket": schema.StringAttribute{
				Description: "The name of the GCS bucket",
				Optional:    true,
			},
			"gcs_scopes": schema.SetAttribute{
				Description: "The GCS scopes needed to be able to write in the bucket",
				Optional:    true,
				ElementType: types.StringType,
			},
			"gcs_key_prefix": schema.StringAttribute{
				Description: "The GCS key prefix for the bucket",
				Optional:    true,
			},
		},
	}
}

type logstreamConfigurationResourceModel struct {
	ID                   types.String `tfsdk:"id"`
	LogType              types.String `tfsdk:"log_type"`
	DestinationType      types.String `tfsdk:"destination_type"`
	URL                  types.String `tfsdk:"url"`
	User                 types.String `tfsdk:"user"`
	Token                types.String `tfsdk:"token"`
	UploadPeriodMinutes  types.Int32  `tfsdk:"upload_period_minutes"`
	CompressionFormat    types.String `tfsdk:"compression_format"`
	S3Bucket             types.String `tfsdk:"s3_bucket"`
	S3Region             types.String `tfsdk:"s3_region"`
	S3KeyPrefix          types.String `tfsdk:"s3_key_prefix"`
	S3AuthenticationType types.String `tfsdk:"s3_authentication_type"`
	S3AccessKeyID        types.String `tfsdk:"s3_access_key_id"`
	S3SecretAccessKey    types.String `tfsdk:"s3_secret_access_key"`
	S3RoleARN            types.String `tfsdk:"s3_role_arn"`
	S3ExternalID         types.String `tfsdk:"s3_external_id"`
	GCSCredentials       types.String `tfsdk:"gcs_credentials"`
	GCSBucket            types.String `tfsdk:"gcs_bucket"`
	GCSScopes            types.Set    `tfsdk:"gcs_scopes"`
	GCSKeyPrefix         types.String `tfsdk:"gcs_key_prefix"`
}

func (d *logstreamConfigurationResourceModel) asRequest(ctx context.Context, diags *diag.Diagnostics) (tailscale.LogType, tailscale.SetLogstreamConfigurationRequest) {
	logType := tailscale.LogType(d.LogType.ValueString())

	var gcsScopes []string
	diags.Append(d.GCSScopes.ElementsAs(ctx, &gcsScopes, false)...)

	request := tailscale.SetLogstreamConfigurationRequest{
		DestinationType:      tailscale.LogstreamEndpointType(d.DestinationType.ValueString()),
		URL:                  d.URL.ValueString(),
		User:                 d.User.ValueString(),
		Token:                d.Token.ValueString(),
		UploadPeriodMinutes:  int(d.UploadPeriodMinutes.ValueInt32()),
		CompressionFormat:    tailscale.CompressionFormat(d.CompressionFormat.ValueString()),
		S3Bucket:             d.S3Bucket.ValueString(),
		S3Region:             d.S3Region.ValueString(),
		S3KeyPrefix:          d.S3KeyPrefix.ValueString(),
		S3AuthenticationType: tailscale.S3AuthenticationType(d.S3AuthenticationType.ValueString()),
		S3AccessKeyID:        d.S3AccessKeyID.ValueString(),
		S3SecretAccessKey:    d.S3SecretAccessKey.ValueString(),
		S3RoleARN:            d.S3RoleARN.ValueString(),
		S3ExternalID:         d.S3ExternalID.ValueString(),
		GCSCredentials:       d.GCSCredentials.ValueString(),
		GCSScopes:            gcsScopes,
		GCSKeyPrefix:         d.GCSKeyPrefix.ValueString(),
		GCSBucket:            d.GCSBucket.ValueString(),
	}

	return logType, request
}

// stringOrNull updates a value, but only if the existing value is non-null
// or the new value is non-empty.
//
// This avoids "ghost diffs" from Terraform where a stored `null` value is
// replaced with an empty string from the API, which appears as an empty
// diff during a refresh.
func stringOrNull(existing types.String, updated string) types.String {
	if updated != "" {
		return types.StringValue(updated)
	} else {
		return types.StringNull()
	}
}

func (d *logstreamConfigurationResourceModel) updateFields(ctx context.Context, config *tailscale.LogstreamConfiguration, diags *diag.Diagnostics) {
	d.ID = types.StringValue(string(config.LogType))
	d.LogType = types.StringValue(string(config.LogType))
	d.DestinationType = types.StringValue(string(config.DestinationType))
	d.URL = stringOrNull(d.URL, config.URL)
	d.User = types.StringValue(config.User)

	if config.UploadPeriodMinutes != 0 {
		d.UploadPeriodMinutes = types.Int32Value(int32(config.UploadPeriodMinutes))
	} else {
		d.UploadPeriodMinutes = types.Int32Null()
	}

	d.CompressionFormat = types.StringValue(string(config.CompressionFormat))

	d.S3Bucket = stringOrNull(d.S3Bucket, config.S3Bucket)
	d.S3Region = stringOrNull(d.S3Region, config.S3Region)
	d.S3KeyPrefix = stringOrNull(d.S3KeyPrefix, config.S3KeyPrefix)
	d.S3AuthenticationType = stringOrNull(d.S3AuthenticationType, string(config.S3AuthenticationType))
	d.S3AccessKeyID = stringOrNull(d.S3AccessKeyID, config.S3AccessKeyID)
	d.S3RoleARN = stringOrNull(d.S3RoleARN, config.S3RoleARN)
	d.S3ExternalID = stringOrNull(d.S3ExternalID, config.S3ExternalID)

	gcsScopes, scopeDiags := types.SetValueFrom(ctx, types.StringType, config.GCSScopes)
	diags.Append(scopeDiags...)
	d.GCSScopes = gcsScopes

	d.GCSCredentials = stringOrNull(d.GCSCredentials, config.GCSCredentials)
	d.GCSKeyPrefix = stringOrNull(d.GCSKeyPrefix, config.GCSKeyPrefix)
	d.GCSBucket = stringOrNull(d.GCSBucket, config.GCSBucket)
}

// updateLogstreamConfiguration calls the Tailscale API to set logstream configuration.
func (r *logstreamConfigurationResource) updateLogstreamConfiguration(ctx context.Context, data *logstreamConfigurationResourceModel, diags *diag.Diagnostics) {
	logType, request := data.asRequest(ctx, diags)
	if diags.HasError() {
		return
	}

	if err := r.Client.Logging().SetLogstreamConfiguration(ctx, logType, request); err != nil {
		diags.AddError("Failed to set logstream configuration", err.Error())
	}
}

func (r *logstreamConfigurationResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan logstreamConfigurationResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	r.updateLogstreamConfiguration(ctx, &plan, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	plan.ID = plan.LogType
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *logstreamConfigurationResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state logstreamConfigurationResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	config, err := r.Client.Logging().LogstreamConfiguration(ctx, tailscale.LogType(state.ID.ValueString()))
	if err != nil {
		if tailscale.IsNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Failed to fetch logstream configuration", err.Error())
		return
	}

	state.updateFields(ctx, config, &resp.Diagnostics)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *logstreamConfigurationResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan logstreamConfigurationResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	r.updateLogstreamConfiguration(ctx, &plan, &resp.Diagnostics)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *logstreamConfigurationResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state logstreamConfigurationResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	logType := tailscale.LogType(state.LogType.ValueString())
	err := r.Client.Logging().DeleteLogstreamConfiguration(ctx, logType)
	if err != nil {
		resp.Diagnostics.AddError("Failed to delete logstream configuration", err.Error())
	}
}

func (r *logstreamConfigurationResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
