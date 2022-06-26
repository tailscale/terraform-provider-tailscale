package tailscale_test

import (
	"net/http"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"

	"github.com/davidsbond/tailscale-client-go/tailscale"
)

const testDeviceTags = `
	data "tailscale_device" "test_device" {
		name = "device.example.com"
		wait_for = "60s"
	}
	
	resource "tailscale_device_tags" "test_tags" {
		device_id = data.tailscale_device.test_device.id
		tags = [
			"a:b",
			"b:c",
		]
	}`

func TestProvider_TailscaleDeviceTags(t *testing.T) {
	resource.Test(t, resource.TestCase{
		IsUnitTest: true,
		PreCheck: func() {
			testServer.ResponseCode = http.StatusOK
			testServer.ResponseBody = map[string][]tailscale.Device{
				"devices": {
					{
						Name: "device.example.com",
						ID:   "123",
					},
				},
			}
		},
		ProviderFactories: testProviderFactories(t),
		Steps: []resource.TestStep{
			testResourceCreated("tailscale_device_tags.test_tags", testDeviceTags),
			testResourceDestroyed("tailscale_device_tags.test_tags", testDeviceTags),
		},
	})
}
