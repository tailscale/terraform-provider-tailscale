// Copyright (c) David Bond, Tailscale Inc, & Contributors
// SPDX-License-Identifier: MIT

package tailscale

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"strings"
	"sync"
	"testing"

	fwresource "github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
	"github.com/hashicorp/terraform-plugin-testing/terraform"

	"tailscale.com/client/tailscale/v2"
)

func TestAccTailscaleDeviceAuthorization(t *testing.T) {
	const resourceName = "tailscale_device_authorization.test_authorization"

	const testDeviceAuthorizationCreate = `
		data "tailscale_device" "test_device" {
			name = "%s"
		}
		
		resource "tailscale_device_authorization" "test_authorization" {
			device_id = data.tailscale_device.test_device.id
			authorized = true
		}`

	const testDeviceAuthorizationUpdate = `
		data "tailscale_device" "test_device" {
			name = "%s"
		}
		
		resource "tailscale_device_authorization" "test_authorization" {
			device_id = data.tailscale_device.test_device.id
			authorized = false
		}`

	const testDeviceAuthorizationNextUpdate = `
		data "tailscale_device" "test_device" {
			name = "%s"
		}
		
		resource "tailscale_device_authorization" "test_authorization" {
			device_id = data.tailscale_device.test_device.id
			authorized = true
		}`

	checkAuthorized := func(wantAuthorized bool) func(client *tailscale.Client, rs *terraform.ResourceState) error {
		return func(client *tailscale.Client, rs *terraform.ResourceState) error {
			device, err := client.Devices().Get(context.Background(), rs.Primary.ID)
			if err != nil {
				return err
			}

			if device.Authorized != wantAuthorized {
				return fmt.Errorf("device with id %q is not authorized", rs.Primary.ID)
			}

			return nil
		}
	}

	checkLegacyID := func(client *tailscale.Client, rs *terraform.ResourceState) error {
		// Check that the device ID and State ID Match
		device, err := client.Devices().Get(context.Background(), rs.Primary.ID)
		if err != nil {
			return err
		}

		if device.ID != rs.Primary.ID {
			return fmt.Errorf("state id %q does not match legacy id %q", rs.Primary.ID, device.ID)
		}

		return nil
	}

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV5ProviderFactories: testAccProviderFactories(t),
		// Devices are not currently deauthorized when this resource is deleted,
		// so if the resource is authorized when the resource gets deleted,
		// expect that the device both exists and is still authorized.
		CheckDestroy: checkResourceDestroyed(resourceName, checkAuthorized(true)),
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(testDeviceAuthorizationCreate, os.Getenv("TAILSCALE_TEST_DEVICE_NAME")),
				Check: resource.ComposeTestCheckFunc(
					checkResourceRemoteProperties(resourceName, checkLegacyID),
					checkResourceRemoteProperties(resourceName, checkAuthorized(true)),
					resource.TestCheckResourceAttr(resourceName, "authorized", "true"),
				),
			},
			{
				Config: fmt.Sprintf(testDeviceAuthorizationUpdate, os.Getenv("TAILSCALE_TEST_DEVICE_NAME")),
				Check: resource.ComposeTestCheckFunc(
					checkResourceRemoteProperties(resourceName, checkLegacyID),
					checkResourceRemoteProperties(resourceName, checkAuthorized(false)),
					resource.TestCheckResourceAttr(resourceName, "authorized", "false"),
				),
			},
			{
				Config: fmt.Sprintf(testDeviceAuthorizationNextUpdate, os.Getenv("TAILSCALE_TEST_DEVICE_NAME")),
				Check: resource.ComposeTestCheckFunc(
					checkResourceRemoteProperties(resourceName, checkLegacyID),
					checkResourceRemoteProperties(resourceName, checkAuthorized(true)),
					resource.TestCheckResourceAttr(resourceName, "authorized", "true"),
				),
			},
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})

	checkResourceIsUnchangedInPluginFramework(t,
		fmt.Sprintf(testDeviceAuthorizationCreate, os.Getenv("TAILSCALE_TEST_DEVICE_NAME")),
		resource.ComposeTestCheckFunc(
			checkResourceRemoteProperties(resourceName, checkLegacyID),
			checkResourceRemoteProperties(resourceName, checkAuthorized(true)),
			resource.TestCheckResourceAttr(resourceName, "authorized", "true"),
		),
	)
}

func TestAccTailscaleDeviceAuthorization_UsesNodeID(t *testing.T) {
	const resourceName = "tailscale_device_authorization.test_authorization"

	const testDeviceAuthorizationCreate = `
		data "tailscale_device" "test_device" {
			name = "%s"
		}
		
		resource "tailscale_device_authorization" "test_authorization" {
			device_id = data.tailscale_device.test_device.node_id
			authorized = true
		}`

	const testDeviceAuthorizationUpdate = `
		data "tailscale_device" "test_device" {
			name = "%s"
		}
		
		resource "tailscale_device_authorization" "test_authorization" {
			device_id = data.tailscale_device.test_device.node_id
			authorized = false
		}`

	const testDeviceAuthorizationNextUpdate = `
		data "tailscale_device" "test_device" {
			name = "%s"
		}
		
		resource "tailscale_device_authorization" "test_authorization" {
			device_id = data.tailscale_device.test_device.node_id
			authorized = true
		}`

	checkAuthorized := func(wantAuthorized bool) func(client *tailscale.Client, rs *terraform.ResourceState) error {
		return func(client *tailscale.Client, rs *terraform.ResourceState) error {
			device, err := client.Devices().Get(context.Background(), rs.Primary.ID)
			if err != nil {
				return err
			}

			if device.Authorized != wantAuthorized {
				return fmt.Errorf("device with id %q is not authorized", rs.Primary.ID)
			}

			return nil
		}
	}

	checkNodeID := func(client *tailscale.Client, rs *terraform.ResourceState) error {
		// Check that the device ID and State ID Match
		device, err := client.Devices().Get(context.Background(), rs.Primary.ID)
		if err != nil {
			return err
		}

		if device.NodeID != rs.Primary.ID {
			return fmt.Errorf("state id %q does not match node id %q", rs.Primary.ID, device.NodeID)
		}

		return nil
	}

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV5ProviderFactories: testAccProviderFactories(t),
		// Devices are not currently deauthorized when this resource is deleted,
		// so if the resource is authorized when the resource gets deleted,
		// expect that the device both exists and is still authorized.
		CheckDestroy: checkResourceDestroyed(resourceName, checkAuthorized(true)),
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(testDeviceAuthorizationCreate, os.Getenv("TAILSCALE_TEST_DEVICE_NAME")),
				Check: resource.ComposeTestCheckFunc(
					checkResourceRemoteProperties(resourceName, checkAuthorized(true)),
					checkResourceRemoteProperties(resourceName, checkNodeID),
					resource.TestCheckResourceAttr(resourceName, "authorized", "true"),
				),
			},
			{
				Config: fmt.Sprintf(testDeviceAuthorizationUpdate, os.Getenv("TAILSCALE_TEST_DEVICE_NAME")),
				Check: resource.ComposeTestCheckFunc(
					checkResourceRemoteProperties(resourceName, checkAuthorized(false)),
					checkResourceRemoteProperties(resourceName, checkNodeID),
					resource.TestCheckResourceAttr(resourceName, "authorized", "false"),
				),
			},
			{
				Config: fmt.Sprintf(testDeviceAuthorizationNextUpdate, os.Getenv("TAILSCALE_TEST_DEVICE_NAME")),
				Check: resource.ComposeTestCheckFunc(
					checkResourceRemoteProperties(resourceName, checkAuthorized(true)),
					checkResourceRemoteProperties(resourceName, checkNodeID),
					resource.TestCheckResourceAttr(resourceName, "authorized", "true"),
				),
			},
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})

	checkResourceIsUnchangedInPluginFramework(t,
		fmt.Sprintf(testDeviceAuthorizationCreate, os.Getenv("TAILSCALE_TEST_DEVICE_NAME")),
		resource.ComposeTestCheckFunc(
			checkResourceRemoteProperties(resourceName, checkNodeID),
			checkResourceRemoteProperties(resourceName, checkAuthorized(true)),
			resource.TestCheckResourceAttr(resourceName, "authorized", "true"),
		),
	)
}

// deviceAuthorizationAPI is a mock of the Tailscale device API endpoints used by
// tailscale_device_authorization. It records every request it receives.
type deviceAuthorizationAPI struct {
	mu         sync.Mutex
	requests   []string
	deleteCode int
	deleted    bool
}

func (a *deviceAuthorizationAPI) handle(method, path string) TestResponse {
	a.mu.Lock()
	defer a.mu.Unlock()

	a.requests = append(a.requests, method+" "+path)

	notFound := TestResponse{Code: http.StatusNotFound, Body: map[string]string{"message": "device not found"}}

	switch {
	case method == http.MethodGet && path == "/api/v2/device/nodeABC123CNTRL":
		if a.deleted {
			return notFound
		}
		return TestResponse{Code: http.StatusOK, Body: tailscale.Device{
			ID:         "123456789",
			NodeID:     "nodeABC123CNTRL",
			Authorized: true,
		}}
	case method == http.MethodPost && path == "/api/v2/device/nodeABC123CNTRL/authorized":
		return TestResponse{Code: http.StatusOK}
	case method == http.MethodDelete && path == "/api/v2/device/nodeABC123CNTRL":
		if a.deleteCode == http.StatusNotFound {
			return notFound
		}
		a.deleted = true
		return TestResponse{Code: http.StatusOK}
	}

	return TestResponse{Code: http.StatusInternalServerError, Body: map[string]string{"message": "unexpected request " + method + " " + path}}
}

// count returns how many requests matched method and path, and clears the record.
func (a *deviceAuthorizationAPI) count(method, path string) int {
	a.mu.Lock()
	defer a.mu.Unlock()

	n := 0
	for _, r := range a.requests {
		if r == method+" "+path {
			n++
		}
	}
	a.requests = nil
	return n
}

// expectRequests is a check that asserts how many times method and path
// were requested since the previous check.
func (a *deviceAuthorizationAPI) expectRequests(method, path string, want int) resource.TestCheckFunc {
	return func(_ *terraform.State) error {
		if got := a.count(method, path); got != want {
			return fmt.Errorf("got %d %s %s requests, want %d", got, method, path, want)
		}
		return nil
	}
}

func newDeviceAuthorizationAPI(deleteCode int) *deviceAuthorizationAPI {
	api := &deviceAuthorizationAPI{deleteCode: deleteCode}
	testServer.HandleRequest = api.handle
	return api
}

const (
	testDeviceAuthorizationResourceName = "tailscale_device_authorization.test"
	testDeviceAuthorizationDevicePath   = "/api/v2/device/nodeABC123CNTRL"

	// testDeviceAuthorizationRemoved is the configuration after the resource
	// block (and any import block) has been removed.
	testDeviceAuthorizationRemoved = `provider "tailscale" {}`
)

func testDeviceAuthorizationConfig(extra string) string {
	return `
		resource "tailscale_device_authorization" "test" {
			device_id  = "nodeABC123CNTRL"
			authorized = true
			` + extra + `
		}`
}

func TestProvider_TailscaleDeviceAuthorization_DeleteOnDestroy(t *testing.T) {
	var api *deviceAuthorizationAPI

	resource.Test(t, resource.TestCase{
		IsUnitTest:               true,
		ProtoV5ProviderFactories: testProviderFactories(t),
		PreCheck:                 func() { api = newDeviceAuthorizationAPI(http.StatusOK) },
		Steps: []resource.TestStep{
			{
				Config: testDeviceAuthorizationConfig("delete_on_destroy = true"),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(testDeviceAuthorizationResourceName, "delete_on_destroy", "true"),
					func(s *terraform.State) error {
						return api.expectRequests(http.MethodDelete, testDeviceAuthorizationDevicePath, 0)(s)
					},
				),
			},
			{
				// Removing the resource from the configuration destroys it
				// using delete_on_destroy from the prior state.
				Config: testDeviceAuthorizationRemoved,
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction(testDeviceAuthorizationResourceName, plancheck.ResourceActionDestroy),
					},
				},
				Check: func(s *terraform.State) error {
					return api.expectRequests(http.MethodDelete, testDeviceAuthorizationDevicePath, 1)(s)
				},
			},
		},
	})
}

func TestProvider_TailscaleDeviceAuthorization_NoDeleteOnDestroy(t *testing.T) {
	var api *deviceAuthorizationAPI

	resource.Test(t, resource.TestCase{
		IsUnitTest:               true,
		ProtoV5ProviderFactories: testProviderFactories(t),
		PreCheck:                 func() { api = newDeviceAuthorizationAPI(http.StatusOK) },
		Steps: []resource.TestStep{
			{
				Config: testDeviceAuthorizationConfig("delete_on_destroy = true"),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(testDeviceAuthorizationResourceName, "delete_on_destroy", "true"),
					func(s *terraform.State) error {
						return api.expectRequests(http.MethodPost, testDeviceAuthorizationDevicePath+"/authorized", 1)(s)
					},
				),
			},
			{
				// Turning the flag off is an in-place update that only changes state.
				Config: testDeviceAuthorizationConfig(""),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction(testDeviceAuthorizationResourceName, plancheck.ResourceActionUpdate),
					},
				},
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(testDeviceAuthorizationResourceName, "delete_on_destroy", "false"),
					func(s *terraform.State) error {
						return api.expectRequests(http.MethodPost, testDeviceAuthorizationDevicePath+"/authorized", 0)(s)
					},
				),
			},
			{
				Config: testDeviceAuthorizationRemoved,
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction(testDeviceAuthorizationResourceName, plancheck.ResourceActionDestroy),
					},
				},
				Check: func(s *terraform.State) error {
					return api.expectRequests(http.MethodDelete, testDeviceAuthorizationDevicePath, 0)(s)
				},
			},
		},
	})
}

func TestProvider_TailscaleDeviceAuthorization_DeleteOnDestroyNotFound(t *testing.T) {
	var api *deviceAuthorizationAPI

	resource.Test(t, resource.TestCase{
		IsUnitTest:               true,
		ProtoV5ProviderFactories: testProviderFactories(t),
		PreCheck:                 func() { api = newDeviceAuthorizationAPI(http.StatusNotFound) },
		Steps: []resource.TestStep{
			{
				Config: testDeviceAuthorizationConfig("delete_on_destroy = true"),
			},
			{
				// The device is gone by the time the DELETE is sent, which is not an error.
				Config: testDeviceAuthorizationRemoved,
				Check: func(s *terraform.State) error {
					return api.expectRequests(http.MethodDelete, testDeviceAuthorizationDevicePath, 1)(s)
				},
			},
		},
	})
}

func TestProvider_TailscaleDeviceAuthorization_Import(t *testing.T) {
	const importBlock = `
		import {
			to = tailscale_device_authorization.test
			id = "nodeABC123CNTRL"
		}`

	resource.Test(t, resource.TestCase{
		IsUnitTest:               true,
		ProtoV5ProviderFactories: testProviderFactories(t),
		PreCheck:                 func() { newDeviceAuthorizationAPI(http.StatusOK) },
		Steps: []resource.TestStep{
			{
				// Importing a device without setting delete_on_destroy leaves nothing to change.
				Config: importBlock + testDeviceAuthorizationConfig(""),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction(testDeviceAuthorizationResourceName, plancheck.ResourceActionNoop),
					},
				},
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(testDeviceAuthorizationResourceName, "id", "nodeABC123CNTRL"),
					resource.TestCheckResourceAttr(testDeviceAuthorizationResourceName, "delete_on_destroy", "false"),
				),
			},
			{
				// The import command produces the same state.
				ResourceName:      testDeviceAuthorizationResourceName,
				ImportState:       true,
				ImportStateId:     "nodeABC123CNTRL",
				ImportStateVerify: true,
			},
		},
	})
}

func TestProvider_TailscaleDeviceAuthorization_ImportThenDeleteOnDestroy(t *testing.T) {
	var api *deviceAuthorizationAPI

	const importBlock = `
		import {
			to = tailscale_device_authorization.test
			id = "nodeABC123CNTRL"
		}`

	resource.Test(t, resource.TestCase{
		IsUnitTest:               true,
		ProtoV5ProviderFactories: testProviderFactories(t),
		PreCheck:                 func() { api = newDeviceAuthorizationAPI(http.StatusOK) },
		Steps: []resource.TestStep{
			{
				// Import a device that was approved outside Terraform and set the flag
				// in the same apply. This updates state only.
				Config: importBlock + testDeviceAuthorizationConfig("delete_on_destroy = true"),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction(testDeviceAuthorizationResourceName, plancheck.ResourceActionUpdate),
					},
				},
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(testDeviceAuthorizationResourceName, "delete_on_destroy", "true"),
					func(s *terraform.State) error {
						return api.expectRequests(http.MethodPost, testDeviceAuthorizationDevicePath+"/authorized", 0)(s)
					},
				),
			},
			{
				// Removing both blocks deletes the device.
				Config: testDeviceAuthorizationRemoved,
				Check: func(s *terraform.State) error {
					return api.expectRequests(http.MethodDelete, testDeviceAuthorizationDevicePath, 1)(s)
				},
			},
		},
	})
}

func TestDeviceAuthorizationModifyPlanWarnings(t *testing.T) {
	ctx := context.Background()

	r := deviceAuthorizationResource{}
	var schemaResp fwresource.SchemaResponse
	r.Schema(ctx, fwresource.SchemaRequest{}, &schemaResp)
	objType := schemaResp.Schema.Type().TerraformType(ctx)

	value := func(deviceID string, deleteOnDestroy bool) tftypes.Value {
		return tftypes.NewValue(objType, map[string]tftypes.Value{
			"id":                tftypes.NewValue(tftypes.String, "nodeABC123CNTRL"),
			"device_id":         tftypes.NewValue(tftypes.String, deviceID),
			"authorized":        tftypes.NewValue(tftypes.Bool, true),
			"delete_on_destroy": tftypes.NewValue(tftypes.Bool, deleteOnDestroy),
		})
	}
	null := tftypes.NewValue(objType, nil)

	tests := []struct {
		name        string
		state, plan tftypes.Value
		want        []string
	}{
		{
			name:  "create",
			state: null,
			plan:  value("nodeABC123CNTRL", true),
		},
		{
			name:  "destroy without delete_on_destroy",
			state: value("nodeABC123CNTRL", false),
			plan:  null,
			want:  []string{"will only remove the resource from the Terraform state"},
		},
		{
			name:  "destroy with delete_on_destroy",
			state: value("nodeABC123CNTRL", true),
			plan:  null,
			want:  []string{"will delete the device nodeABC123CNTRL from the tailnet"},
		},
		{
			name:  "update with delete_on_destroy",
			state: value("nodeABC123CNTRL", true),
			plan:  value("nodeABC123CNTRL", false),
		},
		{
			name:  "replace without delete_on_destroy",
			state: value("nodeABC123CNTRL", false),
			plan:  value("nodeDEF456CNTRL", false),
		},
		{
			name:  "replace with delete_on_destroy",
			state: value("nodeABC123CNTRL", true),
			plan:  value("nodeDEF456CNTRL", true),
			want:  []string{"the replacement will delete the device nodeABC123CNTRL"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := fwresource.ModifyPlanRequest{
				State: tfsdk.State{Schema: schemaResp.Schema, Raw: tt.state},
				Plan:  tfsdk.Plan{Schema: schemaResp.Schema, Raw: tt.plan},
			}
			resp := fwresource.ModifyPlanResponse{
				Plan: tfsdk.Plan{Schema: schemaResp.Schema, Raw: tt.plan},
			}
			r.ModifyPlan(ctx, req, &resp)

			if resp.Diagnostics.HasError() {
				t.Fatalf("unexpected error: %v", resp.Diagnostics)
			}
			warnings := resp.Diagnostics.Warnings()
			if len(warnings) != len(tt.want) {
				t.Fatalf("got %d warnings, want %d: %v", len(warnings), len(tt.want), warnings)
			}
			for i, want := range tt.want {
				if !strings.Contains(warnings[i].Detail(), want) {
					t.Errorf("warning %q does not contain %q", warnings[i].Detail(), want)
				}
			}
		})
	}
}
