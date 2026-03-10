// Copyright (c) David Bond, Tailscale Inc, & Contributors
// SPDX-License-Identifier: MIT

package tailscale

import (
	"context"
	"fmt"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"

	"tailscale.com/client/tailscale/v2"

	"github.com/tailscale/hujson"
)

func TestAccTailscaleACL(t *testing.T) {
	resourceName := "data.tailscale_acl.acl"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV5ProviderFactories: testAccProviderFactories(t),
		Steps: []resource.TestStep{
			{
				Config: `data "tailscale_acl" "acl" {}`,
				Check: func(s *terraform.State) error {
					client := testAccProvider.Meta().(*tailscale.Client)
					acl, err := client.PolicyFile().Raw(context.Background())
					if err != nil {
						return fmt.Errorf("unable to get ACL: %s", err)
					}

					huj, err := hujson.Parse([]byte(acl.HuJSON))
					if err != nil {
						return fmt.Errorf("Failed to parse ACL as HuJSON: %s", err)
					}
					expected := huj.String()

					rs := s.RootModule().Resources[resourceName].Primary
					actual := rs.Attributes["hujson"]
					if err := assertEqual(expected, actual, "wrong ACL"); err != nil {
						return err
					}

					return nil
				},
			},
		},
	})
}

// Check that the data source behaves the same between the plugin SDK
// and the plugin framework.
//
// Note: we only check the `hujson` field because we fixed a bug in the
// `json` field while doing the migration; it was returning HuJSON rather
// than regular JSON.
func TestAccTailscaleACL_UpgradeToPluginFramework(t *testing.T) {
	checkDataSourceIsUnchangedInPluginFramework(t, `
		data "tailscale_acl" "acl" {}
                        
		resource "terraform_data" "hujson" {
			input = data.tailscale_acl.acl.hujson
		}`)
}

// TestToAclDataSourceModel checks that we convert a response from the
// Taislscale API to an [aclDataSourceModel] correctly.
func TestToAclDataSourceModel(t *testing.T) {
	hujson := `
		// This is a comment
		{
			"grants": [
				{"action": "accept", "src": ["*"], "dst": ["*:*"]},
			]
		}
	`
	json := `{"grants":[{"action":"accept","src":["*"],"dst":["*:*"]}]}`

	acl := tailscale.RawACL{HuJSON: hujson}
	data, diag := toAclDataSourceModel(&acl)
	if diag != nil {
		t.Fatalf("expected diag to be nil, got %v", diag)
	}
	if data == nil {
		t.Fatalf("expected to get data, got nil")
	}
	if diff := cmp.Diff(hujson, data.HuJSON.ValueString()); diff != "" {
		t.Fatalf("incorrect HuJSON (-want, +got):\n%s", diff)
	}
	if diff := cmp.Diff(json, data.JSON.ValueString()); diff != "" {
		t.Fatalf("incorrect JSON (-want, +got):\n%s", diff)
	}
}

// TestToAclDataSourceModelFailsIfInvalidHuJSON checks that if the Tailscale API
// returns invalid HuJSON, we can't convert it to an [aclDataSourceModel].
func TestToAclDataSourceModelFailsIfInvalidHuJSON(t *testing.T) {
	acl := tailscale.RawACL{
		HuJSON: "<summary>This is XML, not JSON</summary>",
	}

	data, diag := toAclDataSourceModel(&acl)
	if data != nil {
		t.Fatalf("expected data to be nil, got %v", data)
	}
	if diag == nil || diag.Summary() != "Failed to parse ACL as HuJSON" {
		t.Fatalf("expected diag to be a HuJSON parsing failure, got %v", data)
	}
}
