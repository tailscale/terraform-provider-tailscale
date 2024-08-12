package tailscale_test

import (
	"context"
	"net/http"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"

	tsclient "github.com/tailscale/tailscale-client-go/v2"
)

const testDNSPreferencesCreate = `
	resource "tailscale_dns_preferences" "test_preferences" {
		magic_dns = true
	}`

const testDNSPreferencesUpdate = `
	resource "tailscale_dns_preferences" "test_preferences" {
		magic_dns = false
	}`

func TestProvider_TailscaleDNSPreferences(t *testing.T) {
	resource.Test(t, resource.TestCase{
		IsUnitTest: true,
		PreCheck: func() {
			testServer.ResponseCode = http.StatusOK
			testServer.ResponseBody = nil
		},
		ProviderFactories: testProviderFactories(t),
		Steps: []resource.TestStep{
			testResourceCreated("tailscale_dns_preferences.test_preferences", testDNSPreferencesCreate),
			testResourceDestroyed("tailscale_dns_preferences.test_preferences", testDNSPreferencesCreate),
		},
	})
}

func TestAccTailscaleDNSPreferences(t *testing.T) {
	const resourceName = "tailscale_dns_preferences.test_preferences"

	checkProperties := func(expected *tsclient.DNSPreferences) func(client *tsclient.Client, rs *terraform.ResourceState) error {
		return func(client *tsclient.Client, rs *terraform.ResourceState) error {
			actual, err := client.DNS().Preferences(context.Background())
			if err != nil {
				return err
			}

			if diff := cmp.Diff(expected, actual); diff != "" {
				t.Fatalf("diff found (-got, +want): %s", diff)
			}

			return nil
		}
	}

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories(t),
		CheckDestroy:      checkResourceDestroyed(resourceName, checkProperties(&tsclient.DNSPreferences{})),
		Steps: []resource.TestStep{
			{
				Config: testDNSPreferencesCreate,
				Check: resource.ComposeTestCheckFunc(
					checkResourceRemoteProperties(resourceName,
						checkProperties(&tsclient.DNSPreferences{MagicDNS: true}),
					),
					resource.TestCheckResourceAttr(resourceName, "magic_dns", "true"),
				),
			},
			{
				Config: testDNSPreferencesUpdate,
				Check: resource.ComposeTestCheckFunc(
					checkResourceRemoteProperties(resourceName,
						checkProperties(&tsclient.DNSPreferences{MagicDNS: false}),
					),
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
