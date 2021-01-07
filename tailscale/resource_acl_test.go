package tailscale_test

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

const testACL = `
	resource "tailscale_acl" "test_acl" {
		acl = <<EOF
		{
			"acls": [
				{
					"action": "accept",
					"users": ["*"],
					"ports": ["*:*"]
				}
			]
		}
		EOF
	}`

func TestProvider_TailscaleACL(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testProviderPreCheck(t) },
		ProviderFactories: providerFactories,
		Steps: []resource.TestStep{
			testResourceCreated("tailscale_acl.test_acl", testACL),
			testResourceDestroyed("tailscale_acl.test_acl", testACL),
		},
	})
}
