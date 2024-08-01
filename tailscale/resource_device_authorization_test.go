package tailscale_test

import (
	"context"
	"fmt"
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"

	"github.com/tailscale/tailscale-client-go/tailscale"
)

const testDeviceAuthorization = `
	data "tailscale_device" "test_device" {
		name = "%s"
	}
	
	resource "tailscale_device_authorization" "test_authorization" {
		device_id = data.tailscale_device.test_device.id
		authorized = true
	}`

func TestAccTailscaleDeviceAuthorization_Basic(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories(t),
		CheckDestroy:      testAccCheckDeviceAuthorizationDestroy,
		Steps: []resource.TestStep{
			{
				Config: generateBasicConfig(),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckDeviceAuthorized("tailscale_device_authorization.test_authorization"),
					resource.TestCheckResourceAttr("tailscale_device_authorization.test_authorization", "authorized", "true"),
				),
			},
			{
				ResourceName:      "tailscale_device_authorization.test_authorization",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func generateBasicConfig() string {
	return fmt.Sprintf(testDeviceAuthorization, os.Getenv("TAILSCALE_TEST_DEVICE_NAME"))
}

func testAccCheckDeviceAuthorized(resourceName string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[resourceName]
		if !ok {
			return fmt.Errorf("resource not found: %s", resourceName)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("resource has no ID set")
		}

		client := testAccProvider.Meta().(*tailscale.Client)
		// Devices are not currently deauthorized when this resource is deleted,
		// expect that the device both exists and is still authorized.
		devices, err := client.Devices(context.Background())
		if err != nil {
			return err
		}

		var selected *tailscale.Device
		for _, device := range devices {
			if device.ID == rs.Primary.ID {
				selected = &device
				break
			}
		}

		if selected == nil {
			return fmt.Errorf("expected device with id %q to exist", rs.Primary.ID)
		}

		if selected.Authorized != true {
			return fmt.Errorf("device with id %q is not authorized", rs.Primary.ID)
		}

		return nil
	}
}

func testAccCheckDeviceAuthorizationDestroy(s *terraform.State) error {
	client := testAccProvider.Meta().(*tailscale.Client)

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "tailscale_device_authorization" {
			continue
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("resource has no ID set")
		}

		// Devices are not currently deauthorized when this resource is deleted,
		// expect that the device both exists and is still authorized.
		devices, err := client.Devices(context.Background())
		if err != nil {
			return err
		}

		var selected *tailscale.Device
		for _, device := range devices {
			if device.ID == rs.Primary.ID {
				selected = &device
				break
			}
		}

		if selected == nil {
			return fmt.Errorf("expected device with id %q to exist", rs.Primary.ID)
		}

		if selected.Authorized != true {
			return fmt.Errorf("device with id %q is not authorized", rs.Primary.ID)
		}
	}

	return nil
}
