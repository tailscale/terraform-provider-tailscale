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

	"github.com/tailscale/tailscale-client-go/tailscale"
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
			testServer.ResponseBody = tailscale.Webhook{
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

func TestAccTailscaleWebhook_Basic(t *testing.T) {
	webhook := &tailscale.Webhook{}

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories(t),
		CheckDestroy:      testAccCheckWebhookDestroy,
		Steps: []resource.TestStep{
			{
				Config: testWebhook,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckWebhookExists("tailscale_webhook.test_webhook", webhook),
					testAccCheckWebhookProperties(webhook),
					resource.TestCheckResourceAttr("tailscale_webhook.test_webhook", "endpoint_url", "https://example.com/endpoint"),
					resource.TestCheckResourceAttr("tailscale_webhook.test_webhook", "provider_type", "slack"),
					resource.TestCheckTypeSetElemAttr("tailscale_webhook.test_webhook", "subscriptions.*", "userNeedsApproval"),
					resource.TestCheckTypeSetElemAttr("tailscale_webhook.test_webhook", "subscriptions.*", "nodeCreated"),
					resource.TestCheckResourceAttrSet("tailscale_webhook.test_webhook", "secret"),
				),
			},
			{
				ResourceName:            "tailscale_webhook.test_webhook",
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"secret"},
			},
		},
	})
}

func TestAccTailscaleWebhook_Update(t *testing.T) {
	webhook := &tailscale.Webhook{}

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories(t),
		CheckDestroy:      testAccCheckWebhookDestroy,
		Steps: []resource.TestStep{
			{
				Config: testWebhook,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckWebhookExists("tailscale_webhook.test_webhook", webhook),
					testAccCheckWebhookProperties(webhook),
					resource.TestCheckResourceAttr("tailscale_webhook.test_webhook", "endpoint_url", "https://example.com/endpoint"),
					resource.TestCheckResourceAttr("tailscale_webhook.test_webhook", "provider_type", "slack"),
					resource.TestCheckTypeSetElemAttr("tailscale_webhook.test_webhook", "subscriptions.*", "userNeedsApproval"),
					resource.TestCheckTypeSetElemAttr("tailscale_webhook.test_webhook", "subscriptions.*", "nodeCreated"),
					resource.TestCheckResourceAttrSet("tailscale_webhook.test_webhook", "secret"),
				),
			},
			{
				Config: testWebhookUpdate,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckWebhookExists("tailscale_webhook.test_webhook", webhook),
					testAccCheckWebhookPropertiesUpdated(webhook),
					resource.TestCheckResourceAttr("tailscale_webhook.test_webhook", "endpoint_url", "https://example.com/endpoint"),
					resource.TestCheckResourceAttr("tailscale_webhook.test_webhook", "provider_type", "slack"),
					resource.TestCheckTypeSetElemAttr("tailscale_webhook.test_webhook", "subscriptions.*", "nodeCreated"),
					resource.TestCheckTypeSetElemAttr("tailscale_webhook.test_webhook", "subscriptions.*", "userSuspended"),
					resource.TestCheckTypeSetElemAttr("tailscale_webhook.test_webhook", "subscriptions.*", "userRoleUpdated"),
					resource.TestCheckResourceAttrSet("tailscale_webhook.test_webhook", "secret"),
				),
			},
			{
				ResourceName:            "tailscale_webhook.test_webhook",
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"secret"},
			},
		},
	})
}

func testAccCheckWebhookExists(resourceName string, webhook *tailscale.Webhook) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[resourceName]
		if !ok {
			return fmt.Errorf("resource not found: %s", resourceName)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("resource has no ID set")
		}

		client := testAccProvider.Meta().(*tailscale.Client)
		out, err := client.Webhook(context.Background(), rs.Primary.ID)
		if err != nil {
			return err
		}

		*webhook = *out
		return nil
	}
}

func testAccCheckWebhookDestroy(s *terraform.State) error {
	client := testAccProvider.Meta().(*tailscale.Client)

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "tailscale_webhook" {
			continue
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("resource has no ID set")
		}

		_, err := client.Webhook(context.Background(), rs.Primary.ID)
		if err == nil {
			return fmt.Errorf("webhook %s still exists", rs.Primary.ID)
		}
	}
	return nil
}

func testAccCheckWebhookProperties(webhook *tailscale.Webhook) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		if webhook.EndpointURL != "https://example.com/endpoint" {
			return fmt.Errorf("bad webhook.endpoint_url: %s", webhook.EndpointURL)
		}
		if webhook.ProviderType != "slack" {
			return fmt.Errorf("bad webhook.provider_type: %s", webhook.ProviderType)
		}

		expectedSubscriptions := []tailscale.WebhookSubscriptionType{
			tailscale.WebhookNodeCreated,
			tailscale.WebhookUserNeedsApproval,
		}

		slices.Sort(expectedSubscriptions)
		slices.Sort(webhook.Subscriptions)

		if !reflect.DeepEqual(webhook.Subscriptions, expectedSubscriptions) {
			return fmt.Errorf("bad webhook.subscriptions: %#v", webhook.Subscriptions)
		}
		return nil
	}
}

func testAccCheckWebhookPropertiesUpdated(webhook *tailscale.Webhook) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		if webhook.EndpointURL != "https://example.com/endpoint" {
			return fmt.Errorf("bad webhook.endpoint_url: %s", webhook.EndpointURL)
		}
		if webhook.ProviderType != "slack" {
			return fmt.Errorf("bad webhook.provider_type: %s", webhook.ProviderType)
		}

		expectedSubscriptions := []tailscale.WebhookSubscriptionType{
			tailscale.WebhookNodeCreated,
			tailscale.WebhookUserRoleUpdated,
			tailscale.WebhookUserSuspended,
		}

		slices.Sort(expectedSubscriptions)
		slices.Sort(webhook.Subscriptions)

		if !reflect.DeepEqual(webhook.Subscriptions, expectedSubscriptions) {
			return fmt.Errorf("bad webhook.subscriptions: %#v", webhook.Subscriptions)
		}
		return nil
	}
}
