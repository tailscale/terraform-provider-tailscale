// Copyright (c) David Bond, Tailscale Inc, & Contributors
// SPDX-License-Identifier: MIT

package tailscale

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"regexp"
	"testing"
	"time"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
	"tailscale.com/client/tailscale/v2"
)

func TestProvider_TailscaleFederatedIdentity(t *testing.T) {
	const testFederatedIdentity = `
	resource "tailscale_federated_identity" "example_federated_identity" {
		description = "Example federated identity"
		scopes      = ["auth_keys", "devices:core"]
		tags        = ["tag:test"]
        issuer      = "https://example.com"
        subject     = "example-sub-*"
        custom_claim_rules = {
            repo_name = "example-repo-name"
        }
	}`

	resource.Test(t, resource.TestCase{
		IsUnitTest: true,
		PreCheck: func() {
			testServer.ResponseCode = http.StatusOK
			testServer.ResponseBody = tailscale.Key{
				ID:          "test",
				Description: "Example federated identity",
				Scopes:      []string{"auth_keys", "devices:core"},
				Tags:        []string{"tag:test"},
				Issuer:      "https://example.com",
				Subject:     "example-sub-*",
				CustomClaimRules: map[string]string{
					"repo_name": "example-repo-name",
				},
			}
		},
		ProtoV5ProviderFactories: testProviderFactories(t),
		Steps: []resource.TestStep{
			{
				Config: testFederatedIdentity,
				Check: func(s *terraform.State) error {
					rs, ok := s.RootModule().Resources["tailscale_federated_identity.example_federated_identity"]
					if !ok {
						return fmt.Errorf("not found: tailscale_federated_identity.example_federated_identity")
					}
					if rs.Primary.ID == "" {
						return errors.New("no ID set")
					}
					return nil
				},
			},
			testResourceDestroyed("tailscale_federated_identity.example_federated_identity", testFederatedIdentity),
		},
	})
}

func TestProvider_TailscaleFederatedIdentity_ReservedCustomClaimKeys(t *testing.T) {
	for _, reservedKey := range []string{"sub", "iss"} {
		t.Run(reservedKey, func(t *testing.T) {
			config := fmt.Sprintf(`
			resource "tailscale_federated_identity" "test" {
				scopes  = ["auth_keys"]
				issuer  = "https://example.com"
				subject = "example-sub-*"
				custom_claim_rules = {
					%s = "some-value"
				}
			}`, reservedKey)

			resource.Test(t, resource.TestCase{
				IsUnitTest:               true,
				ProtoV5ProviderFactories: testProviderFactories(t),
				Steps: []resource.TestStep{
					{
						Config:      config,
						ExpectError: regexp.MustCompile(`Reserved claim key`),
					},
				},
			})
		})
	}
}

func TestAccTailscaleFederatedIdentity(t *testing.T) {
	const resourceName = "tailscale_federated_identity.test_federated_identity"

	const testFederatedIdentityCreate = `
		resource "tailscale_federated_identity" "test_federated_identity" {
			description = "Example federated identity"
			scopes      = ["auth_keys", "devices:core"]
			tags        = ["tag:test"]
			issuer      = "https://example.com"
			subject     = "example-sub-*"
			custom_claim_rules = {
				repo_name = "example-repo-name"
			}
		}`

	const testFederatedIdentityUpdate = `
		resource "tailscale_federated_identity" "test_federated_identity" {
			description = "Federated identity"
			scopes      = ["auth_keys:read", "devices:core"]
			tags        = ["tag:test"]
			issuer      = "https://example.com"
			subject     = "example-sub-*-other"
		}`

	var expectedFederatedIdentityCreated tailscale.Key
	expectedFederatedIdentityCreated.Description = "Example federated identity"
	expectedFederatedIdentityCreated.KeyType = "federated"
	expectedFederatedIdentityCreated.Scopes = []string{"auth_keys", "devices:core"}
	expectedFederatedIdentityCreated.Tags = []string{"tag:test"}
	expectedFederatedIdentityCreated.Issuer = "https://example.com"
	expectedFederatedIdentityCreated.Subject = "example-sub-*"
	expectedFederatedIdentityCreated.CustomClaimRules = map[string]string{
		"repo_name": "example-repo-name",
	}

	var expectedFederatedIdentityUpdated tailscale.Key
	expectedFederatedIdentityUpdated.Description = "Federated identity"
	expectedFederatedIdentityUpdated.KeyType = "federated"
	expectedFederatedIdentityUpdated.Scopes = []string{"auth_keys:read", "devices:core"}
	expectedFederatedIdentityUpdated.Tags = []string{"tag:test"}
	expectedFederatedIdentityUpdated.Issuer = "https://example.com"
	expectedFederatedIdentityUpdated.Subject = "example-sub-*-other"
	expectedFederatedIdentityUpdated.CustomClaimRules = map[string]string{}

	checkProperties := func(expected *tailscale.Key) func(client *tailscale.Client, rs *terraform.ResourceState) error {
		return func(client *tailscale.Client, rs *terraform.ResourceState) error {
			actual, err := client.Keys().Get(context.Background(), rs.Primary.ID)
			if err != nil {
				return err
			}

			if actual.Created.IsZero() {
				return errors.New("created should be set")
			}

			// don't compare server-side generated fields
			actual.Created = time.Time{}
			actual.Updated = time.Time{}
			actual.Audience = ""
			actual.ID = ""
			actual.UserID = ""

			if err := assertEqual(expected, actual, "wrong key"); err != nil {
				return err
			}

			return nil
		}
	}

	checkFederatedIdentityDeleted := func(client *tailscale.Client, rs *terraform.ResourceState) error {
		key, err := client.Keys().Get(context.Background(), rs.Primary.ID)
		if err != nil {
			return fmt.Errorf("unexpected error while checking if federated identity was deleted: %w", err)
		}

		if !key.Invalid {
			return fmt.Errorf("federated identity is still valid on server")
		}
		if key.Revoked.IsZero() {
			return fmt.Errorf("federated identity was not revoked on server")
		}

		return nil
	}

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV5ProviderFactories: testAccProviderFactories(t),
		CheckDestroy:             checkResourceDestroyed(resourceName, checkFederatedIdentityDeleted),
		Steps: []resource.TestStep{
			{
				PreConfig: func() {
					// Set up ACLs to allow the required tags
					client := testAccProvider.Meta().(*tailscale.Client)
					err := client.PolicyFile().Set(context.Background(), `
					{
					    "tagOwners": {
							"tag:test": ["autogroup:member"],
						},
					}`, "")
					if err != nil {
						panic(err)
					}
				},
				Config: testFederatedIdentityCreate,
				Check: resource.ComposeTestCheckFunc(
					checkResourceRemoteProperties(resourceName,
						checkProperties(&expectedFederatedIdentityCreated),
					),
					resource.TestCheckResourceAttr(resourceName, "description", "Example federated identity"),
					resource.TestCheckResourceAttr(resourceName, "scopes.0", "auth_keys"),
					resource.TestCheckResourceAttr(resourceName, "scopes.1", "devices:core"),
					resource.TestCheckResourceAttr(resourceName, "tags.0", "tag:test"),
					resource.TestCheckResourceAttr(resourceName, "issuer", "https://example.com"),
					resource.TestCheckResourceAttr(resourceName, "subject", "example-sub-*"),
					resource.TestCheckResourceAttr(resourceName, "custom_claim_rules.repo_name", "example-repo-name"),
					resource.TestCheckResourceAttrSet(resourceName, "id"),
					resource.TestCheckResourceAttrSet(resourceName, "created_at"),
					resource.TestCheckResourceAttrSet(resourceName, "user_id"),
				),
			},
			{
				Config: testFederatedIdentityUpdate,
				Check: resource.ComposeTestCheckFunc(
					checkResourceRemoteProperties(resourceName,
						checkProperties(&expectedFederatedIdentityUpdated),
					),
					resource.TestCheckResourceAttr(resourceName, "description", "Federated identity"),
					resource.TestCheckResourceAttr(resourceName, "scopes.0", "auth_keys:read"),
					resource.TestCheckResourceAttr(resourceName, "scopes.1", "devices:core"),
					resource.TestCheckResourceAttr(resourceName, "tags.0", "tag:test"),
					resource.TestCheckResourceAttr(resourceName, "issuer", "https://example.com"),
					resource.TestCheckResourceAttr(resourceName, "subject", "example-sub-*-other"),
					resource.TestCheckResourceAttrSet(resourceName, "id"),
					resource.TestCheckResourceAttrSet(resourceName, "created_at"),
					resource.TestCheckResourceAttrSet(resourceName, "updated_at"),
					resource.TestCheckResourceAttrSet(resourceName, "user_id"),
				),
			},
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

// Migration test to ensure the resource is unchanged when migrating
// from the plugin SDK to the plugin framework.
//
// See https://developer.hashicorp.com/terraform/plugin/framework/migrating/testing#terraform-data-resource-example
func TestAccTailscaleFederatedIdentity_UpgradeToPluginFramework(t *testing.T) {
	const resourceName = "tailscale_federated_identity.test_federated_identity"

	const testFederatedIdentity = `
		resource "tailscale_federated_identity" "test_federated_identity" {
			description = "Example federated identity"
			scopes      = ["devices:core:read"]
			issuer      = "https://example.com"
			subject     = "example-sub-*"
			custom_claim_rules = {
				repo_name = "example-repo-name"
			}
		}`

	testFederatedIdentityCheck := resource.ComposeTestCheckFunc(
		resource.TestCheckResourceAttr(resourceName, "description", "Example federated identity"),
		resource.TestCheckResourceAttr(resourceName, "scopes.#", "1"),
		resource.TestCheckResourceAttr(resourceName, "issuer", "https://example.com"),
		resource.TestCheckResourceAttr(resourceName, "subject", "example-sub-*"),
		resource.TestCheckResourceAttr(resourceName, "custom_claim_rules.repo_name", "example-repo-name"),
		resource.TestCheckResourceAttrSet(resourceName, "id"),
		resource.TestCheckResourceAttrSet(resourceName, "created_at"),
		resource.TestCheckResourceAttrSet(resourceName, "user_id"),
	)

	checkResourceIsUnchangedInPluginFramework(t, testFederatedIdentity, testFederatedIdentityCheck)
}
