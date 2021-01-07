package tailscale_test

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

const testSearchPaths = `
	resource "tailscale_dns_search_paths" "test_search_paths" {
		search_paths = [
			"example.com",
		]
	}`

func TestProvider_TailscaleDNSSearchPaths(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testProviderPreCheck(t) },
		ProviderFactories: providerFactories,
		Steps: []resource.TestStep{
			testResourceCreated("tailscale_dns_search_paths.test_search_paths", testSearchPaths),
			testResourceDestroyed("tailscale_dns_search_paths.test_search_paths", testSearchPaths),
		},
	})
}
