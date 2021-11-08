package tailscale_test

import (
	"net/http"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

const testDNSPreferences = `
	resource "tailscale_dns_preferences" "test_preferences" {
		magic_dns = true
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
			testResourceCreated("tailscale_dns_preferences.test_preferences", testDNSPreferences),
			testResourceDestroyed("tailscale_dns_preferences.test_preferences", testDNSPreferences),
		},
	})
}
