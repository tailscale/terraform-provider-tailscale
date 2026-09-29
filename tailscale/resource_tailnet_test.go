// Copyright (c) David Bond, Tailscale Inc, & Contributors
// SPDX-License-Identifier: MIT

package tailscale

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"testing"
	"time"

	frameworkresource "github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"github.com/stretchr/testify/require"
	"tailscale.com/client/tailscale/v2"
)

const testTailnet = `
	resource "tailscale_tailnet" "test_tailnet" {
		display_name = "Example Tailnet"
	}`

const testTailnetUpdate = `
	resource "tailscale_tailnet" "test_tailnet" {
		display_name = "Updated Tailnet"
	}`

func TestProvider_TailscaleTailnet(t *testing.T) {
	createdAt := time.Date(2025, 1, 1, 12, 0, 0, 0, time.UTC)
	createdTailnet := tailscale.Tailnet{
		ID:          "T123456CNTRL",
		DisplayName: "Example Tailnet",
		OrgID:       "o123456CNTRL",
		DNSName:     "tail1234.ts.net",
		CreatedAt:   createdAt,
		OAuthClient: &tailscale.TailnetOAuthClient{
			ID:     "k123456CNTRL",
			Secret: "tskey-client-xxxxxxxxxxxx-xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx",
		},
	}
	listTailnets := map[string]any{
		"tailnets": []map[string]any{{
			"id": createdTailnet.ID, "displayName": createdTailnet.DisplayName,
			"orgId": createdTailnet.OrgID, "createdAt": createdAt,
		}},
	}

	resource.Test(t, resource.TestCase{
		IsUnitTest: true,
		PreCheck: func() {
			testServer.ResponsesByRequest = map[string]TestResponse{
				http.MethodPost + " /api/v2/organizations/-/tailnets": {
					Code: http.StatusOK,
					Body: createdTailnet,
				},
				http.MethodGet + " /api/v2/organizations/-/tailnets": {
					Code: http.StatusOK,
					Body: listTailnets,
				},
				http.MethodPost + " /api/v2/oauth/token": {
					Code: http.StatusOK,
					Body: map[string]any{
						"access_token": "access-token",
						"token_type":   "Bearer",
						"expires_in":   3600,
					},
				},
				http.MethodDelete + " /api/v2/tailnet/T123456CNTRL": {
					Code: http.StatusOK,
					Body: map[string]any{},
				},
			}
		},
		ProtoV5ProviderFactories: testProviderFactories(t),
		Steps: []resource.TestStep{
			{
				Config: testTailnet,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("tailscale_tailnet.test_tailnet", "id", "T123456CNTRL"),
					resource.TestCheckResourceAttr("tailscale_tailnet.test_tailnet", "display_name", "Example Tailnet"),
					resource.TestCheckResourceAttr("tailscale_tailnet.test_tailnet", "org_id", "o123456CNTRL"),
					resource.TestCheckResourceAttr("tailscale_tailnet.test_tailnet", "dns_name", "tail1234.ts.net"),
					resource.TestCheckResourceAttr("tailscale_tailnet.test_tailnet", "created_at", createdAt.Format(time.RFC3339)),
					resource.TestCheckResourceAttr("tailscale_tailnet.test_tailnet", "oauth_client_id", "k123456CNTRL"),
					resource.TestCheckResourceAttr("tailscale_tailnet.test_tailnet", "oauth_client_secret", "tskey-client-xxxxxxxxxxxx-xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx"),
					func(_ *terraform.State) error {
						createReq, err := findTestRequest(http.MethodPost, "/api/v2/organizations/-/tailnets")
						if err != nil {
							return err
						}
						var body tailscale.CreateTailnetRequest
						if err := json.Unmarshal([]byte(createReq.Body), &body); err != nil {
							return err
						}
						if body.DisplayName != "Example Tailnet" {
							return fmt.Errorf("wrong displayName: want %q, got %q", "Example Tailnet", body.DisplayName)
						}
						return nil
					},
				),
			},
			{
				Config:             testTailnetUpdate,
				PlanOnly:           true,
				ExpectNonEmptyPlan: true,
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PostApplyPreRefresh: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction("tailscale_tailnet.test_tailnet", plancheck.ResourceActionDestroyBeforeCreate),
					},
				},
			},
		},
		CheckDestroy: func(_ *terraform.State) error {
			tokenReq, err := findTestRequest(http.MethodPost, "/api/v2/oauth/token")
			if err != nil {
				return err
			}
			tokenHTTPReq := http.Request{Header: tokenReq.Header}
			id, secret, ok := tokenHTTPReq.BasicAuth()
			if !ok || id != "k123456CNTRL" || secret != "tskey-client-xxxxxxxxxxxx-xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx" {
				return fmt.Errorf("delete OAuth token exchange did not use stored credentials")
			}
			form, err := url.ParseQuery(tokenReq.Body)
			if err != nil || form.Get("scope") != "all" {
				return fmt.Errorf("delete OAuth token exchange did not request the all scope")
			}
			_, err = findTestRequest(http.MethodDelete, "/api/v2/tailnet/T123456CNTRL")
			return err
		},
	})
}

func TestProvider_TailscaleTailnet_RemovesMissingTailnet(t *testing.T) {
	resource.Test(t, resource.TestCase{
		IsUnitTest: true,
		PreCheck: func() {
			testServer.ResponsesByRequest = map[string]TestResponse{
				http.MethodPost + " /api/v2/organizations/-/tailnets": {
					Code: http.StatusOK,
					Body: tailscale.Tailnet{
						ID:          "T123456CNTRL",
						DisplayName: "Example Tailnet",
						OrgID:       "o123456CNTRL",
						DNSName:     "tail1234.ts.net",
						CreatedAt:   time.Date(2025, 1, 1, 12, 0, 0, 0, time.UTC),
						OAuthClient: &tailscale.TailnetOAuthClient{
							ID:     "k123456CNTRL",
							Secret: "tskey-client-xxxxxxxxxxxx-xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx",
						},
					},
				},
				http.MethodGet + " /api/v2/organizations/-/tailnets": {
					Code: http.StatusOK,
					Body: map[string]any{"tailnets": []map[string]any{{
						"id": "T123456CNTRL", "displayName": "Example Tailnet",
						"orgId": "o123456CNTRL", "createdAt": "2025-01-01T12:00:00Z",
					}}},
				},
				http.MethodPost + " /api/v2/oauth/token": {
					Code: http.StatusOK,
					Body: map[string]any{"access_token": "access-token", "token_type": "Bearer", "expires_in": 3600},
				},
				http.MethodDelete + " /api/v2/tailnet/T123456CNTRL": {
					Code: http.StatusNotFound,
					Body: map[string]any{"message": "tailnet not found"},
				},
			}
		},
		ProtoV5ProviderFactories: testProviderFactories(t),
		Steps: []resource.TestStep{
			{
				Config: testTailnet,
			},
			{
				PreConfig: func() {
					testServer.ResponsesByRequest[http.MethodGet+" /api/v2/organizations/-/tailnets"] = TestResponse{
						Code: http.StatusOK,
						Body: map[string]any{"tailnets": []any{}},
					}
				},
				Config:             testTailnet,
				PlanOnly:           true,
				ExpectNonEmptyPlan: true,
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PostApplyPostRefresh: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction("tailscale_tailnet.test_tailnet", plancheck.ResourceActionCreate),
					},
				},
			},
		},
	})
}

func TestProvider_TailscaleTailnetsDataSource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		IsUnitTest: true,
		PreCheck: func() {
			testServer.ResponsesByRequest = map[string]TestResponse{
				"GET /api/v2/organizations/-/tailnets": {
					Code: http.StatusOK,
					Body: map[string]any{
						"tailnets": []map[string]any{{
							"id": "T123456CNTRL", "displayName": "Example Tailnet",
							"orgId": "o123456CNTRL", "createdAt": "2025-01-01T12:00:00Z",
						}},
						"cursor":     "page2",
						"totalCount": 2,
					},
				},
				"GET /api/v2/organizations/-/tailnets?cursor=page2": {
					Code: http.StatusOK,
					Body: map[string]any{"tailnets": []map[string]any{{
						"id": "T654321CNTRL", "displayName": "Another Tailnet",
						"orgId": "o123456CNTRL", "createdAt": "2025-01-02T12:00:00Z",
					}}},
				},
			}
		},
		ProtoV5ProviderFactories: testProviderFactories(t),
		Steps: []resource.TestStep{
			{
				Config: `data "tailscale_tailnets" "all" {}`,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("data.tailscale_tailnets.all", "tailnets.#", "2"),
					resource.TestCheckResourceAttr("data.tailscale_tailnets.all", "tailnets.0.id", "T123456CNTRL"),
					resource.TestCheckResourceAttr("data.tailscale_tailnets.all", "tailnets.0.display_name", "Example Tailnet"),
					resource.TestCheckResourceAttr("data.tailscale_tailnets.all", "tailnets.0.org_id", "o123456CNTRL"),
					resource.TestCheckResourceAttr("data.tailscale_tailnets.all", "tailnets.0.created_at", "2025-01-01T12:00:00Z"),
					resource.TestCheckResourceAttr("data.tailscale_tailnets.all", "tailnets.1.id", "T654321CNTRL"),
					resource.TestCheckResourceAttr("data.tailscale_tailnets.all", "tailnets.1.display_name", "Another Tailnet"),
					resource.TestCheckResourceAttr("data.tailscale_tailnets.all", "tailnets.1.org_id", "o123456CNTRL"),
					resource.TestCheckResourceAttr("data.tailscale_tailnets.all", "tailnets.1.created_at", "2025-01-02T12:00:00Z"),
					func(_ *terraform.State) error {
						_, err := findTestRequest(http.MethodGet, "/api/v2/organizations/-/tailnets")
						return err
					},
				),
			},
		},
	})
}

func findTestRequest(method, path string) (TestRequest, error) {
	for i := len(testServer.Requests) - 1; i >= 0; i-- {
		req := testServer.Requests[i]
		if req.Method == method && req.Path == path {
			return req, nil
		}
	}
	return TestRequest{}, fmt.Errorf("no test request found for %s %s", method, path)
}

func TestTailnetCreatePartialCredentials(t *testing.T) {
	for _, tc := range []struct {
		name       string
		oauth      *tailscale.TailnetOAuthClient
		id, secret types.String
	}{
		{"missing OAuth", nil, types.StringNull(), types.StringNull()},
		{"empty OAuth", &tailscale.TailnetOAuthClient{}, types.StringNull(), types.StringNull()},
		{"missing ID", &tailscale.TailnetOAuthClient{Secret: "secret"}, types.StringNull(), types.StringValue("secret")},
		{"missing secret", &tailscale.TailnetOAuthClient{ID: "client"}, types.StringValue("client"), types.StringNull()},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ctx := context.Background()
			baseURL, server := NewTestHarness(t)
			endpoint, err := url.Parse(baseURL)
			require.NoError(t, err)
			server.ResponseCode = http.StatusOK
			server.ResponseBody = tailscale.Tailnet{
				ID: "T123", DisplayName: "Example Tailnet", OrgID: "o123", DNSName: "tail123.ts.net",
				CreatedAt: time.Date(2025, 1, 1, 12, 0, 0, 0, time.UTC), OAuthClient: tc.oauth,
			}
			r := &tailnetResource{ResourceBase: ResourceBase{Client: &tailscale.Client{BaseURL: endpoint, APIKey: "test"}}}
			var schema frameworkresource.SchemaResponse
			r.Schema(ctx, frameworkresource.SchemaRequest{}, &schema)
			plan := tfsdk.Plan{Schema: schema.Schema}
			require.False(t, plan.Set(ctx, &tailnetResourceModel{
				DisplayName: types.StringValue("Example Tailnet"), ID: types.StringUnknown(),
				OrgID: types.StringUnknown(), DNSName: types.StringUnknown(), CreatedAt: types.StringUnknown(),
				OAuthClientID: types.StringUnknown(), OAuthClientSecret: types.StringUnknown(),
			}).HasError())
			resp := frameworkresource.CreateResponse{State: tfsdk.State{Schema: schema.Schema}}
			r.Create(ctx, frameworkresource.CreateRequest{Plan: plan}, &resp)
			require.True(t, resp.Diagnostics.HasError(), "missing credentials must be reported")
			var state tailnetResourceModel
			require.False(t, resp.State.Get(ctx, &state).HasError())
			require.Equal(t, types.StringValue("T123"), state.ID)
			require.Equal(t, types.StringValue("Example Tailnet"), state.DisplayName)
			require.Equal(t, types.StringValue("o123"), state.OrgID)
			require.Equal(t, types.StringValue("tail123.ts.net"), state.DNSName)
			require.Equal(t, types.StringValue("2025-01-01T12:00:00Z"), state.CreatedAt)
			require.Equal(t, tc.id, state.OAuthClientID)
			require.Equal(t, tc.secret, state.OAuthClientSecret)
		})
	}
}

func TestTailnetDelete(t *testing.T) {
	for _, tc := range []struct {
		name       string
		id, secret types.String
		status     int
		wantError  bool
	}{
		{"not found", types.StringValue("client"), types.StringValue("secret"), http.StatusNotFound, false},
		{"API error", types.StringValue("client"), types.StringValue("secret"), http.StatusForbidden, true},
		{"null ID", types.StringNull(), types.StringValue("secret"), 0, true},
		{"empty secret", types.StringValue("client"), types.StringValue(""), 0, true},
		{"unknown secret", types.StringValue("client"), types.StringUnknown(), 0, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ctx := context.Background()
			baseURL, server := NewTestHarness(t)
			endpoint, err := url.Parse(baseURL)
			require.NoError(t, err)
			server.ResponseCode = http.StatusForbidden
			server.ResponsesByRequest = map[string]TestResponse{
				"POST /api/v2/oauth/token": {Code: http.StatusOK, Body: map[string]any{
					"access_token": "access-token", "token_type": "Bearer", "expires_in": 3600,
				}},
				"DELETE /api/v2/tailnet/T123": {Code: tc.status, Body: map[string]any{"message": "delete rejected"}},
			}
			r := &tailnetResource{ResourceBase: ResourceBase{Client: &tailscale.Client{BaseURL: endpoint, APIKey: "test"}}}
			var schema frameworkresource.SchemaResponse
			r.Schema(ctx, frameworkresource.SchemaRequest{}, &schema)
			state := tfsdk.State{Schema: schema.Schema}
			require.False(t, state.Set(ctx, &tailnetResourceModel{
				ID: types.StringValue("T123"), OAuthClientID: tc.id, OAuthClientSecret: tc.secret,
			}).HasError())
			resp := frameworkresource.DeleteResponse{State: state}
			r.Delete(ctx, frameworkresource.DeleteRequest{State: state}, &resp)
			require.Equal(t, tc.wantError, resp.Diagnostics.HasError(), "%v", resp.Diagnostics)
			if tc.status == 0 {
				require.Empty(t, server.Requests, "missing credentials must fail before HTTP requests")
				require.Contains(t, strings.ToLower(resp.Diagnostics.Errors()[0].Detail()), "oauth")
			} else {
				require.Len(t, server.Requests, 2)
				require.Equal(t, http.MethodDelete, server.Requests[1].Method)
				if tc.wantError {
					require.Contains(t, resp.Diagnostics.Errors()[0].Detail(), "delete rejected")
				}
			}
		})
	}
}

func TestTailnetList(t *testing.T) {
	for _, tc := range []struct {
		name      string
		response  TestResponse
		wantError bool
	}{
		{"empty", TestResponse{Code: http.StatusOK, Body: map[string]any{"tailnets": []any{}}}, false},
		{"missing array", TestResponse{Code: http.StatusOK, Body: map[string]any{}}, true},
		{"null array", TestResponse{Code: http.StatusOK, Body: map[string]any{"tailnets": nil}}, true},
		{"invalid array", TestResponse{Code: http.StatusOK, Body: map[string]any{"tailnets": "invalid"}}, true},
		{"repeated cursor", TestResponse{Code: http.StatusOK, Body: map[string]any{"tailnets": []any{}, "cursor": "same"}}, true},
		{"API error", TestResponse{Code: http.StatusForbidden, Body: map[string]any{"message": "access denied"}}, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			baseURL, server := NewTestHarness(t)
			endpoint, err := url.Parse(baseURL)
			require.NoError(t, err)
			server.ResponseCode, server.ResponseBody = tc.response.Code, tc.response.Body
			result, err := listTailnets(context.Background(), &tailscale.Client{BaseURL: endpoint, APIKey: "test"})
			if tc.wantError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				require.Empty(t, result)
			}
		})
	}
}

func TestTailnetReadSecondPage(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	baseURL, server := NewTestHarness(t)
	endpoint, err := url.Parse(baseURL)
	require.NoError(t, err)
	server.ResponsesByRequest = map[string]TestResponse{
		"GET /api/v2/organizations/-/tailnets": {Code: http.StatusOK, Body: map[string]any{
			"tailnets": []map[string]any{{"id": "other"}}, "cursor": "page+2", "totalCount": 2,
		}},
		"GET /api/v2/organizations/-/tailnets?cursor=page%2B2": {Code: http.StatusOK, Body: map[string]any{
			"tailnets": []map[string]any{{"id": "T123", "displayName": "Updated Tailnet", "orgId": "o123", "createdAt": "2025-01-01T12:00:00Z"}},
		}},
	}
	r := &tailnetResource{ResourceBase: ResourceBase{Client: &tailscale.Client{BaseURL: endpoint, APIKey: "test"}}}
	var schema frameworkresource.SchemaResponse
	r.Schema(ctx, frameworkresource.SchemaRequest{}, &schema)
	state := tfsdk.State{Schema: schema.Schema}
	require.False(t, state.Set(ctx, &tailnetResourceModel{
		ID: types.StringValue("T123"), DisplayName: types.StringValue("Example Tailnet"),
		DNSName: types.StringValue("tail123.ts.net"), OAuthClientID: types.StringValue("client"), OAuthClientSecret: types.StringValue("secret"),
	}).HasError())
	resp := frameworkresource.ReadResponse{State: state}
	r.Read(ctx, frameworkresource.ReadRequest{State: state}, &resp)
	require.False(t, resp.Diagnostics.HasError(), "%v", resp.Diagnostics)
	var got tailnetResourceModel
	require.False(t, resp.State.Get(ctx, &got).HasError())
	require.Equal(t, types.StringValue("T123"), got.ID)
	require.Equal(t, types.StringValue("Updated Tailnet"), got.DisplayName)
	require.Equal(t, types.StringValue("o123"), got.OrgID)
	require.Equal(t, types.StringValue("2025-01-01T12:00:00Z"), got.CreatedAt)
	require.Equal(t, types.StringValue("tail123.ts.net"), got.DNSName)
	require.Equal(t, types.StringValue("client"), got.OAuthClientID)
	require.Equal(t, types.StringValue("secret"), got.OAuthClientSecret)
	require.Len(t, server.Requests, 2)
	require.Equal(t, "cursor=page%2B2", server.Requests[1].RawQuery)

	// A failure on a later page must not discard the resource or its credentials.
	server.ResponsesByRequest["GET /api/v2/organizations/-/tailnets?cursor=page%2B2"] = TestResponse{
		Code: http.StatusForbidden,
		Body: map[string]any{"message": "access denied"},
	}
	state = resp.State
	resp = frameworkresource.ReadResponse{State: state}
	r.Read(ctx, frameworkresource.ReadRequest{State: state}, &resp)
	require.True(t, resp.Diagnostics.HasError())
	require.True(t, state.Raw.Equal(resp.State.Raw), "failed refresh must preserve state")
}
