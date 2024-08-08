package tailscale_test

import (
	"context"
	"fmt"
	"net/http"
	"reflect"
	"slices"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"

	tsclient "github.com/tailscale/tailscale-client-go/v2"
)

const testWebhook = `
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

func TestProvider_TailscaleWebhook(t *testing.T) {
	resource.Test(t, resource.TestCase{
		IsUnitTest: true,
		PreCheck: func() {
			testServer.ResponseCode = http.StatusOK
			testServer.ResponseBody = tsclient.Webhook{
				EndpointID: "12345",
			}
		},
		ProviderFactories: testProviderFactories(t),
		Steps: []resource.TestStep{
			testResourceCreated("tailscale_webhook.test_webhook", testWebhook),
			testResourceDestroyed("tailscale_webhook.test_webhook", testWebhook),
		},
	})
}

func TestAccTailscaleWebhook(t *testing.T) {
	const resourceName = "tailscale_webhook.test_webhook"

	checkProperties := func(expectedSubscriptions []tsclient.WebhookSubscriptionType) func(client *tsclient.Client, rs *terraform.ResourceState) error {
		return func(client *tsclient.Client, rs *terraform.ResourceState) error {
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

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories(t),
		CheckDestroy: checkResourceDestroyed(resourceName, func(client *tsclient.Client, rs *terraform.ResourceState) (*tsclient.Webhook, error) {
			return client.Webhooks().Get(context.Background(), rs.Primary.ID)
		}),
		Steps: []resource.TestStep{
			{
				Config: testWebhook,
				Check: resource.ComposeTestCheckFunc(
					checkResourceRemoteProperties(
						resourceName,
						checkProperties([]tsclient.WebhookSubscriptionType{
							tsclient.WebhookNodeCreated,
							tsclient.WebhookUserNeedsApproval,
						}),
					),
					resource.TestCheckResourceAttr(resourceName, "endpoint_url", "https://example.com/endpoint"),
					resource.TestCheckResourceAttr(resourceName, "provider_type", "slack"),
					resource.TestCheckTypeSetElemAttr(resourceName, "subscriptions.*", "userNeedsApproval"),
					resource.TestCheckTypeSetElemAttr(resourceName, "subscriptions.*", "nodeCreated"),
					resource.TestCheckResourceAttrSet(resourceName, "secret"),
				),
			},
			{
				Config: testWebhookUpdate,
				Check: resource.ComposeTestCheckFunc(
					checkResourceRemoteProperties(
						resourceName,
						checkProperties([]tsclient.WebhookSubscriptionType{
							tsclient.WebhookNodeCreated,
							tsclient.WebhookUserRoleUpdated,
							tsclient.WebhookUserSuspended,
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
}
