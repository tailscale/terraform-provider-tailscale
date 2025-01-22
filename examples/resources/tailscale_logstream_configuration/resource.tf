# Example configuration for a non-S3 logstreaming endpoint

resource "tailscale_logstream_configuration" "sample_logstream_configuration" {
  log_type         = "configuration"
  destination_type = "panther"
  url              = "https://example.com"
  token            = "some-token"
}

# Example configuration for an AWS S3 logstreaming endpoint

resource "tailscale_logstream_configuration" "sample_logstream_configuration_s3" {
  log_type               = "configuration"
  destination_type       = "s3"
  s3_bucket              = aws_s3_bucket.tailscale_logs.id
  s3_region              = "us-west-2"
  s3_authentication_type = "rolearn"
  s3_role_arn            = aws_iam_role.tailscale_logs_writer.arn
  s3_external_id         = tailscale_aws_external_id.prod.external_id
}

# Example configuration for an S3-compatible logstreaming endpoint

resource "tailscale_logstream_configuration" "sample_logstream_configuration_s3_compatible" {
  log_type               = "configuration"
  destination_type       = "s3"
  url                    = "https://s3.example.com"
  s3_bucket              = "example-bucket"
  s3_region              = "us-west-2"
  s3_authentication_type = "accesskey"
  s3_access_key_id       = "some-access-key"
  s3_secret_access_key   = "some-secret-key"
}
