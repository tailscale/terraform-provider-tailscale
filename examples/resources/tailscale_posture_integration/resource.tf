resource "tailscale_posture_integration" "sample_posture_integration" {
  posture_provider = "falcon"
  cloud_id         = "us-1"
  client_id        = "clientid1"
  client_secret    = "test-secret1"
}
