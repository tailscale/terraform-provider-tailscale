package tailscale_test

import (
	"context"
	"errors"
	"fmt"
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"

	"github.com/tailscale/terraform-provider-tailscale/tailscale"
)

var testClients *tailscale.Clients
var testServer *TestServer
var testAccProvider = tailscale.Provider()

// testAccPreCheck ensures that the TAILSCALE_API_KEY and TAILSCALE_BASE_URL variables
// are set and configures the provider. This must be called before running acceptance
// tests.
func testAccPreCheck(t *testing.T) {
	t.Helper()

	if v := os.Getenv("TAILSCALE_API_KEY"); v == "" {
		t.Fatal("TAILSCALE_API_KEY must be set for acceptance tests")
	}

	if v := os.Getenv("TAILSCALE_BASE_URL"); v == "" {
		t.Fatal("TAILSCALE_BASE_URL must be set for acceptance tests")
	}

	if v := os.Getenv("TAILSCALE_TEST_DEVICE_NAME"); v == "" {
		t.Fatal("TAILSCALE_TEST_DEVICE_NAME must be set for acceptance tests")
	}

	if diags := testAccProvider.Configure(context.Background(), &terraform.ResourceConfig{}); diags.HasError() {
		for _, d := range diags {
			if d.Severity == diag.Error {
				t.Fatalf("Failed to configure provider: %s", d.Summary)
			}
		}
	}
}

func testAccProviderFactories(t *testing.T) map[string]func() (*schema.Provider, error) {
	t.Helper()

	return map[string]func() (*schema.Provider, error){
		"tailscale": func() (*schema.Provider, error) {
			return tailscale.Provider(), nil
		},
	}
}

func TestProvider(t *testing.T) {
	if err := tailscale.Provider().InternalValidate(); err != nil {
		t.Fatalf("err: %s", err)
	}
}

func TestProvider_Implemented(t *testing.T) {
	var _ *schema.Provider = tailscale.Provider()
}

func testProviderFactories(t *testing.T) map[string]func() (*schema.Provider, error) {
	t.Helper()

	testClients, testServer = NewTestHarness(t)
	return map[string]func() (*schema.Provider, error){
		"tailscale": func() (*schema.Provider, error) {
			return tailscale.Provider(func(p *schema.Provider) {
				// Set up a test harness for the provider
				p.ConfigureContextFunc = func(ctx context.Context, data *schema.ResourceData) (interface{}, diag.Diagnostics) {
					return testClients, nil
				}

				// Don't require any of the global configuration
				p.Schema = nil
			}), nil
		},
	}
}

func testResourceCreated(name, hcl string) resource.TestStep {
	return resource.TestStep{
		ResourceName:       name,
		Config:             hcl,
		ExpectNonEmptyPlan: true,
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
