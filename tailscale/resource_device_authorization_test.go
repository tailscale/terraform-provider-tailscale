package tailscale_test

import (
	"context"
	"errors"
	"fmt"
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"

	tsclient "github.com/tailscale/tailscale-client-go/v2"
	"github.com/tailscale/terraform-provider-tailscale/tailscale"
)

var testDeviceAuthorization = fmt.Sprintf(`
	data "tailscale_device" "test_device" {
		name = "%s"
	}
	
	resource "tailscale_device_authorization" "test_authorization" {
		device_id = data.tailscale_device.test_device.id
		authorized = true
	}`, os.Getenv("TAILSCALE_TEST_DEVICE_NAME"))

var expectedDeviceAuthorizationBasic = &tsclient.Device{
	Authorized: true,
}

func TestAccTailscaleDeviceAuthorization_Basic(t *testing.T) {
	device := &tsclient.Device{}

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories(t),
		CheckDestroy:      testAccCheckDeviceAuthorizationDestroyBasic,
		Steps: []resource.TestStep{
			{
				Config: testDeviceAuthorization,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAccountAuthorizationExists("tailscale_device_authorization.test_authorization", device),
					testAccCheckDeviceAuthorizationBasic(device),
					resource.TestCheckResourceAttr("tailscale_device_authorization.test_authorization", "authorized", "true"),
				),
			},
		},
	})
}

func testAccCheckAccountAuthorizationExists(resourceName string, device *tsclient.Device) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[resourceName]
		if !ok {
			return fmt.Errorf("resource not found: %s", resourceName)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("resource has no ID set")
		}

		out, err := getDevice(rs)
		if err != nil {
			return err
		}

		*device = *out
		return nil
	}
}

func testAccCheckDeviceAuthorizationBasic(device *tsclient.Device) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		if err := checkDeviceAuthorization(device, expectedDeviceAuthorizationBasic); err != nil {
			return err
		}

		return nil
	}
}

func testAccCheckDeviceAuthorizationDestroyBasic(s *terraform.State) error {
	return testAccCheckDeviceAuthorizationDestroy(s, expectedDeviceAuthorizationBasic)
}

func testAccCheckDeviceAuthorizationDestroy(s *terraform.State, expected *tsclient.Device) error {
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "tailscale_device_authorization" {
			continue
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("resource has no ID set")
		}

		device, err := getDevice(rs)
		if err != nil {
			return err
		}

		return checkDeviceAuthorization(device, expected)
	}
	return nil
}

func getDevice(rs *terraform.ResourceState) (*tsclient.Device, error) {
	client := testAccProvider.Meta().(*tailscale.Clients).V2

	devices, err := client.Devices().List(context.Background())
	if err != nil {
		return nil, err
	}

	for _, device := range devices {
		if device.ID == rs.Primary.ID {
			return &device, nil
		}
	}

	return nil, errors.New("device not found")
}

func checkDeviceAuthorization(actual *tsclient.Device, expected *tsclient.Device) error {
	if actual.Authorized != expected.Authorized {
		return fmt.Errorf("bad authorization status, expected %v, got %v", expected.Authorized, actual.Authorized)
	}

	return nil
}
