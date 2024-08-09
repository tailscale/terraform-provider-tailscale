package tailscale_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"

	tsclient "github.com/tailscale/tailscale-client-go/v2"
)

const testLogStreamConfiguration = `
	resource "tailscale_logstream_configuration" "test_logstream_configuration" {
		log_type         = "configuration"
		destination_type = "panther"
		url              = "https://example.com"
		token            = "some-token"
	}`

const testLogstreamConfigurationUpdate = `
	resource "tailscale_logstream_configuration" "test_logstream_configuration" {
		log_type         = "network"
		destination_type = "cribl"
		url              = "https://example-other.com"
		token            = "some-token"
	}`

func TestAccTailscaleLogstreamConfiguration_basic(t *testing.T) {
	const resourceName = "tailscale_logstream_configuration.test_logstream_configuration"

	checkProperties := func(expectedLogType tsclient.LogType, expectedDestinationType tsclient.LogstreamEndpointType, expectedURL string) func(client *tsclient.Client, rs *terraform.ResourceState) error {
		return func(client *tsclient.Client, rs *terraform.ResourceState) error {
			logstreamConfiguration, err := client.Logging().LogstreamConfiguration(context.Background(), tsclient.LogType(rs.Primary.ID))
			if err != nil {
				return err
			}

			if logstreamConfiguration.LogType != expectedLogType {
				return fmt.Errorf("bad logstream_configuration.log_type: %s", logstreamConfiguration.LogType)
			}
			if logstreamConfiguration.DestinationType != expectedDestinationType {
				return fmt.Errorf("bad logstream_configuration.destination_type: %s", logstreamConfiguration.DestinationType)
			}
			if logstreamConfiguration.URL != expectedURL {
				return fmt.Errorf("bad logstream_configuration.url: %s", logstreamConfiguration.URL)
			}
			return nil
		}
	}

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories(t),
		CheckDestroy: checkResourceDestroyed(resourceName, func(client *tsclient.Client, rs *terraform.ResourceState) error {
			_, err := client.Logging().LogstreamConfiguration(context.Background(), tsclient.LogType(rs.Primary.ID))
			if err == nil {
				return fmt.Errorf("logstream configuration %q still exists on server", resourceName)
			}
			return nil
		}),
		Steps: []resource.TestStep{
			{
				Config: testLogStreamConfiguration,
				Check: resource.ComposeTestCheckFunc(
					checkResourceRemoteProperties(
						resourceName,
						checkProperties(tsclient.LogTypeConfig, tsclient.LogstreamPantherEndpoint, "https://example.com"),
					),
					resource.TestCheckResourceAttr(resourceName, "log_type", "configuration"),
					resource.TestCheckResourceAttr(resourceName, "destination_type", "panther"),
					resource.TestCheckResourceAttr(resourceName, "url", "https://example.com"),
					resource.TestCheckResourceAttr(resourceName, "token", "some-token"),
				),
			},
			{
				Config: testLogstreamConfigurationUpdate,
				Check: resource.ComposeTestCheckFunc(
					checkResourceRemoteProperties(
						resourceName,
						checkProperties(tsclient.LogTypeNetwork, tsclient.LogstreamCriblEndpoint, "https://example-other.com"),
					),
					resource.TestCheckResourceAttr(resourceName, "log_type", "network"),
					resource.TestCheckResourceAttr(resourceName, "destination_type", "cribl"),
					resource.TestCheckResourceAttr(resourceName, "url", "https://example-other.com"),
					resource.TestCheckResourceAttr(resourceName, "token", "some-token"),
				),
			},
		},
	})
}
