package tailscale

import (
	"context"
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"

	tsclient "github.com/tailscale/tailscale-client-go/v2"
)

const testPostureIntegrationCreate = `
	resource "tailscale_posture_integration" "test_posture_integration" {
		posture_provider      = "falcon"
		cloud_id              = "us-1"
		client_id             = "clientid1"
		client_secret         = "test-secret1"
	}`

const testPostureIntegrationUpdateSameProvider = `
	resource "tailscale_posture_integration" "test_posture_integration" {
		posture_provider      = "falcon"
		cloud_id              = "us-2"
		client_id             = "clientid2"
		client_secret         = "test-secret2"
	}`

const testPostureIntegrationUpdateDifferentProvider = `
	resource "tailscale_posture_integration" "test_posture_integration" {
		posture_provider      = "intune"
		cloud_id              = "global"
		client_id             = "fddf23ae-0e3a-4e0c-908d-6f44e80f9400"
		tenant_id             = "fddf23ae-0e3a-4e0c-908d-6f44e80f9401"
		client_secret         = "test-secret3"
	}`

func TestAccTailscalePostureIntegration(t *testing.T) {
	const resourceName = "tailscale_posture_integration.test_posture_integration"

	checkProperties := func(expected tsclient.PostureIntegration) func(client *tsclient.Client, rs *terraform.ResourceState) error {
		return func(client *tsclient.Client, rs *terraform.ResourceState) error {
			integration, err := client.DevicePosture().GetIntegration(context.Background(), rs.Primary.ID)
			if err != nil {
				return err
			}

			if integration.Provider != expected.Provider {
				return fmt.Errorf("wrong provider, want %q got %q", expected.Provider, integration.Provider)
			}
			if integration.CloudID != expected.CloudID {
				return fmt.Errorf("wrong cloud_id, want %q got %q", expected.CloudID, integration.CloudID)
			}
			if integration.ClientID != expected.ClientID {
				return fmt.Errorf("wrong client_id, want %q got %q", expected.ClientID, integration.ClientID)
			}
			if integration.TenantID != expected.TenantID {
				return fmt.Errorf("wrong tenant_id, want %q got %q", expected.TenantID, integration.TenantID)
			}

			return nil
		}
	}

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories(t),
		CheckDestroy: checkResourceDestroyed(resourceName, func(client *tsclient.Client, rs *terraform.ResourceState) error {
			_, err := client.DevicePosture().GetIntegration(context.Background(), rs.Primary.ID)
			if err == nil {
				return fmt.Errorf("posture integration %q still exists on server", resourceName)
			}

			return nil
		}),
		Steps: []resource.TestStep{
			{
				Config: testPostureIntegrationCreate,
				Check: resource.ComposeTestCheckFunc(
					checkResourceRemoteProperties(resourceName,
						checkProperties(tsclient.PostureIntegration{
							Provider: tsclient.PostureIntegrationProviderFalcon,
							CloudID:  "us-1",
							ClientID: "clientid1",
						}),
					),
					resource.TestCheckResourceAttr(resourceName, "posture_provider", "falcon"),
					resource.TestCheckResourceAttr(resourceName, "cloud_id", "us-1"),
					resource.TestCheckResourceAttr(resourceName, "client_id", "clientid1"),
					resource.TestCheckResourceAttr(resourceName, "client_secret", "test-secret1"),
				),
			},
			{
				Config: testPostureIntegrationUpdateSameProvider,
				Check: resource.ComposeTestCheckFunc(
					checkResourceRemoteProperties(resourceName,
						checkProperties(tsclient.PostureIntegration{
							Provider: tsclient.PostureIntegrationProviderFalcon,
							CloudID:  "us-2",
							ClientID: "clientid2",
						}),
					),
					resource.TestCheckResourceAttr(resourceName, "posture_provider", "falcon"),
					resource.TestCheckResourceAttr(resourceName, "cloud_id", "us-2"),
					resource.TestCheckResourceAttr(resourceName, "client_id", "clientid2"),
					resource.TestCheckResourceAttr(resourceName, "client_secret", "test-secret2"),
				),
			},
			{
				Config: testPostureIntegrationUpdateDifferentProvider,
				Check: resource.ComposeTestCheckFunc(
					checkResourceRemoteProperties(resourceName,
						checkProperties(tsclient.PostureIntegration{
							Provider: tsclient.PostureIntegrationProviderIntune,
							CloudID:  "global",
							ClientID: "fddf23ae-0e3a-4e0c-908d-6f44e80f9400",
							TenantID: "fddf23ae-0e3a-4e0c-908d-6f44e80f9401",
						}),
					),
					resource.TestCheckResourceAttr(resourceName, "posture_provider", "intune"),
					resource.TestCheckResourceAttr(resourceName, "cloud_id", "global"),
					resource.TestCheckResourceAttr(resourceName, "client_id", "fddf23ae-0e3a-4e0c-908d-6f44e80f9400"),
					resource.TestCheckResourceAttr(resourceName, "tenant_id", "fddf23ae-0e3a-4e0c-908d-6f44e80f9401"),
					resource.TestCheckResourceAttr(resourceName, "client_secret", "test-secret3"),
				),
			},
		},
	})
}
