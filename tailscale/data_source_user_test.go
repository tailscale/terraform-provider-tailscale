// Copyright (c) David Bond, Tailscale Inc, & Contributors
// SPDX-License-Identifier: MIT

package tailscale

import (
	"regexp"
	"testing"
)

func TestProvider_DataSourceTailscaleUser_InvalidConfig(t *testing.T) {
	testCases := []expectedErrorTestCase{
		{
			Name:        "no-fields",
			Config:      `data "tailscale_user" "example" {}`,
			ExpectError: regexp.MustCompile(`No attribute specified when one \(and only one\) of \[id,login_name\] is required`),
		},
		{
			Name: "too-many-fields",
			Config: `
					data "tailscale_user" "example" {
						id 		   = "example"
						login_name = "example"
					}
				`,
			ExpectError: regexp.MustCompile(`2 attributes specified when one \(and only one\) of \[id,login_name\] is required`),
		},
	}

	runExpectedErrorTests(t, testCases)
}
