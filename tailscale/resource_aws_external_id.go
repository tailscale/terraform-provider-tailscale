// Copyright (c) David Bond, Tailscale Inc, & Contributors
// SPDX-License-Identifier: MIT

package tailscale

import (
	"context"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"

	tsclient "github.com/tailscale/tailscale-client-go/v2"
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
	client := m.(*tsclient.Client)

	// We pass "reusable: false" on purpose.
	//
	// "reusable: true" is an optimization intended for the admin console UI's usage pattern
	// (i.e. to allow an external ID to be shown speculatively on a logstream configuration page
	// even though the user may not ultimately use it), to avoid unnecessarily minting external IDs
	//  that aren't actually going to be used. The Terraform usage pattern won't have this issue.
	//
	// If we did pass "reusable: true" here, provider users might run into potentially-confusing
	// "external ID already linked to a different account" errors in certain circumstances, e.g.:
	// - If they create two tailscale_aws_external_id resources, intended for two different
	//   tailscale_logstream_configuration resources that stream to different AWS accounts.
	//   (The two tailscale_aws_external_id resources would actually correspond to the same
	//   external ID.)
	// - If they create a tailscale_aws_external_id resource, but don't immediately link it to
	//   a tailscale_logstream_configuration resource, and the external ID is reused out-of-band
	//   (e.g. in the admin console UI).
	aid, err := client.Logging().CreateOrGetAwsExternalId(ctx, tsclient.CreateOrGetAwsExternalIdRequest{Reusable: false})
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
