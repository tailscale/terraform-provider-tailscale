package tailscale_test

import (
	"net/http"
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
		IsUnitTest: true,
		PreCheck: func() {
			testServer.ResponseCode = http.StatusOK
			testServer.ResponseBody = nil
		},
		ProviderFactories: testProviderFactories(t),
		Steps: []resource.TestStep{
			testResourceCreated("tailscale_dns_search_paths.test_search_paths", testSearchPaths),
			testResourceDestroyed("tailscale_dns_search_paths.test_search_paths", testSearchPaths),
		},
	})
}
