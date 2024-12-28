// Copyright (c) David Bond, Tailscale Inc, & Contributors
// SPDX-License-Identifier: MIT

package tailscale

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"

	tsclient "github.com/tailscale/tailscale-client-go/v2"
)

const testNameserversCreate = `
	resource "tailscale_dns_nameservers" "test_nameservers" {
		nameservers = [
			"8.8.8.8",
			"8.8.4.4",
		]
	}`

const testNameserversUpdate = `
	resource "tailscale_dns_nameservers" "test_nameservers" {
		nameservers = [
			"1.1.1.1",
		]
	}`

func TestAccTailscaleDNSNameservers(t *testing.T) {
	const resourceName = "tailscale_dns_nameservers.test_nameservers"

	checkProperties := func(expected []string) func(client *tsclient.Client, rs *terraform.ResourceState) error {
		return func(client *tsclient.Client, rs *terraform.ResourceState) error {
			actual, err := client.DNS().Nameservers(context.Background())
			if err != nil {
				return err
			}

			if err := assertEqual(expected, actual, "wrong nameservers"); err != nil {
				return err
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
				Config: testNameserversCreate,
				Check: resource.ComposeTestCheckFunc(
					checkResourceRemoteProperties(resourceName,
						checkProperties([]string{"8.8.8.8", "8.8.4.4"}),
					),
					resource.TestCheckTypeSetElemAttr(resourceName, "nameservers.*", "8.8.8.8"),
					resource.TestCheckTypeSetElemAttr(resourceName, "nameservers.*", "8.8.4.4"),
				),
			},
			{
				Config: testNameserversUpdate,
				Check: resource.ComposeTestCheckFunc(
					checkResourceRemoteProperties(resourceName,
						checkProperties([]string{"1.1.1.1"}),
					),
					resource.TestCheckTypeSetElemAttr(resourceName, "nameservers.*", "1.1.1.1"),
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
