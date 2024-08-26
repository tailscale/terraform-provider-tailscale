package tailscale

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

func TestAccTailscaleDevices(t *testing.T) {
	resourceName := "data.tailscale_devices.all_devices"

	// This is a string containing tailscale_device datasource configurations
	devicesDataSources := &strings.Builder{}

	toResourceComponent := func(str string) string {
		return strings.ReplaceAll(str, " ", "_")
	}

	// First test the tailscale_devices datasource, which will give us a list of
	// all device IDs.
	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories(t),
		Steps: []resource.TestStep{
			{
				Config: `data "tailscale_devices" "all_devices" {}`,
				Check: func(s *terraform.State) error {
					client := testAccProvider.Meta().(*Clients).V2
					devices, err := client.Devices().List(context.Background())
					if err != nil {
						return fmt.Errorf("unable to list devices: %s", err)
					}

					devicesByID := make(map[string]map[string]any)
					for _, device := range devices {
						m := deviceToMap(&device)
						m["id"] = device.ID
						devicesByID[device.ID] = m
					}

					rs := s.RootModule().Resources[resourceName].Primary

					// first find indexes for devices
					deviceIndexes := make(map[string]string)
					for k, v := range rs.Attributes {
						if strings.HasSuffix(k, ".id") {
							idx := strings.Split(k, ".")[1]
							deviceIndexes[idx] = v
						}
					}

					// make sure we got the right number of devices
					if len(deviceIndexes) != len(devicesByID) {
						return fmt.Errorf("wrong number of devices in datasource, want %d, got %d", len(devicesByID), len(deviceIndexes))
					}

					// now compare datasource attributes to expected values
					for k, v := range rs.Attributes {
						if strings.HasPrefix(k, "devices.") {
							parts := strings.Split(k, ".")
							if len(parts) != 3 {
								continue
							}
							prop := parts[2]
							if prop == "%" {
								continue
							}
							idx := parts[1]
							id := deviceIndexes[idx]
							expected := fmt.Sprint(devicesByID[id][prop])
							if v != expected {
								return fmt.Errorf("wrong value of %s for device %s, want %q, got %q", prop, id, expected, v)
							}
						}
					}

					// Now set up device datasources for each device. This is used in the following test
					// of the tailscale_device datasource.
					for _, device := range devices {
						if device.Hostname != "" {
							devicesDataSources.WriteString(fmt.Sprintf("\ndata \"tailscale_device\" \"%s\" {\n  hostname = \"%s\"\n}\n", toResourceComponent(device.Hostname), device.Hostname))
						} else {
							devicesDataSources.WriteString(fmt.Sprintf("\ndata \"tailscale_device\" \"%s\" {\n  name = \"%s\"\n}\n", toResourceComponent(device.Name), device.Name))
						}
					}

					return nil
				},
			},
		},
	})

	// Now test the individual tailscale_device data sources for each device,
	// making sure that it pulls in the relevant details for each device.
	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories(t),
		Steps: []resource.TestStep{
			{
				Config: devicesDataSources.String(),
				Check: func(s *terraform.State) error {
					client := testAccProvider.Meta().(*Clients).V2
					devices, err := client.Devices().List(context.Background())
					if err != nil {
						return fmt.Errorf("unable to list devices: %s", err)
					}

					for _, device := range devices {
						expected := deviceToMap(&device)
						expected["id"] = device.ID
						var nameComponent string
						if device.Hostname != "" {
							nameComponent = device.Hostname
						} else {
							nameComponent = device.Name
						}
						resourceName := fmt.Sprintf("data.tailscale_device.%s", toResourceComponent(nameComponent))
						if err := checkPropertiesMatch(resourceName, s, expected); err != nil {
							return err
						}
					}

					return nil
				},
			},
		},
	})
}
