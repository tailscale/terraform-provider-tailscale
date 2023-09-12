package tailscale_test

import (
	"net/http"
	"testing"
	"time"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"

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
				ID:      "test",
				Key:     "thisisatestkey",
				Expires: time.Now().Add(time.Hour * 24),
			}
		},
		ProviderFactories: testProviderFactories(t),
		Steps: []resource.TestStep{
			testResourceCreated("tailscale_tailnet_key.example_key", testTailnetKey),
			testResourceDestroyed("tailscale_tailnet_key.example_key", testTailnetKey),
		},
	})
}

func TestProvider_TailscaleTailnetKeyExpired(t *testing.T) {
	resource.Test(t, resource.TestCase{
		IsUnitTest: true,
		PreCheck: func() {
			testServer.ResponseCode = http.StatusOK
			testServer.ResponseBody = tailscale.Key{
				ID:      "test",
				Key:     "thisisatestkey",
				Expires: time.Now().Add(-time.Hour * 24),
			}
		},
		ProviderFactories: testProviderFactories(t),
		Steps: []resource.TestStep{
			testResourceCreated("tailscale_tailnet_key.example_key", testTailnetKey),
			// expect the resource to be re-created immediately
			testResourceCreated("tailscale_tailnet_key.example_key", testTailnetKey),
		},
	})
}
