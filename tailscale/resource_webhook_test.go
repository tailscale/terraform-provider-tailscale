package tailscale_test

import (
	"net/http"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"

	"github.com/tailscale/tailscale-client-go/tailscale"
)

const testWebhook = `
	resource "tailscale_webhook" "test_webhook" {
		endpoint_url = "https://example.com/endpoint"
		provider_type = "slack"
		subscriptions = ["userNeedsApproval", "nodeCreated"]
	}`

func TestProvider_TailscaleWebhook(t *testing.T) {
	resource.Test(t, resource.TestCase{
		IsUnitTest: true,
		PreCheck: func() {
			testServer.ResponseCode = http.StatusOK
			testServer.ResponseBody = tailscale.Webhook{
				EndpointID: "12345",
			}
		},
		ProviderFactories: testProviderFactories(t),
		Steps: []resource.TestStep{
			testResourceCreated("tailscale_webhook.test_webhook", testWebhook),
			testResourceDestroyed("tailscale_webhook.test_webhook", testWebhook),
		},
	})
}
