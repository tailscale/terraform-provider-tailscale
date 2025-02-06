// Copyright (c) David Bond, Tailscale Inc, & Contributors
// SPDX-License-Identifier: MIT

package tailscale

import (
	"context"
	"time"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"

	"tailscale.com/client/tailscale/v2"
)

var commonUserSchema = map[string]*schema.Schema{
	"display_name": {
		Type:        schema.TypeString,
		Description: "The name of the user.",
		Computed:    true,
	},
	"profile_pic_url": {
		Type:        schema.TypeString,
		Description: "The profile pic URL for the user.",
		Computed:    true,
	},
	"tailnet_id": {
		Type:        schema.TypeString,
		Description: "The tailnet that owns the user.",
		Computed:    true,
	},
	"created": {
		Type:        schema.TypeString,
		Description: "The time the user joined their tailnet.",
		Computed:    true,
	},
	"type": {
		Type:        schema.TypeString,
		Description: "The type of relation this user has to the tailnet associated with the request.",
		Computed:    true,
	},
	"role": {
		Type:        schema.TypeString,
		Description: "The role of the user.",
		Computed:    true,
	},
	"status": {
		Type:        schema.TypeString,
		Description: "The status of the user.",
		Computed:    true,
	},
	"device_count": {
		Type:        schema.TypeInt,
		Description: "Number of devices the user owns.",
		Computed:    true,
	},
	"last_seen": {
		Type:        schema.TypeString,
		Description: "The later of either: a) The last time any of the user's nodes were connected to the network or b) The last time the user authenticated to any tailscale service, including the admin panel.",
		Computed:    true,
	},
	"currently_connected": {
		Type:        schema.TypeBool,
		Description: "true when the user has a node currently connected to the control server.",
		Computed:    true,
	},
}

func dataSourceUser() *schema.Resource {
	return &schema.Resource{
		Description: "The user data source describes a single user in a tailnet",
		ReadContext: dataSourceUserRead,
		Schema: combinedSchemas(commonUserSchema, map[string]*schema.Schema{
			"id": {
				Type:         schema.TypeString,
				Description:  "The unique identifier for the user.",
				Optional:     true,
				ExactlyOneOf: []string{"id", "login_name"},
			},
			"login_name": {
				Type:         schema.TypeString,
				Description:  "The emailish login name of the user.",
				Optional:     true,
				ExactlyOneOf: []string{"id", "login_name"},
			},
		}),
	}
}

func dataSourceUserRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	client := m.(*tailscale.Client)

	if id := d.Id(); id != "" {
		user, err := client.Users().Get(ctx, id)
		if err != nil {
			return diagnosticsError(err, "Failed to fetch user with id %s", id)
		}
		return setProperties(d, userToMap(user))
	}

	loginName, ok := d.GetOk("login_name")
	if !ok {
		return diag.Errorf("please specify an id or login_name for the user")
	}

	users, err := client.Users().List(ctx, nil, nil)
	if err != nil {
		return diagnosticsError(err, "Failed to fetch users")
	}

	var selected *tailscale.User
	for _, user := range users {
		if user.LoginName == loginName.(string) {
			selected = &user
			break
		}
	}

	if selected == nil {
		return diag.Errorf("Could not find user with login name %s", loginName)
	}

	d.SetId(selected.ID)
	return setProperties(d, userToMap(selected))
}

// userToMap converts the given user into a map representing the user as a
// resource in Terraform. This omits the "id" which is expected to be set
// using [schema.ResourceData.SetId].
func userToMap(user *tailscale.User) map[string]any {
	return map[string]any{
		"id":                  user.ID,
		"display_name":        user.DisplayName,
		"login_name":          user.LoginName,
		"profile_pic_url":     user.ProfilePicURL,
		"tailnet_id":          user.TailnetID,
		"created":             user.Created.Format(time.RFC3339),
		"type":                user.Type,
		"role":                user.Role,
		"status":              user.Status,
		"device_count":        user.DeviceCount,
		"last_seen":           user.LastSeen.Format(time.RFC3339),
		"currently_connected": user.CurrentlyConnected,
	}
}
