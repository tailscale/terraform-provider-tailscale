package tailscale_test

import (
	"context"
	"fmt"
	"net/http"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"

	tsclient "github.com/tailscale/tailscale-client-go/v2"
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

func TestAccTailscaleDNSSplitNameservers(t *testing.T) {
	const resourceName = "tailscale_dns_split_nameservers.test_nameservers"

	const testSplitNameserversCreate = `
		resource "tailscale_dns_split_nameservers" "test_nameservers" {
			domain = "example.com"
			nameservers = ["1.2.3.4", "4.5.6.7"]
		}`

	const testSplitNameserversUpdate = `
		resource "tailscale_dns_split_nameservers" "test_nameservers" {
			domain = "sub.example.com"
			nameservers = ["8.8.9.9"]
		}`

	const testSplitNameserversUpdateSameDomain = `
		resource "tailscale_dns_split_nameservers" "test_nameservers" {
			domain = "sub.example.com"
			nameservers = ["8.8.7.7", "8.8.9.9"]
		}`

	checkProperties := func(expected tsclient.SplitDNSResponse) func(client *tsclient.Client, rs *terraform.ResourceState) error {
		return func(client *tsclient.Client, rs *terraform.ResourceState) error {
			actual, err := client.DNS().SplitDNS(context.Background())
			if err != nil {
				return err
			}

			if diff := cmp.Diff(actual, expected); diff != "" {
				return fmt.Errorf("wrong split dns: (-got+want) \n%s", diff)
			}

			return nil
		}
	}

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories(t),
		CheckDestroy:      checkResourceDestroyed(resourceName, checkProperties(tsclient.SplitDNSResponse{})),
		Steps: []resource.TestStep{
			{
				Config: testSplitNameserversCreate,
				Check: resource.ComposeTestCheckFunc(
					checkResourceRemoteProperties(resourceName,
						checkProperties(tsclient.SplitDNSResponse{
							"example.com": []string{"1.2.3.4", "4.5.6.7"},
						}),
					),
					resource.TestCheckResourceAttr(resourceName, "domain", "example.com"),
					resource.TestCheckTypeSetElemAttr(resourceName, "nameservers.*", "1.2.3.4"),
					resource.TestCheckTypeSetElemAttr(resourceName, "nameservers.*", "4.5.6.7"),
				),
			},
			{
				Config: testSplitNameserversUpdate,
				Check: resource.ComposeTestCheckFunc(
					checkResourceRemoteProperties(resourceName,
						checkProperties(tsclient.SplitDNSResponse{
							"sub.example.com": []string{"8.8.9.9"},
						}),
					),
					resource.TestCheckResourceAttr(resourceName, "domain", "sub.example.com"),
					resource.TestCheckTypeSetElemAttr(resourceName, "nameservers.*", "8.8.9.9"),
				),
			},
			{
				Config: testSplitNameserversUpdateSameDomain,
				Check: resource.ComposeTestCheckFunc(
					checkResourceRemoteProperties(resourceName,
						checkProperties(tsclient.SplitDNSResponse{
							"sub.example.com": []string{"8.8.7.7", "8.8.9.9"},
						}),
					),
					resource.TestCheckResourceAttr(resourceName, "domain", "sub.example.com"),
					resource.TestCheckTypeSetElemAttr(resourceName, "nameservers.*", "8.8.7.7"),
					resource.TestCheckTypeSetElemAttr(resourceName, "nameservers.*", "8.8.9.9"),
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
