package tailscale

import (
	"context"
	"fmt"
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"

	tsclient "github.com/tailscale/tailscale-client-go/v2"
)

func TestAccTailscaleDeviceAuthorization(t *testing.T) {
	const resourceName = "tailscale_device_authorization.test_authorization"

	const testDeviceAuthorization = `
		data "tailscale_device" "test_device" {
			name = "%s"
		}
		
		resource "tailscale_device_authorization" "test_authorization" {
			device_id = data.tailscale_device.test_device.id
			authorized = true
		}`

	checkAuthorized := func(client *tsclient.Client, rs *terraform.ResourceState) error {
		// Check that the device both exists and is still authorized.
		device, err := client.Devices().Get(context.Background(), rs.Primary.ID)
		if err != nil {
			return err
		}

		if device.Authorized != true {
			return fmt.Errorf("device with id %q is not authorized", rs.Primary.ID)
		}

		return nil
	}

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories(t),
		// Devices are not currently deauthorized when this resource is deleted,
		// expect that the device both exists and is still authorized.
		CheckDestroy: checkResourceDestroyed(resourceName, checkAuthorized),
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(testDeviceAuthorization, os.Getenv("TAILSCALE_TEST_DEVICE_NAME")),
				Check: resource.ComposeTestCheckFunc(
					checkResourceRemoteProperties(resourceName, checkAuthorized),
					resource.TestCheckResourceAttr(resourceName, "authorized", "true"),
				),
			},
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}
