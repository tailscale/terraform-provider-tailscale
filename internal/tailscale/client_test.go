package tailscale_test

import (
	"encoding/json"
	"testing"

	"github.com/davidsbond/terraform-provider-tailscale/internal/tailscale"
	"github.com/google/go-cmp/cmp"
	"github.com/tailscale/hujson"
)

func TestDomainACL_HuJSON_Unmarshal(t *testing.T) {
	acl := `
	{
		// Allow all users access to all ports.
		"ACLS": [
			{
				"Action": "accept",
				"Users": ["*"],
				"Ports": ["*:*"]
			}
		],
		"TagOwners": {
			"tag:example": [
				"group:example",
			]
		},
		"Groups": {
			"group:example": [
				"user1@example.com",
				"user2@example.com",
			]
		},
		"Hosts": {
			"example-host-1": "100.100.100.100",
			"example-host-2": "100.100.101.100/24",
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
				"Deny": ["exapmle-host-2:100"],
			},
			{
				"User": "user2@example.com",
				"Allow": ["100.60.3.4:22"],
			}
		]
	}`

	var actual tailscale.ACL
	if err := hujson.Unmarshal([]byte(acl), &actual); err != nil {
		t.Fatal(err)
	}

	expected := tailscale.ACL{
		ACLs: []tailscale.ACLEntry{
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
		DerpMap: &tailscale.DERPMap{
			Regions: map[int]*tailscale.DERPRegion{
				900: {
					RegionID:   900,
					RegionCode: "example",
					RegionName: "example",
					Nodes: []*tailscale.DERPNode{
						{
							Name:     "1",
							RegionID: 900,
							HostName: "example.com",
						},
					},
				},
			},
		},
		Tests: []tailscale.ACLTest{
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

func TestDomainACL_JSON_Unmarshal(t *testing.T) {
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
		"derpMap": {
			"regions": {
				"900": {
					"regionID": 900,
					"regionCode": "example",
					"regionName": "example",
					"nodes": [{
						"name": "1",
						"regionID": 900,
						"hostName": "example.com"
					}]
				}
			}
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

	var actual tailscale.ACL
	if err := json.Unmarshal([]byte(acl), &actual); err != nil {
		t.Fatal(err)
	}

	expected := tailscale.ACL{
		ACLs: []tailscale.ACLEntry{
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
		DerpMap: &tailscale.DERPMap{
			Regions: map[int]*tailscale.DERPRegion{
				900: {
					RegionID:   900,
					RegionCode: "example",
					RegionName: "example",
					Nodes: []*tailscale.DERPNode{
						{
							Name:     "1",
							RegionID: 900,
							HostName: "example.com",
						},
					},
				},
			},
		},
		Tests: []tailscale.ACLTest{
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
