package tailscale_test

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

const testACL = `
	resource "tailscale_acl" "test_acl" {
		acl = <<EOF
		{
			// Access control lists.
			"ACLs": [
				{
					"Action": "accept",
					"Users": ["*"],
					"Ports": ["*:*"]
				}
			],
			"TagOwners": {
				"tag:example": [
					"group:example"
				]
			},
			// Declare static groups of users
			"Groups": {
				"group:example": [
					"user1@example.com",
					"user2@example.com"
				]
			},
			// Declare convenient hostname aliases to use in place of IP addresses.
			"Hosts": {
				"example-host-1": "100.100.100.100",
				"example-host-2": "100.100.101.100/24"
			},
			"Tests": [
				{
					"User": "user1@example.com",
					"Allow": ["example-host-1:22", "example-host-2:80"],
					"Deny": ["exapmle-host-2:100"]
				},
				{
					"User": "user2@example.com",
					"Allow": ["100.60.3.4:22"]
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
