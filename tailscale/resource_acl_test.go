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
			],
			"tagowners": {
				"tag:example": [
					"group:example"
				]
			},
			"groups": {
				"group:example": [
					"user1@example.com",
					"user2@example.com"
				]
			},
			"hosts": {
				"example-host-1": "100.100.100.100",
				"example-host-2": "100.100.101.100/24"
			},
			"tests": [
				{
					"user": "user1@example.com",
					"allow": ["example-host-1:22", "example-host-2:80"],
					"deny": ["exapmle-host-2:100"]
				},
				{
					"user": "user2@example.com",
					"allow": ["100.60.3.4:22"]
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
