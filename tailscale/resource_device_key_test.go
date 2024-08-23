package tailscale

import (
	"context"
	"errors"
	"fmt"
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"

	tsclient "github.com/tailscale/tailscale-client-go/v2"
)

func TestAccTailscaleDeviceKey(t *testing.T) {
	const resourceName = "tailscale_device_key.test_key"

	const testDeviceKeyCreate = `
		data "tailscale_device" "test_device" {
			name = "%s"
		}
		
		resource "tailscale_device_key" "test_key" {
			device_id = data.tailscale_device.test_device.id
			key_expiry_disabled = false
		}`

	const testDeviceKeyUpdate = `
		data "tailscale_device" "test_device" {
			name = "%s"
		}
		
		resource "tailscale_device_key" "test_key" {
			device_id = data.tailscale_device.test_device.id
			key_expiry_disabled = true
		}`

	checkProperties := func(expectExpiryDisabled bool) func(client *tsclient.Client, rs *terraform.ResourceState) error {
		return func(client *tsclient.Client, rs *terraform.ResourceState) error {
			device, err := client.Devices().Get(context.Background(), rs.Primary.ID)
			if err != nil {
				return err
			}

			if expectExpiryDisabled && !device.KeyExpiryDisabled {
				return errors.New("key expiry should be disabled")
			} else if !expectExpiryDisabled && device.KeyExpiryDisabled {
				return errors.New("key expiry should not be disabled")
			}

			return nil
		}
	}

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories(t),
		// After delete, device key should revert to its default properties
		// This is probably not how we actually want things to work, but it's the released behavior.
		// See https://github.com/tailscale/terraform-provider-tailscale/issues/401.
		CheckDestroy: checkResourceDestroyed(resourceName, checkProperties(false)),
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(testDeviceKeyCreate, os.Getenv("TAILSCALE_TEST_DEVICE_NAME")),
				Check: resource.ComposeTestCheckFunc(
					checkResourceRemoteProperties(resourceName, checkProperties(false)),
					resource.TestCheckResourceAttr(resourceName, "key_expiry_disabled", "false"),
				),
			},
			{
				Config: fmt.Sprintf(testDeviceKeyUpdate, os.Getenv("TAILSCALE_TEST_DEVICE_NAME")),
				Check: resource.ComposeTestCheckFunc(
					checkResourceRemoteProperties(resourceName, checkProperties(true)),
					resource.TestCheckResourceAttr(resourceName, "key_expiry_disabled", "true"),
				),
			},
		},
	})
}
