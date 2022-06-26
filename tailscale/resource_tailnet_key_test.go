package tailscale_test

import (
	"net/http"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"

	"github.com/davidsbond/tailscale-client-go/tailscale"
)

const testTailnetKey = `
	resource "tailscale_tailnet_key" "example_key" {
		reusable = true
		ephemeral = true
		preauthorized = true
		tags = ["tag:server"]
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
