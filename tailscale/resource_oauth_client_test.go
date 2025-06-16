// Copyright (c) David Bond, Tailscale Inc, & Contributors
// SPDX-License-Identifier: MIT

package tailscale

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
	"tailscale.com/client/tailscale/v2"
)

func TestProvider_TailscaleOAuthClient(t *testing.T) {
	const testOAuthClient = `
	resource "tailscale_oauth_client" "example_oauth_client" {
		description = "Example OAuth client"
		scopes      = ["auth_keys", "devices:core"]
		tags        = ["tag:test"]
	}`

	resource.Test(t, resource.TestCase{
		IsUnitTest: true,
		PreCheck: func() {
			testServer.ResponseCode = http.StatusOK
			testServer.ResponseBody = tailscale.Key{
				ID:  "test",
				Key: "thisisatestclient",
			}
		},
		ProviderFactories: testProviderFactories(t),
		Steps: []resource.TestStep{
			testResourceCreated("tailscale_oauth_client.example_oauth_client", testOAuthClient),
			testResourceDestroyed("tailscale_oauth_client.example_oauth_client", testOAuthClient),
		},
	})
}

func TestAccTailscaleOAuthClient(t *testing.T) {
	const resourceName = "tailscale_oauth_client.test_client"

	const testOAuthClientCreate = `
		resource "tailscale_oauth_client" "test_client" {
			description = "Test client"
			scopes      = ["auth_keys", "devices:core"]
			tags        = ["tag:test"]
		}`

	const testOAuthClientUpdate = `
		resource "tailscale_oauth_client" "test_client" {
			description = "Updated description"
			scopes      = ["auth_keys:read"]
		}`

	var expectedOAuthClientCreated tailscale.Key
	expectedOAuthClientCreated.Description = "Test client"
	expectedOAuthClientCreated.KeyType = "client"
	expectedOAuthClientCreated.Scopes = []string{"auth_keys", "devices:core"}
	expectedOAuthClientCreated.Tags = []string{"tag:test"}

	var expectedOAuthClientUpdated tailscale.Key
	expectedOAuthClientUpdated.Description = "Updated description"
	expectedOAuthClientUpdated.KeyType = "client"
	expectedOAuthClientUpdated.Scopes = []string{"auth_keys:read"}
	expectedOAuthClientUpdated.Tags = nil

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
			actual.ID = ""
			actual.UserID = ""

			if err := assertEqual(expected, actual, "wrong key"); err != nil {
				return err
			}

			return nil
		}
	}

	checkOAuthClientDeleted := func(client *tailscale.Client, rs *terraform.ResourceState) error {
		key, err := client.Keys().Get(context.Background(), rs.Primary.ID)
		if err != nil {
			return fmt.Errorf("unexpected error while checking if oauth client was deleted: %w", err)
		}

		if !key.Invalid {
			return fmt.Errorf("oauth client is still valid on server")
		}
		if key.Revoked.IsZero() {
			return fmt.Errorf("oauth client was not revoked on server")
		}

		return nil
	}

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories(t),
		CheckDestroy:      checkResourceDestroyed(resourceName, checkOAuthClientDeleted),
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
				Config: testOAuthClientCreate,
				Check: resource.ComposeTestCheckFunc(
					checkResourceRemoteProperties(resourceName,
						checkProperties(&expectedOAuthClientCreated),
					),
					resource.TestCheckResourceAttr(resourceName, "description", "Test client"),
					resource.TestCheckResourceAttr(resourceName, "scopes.0", "auth_keys"),
					resource.TestCheckResourceAttr(resourceName, "scopes.1", "devices:core"),
					resource.TestCheckResourceAttr(resourceName, "tags.0", "tag:test"),
					resource.TestCheckResourceAttrSet(resourceName, "id"),
					resource.TestCheckResourceAttrSet(resourceName, "key"),
					resource.TestCheckResourceAttrSet(resourceName, "created_at"),
					resource.TestCheckResourceAttrSet(resourceName, "user_id"),
				),
			},
			{
				Config: testOAuthClientUpdate,
				Check: resource.ComposeTestCheckFunc(
					checkResourceRemoteProperties(resourceName,
						checkProperties(&expectedOAuthClientUpdated),
					),
					resource.TestCheckResourceAttr(resourceName, "description", "Updated description"),
					resource.TestCheckResourceAttr(resourceName, "scopes.0", "auth_keys:read"),
					resource.TestCheckResourceAttrSet(resourceName, "id"),
					resource.TestCheckResourceAttrSet(resourceName, "key"),
					resource.TestCheckResourceAttrSet(resourceName, "created_at"),
					resource.TestCheckResourceAttrSet(resourceName, "user_id"),
				),
			},
			{
				ResourceName:            resourceName,
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"key"}, // sensitive material not returned by the API
			},
		},
	})
}
