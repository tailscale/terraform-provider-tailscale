package tailscale

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

const testLogstreamConfigurationUpdateSameLogtype = `
	resource "tailscale_logstream_configuration" "test_logstream_configuration" {
		log_type         = "configuration"
		destination_type = "cribl"
		url              = "https://example.com/other"
		token            = "some-token"
	}`

const testLogstreamConfigurationUpdateDifferentLogtype = `
	resource "tailscale_logstream_configuration" "test_logstream_configuration" {
		log_type         = "network"
		destination_type = "datadog"
		url              = "https://example.com/other/other"
		token            = "some-token"
	}`

func TestAccTailscaleLogstreamConfiguration_basic(t *testing.T) {
	const resourceName = "tailscale_logstream_configuration.test_logstream_configuration"

	checkProperties := func(expectedConfiguration tsclient.LogstreamConfiguration) func(client *tsclient.Client, rs *terraform.ResourceState) error {
		return func(client *tsclient.Client, rs *terraform.ResourceState) error {
			var selectedConfig *tsclient.LogstreamConfiguration
			logstreamConfigurationLogTypeConfig, err := client.Logging().LogstreamConfiguration(context.Background(), tsclient.LogTypeConfig)
			if expectedConfiguration.LogType == tsclient.LogTypeConfig {
				if err != nil {
					return err
				} else {
					selectedConfig = logstreamConfigurationLogTypeConfig
				}
			} else if expectedConfiguration.LogType == tsclient.LogTypeNetwork && err == nil {
				return fmt.Errorf("expected no configuration logstream configuration but got %+v", logstreamConfigurationLogTypeConfig)
			}

			logstreamConfigurationLogTypeNetwork, err := client.Logging().LogstreamConfiguration(context.Background(), tsclient.LogTypeNetwork)
			if expectedConfiguration.LogType == tsclient.LogTypeNetwork {
				if err != nil {
					return err
				} else {
					selectedConfig = logstreamConfigurationLogTypeNetwork
				}
			} else if expectedConfiguration.LogType == tsclient.LogTypeNetwork && err == nil {
				return fmt.Errorf("expected no network logstream configuration but got %+v", logstreamConfigurationLogTypeNetwork)
			}

			if selectedConfig.LogType != expectedConfiguration.LogType {
				return fmt.Errorf("bad logstream_configuration.log_type: %s", selectedConfig.LogType)
			}
			if selectedConfig.DestinationType != expectedConfiguration.DestinationType {
				return fmt.Errorf("bad logstream_configuration.destination_type: %s", selectedConfig.DestinationType)
			}
			if selectedConfig.URL != expectedConfiguration.URL {
				return fmt.Errorf("bad logstream_configuration.url: %s", selectedConfig.URL)
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
						checkProperties(tsclient.LogstreamConfiguration{
							LogType:         tsclient.LogTypeConfig,
							DestinationType: tsclient.LogstreamPantherEndpoint,
							URL:             "https://example.com",
						}),
					),
					resource.TestCheckResourceAttr(resourceName, "log_type", "configuration"),
					resource.TestCheckResourceAttr(resourceName, "destination_type", "panther"),
					resource.TestCheckResourceAttr(resourceName, "url", "https://example.com"),
					resource.TestCheckResourceAttr(resourceName, "token", "some-token"),
				),
			},
			{
				Config: testLogstreamConfigurationUpdateSameLogtype,
				Check: resource.ComposeTestCheckFunc(
					checkResourceRemoteProperties(
						resourceName,
						checkProperties(tsclient.LogstreamConfiguration{
							LogType:         tsclient.LogTypeConfig,
							DestinationType: tsclient.LogstreamCriblEndpoint,
							URL:             "https://example.com/other",
						}),
					),
					resource.TestCheckResourceAttr(resourceName, "log_type", "configuration"),
					resource.TestCheckResourceAttr(resourceName, "destination_type", "cribl"),
					resource.TestCheckResourceAttr(resourceName, "url", "https://example.com/other"),
					resource.TestCheckResourceAttr(resourceName, "token", "some-token"),
				),
			},
			{
				Config: testLogstreamConfigurationUpdateDifferentLogtype,
				Check: resource.ComposeTestCheckFunc(
					checkResourceRemoteProperties(
						resourceName,
						checkProperties(tsclient.LogstreamConfiguration{
							LogType:         tsclient.LogTypeNetwork,
							DestinationType: tsclient.LogstreamDatadogEndpoint,
							URL:             "https://example.com/other/other",
						}),
					),
					resource.TestCheckResourceAttr(resourceName, "log_type", "network"),
					resource.TestCheckResourceAttr(resourceName, "destination_type", "datadog"),
					resource.TestCheckResourceAttr(resourceName, "url", "https://example.com/other/other"),
					resource.TestCheckResourceAttr(resourceName, "token", "some-token"),
				),
			},
		},
	})
}
