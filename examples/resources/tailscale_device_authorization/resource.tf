data "tailscale_device" "sample_device" {
  name = "device.example.com"
}

resource "tailscale_device_authorization" "sample_authorization" {
  # Prefer the new, stable `node_id` attribute; the legacy `.id` field still works.
  device_id  = data.tailscale_device.sample_device.node_id
  authorized = true
}

