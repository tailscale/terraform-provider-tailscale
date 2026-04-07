// Copyright (c) David Bond, Tailscale Inc, & Contributors
// SPDX-License-Identifier: MIT

package tailscale

import (
	"context"
	"fmt"
	"net/http"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"

	"tailscale.com/client/tailscale/v2"
)

const testDataSourceService = `
	data "tailscale_service" "test_service" {
		name = "svc:test-service"
	}`

func TestProvider_DataSourceTailscaleService(t *testing.T) {
	resource.Test(t, resource.TestCase{
		IsUnitTest: true,
		PreCheck: func() {
			testServer.ResponseCode = http.StatusOK
			testServer.ResponseBody = tailscale.VIPService{
				Name:    "svc:test-service",
				Addrs:   []string{"100.64.0.1", "fd7a:115c:a1e0::1"},
				Comment: "a test Service",
				Ports:   []string{"tcp:443"},
				Tags:    []string{"tag:web"},
			}
		},
		ProtoV5ProviderFactories: testProviderFactories(t),
		Steps: []resource.TestStep{
			{
				Config: testDataSourceService,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("data.tailscale_service.test_service", "name", "svc:test-service"),
					resource.TestCheckResourceAttr("data.tailscale_service.test_service", "addrs.#", "2"),
					resource.TestCheckTypeSetElemAttr("data.tailscale_service.test_service", "addrs.*", "100.64.0.1"),
					resource.TestCheckTypeSetElemAttr("data.tailscale_service.test_service", "addrs.*", "fd7a:115c:a1e0::1"),
					resource.TestCheckResourceAttr("data.tailscale_service.test_service", "comment", "a test Service"),
					resource.TestCheckResourceAttr("data.tailscale_service.test_service", "ports.#", "1"),
					resource.TestCheckResourceAttr("data.tailscale_service.test_service", "ports.0", "tcp:443"),
				),
			},
		},
	})
}

func TestAccTailscaleDataSourceService(t *testing.T) {
	const resourceName = "data.tailscale_service.test_service"

	// This test requires a Service to already exist. We create one first
	// using the resource, then look it up with the data source.
	const config = `
		resource "tailscale_service" "test_service" {
			name    = "svc:tf-test-ds"
			comment = "data source test"
			ports   = ["tcp:443"]
			tags    = ["tag:test"]
		}

		data "tailscale_service" "test_service" {
			name = tailscale_service.test_service.name
		}`

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV5ProviderFactories: testAccProviderFactories(t),
		CheckDestroy: checkResourceDestroyed("tailscale_service.test_service", func(client *tailscale.Client, rs *terraform.ResourceState) error {
			_, err := client.VIPServices().Get(context.Background(), rs.Primary.ID)
			if err == nil {
				return fmt.Errorf("Service %q still exists on server", rs.Primary.ID)
			}
			return nil
		}),
		Steps: []resource.TestStep{
			{
				PreConfig: func() {
					client := testAccProvider.Meta().(*tailscale.Client)
					err := client.PolicyFile().Set(context.Background(), `
					{
						"tagOwners": {
							"tag:test": ["autogroup:member"]
						}
					}`, "")
					if err != nil {
						panic(err)
					}
				},
				Config: config,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "name", "svc:tf-test-ds"),
					resource.TestCheckResourceAttr(resourceName, "comment", "data source test"),
					resource.TestCheckResourceAttrSet(resourceName, "addrs.#"),
				),
			},
		},
	})
}
