package tailscale_test

import (
	"fmt"
	"net/http"
	"strconv"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
	"github.com/tailscale/tailscale-client-go/tailscale"
)

const testDataSourceDevicesPrefix = `
data "tailscale_devices" "sample_devices_prefix" {
  name_prefix = "device"
}
`
const testDataSourceDevicesRegexp = `
data "tailscale_devices" "sample_devices_regexp" {
  name_regexp = "-(mobile|laptop)$"
}
`

func TestProvider_DataSourceTailscaleDevices(t *testing.T) {
	resource.Test(t, resource.TestCase{
		IsUnitTest: true,
		PreCheck: func() {
			testServer.ResponseCode = http.StatusOK
			testServer.ResponseBody = map[string][]tailscale.Device{
				"devices": {
					{
						Name: "device.example.com",
						ID:   "123",
					},
					{
						Name: "unknown-mobile",
						ID:   "234",
					},
				},
			}
		},
		ProviderFactories: testProviderFactories(t),
		Steps: []resource.TestStep{
			{
				Config: testDataSourceDevicesPrefix,
				Check:  checkDevicesResult("data.tailscale_devices.sample_devices_prefix"),
			},
			{
				Config: testDataSourceDevicesRegexp,
				Check:  checkDevicesResult("data.tailscale_devices.sample_devices_regexp"),
			},
		},
	})
}

func checkDevicesResult(n string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("can't find tailscale_devices resource: %s", n)
		}
		if rs.Primary.ID == "" {
			return fmt.Errorf("devices data source ID not set")
		}

		lenDevices, _ := strconv.Atoi(rs.Primary.Attributes["devices.#"])
		if lenDevices != 1 {
			return fmt.Errorf("devices size expected to not be 1 but was %d", lenDevices)
		}

		if _, ok := rs.Primary.Attributes["name_prefix"]; ok {
			if rs.Primary.Attributes["devices.0.name"] != "device.example.com" {
				return fmt.Errorf("device name expected to be device.example.com")
			}
		} else if _, ok := rs.Primary.Attributes["name_regexp"]; ok {
			if rs.Primary.Attributes["devices.0.name"] != "unknown-mobile" {
				return fmt.Errorf("device name expected to be unknown-mobile")
			}
		} else {
			return fmt.Errorf("tailscale_devices expects name_prefix or name_regexp")
		}

		return nil
	}
}
