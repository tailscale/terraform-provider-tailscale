package tailscale_test

import (
	"context"
	"net/http"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"

	tsclient "github.com/tailscale/tailscale-client-go/v2"
)

const testSearchPathsCreate = `
	resource "tailscale_dns_search_paths" "test_search_paths" {
		search_paths = [
			"sub1.example.com",
			"sub2.example.com",
		]
	}`

const testSearchPathsUpdate = `
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
			testResourceCreated("tailscale_dns_search_paths.test_search_paths", testSearchPathsCreate),
			testResourceDestroyed("tailscale_dns_search_paths.test_search_paths", testSearchPathsCreate),
		},
	})
}

func TestAccTailscaleDNSSearchPaths(t *testing.T) {
	const resourceName = "tailscale_dns_search_paths.test_search_paths"

	checkProperties := func(expected []string) func(client *tsclient.Client, rs *terraform.ResourceState) error {
		return func(client *tsclient.Client, rs *terraform.ResourceState) error {
			actual, err := client.DNS().SearchPaths(context.Background())
			if err != nil {
				return err
			}

			if err := assertEqual(expected, actual, "wrong DNS search paths"); err != nil {
				return err
			}

			return nil
		}
	}

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories(t),
		CheckDestroy:      checkResourceDestroyed(resourceName, checkProperties([]string{})),
		Steps: []resource.TestStep{
			{
				Config: testSearchPathsCreate,
				Check: resource.ComposeTestCheckFunc(
					checkResourceRemoteProperties(resourceName,
						checkProperties([]string{"sub1.example.com", "sub2.example.com"}),
					),
					resource.TestCheckTypeSetElemAttr(resourceName, "search_paths.*", "sub1.example.com"),
					resource.TestCheckTypeSetElemAttr(resourceName, "search_paths.*", "sub2.example.com"),
				),
			},
			{
				Config: testSearchPathsUpdate,
				Check: resource.ComposeTestCheckFunc(
					checkResourceRemoteProperties(resourceName,
						checkProperties([]string{"example.com"}),
					),
					resource.TestCheckTypeSetElemAttr(resourceName, "search_paths.*", "example.com"),
				),
			},
		},
	})
}
