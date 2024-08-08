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

func TestProvider_TailscaleWebhook(t *testing.T) {
	const testWebhook = `
		resource "tailscale_webhook" "test_webhook" {
			endpoint_url = "https://example.com/endpoint"
			provider_type = "slack"
			subscriptions = ["userNeedsApproval", "nodeCreated"]
		}`

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
	checkRemote := func(expectedSubscriptions []tsclient.WebhookSubscriptionType) func(c *tsclient.Client, rs *terraform.ResourceState) error {
		return func(c *tsclient.Client, rs *terraform.ResourceState) error {
			webhook, err := c.Webhooks().Get(context.Background(), rs.Primary.ID)
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

	acceptanceTest{
		resourceType: "tailscale_webhook",
		resourceName: "test_webhook",

		// Create
		initialValue: `{
			endpoint_url = "https://example.com/endpoint"
			provider_type = "slack"
			subscriptions = ["userNeedsApproval", "nodeCreated"]
		}`,
		createCheckRemoteProperties: checkRemote([]tsclient.WebhookSubscriptionType{
			tsclient.WebhookNodeCreated,
			tsclient.WebhookUserNeedsApproval,
		}),
		createCheckResourceAttributes: map[string]any{
			"endpoint_url":    "https://example.com/endpoint",
			"provider_type":   "slack",
			"subscriptions.*": []string{"userNeedsApproval", "nodeCreated"},
			"secret":          anyValue,
		},

		// Update
		updatedValue: `{
			endpoint_url = "https://example.com/endpoint"
			provider_type = "slack"
			subscriptions = ["nodeCreated", "userSuspended", "userRoleUpdated"]
		}`,
		updateCheckRemoteProperties: checkRemote([]tsclient.WebhookSubscriptionType{
			tsclient.WebhookNodeCreated,
			tsclient.WebhookUserRoleUpdated,
			tsclient.WebhookUserSuspended,
		}),
		updateCheckResourceAttributes: map[string]any{
			"endpoint_url":    "https://example.com/endpoint",
			"provider_type":   "slack",
			"subscriptions.*": []string{"nodeCreated", "userSuspended", "userRoleUpdated"},
			"secret":          anyValue,
		},
		checkRemoteDestroyed: func(c *tsclient.Client, rs *terraform.ResourceState) error {
			_, err := c.Webhooks().Get(context.Background(), rs.Primary.ID)
			if err == nil {
				return fmt.Errorf("webhook %s still exists", rs.Primary.ID)
			}
			return nil
		},
	}.run(t)
}
