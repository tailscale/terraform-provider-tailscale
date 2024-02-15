package tailscale_test

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"

	"github.com/tailscale/tailscale-client-go/tailscale"
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
			testServer.ResponseBody = tailscale.Key{
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

func testTailnetKeyStruct(reusable bool) tailscale.Key {
	var keyCapabilities tailscale.KeyCapabilities
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
	return tailscale.Key{
		ID:           "test",
		Key:          "thisisatestkey",
		Description:  "Example key",
		Capabilities: keyCapabilities,
	}
}

func setKeyStep(reusable bool, recreateIfInvalid string) resource.TestStep {
	return resource.TestStep{
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
			testServer.ResponseBody = tailscale.Key{
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
