// Copyright (c) David Bond, Tailscale Inc, & Contributors
// SPDX-License-Identifier: MIT

package tailscale

import (
	"context"
	"fmt"
	"os"
	"reflect"
	"strings"
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

	var deviceId string
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
			{
				ResourceName: resourceName,
				ImportState:  true,
				// Need import state ID func to dynamically grab device_id for import.
				ImportStateIdFunc: func(state *terraform.State) (string, error) {
					rs, ok := state.RootModule().Resources[resourceName]
					if !ok {
						return "", fmt.Errorf("resource not found: %s", resourceName)

					}

					deviceId = rs.Primary.Attributes["device_id"]

					return deviceId, nil
				},
				// Need a custom import state check due to the fact that the ID for this
				// resource is re-generated on import.
				ImportStateCheck: func(states []*terraform.InstanceState) error {
					if len(states) != 1 {
						return fmt.Errorf("expected 1 state: %+v", states)
					}

					rs := states[0]
					elemCheck := func(attr string, value string) bool {
						attrParts := strings.Split(attr, ".")
						for stateKey, stateValue := range rs.Attributes {
							if stateValue == value {
								stateKeyParts := strings.Split(stateKey, ".")
								if len(stateKeyParts) == len(attrParts) {
									for i := range attrParts {
										if attrParts[i] != stateKeyParts[i] && attrParts[i] != "*" {
											break
										}
										if i == len(attrParts)-1 {
											return true
										}
									}
								}
							}
						}

						return false
					}

					if rs.Attributes["device_id"] != deviceId {
						return fmt.Errorf("expected device_id to be %q but was: %q", deviceId, rs.Attributes["device_id"])
					}

					if !elemCheck("routes.*", "10.0.1.0/24") {
						return fmt.Errorf("expected routes to contain '10.0.1.0/24': %#v", rs.Attributes)
					}
					if !elemCheck("routes.*", "1.2.0.0/16") {
						return fmt.Errorf("expected routes to contain '1.2.0.0/16': %#v", rs.Attributes)
					}
					if !elemCheck("routes.*", "2.0.0.0/24") {
						return fmt.Errorf("expected routes to contain '2.0.0.0/24': %#v", rs.Attributes)
					}

					return nil
				},
			},
		},
	})
}
