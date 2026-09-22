// Copyright (c) David Bond, Tailscale Inc, & Contributors
// SPDX-License-Identifier: MIT

package tailscale

import (
	"context"
	"net/http"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"

	"tailscale.com/client/tailscale/v2"
)

const testDNSPreferencesCreate = `
	resource "tailscale_dns_preferences" "test_preferences" {
		magic_dns         = true
		override_local_dns = true
	}`

const testDNSPreferencesUpdate = `
	resource "tailscale_dns_preferences" "test_preferences" {
		magic_dns          = false
		override_local_dns = false
	}`

const testDNSPreferencesLegacy = `
	resource "tailscale_dns_preferences" "test_preferences" {
		magic_dns = true
	}`

func TestProvider_TailscaleDNSPreferences(t *testing.T) {
	resource.Test(t, resource.TestCase{
		IsUnitTest: true,
		PreCheck: func() {
			testServer.ResponseCode = http.StatusOK
			testServer.ResponseBody = nil
		},
		ProtoV5ProviderFactories: testProviderFactories(t),
		Steps: []resource.TestStep{
			testResourceCreated("tailscale_dns_preferences.test_preferences", testDNSPreferencesCreate),
			testResourceDestroyed("tailscale_dns_preferences.test_preferences", testDNSPreferencesCreate),
		},
	})
}

func checkDNSProperties(expected tailscale.DNSConfigurationPreferences) func(client *tailscale.Client, rs *terraform.ResourceState) error {
	return func(client *tailscale.Client, rs *terraform.ResourceState) error {
		configuration, err := client.DNS().Configuration(context.Background())
		if err != nil {
			return err
		}

		if err := assertEqual(expected, configuration.Preferences, "wrong DNS preferences"); err != nil {
			return err
		}

		return nil
	}
}

func TestAccTailscaleDNSPreferences(t *testing.T) {
	const resourceName = "tailscale_dns_preferences.test_preferences"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV5ProviderFactories: testAccProviderFactories(t),
		CheckDestroy:             checkResourceDestroyed(resourceName, checkDNSProperties(tailscale.DNSConfigurationPreferences{})),
		Steps: []resource.TestStep{
			{
				Config: testDNSPreferencesCreate,
				Check: resource.ComposeTestCheckFunc(
					checkResourceRemoteProperties(resourceName,
						checkDNSProperties(tailscale.DNSConfigurationPreferences{MagicDNS: true, OverrideLocalDNS: true}),
					),
					resource.TestCheckResourceAttr(resourceName, "magic_dns", "true"),
					resource.TestCheckResourceAttr(resourceName, "override_local_dns", "true"),
				),
			},
			{
				Config: testDNSPreferencesUpdate,
				Check: resource.ComposeTestCheckFunc(
					checkResourceRemoteProperties(resourceName,
						checkDNSProperties(tailscale.DNSConfigurationPreferences{}),
					),
					resource.TestCheckResourceAttr(resourceName, "magic_dns", "false"),
					resource.TestCheckResourceAttr(resourceName, "override_local_dns", "false"),
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

// Migration test to ensure the resource is unchanged when migrating
// from the plugin SDK to the plugin framework.
//
// See https://developer.hashicorp.com/terraform/plugin/framework/migrating/testing#terraform-data-resource-example
func TestAccTailscaleDNSPreferences_UpgradeToPluginFramework(t *testing.T) {
	checkResourceIsUnchangedInPluginFramework(t, testDNSPreferencesLegacy, resource.ComposeTestCheckFunc(
		checkResourceRemoteProperties("tailscale_dns_preferences.test_preferences",
			checkDNSProperties(tailscale.DNSConfigurationPreferences{MagicDNS: true}),
		),
		resource.TestCheckResourceAttr("tailscale_dns_preferences.test_preferences", "magic_dns", "true"),
		resource.TestCheckResourceAttr("tailscale_dns_preferences.test_preferences", "override_local_dns", "false"),
	))
}
