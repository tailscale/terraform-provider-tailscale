package tailscale_test

import (
	"context"
	"errors"
	"fmt"
	"os"
	"reflect"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"

	tsclient "github.com/tailscale/tailscale-client-go/v2"
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

const anyValue = "*****"

// acceptanceTest provides a standard structure for acceptance tests. To use,
// construct a new acceptanceTest and then call [run].
type acceptanceTest struct {
	// resourceType is the type of resource being tested, e.g. "tailscale_webhook"
	// Required.
	resourceType string
	// resourceName is the name of the resource being tested, e.g. "test_webhook"
	// Required.
	resourceName string
	// initialValue is the intial value of the new resource, excluding the resource "type" "name" portion
	initialValue string
	// createCheckRemoteProperties is a function that uses the tsclient to check that the resource is correctly represented on the server.
	// This function is required.
	createCheckRemoteProperties func(client *tsclient.Client, rs *terraform.ResourceState) error
	// createCheckResourceAttributes is a map of attributes that the stored resource should have locally.
	// A value of [anyValue] ("*****") means to only check for the existence of the attribute, whatever its value.
	// Other string values will trigger a direct check for that attribute value.
	// Arrays of strings will trigger a TestCheckTypeSetElemAttr for each string in the array.
	// Required.
	createCheckResourceAttributes map[string]any
	// updatedValue is an optional updated value for the resource, used to test update logic.
	updatedValue string
	// updateCheckRemoteProperties is a function that uses the tsclient to check that the resource is correctly represented on the server after update.
	// Required if updatedValue is non-empty.
	updateCheckRemoteProperties func(client *tsclient.Client, rs *terraform.ResourceState) error
	// updateCheckResourceAttributes is a map of attributes that the stored resource should have locally after update.
	// A value of [anyValue] ("*****") means to only check for the existence of the attribute, whatever its value.
	// Other string values will trigger a direct check for that attribute value.
	// Arrays of strings will trigger a TestCheckTypeSetElemAttr for each string in the array.
	// Required if updatedValue is non-empty.
	updateCheckResourceAttributes map[string]any
	// checkRemoteDestroyed is a function that uses the tsclient to check that the resource has been deleted on the server.
	// Required.
	checkRemoteDestroyed func(client *tsclient.Client, rs *terraform.ResourceState) error
	// verifyImport, if true, will cause test to verify import of resource
	verifyImport bool
	// verifyImportIgnore is an optional list of attributes to ignore during import verification
	verifyImportIgnore []string
}

func (at acceptanceTest) run(t *testing.T) {
	if at.resourceType == "" {
		t.Fatalf("acceptance test failed to specify resourceType")
	}
	if at.resourceName == "" {
		t.Fatalf("acceptance test failed to specify resourceName")
	}
	if at.createCheckRemoteProperties == nil {
		t.Fatalf("acceptance test failed to specify createCheckRemoteProperties")
	}
	if at.createCheckResourceAttributes == nil {
		t.Fatalf("acceptance test failed to specify createCheckResourceAttributes")
	}
	if at.updatedValue != "" {
		if at.updateCheckRemoteProperties == nil {
			t.Fatalf("acceptance test failed to specify updateCheckRemoteProperties")
		}
		if at.updateCheckResourceAttributes == nil {
			t.Fatalf("acceptance test failed to specify updateCheckResourceAttributes")
		}
	}
	if at.checkRemoteDestroyed == nil {
		t.Fatalf("acceptance test failed to specify checkRemoteDestroyed")
	}

	fullName := fmt.Sprintf("%s.%s", at.resourceType, at.resourceName)
	resourceFromState := func(s *terraform.State) (*terraform.ResourceState, error) {
		rs, ok := s.RootModule().Resources[fullName]
		if !ok {
			return nil, fmt.Errorf("resource not found: %s", fullName)
		}

		if rs.Primary.ID == "" {
			return nil, fmt.Errorf("resource has no ID set")
		}

		return rs, nil
	}

	tc := resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories(t),
		Steps:             []resource.TestStep{},
	}

	tc.Steps = append(tc.Steps, resource.TestStep{
		Config: fmt.Sprintf("resource %q %s %s", at.resourceType, at.resourceName, at.initialValue),
		Check: resource.ComposeTestCheckFunc(
			func(s *terraform.State) error {
				rs, err := resourceFromState(s)
				if err != nil {
					return err
				}
				client := testAccProvider.Meta().(*tailscale.Clients).V2
				return at.createCheckRemoteProperties(client, rs)
			},
			func(s *terraform.State) error {
				for k, _v := range at.createCheckResourceAttributes {
					switch v := _v.(type) {
					case string:
						if v == anyValue {
							if err := resource.TestCheckResourceAttrSet(fullName, k)(s); err != nil {
								return err
							}
						} else if err := resource.TestCheckResourceAttr(fullName, k, v)(s); err != nil {
							return err
						}
					case []string:
						for _, v := range v {
							if err := resource.TestCheckTypeSetElemAttr(fullName, k, v)(s); err != nil {
								return err
							}
						}
					default:
						return fmt.Errorf("attribute %q had unknown expected value type %s", k, reflect.TypeOf(_v))
					}
				}
				return nil
			},
		),
	})

	if at.updatedValue != "" {
		tc.Steps = append(tc.Steps, resource.TestStep{
			Config: fmt.Sprintf("resource %q %s %s", at.resourceType, at.resourceName, at.updatedValue),
			Check: resource.ComposeTestCheckFunc(
				func(s *terraform.State) error {
					rs, err := resourceFromState(s)
					if err != nil {
						return err
					}
					client := testAccProvider.Meta().(*tailscale.Clients).V2
					return at.updateCheckRemoteProperties(client, rs)
				},
				func(s *terraform.State) error {
					for k, _v := range at.updateCheckResourceAttributes {
						switch v := _v.(type) {
						case string:
							if v == anyValue {
								if err := resource.TestCheckResourceAttrSet(fullName, k)(s); err != nil {
									return err
								}
							} else if err := resource.TestCheckResourceAttr(fullName, k, v)(s); err != nil {
								return err
							}
						case []string:
							for _, v := range v {
								if err := resource.TestCheckTypeSetElemAttr(fullName, k, v)(s); err != nil {
									return err
								}
							}
						default:
							return fmt.Errorf("attribute %q had unknown expected value type %s", k, reflect.TypeOf(_v))
						}
					}
					return nil
				},
			),
		})
	}

	tc.CheckDestroy = func(s *terraform.State) error {
		rs, err := resourceFromState(s)
		if err != nil {
			return err
		}
		client := testAccProvider.Meta().(*tailscale.Clients).V2
		return at.checkRemoteDestroyed(client, rs)
	}

	if at.verifyImport {
		tc.Steps = append(tc.Steps, resource.TestStep{
			ResourceName:            fullName,
			ImportState:             true,
			ImportStateVerify:       true,
			ImportStateVerifyIgnore: at.verifyImportIgnore,
		})
	}

	resource.Test(t, tc)
}
