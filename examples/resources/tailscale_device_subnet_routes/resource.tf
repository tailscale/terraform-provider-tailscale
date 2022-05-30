data "tailscale_device" "sample_device" {
  name = "device.example.com"
}

resource "tailscale_device_subnet_routes" "sample_routes" {
  device_id = data.tailscale_device.sample_device.id
  routes = [
    "10.0.1.0/24",
    "1.2.0.0/16",
    "2.0.0.0/24"
  ]
}
