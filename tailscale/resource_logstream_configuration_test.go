// Copyright (c) David Bond, Tailscale Inc, & Contributors
// SPDX-License-Identifier: MIT

package tailscale

import (
	"context"
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"

	"tailscale.com/client/tailscale/v2"
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
		user             = "cribl-user"
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

const testLogstreamConfigurationUpdateS3RoleARN = `
	resource "tailscale_logstream_configuration" "test_logstream_configuration" {
		log_type               = "network"
		destination_type       = "s3"
		s3_bucket              = "example-bucket"
		s3_region              = "us-west-2"
		s3_key_prefix          = "logs/"
		s3_authentication_type = "rolearn"
		s3_role_arn            = "arn:aws:iam::123456789012:role/example-role"
		s3_external_id         = tailscale_aws_external_id.external_id.external_id
	}
	resource "tailscale_aws_external_id" "external_id" {}
	`

const testLogstreamConfigurationUpdateS3AccessKey = `
	resource "tailscale_logstream_configuration" "test_logstream_configuration" {
		log_type               = "network"
		destination_type       = "s3"
		s3_bucket              = "example-bucket"
		s3_region			   = "us-west-2"
		s3_authentication_type = "accesskey"
		s3_access_key_id       = "example-access-key-id"
		s3_secret_access_key   = "example-secret-access-key"
		url                    = "https://example.com/s3"
		upload_period_minutes  = 5
		compression_format     = "zstd"
	}`

func TestAccTailscaleLogstreamConfiguration(t *testing.T) {
	const resourceName = "tailscale_logstream_configuration.test_logstream_configuration"

	checkProperties := func(expectedConfiguration tailscale.LogstreamConfiguration) func(client *tailscale.Client, rs *terraform.ResourceState) error {
		return func(client *tailscale.Client, rs *terraform.ResourceState) error {
			var selectedConfig *tailscale.LogstreamConfiguration
			logstreamConfigurationLogTypeConfig, err := client.Logging().LogstreamConfiguration(context.Background(), tailscale.LogTypeConfig)
			if expectedConfiguration.LogType == tailscale.LogTypeConfig {
				if err != nil {
					return err
				} else {
					selectedConfig = logstreamConfigurationLogTypeConfig
				}
			} else if expectedConfiguration.LogType == tailscale.LogTypeNetwork && err == nil {
				return fmt.Errorf("expected no configuration logstream configuration but got %+v", logstreamConfigurationLogTypeConfig)
			}

			logstreamConfigurationLogTypeNetwork, err := client.Logging().LogstreamConfiguration(context.Background(), tailscale.LogTypeNetwork)
			if expectedConfiguration.LogType == tailscale.LogTypeNetwork {
				if err != nil {
					return err
				} else {
					selectedConfig = logstreamConfigurationLogTypeNetwork
				}
			} else if expectedConfiguration.LogType == tailscale.LogTypeNetwork && err == nil {
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
			if selectedConfig.S3Bucket != expectedConfiguration.S3Bucket {
				return fmt.Errorf("bad logstream_configuration.s3_bucket: %s", selectedConfig.S3Bucket)
			}
			if selectedConfig.S3Region != expectedConfiguration.S3Region {
				return fmt.Errorf("bad logstream_configuration.s3_region: %s", selectedConfig.S3Region)
			}
			if selectedConfig.S3KeyPrefix != expectedConfiguration.S3KeyPrefix {
				return fmt.Errorf("bad logstream_configuration.s3_key_prefix: %s", selectedConfig.S3KeyPrefix)
			}
			if selectedConfig.S3AuthenticationType != expectedConfiguration.S3AuthenticationType {
				return fmt.Errorf("bad logstream_configuration.s3_authentication_type: %s", selectedConfig.S3AuthenticationType)
			}
			if selectedConfig.S3AccessKeyID != expectedConfiguration.S3AccessKeyID {
				return fmt.Errorf("bad logstream_configuration.s3_access_key_id: %s", selectedConfig.S3AccessKeyID)
			}
			if selectedConfig.S3RoleARN != expectedConfiguration.S3RoleARN {
				return fmt.Errorf("bad logstream_configuration.s3_role_arn: %s", selectedConfig.S3RoleARN)
			}
			if selectedConfig.S3ExternalID != expectedConfiguration.S3ExternalID {
				return fmt.Errorf("bad logstream_configuration.s3_external_id: %s", selectedConfig.S3ExternalID)
			}

			if selectedConfig.User != expectedConfiguration.User {
				// We have a default value of user = 'user'.
				if expectedConfiguration.User != "" || selectedConfig.User != "user" {
					return fmt.Errorf("bad logstream_configuration.user: %s", selectedConfig.User)
				}
			}

			if selectedConfig.UploadPeriodMinutes != expectedConfiguration.UploadPeriodMinutes {
				return fmt.Errorf("bad logstream_configuration.upload_period_minutes: %d", selectedConfig.UploadPeriodMinutes)
			}

			if selectedConfig.CompressionFormat != expectedConfiguration.CompressionFormat {
				// We have a default value of compression_format = 'none'.
				if expectedConfiguration.CompressionFormat != "" || selectedConfig.CompressionFormat != tailscale.CompressionFormatNone {
					return fmt.Errorf("bad logstream_configuration.compression_format: %s", selectedConfig.CompressionFormat)
				}
			}

			return nil
		}
	}

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories(t),
		CheckDestroy: checkResourceDestroyed(resourceName, func(client *tailscale.Client, rs *terraform.ResourceState) error {
			_, err := client.Logging().LogstreamConfiguration(context.Background(), tailscale.LogType(rs.Primary.ID))
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
						checkProperties(tailscale.LogstreamConfiguration{
							LogType:         tailscale.LogTypeConfig,
							DestinationType: tailscale.LogstreamPantherEndpoint,
							URL:             "https://example.com",
						}),
					),
					resource.TestCheckResourceAttr(resourceName, "log_type", "configuration"),
					resource.TestCheckResourceAttr(resourceName, "destination_type", "panther"),
					resource.TestCheckResourceAttr(resourceName, "url", "https://example.com"),
					resource.TestCheckResourceAttr(resourceName, "user", "user"),
					resource.TestCheckResourceAttr(resourceName, "token", "some-token"),
				),
			},
			{
				Config: testLogstreamConfigurationUpdateSameLogtype,
				Check: resource.ComposeTestCheckFunc(
					checkResourceRemoteProperties(
						resourceName,
						checkProperties(tailscale.LogstreamConfiguration{
							LogType:         tailscale.LogTypeConfig,
							DestinationType: tailscale.LogstreamCriblEndpoint,
							URL:             "https://example.com/other",
							User:            "cribl-user",
						}),
					),
					resource.TestCheckResourceAttr(resourceName, "log_type", "configuration"),
					resource.TestCheckResourceAttr(resourceName, "destination_type", "cribl"),
					resource.TestCheckResourceAttr(resourceName, "url", "https://example.com/other"),
					resource.TestCheckResourceAttr(resourceName, "user", "cribl-user"),
					resource.TestCheckResourceAttr(resourceName, "token", "some-token"),
				),
			},
			{
				Config: testLogstreamConfigurationUpdateDifferentLogtype,
				Check: resource.ComposeTestCheckFunc(
					checkResourceRemoteProperties(
						resourceName,
						checkProperties(tailscale.LogstreamConfiguration{
							LogType:         tailscale.LogTypeNetwork,
							DestinationType: tailscale.LogstreamDatadogEndpoint,
							URL:             "https://example.com/other/other",
						}),
					),
					resource.TestCheckResourceAttr(resourceName, "log_type", "network"),
					resource.TestCheckResourceAttr(resourceName, "destination_type", "datadog"),
					resource.TestCheckResourceAttr(resourceName, "url", "https://example.com/other/other"),
					resource.TestCheckResourceAttr(resourceName, "user", "user"),
					resource.TestCheckResourceAttr(resourceName, "token", "some-token"),
				),
			},
			{
				ResourceName:            resourceName,
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"token"},
			},
			{
				Config: testLogstreamConfigurationUpdateS3RoleARN,
				Check: resource.ComposeTestCheckFunc(
					func(s *terraform.State) error {
						externalIdResource, ok := s.RootModule().Resources["tailscale_aws_external_id.external_id"]
						if !ok {
							return fmt.Errorf("resource not found: tailscale_aws_external_id.external_id")
						}

						return checkResourceRemoteProperties(
							resourceName,
							checkProperties(tailscale.LogstreamConfiguration{
								LogType:              tailscale.LogTypeNetwork,
								DestinationType:      tailscale.LogstreamS3Endpoint,
								S3Bucket:             "example-bucket",
								S3Region:             "us-west-2",
								S3KeyPrefix:          "logs/",
								S3AuthenticationType: tailscale.S3RoleARNAuthentication,
								S3RoleARN:            "arn:aws:iam::123456789012:role/example-role",
								S3ExternalID:         externalIdResource.Primary.Attributes["external_id"],
								CompressionFormat:    tailscale.CompressionFormatNone,
							}),
						)(s)
					},
					resource.TestCheckResourceAttr(resourceName, "log_type", "network"),
					resource.TestCheckResourceAttr(resourceName, "destination_type", "s3"),
					resource.TestCheckResourceAttr(resourceName, "s3_bucket", "example-bucket"),
					resource.TestCheckResourceAttr(resourceName, "s3_region", "us-west-2"),
					resource.TestCheckResourceAttr(resourceName, "s3_key_prefix", "logs/"),
					resource.TestCheckResourceAttr(resourceName, "s3_authentication_type", "rolearn"),
					resource.TestCheckResourceAttr(resourceName, "s3_role_arn", "arn:aws:iam::123456789012:role/example-role"),
					resource.TestCheckResourceAttrPair(resourceName, "s3_external_id", "tailscale_aws_external_id.external_id", "external_id"),
					resource.TestCheckResourceAttr(resourceName, "compression_format", "none"),
				),
			},
			{
				Config: testLogstreamConfigurationUpdateS3AccessKey,
				Check: resource.ComposeTestCheckFunc(
					checkResourceRemoteProperties(
						resourceName,
						checkProperties(tailscale.LogstreamConfiguration{
							LogType:              tailscale.LogTypeNetwork,
							DestinationType:      tailscale.LogstreamS3Endpoint,
							S3Bucket:             "example-bucket",
							S3Region:             "us-west-2",
							S3AuthenticationType: tailscale.S3AccessKeyAuthentication,
							S3AccessKeyID:        "example-access-key-id",
							URL:                  "https://example.com/s3",
							UploadPeriodMinutes:  5,
							CompressionFormat:    tailscale.CompressionFormatZstd,
						}),
					),
					resource.TestCheckResourceAttr(resourceName, "log_type", "network"),
					resource.TestCheckResourceAttr(resourceName, "destination_type", "s3"),
					resource.TestCheckResourceAttr(resourceName, "s3_bucket", "example-bucket"),
					resource.TestCheckResourceAttr(resourceName, "s3_region", "us-west-2"),
					resource.TestCheckResourceAttr(resourceName, "s3_authentication_type", "accesskey"),
					resource.TestCheckResourceAttr(resourceName, "s3_access_key_id", "example-access-key-id"),
					resource.TestCheckResourceAttr(resourceName, "s3_secret_access_key", "example-secret-access-key"),
					resource.TestCheckResourceAttr(resourceName, "url", "https://example.com/s3"),
					resource.TestCheckResourceAttr(resourceName, "upload_period_minutes", "5"),
					resource.TestCheckResourceAttr(resourceName, "compression_format", "zstd"),
				),
			},
			{
				ResourceName:            resourceName,
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"token", "s3_secret_access_key"},
			},
		},
	})
}
