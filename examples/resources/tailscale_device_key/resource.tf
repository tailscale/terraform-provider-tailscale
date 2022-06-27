data "tailscale_device" "sample_device" {
  name = "device.example.com"
}

resource "tailscale_device_key" "sample_key" {
  device_id           = data.tailscale_device.sample_device.id
  key_expiry_disabled = true
}
