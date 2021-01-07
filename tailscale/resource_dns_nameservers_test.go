package tailscale_test

import (
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
		PreCheck:          func() { testProviderPreCheck(t) },
		ProviderFactories: providerFactories,
		Steps: []resource.TestStep{
			testResourceCreated("tailscale_dns_nameservers.test_nameservers", testNameservers),
			testResourceDestroyed("tailscale_dns_nameservers.test_nameservers", testNameservers),
		},
	})
}
