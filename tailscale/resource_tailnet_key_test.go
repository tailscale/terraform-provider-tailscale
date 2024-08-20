package tailscale_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"

	tsclient "github.com/tailscale/tailscale-client-go/v2"
	"github.com/tailscale/terraform-provider-tailscale/tailscale"
)

const testTailnetKey = `
	resource "tailscale_tailnet_key" "example_key" {
		reusable = true
		ephemeral = true
		preauthorized = true
		tags = ["tag:server"]
		expiry = 3600
		description = "Example key"
	}
`

func TestProvider_TailscaleTailnetKey(t *testing.T) {
	resource.Test(t, resource.TestCase{
		IsUnitTest: true,
		PreCheck: func() {
			testServer.ResponseCode = http.StatusOK
			testServer.ResponseBody = tsclient.Key{
				ID:  "test",
				Key: "thisisatestkey",
			}
		},
		ProviderFactories: testProviderFactories(t),
		Steps: []resource.TestStep{
			testResourceCreated("tailscale_tailnet_key.example_key", testTailnetKey),
			testResourceDestroyed("tailscale_tailnet_key.example_key", testTailnetKey),
		},
	})
}

func testTailnetKeyStruct(reusable bool) tsclient.Key {
	var keyCapabilities tsclient.KeyCapabilities
	json.Unmarshal([]byte(`
		{
			"devices": {
				"create": {
					"ephemeral": true,
					"preauthorized": true,
					"tags": [
						"tag:server"
					]
				}
			}
		}`), &keyCapabilities)
	keyCapabilities.Devices.Create.Reusable = reusable
	return tsclient.Key{
		ID:           "test",
		Key:          "thisisatestkey",
		Description:  "Example key",
		Capabilities: keyCapabilities,
	}
}

func setKeyStep(reusable bool, recreateIfInvalid string) resource.TestStep {
	return resource.TestStep{
		PreConfig: func() {
			testServer.ResponseBody = testTailnetKeyStruct(reusable)
		},
		ResourceName: "tailscale_tailnet_key.example_key",
		Config: fmt.Sprintf(`
			resource "tailscale_tailnet_key" "example_key" {
				reusable = %v
				recreate_if_invalid = "%s"
				ephemeral = true
				preauthorized = true
				tags = ["tag:server"]
				expiry = 3600
				description = "Example key"
			}
		`, reusable, recreateIfInvalid),
		Check: func(s *terraform.State) error {
			rs, ok := s.RootModule().Resources["tailscale_tailnet_key.example_key"]

			if !ok {
				return errors.New("key not found")
			}

			if rs.Primary.ID == "" {
				return errors.New("no ID set")
			}

			// Make sure the next API call to the test server returns the key
			// matching the one we have just set.
			testServer.ResponseBody = testTailnetKeyStruct(reusable)

			return nil
		},
	}
}

func checkInvalidKeyRecreated(reusable, wantRecreated bool) resource.TestStep {
	return resource.TestStep{
		RefreshState:       true,
		ExpectNonEmptyPlan: true,
		PreConfig: func() {
			testServer.ResponseCode = http.StatusOK
			key := testTailnetKeyStruct(reusable)
			key.Invalid = true
			testServer.ResponseBody = key
		},
		Check: func(s *terraform.State) error {
			_, ok := s.RootModule().Resources["tailscale_tailnet_key.example_key"]

			if ok == wantRecreated {
				return fmt.Errorf("found=%v, wantRecreated=%v", ok, wantRecreated)
			}

			return nil
		},
	}
}
func TestProvider_TailscaleTailnetKeyInvalid(t *testing.T) {
	resource.Test(t, resource.TestCase{
		IsUnitTest: true,
		PreCheck: func() {
			testServer.ResponseCode = http.StatusOK
			testServer.ResponseBody = tsclient.Key{
				ID:  "test",
				Key: "thisisatestkey",
			}
		},
		ProviderFactories: testProviderFactories(t),
		Steps: []resource.TestStep{
			// Create a reusable key.
			setKeyStep(true, ""),
			// Confirm that the reusable key will be recreated when invalid.
			checkInvalidKeyRecreated(true, true),

			// Now make it a single-use key.
			setKeyStep(false, ""),
			// Confirm that the single-use key is not recreated.
			checkInvalidKeyRecreated(false, false),

			// A single-use key with recreate=always, should be recreated.
			setKeyStep(false, "always"),
			checkInvalidKeyRecreated(false, true),

			// A single-use key with recreate=never, should not be recreated.
			setKeyStep(false, "never"),
			checkInvalidKeyRecreated(false, false),

			// A reusable key with recreate=always, should be recreated.
			setKeyStep(true, "always"),
			checkInvalidKeyRecreated(true, true),

			// A reusable key with recreate=always, should be recreated.
			setKeyStep(true, "always"),
			checkInvalidKeyRecreated(true, true),
		},
	})
}

func TestAccTailscaleTailnetKey(t *testing.T) {
	const resourceName = "tailscale_tailnet_key.test_key"

	const testTailnetKeyCreate = `
		resource "tailscale_tailnet_key" "test_key" {
			reusable = true
			ephemeral = true
			preauthorized = true
			tags = ["tag:a"]
			expiry = 3600
			description = "Test key"
		}`

	const testTailnetKeyUpdate = `
		resource "tailscale_tailnet_key" "test_key" {
			reusable = false
			ephemeral = false
			preauthorized = false
			tags = ["tag:b"]
			expiry = 7200
			description = "Test key changed"
		}`

	checkProperties := func(expected *tsclient.Key, expectedExpirySeconds float64) func(client *tsclient.Client, rs *terraform.ResourceState) error {
		return func(client *tsclient.Client, rs *terraform.ResourceState) error {
			actual, err := client.Keys().Get(context.Background(), rs.Primary.ID)
			if err != nil {
				return err
			}

			if actual.Created.IsZero() {
				return errors.New("created should be set")
			}
			if actual.Expires.Sub(actual.Created).Seconds() != expectedExpirySeconds {
				return fmt.Errorf("wrong expires, want %s, got %s", actual.Created.Add(time.Duration(expectedExpirySeconds)*time.Second), actual.Expires)
			}

			// don't compare times
			actual.Created = time.Time{}
			actual.Expires = time.Time{}

			// don't compare IDs
			actual.ID = ""

			if diff := cmp.Diff(expected, actual); diff != "" {
				return fmt.Errorf("diff found (-got, +want): %s", diff)
			}

			return nil
		}
	}

	var expectedKey tsclient.Key
	expectedKey.Description = "Test key"
	expectedKey.Capabilities.Devices.Create.Reusable = true
	expectedKey.Capabilities.Devices.Create.Ephemeral = true
	expectedKey.Capabilities.Devices.Create.Preauthorized = true
	expectedKey.Capabilities.Devices.Create.Tags = []string{"tag:a"}

	var expectedKeyUpdated tsclient.Key
	expectedKeyUpdated.Description = "Test key changed"
	expectedKeyUpdated.Capabilities.Devices.Create.Reusable = false
	expectedKeyUpdated.Capabilities.Devices.Create.Ephemeral = false
	expectedKeyUpdated.Capabilities.Devices.Create.Preauthorized = false
	expectedKeyUpdated.Capabilities.Devices.Create.Tags = []string{"tag:b"}

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories(t),
		Steps: []resource.TestStep{
			{
				PreConfig: func() {
					// Set up ACLs to allow the required tags
					client := testAccProvider.Meta().(*tailscale.Clients).V2
					err := client.PolicyFile().Set(context.Background(), `
					{
					    "tagOwners": {
							"tag:a": ["autogroup:member"],
							"tag:b": ["autogroup:member"],
						},
					}`, "")
					if err != nil {
						panic(err)
					}
				},
				Config: testTailnetKeyCreate,
				Check: resource.ComposeTestCheckFunc(
					checkResourceRemoteProperties(resourceName,
						checkProperties(&expectedKey, 3600),
					),
					resource.TestCheckResourceAttr(resourceName, "reusable", "true"),
					resource.TestCheckResourceAttr(resourceName, "ephemeral", "true"),
					resource.TestCheckResourceAttr(resourceName, "preauthorized", "true"),
					resource.TestCheckTypeSetElemAttr(resourceName, "tags.*", "tag:a"),
					resource.TestCheckResourceAttr(resourceName, "expiry", "3600"),
					resource.TestCheckResourceAttr(resourceName, "description", "Test key"),
				),
			},
			{
				Config: testTailnetKeyUpdate,
				Check: resource.ComposeTestCheckFunc(
					checkResourceRemoteProperties(resourceName,
						checkProperties(&expectedKeyUpdated, 7200),
					),
					resource.TestCheckResourceAttr(resourceName, "reusable", "false"),
					resource.TestCheckResourceAttr(resourceName, "ephemeral", "false"),
					resource.TestCheckResourceAttr(resourceName, "preauthorized", "false"),
					resource.TestCheckTypeSetElemAttr(resourceName, "tags.*", "tag:b"),
					resource.TestCheckResourceAttr(resourceName, "expiry", "7200"),
					resource.TestCheckResourceAttr(resourceName, "description", "Test key changed"),
				),
			},
		},
	})
}
