// Copyright (c) David Bond, Tailscale Inc, & Contributors
// SPDX-License-Identifier: MIT

package tailscale

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// NewAWSExternalIDResource returns a new AWS External ID resource.
func NewAWSExternalIDResource() resource.Resource {
	return &awsExternalIDResource{}
}

type awsExternalIDResource struct {
	ResourceBase
}

// Metadata defines the resource name as it appears in Terraform configurations.
func (r *awsExternalIDResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_aws_external_id"
}

// Schema defines a schema describing what fields can be defined in the resource.
func (r *awsExternalIDResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "The aws_external_id resource allows you to mint an AWS External ID that Tailscale can use to assume an AWS IAM role that you create for the purposes of allowing Tailscale to stream logs to your S3 bucket. See the logstream_configuration resource for more details.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed: true,
			},
			"external_id": schema.StringAttribute{
				Computed:    true,
				Description: "The External ID that Tailscale will supply when assuming your role. You must reference this in your IAM role's trust policy. See https://docs.aws.amazon.com/IAM/latest/UserGuide/id_roles_common-scenarios_third-party.html for more information on external IDs.",
			},
			"tailscale_aws_account_id": schema.StringAttribute{
				Computed:    true,
				Description: "The AWS account from which Tailscale will assume your role. You must reference this in your IAM role's trust policy. See https://docs.aws.amazon.com/IAM/latest/UserGuide/id_roles_common-scenarios_third-party.html for more information on external IDs.",
			},
		},
	}
}

type awsExternalIDResourceData struct {
	ID                    types.String `tfsdk:"id"`
	ExternalID            types.String `tfsdk:"external_id"`
	TailscaleAWSAccountID types.String `tfsdk:"tailscale_aws_account_id"`
}

// Create creates a new AWS external ID.
func (r *awsExternalIDResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	// We pass "reusable: false" on purpose. Otherwise, two tailscale_aws_external_id resources
	// could end up with the same resource ID (because we use the actual external ID).
	//
	// Also, "reusable: true" is an optimization intended for the admin console UI's usage
	// pattern, and it's not really necessary for Terraform use cases.
	aid, err := r.Client.Logging().CreateOrGetAwsExternalId(ctx, false)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error creating AWS External ID",
			"Could not create AWS external ID, received error:"+err.Error(),
		)
		return
	}

	data := awsExternalIDResourceData{
		ID:                    types.StringValue(aid.ExternalID),
		ExternalID:            types.StringValue(aid.ExternalID),
		TailscaleAWSAccountID: types.StringValue(aid.TailscaleAWSAccountID),
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// There are no GET or DELETE endpoints in the API; this is a create-only resource.
// These methods are no-ops.
func (r *awsExternalIDResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
}
func (r *awsExternalIDResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
}
func (r *awsExternalIDResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
}
