data "tailscale_device" "sample_device" {
  name = "device.example.com"
}

resource "tailscale_device_subnet_routes" "sample_routes" {
  # Prefer the new, stable `node_id` attribute; the legacy `.id` field still works.
  device_id = data.tailscale_device.sample_device.node_id
  routes = [
    "10.0.1.0/24",
    "1.2.0.0/16",
    "2.0.0.0/24"
  ]
}

resource "tailscale_device_subnet_routes" "sample_exit_node" {
  # Prefer the new, stable `node_id` attribute; the legacy `.id` field still works.
  device_id = data.tailscale_device.sample_device.node_id
  routes = [
    # Configure as an exit node
    "0.0.0.0/0",
    "::/0"
  ]
}
