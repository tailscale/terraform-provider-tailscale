// Copyright (c) David Bond, Tailscale Inc, & Contributors
// SPDX-License-Identifier: MIT

package tailscale

import (
	"context"
	"net/http"
	"regexp"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
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

const testDNSConfigurationOptionalUnset = `
	resource "tailscale_dns_configuration" "test_configuration" {
	}`

const testDNSConfigurationOptionalEmpty = `
	resource "tailscale_dns_configuration" "test_configuration" {
		search_paths       = []
	}`

const testDNSConfigurationNoNameservers = `
	resource "tailscale_dns_configuration" "test_configuration" {
		split_dns {
			domain = "bar.example.com"
		}
	}`

func TestProvider_TailscaleDNSConfiguration(t *testing.T) {
	resource.Test(t, resource.TestCase{
		IsUnitTest: true,
		PreCheck: func() {
			testServer.ResponseCode = http.StatusOK
			testServer.ResponseBody = nil
		},
		ProtoV5ProviderFactories: testProviderFactories(t),
		Steps: []resource.TestStep{
			testResourceCreated("tailscale_dns_configuration.test_configuration", testDNSConfigurationCreate),
			testResourceDestroyed("tailscale_dns_configuration.test_configuration", testDNSConfigurationCreate),
			resource.TestStep{
				PreConfig: func() {

				},
				Config:      testDNSConfigurationNoNameservers,
				ExpectError: regexp.MustCompile(`nameservers.*blocks`),
			},
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

	createCheck := resource.ComposeTestCheckFunc(
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
	)

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV5ProviderFactories: testAccProviderFactories(t),
		CheckDestroy:             checkResourceDestroyed(resourceName, checkProperties(&tailscale.DNSConfiguration{})),
		Steps: []resource.TestStep{
			{
				Config: testDNSConfigurationCreate,
				Check:  createCheck,
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
				Config: testDNSConfigurationOptionalUnset,
			},
			{
				Config: testDNSConfigurationOptionalEmpty,
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
	checkResourceIsUnchangedInPluginFramework(t, testDNSConfigurationCreate, createCheck)
}

func TestReconcileNameservers(t *testing.T) {
	testCases := []struct {
		name     string
		existing []nameserverModel
		updates  []tailscale.DNSConfigurationResolver
		want     []nameserverModel
	}{
		{
			name:     "empty-inputs",
			existing: nil,
			updates:  nil,
			want:     []nameserverModel{},
		},
		{
			// new nameservers are appended when there's no existing state
			name:     "new-nameservers-on-empty",
			existing: nil,
			updates: []tailscale.DNSConfigurationResolver{
				{Address: "1.1.1.1", UseWithExitNode: true},
				{Address: "8.8.8.8", UseWithExitNode: false},
			},
			want: []nameserverModel{
				{Address: types.StringValue("1.1.1.1"), UseWithExitNode: types.BoolValue(true)},
				{Address: types.StringValue("8.8.8.8"), UseWithExitNode: types.BoolValue(false)},
			},
		},
		{
			// When you're updating fields, the existing order is preserved
			name: "preserves-existing-order",
			existing: []nameserverModel{
				{Address: types.StringValue("8.8.8.8"), UseWithExitNode: types.BoolValue(false)},
				{Address: types.StringValue("1.1.1.1"), UseWithExitNode: types.BoolValue(false)},
			},
			updates: []tailscale.DNSConfigurationResolver{
				// both configs: UseWithExitNode: false -> true
				{Address: "1.1.1.1", UseWithExitNode: true},
				{Address: "8.8.8.8", UseWithExitNode: true},
			},
			want: []nameserverModel{
				{Address: types.StringValue("8.8.8.8"), UseWithExitNode: types.BoolValue(true)},
				{Address: types.StringValue("1.1.1.1"), UseWithExitNode: types.BoolValue(true)},
			},
		},
		{
			// When the update changes one field and removes another, the missing entry is removed
			// and the remaining entry is updated.
			name: "mix-of-update-and-removal",
			existing: []nameserverModel{
				{Address: types.StringValue("1.1.1.1"), UseWithExitNode: types.BoolValue(false)},
				{Address: types.StringValue("9.9.9.9"), UseWithExitNode: types.BoolValue(false)},
			},
			updates: []tailscale.DNSConfigurationResolver{
				// 1.1.1.1: UseWithExitNode: false -> true
				{Address: "1.1.1.1", UseWithExitNode: types.BoolValue(true).ValueBool()},
				// 8.8.8.8: added in update, not in existing
				{Address: "8.8.8.8", UseWithExitNode: false},
				// 9.9.9.9: present in existing, not in update
			},
			want: []nameserverModel{
				{Address: types.StringValue("1.1.1.1"), UseWithExitNode: types.BoolValue(true)},
				{Address: types.StringValue("8.8.8.8"), UseWithExitNode: types.BoolValue(false)},
			},
		},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := reconcileNameservers(tt.existing, tt.updates)
			if diff := cmp.Diff(tt.want, got); diff != "" {
				t.Errorf("reconcileNameservers() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestReconcileSplitDNS(t *testing.T) {
	testCases := []struct {
		name     string
		existing []splitDNSModel
		updates  map[string][]tailscale.DNSConfigurationResolver
		want     []splitDNSModel
	}{
		{
			name:     "empty-inputs",
			existing: nil,
			updates:  nil,
			want:     []splitDNSModel{},
		},
		{
			// When you're updating domains, the existing order is preserved
			name: "preserves-domain-order",
			existing: []splitDNSModel{
				{
					Domain: types.StringValue("example.com"),
					Nameservers: []nameserverModel{
						{Address: types.StringValue("10.0.0.1"), UseWithExitNode: types.BoolValue(false)},
					},
				},
				{
					Domain: types.StringValue("internal.net"),
					Nameservers: []nameserverModel{
						{Address: types.StringValue("192.168.1.1"), UseWithExitNode: types.BoolValue(false)},
					},
				},
			},
			updates: map[string][]tailscale.DNSConfigurationResolver{
				// both domains: UseWithExitNode: false -> true
				"example.com":  {{Address: "10.0.0.1", UseWithExitNode: true}},
				"internal.net": {{Address: "192.168.1.1", UseWithExitNode: true}},
			},
			want: []splitDNSModel{
				{
					Domain: types.StringValue("example.com"),
					Nameservers: []nameserverModel{
						{Address: types.StringValue("10.0.0.1"), UseWithExitNode: types.BoolValue(true)},
					},
				},
				{
					Domain: types.StringValue("internal.net"),
					Nameservers: []nameserverModel{
						{Address: types.StringValue("192.168.1.1"), UseWithExitNode: types.BoolValue(true)},
					},
				},
			},
		},
		{
			// When the update changes one field and removes another, the missing entry is removed
			// and the remaining entry is updated.
			name: "mix-of-update-and-removal",
			existing: []splitDNSModel{
				{
					Domain: types.StringValue("example.com"),
					Nameservers: []nameserverModel{
						{Address: types.StringValue("10.0.0.1"), UseWithExitNode: types.BoolValue(false)},
					},
				},
				{
					Domain: types.StringValue("old-domain.com"),
					Nameservers: []nameserverModel{
						{Address: types.StringValue("10.0.0.1"), UseWithExitNode: types.BoolValue(false)},
					},
				},
			},
			updates: map[string][]tailscale.DNSConfigurationResolver{
				// example.com: Address: 10.0.0.1 -> 1.1.1.1, UseWithExitNode: false -> true
				"example.com": {{Address: "1.1.1.1", UseWithExitNode: true}},
				// new-domain.com: added in update, not in existing
				"new-domain.com": {
					{Address: "1.1.1.1", UseWithExitNode: false},
				},
				// old-domain.com: present in existing, not in update
			},
			want: []splitDNSModel{
				{
					Domain: types.StringValue("example.com"),
					Nameservers: []nameserverModel{
						{Address: types.StringValue("1.1.1.1"), UseWithExitNode: types.BoolValue(true)},
					},
				},
				{
					Domain: types.StringValue("new-domain.com"),
					Nameservers: []nameserverModel{
						{Address: types.StringValue("1.1.1.1"), UseWithExitNode: types.BoolValue(false)},
					},
				},
			},
		},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := reconcileSplitDNS(tt.existing, tt.updates)
			if diff := cmp.Diff(tt.want, got); diff != "" {
				t.Errorf("reconcileSplitDNS() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}
