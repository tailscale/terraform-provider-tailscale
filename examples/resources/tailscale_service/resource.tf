resource "tailscale_service" "example" {
  name    = "svc:my-service"
  comment = "My service"
  ports   = ["tcp:443"]
  tags    = ["tag:web"]
}
