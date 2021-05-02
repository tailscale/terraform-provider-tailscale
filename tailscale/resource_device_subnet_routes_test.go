package tailscale_test

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

const testDeviceSubnetRoutes = `
	data "tailscale_device" "test_device" {
		name = "device.example.com"
	}
	
	resource "tailscale_device_subnet_routes" "test_subnet_routes" {
		device_id = tailscale_device.test_device.id,
		routes = [
			"10.0.1.0/24", 
			"1.2.0.0/16", 
			"2.0.0.0/24",
		]
	}`

func TestProvider_TailscaleDeviceSubnetRoutes(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testProviderPreCheck(t) },
		ProviderFactories: providerFactories,
		Steps: []resource.TestStep{
			testResourceCreated("tailscale_device_subnet_routes.test_subnet_routes", testDeviceSubnetRoutes),
			testResourceDestroyed("tailscale_device_subnet_routes.test_subnet_routes", testDeviceSubnetRoutes),
		},
	})
}
