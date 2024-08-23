package tailscale

import (
	"context"
	"errors"
	"fmt"
	"os"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"

	tsclient "github.com/tailscale/tailscale-client-go/v2"
)

var testClients *Clients
var testServer *TestServer
var testAccProvider = Provider()

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
			return Provider(), nil
		},
	}
}

func TestProvider(t *testing.T) {
	if err := Provider().InternalValidate(); err != nil {
		t.Fatalf("err: %s", err)
	}
}

func TestProvider_Implemented(t *testing.T) {
	var _ *schema.Provider = Provider()
}

func testProviderFactories(t *testing.T) map[string]func() (*schema.Provider, error) {
	t.Helper()

	testClients, testServer = NewTestHarness(t)
	return map[string]func() (*schema.Provider, error){
		"tailscale": func() (*schema.Provider, error) {
			return Provider(func(p *schema.Provider) {
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

func checkResourceRemoteProperties(resourceName string, check func(client *tsclient.Client, rs *terraform.ResourceState) error) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[resourceName]
		if !ok {
			return fmt.Errorf("resource not found: %s", resourceName)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("resource has no ID set")
		}

		client := testAccProvider.Meta().(*Clients).V2
		return check(client, rs)
	}
}

func checkResourceDestroyed(resourceName string, check func(client *tsclient.Client, rs *terraform.ResourceState) error) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[resourceName]
		if !ok {
			return fmt.Errorf("resource not found: %s", resourceName)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("resource has no ID set")
		}

		client := testAccProvider.Meta().(*Clients).V2
		return check(client, rs)
	}
}

// checkPropertiesMatch compares the properties on a named resource to the
// expected values in a map. All values in the [terraform.ResourceState] will
// be strings, while the map may contain strings, booleans or ints.
// This function returns an error if the resource is not found, or if any of
// the properties don't match.
func checkPropertiesMatch(resourceName string, s *terraform.State, expected map[string]any) error {
	rs := s.RootModule().Resources[resourceName]
	if rs == nil {
		return fmt.Errorf("no resource found for user %s", resourceName)
	}

	actual := rs.Primary.Attributes
	for k, v := range expected {
		switch t := v.(type) {
		case int:
			if actual[k] != fmt.Sprint(t) {
				return fmt.Errorf("wrong value for property %s of user %s, want %d, got %s", k, resourceName, t, actual[k])
			}
		case bool:
			if actual[k] != fmt.Sprint(t) {
				return fmt.Errorf("wrong value for property %s of user %s, want %v, got %s", k, resourceName, t, actual[k])
			}
		case string:
			if actual[k] != t {
				return fmt.Errorf("wrong value for property %s of user %s, want %s, got %s", k, resourceName, t, actual[k])
			}
		}
	}

	return nil
}

// assertEqual compares the expected and actual using [cmp.Diff] and reports an
// error if they're not equal.
func assertEqual(want, got any, errorMessage string) error {
	if diff := cmp.Diff(want, got); diff != "" {
		return fmt.Errorf("%s (-want +got): %s", errorMessage, diff)
	}
	return nil
}
