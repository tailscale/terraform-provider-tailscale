// Copyright (c) David Bond, Tailscale Inc, & Contributors
// SPDX-License-Identifier: MIT

package tailscale

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-go/tfprotov5"
	"github.com/hashicorp/terraform-plugin-mux/tf5muxserver"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"

	"tailscale.com/client/tailscale/v2"
)

var testClient *tailscale.Client
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

// testAccProviderFactories creates a mux server that serves both the SDKv2 and
// the plugin framework providers.
//
// See https://developer.hashicorp.com/terraform/plugin/framework/migrating/mux#protocol-version-5
func testAccProviderFactories(t *testing.T) map[string]func() (tfprotov5.ProviderServer, error) {
	t.Helper()

	return map[string]func() (tfprotov5.ProviderServer, error){
		"tailscale": func() (tfprotov5.ProviderServer, error) {
			ctx := context.Background()

			providers := []func() tfprotov5.ProviderServer{
				providerserver.NewProtocol5(NewFrameworkProvider()),
				Provider().GRPCProvider,
			}

			muxServer, err := tf5muxserver.NewMuxServer(ctx, providers...)

			if err != nil {
				return nil, err
			}

			return muxServer.ProviderServer(), nil
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
	var _ provider.Provider = NewFrameworkProvider()
}

// testProviderFactories creates a mux server that serves both the SDKv2 and
// the plugin framework providers.
//
// See https://developer.hashicorp.com/terraform/plugin/framework/migrating/mux#protocol-version-5
func testProviderFactories(t *testing.T) map[string]func() (tfprotov5.ProviderServer, error) {
	t.Helper()

	testClient, testServer = NewTestHarness(t)
	return map[string]func() (tfprotov5.ProviderServer, error){
		"tailscale": func() (tfprotov5.ProviderServer, error) {
			ctx := context.Background()

			t.Setenv("TAILSCALE_API_KEY", "api_123")
			t.Setenv("TAILSCALE_BASE_URL", testClient.BaseURL.String())

			providers := []func() tfprotov5.ProviderServer{
				providerserver.NewProtocol5(NewFrameworkProvider()),
				Provider(func(p *schema.Provider) {
					// Set up a test harness for the provider
					p.ConfigureContextFunc = func(ctx context.Context, data *schema.ResourceData) (interface{}, diag.Diagnostics) {
						return testClient, nil
					}
				}).GRPCProvider,
			}

			muxServer, err := tf5muxserver.NewMuxServer(ctx, providers...)

			if err != nil {
				return nil, err
			}

			return muxServer.ProviderServer(), nil
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

		client := testAccProvider.Meta().(*tailscale.Client)
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

		client := testAccProvider.Meta().(*tailscale.Client)
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
			name:          "oauth_client_id without secret or token",
			oauthClientID: "client-id",
			wantErr:       "oauth_client_secret' or 'identity_token' are mandatory",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			diags := validateProviderCreds(tt.apiKey, tt.oauthClientID, tt.oauthSecret, tt.idToken)

			if tt.wantErr == "" && diags.HasError() {
				t.Errorf("unexpected error: %v", diags)

			}

			if tt.wantErr != "" && !diags.HasError() {
				t.Errorf("expected error containing %q but got none", tt.wantErr)
				return
			}

			if tt.wantErr != "" {
				match := false
				for _, d := range diags {
					if d.Severity == diag.Error {
						errMsg := d.Summary + d.Detail
						if strings.Contains(errMsg, tt.wantErr) {
							match = true
							break
						}
					}
				}
				if !match {
					t.Errorf("expected error containing %q but got: %v", tt.wantErr, diags)
				}
			}
		})
	}
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
				PlanOnly:                 true,
			},
		},
	})
}

func TestRetryWithDeadline_SucceedsEventually(t *testing.T) {
	ctx := context.Background()

	var calls int32
	err := retryWithDeadline(ctx, 100*time.Millisecond, 10*time.Millisecond, func(ctx context.Context) error {
		calls += 1
		if calls < 2 {
			return errors.New("not found")
		}
		return nil
	})

	if err != nil {
		t.Fatalf("want no error but got one: %v", err)
	}

	if got := calls; got < 2 {
		t.Fatalf("want at least 2 calls but got %d", got)
	}
}

func TestRetryWithDeadline_WrapsEventualErrorOnFailure(t *testing.T) {
	ctx := context.Background()

	var calls int32
	err := retryWithDeadline(ctx, 100*time.Millisecond, 10*time.Millisecond, func(ctx context.Context) error {
		calls += 1
		if calls < 2 {
			return errors.New("not found")
		}
		return errors.New("something else went wrong")
	})

	if err == nil {
		t.Fatal("want error but got none")
	}

	if got := err.Error(); !strings.Contains(got, "something else went wrong") {
		t.Fatalf("want error to contain \"something else went wrong\" but got %v", got)
	}
}
