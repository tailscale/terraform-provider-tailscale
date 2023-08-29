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

// TestProvider_TailscaleACLDiffs checks that ACL keys specified with
// different casing than the one used by the API client do not result
// in spurious diffs in Terraform plan.
func TestProvider_TailscaleACLDiffs(t *testing.T) {
	// policyObject returns a map that, when serialized to JSON,
	// is a valid Tailscale policy with only the "Hosts" field set.
	policyObject := func(hostsKey string) map[string]map[string]string {
		return map[string]map[string]string{
			hostsKey: {"example": "100.101.102.103"},
		}
	}
	policyHCL := func(hostsKey string) string {
		j, err := json.MarshalIndent(policyObject(hostsKey), "", " ")
		if err != nil {
			t.Fatal(err)
		}
		return fmt.Sprintf(
			`resource "tailscale_acl" "test_acl" {
				acl = <<EOF
					%s
				EOF
			}`, j)
	}

	resource.Test(t, resource.TestCase{
		IsUnitTest:        true,
		ProviderFactories: testProviderFactories(t),
		PreCheck: func() {
			testServer.ResponseCode = http.StatusOK
		},
		Steps: []resource.TestStep{
			testResourceCreated("tailscale_acl.test_acl", policyHCL("hosts")),

			// Now we check that, whatever spelling of "hosts" we use, the
			// Terraform plan will be empty.
			{ResourceName: "tailscale_acl.test_acl", Config: policyHCL("hosts"),
				PreConfig: func() {
					testServer.ResponseBody = policyObject("HOSTS")
				},
			},
			{ResourceName: "tailscale_acl.test_acl", Config: policyHCL("Hosts")},
			{ResourceName: "tailscale_acl.test_acl", Config: policyHCL("HoStS")},
			{ResourceName: "tailscale_acl.test_acl", Config: policyHCL("HOSTS")},
		},
	})
}
