// Copyright (c) David Bond, Tailscale Inc, & Contributors
// SPDX-License-Identifier: MIT

package tailscale

import (
	"context"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"

	"tailscale.com/client/tailscale/v2"
)

func dataSourceUsers() *schema.Resource {
	return &schema.Resource{
		Description: "The users data source describes a list of users in a tailnet",
		ReadContext: dataSourceUsersRead,
		Schema: map[string]*schema.Schema{
			"type": {
				Optional:    true,
				Type:        schema.TypeString,
				Description: "Filter the results to only include users of a specific type. Valid values are `member` or `shared`.",
				ValidateFunc: validation.StringInSlice(
					[]string{
						string(tailscale.UserTypeMember),
						string(tailscale.UserTypeShared),
					},
					false,
				),
			},
			"role": {
				Optional:    true,
				Type:        schema.TypeString,
				Description: "Filter the results to only include users with a specific role. Valid values are `owner`, `member`, `admin`, `it-admin`, `network-admin`, `billing-admin`, and `auditor`.",
				ValidateFunc: validation.StringInSlice(
					[]string{
						string(tailscale.UserRoleOwner),
						string(tailscale.UserRoleMember),
						string(tailscale.UserRoleAdmin),
						string(tailscale.UserRoleITAdmin),
						string(tailscale.UserRoleNetworkAdmin),
						string(tailscale.UserRoleBillingAdmin),
						string(tailscale.UserRoleAuditor),
					},
					false,
				),
			},
			"users": {
				Computed:    true,
				Type:        schema.TypeList,
				Description: "The list of users in the tailnet",
				Elem: &schema.Resource{
					Schema: combinedSchemas(commonUserSchema, map[string]*schema.Schema{
						"id": {
							Type:        schema.TypeString,
							Description: "The unique identifier for the user.",
							Computed:    true,
						},
						"login_name": {
							Type:        schema.TypeString,
							Description: "The emailish login name of the user.",
							Computed:    true,
						},
					}),
				},
			},
		},
	}
}

func dataSourceUsersRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	client := m.(*tailscale.Client)

	var userType *tailscale.UserType
	if _userType, ok := d.Get("type").(string); ok {
		userType = tailscale.PointerTo(tailscale.UserType(_userType))
	}

	var userRole *tailscale.UserRole
	if _userRole, ok := d.Get("role").(string); ok {
		userRole = tailscale.PointerTo(tailscale.UserRole(_userRole))
	}

	users, err := client.Users().List(ctx, userType, userRole)
	if err != nil {
		return diagnosticsError(err, "Failed to fetch users")
	}

	userMaps := make([]map[string]interface{}, 0, len(users))
	for _, user := range users {
		m := userToMap(&user)
		userMaps = append(userMaps, m)
	}

	if err = d.Set("users", userMaps); err != nil {
		return diag.FromErr(err)
	}

	d.SetId(createUUID())
	return nil
}
