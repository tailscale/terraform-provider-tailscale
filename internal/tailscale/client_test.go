package tailscale_test

import (
	"encoding/json"
	"testing"

	"github.com/davidsbond/terraform-provider-tailscale/internal/tailscale"
	"github.com/google/go-cmp/cmp"
)

func TestDomainACL_Unmarshal(t *testing.T) {
	acl := `
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
	}`

	var actual tailscale.DomainACL
	if err := json.Unmarshal([]byte(acl), &actual); err != nil {
		t.Fatal(err)
	}

	expected := tailscale.DomainACL{
		ACLs: []tailscale.DomainACLEntry{
			{
				Action: "accept",
				Ports:  []string{"*:*"},
				Users:  []string{"*"},
			},
		},
		TagOwners: map[string][]string{
			"tag:example": {"group:example"},
		},
		Hosts: map[string]string{
			"example-host-1": "100.100.100.100",
			"example-host-2": "100.100.101.100/24",
		},
		Groups: map[string][]string{
			"group:example": {
				"user1@example.com",
				"user2@example.com",
			},
		},
		Tests: []tailscale.DomainACLTest{
			{
				User:  "user1@example.com",
				Allow: []string{"example-host-1:22", "example-host-2:80"},
				Deny:  []string{"exapmle-host-2:100"},
			},
			{
				User:  "user2@example.com",
				Allow: []string{"100.60.3.4:22"},
			},
		},
	}

	if !cmp.Equal(expected, actual) {
		t.Fatal("unmarshalled ACL does not match expected value")
	}
}
