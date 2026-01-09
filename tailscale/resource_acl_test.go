// Copyright (c) David Bond, Tailscale Inc, & Contributors
// SPDX-License-Identifier: MIT

package tailscale

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"

	"tailscale.com/client/tailscale/v2"
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

func TestAccACL(t *testing.T) {
	const resourceName = "tailscale_acl.test_acl"

	const testACLCreate = `
		resource "tailscale_acl" "test_acl" {
		    overwrite_existing_content = true
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
			}
			EOF
		}`

	const testACLUpdate = `
		resource "tailscale_acl" "test_acl" {
			acl = <<EOF
			{
				// Tag owners.
				"TagOwners": {
					"tag:example": [
						"autogroup:member"
					]
				},
			}
			EOF
		}`

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories(t),
		Steps: []resource.TestStep{
			{
				Config: testACLCreate,
				Check: resource.ComposeTestCheckFunc(
					checkResourceRemoteProperties(resourceName,
						checkProperties(&tailscale.ACL{
							ACLs: []tailscale.ACLEntry{
								{
									Action: "accept",
									Users:  []string{"*"},
									Ports:  []string{"*:*"},
								},
							},
						}),
					),
				),
			},
			{
				Config: testACLUpdate,
				Check: resource.ComposeTestCheckFunc(
					checkResourceRemoteProperties(resourceName,
						checkProperties(&tailscale.ACL{
							TagOwners: map[string][]string{
								"tag:example": {"autogroup:member"},
							},
						}),
					),
				),
			},
			{
				ResourceName:            resourceName,
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"overwrite_existing_content"},
			},
		},
	})
}

func TestAccACL_resetOnDestroy(t *testing.T) {
	const resourceName = "tailscale_acl.test_acl"

	const testACLCreate = `
		resource "tailscale_acl" "test_acl" {
            reset_acl_on_destroy = true
		    overwrite_existing_content = true
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
			}
			EOF
		}`

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories(t),
		CheckDestroy: checkResourceDestroyed(resourceName, func(client *tailscale.Client, rs *terraform.ResourceState) error {
			aclAfterDestroy, err := client.PolicyFile().Raw(context.Background())
			if err != nil {
				return err
			}

			// Reset the ACL through the API client
			err = client.PolicyFile().Set(context.Background(), "", "")
			if err != nil {
				return err
			}

			aclAfterReset, err := client.PolicyFile().Raw(context.Background())
			if err != nil {
				return err
			}

			if diff := cmp.Diff(aclAfterDestroy.HuJSON, aclAfterReset.HuJSON); diff != "" {
				return fmt.Errorf("wrong ACL after destroy: (-got+want) \n%s", diff)
			}

			return nil
		}),
		Steps: []resource.TestStep{
			{
				Config: testACLCreate,
				Check: resource.ComposeTestCheckFunc(
					checkResourceRemoteProperties(resourceName,
						checkProperties(&tailscale.ACL{
							ACLs: []tailscale.ACLEntry{
								{
									Action: "accept",
									Users:  []string{"*"},
									Ports:  []string{"*:*"},
								},
							},
						}),
					),
				),
			},
		},
	})
}

func TestAccACLValidation(t *testing.T) {
	const resourceName = "tailscale_acl.test_acl_validation"

	const testACLInvalidSyntax = `
		resource "tailscale_acl" "test_acl" {
			acl = <<EOF
			{
                "grants": [
					{
						"src": ["group:scim-that-does-not-exist-yet"], 
						"dst": ["*"], 
						"ip": ["*"]
					},
    			// ] <- Commented out to create invalid syntax, invalid JSON should fail the plan
			}
			EOF
		}`
	const testACLUndefinedReferences = `
		resource "tailscale_acl" "test_acl" {
			acl = <<EOF
			{
                "grants": [
					{
						"src": ["group:scim-that-does-not-exist-yet"], // <- Undefined reference do not fail plan because they could be created in the same run
						"dst": ["*"], 
						"ip": ["*"]
					},
    			]
			}
			EOF
		}`

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories(t),
		Steps: []resource.TestStep{
			{
				ResourceName: resourceName,
				Config:       testACLInvalidSyntax,
				PlanOnly:     true,
				ExpectError:  regexp.MustCompile("Error: ACL is not a valid HuJSON string"),
			},
			{
				ResourceName:       resourceName,
				Config:             testACLUndefinedReferences,
				PlanOnly:           true,
				ExpectNonEmptyPlan: true,
			},
			{
				ResourceName: resourceName,
				Config:       testACLUndefinedReferences,
				ExpectError:  regexp.MustCompile("Error: Failed to set ACL"), // Apply should still fail if the pre-requisite resource is not created before the ACLs are applied
			},
		},
	})
}

func checkProperties(expected *tailscale.ACL) func(client *tailscale.Client, rs *terraform.ResourceState) error {
	return func(client *tailscale.Client, rs *terraform.ResourceState) error {
		actual, err := client.PolicyFile().Get(context.Background())
		if err != nil {
			return err
		}

		// Clear out ETag before comparing to expected.
		actual.ETag = ""
		if err := assertEqual(expected, actual, "wrong ACL"); err != nil {
			return err
		}

		return nil
	}
}
