// Copyright (c) David Bond, Tailscale Inc, & Contributors
// SPDX-License-Identifier: MIT

package tailscale

import (
	"context"
	"fmt"
	"net/http"
	"reflect"
	"slices"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"

	"tailscale.com/client/tailscale/v2"
)

const testWebhookCreate = `
	resource "tailscale_webhook" "test_webhook" {
		endpoint_url = "https://example.com/endpoint"
		provider_type = "slack"
		subscriptions = ["userNeedsApproval", "nodeCreated"]
	}`

const testWebhookUpdate = `
	resource "tailscale_webhook" "test_webhook" {
		endpoint_url = "https://example.com/endpoint"
		provider_type = "slack"
		subscriptions = ["nodeCreated", "userSuspended", "userRoleUpdated"]
	}`

const testWebhookNoProvider = `
	resource "tailscale_webhook" "test_webhook" {
		endpoint_url = "https://example.com/endpoint"
		subscriptions = ["userNeedsApproval", "nodeCreated"]
	}`

func TestProvider_TailscaleWebhook(t *testing.T) {
	resource.Test(t, resource.TestCase{
		IsUnitTest: true,
		PreCheck: func() {
			testServer.ResponseCode = http.StatusOK
			testServer.ResponseBody = tailscale.Webhook{
				EndpointID: "12345",
				Secret:     new("123"),
			}
		},
		ProtoV5ProviderFactories: testProviderFactories(t),
		Steps: []resource.TestStep{
			testResourceCreated("tailscale_webhook.test_webhook", testWebhookCreate),
			testResourceDestroyed("tailscale_webhook.test_webhook", testWebhookCreate),
		},
	})
}

// TestProvider_TailscaleWebhookDeletedOutsideTerraform checks that a webhook
// that no longer exists is removed from state on refresh, and that deleting a
// webhook that is already gone succeeds.
func TestProvider_TailscaleWebhookDeletedOutsideTerraform(t *testing.T) {
	webhook := tailscale.Webhook{
		EndpointID:    "12345",
		EndpointURL:   "https://example.com/endpoint",
		ProviderType:  tailscale.WebhookSlackProviderType,
		Subscriptions: []tailscale.WebhookSubscriptionType{"userNeedsApproval", "nodeCreated"},
		Secret:        new("123"),
	}

	resource.Test(t, resource.TestCase{
		IsUnitTest:               true,
		ProtoV5ProviderFactories: testProviderFactories(t),
		Steps: []resource.TestStep{
			{
				PreConfig: func() {
					testServer.ResponseCode = http.StatusOK
					testServer.ResponseBody = webhook
				},
				Config: testWebhookCreate,
			},
			{
				PreConfig: func() {
					testServer.ResponseCode = http.StatusNotFound
					testServer.ResponseBody = nil
				},
				Config:             testWebhookCreate,
				PlanOnly:           true,
				ExpectNonEmptyPlan: true,
			},
			{
				PreConfig: func() {
					testServer.HandleRequest = func(method, _ string) TestResponse {
						if method == http.MethodDelete {
							return TestResponse{Code: http.StatusNotFound}
						}
						return TestResponse{Code: http.StatusOK, Body: webhook}
					}
				},
				Config:  testWebhookCreate,
				Destroy: true,
			},
		},
	})
}

func TestAccTailscaleWebhook(t *testing.T) {
	const resourceName = "tailscale_webhook.test_webhook"

	checkProperties := func(expectedSubscriptions []tailscale.WebhookSubscriptionType) func(client *tailscale.Client, rs *terraform.ResourceState) error {
		return func(client *tailscale.Client, rs *terraform.ResourceState) error {
			webhook, err := client.Webhooks().Get(context.Background(), rs.Primary.ID)
			if err != nil {
				return err
			}

			if webhook.EndpointURL != "https://example.com/endpoint" {
				return fmt.Errorf("bad webhook.endpoint_url: %s", webhook.EndpointURL)
			}
			if webhook.ProviderType != "slack" {
				return fmt.Errorf("bad webhook.provider_type: %s", webhook.ProviderType)
			}

			slices.Sort(expectedSubscriptions)
			slices.Sort(webhook.Subscriptions)

			if !reflect.DeepEqual(webhook.Subscriptions, expectedSubscriptions) {
				return fmt.Errorf("bad webhook.subscriptions: %#v", webhook.Subscriptions)
			}
			return nil
		}
	}

	createCheck := resource.ComposeTestCheckFunc(
		checkResourceRemoteProperties(
			resourceName,
			checkProperties([]tailscale.WebhookSubscriptionType{
				tailscale.WebhookNodeCreated,
				tailscale.WebhookUserNeedsApproval,
			}),
		),
		resource.TestCheckResourceAttr(resourceName, "endpoint_url", "https://example.com/endpoint"),
		resource.TestCheckResourceAttr(resourceName, "provider_type", "slack"),
		resource.TestCheckTypeSetElemAttr(resourceName, "subscriptions.*", "userNeedsApproval"),
		resource.TestCheckTypeSetElemAttr(resourceName, "subscriptions.*", "nodeCreated"),
		resource.TestCheckResourceAttrSet(resourceName, "secret"),
	)

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV5ProviderFactories: testAccProviderFactories(t),
		CheckDestroy: checkResourceDestroyed(resourceName, func(client *tailscale.Client, rs *terraform.ResourceState) error {
			_, err := client.Webhooks().Get(context.Background(), rs.Primary.ID)
			if err == nil {
				return fmt.Errorf("webhook %q still exists on server", resourceName)
			}
			return nil
		}),
		Steps: []resource.TestStep{
			{
				Config: testWebhookCreate,
				Check:  createCheck,
			},
			{
				Config: testWebhookUpdate,
				Check: resource.ComposeTestCheckFunc(
					checkResourceRemoteProperties(
						resourceName,
						checkProperties([]tailscale.WebhookSubscriptionType{
							tailscale.WebhookNodeCreated,
							tailscale.WebhookUserRoleUpdated,
							tailscale.WebhookUserSuspended,
						}),
					),
					resource.TestCheckResourceAttr(resourceName, "endpoint_url", "https://example.com/endpoint"),
					resource.TestCheckResourceAttr(resourceName, "provider_type", "slack"),
					resource.TestCheckTypeSetElemAttr(resourceName, "subscriptions.*", "nodeCreated"),
					resource.TestCheckTypeSetElemAttr(resourceName, "subscriptions.*", "userSuspended"),
					resource.TestCheckTypeSetElemAttr(resourceName, "subscriptions.*", "userRoleUpdated"),
					resource.TestCheckResourceAttrSet(resourceName, "secret"),
				),
			},
			{
				ResourceName:            resourceName,
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"secret"},
			},
		},
	})

	// Migration test to ensure the resource is unchanged when migrating
	// from the plugin SDK to the plugin framework.
	//
	// See https://developer.hashicorp.com/terraform/plugin/framework/migrating/testing#terraform-data-resource-example
	checkResourceIsUnchangedInPluginFramework(t, testWebhookCreate, createCheck)
}

// TestAccTailscaleWebhookOptionalFields checks that we can round-trip set /
// update / read the provider field, which is an optional string, without
// getting complaints that it is changing between null and StringVal("").
func TestAccTailscaleWebhookOptionalFields(t *testing.T) {
	const resourceName = "tailscale_webhook.test_webhook"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV5ProviderFactories: testAccProviderFactories(t),
		CheckDestroy: checkResourceDestroyed(resourceName, func(client *tailscale.Client, rs *terraform.ResourceState) error {
			_, err := client.Webhooks().Get(context.Background(), rs.Primary.ID)
			if err == nil {
				return fmt.Errorf("webhook %q still exists on server", resourceName)
			}
			return nil
		}),
		Steps: []resource.TestStep{
			{
				Config: testWebhookNoProvider,
			},
		},
	})

	checkResourceIsUnchangedInPluginFramework(t, testWebhookNoProvider, nil)
}
