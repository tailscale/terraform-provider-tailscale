resource "tailscale_webhook" "sample_webhook" {
  endpoint_url  = "https://example.com/webhook/endpoint"
  provider_type = "slack"
  subscriptions = ["nodeCreated", "userDeleted"]
}