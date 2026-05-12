// Copyright (c) David Bond, Tailscale Inc, & Contributors
// SPDX-License-Identifier: MIT

package tailscale

import (
	"errors"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tfprotov5"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
	"github.com/hashicorp/terraform-plugin-testing/terraform"

	"tailscale.com/client/tailscale/v2"
)

// getTestAccClient returns an instance of [tailscale.Client] for use in
// the acceptance tests, which is authenticated against devcontrol.
func getAccTestClient() *tailscale.Client {
	baseURL, err := url.Parse(os.Getenv("TAILSCALE_BASE_URL"))
	if err != nil {
		panic(fmt.Sprintf("unable to parse base URL %q: %v", baseURL, err))
	}

	client := tailscale.Client{
		BaseURL:   baseURL,
		UserAgent: "tailscale-terraform-provider tests",
		APIKey:    os.Getenv("TAILSCALE_API_KEY"),
	}

	return &client
}

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

	getAccTestClient()
}

// testAccProviderFactories sets up the Terraform provider for acceptance tests,
// connecting it to a running instance of devcontrol.
func testAccProviderFactories(t *testing.T) map[string]func() (tfprotov5.ProviderServer, error) {
	t.Helper()

	return map[string]func() (tfprotov5.ProviderServer, error){
		"tailscale": func() (tfprotov5.ProviderServer, error) {
			provider := NewFrameworkProvider()
			tfServer := providerserver.NewProtocol5(provider)
			return tfServer(), nil
		},
	}
}

// testServer is a mock HTTP server uses to simulate the Tailscale API.
//
// Tests can define mock responses in the PreCheck step of a test.
var testServer *TestServer

// testProviderFactories sets up the Terraform provider for non-acceptance tests,
// connecting it to [testServer].
func testProviderFactories(t *testing.T) map[string]func() (tfprotov5.ProviderServer, error) {
	t.Helper()

	baseURL, server := NewTestHarness(t)
	testServer = server
	return map[string]func() (tfprotov5.ProviderServer, error){
		"tailscale": func() (tfprotov5.ProviderServer, error) {
			t.Setenv("TAILSCALE_API_KEY", "api_123")
			t.Setenv("TAILSCALE_BASE_URL", baseURL)

			provider := NewFrameworkProvider()
			return providerserver.NewProtocol5(provider)(), nil
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

func checkResourceRemoteProperties(resourceName string, check func(client *tailscale.Client, rs *terraform.ResourceState) error) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[resourceName]
		if !ok {
			return fmt.Errorf("resource not found: %s", resourceName)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("resource has no ID set")
		}

		client := getAccTestClient()
		return check(client, rs)
	}
}

func checkResourceDestroyed(resourceName string, check func(client *tailscale.Client, rs *terraform.ResourceState) error) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[resourceName]
		if !ok {
			return fmt.Errorf("resource not found: %s", resourceName)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("resource has no ID set")
		}

		client := getAccTestClient()
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

func TestValidateProviderCreds(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		apiKey        string
		oauthClientID string
		oauthSecret   string
		idToken       string
		audience      string
		wantErr       string
	}{
		{
			name:    "valid api_key only",
			apiKey:  "test-api-key",
			wantErr: "",
		},
		{
			name:          "valid oauth with client secret",
			oauthClientID: "client-id",
			oauthSecret:   "client-secret",
			wantErr:       "",
		},
		{
			name:          "valid oauth with identity token",
			oauthClientID: "client-id",
			idToken:       "id-token",
			wantErr:       "",
		},
		{
			name:          "valid oauth with audience",
			oauthClientID: "client-id",
			audience:      "tailscale-aud",
			wantErr:       "",
		},
		{
			name:    "all credentials empty",
			wantErr: "credentials are empty",
		},
		{
			name:          "api_key conflicts with oauth_client_id",
			apiKey:        "test-api-key",
			oauthClientID: "client-id",
			wantErr:       "credentials are conflicting",
		},
		{
			name:        "api_key conflicts with oauth_client_secret",
			apiKey:      "test-api-key",
			oauthSecret: "client-secret",
			wantErr:     "credentials are conflicting",
		},
		{
			name:    "api_key conflicts with identity_token",
			apiKey:  "test-api-key",
			idToken: "id-token",
			wantErr: "credentials are conflicting",
		},
		{
			name:     "api_key conflicts with audience",
			apiKey:   "test-api-key",
			audience: "tailscale-aud",
			wantErr:  "credentials are conflicting",
		},
		{
			name:        "oauth_client_id missing with only oauth_client_secret",
			oauthSecret: "client-secret",
			wantErr:     "oauth_client_id' is empty",
		},
		{
			name:    "oauth_client_id missing with only identity_token",
			idToken: "id-token",
			wantErr: "oauth_client_id' is empty",
		},
		{
			name:     "oauth_client_id missing with only audience",
			audience: "tailscale-aud",
			wantErr:  "oauth_client_id' is empty",
		},
		{
			name:          "oauth_client_id without secret, token, or audience",
			oauthClientID: "client-id",
			wantErr:       "'oauth_client_secret', 'identity_token', or 'audience' are mandatory",
		},
		{
			name:          "audience conflicts with oauth_client_secret",
			oauthClientID: "client-id",
			oauthSecret:   "client-secret",
			audience:      "tailscale-aud",
			wantErr:       "audience' conflicts",
		},
		{
			name:          "audience conflicts with identity_token",
			oauthClientID: "client-id",
			idToken:       "id-token",
			audience:      "tailscale-aud",
			wantErr:       "audience' conflicts",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateProviderCreds(tt.apiKey, tt.oauthClientID, tt.oauthSecret, tt.idToken, tt.audience)

			if tt.wantErr == "" && err != nil {
				t.Errorf("unexpected error: %v", err)

			}

			if tt.wantErr != "" && err == nil {
				t.Errorf("expected error containing %q but got none", tt.wantErr)
				return
			}

			if tt.wantErr != "" {
				if !strings.Contains(err.Error(), tt.wantErr) {
					t.Errorf("expected error containing %q but got: %v", tt.wantErr, err)
				}
			}
		})
	}
}

func TestResolveValueFromFile(t *testing.T) {
	t.Parallel()

	t.Run("not-using-file-format", func(t *testing.T) {
		dir := t.TempDir()
		// We expect that this temporary file will _not_ be read since we will
		// not be prefixing the value passed to resolveValueFromFile with "file:".
		// Create it with a known sentinel value for erroring out in the test if
		// we do incorrectly read it.
		tempFile := filepath.Join(dir, "file-exists")
		err := os.WriteFile(tempFile, []byte("this-should-not-have-been-read"), 0600)
		if err != nil {
			t.Fatal(err)
		}

		var diags diag.Diagnostics
		// Expect that we get the file name as a value and not the contents of
		// the file since the value was not prefixed with "file:"
		val := resolveValueFromFile(&diags, "example-key", types.StringValue(tempFile))
		if val != tempFile {
			t.Errorf("expected value %q but got %s", tempFile, val)
		}

		if diags.HasError() {
			t.Fatalf("expected no diag errors but got %v", diags.Errors())
		}
	})

	t.Run("using-file-format-and-file-exists", func(t *testing.T) {
		dir := t.TempDir()
		tempFile := filepath.Join(dir, "file-exists")
		err := os.WriteFile(tempFile, []byte("example-value"), 0600)
		if err != nil {
			t.Fatal(err)
		}

		var diags diag.Diagnostics
		val := resolveValueFromFile(&diags, "example-key", types.StringValue("file:"+tempFile))
		if val != "example-value" {
			t.Errorf("expected value 'example-value' but got %s", val)
		}

		if diags.HasError() {
			t.Fatalf("expected no diag errors but got %v", diags.Errors())
		}
	})

	t.Run("using-file-format-in-fallbacks-and-file-exists", func(t *testing.T) {
		dir := t.TempDir()
		tempFile := filepath.Join(dir, "file-exists")
		err := os.WriteFile(tempFile, []byte("example-value"), 0600)
		if err != nil {
			t.Fatal(err)
		}

		var diags diag.Diagnostics
		// This emulates file:tempFile coming from an env-var or elsewhere
		val := resolveValueFromFile(&diags, "example-key", types.StringNull(), "file:"+tempFile)
		if val != "example-value" {
			t.Errorf("expected value 'example-value' but got %s", val)
		}

		if diags.HasError() {
			t.Fatalf("expected no diag errors but got %v", diags.Errors())
		}
	})

	t.Run("using-file-format-and-file-does-not-exist", func(t *testing.T) {
		var diags diag.Diagnostics
		keyName := "example-key"
		fileName := "does-not-exist"

		val := resolveValueFromFile(&diags, keyName, types.StringValue("file:"+fileName))
		if val != "" {
			t.Errorf("expected empty string but got %s", val)
		}

		if !diags.HasError() {
			t.Fatal("expected diag errors but got none")
		}

		if diags.ErrorsCount() != 1 {
			t.Fatalf("expected 1 diag error but got %d", diags.ErrorsCount())
		}

		diagError := diags.Errors()[0]

		if !strings.Contains(diagError.Summary(), keyName) {
			t.Fatalf("expected summary to contain error text with %q but was %q", keyName, diagError.Summary())
		}

		expectedErrorFragment := "could not be read"
		if !strings.Contains(diagError.Detail(), expectedErrorFragment) ||
			!strings.Contains(diagError.Detail(), fileName) ||
			!strings.Contains(diagError.Detail(), keyName) {
			t.Fatalf("expected error detail to contain %q, %q, and %q but was %q", expectedErrorFragment, fileName, keyName, diagError.Detail())
		}
	})
}

// checkDataSourceIsUnchangedInPluginFramework runs a migration test to check
// that a data source returns the same data from the plugin SDK and
// the plugin framework.
//
// See https://developer.hashicorp.com/terraform/plugin/framework/migrating/testing#terraform-data-resource-example
func checkDataSourceIsUnchangedInPluginFramework(t *testing.T, config string) {
	t.Helper()

	resource.Test(t, resource.TestCase{
		Steps: []resource.TestStep{
			{
				ExternalProviders: map[string]resource.ExternalProvider{
					"tailscale": {
						VersionConstraint: "0.28.0",
						Source:            "tailscale/tailscale",
					},
				},
				Config: config,
			},
			{
				ProtoV5ProviderFactories: testAccProviderFactories(t),
				Config:                   config,
				PlanOnly:                 true,
			},
		},
	})
}

// checkResourceIsUnchangedInPluginFramework runs a migration test to check
// that a resource created by the plugin SDK is a no-op plan in the framework.
//
// See https://developer.hashicorp.com/terraform/plugin/framework/migrating/testing#external-providers
func checkResourceIsUnchangedInPluginFramework(t *testing.T, config string, check resource.TestCheckFunc) {
	t.Helper()
	resource.Test(t, resource.TestCase{
		Steps: []resource.TestStep{
			{
				ExternalProviders: map[string]resource.ExternalProvider{
					"tailscale": {
						VersionConstraint: "0.28.0",
						Source:            "tailscale/tailscale",
					},
				},
				Config: config,
				Check:  check,
			},
			{
				ProtoV5ProviderFactories: testAccProviderFactories(t),
				Config:                   config,
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectEmptyPlan(),
					},
				},
			},
		},
	})
}
