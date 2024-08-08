resource "tailscale_logstream_configuration" "sample_logstream_configuration" {
  log_type         = "configuration"
  destination_type = "panther"
  url              = "https://example.com"
  token            = "some-token"
}
