// Copyright (c) David Bond, Tailscale Inc, & Contributors
// SPDX-License-Identifier: MIT

package tailscale

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

const testAWSExternalID = `
	resource "tailscale_aws_external_id" "test" {}
`

var (
	testAWSExternalIDCheck = resource.ComposeTestCheckFunc(
		resource.TestCheckResourceAttrSet("tailscale_aws_external_id.test", "external_id"),
		resource.TestCheckResourceAttrSet("tailscale_aws_external_id.test", "tailscale_aws_account_id"),
	)
)

func TestAccTailscaleAWSExternalID(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV5ProviderFactories: testAccProviderFactories(t),
		Steps: []resource.TestStep{
			{
				Config: testAWSExternalID,
				Check:  testAWSExternalIDCheck,
			},
		},
	})
}

// Migration test to ensure the resource is unchanged when migrating
// from the plugin SDK to the plugin framework.
//
// See https://developer.hashicorp.com/terraform/plugin/framework/migrating/testing#terraform-data-resource-example
func TestAccTailscaleAWSExternalID_UpgradeToPluginFramework(t *testing.T) {
	checkResourceIsUnchangedInPluginFramework(t, testAWSExternalID, testAWSExternalIDCheck)
}
