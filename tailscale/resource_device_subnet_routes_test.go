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

	tsclient "github.com/tailscale/tailscale-client-go/v2"
)

func TestAccTailscaleDeviceSubnetRoutes(t *testing.T) {
	const resourceName = "tailscale_device_subnet_routes.test_subnet_routes"

	const testDeviceSubnetRoutesCreate = `
		data "tailscale_device" "test_device" {
			name = "%s"
		}
		
		resource "tailscale_device_subnet_routes" "test_subnet_routes" {
			device_id = data.tailscale_device.test_device.id
			routes = [
				"10.0.1.0/24", 
				"2.0.0.0/24",
			]
		}`

	const testDeviceSubnetRoutesUpdate = `
		data "tailscale_device" "test_device" {
			name = "%s"
		}
		
		resource "tailscale_device_subnet_routes" "test_subnet_routes" {
			device_id = data.tailscale_device.test_device.id
			routes = [
				"10.0.1.0/24", 
				"1.2.0.0/16", 
				"2.0.0.0/24",
			]
		}`

	checkProperties := func(expectedRoutes []string) func(client *tsclient.Client, rs *terraform.ResourceState) error {
		return func(client *tsclient.Client, rs *terraform.ResourceState) error {
			deviceID := rs.Primary.Attributes["device_id"]

			routes, err := client.Devices().SubnetRoutes(context.Background(), deviceID)
			if err != nil {
				return err
			}

			if !reflect.DeepEqual(routes.Enabled, expectedRoutes) {
				return fmt.Errorf("bad enabled subnet routes: %#v", routes)
			}
			return nil
		}
	}

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories(t),
		CheckDestroy:      checkResourceDestroyed(resourceName, checkProperties([]string{})),
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(testDeviceSubnetRoutesCreate, os.Getenv("TAILSCALE_TEST_DEVICE_NAME")),
				Check: resource.ComposeTestCheckFunc(
					checkResourceRemoteProperties(resourceName, checkProperties([]string{"2.0.0.0/24", "10.0.1.0/24"})),
					resource.TestCheckTypeSetElemAttr(resourceName, "routes.*", "10.0.1.0/24"),
					resource.TestCheckTypeSetElemAttr(resourceName, "routes.*", "2.0.0.0/24"),
				),
			},
			{
				Config: fmt.Sprintf(testDeviceSubnetRoutesUpdate, os.Getenv("TAILSCALE_TEST_DEVICE_NAME")),
				Check: resource.ComposeTestCheckFunc(
					checkResourceRemoteProperties(resourceName, checkProperties([]string{"1.2.0.0/16", "2.0.0.0/24", "10.0.1.0/24"})),
					resource.TestCheckTypeSetElemAttr(resourceName, "routes.*", "10.0.1.0/24"),
					resource.TestCheckTypeSetElemAttr(resourceName, "routes.*", "1.2.0.0/16"),
					resource.TestCheckTypeSetElemAttr(resourceName, "routes.*", "2.0.0.0/24"),
				),
			},
		},
	})
}
