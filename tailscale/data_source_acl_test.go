package tailscale

import (
	"context"
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"

	"github.com/tailscale/hujson"
)

func TestAccTailscaleACL(t *testing.T) {
	resourceName := "data.tailscale_acl.acl"

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories(t),
		Steps: []resource.TestStep{
			{
				Config: `data "tailscale_acl" "acl" {}`,
				Check: func(s *terraform.State) error {
					client := testAccProvider.Meta().(*Clients).V2
					acl, err := client.PolicyFile().Raw(context.Background())
					if err != nil {
						return fmt.Errorf("unable to get ACL: %s", err)
					}

					huj, err := hujson.Parse([]byte(acl))
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
