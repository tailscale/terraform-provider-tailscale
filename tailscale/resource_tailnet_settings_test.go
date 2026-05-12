// Copyright (c) David Bond, Tailscale Inc, & Contributors
// SPDX-License-Identifier: MIT

package tailscale

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"

	"tailscale.com/client/tailscale/v2"
)

func TestAccTailscaleTailnetSettings(t *testing.T) {
	const resourceName = "tailscale_tailnet_settings.test_settings"

	const testTailnetSettingsCreate = `
		resource "tailscale_tailnet_settings" "test_settings" {
			acls_externally_managed_on = true
			acls_external_link = "https://foo.com"
			devices_approval_on = true
			devices_auto_updates_on = true
			devices_key_duration_days = 5
			users_approval_on = true
			users_role_allowed_to_join_external_tailnet = "member"
			posture_identity_collection_on = true
			https_enabled = true
		}`

	const testTailnetSettingsUpdate = `
		resource "tailscale_tailnet_settings" "test_settings" {
			acls_externally_managed_on = false
			acls_external_link = "https://bar.com"
			devices_approval_on = false
			devices_auto_updates_on = false
			devices_key_duration_days = 10
			users_approval_on = false
			users_role_allowed_to_join_external_tailnet = "admin"
			posture_identity_collection_on = false
			https_enabled = false
		}`

	const testTailnetSettingsEmpty = `
		resource "tailscale_tailnet_settings" "test_settings" {
		}`

	const testTailnetSettingsOneSet = `
		resource "tailscale_tailnet_settings" "test_settings" {
			acls_externally_managed_on = true
		}`

	checkProperties := func(expected *tailscale.TailnetSettings) func(client *tailscale.Client, rs *terraform.ResourceState) error {
		return func(client *tailscale.Client, rs *terraform.ResourceState) error {
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
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV5ProviderFactories: testAccProviderFactories(t),
		Steps: []resource.TestStep{
			{
				Config: testTailnetSettingsCreate,
				Check: resource.ComposeTestCheckFunc(
					checkResourceRemoteProperties(resourceName,
						checkProperties(&tailscale.TailnetSettings{
							ACLsExternallyManagedOn:                true,
							ACLsExternalLink:                       "https://foo.com",
							DevicesApprovalOn:                      true,
							DevicesAutoUpdatesOn:                   true,
							DevicesKeyDurationDays:                 5,
							UsersApprovalOn:                        true,
							UsersRoleAllowedToJoinExternalTailnets: tailscale.RoleAllowedToJoinExternalTailnetsMember,
							PostureIdentityCollectionOn:            true,
							HTTPSEnabled:                           true,
						}),
					),
					resource.TestCheckResourceAttr(resourceName, "acls_externally_managed_on", "true"),
					resource.TestCheckResourceAttr(resourceName, "acls_external_link", "https://foo.com"),
					resource.TestCheckResourceAttr(resourceName, "devices_approval_on", "true"),
					resource.TestCheckResourceAttr(resourceName, "devices_auto_updates_on", "true"),
					resource.TestCheckResourceAttr(resourceName, "devices_key_duration_days", "5"),
					resource.TestCheckResourceAttr(resourceName, "users_approval_on", "true"),
					resource.TestCheckResourceAttr(resourceName, "users_role_allowed_to_join_external_tailnet", "member"),
					resource.TestCheckResourceAttr(resourceName, "posture_identity_collection_on", "true"),
					resource.TestCheckResourceAttr(resourceName, "https_enabled", "true"),
				),
			},
			{
				Config: testTailnetSettingsUpdate,
				Check: resource.ComposeTestCheckFunc(
					checkResourceRemoteProperties(resourceName,
						checkProperties(&tailscale.TailnetSettings{
							ACLsExternallyManagedOn:                false,
							ACLsExternalLink:                       "https://bar.com",
							DevicesApprovalOn:                      false,
							DevicesAutoUpdatesOn:                   false,
							DevicesKeyDurationDays:                 10,
							UsersApprovalOn:                        false,
							UsersRoleAllowedToJoinExternalTailnets: tailscale.RoleAllowedToJoinExternalTailnetsAdmin,
							PostureIdentityCollectionOn:            false,
							HTTPSEnabled:                           false,
						}),
					),
					resource.TestCheckResourceAttr(resourceName, "acls_externally_managed_on", "false"),
					resource.TestCheckResourceAttr(resourceName, "acls_external_link", "https://bar.com"),
					resource.TestCheckResourceAttr(resourceName, "devices_approval_on", "false"),
					resource.TestCheckResourceAttr(resourceName, "devices_auto_updates_on", "false"),
					resource.TestCheckResourceAttr(resourceName, "devices_key_duration_days", "10"),
					resource.TestCheckResourceAttr(resourceName, "users_approval_on", "false"),
					resource.TestCheckResourceAttr(resourceName, "users_role_allowed_to_join_external_tailnet", "admin"),
					resource.TestCheckResourceAttr(resourceName, "posture_identity_collection_on", "false"),
					resource.TestCheckResourceAttr(resourceName, "https_enabled", "false"),
				),
			},
			{
				Config: testTailnetSettingsEmpty,
				Check: resource.ComposeTestCheckFunc(
					checkResourceRemoteProperties(resourceName,
						checkProperties(&tailscale.TailnetSettings{
							ACLsExternallyManagedOn:                false,
							ACLsExternalLink:                       "https://bar.com",
							DevicesApprovalOn:                      false,
							DevicesAutoUpdatesOn:                   false,
							DevicesKeyDurationDays:                 10,
							UsersApprovalOn:                        false,
							UsersRoleAllowedToJoinExternalTailnets: tailscale.RoleAllowedToJoinExternalTailnetsAdmin,
							PostureIdentityCollectionOn:            false,
							HTTPSEnabled:                           false,
						}),
					),
					resource.TestCheckResourceAttr(resourceName, "acls_externally_managed_on", "false"),
					resource.TestCheckResourceAttr(resourceName, "acls_external_link", "https://bar.com"),
					resource.TestCheckResourceAttr(resourceName, "devices_approval_on", "false"),
					resource.TestCheckResourceAttr(resourceName, "devices_auto_updates_on", "false"),
					resource.TestCheckResourceAttr(resourceName, "devices_key_duration_days", "10"),
					resource.TestCheckResourceAttr(resourceName, "users_approval_on", "false"),
					resource.TestCheckResourceAttr(resourceName, "users_role_allowed_to_join_external_tailnet", "admin"),
					resource.TestCheckResourceAttr(resourceName, "posture_identity_collection_on", "false"),
					resource.TestCheckResourceAttr(resourceName, "https_enabled", "false"),
				),
			},
			{
				Config: testTailnetSettingsOneSet,
				Check: resource.ComposeTestCheckFunc(
					checkResourceRemoteProperties(resourceName,
						checkProperties(&tailscale.TailnetSettings{
							ACLsExternallyManagedOn:                true,
							ACLsExternalLink:                       "https://bar.com",
							DevicesApprovalOn:                      false,
							DevicesAutoUpdatesOn:                   false,
							DevicesKeyDurationDays:                 10,
							UsersApprovalOn:                        false,
							UsersRoleAllowedToJoinExternalTailnets: tailscale.RoleAllowedToJoinExternalTailnetsAdmin,
							PostureIdentityCollectionOn:            false,
							HTTPSEnabled:                           false,
						}),
					),
					resource.TestCheckResourceAttr(resourceName, "acls_externally_managed_on", "true"),
					resource.TestCheckResourceAttr(resourceName, "acls_external_link", "https://bar.com"),
					resource.TestCheckResourceAttr(resourceName, "devices_approval_on", "false"),
					resource.TestCheckResourceAttr(resourceName, "devices_auto_updates_on", "false"),
					resource.TestCheckResourceAttr(resourceName, "devices_key_duration_days", "10"),
					resource.TestCheckResourceAttr(resourceName, "users_approval_on", "false"),
					resource.TestCheckResourceAttr(resourceName, "users_role_allowed_to_join_external_tailnet", "admin"),
					resource.TestCheckResourceAttr(resourceName, "posture_identity_collection_on", "false"),
					resource.TestCheckResourceAttr(resourceName, "https_enabled", "false"),
				),
			},
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})

	// Test that the resource remains unchanged when upgrading
	checkResourceIsUnchangedInPluginFramework(t,
		testTailnetSettingsCreate,
		resource.ComposeTestCheckFunc(
			checkResourceRemoteProperties(resourceName,
				checkProperties(&tailscale.TailnetSettings{
					ACLsExternallyManagedOn:                true,
					ACLsExternalLink:                       "https://foo.com",
					DevicesApprovalOn:                      true,
					DevicesAutoUpdatesOn:                   true,
					DevicesKeyDurationDays:                 5,
					UsersApprovalOn:                        true,
					UsersRoleAllowedToJoinExternalTailnets: tailscale.RoleAllowedToJoinExternalTailnetsMember,
					PostureIdentityCollectionOn:            true,
					HTTPSEnabled:                           true,
				}),
			),
			resource.TestCheckResourceAttr(resourceName, "acls_externally_managed_on", "true"),
			resource.TestCheckResourceAttr(resourceName, "acls_external_link", "https://foo.com"),
			resource.TestCheckResourceAttr(resourceName, "devices_approval_on", "true"),
			resource.TestCheckResourceAttr(resourceName, "devices_auto_updates_on", "true"),
			resource.TestCheckResourceAttr(resourceName, "devices_key_duration_days", "5"),
			resource.TestCheckResourceAttr(resourceName, "users_approval_on", "true"),
			resource.TestCheckResourceAttr(resourceName, "users_role_allowed_to_join_external_tailnet", "member"),
			resource.TestCheckResourceAttr(resourceName, "posture_identity_collection_on", "true"),
			resource.TestCheckResourceAttr(resourceName, "https_enabled", "true"),
		))

	resource.Test(t, resource.TestCase{
		Steps: []resource.TestStep{
			{
				ExternalProviders: map[string]resource.ExternalProvider{
					"tailscale": {
						VersionConstraint: "0.28.0",
						Source:            "tailscale/tailscale",
					},
				},
				PreConfig: func() {
					client := getAccTestClient()

					// Set all these optional fields to true / something, so if the
					// unset value is misinterpreted as the empty value, a change will be
					// made and the test will fail.
					settingsRequest := tailscale.UpdateTailnetSettingsRequest{
						ACLsExternallyManagedOn:                new(true),
						ACLsExternalLink:                       new("https://foo.com"),
						DevicesApprovalOn:                      new(true),
						DevicesAutoUpdatesOn:                   new(true),
						DevicesKeyDurationDays:                 new(5),
						HTTPSEnabled:                           new(true),
						NetworkFlowLoggingOn:                   new(true),
						PostureIdentityCollectionOn:            new(true),
						RegionalRoutingOn:                      new(true),
						UsersApprovalOn:                        new(true),
						UsersRoleAllowedToJoinExternalTailnets: new(tailscale.RoleAllowedToJoinExternalTailnets("member")),
					}
					err := client.TailnetSettings().Update(context.Background(), settingsRequest)
					if err != nil {
						panic(err)
					}
				},
				Config: testTailnetSettingsEmpty,
			},
			{
				ProtoV5ProviderFactories: testAccProviderFactories(t),
				Config:                   testTailnetSettingsEmpty,
				PlanOnly:                 true,
				Check: resource.ComposeTestCheckFunc(
					checkResourceRemoteProperties(resourceName,
						checkProperties(&tailscale.TailnetSettings{
							ACLsExternallyManagedOn:                true,
							ACLsExternalLink:                       "https://foo.com",
							DevicesApprovalOn:                      true,
							DevicesAutoUpdatesOn:                   true,
							DevicesKeyDurationDays:                 5,
							HTTPSEnabled:                           true,
							NetworkFlowLoggingOn:                   true,
							PostureIdentityCollectionOn:            true,
							RegionalRoutingOn:                      true,
							UsersApprovalOn:                        true,
							UsersRoleAllowedToJoinExternalTailnets: "member",
						}),
					),
				),
			},
		},
	})

}
