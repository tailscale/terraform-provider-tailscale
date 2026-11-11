// Copyright (c) David Bond, Tailscale Inc, & Contributors
// SPDX-License-Identifier: MIT

package tailscale

import (
	"context"
	"encoding/json"
	"net/http"
	"net/url"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/types"
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
			network_flow_logging_on = false
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
			route_selection = "magicroute"
		}`

	const testTailnetSettingsEmpty = `
		resource "tailscale_tailnet_settings" "test_settings" {
		}`

	const testTailnetSettingsOneSet = `
		resource "tailscale_tailnet_settings" "test_settings" {
			acls_externally_managed_on = true
		}`

	// regional_routing_on and route_selection are two views of a single setting. These
	// configs set exactly one of the two at a time and are expected to pass.
	const testTailnetSettingsRouteSelectionRegional = `
		resource "tailscale_tailnet_settings" "test_settings" {
			route_selection = "regional-routing"
		}`
	const testTailnetSettingsRegionalRoutingOn = `
		resource "tailscale_tailnet_settings" "test_settings" {
			regional_routing_on = true
		}`

	// Restores the route selection left by testTailnetSettingsUpdate, so that the steps
	// above are self-contained and the checks that follow this test case still hold.
	const testTailnetSettingsRouteSelectionMagicRoute = `
		resource "tailscale_tailnet_settings" "test_settings" {
			route_selection = "magicroute"
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

	// So we can run this suite multiple times using the same devcontrol fixture
	resetTailnetSettings := func(t *testing.T) {
		t.Helper()
		client := getAccTestClient()
		err := client.TailnetSettings().Update(context.Background(), tailscale.UpdateTailnetSettingsRequest{
			ACLsExternallyManagedOn:                new(false),
			ACLsExternalLink:                       new(""),
			DevicesApprovalOn:                      new(false),
			DevicesAutoUpdatesOn:                   new(false),
			DevicesKeyDurationDays:                 new(180),
			UsersApprovalOn:                        new(false),
			UsersRoleAllowedToJoinExternalTailnets: new(tailscale.RoleAllowedToJoinExternalTailnetsNone),
			NetworkFlowLoggingOn:                   new(false),
			PostureIdentityCollectionOn:            new(false),
			HTTPSEnabled:                           new(false),
			RouteSelection:                         new(tailscale.RouteSelectionActivePassiveFailover),
		})
		if err != nil {
			t.Fatalf("resetting tailnet settings: %s", err)
		}
	}

	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
			resetTailnetSettings(t)
		},
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
							RouteSelection:                         tailscale.RouteSelectionActivePassiveFailover,
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
					resource.TestCheckResourceAttr(resourceName, "route_selection", "active-passive-failover"),
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
							RouteSelection:                         tailscale.RouteSelectionMagicRoute,
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
					resource.TestCheckResourceAttr(resourceName, "route_selection", "magicroute"),
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
							RouteSelection:                         tailscale.RouteSelectionMagicRoute,
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
					resource.TestCheckResourceAttr(resourceName, "route_selection", "magicroute"),
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
							RouteSelection:                         tailscale.RouteSelectionMagicRoute,
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
					resource.TestCheckResourceAttr(resourceName, "route_selection", "magicroute"),
				),
			},
			{
				// Changing only route_selection must not send the derived
				// regional_routing_on alongside it, and must pass.
				Config: testTailnetSettingsRouteSelectionRegional,
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
							RegionalRoutingOn:                      true,
							RouteSelection:                         tailscale.RouteSelectionRegionalRouting,
						}),
					),
					resource.TestCheckResourceAttr(resourceName, "route_selection", "regional-routing"),
					resource.TestCheckResourceAttr(resourceName, "regional_routing_on", "true"),
				),
			},
			{
				// route_selection moving between two non-default values.
				Config: testTailnetSettingsRouteSelectionMagicRoute,
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
							RegionalRoutingOn:                      false,
							RouteSelection:                         tailscale.RouteSelectionMagicRoute,
						}),
					),
					resource.TestCheckResourceAttr(resourceName, "route_selection", "magicroute"),
					resource.TestCheckResourceAttr(resourceName, "regional_routing_on", "false"),
				),
			},
			{
				// The deprecated attribute still works, and updates the derived route_selection.
				Config: testTailnetSettingsRegionalRoutingOn,
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
							RegionalRoutingOn:                      true,
							RouteSelection:                         tailscale.RouteSelectionRegionalRouting,
						}),
					),
					resource.TestCheckResourceAttr(resourceName, "regional_routing_on", "true"),
					resource.TestCheckResourceAttr(resourceName, "route_selection", "regional-routing"),
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
					RegionalRoutingOn:                      true,
					RouteSelection:                         tailscale.RouteSelectionRegionalRouting,
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
			resource.TestCheckResourceAttr(resourceName, "regional_routing_on", "true"),
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
						UsersApprovalOn:                        new(true),
						UsersRoleAllowedToJoinExternalTailnets: new(tailscale.RoleAllowedToJoinExternalTailnets("member")),
						RouteSelection:                         new(tailscale.RouteSelectionRegionalRouting),
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
							RouteSelection:                         tailscale.RouteSelectionRegionalRouting,
						}),
					),
				),
			},
		},
	})

}

// TestProvider_ResourceTailscaleTailnetSettings_InvalidConfig asserts that we reject
// setting both regional_routing_on and route_selection in the same configuration.
func TestProvider_ResourceTailscaleTailnetSettings_InvalidConfig(t *testing.T) {
	const wantErr = `These attributes cannot be configured together:\s\[regional_routing_on,route_selection\]`

	testCases := []expectedErrorTestCase{
		{
			Name: "agreeing-values",
			Config: `
					resource "tailscale_tailnet_settings" "test_settings" {
						regional_routing_on = true
						route_selection     = "regional-routing"
					}
				`,
			ExpectError: regexp.MustCompile(wantErr),
		},
		{
			Name: "contradictory-values",
			Config: `
					resource "tailscale_tailnet_settings" "test_settings" {
						regional_routing_on = false
						route_selection     = "regional-routing"
					}
				`,
			ExpectError: regexp.MustCompile(wantErr),
		},
		{
			// Terraform resolves references after an initial validation pass, so we could construct
			// a chain of references that skips validation because they're "unknown" at first.
			// Then once it's time to plan or apply, the reference may be resolved to a concrete
			// type. Our ConfigValidators should run again at that point and reject this config.
			Name: "conflict-reference",
			Config: `
					resource "terraform_data" "rs" {
						input = "magicroute"
					}
					resource "tailscale_tailnet_settings" "test_settings" {
						regional_routing_on = false
						route_selection     = terraform_data.rs.output
					}
				`,
			ExpectError: regexp.MustCompile(wantErr),
		},
	}

	runExpectedErrorTests(t, testCases)
}

// TestResourceTailnetSettingsDoUpdate_SkipsUnknown asserts that certain attributes which are
// unknown in the plan are omitted from the update request. Specifically, neither regional_routing_on
// nor route_selection have the UseStateForUnknown plan modifier (see [tailnetSettingsResource.Schema]),
// so we're testing that unknown states are fine and we don't propagate some assumed or zero value.
func TestResourceTailnetSettingsDoUpdate_SkipsUnknown(t *testing.T) {
	tests := []struct {
		name        string
		state, plan tailnetSettingsResourceModel

		wantRouteSelection    *tailscale.RouteSelection
		wantRegionalRoutingOn *bool
	}{
		{
			name: "route-selection-unknown",
			state: tailnetSettingsResourceModel{
				RouteSelection:    types.StringValue(string(tailscale.RouteSelectionMagicRoute)),
				RegionalRoutingOn: types.BoolValue(false),
			},
			plan: tailnetSettingsResourceModel{
				RouteSelection:    types.StringUnknown(),
				RegionalRoutingOn: types.BoolValue(true),
			},
			wantRegionalRoutingOn: new(true),
		},
		{
			name: "regional-routing-unknown",
			state: tailnetSettingsResourceModel{
				RouteSelection:    types.StringValue(string(tailscale.RouteSelectionRegionalRouting)),
				RegionalRoutingOn: types.BoolValue(true),
			},
			plan: tailnetSettingsResourceModel{
				RouteSelection:    types.StringValue(string(tailscale.RouteSelectionMagicRoute)),
				RegionalRoutingOn: types.BoolUnknown(),
			},
			wantRouteSelection: new(tailscale.RouteSelectionMagicRoute),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			baseURL, server := NewTestHarness(t)
			server.ResponseCode = http.StatusOK
			server.ResponseBody = nil

			parsedURL, err := url.Parse(baseURL)
			if err != nil {
				t.Fatalf("parsing test harness URL: %s", err)
			}

			res := &tailnetSettingsResource{ResourceBase: ResourceBase{
				Client: &tailscale.Client{BaseURL: parsedURL, APIKey: "test-api-key"},
			}}

			if err := res.resourceTailnetSettingsDoUpdate(context.Background(), tt.plan, tt.state); err != nil {
				t.Fatalf("updating tailnet settings: %s", err)
			}

			var got tailscale.UpdateTailnetSettingsRequest
			if err := json.Unmarshal(server.Body.Bytes(), &got); err != nil {
				t.Fatalf("unmarshalling update request: %s", err)
			}

			if err := assertEqual(tt.wantRouteSelection, got.RouteSelection, "wrong route_selection sent"); err != nil {
				t.Error(err)
			}
			if err := assertEqual(tt.wantRegionalRoutingOn, got.RegionalRoutingOn, "wrong regional_routing_on sent"); err != nil {
				t.Error(err)
			}
		})
	}
}
