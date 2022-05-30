data "tailscale_device" "sample_device" {
  name = "device.example.com"
}

resource "tailscale_device_authorization" "sample_authorization" {
  device_id  = data.tailscale_device.sample_device.id
  authorized = true
}

