// Copyright (c) David Bond, Tailscale Inc, & Contributors
// SPDX-License-Identifier: MIT

package tailscale

import (
	"context"
	"net/http"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"

	"tailscale.com/client/tailscale/v2"
)

const testDNSPreferencesCreate = `
	resource "tailscale_dns_preferences" "test_preferences" {
		magic_dns = true
	}`

const testDNSPreferencesUpdate = `
	resource "tailscale_dns_preferences" "test_preferences" {
		magic_dns = false
	}`

func TestProvider_TailscaleDNSPreferences(t *testing.T) {
	resource.Test(t, resource.TestCase{
		IsUnitTest: true,
		PreCheck: func() {
			testServer.ResponseCode = http.StatusOK
			testServer.ResponseBody = nil
		},
		ProviderFactories: testProviderFactories(t),
		Steps: []resource.TestStep{
			testResourceCreated("tailscale_dns_preferences.test_preferences", testDNSPreferencesCreate),
			testResourceDestroyed("tailscale_dns_preferences.test_preferences", testDNSPreferencesCreate),
		},
	})
}

func TestAccTailscaleDNSPreferences(t *testing.T) {
	const resourceName = "tailscale_dns_preferences.test_preferences"

	checkProperties := func(expected *tailscale.DNSPreferences) func(client *tailscale.Client, rs *terraform.ResourceState) error {
		return func(client *tailscale.Client, rs *terraform.ResourceState) error {
			actual, err := client.DNS().Preferences(context.Background())
			if err != nil {
				return err
			}

			if err := assertEqual(expected, actual, "wrong DNS preferences"); err != nil {
				return err
			}

			return nil
		}
	}

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories(t),
		CheckDestroy:      checkResourceDestroyed(resourceName, checkProperties(&tailscale.DNSPreferences{})),
		Steps: []resource.TestStep{
			{
				Config: testDNSPreferencesCreate,
				Check: resource.ComposeTestCheckFunc(
					checkResourceRemoteProperties(resourceName,
						checkProperties(&tailscale.DNSPreferences{MagicDNS: true}),
					),
					resource.TestCheckResourceAttr(resourceName, "magic_dns", "true"),
				),
			},
			{
				Config: testDNSPreferencesUpdate,
				Check: resource.ComposeTestCheckFunc(
					checkResourceRemoteProperties(resourceName,
						checkProperties(&tailscale.DNSPreferences{MagicDNS: false}),
					),
					resource.TestCheckResourceAttr(resourceName, "magic_dns", "false"),
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
