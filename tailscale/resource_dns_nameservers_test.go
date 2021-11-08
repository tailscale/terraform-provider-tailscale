package tailscale_test

import (
	"net/http"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

const testNameservers = `
	resource "tailscale_dns_nameservers" "test_nameservers" {
		nameservers = [
			"8.8.8.8",
			"8.8.4.4",
		]
	}`

func TestProvider_TailscaleDNSNameservers(t *testing.T) {
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
