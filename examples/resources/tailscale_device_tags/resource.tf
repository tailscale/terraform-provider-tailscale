data "tailscale_device" "sample_device" {
  name = "device.example.com"
}

resource "tailscale_device_tags" "sample_tags" {
  # Prefer the new, stable `node_id` attribute; the legacy `.id` field still works.
  device_id = data.tailscale_device.sample_device.node_id
  tags      = ["room:bedroom"]
}
