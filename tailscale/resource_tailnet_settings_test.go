package tailscale

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"

	tsclient "github.com/tailscale/tailscale-client-go/v2"
)

func TestAccTailscaleTailnetSettings(t *testing.T) {
	const resourceName = "tailscale_tailnet_settings.test_settings"

	const testTailnetSettingsCreate = `
		resource "tailscale_tailnet_settings" "test_settings" {
			devices_approval_on = true
			devices_auto_updates_on = true
			devices_key_duration_days = 5
			users_approval_on = true
			users_role_allowed_to_join_external_tailnet = "member"
			posture_identity_collection_on = true
		}`

	const testTailnetSettingsUpdate = `
		resource "tailscale_tailnet_settings" "test_settings" {
			devices_approval_on = false
			devices_auto_updates_on = false
			devices_key_duration_days = 10
			users_approval_on = false
			users_role_allowed_to_join_external_tailnet = "admin"
			posture_identity_collection_on = false
		}`

	checkProperties := func(expected *tsclient.TailnetSettings) func(client *tsclient.Client, rs *terraform.ResourceState) error {
		return func(client *tsclient.Client, rs *terraform.ResourceState) error {
			actual, err := client.TailnetSettings().Get(context.Background())
			if err != nil {
				return err
			}

			if err := assertEqual(expected, actual, "wrong Tailnet settings"); err != nil {
				return err
			}

			return nil
		}
	}

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories(t),
		Steps: []resource.TestStep{
			{
				Config: testTailnetSettingsCreate,
				Check: resource.ComposeTestCheckFunc(
					checkResourceRemoteProperties(resourceName,
						checkProperties(&tsclient.TailnetSettings{
							DevicesApprovalOn:                      true,
							DevicesAutoUpdatesOn:                   true,
							DevicesKeyDurationDays:                 5,
							UsersApprovalOn:                        true,
							UsersRoleAllowedToJoinExternalTailnets: tsclient.RoleAllowedToJoinExternalTailnetsMember,
							PostureIdentityCollectionOn:            true,
						}),
					),
					resource.TestCheckResourceAttr(resourceName, "devices_approval_on", "true"),
					resource.TestCheckResourceAttr(resourceName, "devices_auto_updates_on", "true"),
					resource.TestCheckResourceAttr(resourceName, "devices_key_duration_days", "5"),
					resource.TestCheckResourceAttr(resourceName, "users_approval_on", "true"),
					resource.TestCheckResourceAttr(resourceName, "users_role_allowed_to_join_external_tailnet", "member"),
					resource.TestCheckResourceAttr(resourceName, "posture_identity_collection_on", "true"),
				),
			},
			{
				Config: testTailnetSettingsUpdate,
				Check: resource.ComposeTestCheckFunc(
					checkResourceRemoteProperties(resourceName,
						checkProperties(&tsclient.TailnetSettings{
							DevicesApprovalOn:                      false,
							DevicesAutoUpdatesOn:                   false,
							DevicesKeyDurationDays:                 10,
							UsersApprovalOn:                        false,
							UsersRoleAllowedToJoinExternalTailnets: tsclient.RoleAllowedToJoinExternalTailnetsAdmin,
							PostureIdentityCollectionOn:            false,
						}),
					),
					resource.TestCheckResourceAttr(resourceName, "devices_approval_on", "false"),
					resource.TestCheckResourceAttr(resourceName, "devices_auto_updates_on", "false"),
					resource.TestCheckResourceAttr(resourceName, "devices_key_duration_days", "10"),
					resource.TestCheckResourceAttr(resourceName, "users_approval_on", "false"),
					resource.TestCheckResourceAttr(resourceName, "users_role_allowed_to_join_external_tailnet", "admin"),
					resource.TestCheckResourceAttr(resourceName, "posture_identity_collection_on", "false"),
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
