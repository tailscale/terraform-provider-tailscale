data "tailscale_device" "sample_device" {
  name = "device.example.com"
}

resource "tailscale_device_authorization" "sample_authorization" {
  # Prefer the new, stable `node_id` attribute; the legacy `.id` field still works.
  device_id  = data.tailscale_device.sample_device.node_id
  authorized = true
}

# Delete a device from the tailnet when this resource is destroyed.
# Destroy uses the value stored in state, so apply this setting before
# removing the resource from the configuration.
resource "tailscale_device_authorization" "old_node" {
  device_id         = "nodeidCNTRL"
  authorized        = true
  delete_on_destroy = true
}
