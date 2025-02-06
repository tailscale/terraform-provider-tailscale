// Copyright (c) David Bond, Tailscale Inc, & Contributors
// SPDX-License-Identifier: MIT

package tailscale

import (
	"context"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"

	"tailscale.com/client/tailscale/v2"
)

func resourceAWSExternalID() *schema.Resource {
	return &schema.Resource{
		Description:   "The aws_external_id resource allows you to mint an AWS External ID that Tailscale can use to assume an AWS IAM role that you create for the purposes of allowing Tailscale to stream logs to your S3 bucket. See the logstream_configuration resource for more details.",
		CreateContext: resourceAWSExternalIDCreate,

		// No GET or DELETE endpoints in the API. This is a create-only resource.
		ReadContext:   schema.NoopContext,
		DeleteContext: schema.NoopContext,

		Schema: map[string]*schema.Schema{
			"external_id": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The External ID that Tailscale will supply when assuming your role. You must reference this in your IAM role's trust policy. See https://docs.aws.amazon.com/IAM/latest/UserGuide/id_roles_common-scenarios_third-party.html for more information on external IDs.",
			},
			"tailscale_aws_account_id": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The AWS account from which Tailscale will assume your role. You must reference this in your IAM role's trust policy. See https://docs.aws.amazon.com/IAM/latest/UserGuide/id_roles_common-scenarios_third-party.html for more information on external IDs.",
			},
		},
	}
}

func resourceAWSExternalIDCreate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	client := m.(*tailscale.Client)

	// We pass "reusable: false" on purpose. Otherwise, two tailscale_aws_external_id resources
	// could end up with the same resource ID (because we use the actual external ID).
	//
	// Also, "reusable: true" is an optimization intended for the admin console UI's usage
	// pattern, and it's not really necessary for Terraform use cases.
	aid, err := client.Logging().CreateOrGetAwsExternalId(ctx, false)
	if err != nil {
		return diagnosticsError(err, "Failed to create AWS External ID")
	}

	d.SetId(aid.ExternalID)
	if err = d.Set("external_id", aid.ExternalID); err != nil {
		return diagnosticsError(err, "Failed to set externalId")
	}
	if err = d.Set("tailscale_aws_account_id", aid.TailscaleAWSAccountID); err != nil {
		return diagnosticsError(err, "Failed to set AWSAccountID")
	}

	return nil
}
