package tailscale_test

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

const testDeviceAuthorization = `
	data "tailscale_device" "test_device" {
		name = "device.example.com"
	}
	
	resource "tailscale_device_authorization" "test_authorization" {
		device_id = data.tailscale_device.test_device.id,
		authorized = true
	}`

func TestProvider_TailscaleDeviceAuthorization(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testProviderPreCheck(t) },
		ProviderFactories: providerFactories,
		Steps: []resource.TestStep{
			testResourceCreated("tailscale_device_authorization.test_authorization", testDeviceAuthorization),
			testResourceDestroyed("tailscale_device_authorization.test_authorization", testDeviceAuthorization),
		},
	})
}
