package tailscale_test

import (
	"encoding/json"
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
			testResourceCreated("tailscale_tailnet_key.example_key", testTailnetKey),
			{
				// expect Invalid tailnet key to be re-created
				RefreshState:       true,
				ExpectNonEmptyPlan: true,
				PreConfig: func() {
					var keyCapabilities tailscale.KeyCapabilities
					json.Unmarshal([]byte(`
					{
						"devices": {
							"create": {
								"reusable": true,
								"ephemeral": true,
								"preauthorized": true,
								"tags": [
									"tag:server"
								]
							}
						}
					}`), &keyCapabilities)

					testServer.ResponseCode = http.StatusOK
					testServer.ResponseBody = tailscale.Key{
						ID:           "test",
						Key:          "thisisatestkey",
						Description:  "Example key",
						Capabilities: keyCapabilities,
						Invalid:      true, // causes replacement
					}
				},
				Check: func(s *terraform.State) error {
					_, ok := s.RootModule().Resources["tailscale_tailnet_key.example_key"]

					// an Invalid tailnet key will have be removed from terraform state during the Read operation
					if ok {
						// fail here if the resource still exists in state
						return fmt.Errorf("found: %s", "tailscale_tailnet_key.example_key")
					}

					return nil
				},
			},
		},
	})
}
