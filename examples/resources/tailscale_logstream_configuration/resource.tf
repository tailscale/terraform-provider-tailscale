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

# Example configuration for a GCS logstreaming endpoint using workload identity

resource "tailscale_logstream_configuration" "sample_logstream_configuration_gcs_wif" {
  log_type         = "configuration"
  destination_type = "gcs"
  gcs_bucket       = "example-gcs-bucket"
  gcs_credentials  = jsonencode({
    type = "external_account"
    audience = "//iam.googleapis.com/projects/12345678/locations/global/workloadIdentityPools/example-pool/providers/example-provider"
    subject_token_type = "urn:ietf:params:aws:token-type:aws4_request"
    service_account_impersonation_url = "https://iamcredentials.googleapis.com/v1/projects/-/serviceAccounts/example@example.iam.gserviceaccount.com:generateAccessToken"
    token_url = "https://sts.googleapis.com/v1/token"
    credential_source = {
      environment_id = "aws1"
      region_url = "http://169.254.169.254/latest/meta-data/placement/availability-zone"
      url = "http://169.254.169.254/latest/meta-data/iam/security-credentials"
      regional_cred_verification_url = "https://sts.{region}.amazonaws.com?Action=GetCallerIdentity&Version=2011-06-15"
      imdsv2_session_token_url = "http://169.254.169.254/latest/api/token"
    }
  })
}


