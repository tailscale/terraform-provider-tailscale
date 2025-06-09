// Copyright (c) David Bond, Tailscale Inc, & Contributors
// SPDX-License-Identifier: MIT

package tailscale

import (
	"context"
	"errors"
	"fmt"
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"

	"tailscale.com/client/tailscale/v2"
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

	checkProperties := func(expectExpiryDisabled bool) func(client *tailscale.Client, rs *terraform.ResourceState) error {
		return func(client *tailscale.Client, rs *terraform.ResourceState) error {
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

	checkLegacyID := func(client *tailscale.Client, rs *terraform.ResourceState) error {
		// Check that the device ID and State ID Match
		device, err := client.Devices().Get(context.Background(), rs.Primary.ID)
		if err != nil {
			return err
		}

		if device.ID != rs.Primary.ID {
			return fmt.Errorf("state id %q does not match legacy id %q", rs.Primary.ID, device.NodeID)
		}

		return nil
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
					checkResourceRemoteProperties(resourceName, checkLegacyID),
					resource.TestCheckResourceAttr(resourceName, "key_expiry_disabled", "false"),
				),
			},
			{
				Config: fmt.Sprintf(testDeviceKeyUpdate, os.Getenv("TAILSCALE_TEST_DEVICE_NAME")),
				Check: resource.ComposeTestCheckFunc(
					checkResourceRemoteProperties(resourceName, checkProperties(true)),
					checkResourceRemoteProperties(resourceName, checkLegacyID),
					resource.TestCheckResourceAttr(resourceName, "key_expiry_disabled", "true"),
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
func TestAccTailscaleDeviceKey_UsesNodeID(t *testing.T) {
	const resourceName = "tailscale_device_key.test_key"

	const testDeviceKeyCreate = `
		data "tailscale_device" "test_device" {
			name = "%s"
		}
		
		resource "tailscale_device_key" "test_key" {
			device_id = data.tailscale_device.test_device.node_id
			key_expiry_disabled = false
		}`

	const testDeviceKeyUpdate = `
		data "tailscale_device" "test_device" {
			name = "%s"
		}
		
		resource "tailscale_device_key" "test_key" {
			device_id = data.tailscale_device.test_device.node_id
			key_expiry_disabled = true
		}`

	checkProperties := func(expectExpiryDisabled bool) func(client *tailscale.Client, rs *terraform.ResourceState) error {
		return func(client *tailscale.Client, rs *terraform.ResourceState) error {
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
		// After delete, device key should revert to its default properties
		// This is probably not how we actually want things to work, but it's the released behavior.
		// See https://github.com/tailscale/terraform-provider-tailscale/issues/401.
		CheckDestroy: checkResourceDestroyed(resourceName, checkProperties(false)),
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(testDeviceKeyCreate, os.Getenv("TAILSCALE_TEST_DEVICE_NAME")),
				Check: resource.ComposeTestCheckFunc(
					checkResourceRemoteProperties(resourceName, checkProperties(false)),
					checkResourceRemoteProperties(resourceName, checkNodeID),
					resource.TestCheckResourceAttr(resourceName, "key_expiry_disabled", "false"),
				),
			},
			{
				Config: fmt.Sprintf(testDeviceKeyUpdate, os.Getenv("TAILSCALE_TEST_DEVICE_NAME")),
				Check: resource.ComposeTestCheckFunc(
					checkResourceRemoteProperties(resourceName, checkProperties(true)),
					checkResourceRemoteProperties(resourceName, checkNodeID),
					resource.TestCheckResourceAttr(resourceName, "key_expiry_disabled", "true"),
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
