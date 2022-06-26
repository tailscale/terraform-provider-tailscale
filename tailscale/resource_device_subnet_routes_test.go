package tailscale_test

import (
	"net/http"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"

	"github.com/davidsbond/tailscale-client-go/tailscale"
)

const testDeviceSubnetRoutes = `
	data "tailscale_device" "test_device" {
		name = "device.example.com"
	}
	
	resource "tailscale_device_subnet_routes" "test_subnet_routes" {
		device_id = data.tailscale_device.test_device.id
		routes = [
			"10.0.1.0/24", 
			"1.2.0.0/16", 
			"2.0.0.0/24",
		]
	}`

func TestProvider_TailscaleDeviceSubnetRoutes(t *testing.T) {
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
			testResourceCreated("tailscale_device_subnet_routes.test_subnet_routes", testDeviceSubnetRoutes),
			testResourceDestroyed("tailscale_device_subnet_routes.test_subnet_routes", testDeviceSubnetRoutes),
		},
	})
}
