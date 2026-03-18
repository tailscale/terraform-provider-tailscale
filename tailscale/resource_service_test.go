// Copyright (c) David Bond, Tailscale Inc, & Contributors
// SPDX-License-Identifier: MIT

package tailscale

import (
	"context"
	"fmt"
	"net/http"
	"slices"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"

	"tailscale.com/client/tailscale/v2"
)

const testService = `
	resource "tailscale_service" "test_service" {
		name    = "svc:test-service"
		comment = "a test Service"
		ports   = ["tcp:443"]
		tags    = ["tag:web"]
	}`

const testServiceUpdate = `
	resource "tailscale_service" "test_service" {
		name    = "svc:test-service"
		comment = "an updated test Service"
		ports   = ["tcp:443", "tcp:8080"]
		tags    = ["tag:web", "tag:api"]
	}`

func TestProvider_TailscaleService(t *testing.T) {
	resource.Test(t, resource.TestCase{
		IsUnitTest: true,
		PreCheck: func() {
			testServer.ResponseCode = http.StatusOK
			testServer.ResponseBody = tailscale.VIPService{
				Name:  "svc:test-service",
				Addrs: []string{"100.64.0.1", "fd7a:115c:a1e0::1"},
			}
		},
		ProtoV5ProviderFactories: testProviderFactories(t),
		Steps: []resource.TestStep{
			testResourceCreated("tailscale_service.test_service", testService),
			testResourceDestroyed("tailscale_service.test_service", testService),
		},
	})
}

func TestAccTailscaleService(t *testing.T) {
	const resourceName = "tailscale_service.test_service"

	checkProperties := func(expectedComment string, expectedPorts []string, expectedTags []string) func(client *tailscale.Client, rs *terraform.ResourceState) error {
		return func(client *tailscale.Client, rs *terraform.ResourceState) error {
			svc, err := client.VIPServices().Get(context.Background(), rs.Primary.ID)
			if err != nil {
				return err
			}

			if svc.Comment != expectedComment {
				return fmt.Errorf("bad Service.comment: want %q, got %q", expectedComment, svc.Comment)
			}

			slices.Sort(expectedPorts)
			slices.Sort(svc.Ports)
			if err := assertEqual(expectedPorts, svc.Ports, "bad Service.ports"); err != nil {
				return err
			}

			slices.Sort(expectedTags)
			slices.Sort(svc.Tags)
			if err := assertEqual(expectedTags, svc.Tags, "bad Service.tags"); err != nil {
				return err
			}

			return nil
		}
	}

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV5ProviderFactories: testAccProviderFactories(t),
		CheckDestroy: checkResourceDestroyed(resourceName, func(client *tailscale.Client, rs *terraform.ResourceState) error {
			_, err := client.VIPServices().Get(context.Background(), rs.Primary.ID)
			if err == nil {
				return fmt.Errorf("Service %q still exists on server", resourceName)
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
							"tag:web": ["autogroup:member"],
							"tag:api": ["autogroup:member"]
						}
					}`, "")
					if err != nil {
						panic(err)
					}
				},
				Config: testService,
				Check: resource.ComposeTestCheckFunc(
					checkResourceRemoteProperties(resourceName,
						checkProperties("a test Service", []string{"tcp:443"}, []string{"tag:web"}),
					),
					resource.TestCheckResourceAttr(resourceName, "name", "svc:test-service"),
					resource.TestCheckResourceAttr(resourceName, "comment", "a test Service"),
					resource.TestCheckResourceAttr(resourceName, "ports.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "ports.0", "tcp:443"),
					resource.TestCheckResourceAttrSet(resourceName, "addrs.#"),
				),
			},
			{
				Config: testServiceUpdate,
				Check: resource.ComposeTestCheckFunc(
					checkResourceRemoteProperties(resourceName,
						checkProperties("an updated test Service", []string{"tcp:443", "tcp:8080"}, []string{"tag:api", "tag:web"}),
					),
					resource.TestCheckResourceAttr(resourceName, "name", "svc:test-service"),
					resource.TestCheckResourceAttr(resourceName, "comment", "an updated test Service"),
					resource.TestCheckResourceAttr(resourceName, "ports.#", "2"),
					resource.TestCheckTypeSetElemAttr(resourceName, "ports.*", "tcp:443"),
					resource.TestCheckTypeSetElemAttr(resourceName, "ports.*", "tcp:8080"),
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
