// Copyright (c) David Bond, Tailscale Inc, & Contributors
// SPDX-License-Identifier: MIT

package tailscale

import (
	"fmt"
	"net/netip"
	"regexp"
	"strconv"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
	"tailscale.com/net/tsaddr"
)

const testDataSource4Via6 = `
data "tailscale_4via6" "example" {
  site = 7
  cidr = "10.1.1.0/24"
}
`

const testDataSource4Via6InvalidSite = `
data "tailscale_4via6" "invalid" {
	site = 70000
	cidr = "10.1.1.0/24"
}
`

func TestProvider_DataSourceTailscale4Via6(t *testing.T) {
	resource.ParallelTest(t, resource.TestCase{
		IsUnitTest:        true,
		ProviderFactories: testProviderFactories(t),
		Steps: []resource.TestStep{
			{
				Config: testDataSource4Via6,
				Check:  check4Via6Result("data.tailscale_4via6.example"),
			},
		},
	})
}

func TestProvider_DataSourceTailscale4Via6_InvalidSite(t *testing.T) {
	resource.ParallelTest(t, resource.TestCase{
		IsUnitTest:        true,
		ProviderFactories: testProviderFactories(t),
		Steps: []resource.TestStep{
			{
				Config:      testDataSource4Via6InvalidSite,
				ExpectError: regexp.MustCompile(`expected site to be in the range \(0 - 65535\), got 70000`),
			},
		},
	})
}

func check4Via6Result(n string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("can't find 4via6 resource: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("4via6 data source ID not set.")
		}

		siteAttr := rs.Primary.Attributes["site"]
		if siteAttr == "" {
			return fmt.Errorf("attribute site expected to not be nil")
		}

		site, err := strconv.ParseUint(siteAttr, 10, 32)
		if err != nil {
			return fmt.Errorf("invalid site ID %q: %s", siteAttr, err)
		}

		if site > 65535 {
			return fmt.Errorf("site ID %d is higher than the maximum allowed value of 65535", site)
		}

		cidrAttr := rs.Primary.Attributes["cidr"]
		if cidrAttr == "" {
			return fmt.Errorf("attribute cidr expected to not be nil")
		}

		cidr, err := netip.ParsePrefix(cidrAttr)
		if err != nil {
			return fmt.Errorf("invalid CIDR %q: %s", cidrAttr, err)
		}

		via, err := tsaddr.MapVia(uint32(site), cidr)
		if err != nil {
			return fmt.Errorf("failed to map 4via6: %s", err)
		}

		expected := via.String()
		if got := rs.Primary.Attributes["ipv6"]; expected != got {
			return fmt.Errorf("expected ipv6 to be %q but got %q", expected, got)
		}

		if expected != "fd7a:115c:a1e0:b1a:0:7:a01:100/120" {
			return fmt.Errorf("calculated %q, which is different than the value in Tailscale docs", expected)
		}

		return nil
	}
}
