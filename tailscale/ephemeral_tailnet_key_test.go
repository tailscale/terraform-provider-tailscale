// Copyright (c) David Bond, Tailscale Inc, & Contributors
// SPDX-License-Identifier: MIT

package tailscale

import (
	"net/http"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	"github.com/hashicorp/terraform-plugin-testing/echoprovider"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/tfversion"

	"tailscale.com/client/tailscale/v2"
)

func TestProvider_TailscaleTailnetKeyEphemeral(t *testing.T) {
	key := tailscale.Key{
		ID:            "ephemeral-key-id",
		KeyType:       "auth",
		Key:           "thisisatestkey",
		ExpirySeconds: new(time.Duration(900)),
		Created:       time.Date(2026, time.October, 25, 21, 0, 0, 0, time.UTC),
		Expires:       time.Date(2026, time.October, 25, 21, 15, 0, 0, time.UTC),
	}

	var requests []string
	providerFactories := testProviderV6Factories(t, func(method, path string) TestResponse {
		requests = append(requests, method+" "+path)
		switch method {
		case http.MethodPost:
			return TestResponse{Code: http.StatusOK, Body: key}
		case http.MethodDelete:
			return TestResponse{Code: http.StatusNotFound}
		default:
			return TestResponse{Code: http.StatusMethodNotAllowed}
		}
	})

	resource.Test(t, resource.TestCase{
		IsUnitTest: true,
		TerraformVersionChecks: []tfversion.TerraformVersionCheck{
			tfversion.SkipBelow(tfversion.Version1_10_0),
		},
		ProtoV6ProviderFactories: providerFactories,
		Steps: []resource.TestStep{
			{
				Config: `
					ephemeral "tailscale_tailnet_key" "test" {
						ephemeral     = true
						preauthorized = true
						tags          = ["tag:server"]
						expiry        = 900
					}

					provider "echo" {
						data = ephemeral.tailscale_tailnet_key.test
					}

					resource "echo" "test" {}
				`,
				Check: resource.TestCheckResourceAttr("echo.test", "data.key", key.Key),
			},
		},
	})

	if len(requests) == 0 {
		t.Fatal("expected ephemeral auth key API requests")
	}

	createCount, deleteCount := 0, 0
	for _, request := range requests {
		switch {
		case strings.HasPrefix(request, http.MethodPost+" "):
			createCount++
		case strings.HasPrefix(request, http.MethodDelete+" "):
			deleteCount++
		}
	}
	if createCount == 0 || createCount != deleteCount {
		t.Fatalf("expected all opened keys to be deleted; got %d creates and %d deletes: %v", createCount, deleteCount, requests)
	}

	for _, request := range requests {
		if strings.HasPrefix(request, http.MethodDelete+" ") && !strings.HasSuffix(request, "/"+key.ID) {
			t.Fatalf("expected cleanup to delete key %q, got request %q", key.ID, request)
		}
	}
}

func TestProvider_TailscaleOAuthClientEphemeral(t *testing.T) {
	client := tailscale.Key{
		ID:      "ephemeral-oauth-client-id",
		KeyType: "client",
		Key:     "thisisatestsecret",
		Created: time.Date(2026, time.October, 25, 21, 0, 0, 0, time.UTC),
		Updated: time.Date(2026, time.October, 25, 21, 0, 0, 0, time.UTC),
		UserID:  "test-user-id",
		Scopes:  []string{"devices:core:read"},
		Tags:    []string{},
	}

	var requests []string
	providerFactories := testProviderV6Factories(t, func(method, path string) TestResponse {
		requests = append(requests, method+" "+path)
		switch method {
		case http.MethodPost:
			return TestResponse{Code: http.StatusOK, Body: client}
		case http.MethodDelete:
			return TestResponse{Code: http.StatusOK}
		default:
			return TestResponse{Code: http.StatusMethodNotAllowed}
		}
	})

	resource.Test(t, resource.TestCase{
		IsUnitTest: true,
		TerraformVersionChecks: []tfversion.TerraformVersionCheck{
			tfversion.SkipBelow(tfversion.Version1_10_0),
		},
		ProtoV6ProviderFactories: providerFactories,
		Steps: []resource.TestStep{
			{
				Config: `
					ephemeral "tailscale_oauth_client" "test" {
						description = "Temporary metrics client"
						scopes      = ["devices:core:read"]
					}

					provider "echo" {
						data = ephemeral.tailscale_oauth_client.test
					}

					resource "echo" "test" {}
				`,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("echo.test", "data.id", client.ID),
					resource.TestCheckResourceAttr("echo.test", "data.key", client.Key),
				),
			},
		},
	})

	if len(requests) == 0 {
		t.Fatal("expected ephemeral OAuth client API requests")
	}

	createCount, deleteCount := 0, 0
	for _, request := range requests {
		switch {
		case strings.HasPrefix(request, http.MethodPost+" "):
			createCount++
		case strings.HasPrefix(request, http.MethodDelete+" "):
			deleteCount++
		}
	}
	if createCount == 0 || createCount != deleteCount {
		t.Fatalf("expected all opened OAuth clients to be deleted; got %d creates and %d deletes: %v", createCount, deleteCount, requests)
	}

	for _, request := range requests {
		if strings.HasPrefix(request, http.MethodDelete+" ") && !strings.HasSuffix(request, "/"+client.ID) {
			t.Fatalf("expected cleanup to delete OAuth client %q, got request %q", client.ID, request)
		}
	}
}

func TestProvider_TailscaleTailnetKeyEphemeralDeleteError(t *testing.T) {
	key := tailscale.Key{
		ID:      "ephemeral-key-id",
		KeyType: "auth",
		Key:     "thisisatestkey",
	}

	providerFactories := testProviderV6Factories(t, func(method, path string) TestResponse {
		switch method {
		case http.MethodPost:
			return TestResponse{Code: http.StatusOK, Body: key}
		case http.MethodDelete:
			return TestResponse{Code: http.StatusInternalServerError}
		default:
			return TestResponse{Code: http.StatusMethodNotAllowed}
		}
	})

	resource.Test(t, resource.TestCase{
		IsUnitTest: true,
		TerraformVersionChecks: []tfversion.TerraformVersionCheck{
			tfversion.SkipBelow(tfversion.Version1_10_0),
		},
		ProtoV6ProviderFactories: providerFactories,
		Steps: []resource.TestStep{
			{
				Config: `
					ephemeral "tailscale_tailnet_key" "test" {
						tags = ["tag:server"]
					}

					provider "echo" {
						data = ephemeral.tailscale_tailnet_key.test
					}

					resource "echo" "test" {}
				`,
				ExpectError: regexp.MustCompile("Failed to delete key"),
			},
		},
	})
}

func testProviderV6Factories(t *testing.T, handleRequest func(method, path string) TestResponse) map[string]func() (tfprotov6.ProviderServer, error) {
	t.Helper()

	baseURL, server := NewTestHarness(t)
	testServer = server
	server.HandleRequest = handleRequest

	return map[string]func() (tfprotov6.ProviderServer, error){
		"tailscale": func() (tfprotov6.ProviderServer, error) {
			t.Setenv("TAILSCALE_API_KEY", "api_123")
			t.Setenv("TAILSCALE_BASE_URL", baseURL)

			return providerserver.NewProtocol6(NewFrameworkProvider())(), nil
		},
		"echo": echoprovider.NewProviderServer(),
	}
}
