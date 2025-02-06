// Copyright (c) David Bond, Tailscale Inc, & Contributors
// SPDX-License-Identifier: MIT

package tailscale

import (
	"context"
	"fmt"
	"os"
	"reflect"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"

	"tailscale.com/client/tailscale/v2"
)

func TestAccTailscaleDeviceTags(t *testing.T) {
	const resourceName = "tailscale_device_tags.test_tags"

	const testDeviceTagsCreate = `
		data "tailscale_device" "test_device" {
			name = "%s"
			wait_for = "60s"
		}
		
		resource "tailscale_device_tags" "test_tags" {
			device_id = data.tailscale_device.test_device.id
			tags = [
				"tag:a",
				"tag:b",
			]
		}`

	const testDeviceTagsUpdate = `
		data "tailscale_device" "test_device" {
			name = "%s"
			wait_for = "60s"
		}

		resource "tailscale_device_tags" "test_tags" {
			device_id = data.tailscale_device.test_device.id
			tags = [
				"tag:b",
				"tag:c",
			]
		}`

	checkProperties := func(expectedTags []string) func(client *tailscale.Client, rs *terraform.ResourceState) error {
		return func(client *tailscale.Client, rs *terraform.ResourceState) error {
			deviceID := rs.Primary.Attributes["device_id"]

			device, err := client.Devices().Get(context.Background(), deviceID)
			if err != nil {
				return fmt.Errorf("failed to fetch device: %s", err)
			}

			if !reflect.DeepEqual(device.Tags, expectedTags) {
				return fmt.Errorf("bad tags: %#v", device.Tags)
			}
			return nil
		}
	}

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories(t),
		Steps: []resource.TestStep{
			{
				PreConfig: func() {
					// Set up ACLs to allow the required tags
					client := testAccProvider.Meta().(*tailscale.Client)
					err := client.PolicyFile().Set(context.Background(), `
					{
					    "tagOwners": {
							"tag:a": ["autogroup:member"],
							"tag:b": ["autogroup:member"],
							"tag:c": ["autogroup:member"],
						},
					}`, "")
					if err != nil {
						panic(err)
					}
				},
				Config: fmt.Sprintf(testDeviceTagsCreate, os.Getenv("TAILSCALE_TEST_DEVICE_NAME")),
				Check: resource.ComposeTestCheckFunc(
					checkResourceRemoteProperties(resourceName, checkProperties([]string{"tag:a", "tag:b"})),
					resource.TestCheckTypeSetElemAttr(resourceName, "tags.*", "tag:a"),
					resource.TestCheckTypeSetElemAttr(resourceName, "tags.*", "tag:b"),
				),
			},
			{
				Config: fmt.Sprintf(testDeviceTagsUpdate, os.Getenv("TAILSCALE_TEST_DEVICE_NAME")),
				Check: resource.ComposeTestCheckFunc(
					checkResourceRemoteProperties(resourceName, checkProperties([]string{"tag:b", "tag:c"})),
					resource.TestCheckTypeSetElemAttr(resourceName, "tags.*", "tag:b"),
					resource.TestCheckTypeSetElemAttr(resourceName, "tags.*", "tag:c"),
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
