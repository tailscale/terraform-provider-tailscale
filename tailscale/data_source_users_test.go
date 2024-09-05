package tailscale

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"

	tsclient "github.com/tailscale/tailscale-client-go/v2"
)

func TestAccTailscaleUsers(t *testing.T) {
	resourceName := "data.tailscale_users.all_users"

	// This is a string containing tailscale_user datasource configurations
	userDataSources := &strings.Builder{}

	// First test the tailscale_users datasource, which will give us a list of
	// all user IDs.
	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories(t),
		Steps: []resource.TestStep{
			{
				Config: `data "tailscale_users" "all_users" {}`,
				Check: func(s *terraform.State) error {
					client := testAccProvider.Meta().(*tsclient.Client)
					users, err := client.Users().List(context.Background(), nil, nil)
					if err != nil {
						return fmt.Errorf("unable to list users: %s", err)
					}

					usersByLoginName := make(map[string]map[string]any)
					for _, user := range users {
						m := userToMap(&user)
						usersByLoginName[user.LoginName] = m
					}

					rs := s.RootModule().Resources[resourceName].Primary

					// first find indexes for users
					userIndexes := make(map[string]string)
					for k, v := range rs.Attributes {
						if strings.HasSuffix(k, ".login_name") {
							idx := strings.Split(k, ".")[1]
							userIndexes[idx] = v
						}
					}

					// make sure we got the right number of users
					if len(userIndexes) != len(usersByLoginName) {
						return fmt.Errorf("wrong number of users in datasource, want %d, got %d", len(usersByLoginName), len(userIndexes))
					}

					// now compare datasource attributes to expected values
					for k, v := range rs.Attributes {
						if strings.HasPrefix(k, "users.") {
							parts := strings.Split(k, ".")
							if len(parts) != 3 {
								continue
							}
							prop := parts[2]
							if prop == "%" {
								continue
							}
							idx := parts[1]
							loginName := userIndexes[idx]
							expected := fmt.Sprint(usersByLoginName[loginName][prop])
							if v != expected {
								return fmt.Errorf("wrong value of %s for user %s, want %q, got %q", prop, loginName, expected, v)
							}
						}
					}

					// Now set up user datasources for each user. This is used in the following test
					// of the tailscale_user datasource.
					for loginName, user := range usersByLoginName {
						userDataSources.WriteString(fmt.Sprintf("\ndata \"tailscale_user\" \"%s\" {\n  login_name = \"%s\"\n}\n", user["id"], loginName))
					}

					return nil
				},
			},
		},
	})

	// Now test the individual tailscale_user data sources for each user,
	// making sure that it pulls in the relevant details for each user.
	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories(t),
		Steps: []resource.TestStep{
			{
				Config: userDataSources.String(),
				Check: func(s *terraform.State) error {
					client := testAccProvider.Meta().(*tsclient.Client)
					users, err := client.Users().List(context.Background(), nil, nil)
					if err != nil {
						return fmt.Errorf("unable to list users: %s", err)
					}

					for _, user := range users {
						expected := userToMap(&user)
						expected["id"] = user.ID
						resourceName := fmt.Sprintf("data.tailscale_user.%s", user.ID)
						if err := checkPropertiesMatch(resourceName, s, expected); err != nil {
							return err
						}
					}

					return nil
				},
			},
		},
	})
}
