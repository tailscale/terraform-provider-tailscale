package tailscale_test

import (
	"errors"
	"fmt"
	"os"
	"testing"

	"github.com/davidsbond/terraform-provider-tailscale/tailscale"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

var providerFactories = map[string]func() (*schema.Provider, error){
	"tailscale": func() (*schema.Provider, error) {
		return tailscale.Provider(), nil
	},
}

func TestProvider(t *testing.T) {
	if err := tailscale.Provider().InternalValidate(); err != nil {
		t.Fatalf("err: %s", err)
	}
}

func TestProvider_Implemented(t *testing.T) {
	var _ *schema.Provider = tailscale.Provider()
}

func testProviderPreCheck(t *testing.T) {
	if err := os.Getenv("TAILSCALE_API_KEY"); err == "" {
		t.Fatal("TAILSCALE_API_KEY must be set for acceptance tests")
	}
	if err := os.Getenv("TAILSCALE_TAILNET"); err == "" {
		t.Fatal("TAILSCALE_TAILNET must be set for acceptance tests")
	}
}

func testResourceCreated(name, hcl string) resource.TestStep {
	return resource.TestStep{
		ResourceName: name,
		Config:       hcl,
		Check: func(s *terraform.State) error {
			rs, ok := s.RootModule().Resources[name]

			if !ok {
				return fmt.Errorf("not found: %s", name)
			}

			if rs.Primary.ID == "" {
				return errors.New("no ID set")
			}

			return nil
		},
	}
}

func testResourceDestroyed(name string, hcl string) resource.TestStep {
	return resource.TestStep{
		ResourceName: name,
		Destroy:      true,
		Config:       hcl,
		Check: func(s *terraform.State) error {
			rs, ok := s.RootModule().Resources[name]

			if !ok {
				return fmt.Errorf("not found: %s", name)
			}

			if rs.Primary.ID == "" {
				return errors.New("no ID set")
			}

			return nil
		},
	}
}
