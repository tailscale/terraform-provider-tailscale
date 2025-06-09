data "tailscale_device" "example_device" {
  name = "device.example.com"
}

resource "tailscale_device_key" "example_key" {
  # Prefer the new, stable `node_id` attribute; the legacy `.id` field still works.
  device_id           = data.tailscale_device.example_device.node_id
  key_expiry_disabled = true
}
