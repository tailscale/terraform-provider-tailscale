// Copyright (c) David Bond, Tailscale Inc, & Contributors
// SPDX-License-Identifier: MIT

package tailscale

import (
	"context"
	"errors"
	"fmt"
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"

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

	const testDeviceKeyEmpty = `
		data "tailscale_device" "test_device" {
			name = "%s"
		}

		resource "tailscale_device_key" "test_key" {
			device_id = data.tailscale_device.test_device.id
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
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV5ProviderFactories: testAccProviderFactories(t),
		// After delete, device key should revert to its default properties
		// This is probably not how we actually want things to work, but it's the released behavior.
		// See https://github.com/tailscale/terraform-provider-tailscale/issues/401.
		CheckDestroy: checkResourceDestroyed(resourceName, checkProperties(true)),
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
				Config: fmt.Sprintf(testDeviceKeyEmpty, os.Getenv("TAILSCALE_TEST_DEVICE_NAME")),
				Check: resource.ComposeTestCheckFunc(
					checkResourceRemoteProperties(resourceName, checkProperties(false)),
					checkResourceRemoteProperties(resourceName, checkLegacyID),
					resource.TestCheckResourceAttr(resourceName, "key_expiry_disabled", "false"),
				),
			},
			// set it to true again to test what happens on delete
			{
				Config: fmt.Sprintf(testDeviceKeyUpdate, os.Getenv("TAILSCALE_TEST_DEVICE_NAME")),
				Check: resource.ComposeTestCheckFunc(
					checkResourceRemoteProperties(resourceName, checkProperties(true)),
					checkResourceRemoteProperties(resourceName, checkLegacyID),
					resource.TestCheckResourceAttr(resourceName, "key_expiry_disabled", "true"),
				),
			},
			{
				Config:  fmt.Sprintf(testDeviceKeyUpdate, os.Getenv("TAILSCALE_TEST_DEVICE_NAME")),
				Destroy: true,
				Check: resource.ComposeTestCheckFunc(
					checkResourceRemoteProperties(resourceName, checkProperties(true)), // should not be changed by destruction
				),
			},
			// recreate it to test import
			{
				Config: fmt.Sprintf(testDeviceKeyUpdate, os.Getenv("TAILSCALE_TEST_DEVICE_NAME")),
			},
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})

	// Migration test to ensure the resource is unchanged when migrating
	// from the plugin SDK to the plugin framework.
	//
	// See https://developer.hashicorp.com/terraform/plugin/framework/migrating/testing#terraform-data-resource-example
	checkResourceIsUnchangedInPluginFramework(t,
		fmt.Sprintf(testDeviceKeyCreate, os.Getenv("TAILSCALE_TEST_DEVICE_NAME")),
		resource.ComposeTestCheckFunc(
			checkResourceRemoteProperties(resourceName, checkProperties(false)),
			checkResourceRemoteProperties(resourceName, checkLegacyID),
			resource.TestCheckResourceAttr(resourceName, "key_expiry_disabled", "false"),
		))
	checkResourceIsUnchangedInPluginFramework(t,
		fmt.Sprintf(testDeviceKeyUpdate, os.Getenv("TAILSCALE_TEST_DEVICE_NAME")),
		resource.ComposeTestCheckFunc(
			checkResourceRemoteProperties(resourceName, checkProperties(true)),
			checkResourceRemoteProperties(resourceName, checkLegacyID),
			resource.TestCheckResourceAttr(resourceName, "key_expiry_disabled", "true"),
		))
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
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV5ProviderFactories: testAccProviderFactories(t),
		// After delete, device key should revert to its default properties
		// This is probably not how we actually want things to work, but it's the released behavior.
		// See https://github.com/tailscale/terraform-provider-tailscale/issues/401.
		CheckDestroy: checkResourceDestroyed(resourceName, checkProperties(true)),
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

	// Migration test to ensure the resource is unchanged when migrating
	// from the plugin SDK to the plugin framework.
	//
	// See https://developer.hashicorp.com/terraform/plugin/framework/migrating/testing#terraform-data-resource-example
	checkResourceIsUnchangedInPluginFramework(t,
		fmt.Sprintf(testDeviceKeyCreate, os.Getenv("TAILSCALE_TEST_DEVICE_NAME")),
		resource.ComposeTestCheckFunc(
			checkResourceRemoteProperties(resourceName, checkProperties(false)),
			checkResourceRemoteProperties(resourceName, checkNodeID),
			resource.TestCheckResourceAttr(resourceName, "key_expiry_disabled", "false"),
		))
	checkResourceIsUnchangedInPluginFramework(t,
		fmt.Sprintf(testDeviceKeyUpdate, os.Getenv("TAILSCALE_TEST_DEVICE_NAME")),
		resource.ComposeTestCheckFunc(
			checkResourceRemoteProperties(resourceName, checkProperties(true)),
			checkResourceRemoteProperties(resourceName, checkNodeID),
			resource.TestCheckResourceAttr(resourceName, "key_expiry_disabled", "true"),
		))
}
