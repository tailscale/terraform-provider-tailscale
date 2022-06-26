package tailscale_test

import (
	"net/http"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"

	"github.com/davidsbond/tailscale-client-go/tailscale"
)

const testDeviceAuthorization = `
	data "tailscale_device" "test_device" {
		name = "device.example.com"
	}
	
	resource "tailscale_device_authorization" "test_authorization" {
		device_id = data.tailscale_device.test_device.id
		authorized = true
	}`

func TestProvider_TailscaleDeviceAuthorization(t *testing.T) {
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
			testResourceCreated("tailscale_device_authorization.test_authorization", testDeviceAuthorization),
			testResourceDestroyed("tailscale_device_authorization.test_authorization", testDeviceAuthorization),
		},
	})
}
