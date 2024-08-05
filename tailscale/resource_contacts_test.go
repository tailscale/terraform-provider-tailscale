package tailscale_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"

	tsclient "github.com/tailscale/tailscale-client-go/v2"
	"github.com/tailscale/terraform-provider-tailscale/tailscale"
)

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

var expectedContactsBasic = &tsclient.Contacts{
	Account: tsclient.Contact{
		Email: "account@example.com",
	},
	Support: tsclient.Contact{
		Email: "support@example.com",
	},
	Security: tsclient.Contact{
		Email: "security@example.com",
	},
}

var expectedContactsUpdated = &tsclient.Contacts{
	Account: tsclient.Contact{
		Email: "otheraccount@example.com",
	},
	Support: tsclient.Contact{
		Email: "support@example.com",
	},
	Security: tsclient.Contact{
		Email: "security2@example.com",
	},
}

func TestAccTailscaleContacts_Basic(t *testing.T) {
	contacts := &tsclient.Contacts{}

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories(t),
		CheckDestroy:      testAccCheckContactsDestroyBasic,
		Steps: []resource.TestStep{
			{
				Config: testContactsBasic,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckContactsExists("tailscale_contacts.test_contacts", contacts),
					testAccCheckContactsPropertiesBasic(contacts),
					resource.TestCheckResourceAttr("tailscale_contacts.test_contacts", "account.0.email", "account@example.com"),
					resource.TestCheckResourceAttr("tailscale_contacts.test_contacts", "support.0.email", "support@example.com"),
					resource.TestCheckResourceAttr("tailscale_contacts.test_contacts", "security.0.email", "security@example.com"),
				),
			},
			{
				ResourceName:      "tailscale_contacts.test_contacts",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func TestAccTailscaleContacts_Update(t *testing.T) {
	contacts := &tsclient.Contacts{}

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories(t),
		CheckDestroy:      testAccCheckContactsDestroyUpdated,
		Steps: []resource.TestStep{
			{
				Config: testContactsBasic,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckContactsExists("tailscale_contacts.test_contacts", contacts),
					testAccCheckContactsPropertiesBasic(contacts),
					resource.TestCheckResourceAttr("tailscale_contacts.test_contacts", "account.0.email", "account@example.com"),
					resource.TestCheckResourceAttr("tailscale_contacts.test_contacts", "support.0.email", "support@example.com"),
					resource.TestCheckResourceAttr("tailscale_contacts.test_contacts", "security.0.email", "security@example.com"),
				),
			},
			{
				Config: testContactsUpdated,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckContactsExists("tailscale_contacts.test_contacts", contacts),
					testAccCheckContactsPropertiesUpdated(contacts),
					resource.TestCheckResourceAttr("tailscale_contacts.test_contacts", "account.0.email", "otheraccount@example.com"),
					resource.TestCheckResourceAttr("tailscale_contacts.test_contacts", "support.0.email", "support@example.com"),
					resource.TestCheckResourceAttr("tailscale_contacts.test_contacts", "security.0.email", "security2@example.com"),
				),
			},
			{
				ResourceName:      "tailscale_contacts.test_contacts",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func testAccCheckContactsExists(resourceName string, contacts *tsclient.Contacts) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[resourceName]
		if !ok {
			return fmt.Errorf("resource not found: %s", resourceName)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("resource has no ID set")
		}

		client := testAccProvider.Meta().(*tailscale.Clients).V2
		out, err := client.Contacts().Get(context.Background())
		if err != nil {
			return err
		}

		*contacts = *out
		return nil
	}
}

func testAccCheckContactsPropertiesBasic(contacts *tsclient.Contacts) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		if err := checkContacts(contacts, expectedContactsBasic); err != nil {
			return err
		}

		return nil
	}
}

func testAccCheckContactsPropertiesUpdated(contacts *tsclient.Contacts) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		if err := checkContacts(contacts, expectedContactsUpdated); err != nil {
			return err
		}

		return nil
	}
}

func testAccCheckContactsDestroyBasic(s *terraform.State) error {
	return testAccCheckContactsDestroy(s, expectedContactsBasic)
}

func testAccCheckContactsDestroyUpdated(s *terraform.State) error {
	return testAccCheckContactsDestroy(s, expectedContactsUpdated)
}

func testAccCheckContactsDestroy(s *terraform.State, expectedContacts *tsclient.Contacts) error {
	client := testAccProvider.Meta().(*tailscale.Clients).V2

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "tailscale_contacts" {
			continue
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("resource has no ID set")
		}

		// Contacts are not destroyed in the control plane upon resource deletion since
		// contacts cannot be empty.
		contacts, err := client.Contacts().Get(context.Background())
		if err != nil {
			return fmt.Errorf("expected contacts to still exist")
		}

		return checkContacts(contacts, expectedContacts)
	}
	return nil
}

func checkContacts(contacts *tsclient.Contacts, expectedContacts *tsclient.Contacts) error {
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
