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

const testDNSConfigurationCreate = `
	resource "tailscale_dns_configuration" "test_configuration" {
		nameservers {
			address            = "8.8.8.8"
		}
		nameservers {
			address            = "1.1.1.1"
			use_with_exit_node = true
		}
		split_dns {
			domain             = "foo.example.com"
			nameservers {
				address            = "1.1.1.2"
				use_with_exit_node = true
			}
			nameservers {
				address            = "1.1.1.3"
			}
		}
		split_dns {
			domain             = "bar.example.com"
			nameservers {
				address            = "8.8.8.2"
				use_with_exit_node = true
			}
		}
		search_paths       = ["example.com", "anotherexample.com"]
		override_local_dns = true
		magic_dns = true
	}`

const testDNSConfigurationUpdate = `
	resource "tailscale_dns_configuration" "test_configuration" {
		nameservers {
			address            = "8.8.8.8"
			use_with_exit_node = true
		}
		split_dns {
			domain             = "bar.example.com"
			nameservers {
				address            = "8.8.8.2"
				use_with_exit_node = false
			}
		}
		search_paths       = ["anotherexample.com"]
		override_local_dns = false
		magic_dns = false
	}`

func TestProvider_TailscaleDNSConfiguration(t *testing.T) {
	resource.Test(t, resource.TestCase{
		IsUnitTest: true,
		PreCheck: func() {
			testServer.ResponseCode = http.StatusOK
			testServer.ResponseBody = nil
		},
		ProviderFactories: testProviderFactories(t),
		Steps: []resource.TestStep{
			testResourceCreated("tailscale_dns_configuration.test_configuration", testDNSConfigurationCreate),
			testResourceDestroyed("tailscale_dns_configuration.test_configuration", testDNSConfigurationCreate),
		},
	})
}

func TestAccTailscaleDNSConfiguration(t *testing.T) {
	const resourceName = "tailscale_dns_configuration.test_configuration"

	checkProperties := func(expected *tailscale.DNSConfiguration) func(client *tailscale.Client, rs *terraform.ResourceState) error {
		return func(client *tailscale.Client, rs *terraform.ResourceState) error {
			actual, err := client.DNS().Configuration(context.Background())
			if err != nil {
				return err
			}

			if err := assertEqual(expected, actual, "wrong DNS configuration"); err != nil {
				return err
			}

			return nil
		}
	}

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories(t),
		CheckDestroy:      checkResourceDestroyed(resourceName, checkProperties(&tailscale.DNSConfiguration{})),
		Steps: []resource.TestStep{
			{
				Config: testDNSConfigurationCreate,
				Check: resource.ComposeTestCheckFunc(
					checkResourceRemoteProperties(resourceName,
						checkProperties(&tailscale.DNSConfiguration{
							Nameservers: []tailscale.DNSConfigurationResolver{{Address: "8.8.8.8"}, {Address: "1.1.1.1", UseWithExitNode: true}},
							SplitDNS: map[string][]tailscale.DNSConfigurationResolver{
								"bar.example.com": {{Address: "8.8.8.2", UseWithExitNode: true}},
								"foo.example.com": {{Address: "1.1.1.2", UseWithExitNode: true}, {Address: "1.1.1.3"}},
							},
							SearchPaths: []string{"example.com", "anotherexample.com"},
							Preferences: tailscale.DNSConfigurationPreferences{
								OverrideLocalDNS: true,
								MagicDNS:         true,
							},
						}),
					),
					resource.TestCheckResourceAttr(resourceName, "nameservers.0.address", "8.8.8.8"),
					resource.TestCheckResourceAttr(resourceName, "nameservers.0.use_with_exit_node", "false"),
					resource.TestCheckResourceAttr(resourceName, "nameservers.1.address", "1.1.1.1"),
					resource.TestCheckResourceAttr(resourceName, "nameservers.1.use_with_exit_node", "true"),
					resource.TestCheckResourceAttr(resourceName, "split_dns.0.domain", "foo.example.com"),
					resource.TestCheckResourceAttr(resourceName, "split_dns.0.nameservers.0.address", "1.1.1.2"),
					resource.TestCheckResourceAttr(resourceName, "split_dns.0.nameservers.0.use_with_exit_node", "true"),
					resource.TestCheckResourceAttr(resourceName, "split_dns.0.nameservers.1.address", "1.1.1.3"),
					resource.TestCheckResourceAttr(resourceName, "split_dns.0.nameservers.1.use_with_exit_node", "false"),
					resource.TestCheckResourceAttr(resourceName, "split_dns.1.domain", "bar.example.com"),
					resource.TestCheckResourceAttr(resourceName, "split_dns.1.nameservers.0.address", "8.8.8.2"),
					resource.TestCheckResourceAttr(resourceName, "split_dns.1.nameservers.0.use_with_exit_node", "true"),
					resource.TestCheckResourceAttr(resourceName, "search_paths.0", "example.com"),
					resource.TestCheckResourceAttr(resourceName, "search_paths.1", "anotherexample.com"),
					resource.TestCheckResourceAttr(resourceName, "override_local_dns", "true"),
					resource.TestCheckResourceAttr(resourceName, "magic_dns", "true"),
				),
			},
			{
				Config: testDNSConfigurationUpdate,
				Check: resource.ComposeTestCheckFunc(
					checkResourceRemoteProperties(resourceName,
						checkProperties(&tailscale.DNSConfiguration{
							Nameservers: []tailscale.DNSConfigurationResolver{{Address: "8.8.8.8", UseWithExitNode: true}},
							SplitDNS: map[string][]tailscale.DNSConfigurationResolver{
								"bar.example.com": {{Address: "8.8.8.2", UseWithExitNode: false}},
							},
							SearchPaths: []string{"anotherexample.com"},
							Preferences: tailscale.DNSConfigurationPreferences{
								OverrideLocalDNS: false,
								MagicDNS:         false,
							},
						}),
					),
					resource.TestCheckResourceAttr(resourceName, "nameservers.0.address", "8.8.8.8"),
					resource.TestCheckResourceAttr(resourceName, "nameservers.0.use_with_exit_node", "true"),
					resource.TestCheckResourceAttr(resourceName, "split_dns.0.domain", "bar.example.com"),
					resource.TestCheckResourceAttr(resourceName, "split_dns.0.nameservers.0.address", "8.8.8.2"),
					resource.TestCheckResourceAttr(resourceName, "split_dns.0.nameservers.0.use_with_exit_node", "false"),
					resource.TestCheckResourceAttr(resourceName, "search_paths.0", "anotherexample.com"),
					resource.TestCheckResourceAttr(resourceName, "override_local_dns", "false"),
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
