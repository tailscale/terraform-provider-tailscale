# Requires Terraform 1.11+ or OpenTofu 1.11+.
variable "splunk_hec_token" {
  description = "The Splunk HEC token used to authenticate log streaming."
  type        = string
  sensitive   = true
  ephemeral   = true
}

resource "tailscale_logstream_configuration" "splunk" {
  log_type         = "configuration"
  destination_type = "splunk"
  url              = "https://splunk.example.com/services/collector/event"
  token_wo         = var.splunk_hec_token
  token_wo_version = 1 # Increment when rotating the token.
}
