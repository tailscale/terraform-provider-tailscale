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
	"github.com/tailscale/terraform-provider-tailscale/tailscale"
)

const testNameservers = `
	resource "tailscale_dns_nameservers" "test_nameservers" {
		nameservers = [
			"8.8.8.8",
			"8.8.4.4",
		]
	}`

const testNameserversUpdate = `
	resource "tailscale_dns_nameservers" "test_nameservers" {
		nameservers = [
			"8.8.8.8",
			"1.1.1.1",
		]
	}`

func TestProvider_TailscaleNameservers(t *testing.T) {
	resource.Test(t, resource.TestCase{
		IsUnitTest: true,
		PreCheck: func() {
			testServer.ResponseCode = http.StatusOK
			testServer.ResponseBody = nil
		},
		ProviderFactories: testProviderFactories(t),
		Steps: []resource.TestStep{
			testResourceCreated("tailscale_dns_nameservers.test_nameservers", testNameservers),
			testResourceDestroyed("tailscale_dns_nameservers.test_nameservers", testNameservers),
		},
	})
}

func TestAccTailscaleNameservers_Basic(t *testing.T) {
	Nameservers := []string{}

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories(t),
		CheckDestroy:      testAccCheckNameserversDestroy,
		Steps: []resource.TestStep{
			{
				Config: testNameservers,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckNameserversExists("tailscale_dns_nameservers.test_nameservers", Nameservers),
					testAccCheckNameserversProperties(Nameservers),
					resource.TestCheckResourceAttr("tailscale_dns_nameservers.test_nameservers", "nameservers", "https://example.com/endpoint"),
					resource.TestCheckResourceAttr("tailscale_dns_nameservers.test_nameservers", "provider_type", "slack"),
					resource.TestCheckTypeSetElemAttr("tailscale_dns_nameservers.test_nameservers", "subscriptions.*", "userNeedsApproval"),
					resource.TestCheckTypeSetElemAttr("tailscale_dns_nameservers.test_nameservers", "subscriptions.*", "nodeCreated"),
					resource.TestCheckResourceAttrSet("tailscale_dns_nameservers.test_nameservers", "secret"),
				),
			},
			{
				ResourceName:            "tailscale_dns_nameservers.test_nameservers",
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"secret"},
			},
		},
	})
}

func TestAccTailscaleNameservers_Update(t *testing.T) {
	Nameservers := &tsclient.Nameservers{}

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories(t),
		CheckDestroy:      testAccCheckNameserversDestroy,
		Steps: []resource.TestStep{
			{
				Config: testNameservers,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckNameserversExists("tailscale_dns_nameservers.test_nameservers", Nameservers),
					testAccCheckNameserversProperties(Nameservers),
					resource.TestCheckResourceAttr("tailscale_dns_nameservers.test_nameservers", "endpoint_url", "https://example.com/endpoint"),
					resource.TestCheckResourceAttr("tailscale_dns_nameservers.test_nameservers", "provider_type", "slack"),
					resource.TestCheckTypeSetElemAttr("tailscale_dns_nameservers.test_nameservers", "subscriptions.*", "userNeedsApproval"),
					resource.TestCheckTypeSetElemAttr("tailscale_dns_nameservers.test_nameservers", "subscriptions.*", "nodeCreated"),
					resource.TestCheckResourceAttrSet("tailscale_dns_nameservers.test_nameservers", "secret"),
				),
			},
			{
				Config: testNameserversUpdate,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckNameserversExists("tailscale_dns_nameservers.test_nameservers", Nameservers),
					testAccCheckNameserversPropertiesUpdated(Nameservers),
					resource.TestCheckResourceAttr("tailscale_dns_nameservers.test_nameservers", "endpoint_url", "https://example.com/endpoint"),
					resource.TestCheckResourceAttr("tailscale_dns_nameservers.test_nameservers", "provider_type", "slack"),
					resource.TestCheckTypeSetElemAttr("tailscale_dns_nameservers.test_nameservers", "subscriptions.*", "nodeCreated"),
					resource.TestCheckTypeSetElemAttr("tailscale_dns_nameservers.test_nameservers", "subscriptions.*", "userSuspended"),
					resource.TestCheckTypeSetElemAttr("tailscale_dns_nameservers.test_nameservers", "subscriptions.*", "userRoleUpdated"),
					resource.TestCheckResourceAttrSet("tailscale_dns_nameservers.test_nameservers", "secret"),
				),
			},
			{
				ResourceName:            "tailscale_dns_nameservers.test_nameservers",
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"secret"},
			},
		},
	})
}

func testAccCheckNameserversExists(resourceName string, Nameservers *tsclient.Nameservers) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[resourceName]
		if !ok {
			return fmt.Errorf("resource not found: %s", resourceName)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("resource has no ID set")
		}

		client := testAccProvider.Meta().(*tailscale.Clients).V2
		out, err := client.Nameserverss().Get(context.Background(), rs.Primary.ID)
		if err != nil {
			return err
		}

		*Nameservers = *out
		return nil
	}
}

func testAccCheckNameserversDestroy(s *terraform.State) error {
	client := testAccProvider.Meta().(*tailscale.Clients).V2

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "tailscale_Nameservers" {
			continue
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("resource has no ID set")
		}

		_, err := client.Nameserverss().Get(context.Background(), rs.Primary.ID)
		if err == nil {
			return fmt.Errorf("Nameservers %s still exists", rs.Primary.ID)
		}
	}
	return nil
}

func testAccCheckNameserversProperties(Nameservers *tsclient.Nameservers) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		if Nameservers.EndpointURL != "https://example.com/endpoint" {
			return fmt.Errorf("bad Nameservers.endpoint_url: %s", Nameservers.EndpointURL)
		}
		if Nameservers.ProviderType != "slack" {
			return fmt.Errorf("bad Nameservers.provider_type: %s", Nameservers.ProviderType)
		}

		expectedSubscriptions := []tsclient.NameserversSubscriptionType{
			tsclient.NameserversNodeCreated,
			tsclient.NameserversUserNeedsApproval,
		}

		slices.Sort(expectedSubscriptions)
		slices.Sort(Nameservers.Subscriptions)

		if !reflect.DeepEqual(Nameservers.Subscriptions, expectedSubscriptions) {
			return fmt.Errorf("bad Nameservers.subscriptions: %#v", Nameservers.Subscriptions)
		}
		return nil
	}
}

func testAccCheckNameserversPropertiesUpdated(Nameservers *tsclient.Nameservers) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		if Nameservers.EndpointURL != "https://example.com/endpoint" {
			return fmt.Errorf("bad Nameservers.endpoint_url: %s", Nameservers.EndpointURL)
		}
		if Nameservers.ProviderType != "slack" {
			return fmt.Errorf("bad Nameservers.provider_type: %s", Nameservers.ProviderType)
		}

		expectedSubscriptions := []tsclient.NameserversSubscriptionType{
			tsclient.NameserversNodeCreated,
			tsclient.NameserversUserRoleUpdated,
			tsclient.NameserversUserSuspended,
		}

		slices.Sort(expectedSubscriptions)
		slices.Sort(Nameservers.Subscriptions)

		if !reflect.DeepEqual(Nameservers.Subscriptions, expectedSubscriptions) {
			return fmt.Errorf("bad Nameservers.subscriptions: %#v", Nameservers.Subscriptions)
		}
		return nil
	}
}
