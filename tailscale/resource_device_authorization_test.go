// Copyright (c) David Bond, Tailscale Inc, & Contributors
// SPDX-License-Identifier: MIT

package tailscale

import (
	"context"
	"fmt"
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"

	"tailscale.com/client/tailscale/v2"
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

	checkAuthorized := func(client *tailscale.Client, rs *terraform.ResourceState) error {
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

	checkLegacyID := func(client *tailscale.Client, rs *terraform.ResourceState) error {
		// Check that the device ID and State ID Match
		device, err := client.Devices().Get(context.Background(), rs.Primary.ID)
		if err != nil {
			return err
		}

		if device.ID != rs.Primary.ID {
			return fmt.Errorf("state id %q does not match legacy id %q", rs.Primary.ID, device.ID)
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
					checkResourceRemoteProperties(resourceName, checkLegacyID),
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

func TestAccTailscaleDeviceAuthorization_UsesNodeID(t *testing.T) {
	const resourceName = "tailscale_device_authorization.test_authorization"

	const testDeviceAuthorization = `
		data "tailscale_device" "test_device" {
			name = "%s"
		}
		
		resource "tailscale_device_authorization" "test_authorization" {
			device_id = data.tailscale_device.test_device.node_id
			authorized = true
		}`

	checkAuthorized := func(client *tailscale.Client, rs *terraform.ResourceState) error {
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

	checkNodeID := func(client *tailscale.Client, rs *terraform.ResourceState) error {
		// Check that the device ID and State ID Match
		device, err := client.Devices().Get(context.Background(), rs.Primary.ID)
		if err != nil {
			return err
		}

		if device.NodeID != rs.Primary.ID {
			return fmt.Errorf("state id %q does not match node id %q", rs.Primary.ID, device.NodeID)
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
					checkResourceRemoteProperties(resourceName, checkNodeID),
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
