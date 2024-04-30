package tailscale_test

import (
	"net/http"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

const testSplitNameservers = `
	resource "tailscale_dns_split_nameservers" "test_nameservers" {
        domain = "example.com"
		nameservers = ["1.2.3.4", "4.5.6.7"]
	}`

func TestProvider_TailscaleSplitDNSNameservers(t *testing.T) {
	resource.Test(t, resource.TestCase{
		IsUnitTest: true,
		PreCheck: func() {
			testServer.ResponseCode = http.StatusOK
			testServer.ResponseBody = nil
		},
		ProviderFactories: testProviderFactories(t),
		Steps: []resource.TestStep{
			testResourceCreated("tailscale_dns_split_nameservers.test_nameservers", testSplitNameservers),
			testResourceDestroyed("tailscale_dns_split_nameservers.test_nameservers", testSplitNameservers),
		},
	})
}
