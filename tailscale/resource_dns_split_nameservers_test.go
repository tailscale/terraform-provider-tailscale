// Copyright (c) David Bond, Tailscale Inc, & Contributors
// SPDX-License-Identifier: MIT

package tailscale

import (
	"context"
	"fmt"
	"net/http"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"

	"tailscale.com/client/tailscale/v2"
)

const testSplitNameservers = `
	resource "tailscale_dns_split_nameservers" "test_nameservers" {
        domain = "example.com"
		nameservers = ["1.2.3.4", "4.5.6.7"]
		use_with_exit_node = true
	}`

func TestProvider_TailscaleSplitDNSNameservers(t *testing.T) {
	resource.Test(t, resource.TestCase{
		IsUnitTest: true,
		PreCheck: func() {
			testServer.ResponseCode = http.StatusOK
			testServer.ResponseBody = nil
		},
		ProtoV5ProviderFactories: testProviderFactories(t),
		Steps: []resource.TestStep{
			testResourceCreated("tailscale_dns_split_nameservers.test_nameservers", testSplitNameservers),
			testResourceDestroyed("tailscale_dns_split_nameservers.test_nameservers", testSplitNameservers),
		},
	})
}

func TestAccTailscaleDNSSplitNameservers(t *testing.T) {
	const resourceName = "tailscale_dns_split_nameservers.test_nameservers"

	const testSplitNameserversCreate = `
		resource "tailscale_dns_split_nameservers" "test_nameservers" {
			domain = "example.com"
			nameservers = ["1.2.3.4", "4.5.6.7"]
			use_with_exit_node = true
		}`

	const testSplitNameserversUpdate = `
		resource "tailscale_dns_split_nameservers" "test_nameservers" {
			domain = "sub.example.com"
			nameservers = ["8.8.9.9"]
			use_with_exit_node = false
		}`

	const testSplitNameserversEmpty = `
		resource "tailscale_dns_split_nameservers" "test_nameservers" {
			domain = "sub.example.com"
			nameservers = []
			use_with_exit_node = false
		}`

	const testSplitNameserversUpdateSameDomain = `
		resource "tailscale_dns_split_nameservers" "test_nameservers" {
			domain = "sub.example.com"
			nameservers = ["8.8.7.7", "8.8.9.9"]
			use_with_exit_node = true
		}`

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV5ProviderFactories: testAccProviderFactories(t),
		CheckDestroy:             checkResourceDestroyed(resourceName, checkSplitDNSProperties(nil)),
		Steps: []resource.TestStep{
			{
				Config: testSplitNameserversCreate,
				Check: resource.ComposeTestCheckFunc(
					checkResourceRemoteProperties(resourceName,
						checkSplitDNSProperties(map[string][]tailscale.DNSConfigurationResolver{
							"example.com": {
								{Address: "1.2.3.4", UseWithExitNode: true},
								{Address: "4.5.6.7", UseWithExitNode: true},
							},
						}),
					),
					resource.TestCheckResourceAttr(resourceName, "domain", "example.com"),
					resource.TestCheckTypeSetElemAttr(resourceName, "nameservers.*", "1.2.3.4"),
					resource.TestCheckTypeSetElemAttr(resourceName, "nameservers.*", "4.5.6.7"),
					resource.TestCheckResourceAttr(resourceName, "use_with_exit_node", "true"),
				),
			},
			{
				Config: testSplitNameserversUpdate,
				Check: resource.ComposeTestCheckFunc(
					checkResourceRemoteProperties(resourceName,
						checkSplitDNSProperties(map[string][]tailscale.DNSConfigurationResolver{
							"sub.example.com": {{Address: "8.8.9.9"}},
						}),
					),
					resource.TestCheckResourceAttr(resourceName, "domain", "sub.example.com"),
					resource.TestCheckTypeSetElemAttr(resourceName, "nameservers.*", "8.8.9.9"),
					resource.TestCheckResourceAttr(resourceName, "use_with_exit_node", "false"),
				),
			},
			{
				Config: testSplitNameserversUpdateSameDomain,
				Check: resource.ComposeTestCheckFunc(
					checkResourceRemoteProperties(resourceName,
						checkSplitDNSProperties(map[string][]tailscale.DNSConfigurationResolver{
							"sub.example.com": {
								{Address: "8.8.7.7", UseWithExitNode: true},
								{Address: "8.8.9.9", UseWithExitNode: true},
							},
						}),
					),
					resource.TestCheckResourceAttr(resourceName, "domain", "sub.example.com"),
					resource.TestCheckTypeSetElemAttr(resourceName, "nameservers.*", "8.8.7.7"),
					resource.TestCheckTypeSetElemAttr(resourceName, "nameservers.*", "8.8.9.9"),
					resource.TestCheckResourceAttr(resourceName, "use_with_exit_node", "true"),
				),
			},
			{
				Config: testSplitNameserversEmpty,
				Check: resource.ComposeTestCheckFunc(
					checkResourceRemoteProperties(resourceName,
						checkSplitDNSProperties(nil),
					),
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

func checkSplitDNSProperties(expected map[string][]tailscale.DNSConfigurationResolver) func(client *tailscale.Client, rs *terraform.ResourceState) error {
	return func(client *tailscale.Client, rs *terraform.ResourceState) error {
		configuration, err := client.DNS().Configuration(context.Background())
		if err != nil {
			return err
		}

		if diff := cmp.Diff(configuration.SplitDNS, expected); diff != "" {
			return fmt.Errorf("wrong split dns: (-got+want) \n%s", diff)
		}

		return nil
	}
}

// Migration test to ensure the resource is unchanged when migrating
// from the plugin SDK to the plugin framework.
//
// See https://developer.hashicorp.com/terraform/plugin/framework/migrating/testing#terraform-data-resource-example
func TestAccTailscaleDNSSplitNameServers_UpgradeToPluginFramework(t *testing.T) {
	resourceName := "tailscale_dns_split_nameservers.test_nameservers"

	checkResourceIsUnchangedInPluginFramework(t,
		`resource "tailscale_dns_split_nameservers" "test_nameservers" {
			domain = "example.com"
			nameservers = ["1.2.3.4", "4.5.6.7"]
		}`,
		resource.ComposeTestCheckFunc(
			checkResourceRemoteProperties(resourceName,
				checkSplitDNSProperties(map[string][]tailscale.DNSConfigurationResolver{
					"example.com": {{Address: "1.2.3.4"}, {Address: "4.5.6.7"}},
				}),
			),
			resource.TestCheckResourceAttr(resourceName, "domain", "example.com"),
			resource.TestCheckTypeSetElemAttr(resourceName, "nameservers.*", "1.2.3.4"),
			resource.TestCheckTypeSetElemAttr(resourceName, "nameservers.*", "4.5.6.7"),
		))
}
