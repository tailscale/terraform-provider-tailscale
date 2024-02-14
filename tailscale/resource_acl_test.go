package tailscale_test

import (
	"encoding/json"
	"fmt"
	"net/http"
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
			"DerpMap": {
				"Regions": {
					"900": {
						"RegionID": 900,
						"RegionCode": "example",
						"RegionName": "example",
						"Nodes": [{
							"Name": "1",
							"RegionID": 900,
							"HostName": "example.com"
						}]
					}
				}
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
		IsUnitTest: true,
		PreCheck: func() {
			testServer.ResponseCode = http.StatusOK
			testServer.ResponseBody = nil
		},
		ProviderFactories: testProviderFactories(t),
		Steps: []resource.TestStep{
			testResourceCreated("tailscale_acl.test_acl", testACL),
			testResourceDestroyed("tailscale_acl.test_acl", testACL),
		},
	})
}

// TestProvider_TailscaleACLDiffs checks that changes in whitespace
// do not cause diffs in the Terraform plan.
func TestProvider_TailscaleACLDiffs(t *testing.T) {
	policyJSON := func(indent string) []byte {
		j, err := json.MarshalIndent(map[string]map[string]string{
			"hosts": {"example": "100.101.102.103"},
		}, "", indent)
		if err != nil {
			t.Fatal(err)
		}
		return j
	}
	toHuJSON := func(j []byte) []byte {
		return []byte(fmt.Sprintf("// This is a HuJSON policy\n%s", j))
	}
	toHCL := func(policy []byte) string {
		return fmt.Sprintf(
			`resource "tailscale_acl" "test_acl" {
				acl = <<EOF
					%s
				EOF
			}`, policy)
	}

	resource.Test(t, resource.TestCase{
		IsUnitTest:        true,
		ProviderFactories: testProviderFactories(t),
		PreCheck: func() {
			testServer.ResponseCode = http.StatusOK
			testServer.ResponseBody = []byte("{}")
		},
		Steps: []resource.TestStep{
			testResourceCreated("tailscale_acl.test_acl", toHCL(policyJSON(" "))),

			// Now we check that whitespace changes result in empty plan.
			{ResourceName: "tailscale_acl.test_acl", Config: toHCL(policyJSON(" ")),
				PreConfig: func() {
					testServer.ResponseBody = policyJSON(" ")
				},
			},
			{ResourceName: "tailscale_acl.test_acl", Config: toHCL(policyJSON("\t"))},
			{ResourceName: "tailscale_acl.test_acl", Config: toHCL(policyJSON("      "))},

			// The same policy in HuJSON will result in a diff.
			{
				ResourceName: "tailscale_acl.test_acl", Config: toHCL(toHuJSON(policyJSON("  "))),
				ExpectNonEmptyPlan: true,
			},
			// Further changes in whitespace are not causing a diff.
			{ResourceName: "tailscale_acl.test_acl", Config: toHCL(toHuJSON(policyJSON("\t"))),
				PreConfig: func() {
					testServer.ResponseBody = toHuJSON(policyJSON("  "))
				},
			},
		},
	})
}
