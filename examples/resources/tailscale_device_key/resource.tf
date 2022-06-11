data "tailscale_device" "example_device" {
  name = "device.example.com"
}

resource "tailscale_device_key" "example_key" {
  device_id           = data.tailscale_device.example_device.id
  key_expiry_disabled = true
}
