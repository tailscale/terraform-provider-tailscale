// Copyright (c) David Bond, Tailscale Inc, & Contributors
// SPDX-License-Identifier: MIT

package tailscale

import (
	"context"
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"

	"tailscale.com/client/tailscale/v2"
)

func TestAccTailscaleContacts(t *testing.T) {
	const resourceName = "tailscale_contacts.test_contacts"

	const testContactsBasic = `
	resource "tailscale_contacts" "test_contacts" {
		account {
			email = "account@example.com"
		}

		support {
			email = "support@example.com"
		}

		security {
			email = "security@example.com"
		}
	}`

	const testContactsUpdated = `
	resource "tailscale_contacts" "test_contacts" {
		account {
			email = "otheraccount@example.com"
		}

		support {
			email = "support@example.com"
		}

		security {
			email = "security2@example.com"
		}
	}`

	expectedContactsBasic := &tailscale.Contacts{
		Account: tailscale.Contact{
			Email: "account@example.com",
		},
		Support: tailscale.Contact{
			Email: "support@example.com",
		},
		Security: tailscale.Contact{
			Email: "security@example.com",
		},
	}

	expectedContactsUpdated := &tailscale.Contacts{
		Account: tailscale.Contact{
			Email: "otheraccount@example.com",
		},
		Support: tailscale.Contact{
			Email: "support@example.com",
		},
		Security: tailscale.Contact{
			Email: "security2@example.com",
		},
	}

	checkProperties := func(expectedContacts *tailscale.Contacts) func(client *tailscale.Client, rs *terraform.ResourceState) error {
		return func(client *tailscale.Client, rs *terraform.ResourceState) error {
			contacts, err := client.Contacts().Get(context.Background())
			if err != nil {
				return err
			}

			if contacts.Account.Email != expectedContacts.Account.Email {
				return fmt.Errorf("bad account email, expected %q, got %q", expectedContacts.Account.Email, contacts.Account.Email)
			}

			if contacts.Support.Email != expectedContacts.Support.Email {
				return fmt.Errorf("bad support email, expected %q, got %q", expectedContacts.Support.Email, contacts.Support.Email)
			}

			if contacts.Security.Email != expectedContacts.Security.Email {
				return fmt.Errorf("bad security email, expected %q, got %q", expectedContacts.Security.Email, contacts.Security.Email)
			}

			return nil
		}
	}

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories(t),
		// Contacts are not destroyed in the control plane upon resource deletion since
		// contacts cannot be empty, so make sure that contacts are still the updated contacts.
		CheckDestroy: checkResourceDestroyed(resourceName, checkProperties(expectedContactsUpdated)),
		Steps: []resource.TestStep{
			{
				Config: testContactsBasic,
				Check: resource.ComposeTestCheckFunc(
					checkResourceRemoteProperties(resourceName, checkProperties(expectedContactsBasic)),
					resource.TestCheckResourceAttr(resourceName, "account.0.email", "account@example.com"),
					resource.TestCheckResourceAttr(resourceName, "support.0.email", "support@example.com"),
					resource.TestCheckResourceAttr(resourceName, "security.0.email", "security@example.com"),
				),
			},
			{
				Config: testContactsUpdated,
				Check: resource.ComposeTestCheckFunc(
					checkResourceRemoteProperties(resourceName, checkProperties(expectedContactsUpdated)),
					resource.TestCheckResourceAttr(resourceName, "account.0.email", "otheraccount@example.com"),
					resource.TestCheckResourceAttr(resourceName, "support.0.email", "support@example.com"),
					resource.TestCheckResourceAttr(resourceName, "security.0.email", "security2@example.com"),
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
