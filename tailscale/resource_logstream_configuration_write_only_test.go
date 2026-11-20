// Copyright (c) David Bond, Tailscale Inc, & Contributors
// SPDX-License-Identifier: MIT

package tailscale

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"regexp"
	"sync"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/hashicorp/terraform-plugin-testing/config"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"github.com/hashicorp/terraform-plugin-testing/tfjsonpath"
	"github.com/hashicorp/terraform-plugin-testing/tfversion"
)

// logstreamTestAPI records writes and returns configuration without the token,
// matching the logstream API's handling of credentials.
type logstreamTestAPI struct {
	mu            sync.Mutex
	configuration map[string]any
	tokens        []string
}

func (api *logstreamTestAPI) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	api.mu.Lock()
	defer api.mu.Unlock()

	if r.URL.Path != "/api/v2/tailnet/test-tailnet/logging/configuration/stream" {
		http.NotFound(w, r)
		return
	}

	switch r.Method {
	case http.MethodPut:
		var configuration map[string]any
		if err := json.NewDecoder(r.Body).Decode(&configuration); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		token, _ := configuration["token"].(string)
		api.tokens = append(api.tokens, token)
		delete(configuration, "token")
		configuration["logType"] = "configuration"
		api.configuration = configuration
		w.WriteHeader(http.StatusOK)
	case http.MethodGet:
		if api.configuration == nil {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(api.configuration)
	case http.MethodDelete:
		api.configuration = nil
		w.WriteHeader(http.StatusOK)
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

func (api *logstreamTestAPI) checkTokens(want ...string) resource.TestCheckFunc {
	return func(_ *terraform.State) error {
		api.mu.Lock()
		defer api.mu.Unlock()
		if diff := cmp.Diff(want, api.tokens); diff != "" {
			return fmt.Errorf("unexpected logstream API tokens (-want +got): %s", diff)
		}
		return nil
	}
}

func testLogstreamWriteOnlyConfig(baseURL, url string, version int) string {
	return fmt.Sprintf(`
		provider "tailscale" {
			api_key  = "test-api-key"
			base_url = %q
			tailnet  = "test-tailnet"
		}
		variable "logstream_token" {
			type      = string
			sensitive = true
			ephemeral = true
		}
		resource "tailscale_logstream_configuration" "test" {
			log_type         = "configuration"
			destination_type = "splunk"
			url              = %q
			token_wo         = var.logstream_token
			token_wo_version = %d
		}
	`, baseURL, url, version)
}

func TestProvider_TailscaleLogstreamConfiguration_WriteOnly(t *testing.T) {
	const resourceName = "tailscale_logstream_configuration.test"
	api := &logstreamTestAPI{}
	server := httptest.NewServer(api)
	t.Cleanup(server.Close)

	tokenVariable := func(value string) config.Variables {
		return config.Variables{"logstream_token": config.StringVariable(value)}
	}
	checkState := func(version string, tokens ...string) resource.TestCheckFunc {
		return resource.ComposeTestCheckFunc(
			resource.TestCheckResourceAttr(resourceName, "id", "configuration"),
			resource.TestCheckResourceAttr(resourceName, "token_wo_version", version),
			resource.TestCheckNoResourceAttr(resourceName, "token"),
			resource.TestCheckNoResourceAttr(resourceName, "token_wo"),
			api.checkTokens(tokens...),
		)
	}

	resource.Test(t, resource.TestCase{
		IsUnitTest:               true,
		ProtoV5ProviderFactories: testAccProviderFactories(t),
		TerraformVersionChecks:   []tfversion.TerraformVersionCheck{tfversion.SkipBelow(tfversion.Version1_11_0)},
		Steps: []resource.TestStep{
			{
				Config:          testLogstreamWriteOnlyConfig(server.URL, "https://example.com", 1),
				ConfigVariables: tokenVariable("initial-token"),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectKnownValue(resourceName, tfjsonpath.New("token"), knownvalue.Null()),
						plancheck.ExpectKnownValue(resourceName, tfjsonpath.New("token_wo"), knownvalue.Null()),
					},
				},
				Check: checkState("1", "initial-token"),
			},
			{
				// A different ephemeral value alone must not trigger an update.
				Config:          testLogstreamWriteOnlyConfig(server.URL, "https://example.com", 1),
				ConfigVariables: tokenVariable("rotated-token"),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()},
				},
				Check: checkState("1", "initial-token"),
			},
			{
				Config:          testLogstreamWriteOnlyConfig(server.URL, "https://example.com", 2),
				ConfigVariables: tokenVariable("rotated-token"),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{plancheck.ExpectResourceAction(resourceName, plancheck.ResourceActionUpdate)},
				},
				Check: checkState("2", "initial-token", "rotated-token"),
			},
			{
				// The API needs the current token on updates to other settings too.
				Config:          testLogstreamWriteOnlyConfig(server.URL, "https://example.com/other", 2),
				ConfigVariables: tokenVariable("current-token"),
				Check: resource.ComposeTestCheckFunc(
					checkState("2", "initial-token", "rotated-token", "current-token"),
					resource.TestCheckResourceAttr(resourceName, "url", "https://example.com/other"),
				),
			},
			{
				ConfigVariables:         tokenVariable("current-token"),
				ResourceName:            resourceName,
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"token_wo_version"},
			},
		},
	})
}

func TestProvider_TailscaleLogstreamConfiguration_WriteOnlyMigration(t *testing.T) {
	const resourceName = "tailscale_logstream_configuration.test"
	api := &logstreamTestAPI{}
	server := httptest.NewServer(api)
	t.Cleanup(server.Close)

	legacyConfig := fmt.Sprintf(`
		provider "tailscale" {
			api_key  = "test-api-key"
			base_url = %q
			tailnet  = "test-tailnet"
		}
		resource "tailscale_logstream_configuration" "test" {
			log_type         = "configuration"
			destination_type = "splunk"
			url              = "https://example.com"
			token            = "legacy-token"
		}
	`, server.URL)

	resource.Test(t, resource.TestCase{
		IsUnitTest:               true,
		ProtoV5ProviderFactories: testAccProviderFactories(t),
		TerraformVersionChecks:   []tfversion.TerraformVersionCheck{tfversion.SkipBelow(tfversion.Version1_11_0)},
		Steps: []resource.TestStep{
			{
				Config: legacyConfig,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "token", "legacy-token"),
					api.checkTokens("legacy-token"),
				),
			},
			{
				Config: testLogstreamWriteOnlyConfig(server.URL, "https://example.com", 1),
				ConfigVariables: config.Variables{
					"logstream_token": config.StringVariable("legacy-token"),
				},
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{plancheck.ExpectResourceAction(resourceName, plancheck.ResourceActionUpdate)},
				},
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckNoResourceAttr(resourceName, "token"),
					resource.TestCheckNoResourceAttr(resourceName, "token_wo"),
					resource.TestCheckResourceAttr(resourceName, "token_wo_version", "1"),
					api.checkTokens("legacy-token", "legacy-token"),
				),
			},
			{
				Config: legacyConfig,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "token", "legacy-token"),
					resource.TestCheckNoResourceAttr(resourceName, "token_wo"),
					resource.TestCheckNoResourceAttr(resourceName, "token_wo_version"),
					api.checkTokens("legacy-token", "legacy-token", "legacy-token"),
				),
			},
		},
	})
}

func TestProvider_TailscaleLogstreamConfiguration_WriteOnlyValidation(t *testing.T) {
	for _, tt := range []struct {
		name       string
		attributes string
		err        string
	}{
		{"conflicting tokens", `token = "legacy-token"
			token_wo = "write-only-token"
			token_wo_version = 1`, "Invalid Attribute Combination"},
		{"missing version", `token_wo = "write-only-token"`, `Attribute "token_wo_version" must be specified`},
		{"missing token", `token_wo_version = 1`, `Attribute "token_wo" must be specified`},
		{"empty token", `token_wo = ""
			token_wo_version = 1`, "Invalid Attribute Value Length"},
		{"invalid version", `token_wo = "write-only-token"
			token_wo_version = 0`, "Invalid Attribute Value"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			resource.Test(t, resource.TestCase{
				IsUnitTest:               true,
				ProtoV5ProviderFactories: testAccProviderFactories(t),
				TerraformVersionChecks:   []tfversion.TerraformVersionCheck{tfversion.SkipBelow(tfversion.Version1_11_0)},
				Steps: []resource.TestStep{{
					Config: fmt.Sprintf(`
						resource "tailscale_logstream_configuration" "test" {
							log_type         = "configuration"
							destination_type = "splunk"
							url              = "https://example.com"
							%s
						}
					`, tt.attributes),
					ExpectError: regexp.MustCompile(tt.err),
				}},
			})
		})
	}
}
