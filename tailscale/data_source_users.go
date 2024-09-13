// Copyright (c) David Bond, Tailscale Inc, & Contributors
// SPDX-License-Identifier: MIT

package tailscale

import (
	"context"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"

	tsclient "github.com/tailscale/tailscale-client-go/v2"
)

func dataSourceUsers() *schema.Resource {
	return &schema.Resource{
		Description: "The users data source describes a list of users in a tailnet",
		ReadContext: dataSourceUsersRead,
		Schema: map[string]*schema.Schema{
			"type": {
				Optional:    true,
				Type:        schema.TypeString,
				Description: "Filters the users list to elements whose type is the provided value.",
				ValidateFunc: validation.StringInSlice(
					[]string{
						string(tsclient.UserTypeMember),
						string(tsclient.UserTypeShared),
					},
					false,
				),
			},
			"role": {
				Optional:    true,
				Type:        schema.TypeString,
				Description: "Filters the users list to elements whose role is the provided value.",
				ValidateFunc: validation.StringInSlice(
					[]string{
						string(tsclient.UserRoleOwner),
						string(tsclient.UserRoleMember),
						string(tsclient.UserRoleAdmin),
						string(tsclient.UserRoleITAdmin),
						string(tsclient.UserRoleNetworkAdmin),
						string(tsclient.UserRoleBillingAdmin),
						string(tsclient.UserRoleAuditor),
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
	client := m.(*tsclient.Client)

	var userType *tsclient.UserType
	if _userType, ok := d.Get("type").(string); ok {
		userType = tsclient.PointerTo(tsclient.UserType(_userType))
	}

	var userRole *tsclient.UserRole
	if _userRole, ok := d.Get("role").(string); ok {
		userRole = tsclient.PointerTo(tsclient.UserRole(_userRole))
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
